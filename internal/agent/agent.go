package agent

import (
	"bet-agent/internal/config"
	"bet-agent/internal/email"
	"bet-agent/internal/models"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/genai"
)

// ============================================================================
// BET AGENT
// ============================================================================

type BetAgent struct {
	client      *genai.Client
	config      *config.Config
	emailSender *email.Sender
}

func NewBetAgent(client *genai.Client, config *config.Config, emailSender *email.Sender) *BetAgent {
	return &BetAgent{
		client:      client,
		config:      config,
		emailSender: emailSender,
	}
}

// Load prompt from file
func (b *BetAgent) loadPrompt(filename string) (string, error) {
	path := filepath.Join(b.config.PromptsDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to load prompt %s: %w", filename, err)
	}
	return string(data), nil
}

// Load recipients from JSON file, default to config email if empty/error
func (b *BetAgent) loadRecipients() []models.Recipient {
	path := filepath.Join(b.config.DataDir, "mailing_list.json")
	data, err := os.ReadFile(path)
	if err != nil {
		// If file doesn't exist or error, just return main email
		log.Printf("⚠️ Could not load mailing list (using default): %v", err)
		return []models.Recipient{{Email: b.config.EmailTo}}
	}

	var recipients []models.Recipient
	if err := json.Unmarshal(data, &recipients); err != nil {
		log.Printf("⚠️ Could not parse mailing list (using default): %v", err)
		return []models.Recipient{{Email: b.config.EmailTo}}
	}

	if len(recipients) == 0 {
		return []models.Recipient{{Email: b.config.EmailTo}}
	}

	return recipients
}

// Find today's games
func (b *BetAgent) FindTodaysGames(ctx context.Context) ([]models.Game, error) {
	promptTemplate, err := b.loadPrompt("discovery.txt")
	if err != nil {
		return nil, err
	}

	today := time.Now().Format("Monday, January 2, 2006")
	prompt := fmt.Sprintf(promptTemplate, today)

	tools := []*genai.Tool{
		{GoogleSearch: &genai.GoogleSearch{}},
		{URLContext: &genai.URLContext{}},
	}

	config := &genai.GenerateContentConfig{
		Tools: tools,
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{
				genai.NewPartFromText("You return only valid JSON. No markdown, no explanations."),
			},
		},
	}

	contents := []*genai.Content{
		{
			Role: "user",
			Parts: []*genai.Part{
				genai.NewPartFromText(prompt),
			},
		},
	}

	result, err := b.client.Models.GenerateContent(ctx, b.config.DiscoveryModel, contents, config)
	if err != nil {
		return nil, fmt.Errorf("discovery failed: %w", err)
	}

	jsonStr := extractTextFromResult(result)

	var games []models.Game
	if err := json.Unmarshal([]byte(jsonStr), &games); err != nil {
		return nil, fmt.Errorf("failed to parse games JSON: %w\nResponse: %s", err, jsonStr)
	}

	// Add metadata
	for i := range games {
		games[i].ID = fmt.Sprintf("game_%d_%s", i+1, time.Now().Format("20060102"))
		games[i].Date = time.Now()
	}

	return games, nil
}

// Analyze a single game with thinking
func (b *BetAgent) AnalyzeGame(ctx context.Context, game models.Game) (*models.BetRecommendation, error) {
	promptTemplate, err := b.loadPrompt("analysis.txt")
	if err != nil {
		return nil, err
	}

	prompt := fmt.Sprintf(promptTemplate,
		game.HomeTeam,
		game.AwayTeam,
		game.League,
		game.KickoffTime,
	)

	tools := []*genai.Tool{
		{GoogleSearch: &genai.GoogleSearch{}},
		{URLContext: &genai.URLContext{}},
	}

	systemPrompt, err := b.loadPrompt("analysis_system.txt")
	if err != nil {
		systemPrompt = "You are a professional betting analyst. You think deeply and return only valid JSON."
	}

	config := &genai.GenerateContentConfig{
		Tools: tools,
		ThinkingConfig: &genai.ThinkingConfig{
			ThinkingBudget: genai.Ptr[int32](-1),
		},
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{
				genai.NewPartFromText(systemPrompt),
			},
		},
	}

	contents := []*genai.Content{
		{
			Role: "user",
			Parts: []*genai.Part{
				genai.NewPartFromText(prompt),
			},
		},
	}

	result, err := b.client.Models.GenerateContent(ctx, b.config.AnalysisModel, contents, config)
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	jsonStr := extractTextFromResult(result)

	var rec models.BetRecommendation
	if err := json.Unmarshal([]byte(jsonStr), &rec); err != nil {
		return nil, fmt.Errorf("failed to parse recommendation JSON: %w\nResponse: %s", err, jsonStr)
	}

	rec.Game = game

	return &rec, nil
}

