package repository

import (
	"fmt"

	"github.com/nothim0/fretboardAI-Go-backend/internal/models"
	"gorm.io/gorm"
)

//handles database operations for note repository
type NoteRepository struct {
	db *gorm.DB
}

//creates a new note repository
func NewNoteRepository(db *gorm.DB) *NoteRepository {
	return &NoteRepository{db: db}
}

func (r *NoteRepository) CreateInBulk(notes []models.Note) error {
	if len(notes) == 0 {
		return nil
	}

	err := r.db.Transaction(func (tx *gorm.DB) error {

	var batchsize int = 100
	for  i := 0; i<len(notes); i+=batchsize {
		end := min(i + batchsize, len(notes))

		if err := tx.Create(notes[i:end]).Error; err != nil {
			return err
		}

	}
	return nil
})

	if err != nil {
		return fmt.Errorf("failed to create notes in bulk: %w", err)
	}
	return nil 
}

//GetByAnalysisId lists all the notes for the given analysis
func (r *NoteRepository) GetByAnalysisId(analysisId uint) ([]models.Note, error) {
	var notes []models.Note

	if err := r.db.Where("analysis_id = ?", analysisId).Order("time ASC").Find(&notes).Error; err != nil {
		return nil, fmt.Errorf("failed to get notes by analysis: %w", err)
	}

	return notes, nil
}

//GetByGroupId lists all the notes for the given group
func (r *NoteRepository) GetByGroupId(groupId uint) ([]models.Note, error){
	var notes []models.Note

	if err := r.db.Where("group_id = ?", groupId).Order("time ASC").Find(&notes).Error; err != nil {
		return nil, fmt.Errorf("failed to get notes by group: %w", err)
	}

	return notes, nil
}

//GetByTimeRange lists all notes by in the time range given
func (r *NoteRepository) GetByTimeRange(analysisId uint, startTime, endTime float64) ([]models.Note, error) {
	var notes []models.Note

	if err := r.db.Where("analysis_id = ? AND time >= ? AND time <= ?", analysisId, startTime, endTime).Order("time ASC").Find(&notes).Error; err != nil {
		return nil, fmt.Errorf("failed to get notes by time range: %w", err)
	}

	return notes, nil
}


//UpdateGroupId updates the group id of a note
func (r *NoteRepository) UpdateGroupId(noteId uint, groupId *uint) error {
	if err := r.db.Model(&models.Note{}).Where("id = ? ", noteId).Update("group_id", groupId).Error; err != nil {
		return fmt.Errorf("failed to update group id: %w", err)
	}

	return nil

}

//Delete deletes a note by note id
func (r *NoteRepository) Delete(id uint) error {
	if err := r.db.Delete(&models.Note{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete note: %w", err)
	}
	
	return nil
}

// DeleteByAnalysisID deletes all notes for a given analysis
func (r *NoteRepository) DeleteByAnalysisID(analysisID uint) error {
	if err := r.db.Where("analysis_id = ?", analysisID).Delete(&models.Note{}).Error; err != nil {
		return fmt.Errorf("failed to delete notes: %w", err)
	}
	return nil
}

//CountByAnalysisId count total number of notes in a analysis
func (r *NoteRepository) CountByAnalysisId(analysisId uint) (int64, error) {
	var count int64

	if err := r.db.Model(&models.Note{}).Where("analysis_id = ?", analysisId).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to get note count for the analysis: %w", err)
	}

	return count, nil
}

// GetLowConfidenceNotes retrieves notes below a confidence threshold
func (r *NoteRepository) GetLowConfidenceNotes(analysisID uint, threshold float64) ([]models.Note, error) {
	var notes []models.Note
	if err := r.db.Where("analysis_id = ? AND confidence < ?", analysisID, threshold).Order("time ASC").Find(&notes).Error; err != nil {
		return nil, fmt.Errorf("failed to get low confidence notes: %w", err)
	}
	return notes, nil
}