package main

import (
	"bet-agent/internal/config"
	"bet-agent/internal/db"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	log.Println("🚀 Starting Leapcell Data Migration...")

	// 1. Load config (needs DATABASE_URL)
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	if cfg.DatabaseURL == "" {
		log.Fatal("❌ DATABASE_URL is not set. Cannot run migration.")
	}

	// 2. Connect to Database (Initializes GORM and AutoMigrates tables)
	database, err := db.NewDatabase(cfg)
	if err != nil {
		log.Fatalf("❌ Failed to connect to DB: %v", err)
	}

	// 3. Scan the local `data/` directory
	dataDir := "data"
	files, err := os.ReadDir(dataDir)
	if err != nil {
		log.Fatalf("❌ Failed to read data directory: %v", err)
	}

	insertedCount := 0

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		// Skip mailing list as it was migrated differently
		if file.Name() == "mailing_list.json" {
			continue
		}

		filePath := filepath.Join(dataDir, file.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("⚠️ Failed to read %s: %v", file.Name(), err)
			continue
		}

		// Calculate DateKey exactly how agent.go does it
		// e.g., "2026-02-19_slip.json" -> "2026-02-19_slip"
		dateKey := strings.TrimSuffix(file.Name(), ".json")

		log.Printf("📥 Migrating %s...", dateKey)

		// SaveJSON handles UPSERT (Update if exists, Insert if new)
		err = database.SaveJSON(dateKey, content)
		if err != nil {
			log.Printf("❌ Failed to save %s to DB: %v", dateKey, err)
			continue
		}

		insertedCount++
		log.Printf("✅ Saved %s", dateKey)
	}

	fmt.Println(strings.Repeat("=", 50))
	log.Printf("🎉 Migration Complete! Successfully migrated %d files.", insertedCount)
}
