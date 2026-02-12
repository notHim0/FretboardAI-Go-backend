package config

import (
	"os"
	"strconv"
)

// Config holds all application configuration
type Config struct {
	// Server settings
	Port string

	// Database settings
	DatabasePath string

	// Python service settings
	PythonServiceURL string

	// OpenAI settings
	OpenAIAPIKey string
	OpenAIModel  string

	// File upload settings
	MaxFileSize      int64  // in bytes
	MaxDuration      int    // in seconds
	UploadDir        string
	ProcessedAudioDir string

	// Processing settings
	ChordTimeThreshold float64 // milliseconds to group notes as chord
	LowConfidenceThreshold float64 // notes below this are flagged
}

// LoadConfig loads configuration from environment variables with defaults
func LoadConfig() *Config {
	return &Config{
		// Server
		Port: getEnv("PORT", "8080"),

		// Database
		DatabasePath: getEnv("DATABASE_PATH", "./guitar-transcriber.db"),

		// Python service
		PythonServiceURL: getEnv("PYTHON_SERVICE_URL", "http://localhost:5000"),

		// OpenAI
		OpenAIAPIKey: getEnv("OPENAI_API_KEY", ""),
		OpenAIModel:  getEnv("OPENAI_MODEL", "gpt-4o"),

		// File upload
		MaxFileSize:       getEnvInt64("MAX_FILE_SIZE", 50*1024*1024), // 50MB default
		MaxDuration:       getEnvInt("MAX_DURATION", 600),              // 10 minutes default
		UploadDir:         getEnv("UPLOAD_DIR", "./uploads"),
		ProcessedAudioDir: getEnv("PROCESSED_AUDIO_DIR", "./processed"),

		// Processing
		ChordTimeThreshold:     getEnvFloat64("CHORD_TIME_THRESHOLD", 50.0), // 50ms default
		LowConfidenceThreshold: getEnvFloat64("LOW_CONFIDENCE_THRESHOLD", 0.7),
	}
}

// Helper functions to get environment variables with defaults
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvFloat64(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}