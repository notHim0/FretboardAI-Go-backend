package service

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nothim0/fretboardAI-Go-backend/config"
	"github.com/nothim0/fretboardAI-Go-backend/internal/models"
	"github.com/nothim0/fretboardAI-Go-backend/internal/repository"
	"github.com/nothim0/fretboardAI-Go-backend/pkg/llm_client"
	"github.com/nothim0/fretboardAI-Go-backend/pkg/python_client"
)

// TranscriptionService orchestrates the entire transcription pipeline
type TranscriptionService struct {
	jobRepo       *repository.JobRepository
	analysisRepo  *repository.AnalysisRepository
	noteRepo      *repository.NoteRepository
	noteGroupRepo *repository.NoteGroupRepository
	pythonClient  *python_client.Client
	llmClient     *llm_client.Client
	config        *config.Config
}

// NewTranscriptionService creates a new transcription service
func NewTranscriptionService(
	jobRepo *repository.JobRepository,
	analysisRepo *repository.AnalysisRepository,
	noteRepo *repository.NoteRepository,
	noteGroupRepo *repository.NoteGroupRepository,
	pythonClient *python_client.Client,
	llmClient *llm_client.Client,
	cfg *config.Config,
) *TranscriptionService {
	return &TranscriptionService{
		jobRepo:       jobRepo,
		analysisRepo:  analysisRepo,
		noteRepo:      noteRepo,
		noteGroupRepo: noteGroupRepo,
		pythonClient:  pythonClient,
		llmClient:     llmClient,
		config:        cfg,
	}
}

func (s *TranscriptionService) ProcessTranscription(jobID uint, audioFilePath string) error {
	log.Printf("[Service] Starting transcription for job %d", jobID)

	if err := s.jobRepo.UpdateStatus("processing", jobID, ""); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	//call python service (spleeter + basic pitch)
	log.Printf("[Service] Step 1/5: Calling Python service for audio transcription")
	transcribeResp, err := s.pythonClient.TranscribeAudio(audioFilePath)
	if err != nil {
		errMsg := fmt.Sprintf("Python transcription failed: %v", err)
		s.jobRepo.UpdateStatus("failed", jobID, errMsg)
		return fmt.Errorf("transcription failed: %w", err)
	}
	if len(transcribeResp.Notes) == 0 {
		errMsg := "No notes detected in audio"
		s.jobRepo.UpdateStatus("failed", jobID, errMsg)
		return fmt.Errorf("no notes detected")
	}

	log.Printf("[Service] Python service returned %d raw notes", len(transcribeResp.Notes))

	//Map MIDI notes to guitar fret/string positions
	log.Printf("[Service] Step 2/5: Mapping notes to guitar fretboard")
	guitarNotes := MapToGuitar(transcribeResp.Notes)
	log.Printf("[Service] Mapped %d notes to guitar positions", len(guitarNotes))

	//Detect chords and note groups
	log.Printf("[Service] Step 3/5: Detecting chords and note groups")
	noteGroups := DetectChords(guitarNotes, s.config.ChordTimeThreshold)
	log.Printf("[Service] Detected %d note groups (chords/appeggios)", len(noteGroups))

	//Prepare data for LLM Analysis
	log.Printf("[Service] Step 4/5: Running LLM music thoery analysis")
	noteInputs := convertToNoteInputs(guitarNotes)
	groupInputs := convertToGroupInputs(noteGroups)

	analysisResult, err := s.llmClient.AnalyzeNotes(noteInputs, groupInputs)
	if err != nil {
		log.Printf("[Service] WARNING: LLM analysis failed: %v", err)
		log.Printf("[Service] Continuing without muisc theory analysis")
		analysisResult = &llm_client.AnalysisResult{
			KeySignature: "Unknown",
			ScaleType:    "Unknown",
			Explanation:  fmt.Sprintf("Music theory analysis unavaliable: %v", err),
			Confidence:   0.0,
			Techniques:   []llm_client.TechniqueSuggestion{},
		}
	}

	log.Printf("[Service] Step 5/5: Saving results to database")

	analysis := &models.Analysis{
		JobID:        jobID,
		KeySignature: analysisResult.KeySignature,
		ScaleType:    analysisResult.ScaleType,
		Explanation:  analysisResult.Explanation,
		Confidence:   analysisResult.Confidence,
		CreatedAt:    time.Now(),
	}

	if err := s.analysisRepo.Create(analysis); err != nil {
		errMsg := fmt.Sprintf("Failed to save analysis: %v", err)
		s.jobRepo.UpdateStatus("failed", jobID, errMsg)
		return fmt.Errorf("failed to save analysis: %w", err)
	}

	log.Printf("[Service] Analysis saved with ID %d", err)

	//convert guitar notes to models.Note with analysis ID
	dbNotes := make([]models.Note, len(guitarNotes))
	for i, gn := range guitarNotes {
		technique, techniqueData := extractTechnique(i, analysisResult.Techniques)

		dbNotes[i] = models.Note{
			AnalysisID:    analysis.ID,
			Time:          gn.Time,
			Pitch:         gn.Pitch,
			Duration:      gn.Duration,
			Fret:          gn.Fret,
			String:        gn.String,
			Confidence:    gn.Confidence,
			Technique:     technique,
			TechniqueData: techniqueData,
			Attack:        "pick",
			Dynamics:      "medium",
		}
	}

	//Save note in bulk
	if err := s.noteRepo.CreateInBulk(dbNotes); err != nil {
		errMsg := fmt.Sprintf("Failed to save notes: %v", err)
		s.jobRepo.UpdateStatus("failed", jobID, errMsg)
		return fmt.Errorf("failed to save notes: %w", err)
	}

	log.Printf("[Service] Saved %d notes", len(dbNotes))

	//Convert and save note groups
	dbGroups := make([]models.NoteGroup, len(noteGroups))
	for i, ng := range noteGroups {
		dbGroups[i] = models.NoteGroup{
			AnalysisID:   analysis.ID,
			Time:         ng.Time,
			Duration:     ng.Duration,
			GroupType:    ng.GroupType,
			Name:         ng.Name,
			Confidence:   ng.Confidence,
			PlayingStyle: detectPlayingStyle(ng),
		}
	}

	if err := s.noteGroupRepo.CreateBulk(dbGroups); err != nil {
		log.Printf("[Service] WARNING: Failed to save note groups: %v", err)
	} else {
		log.Printf("[Service] Saved %d note groups", len(dbGroups))
	}

	if err := s.jobRepo.UpdateStatus("completed", jobID, ""); err != nil {
		return fmt.Errorf("failed to mark job as completed: %w", err)
	}

	log.Printf("[Service] Job %d completed successfully", jobID)
	return nil
}