// Optimize based on yesterday's results
func (b *BetAgent) OptimizeFromYesterday(ctx context.Context, yesterdaySlip *models.DailySlip) (*models.OptimizationReport, error) {
	promptTemplate, err := b.loadPrompt("optimization.txt")
	if err != nil {
		return nil, err
	}

	slipJSON, _ := json.MarshalIndent(yesterdaySlip, "", "  ")
	prompt := fmt.Sprintf(promptTemplate, string(slipJSON))

	tools := []*genai.Tool{
		{GoogleSearch: &genai.GoogleSearch{}},
		{URLContext: &genai.URLContext{}},
	}

	systemPrompt, err := b.loadPrompt("optimization_system.txt")
	if err != nil {
		systemPrompt = "You are a performance analyst focused on continuous improvement."
	}

	config := &genai.GenerateContentConfig{
		Tools: tools,
		ThinkingConfig: &genai.ThinkingConfig{
			ThinkingBudget: genai.Ptr[int32](-1),
		},
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{
				genai.NewPartFromText(systemPrompt),
			},
		},
	}

	contents := []*genai.Content{
		{
			Role: "user",
			Parts: []*genai.Part{
				genai.NewPartFromText(prompt),
			},
		},
	}

	result, err := b.client.Models.GenerateContent(ctx, b.config.OptimizationModel, contents, config)
	if err != nil {
		return nil, err
	}

	jsonStr := extractTextFromResult(result)

	var report models.OptimizationReport
	if err := json.Unmarshal([]byte(jsonStr), &report); err != nil {
		return nil, fmt.Errorf("failed to parse optimization JSON: %w\nResponse: %s", err, jsonStr)
	}

	report.Date = time.Now()

	return &report, nil
}

// Extract text from Gemini result
func extractTextFromResult(result *genai.GenerateContentResponse) string {
	if len(result.Candidates) == 0 {
		return ""
	}

	candidate := result.Candidates[0]
	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		return ""
	}

	var textParts []string
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			textParts = append(textParts, part.Text)
		}
	}

	text := strings.Join(textParts, "")

	// Strip markdown code blocks if present
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```json") {
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimSuffix(text, "```")
	} else if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(text, "```")
	}

	return strings.TrimSpace(text)
}

// ============================================================================
// MAIN WORKFLOW
// ============================================================================

