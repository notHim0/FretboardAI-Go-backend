package python_client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	retryDelay time.Duration
	maxRetries int
}

// Creates a new python service client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			// Python processing can take a long time
			// Spleeter + Basic Pitch on a 5min song ~ 45-60 seconds
			Timeout: 5 * time.Minute,
		},
		retryDelay: 3 * time.Second,
		maxRetries: 3,
	}
}

func (c *Client) TranscribeAudio(filePath string) (*TranscribeResponse, error) {
	req := TranscribeRequest{FilePath: filePath}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transcribe request: %w", err)
	}

	var lastErr error

	for attempt := 1; attempt <= c.maxRetries; attempt++ {
		log.Printf("[Python Client] Transcribe attempt %d/%d for file %s", attempt, c.maxRetries, filePath)

		resp, err := c.post("/transcribe", body)
		if err != nil {
			lastErr = err
			log.Printf("[Python Client] Attempt %d failed: %v", attempt, err)

			if attempt < c.maxRetries {
				log.Print("[Python_Client] retrying in %w", c.retryDelay)
				time.Sleep(c.retryDelay)

				c.retryDelay *= 2
			}
			continue
		}
		var transcribeResp TranscribeResponse
		if err := json.Unmarshal(resp, &transcribeResp); err != nil {
			return nil, fmt.Errorf("failed to parse transcribe response: %w", err)
		}
		// Check if Python reported an error
		if !transcribeResp.Success {
			return nil, fmt.Errorf("python service error: %s", transcribeResp.Error)
		}

		log.Printf("[Python Client] Transcription successful: %d notes detected", transcribeResp.Metadata.TotalNotes)
		return &transcribeResp, nil
	}
	return nil, fmt.Errorf("transcription failed after %d attempts: %w", c.maxRetries, lastErr)
}

// HealthCheck verifies the Python service is running and ready
func (c *Client) HealthCheck() error {
	resp, err := c.get("/health")
	if err != nil {
		return fmt.Errorf("python service is unreachable: %w", err)
	}

	var healthResp HealthResponse
	if err := json.Unmarshal(resp, &healthResp); err != nil {
		return fmt.Errorf("failed to parse health response: %w", err)
	}

	if healthResp.Status != "ok" {
		return fmt.Errorf("python service unhealthy: %s", healthResp.Message)
	}

	log.Printf("[Python Client] Health check passed: %s", healthResp.Message)
	return nil
}

func (c *Client) post(endpoint string, body []byte) ([]byte, error) {
	url := c.baseURL + endpoint

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[Python_Client] returned a status code of %d: %s", resp.StatusCode, resp.Body)
	}

	return responseBody, nil
}

// get makes a GET request to the Python service
func (c *Client) get(endpoint string) ([]byte, error) {
	url := c.baseURL + endpoint

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("python service returned status %d: %s", resp.StatusCode, string(responseBody))
	}

	return responseBody, nil
}
