package repository

import (
	"fmt"
	"time"

	"github.com/nothim0/fretboardAI-Go-backend/internal/models"
	"gorm.io/gorm"
)


type JobRepository struct {
	db *gorm.DB
}

func NewJobRepository (db *gorm.DB)  *JobRepository{
	return &JobRepository{db: db}
}

func (r *JobRepository) Create(filename string) (*models.Job, error){
	var job *models.Job = &models.Job{
		Filename: filename,
		Status: "pending",
		CreatedAt: time.Now(),
	}

	if err := r.db.Create(job).Error; err != nil {
		return nil, fmt.Errorf("Unable to create job: %w", err);
	}

	return job,nil
}

func (r *JobRepository) GetById(id uint) (*models.Job, error) {
	var job models.Job
	if err := r.db.First(&job, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("Job entry not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return &job, nil
}

func (r *JobRepository) UpdateStatus(status string, id uint, errMsg error) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if status == "completed" || status == "failed" {
		now := time.Now()
		updates["completed_at"] = &now
	}

	if errMsg != nil {
		updates["error_message"] = errMsg
	}

	if err := r.db.Model(&models.Job{}).Where("id: ?", id).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}
	return nil
}

func (r *JobRepository) Delete(id uint) error {
	if err := r.db.Delete(&models.Job{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	return nil
}

func (r *JobRepository) List(offset int, limit int) ([]models.Job, error) {
	var jobs []models.Job

	if err := r.db.Order("created_at DESC").Limit(limit).Offset(offset).Find(&jobs).Error; err != nil {
		return nil, fmt.Errorf("Failed to list jobs: %w", err)
	}

	return jobs, nil
}

func (r *JobRepository) GetPendingJobs() ([]models.Job, error) {
	var pendingJobs []models.Job

	if err := r.db.Where("status = ?", "pending").Order("created_at ASC").Find(&pendingJobs).Error; err != nil {
		return nil, fmt.Errorf("failed to get pending jobs: %w", err)
	}

	return pendingJobs, nil
}