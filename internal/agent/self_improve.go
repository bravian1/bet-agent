package agent

import (
	"bet-agent/internal/db"
	"bet-agent/internal/models"
	"bet-agent/prompts"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"google.golang.org/genai"
)

// promptNames lists the prompts that the self-improvement system can evolve.
var promptNames = []string{"analysis_system", "discovery"}

// promptFileMap maps prompt names to their corresponding .txt files.
var promptFileMap = map[string]string{
	"analysis_system": "analysis_system.txt",
	"discovery":       "discovery.txt",
}

// ============================================================================
// SELF-IMPROVEMENT WORKFLOW
// ============================================================================

// RunSelfImprovementWorkflow is the main orchestrator for the self-improvement loop.
// It checks rolling accuracy and, if below the target, generates improved prompts.
func (b *BetAgent) RunSelfImprovementWorkflow(ctx context.Context) ([]models.SelfImprovementResult, error) {
	log.Println("=" + strings.Repeat("=", 60))
	log.Println("🧬 BET AGENT - SELF-IMPROVEMENT WORKFLOW")
	log.Println("=" + strings.Repeat("=", 60))

	targetAccuracy := b.config.TargetAccuracy
	windowDays := b.config.RollingWindowDays

	// Step 1: Load recent optimization reports
	log.Printf("\n📊 Step 1: Loading last %d days of optimization reports...\n", windowDays)
	recentReports, err := b.database.GetRecentOptimizationReports(windowDays)
	if err != nil {
		return nil, fmt.Errorf("failed to load recent reports: %w", err)
	}

	if len(recentReports) < 2 {
		log.Println("⚠️ Not enough historical data (need at least 2 days). Skipping self-improvement.")
		return []models.SelfImprovementResult{{
			SkippedReason: fmt.Sprintf("Insufficient data: only %d reports available (need >= 2)", len(recentReports)),
			Improved:      false,
		}}, nil
	}

	// Step 2: Calculate rolling accuracy
	log.Println("\n📈 Step 2: Calculating rolling accuracy...")
	rollingAccuracy, totalBets, totalWins, performanceSummary, improvementSuggestions := b.analyzeRecentPerformance(recentReports)

	log.Printf("   Rolling accuracy: %.1f%% (%d/%d bets over %d reports)\n",
		rollingAccuracy*100, totalWins, totalBets, len(recentReports))
	log.Printf("   Target: %.1f%%\n", targetAccuracy*100)

	if rollingAccuracy >= targetAccuracy {
		log.Printf("✅ Already meeting target accuracy (%.1f%% >= %.1f%%). No changes needed.\n",
			rollingAccuracy*100, targetAccuracy*100)
		return []models.SelfImprovementResult{{
			RollingAccuracy: rollingAccuracy,
			TargetAccuracy:  targetAccuracy,
			SkippedReason:   fmt.Sprintf("Already meeting target: %.1f%% >= %.1f%%", rollingAccuracy*100, targetAccuracy*100),
			Improved:        false,
		}}, nil
	}

	// Step 3: Improve each prompt
	log.Println("\n🧠 Step 3: Generating improved prompts...")
	var results []models.SelfImprovementResult

	for _, promptName := range promptNames {
		log.Printf("\n   🔧 Improving prompt: %s\n", promptName)
		result, err := b.improvePrompt(ctx, promptName, rollingAccuracy, targetAccuracy,
			performanceSummary, improvementSuggestions, len(recentReports))
		if err != nil {
			log.Printf("   ❌ Failed to improve %s: %v\n", promptName, err)
			results = append(results, models.SelfImprovementResult{
				PromptName:    promptName,
				SkippedReason: fmt.Sprintf("Error: %v", err),
				Improved:      false,
			})
			continue
		}
		results = append(results, *result)
	}

	log.Println("\n" + strings.Repeat("=", 60))
	log.Println("🎉 Self-improvement workflow completed!")
	log.Println(strings.Repeat("=", 60))

	return results, nil
}

// analyzeRecentPerformance parses recent optimization reports and calculates aggregate stats.
func (b *BetAgent) analyzeRecentPerformance(reports []db.SlipStore) (
	rollingAccuracy float64,
	totalBets int,
	totalWins int,
	performanceSummary string,
	improvementSuggestions string,
) {
	var summaryParts []string
	var suggestionParts []string

	for _, report := range reports {
		var opt models.OptimizationReport
		if err := json.Unmarshal(report.Data, &opt); err != nil {
			log.Printf("⚠️ Failed to parse report %s: %v", report.DateKey, err)
			continue
		}

		totalBets += opt.TotalBets
		totalWins += opt.Wins

		dateStr := strings.TrimSuffix(report.DateKey, "_optimization")
		summaryParts = append(summaryParts, fmt.Sprintf(
			"[%s] %d/%d bets won (%.0f%%). Winners: %s. Losers: %s. Insights: %s",
			dateStr, opt.Wins, opt.TotalBets, opt.SuccessRate*100,
			strings.Join(opt.WinningMarkets, ", "),
			strings.Join(opt.LosingMarkets, ", "),
			opt.Insights,
		))

		if opt.PromptImprovements != "" {
			suggestionParts = append(suggestionParts, fmt.Sprintf("[%s] %s", dateStr, opt.PromptImprovements))
		}
	}

	if totalBets > 0 {
		rollingAccuracy = float64(totalWins) / float64(totalBets)
	}

	performanceSummary = strings.Join(summaryParts, "\n\n")
	improvementSuggestions = strings.Join(suggestionParts, "\n\n")

	if improvementSuggestions == "" {
		improvementSuggestions = "(No specific suggestions accumulated yet)"
	}

	return
}

