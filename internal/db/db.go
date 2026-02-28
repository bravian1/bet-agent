package db

import (
	"bet-agent/internal/config"
	"bet-agent/internal/models"
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Database struct {
	db *gorm.DB
}

// Subscriber model for GORM
type Subscriber struct {
	ID        uint   `gorm:"primaryKey"`
	Email     string `gorm:"uniqueIndex;not null"`
	CreatedAt int64  // Unix timestamp
}

// SlipStore model for storing JSON reports in the DB
type SlipStore struct {
	ID        uint   `gorm:"primaryKey"`
	DateKey   string `gorm:"uniqueIndex;not null"` // e.g. "2006-01-02_slip" or "2006-01-02_opt"
	Data      []byte // JSON serialized data
	CreatedAt int64
}

func NewDatabase(cfg *config.Config) (*Database, error) {
	if cfg.DatabaseURL == "" {
		log.Println("⚠️ DATABASE_URL not set. Running without a database connection (some actions may fail).")
		return &Database{}, nil
	}

	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set custom schema required by Leapcell
	if err := db.Exec("SET search_path TO myschema").Error; err != nil {
		log.Printf("⚠️ Failed to set schema 'myschema': %v", err)
	}

	// Auto-migrate the schemas
	if err := db.AutoMigrate(&Subscriber{}, &SlipStore{}); err != nil {
		return nil, fmt.Errorf("failed to auto-migrate database: %w", err)
	}

	log.Println("✅ Connected to database via GORM")
	return &Database{db: db}, nil
}

func (d *Database) IsConnected() bool {
	return d.db != nil
}

// -- Subscribers --

func (d *Database) AddSubscriber(email string) error {
	if !d.IsConnected() {
		return fmt.Errorf("database not connected")
	}
	sub := Subscriber{Email: email, CreatedAt: time.Now().Unix()}
	// Ignore duplicate key errors if email already exists
	return d.db.Where(Subscriber{Email: email}).FirstOrCreate(&sub).Error
}

func (d *Database) GetSubscribers() ([]models.Recipient, error) {
	if !d.IsConnected() {
		return nil, fmt.Errorf("database not connected")
	}
	var subs []Subscriber
	if err := d.db.Find(&subs).Error; err != nil {
		return nil, err
	}

	var recipients []models.Recipient
	for _, sub := range subs {
		recipients = append(recipients, models.Recipient{
			Email: sub.Email,
			// Simplified CreatedAt logic since time isn't strictly necessary for Recipient model
		})
	}
	return recipients, nil
}

func (d *Database) SubscriberExists(email string) (bool, error) {
	if !d.IsConnected() {
		return false, fmt.Errorf("database not connected")
	}
	var count int64
	err := d.db.Model(&Subscriber{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

// -- Slips and Reports --

func (d *Database) SaveJSON(dateKey string, data []byte) error {
	if !d.IsConnected() {
		return fmt.Errorf("database not connected")
	}
	store := SlipStore{
		DateKey:   dateKey,
		Data:      data,
		CreatedAt: time.Now().Unix(),
	}
	// Use Save to update if exists (though DateKey is unique so we can also use OnConflict)
	return d.db.Where(SlipStore{DateKey: dateKey}).Assign(SlipStore{Data: data}).FirstOrCreate(&store).Error
}

func (d *Database) LoadJSON(dateKey string) ([]byte, error) {
	if !d.IsConnected() {
		return nil, fmt.Errorf("database not connected")
	}
	var store SlipStore
	if err := d.db.Where("date_key = ?", dateKey).First(&store).Error; err != nil {
		return nil, err
	}
	return store.Data, nil
}

type AnalyticsPayload struct {
	Optimization []byte
	Slip         []byte
}

func (d *Database) GetLatestAnalytics() (*AnalyticsPayload, error) {
	if !d.IsConnected() {
		return nil, fmt.Errorf("database not connected")
	}

	var latestOpt SlipStore
	// Find the most recent record that ends with _optimization
	err := d.db.Where("date_key LIKE ?", "%_optimization").
		Order("created_at DESC").
		First(&latestOpt).Error

	if err != nil {
		return nil, err
	}

	// Assuming the key is "YYYY-MM-DD_optimization", we extract "YYYY-MM-DD"
	prefix := strings.TrimSuffix(latestOpt.DateKey, "_optimization")
	slipKey := prefix + "_slip"

	var slipStore SlipStore
	err = d.db.Where("date_key = ?", slipKey).First(&slipStore).Error
	if err != nil {
		// It's possible the slip is missing, but return what we have
		log.Printf("⚠️ Matching slip %s not found for optimization %s", slipKey, latestOpt.DateKey)
	}

	return &AnalyticsPayload{
		Optimization: latestOpt.Data,
		Slip:         slipStore.Data,
	}, nil
}
