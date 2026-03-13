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

// PromptVersion tracks the evolution of prompts over time
type PromptVersion struct {
	ID                 uint    `gorm:"primaryKey"`
	PromptName         string  `gorm:"index;not null"`    // e.g. "analysis_system", "discovery"
	Version            int     `gorm:"not null"`           // auto-incrementing per prompt_name
	Content            string  `gorm:"type:text;not null"` // the full prompt text
	IsActive           bool    `gorm:"default:false"`      // only one active per prompt_name
	AccuracyAtCreation float64 // accuracy when this version was created
	TotalBets          int     // bets evaluated under this version
	TotalWins          int     // wins under this version
	Accuracy           float64 // current running accuracy
	Reason             string  `gorm:"type:text"` // why this version was created
	CreatedAt          int64
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
	if err := db.AutoMigrate(&Subscriber{}, &SlipStore{}, &PromptVersion{}); err != nil {
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

func (d *Database) GetAnalyticsByDate(dateParam string) (*AnalyticsPayload, error) {
	if !d.IsConnected() {
		return nil, fmt.Errorf("database not connected")
	}

	var optStore SlipStore
	var err error

	if dateParam == "" {
		// Find the most recent record that ends with _optimization
		err = d.db.Where("date_key LIKE ?", "%_optimization").
			Order("date_key DESC").
			First(&optStore).Error
	} else {
		// Find the specific date optimization
		err = d.db.Where("date_key = ?", dateParam+"_optimization").First(&optStore).Error
	}

	if err != nil {
		return nil, err
	}

	// Extract "YYYY-MM-DD"
	prefix := strings.TrimSuffix(optStore.DateKey, "_optimization")
	slipKey := prefix + "_slip"

	var slipStore SlipStore
	err = d.db.Where("date_key = ?", slipKey).First(&slipStore).Error
	if err != nil {
		log.Printf("⚠️ Matching slip %s not found for optimization %s", slipKey, optStore.DateKey)
	}

	return &AnalyticsPayload{
		Optimization: optStore.Data,
		Slip:         slipStore.Data,
	}, nil
}

func (d *Database) GetAnalyticsHistory() ([][]byte, error) {
	if !d.IsConnected() {
		return nil, fmt.Errorf("database not connected")
	}

	var opts []SlipStore
	// Fetch all optimization records ordered by date_key ascending so the charting library renders left-to-right correctly
	err := d.db.Where("date_key LIKE ?", "%_optimization").Order("date_key ASC").Find(&opts).Error
	if err != nil {
		return nil, err
	}

	var results [][]byte
	for _, opt := range opts {
		results = append(results, opt.Data)
	}

	return results, nil
}

// -- Prompt Version Management --

// SavePromptVersion saves a new prompt version and deactivates all previous versions for that prompt name.
func (d *Database) SavePromptVersion(pv *PromptVersion) error {
	if !d.IsConnected() {
		return fmt.Errorf("database not connected")
	}

	return d.db.Transaction(func(tx *gorm.DB) error {
		// Deactivate all existing versions for this prompt
		if err := tx.Model(&PromptVersion{}).
			Where("prompt_name = ? AND is_active = ?", pv.PromptName, true).
			Update("is_active", false).Error; err != nil {
			return err
		}

		// Find the latest version number
		var maxVersion int
		tx.Model(&PromptVersion{}).
			Where("prompt_name = ?", pv.PromptName).
			Select("COALESCE(MAX(version), 0)").
			Scan(&maxVersion)

		pv.Version = maxVersion + 1
		pv.IsActive = true
		pv.CreatedAt = time.Now().Unix()

		return tx.Create(pv).Error
	})
}

// GetActivePrompt returns the currently active prompt version for a given prompt name.
func (d *Database) GetActivePrompt(promptName string) (*PromptVersion, error) {
	if !d.IsConnected() {
		return nil, fmt.Errorf("database not connected")
	}

	var pv PromptVersion
	err := d.db.Where("prompt_name = ? AND is_active = ?", promptName, true).First(&pv).Error
	if err != nil {
		return nil, err
	}
	return &pv, nil
}

// GetPromptHistory returns all prompt versions for a given prompt name, ordered by version descending.
func (d *Database) GetPromptHistory(promptName string) ([]PromptVersion, error) {
	if !d.IsConnected() {
		return nil, fmt.Errorf("database not connected")
	}

	var versions []PromptVersion
	err := d.db.Where("prompt_name = ?", promptName).Order("version DESC").Find(&versions).Error
	return versions, err
}

// UpdatePromptAccuracy updates the running accuracy stats on the current active prompt version.
func (d *Database) UpdatePromptAccuracy(promptName string, totalBets, totalWins int) error {
	if !d.IsConnected() {
		return fmt.Errorf("database not connected")
	}

	var accuracy float64
	if totalBets > 0 {
		accuracy = float64(totalWins) / float64(totalBets)
	}

	return d.db.Model(&PromptVersion{}).
		Where("prompt_name = ? AND is_active = ?", promptName, true).
		Updates(map[string]interface{}{
			"total_bets": gorm.Expr("total_bets + ?", totalBets),
			"total_wins": gorm.Expr("total_wins + ?", totalWins),
			"accuracy":   accuracy,
		}).Error
}

// GetAllPromptHistory returns all prompt versions across all prompt names.
func (d *Database) GetAllPromptHistory() ([]PromptVersion, error) {
	if !d.IsConnected() {
		return nil, fmt.Errorf("database not connected")
	}

	var versions []PromptVersion
	err := d.db.Order("prompt_name ASC, version DESC").Find(&versions).Error
	return versions, err
}

// GetRecentOptimizationReports loads the last N optimization reports from the DB.
func (d *Database) GetRecentOptimizationReports(n int) ([]SlipStore, error) {
	if !d.IsConnected() {
		return nil, fmt.Errorf("database not connected")
	}

	var opts []SlipStore
	err := d.db.Where("date_key LIKE ?", "%_optimization").
		Order("date_key DESC").
		Limit(n).
		Find(&opts).Error
	return opts, err
}