func (b *BetAgent) RunMainWorkflow(ctx context.Context) error {
	today := time.Now().Format("2006-01-02")
	log.Println("=" + strings.Repeat("=", 60))
	log.Println("🤖 BET AGENT - MAIN WORKFLOW")
	log.Println("=" + strings.Repeat("=", 60))

	// Step 1: Find games
	log.Println("\n🔍 Step 1: Discovering today's games...")
	games, err := b.FindTodaysGames(ctx)
	if err != nil {
		return fmt.Errorf("failed to find games: %w", err)
	}

	if len(games) == 0 {
		log.Println("❌ No games found for today")
		return nil
	}

	log.Printf("✅ Found %d games\n", len(games))

	// Save games
	gamesFile := filepath.Join(b.config.DataDir, fmt.Sprintf("%s_games.json", today))
	gamesJSON, _ := json.MarshalIndent(games, "", "  ")
	if err := os.WriteFile(gamesFile, gamesJSON, 0644); err != nil {
		return fmt.Errorf("failed to save games: %w", err)
	}
	log.Printf("💾 Saved to: %s\n", gamesFile)

	// Step 2: Analyze each game
	log.Println("\n🔬 Step 2: Analyzing games with AI...")
	recommendations := []models.BetRecommendation{}

	for i, game := range games {
		log.Printf("\n[%d/%d] %s vs %s (%s)\n",
			i+1, len(games), game.HomeTeam, game.AwayTeam, game.League)
		log.Println("  🧠 Researching and analyzing...")

		rec, err := b.AnalyzeGame(ctx, game)
		if err != nil {
			log.Printf("  ❌ Error: %v\n", err)
			continue
		}

		log.Printf("  ✅ %s: %s @ %.2f (%s confidence)\n",
			rec.Market, rec.Selection, rec.Odds, rec.Confidence)

		recommendations = append(recommendations, *rec)

		// Rate limiting
		if i < len(games)-1 {
			log.Printf("  ⏱️  Waiting %d seconds (rate limit)...\n", b.config.RateLimitSeconds)
			time.Sleep(time.Duration(b.config.RateLimitSeconds) * time.Second)
		}
	}

	if len(recommendations) == 0 {
		log.Println("\n❌ No recommendations generated")
		return nil
	}

	// Step 3: Compile slip
	log.Println("\n📋 Step 3: Compiling daily slip...")
	totalOdds := 1.0
	for _, rec := range recommendations {
		totalOdds *= rec.Odds
	}

	slip := &models.DailySlip{
		Date:            time.Now(),
		Recommendations: recommendations,
		TotalOdds:       totalOdds,
		GeneratedAt:     time.Now(),
	}

	slipFile := filepath.Join(b.config.DataDir, fmt.Sprintf("%s_slip.json", today))
	slipJSON, _ := json.MarshalIndent(slip, "", "  ")
	if err := os.WriteFile(slipFile, slipJSON, 0644); err != nil {
		return fmt.Errorf("failed to save slip: %w", err)
	}

	log.Printf("✅ Slip saved: %s\n", slipFile)
	log.Printf("💰 Accumulator odds: %.2f\n", totalOdds)

	// Step 4: Send email
	log.Println("\n📧 Step 4: Sending email...")
	recipients := b.loadRecipients()
	if err := b.emailSender.SendSlipEmail(slip, recipients); err != nil {
		log.Printf("❌ Email failed: %v\n", err)
	} else {
		log.Printf("✅ Emails sent to %d recipients\n", len(recipients))
	}

	log.Println("\n" + strings.Repeat("=", 60))
	log.Println("🎉 Main workflow completed successfully!")
	log.Println(strings.Repeat("=", 60))

	return nil
}

func (b *BetAgent) RunOptimizationWorkflow(ctx context.Context) error {
	log.Println("=" + strings.Repeat("=", 60))
	log.Println("📊 BET AGENT - OPTIMIZATION WORKFLOW")
	log.Println("=" + strings.Repeat("=", 60))

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	yesterdayFile := filepath.Join(b.config.DataDir, fmt.Sprintf("%s_slip.json", yesterday))

	if _, err := os.Stat(yesterdayFile); os.IsNotExist(err) {
		log.Printf("❌ No slip found for yesterday (%s)\n", yesterday)
		return nil
	}

	log.Printf("📂 Loading slip from: %s\n", yesterdayFile)

	data, err := os.ReadFile(yesterdayFile)
	if err != nil {
		return fmt.Errorf("failed to read yesterday's slip: %w", err)
	}

	var yesterdaySlip models.DailySlip
	if err := json.Unmarshal(data, &yesterdaySlip); err != nil {
		return fmt.Errorf("failed to parse yesterday's slip: %w", err)
	}

	log.Println("🧠 Analyzing performance with AI...")
	report, err := b.OptimizeFromYesterday(ctx, &yesterdaySlip)
	if err != nil {
		return fmt.Errorf("optimization failed: %w", err)
	}

	// Save report
	reportFile := filepath.Join(b.config.DataDir, fmt.Sprintf("%s_optimization.json", yesterday))
	reportJSON, _ := json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile(reportFile, reportJSON, 0644); err != nil {
		return fmt.Errorf("failed to save report: %w", err)
	}

	log.Printf("✅ Report saved: %s\n", reportFile)
	log.Printf("📈 Success rate: %.1f%% (%d/%d)\n", report.SuccessRate*100, report.Wins, report.TotalBets)

	// Send email
	log.Println("\n📧 Sending optimization report...")
	recipients := b.loadRecipients()
	if err := b.emailSender.SendOptimizationEmail(report, recipients); err != nil {
		log.Printf("❌ Email failed: %v\n", err)
	} else {
		log.Printf("✅ Emails sent to %d recipients\n", len(recipients))
	}

	log.Println("\n" + strings.Repeat("=", 60))
	log.Println("🎉 Optimization workflow completed!")
	log.Println(strings.Repeat("=", 60))

	return nil
}
