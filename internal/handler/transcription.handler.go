package handler

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nothim0/fretboardAI-Go-backend/config"
	"github.com/nothim0/fretboardAI-Go-backend/internal/models"
	"github.com/nothim0/fretboardAI-Go-backend/internal/repository"
	"github.com/nothim0/fretboardAI-Go-backend/internal/service"
)

// TranscriptionHandler handles HTTP request for transcription operations
type TranscriptionHandler struct {
	service *service.TranscriptionService
	jobRepo *repository.JobRepository
	config  *config.Config
}

// NewTranscriptionHandler creates a new transcription handler
func NewTranscriptionHandler(service *service.TranscriptionService, jobRepo *repository.JobRepository, cfg *config.Config) *TranscriptionHandler {
	return &TranscriptionHandler{
		service: service,
		jobRepo: jobRepo,
		config:  cfg,
	}
}

// UploadAudio handles audio file uploads and starts transcription
// POST /api/upload
func (h *TranscriptionHandler) UploadAudio(c *gin.Context) {
	log.Printf("[Handler] Upload request received")

	file, err := c.FormFile("file")
	if err != nil {
		log.Printf("[Handler] Failed to get file from form: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No file uploaded",
		})
		return
	}

	if file.Size > h.config.MaxFileSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("File too large. Maximum size is %d MB", h.config.MaxFileSize/(1024*1024)),
		})

		return
	}

	//Validate file extension
	ext := filepath.Ext(file.Filename)
	validExts := map[string]bool{
		".mp3":  true,
		".wav":  true,
		".m4a":  true,
		".flac": true,
		".ogg":  true,
	}

	if !validExts[ext] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid file type. Supported formats: mp3, wav, m4a, flac, ogg",
		})

		return
	}

	log.Printf("[Handler] File received: %s (%.2f MB)", file.Filename, float64(file.Size)/(1024*1024))

	//Create job record
	job, err := h.jobRepo.Create(file.Filename)
	if err != nil {
		log.Printf("[Handler] Failed to create job: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create job",
		})
		return
	}

	log.Printf("[Handler] Job Created with ID %d", job.ID)

	//Save uploaded file
	uploadPath := filepath.Join(h.config.UploadDir, fmt.Sprintf("%d_%s", job.ID, file.Filename))
	if err := c.SaveUploadedFile(file, uploadPath); err != nil {
		log.Printf("[Handler] Failed to save file: %v", err)
		errMsg := "Failed to save uploaded file"
		h.jobRepo.UpdateStatus("failed", job.ID, errMsg)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save file",
		})
		return
	}
	log.Printf("[Handler] File saved to %s", uploadPath)

	//Start transcription in background goroutine
	go func() {
		log.Printf("[Handler] Starting background transcription for job %d", job.ID)
		if err := h.service.ProcessTranscription(job.ID, uploadPath); err != nil {
			log.Printf("[Handler] Transcription failed for job %d: %v", job.ID, err)
		} else {
			log.Printf("[Handler] Transcription completed for job %d", job.ID)
		}
	}()

	//Estimate processing time based on file size
	//Rough estimate: 1 minute of processing per 1MB of audio

	estimatedSeconds := int(file.Size / (1024 * 1024) * 60)
	if estimatedSeconds < 30 {
		estimatedSeconds = 30
	}
	if estimatedSeconds > int(h.config.MaxDuration) {
		estimatedSeconds = int(h.config.MaxDuration)
	}

	response := models.UploadResponse{
		JobID:         job.ID,
		Status:        "pending",
		Message:       "File uploaded successfully. Transcription started.",
		EstimatedTime: estimatedSeconds,
	}

	c.JSON(http.StatusOK, response)
}

// GetJobStatus returns the current status of a job
// GET /api/jobs/:id/status
func (h *TranscriptionHandler) GetJobStatus(c *gin.Context) {
	jobIDStr := c.Param("id")
	jobID, err := strconv.ParseUint(jobIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid job ID",
		})
		return
	}

	log.Printf("[Handler] Status check for job %d", jobID)
	status, err := h.service.GetJobStatus(uint(jobID))
	if err != nil {
		log.Printf("[Handler] Failed to get job status: %v", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Job not found",
		})
		return
	}

	c.JSON(http.StatusOK, status)
}

// GetJobResult returns the complete transcription result
// GET /api/job/:id/result
func (h *TranscriptionHandler) GetJobResult(c *gin.Context) {
	jobIDStr := c.Param("id")
	jobID, err := strconv.ParseUint(jobIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid job ID",
		})
		return
	}

	log.Printf("[Handler] Result request for job %d", jobID)

	result, err := h.service.GetJobResult(uint(jobID))
	if err != nil {
		log.Printf("[Handler] Failed to get job result: %v", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Job not found",
		})
		return
	}

	if result.Status != "completed" {
		c.JSON(http.StatusOK, result)
		return
	}

	c.JSON(http.StatusOK, result)

}

// GetJobsList returns a list of recent jobs (for debugging/admin)
// GET /api/jobs
func (h *TranscriptionHandler) GetJobsList(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "10")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	log.Printf("[Handler] Jobs list requested (limit=%d, offset=%d)", limit, offset)

	//Get jobs from repository
	jobs, err := h.jobRepo.List(limit, offset)
	if err != nil {
		log.Printf("[Handler] Failed to get jobs list: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve jobs",
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs":   jobs,
		"limit":  limit,
		"offset": offset,
		"count":  len(jobs),
	})

}

// DeleteJob deletes a job and all associated data
// DELETE /api/jobs/:id
func (h *TranscriptionHandler) DeleteJob(c *gin.Context) {
	jobIDStr := c.Param("id")
	jobID, err := strconv.ParseUint(jobIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid job ID",
		})
		return
	}

	log.Printf("[Handler] Delete request for job %d", jobID)

	job, err := h.jobRepo.GetById(uint(jobID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Job not found",
		})
		return
	}

	if job.Status == "processing" {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Cannot delete job that is currently processing",
		})
		return
	}

	if err := h.jobRepo.Delete(uint(jobID)); err != nil {
		log.Printf("[Handler] Failed to delete job: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete job",
		})
		return
	}

	log.Printf("[Handler] Job %d deleted successfully", jobID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Job deleted successfully",
		"job_id":  jobID,
	})
}

// HealthCheck returns API health status
// GET /health
func (h *TranscriptionHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"message":   "Guitar Transcriber API is running",
		"timestamp": time.Now().Unix(),
	})
}
