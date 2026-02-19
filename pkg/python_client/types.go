package python_client

// TranscribeRequest is sent to the Python service
type TranscribeRequest struct {
	FilePath string `json:"file_path"` //absolute path
}

// TranscribeResponse is returned by the Python service
type TranscribeResponse struct {
	Success  bool          `json:"success"`
	Notes    []RawNote     `json:"notes"`
	AudioURL string        `json:"guitar_stem_path"` // path to the extracted guitar stem
	Error    string        `json:"error,omitempty"`  // only set if success=false
	Metadata AudioMetadata `json:"metadata"`
}

// RawNote is a single note as returned by Basic Pitch
// This is the "dumb" raw data before Go processes it into a models.Note
type RawNote struct {
	Time       float64 `json:"time"`       // seconds from start
	Pitch      int     `json:"pitch"`      // MIDI note number (0-127)
	Duration   float64 `json:"duration"`   // note length in seconds
	Confidence float64 `json:"confidence"` // Basic Pitch confidence (0-1)
}

// AudioMetadata contains information about the processed audio
type AudioMetadata struct {
	OriginalDuration float64 `json:"original_duration"` // seconds
	SampleRate       int     `json:"sample_rate"`       // Hz (usually 44100)
	TotalNotes       int     `json:"total_notes"`       // how many notes were detected
	ProcessingTime   float64 `json:"processing_time"`   // seconds taken to process
}

// HealthResponse is returned by the Python service health check
type HealthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
