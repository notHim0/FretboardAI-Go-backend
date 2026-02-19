package repository

import (
	"fmt"

	"github.com/nothim0/fretboardAI-Go-backend/internal/models"
	"gorm.io/gorm"
)

type AnalysisRepository struct {
	db *gorm.DB
}

// creates a new analysis repository
func NewAnalysisRepository(db *gorm.DB) *AnalysisRepository {
	return &AnalysisRepository{db: db}
}

// creates a new analysis
func (r *AnalysisRepository) Create(analysis *models.Analysis) error {
	if err := r.db.Create(analysis).Error; err != nil {
		return fmt.Errorf("failed to create analysis: %w", err)
	}
	return nil
}

// GetById retrieves a analysis by Id
func (r *AnalysisRepository) GetById(id uint) (*models.Analysis, error) {
	var analysis models.Analysis

	if err := r.db.Preload("Job").First(&analysis, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("analysis not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get analysis: %w", err)
	}

	return &analysis, nil
}

// GetByJobId retrieves a analysis by jobId
func (r *AnalysisRepository) GetByJobId(jobId uint) (*models.Analysis, error) {
	var analysis models.Analysis

	if err := r.db.Where("job_id = ?", jobId).Preload("Job").First(&analysis).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("analysis not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get analysis: %w", err)
	}

	return &analysis, nil
}

// Update updates the analysis
func (r *AnalysisRepository) Update(analysis *models.Analysis) error {
	if err := r.db.Save(analysis).Error; err != nil {
		return fmt.Errorf("failed to update analysis: %w", err)
	}
	return nil
}

func (r *AnalysisRepository) Delete(id uint) error {
	if err := r.db.Delete(&models.Analysis{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete analysis: %w", err)
	}
	return nil
}
