package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/nothim0/fretboardAI-Go-backend/config"
	"github.com/nothim0/fretboardAI-Go-backend/internal/handler"
	"github.com/nothim0/fretboardAI-Go-backend/internal/repository"
	"github.com/nothim0/fretboardAI-Go-backend/internal/service"
	"github.com/nothim0/fretboardAI-Go-backend/pkg/llm_client"
	"github.com/nothim0/fretboardAI-Go-backend/pkg/python_client"
)

func main() {
	cfg := config.LoadConfig()
	log.Println("configuration loaded")

	dirs := []string{cfg.UploadDir, cfg.ProcessedAudioDir}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0775); err != nil {
			log.Fatalf("Failed to create dir %s: %v", dir, err)
		}
	}

	log.Println("Directoreis created/verified")

	db, err := repository.NewDatabse(cfg.DatabasePath)

	if err != nil {
		log.Fatalf("failed to initialise database: %v", err)
	}

	defer db.Close()

	log.Println("Database connnect succesfully")

	jobRepo := repository.NewJobRepository(db.DB)
	analysisRepo := repository.NewAnalysisRepository(db.DB)
	noteRepo := repository.NewNoteRepository(db.DB)
	noteGroupRepo := repository.NewNoteGroupRepository(db.DB)

	log.Println("Repositories initialised")

	//Initailise python client
	pythonClient := python_client.NewClient(cfg.PythonServiceURL)
	if err := pythonClient.HealthCheck(); err != nil {
		log.Printf("WARNING: Python service unavailable: %v", err)
		log.Printf("WARNING: Transcription will fail until Python service is running on %s", cfg.PythonServiceURL)
	} else {
		log.Println("Python service connected successfully")
	}
	//Initialize LLM client
	llmClient := llm_client.NewClient(cfg.AnthropicAPIKey, cfg.AnthropicModel)
	if cfg.AnthropicAPIKey == "" {
		log.Printf("WARNING: Anthropic API key is not set - music analysis will fail")
	} else {
		log.Printf("LLM Client initialised successfully")
	}

	//Initialize service layer
	transcriptionService := service.NewTranscriptionService(
		jobRepo,
		analysisRepo,
		noteRepo,
		noteGroupRepo,
		pythonClient,
		llmClient,
		cfg,
	)

	log.Println("Transcription service initialized")
	// TODO: Initialize handlers (Step 6)

	//Initialize handlers
	transcriptionHandler := handler.NewTranscriptionHandler(
		transcriptionService,
		jobRepo,
		cfg,
	)

	log.Println("Handlers initialised")
	router := gin.Default()

	//set amx multipart memory(for file uploads)
	router.MaxMultipartMemory = cfg.MaxFileSize

	//CORS middleware config
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "200",
			"message": "FretBoardAI is running...",
		})
	})

	api := router.Group("/api")
	{
		// Upload audio file for transcription
		api.POST("/upload", transcriptionHandler.UploadAudio)

		// Job management
		api.GET("/jobs", transcriptionHandler.GetJobsList)
		api.GET("/jobs/:id/status", transcriptionHandler.GetJobStatus)
		api.GET("/jobs/:id/result", transcriptionHandler.GetJobResult)
		api.DELETE("/jobs/:id", transcriptionHandler.DeleteJob)
	}

	//static file serving
	router.Static("/audio", cfg.ProcessedAudioDir)

	log.Println("Routes registered:")
	log.Println("  POST   /api/upload              - Upload audio file")
	log.Println("  GET    /api/jobs                - List all jobs")
	log.Println("  GET    /api/jobs/:id/status     - Get job status")
	log.Println("  GET    /api/jobs/:id/result     - Get job result")
	log.Println("  DELETE /api/jobs/:id            - Delete job")
	log.Println("  GET    /health                  - Health check")
	log.Println("  GET    /audio/*filepath         - Serve processed audio")

	log.Printf("Starting server on port %s...", cfg.Port)
	log.Printf("Upload directory: %s", cfg.UploadDir)
	log.Printf("Processed audio directory: %s", cfg.ProcessedAudioDir)

	go func() {
		if err := router.Run(":" + cfg.Port); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	//for gracefull shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
}
