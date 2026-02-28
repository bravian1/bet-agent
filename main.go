package main

import (
	"bet-agent/internal/agent"
	"bet-agent/internal/config"
	"bet-agent/internal/db"
	"bet-agent/internal/email"
	"bet-agent/internal/server"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"google.golang.org/genai"
)

// ============================================================================
// SETUP
// ============================================================================

func setupDirectories(cfg *config.Config) error {
	// Create data directory
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Create prompts directory
	if err := os.MkdirAll(cfg.PromptsDir, 0755); err != nil {
		return fmt.Errorf("failed to create prompts directory: %w", err)
	}

	// Create default prompt files if they don't exist
	for filename, content := range agent.DefaultPrompts {
		path := filepath.Join(cfg.PromptsDir, filename)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to create prompt file %s: %w", filename, err)
			}
		}
	}

	return nil
}

// ============================================================================
// MAIN
// ============================================================================

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Setup directories and default prompts
	if err := setupDirectories(cfg); err != nil {
		log.Fatalf("Setup error: %v", err)
	}

	// Set Gemini API key in environment
	os.Setenv("GOOGLE_GENAI_API_KEY", cfg.GeminiAPIKey)

	// Create Gemini client
	ctx := context.Background()
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to create Gemini client: %v", err)
	}

	// Initialize Database
	database, err := db.NewDatabase(cfg)
	if err != nil {
		log.Fatalf("Database connection error: %v", err)
	}

	// Create dependencies
	emailSender := email.NewSender(cfg)
	betAgent := agent.NewBetAgent(client, cfg, emailSender, database)

	// Start web server (entry point for both subscriptions and cron webhooks)
	srv := server.NewServer(cfg, database, betAgent)
	go func() {
		if err := srv.Start(); err != nil {
			log.Printf("❌ Web server error: %v", err)
		}
	}()

	// Command-line flags
	optimizeFlag := flag.Bool("optimize", false, "Run the optimization workflow immediately and exit")
	flag.Parse()

	// Also check positional arguments
	shouldOptimize := *optimizeFlag
	if !shouldOptimize && flag.NArg() > 0 && flag.Arg(0) == "optimize" {
		shouldOptimize = true
	}

	if shouldOptimize {
		log.Println("🚀 Running optimization workflow manually (optimize command detected)")
		if err := betAgent.RunOptimizationWorkflow(ctx); err != nil {
			log.Fatalf("❌ Optimization workflow error: %v", err)
		}
		log.Println("✅ Optimization workflow completed.")
		return
	}

	// Remove the internal cron functionality - Leapcell or GitHub Actions will
	// trigger /api/internal/run-main and /api/internal/run-optimization

	log.Println("\n✅ Bet Agent is running!")
	log.Println("📍 Waiting for HTTP requests...")
	log.Println("")

	// Keep running
	select {}
}
