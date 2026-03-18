package llm_client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const baseURL = "https://api.anthropic.com/v1/messages"

type Client struct {
	apikey     string
	httpClient *http.Client
	model      string
	maxRetries int
	retryDelay time.Duration
}

func NewClient(apikey, model string) *Client {
	return &Client{
		apikey: apikey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		retryDelay: 2 * time.Second,
		maxRetries: 3,
	}
}

func (c *Client) AnalyzeNotes(notes []NoteInput, groups []GroupInput) (*AnalysisResult, error) {
	if c.apikey == "" {
		return nil, fmt.Errorf("AnthropicAPI key is not set")
	}

	if len(notes) == 0 {
		return nil, fmt.Errorf("no notes provided for analysis")
	}

	notesContexts := BuildNoteContexts(notes)
	groupContexts := BuildGroupContexts(groups)

	userPrompt := buildUserPrompt(notesContexts, groupContexts)

	request := anthropicRequest{
		Model: c.model,
		Messages: []anthropicMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.2,
		MaxTokens:   2000,
	}

	//Attempt with retries and exponential backoff
	var lastErr error
	for attempt := 1; attempt <= c.maxRetries; attempt++ {
		log.Printf("[LLM Client] Analysis attempt %d/%d", attempt, c.maxRetries)

		result, err := c.callAnthropic(request)
		if err != nil {
			lastErr = err
			log.Printf("[LLM Client] Attempt %d failed: %v", attempt, err)

			if attempt < c.maxRetries {
				log.Printf("[LLM Client] Retrying in %v...", c.retryDelay)
				time.Sleep(c.retryDelay)
				c.retryDelay *= 2
			}
			continue
		}

		log.Printf("[LLM Client] Analysis successful: key=%s, scale=%s, techniques=%d", result.KeySignature, result.ScaleType, len(result.Techniques))
		return result, nil
	}

	return nil, fmt.Errorf("LLM Analysis failed after %d attempts: %w", c.maxRetries, lastErr)
}

// callanthropic makes a single request to anthropic API
func (c *Client) callAnthropic(request anthropicRequest) (*AnalysisResult, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal anthropic request: %w", err)
	}
	req, err := http.NewRequest("POST", baseURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apikey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("anthropic request failed: %w", err)
	}

	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read anthropic response: %w", err)
	}

	//Parse the anthropic response envelope
	var anthropicResp anthropicResponse
	if err := json.Unmarshal(responseBody, &anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to parse anthropic response: %w", err)
	}

	//check for api level errors(wrong key, rate limit, etc.)
	if anthropicResp.Error != nil {
		return nil, fmt.Errorf("anthropic API error (%s): %s", anthropicResp.Error.Type, anthropicResp.Error.Message)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic returned status %d: %s", resp.StatusCode, string(responseBody))
	}

	if len(anthropicResp.Choices) == 0 {
		return nil, fmt.Errorf("anthropic returned no choices")
	}

	content := anthropicResp.Choices[0].Message.Content
	return parseAnalysisResult(content)
}

// parseAnalysisResult parses the JSON string returned by GPT-4o
// Defensively strips markdown backticks in case GPT-4o adds them despite instructions
func parseAnalysisResult(content string) (*AnalysisResult, error) {
	content = strings.TrimSpace(content)

	if strings.HasPrefix(content, "```") {
		firstNewLine := strings.Index(content, "\n")
		if firstNewLine != -1 {
			content = content[firstNewLine+1:]
		}
		if idx := strings.LastIndex(content, "```"); idx != -1 {
			content = content[:idx]
		}

		content = strings.TrimSpace(content)
	}

	var result AnalysisResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("failed to parse analysis JSON: %w\nContent: %s", err, content)
	}

	//Validating required fields
	if result.KeySignature == "" {
		return nil, fmt.Errorf("GPT-4o returned empty key signature")
	}

	if result.ScaleType == "" {
		return nil, fmt.Errorf("GPT-4o returned empty scale type")
	}

	if result.Confidence < 0 {
		result.Confidence = 0
	}
	if result.Confidence > 1 {
		result.Confidence = 1
	}

	return &result, nil
}
