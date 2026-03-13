package models

import "time"

// ============================================================================
// DATA STRUCTURES
// ============================================================================

type Game struct {
	ID          string    `json:"id"`
	Date        time.Time `json:"date"`
	HomeTeam    string    `json:"home_team"`
	AwayTeam    string    `json:"away_team"`
	League      string    `json:"league"`
	KickoffTime string    `json:"kickoff_time"`
}

type BetRecommendation struct {
	Game       Game     `json:"game"`
	Market     string   `json:"market"`
	Selection  string   `json:"selection"`
	Odds       float64  `json:"odds"`
	Confidence string   `json:"confidence"`
	Reasoning  string   `json:"reasoning"`
	KeyFactors []string `json:"key_factors"`
}

type DailySlip struct {
	Date            time.Time           `json:"date"`
	Recommendations []BetRecommendation `json:"recommendations"`
	TotalOdds       float64             `json:"total_odds"`
	GeneratedAt     time.Time           `json:"generated_at"`
}

type OptimizationReport struct {
	Date               time.Time `json:"date"`
	TotalBets          int       `json:"total_bets"`
	Wins               int       `json:"wins"`
	Losses             int       `json:"losses"`
	SuccessRate        float64   `json:"success_rate"`
	WinningMarkets     []string  `json:"winning_markets"`
	LosingMarkets      []string  `json:"losing_markets"`
	Insights           string    `json:"insights"`
	PromptImprovements string    `json:"prompt_improvements"`
}

type SelfImprovementResult struct {
	PromptName       string  `json:"prompt_name"`
	PreviousVersion  int     `json:"previous_version"`
	NewVersion       int     `json:"new_version"`
	PreviousAccuracy float64 `json:"previous_accuracy"`
	RollingAccuracy  float64 `json:"rolling_accuracy"`
	TargetAccuracy   float64 `json:"target_accuracy"`
	ChangeSummary    string  `json:"change_summary"`
	NewPromptContent string  `json:"new_prompt_content"`
	SkippedReason    string  `json:"skipped_reason,omitempty"`
	Improved         bool    `json:"improved"`
}

type Recipient struct {
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}
