package models

import (
	"time"
)

// Job tracks the processing status of an uploaded audio file
type Job struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	Filename     string     `json:"filename" gorm:"not null"`
	Status       string     `json:"status" gorm:"not null"` // pending, processing, completed, failed
	CreatedAt    time.Time  `json:"created_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	ErrorMessage *string    `json:"error_message,omitempty"`
}

// Analysis stores the music theory analysis from LLM
type Analysis struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	JobID        uint      `json:"job_id" gorm:"not null"`
	Job          Job       `json:"job" gorm:"foreignKey:JobID"`
	KeySignature string    `json:"key_signature"`
	ScaleType    string    `json:"scale_type"`
	Explanation  string    `json:"explanation" gorm:"type:text"`
	Confidence   float64   `json:"confidence"`
	CreatedAt    time.Time `json:"created_at"`
}

// Note represents a single transcribed musical note with guitar-specific attributes
type Note struct {
	ID         uint     `json:"id" gorm:"primaryKey"`
	AnalysisID uint     `json:"analysis_id" gorm:"not null;index"`
	Analysis   Analysis `json:"-" gorm:"foreignKey:AnalysisID"`
	Time       float64  `json:"time"`                            // seconds from start
	Pitch      int      `json:"pitch"`                           // MIDI note number (0-127)
	Duration   float64  `json:"duration"`                        // note length in seconds
	Fret       int      `json:"fret"`                            // guitar fret (0-24)
	String     int      `json:"string"`                          // guitar string (1-6, where 1 is high E)
	Confidence float64  `json:"confidence"`                      // transcription confidence (0-1)
	GroupID    *uint    `json:"group_id,omitempty" gorm:"index"` // Links simultaneous notes

	// Guitar technique detection
	Technique     string  `json:"technique"`                                 // "normal", "hammer_on", "pull_off", "slide", "bend", "vibrato", "tap", "harmonic"
	TechniqueData *string `json:"technique_data,omitempty" gorm:"type:text"` // JSON with technique-specific data (TechniqueDetails struct)
	Attack        string  `json:"attack"`                                    // "pick", "finger", "thumb", "hybrid", "tap", "slap"
	Dynamics      string  `json:"dynamics"`                                  // "soft", "medium", "hard", "accent"
}

// TechniqueDetails contains technique-specific metadata (stored as JSON in Note.TechniqueData)
type TechniqueDetails struct {
	// Slide specific
	SlideFromFret *int    `json:"slide_from_fret,omitempty"`
	SlideToFret   *int    `json:"slide_to_fret,omitempty"`
	SlideSpeed    *string `json:"slide_speed,omitempty"` // "slow", "fast"

	// Bend specific
	BendAmount    *float64 `json:"bend_amount,omitempty"`    // semitones (0.5, 1.0, 1.5, 2.0)
	BendDirection *string  `json:"bend_direction,omitempty"` // "up", "down"

	// Vibrato specific
	VibratoRate  *float64 `json:"vibrato_rate,omitempty"`  // Hz
	VibratoDepth *float64 `json:"vibrato_depth,omitempty"` // semitones

	// Hammer-on/Pull-off
	ConnectedFret *int  `json:"connected_fret,omitempty"`    // fret it connects to
	ConnectedNote *uint `json:"connected_note_id,omitempty"` // ID of the connected note

	// Tapping
	TapHand *string `json:"tap_hand,omitempty"` // "left", "right", "both"

	// Harmonic
	HarmonicType *string `json:"harmonic_type,omitempty"` // "natural", "artificial", "pinch"
}

// NoteGroup represents simultaneous or related notes (chords, arpeggios, etc.)
type NoteGroup struct {
	ID         uint     `json:"id" gorm:"primaryKey"`
	AnalysisID uint     `json:"analysis_id" gorm:"not null;index"`
	Analysis   Analysis `json:"-" gorm:"foreignKey:AnalysisID"`
	Time       float64  `json:"time"`                            // seconds from start
	Duration   float64  `json:"duration"`                        // group duration in seconds
	GroupType  string   `json:"group_type"`                      // "chord", "arpeggio", "strum", "double_stop", "power_chord", "palm_mute"
	Name       string   `json:"name"`                            // "G Major", "E5 Power Chord", "Fingerpick Pattern", "Am Arpeggio"
	Notes      []Note   `json:"notes" gorm:"foreignKey:GroupID"` // All notes in this group
	Confidence float64  `json:"confidence"`                      // detection confidence (0-1)

	// Playing technique for the group
	PlayingStyle string `json:"playing_style"`   // "fingerpicking", "strumming", "hybrid_picking", "sweep_picking", "alternate_picking", "economy_picking"
	Tempo        *int   `json:"tempo,omitempty"` // BPM for this section (if detected)
}

// UploadResponse is returned immediately after file upload
type UploadResponse struct {
	JobID         uint   `json:"job_id"`
	Status        string `json:"status"`
	Message       string `json:"message"`
	EstimatedTime int    `json:"estimated_seconds"`
}

// JobStatusResponse tracks processing progress
type JobStatusResponse struct {
	JobID    uint   `json:"job_id"`
	Status   string `json:"status"`
	Progress int    `json:"progress"` // 0-100
	Message  string `json:"message"`
}

// JobResultResponse contains the complete transcription result
type JobResultResponse struct {
	JobID      uint        `json:"job_id"`
	Status     string      `json:"status"`
	Analysis   *Analysis   `json:"analysis,omitempty"`
	Notes      []Note      `json:"notes,omitempty"`
	NoteGroups []NoteGroup `json:"note_groups,omitempty"`
	AudioURL   string      `json:"audio_url,omitempty"` // URL to access processed audio
}