// GetJobStatus retrieves the current state of a job
func (s *TranscriptionService) GetJobStatus(jobID uint) (*models.JobStatusResponse, error) {
	job, err := s.jobRepo.GetById(jobID)
	if err != nil {
		return nil, err
	}

	progress := 0
	message := ""

	switch job.Status {
	case "pending":
		progress = 0
		message = "Queued for processing"
	case "processing":
		progress = 50
		message = "Transcribing audio and analyzing music thoery..."
	case "completed":
		progress = 100
		message = "Transcription Complete"
	case "failed":
		progress = 0
		if job.ErrorMessage != nil {
			message = *job.ErrorMessage
		} else {
			message = "Processing failed"
		}
	}

	return &models.JobStatusResponse{
		JobID:    jobID,
		Status:   job.Status,
		Progress: progress,
		Message:  message,
	}, nil
}

// GetJobResult retrieves the complete transcription result
func (s *TranscriptionService) GetJobResult(jobID uint) (*models.JobResultResponse, error) {
	job, err := s.jobRepo.GetById(jobID)
	if err != nil {
		return nil, err
	}

	result := &models.JobResultResponse{
		Status: job.Status,
		JobID:  job.ID,
	}

	if job.Status != "completed" {
		return result, nil
	}

	analysis, err := s.analysisRepo.GetByJobId(jobID)
	if err != nil {
		log.Printf("[Service] WARNING: Could not find analysis for job %d: %v", jobID, err)
	} else {
		result.Analysis = analysis
	}

	if analysis != nil {
		notes, err := s.noteRepo.GetByAnalysisId(analysis.ID)
		if err != nil {
			log.Printf("[Service] WARNING: Could not get notes: %v", err)
		} else {
			result.Notes = notes
		}

		notesGroup, err := s.noteGroupRepo.GetByAnalysisId(analysis.ID)
		if err != nil {
			log.Printf("[Service] WARNING: could not get notes groups: %v", err)
		} else {
			result.NoteGroups = notesGroup
		}
	}

	//Set AudioURL to processed guitar stem path
	result.AudioURL = fmt.Sprintf("/audio/%d/guitar_stem.wav", jobID)
	return result, nil
}

// extractTechnique finds the technique for a specific note index from LLM suggestions
func extractTechnique(noteIndex int, techniques []llm_client.TechniqueSuggestion) (string, *string) {
	for _, tech := range techniques {
		for _, idx := range tech.NoteIndices {
			if idx == noteIndex {
				techData := map[string]interface{}{
					"type":        tech.Type,
					"description": tech.Description,
					"confidence":  tech.Confidence,
				}

				jsonData, _ := json.Marshal(techData)
				jsonStr := string(jsonData)

				return tech.Type, &jsonStr
			}
		}
	}

	return "normal", nil
}

// detectPlayingStyle infers playing style from note group characteristics
func detectPlayingStyle(ng GuitarNoteGroup) string {
	if len(ng.Notes) >= 4 {
		return "strumming"
	}

	return "fingerpicking"
}

// converts GuitarNote to llm_client.NoteInput
func convertToNoteInputs(notes []GuitarNote) []llm_client.NoteInput {
	inputs := make([]llm_client.NoteInput, len(notes))

	for i, n := range notes {
		inputs[i] = llm_client.NoteInput{
			Time:     n.Time,
			Pitch:    n.Pitch,
			Duration: n.Duration,
			Fret:     n.Fret,
			String:   n.String,
		}
	}

	return inputs
}

// convertToGroupInputs converts GuitarNoteGroup to llm_client.GroupInput
func convertToGroupInputs(groups []GuitarNoteGroup) []llm_client.GroupInput {
	inputs := make([]llm_client.GroupInput, len(groups))

	for i, g := range groups {
		inputs[i] = llm_client.GroupInput{
			Time:      g.Time,
			GroupType: g.GroupType,
			NoteNames: g.NoteNames,
			Duration:  g.Duration,
		}
	}

	return inputs
}