// improvePrompt generates an improved version of a specific prompt using AI.
func (b *BetAgent) improvePrompt(
	ctx context.Context,
	promptName string,
	rollingAccuracy, targetAccuracy float64,
	performanceSummary, improvementSuggestions string,
	reportCount int,
) (*models.SelfImprovementResult, error) {
	// Load current prompt content (from DB if available, else from file)
	currentContent, currentVersion, err := b.loadCurrentPrompt(promptName)
	if err != nil {
		return nil, fmt.Errorf("failed to load current prompt: %w", err)
	}

	// Build the meta-optimization prompt
	promptTemplate, err := prompts.Get("self_improve.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to load self_improve template: %w", err)
	}

	prompt := fmt.Sprintf(promptTemplate,
		rollingAccuracy*100,
		reportCount,
		targetAccuracy*100,
		promptName,
		currentContent,
		reportCount,
		performanceSummary,
		improvementSuggestions,
	)

	systemPrompt, err := prompts.Get("self_improve_system.txt")
	if err != nil {
		systemPrompt = "You are an expert prompt engineer. Improve the given prompt based on performance data. Return only valid JSON."
	}

	// Call Gemini
	config := &genai.GenerateContentConfig{
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

	result, err := b.client.Models.GenerateContent(ctx, b.config.SelfImproveModel, contents, config)
	if err != nil {
		return nil, fmt.Errorf("AI generation failed: %w", err)
	}

	jsonStr := extractTextFromResult(result)

	// Parse the response
	var aiResponse struct {
		ImprovedPrompt string   `json:"improved_prompt"`
		ChangeSummary  string   `json:"change_summary"`
		ChangesMade    []string `json:"changes_made"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &aiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w\nResponse: %s", err, jsonStr)
	}

	if aiResponse.ImprovedPrompt == "" {
		return nil, fmt.Errorf("AI returned an empty improved prompt")
	}

	// Save the new version to DB
	pv := &db.PromptVersion{
		PromptName:         promptName,
		Content:            aiResponse.ImprovedPrompt,
		AccuracyAtCreation: rollingAccuracy,
		Reason:             aiResponse.ChangeSummary,
		CreatedAt:          time.Now().Unix(),
	}

	if err := b.database.SavePromptVersion(pv); err != nil {
		return nil, fmt.Errorf("failed to save new prompt version: %w", err)
	}

	log.Printf("   ✅ Saved new version %d for %s\n", pv.Version, promptName)
	log.Printf("   📝 Changes: %s\n", aiResponse.ChangeSummary)

	return &models.SelfImprovementResult{
		PromptName:       promptName,
		PreviousVersion:  currentVersion,
		NewVersion:       pv.Version,
		PreviousAccuracy: rollingAccuracy,
		RollingAccuracy:  rollingAccuracy,
		TargetAccuracy:   targetAccuracy,
		ChangeSummary:    aiResponse.ChangeSummary,
		NewPromptContent: aiResponse.ImprovedPrompt,
		Improved:         true,
	}, nil
}

// loadCurrentPrompt loads the current prompt content, preferring DB version over embedded file.
func (b *BetAgent) loadCurrentPrompt(promptName string) (content string, version int, err error) {
	// Try DB first
	if b.database.IsConnected() {
		pv, dbErr := b.database.GetActivePrompt(promptName)
		if dbErr == nil && pv != nil {
			return pv.Content, pv.Version, nil
		}
	}

	// Fall back to embedded file
	filename, ok := promptFileMap[promptName]
	if !ok {
		return "", 0, fmt.Errorf("unknown prompt name: %s", promptName)
	}

	content, err = prompts.Get(filename)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read embedded prompt %s: %w", filename, err)
	}

	return content, 0, nil
}

// GetActivePromptContent returns the active prompt for use in analysis/discovery,
// preferring the DB-stored evolved version over the embedded file.
func (b *BetAgent) GetActivePromptContent(promptName string) (string, error) {
	content, _, err := b.loadCurrentPrompt(promptName)
	return content, err
}
