package main

import (
	"bet-agent/internal/agent"
	"bet-agent/internal/config"
	"bet-agent/internal/db"
	"bet-agent/internal/email"
	"bet-agent/internal/server"
	"context"
	"flag"
	"log"
	"os"

	"google.golang.org/genai"
)

// ============================================================================
// MAIN
// ============================================================================

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
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
	selfImproveFlag := flag.Bool("self-improve", false, "Run the self-improvement workflow immediately and exit")
	flag.Parse()

	// Also check positional arguments
	shouldOptimize := *optimizeFlag
	shouldSelfImprove := *selfImproveFlag
	if !shouldOptimize && !shouldSelfImprove && flag.NArg() > 0 {
		switch flag.Arg(0) {
		case "optimize":
			shouldOptimize = true
		case "self-improve":
			shouldSelfImprove = true
		}
	}

	if shouldOptimize {
		log.Println("🚀 Running optimization workflow manually (optimize command detected)")
		if err := betAgent.RunOptimizationWorkflow(ctx); err != nil {
			log.Fatalf("❌ Optimization workflow error: %v", err)
		}
		log.Println("✅ Optimization workflow completed.")
		return
	}

	if shouldSelfImprove {
		log.Println("🧬 Running self-improvement workflow manually (self-improve command detected)")
		results, err := betAgent.RunSelfImprovementWorkflow(ctx)
		if err != nil {
			log.Fatalf("❌ Self-improvement workflow error: %v", err)
		}
		for _, r := range results {
			if r.Improved {
				log.Printf("✅ %s: v%d → v%d (%.1f%% accuracy). Changes: %s\n",
					r.PromptName, r.PreviousVersion, r.NewVersion, r.RollingAccuracy*100, r.ChangeSummary)
			} else if r.SkippedReason != "" {
				log.Printf("⏭️  %s: Skipped — %s\n", r.PromptName, r.SkippedReason)
			}
		}
		log.Println("✅ Self-improvement workflow completed.")
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
