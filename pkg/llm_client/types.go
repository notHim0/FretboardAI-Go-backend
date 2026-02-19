package llm_client

type AnalysisResponse struct {
	Notes     []NoteContext  `json:"notes"`
	NoteGroup []GroupContext `json:"note_group"`
}

// NoteContext is a simplified note representation for the LLM prompt
// We don't send the full Note model - only what the LLM needs to reason about
type NoteContext struct {
	Time      float64 `json:"time"`       // seconds from start
	Pitch     int     `json:"pitch"`      // MIDI number
	NoteName  string  `json:"note_name"`  // human readable: "E4", "G#3"
	Duration  float64 `json:"duration"`   // seconds
	Fret      int     `json:"fret"`       // already mapped by Go
	String    int     `json:"string"`     // already mapped by Go
}
// GroupContext is a simplified note group for the LLM prompt
type GroupContext struct {
	Time      float64  `json:"time"`
	GroupType string   `json:"group_type"` // "chord", "arpeggio", etc.
	NoteNames []string `json:"note_names"` // ["E", "G", "B"]
	Duration  float64  `json:"duration"`
}

// AnalysisResult is what we expect back from the LLM
type AnalysisResult struct {
	KeySignature string             `json:"key_signature"` // "E Minor", "G Major"
	ScaleType    string             `json:"scale_type"`    // "Minor Pentatonic", "Dorian"
	Explanation  string             `json:"explanation"`   // full theory explanation
	Confidence   float64            `json:"confidence"`    // LLM self-reported confidence 0-1
	Techniques   []TechniqueSuggestion `json:"techniques"`
}

// TechniqueSuggestion is a single technique the LLM identified
type TechniqueSuggestion struct {
	StartTime   float64 `json:"start_time"`   // when the technique starts
	EndTime     float64 `json:"end_time"`     // when it ends
	Type        string  `json:"type"`         // "hammer_on", "pull_off", "slide", "bend", "vibrato", "tap", "strum", "fingerpick"
	Description string  `json:"description"`  // "Hammer-on from E to G on 4th string"
	Confidence  float64 `json:"confidence"`   // how sure the LLM is (0-1)
	NoteIndices []int   `json:"note_indices"` // which notes in the input this applies to
}

// openAIRequest is the raw OpenAI API request body
type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
	MaxTokens   int             `json:"max_tokens"`
}

// openAIMessage is a single message in the OpenAI conversation
type openAIMessage struct {
	Role    string `json:"role"`    // "system" or "user"
	Content string `json:"content"`
}

// openAIResponse is the raw OpenAI API response
type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// NoteInput is what the service layer passes into BuildRequest
// Keeps the llm_client package decoupled from the models package
type NoteInput struct {
	Time     float64
	Pitch    int
	Duration float64
	Fret     int
	String   int
}

// GroupInput is what the service layer passes into BuildRequest
type GroupInput struct {
	Time      float64
	GroupType string
	NoteNames []string
	Duration  float64
}
