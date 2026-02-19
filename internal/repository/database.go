package repository

import (
	"fmt"

	"github.com/nothim0/fretboardAI-Go-backend/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Database struct {
	DB *gorm.DB
}

func NewDatabse(connStr string) (*Database, error) {
	db, err := gorm.Open(sqlite.Open(connStr), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		return nil, fmt.Errorf("Failed to connect to database %w", err)
	}

	dbConn, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("Failed to get underlying database %w", err)
	}

	dbConn.SetMaxOpenConns(10)
	dbConn.SetMaxIdleConns(5)

	if _, err = dbConn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("Failed to enable WAL mode %w", err)
	}

	if err := db.AutoMigrate(
		&models.Job{},
		&models.Analysis{},
		&models.Note{},
		&models.NoteGroup{},
	); err != nil {
		return nil, fmt.Errorf("Failed to migrate database %w", err)
	}

	fmt.Println("database initialised successfully!!")

	return &Database{DB: db}, nil
}

func (d *Database) Close() error {
	dbConn, err := d.DB.DB()
	if err != nil {
		return err
	}
	return dbConn.Close()
}
