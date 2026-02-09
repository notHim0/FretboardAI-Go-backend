package repository

import (
	"fmt"

	"github.com/nothim0/fretboardAI-Go-backend/internal/models"
	"gorm.io/gorm"
)

//handles database operations for note_group
type NoteGroupRepository struct {
	db *gorm.DB
}

//creates a new note_group repository
func NewNoteGroupRepositoryRepositoryRepository(db *gorm.DB) *NoteGroupRepository {
	return &NoteGroupRepository{db: db}
}

//creates a note group
func (r *NoteGroupRepository) Create(group *models.NoteGroup) error {
	if err := r.db.Create(group).Error; err != nil {
		return fmt.Errorf("failed to create a note group: %w", err)
	}

	return nil
}

// CreateBulk creates multiple note groups in a single transaction
func (r *NoteGroupRepository) CreateBulk(groups []models.NoteGroup) error {
	if len(groups) == 0 {
		return nil
	}

	err := r.db.CreateInBatches(groups, 100).Error

	if err != nil {
		return fmt.Errorf("failed to create in batches: %w", err)
	}
	
	return nil
}

//GetById retrieves a note_group with its notes
func (r *NoteGroupRepository) GetById(id uint) (*models.NoteGroup, error) {
	var notegroup models.NoteGroup

	if err := r.db.Preload("Notes").Find(&notegroup, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("note group not found: %w",err)
		}
		return nil, fmt.Errorf("failed to get note groups: %w", err)
	}

	return &notegroup, nil
}


//GetByAnalysisId retrieves note_groups by analysis id 
func (r *NoteGroupRepository) GetByAnalysisId(analysisId uint) ([]models.NoteGroup, error) {
	var groups []models.NoteGroup

	if err := r.db.Where("analysis_id = ? ", analysisId).Preload("Notes").Order("time ASC").Find(&groups).Error; err != nil {
		return nil, fmt.Errorf("failed to get note groups by analysis id: %w", err)
	}

	return groups, nil
}

//GetByTimeRange retrieves note group by time range using anaylsis id
func (r *NoteGroupRepository) GetByTimeRange(analysisId uint, startTime, endTime float64) ([]models.NoteGroup, error) {
	var groups []models.NoteGroup

	if err := r.db.Where("analysis_id = ? AND time >= ? AND time <= ? ", analysisId, startTime, endTime).Preload("Notes").Order("time ASC").Find(&groups).Error; err != nil {
		return nil, fmt.Errorf("failed to get a note group by time range: %w", err)
	}

	return groups, nil
}

// Update updates an existing note group
func (r *NoteGroupRepository) Update(group *models.NoteGroup) error {
	if err := r.db.Save(group).Error; err != nil {
		return fmt.Errorf("failed to update note group: %w", err)
	}
	return nil
}

// Delete deletes a note group by ID
func (r *NoteGroupRepository) Delete(id uint) error {
	if err := r.db.Delete(&models.NoteGroup{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete note group: %w", err)
	}
	return nil
}

// DeleteByAnalysisID deletes all note groups for a given analysis
func (r *NoteGroupRepository) DeleteByAnalysisID(analysisID uint) error {
	if err := r.db.Where("analysis_id = ?", analysisID).Delete(&models.NoteGroup{}).Error; err != nil {
		return fmt.Errorf("failed to delete note groups: %w", err)
	}
	return nil
}

// CountByAnalysisID counts total note groups for an analysis
func (r *NoteGroupRepository) CountByAnalysisID(analysisID uint) (int64, error) {
	var count int64
	if err := r.db.Model(&models.NoteGroup{}).Where("analysis_id = ?", analysisID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count note groups: %w", err)
	}
	return count, nil
}