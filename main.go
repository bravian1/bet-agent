package main

import (
	"bet-agent/internal/agent"
	"bet-agent/internal/config"
	"bet-agent/internal/email"
	"bet-agent/internal/server"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/robfig/cron/v3"
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

	// Create dependencies
	emailSender := email.NewSender(cfg)
	betAgent := agent.NewBetAgent(client, cfg, emailSender)

	// Start web server
	srv := server.NewServer(cfg)
	go func() {
		if err := srv.Start(); err != nil {
			log.Printf("❌ Web server error: %v", err)
		}
	}()

	// Command-line flags
	optimizeFlag := flag.Bool("optimize", false, "Run the optimization workflow immediately and exit")
	flag.Parse()

	if *optimizeFlag {
		log.Println("🚀 Running optimization workflow manually (--optimize flag detected)")
		if err := betAgent.RunOptimizationWorkflow(ctx); err != nil {
			log.Fatalf("❌ Optimization workflow error: %v", err)
		}
		log.Println("✅ Optimization workflow completed.")
		return
	}

	// Setup cron scheduler
	c := cron.New()

	// Main workflow
	log.Printf("📅 Scheduling main workflow: %s\n", cfg.MainCron)
	_, err = c.AddFunc(cfg.MainCron, func() {
		log.Println("\n🔔 Main workflow triggered by cron")
		if err := betAgent.RunMainWorkflow(context.Background()); err != nil {
			log.Printf("❌ Main workflow error: %v\n", err)
		}
	})
	if err != nil {
		log.Fatalf("Failed to schedule main workflow: %v", err)
	}

	// Optimization workflow
	log.Printf("📅 Scheduling optimization workflow: %s\n", cfg.OptimizationCron)
	_, err = c.AddFunc(cfg.OptimizationCron, func() {
		log.Println("\n🔔 Optimization workflow triggered by cron")
		if err := betAgent.RunOptimizationWorkflow(context.Background()); err != nil {
			log.Printf("❌ Optimization workflow error: %v\n", err)
		}
	})
	if err != nil {
		log.Fatalf("Failed to schedule optimization workflow: %v", err)
	}

	// Start cron
	c.Start()

	log.Println("\n✅ Bet Agent is running!")
	log.Println("📍 Press Ctrl+C to stop")
	log.Println("")

	// Run immediately on startup (optional - comment out if you only want cron)
	log.Println("🚀 Running main workflow immediately on startup...")
	if err := betAgent.RunMainWorkflow(ctx); err != nil {
		log.Printf("❌ Startup workflow error: %v\n", err)
	}

	// Keep running
	select {}
}
