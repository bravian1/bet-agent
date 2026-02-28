package email

import (
	"bet-agent/internal/config"
	"bet-agent/internal/models"
	"fmt"
	"log"
	"strings"

	"github.com/resend/resend-go/v3"
)

// ============================================================================
// EMAIL FUNCTIONALITY
// ============================================================================

type Sender struct {
	config *config.Config
}

func NewSender(cfg *config.Config) *Sender {
	return &Sender{
		config: cfg,
	}
}

func (s *Sender) SendSlipEmail(slip *models.DailySlip, recipients []models.Recipient) error {
	subject := fmt.Sprintf("🎯 Daily Betting Slip - %s", slip.Date.Format("Monday, Jan 2, 2006"))
	body := s.formatSlipAsHTML(slip)

	return s.SendToAll(recipients, subject, body)
}

func (s *Sender) SendOptimizationEmail(report *models.OptimizationReport, recipients []models.Recipient) error {
	subject := fmt.Sprintf("📊 Performance Report - %s", report.Date.Format("Monday, Jan 2, 2006"))
	body := s.formatOptimizationAsHTML(report)

	return s.SendToAll(recipients, subject, body)
}

func (s *Sender) SendToAll(recipients []models.Recipient, subject, htmlBody string) error {
	if s.config.ResendAPIKey == "" {
		return fmt.Errorf("RESEND_API_KEY is not configured")
	}

	client := resend.NewClient(s.config.ResendAPIKey)

	// If no recipients, default to config email
	if len(recipients) == 0 {
		recipients = []models.Recipient{{Email: s.config.EmailTo}}
	}

	// Extract email strings
	var toEmails []string
	for _, r := range recipients {
		toEmails = append(toEmails, r.Email)
	}

	params := &resend.SendEmailRequest{
		From:    s.config.EmailFrom,
		To:      toEmails, // Resend handles multiple recipients
		Subject: subject,
		Html:    htmlBody,
	}

	log.Printf("📧 Sending bulk email via Resend to %d recipients...", len(toEmails))

	// Resend handles the "queue" logic internally when we give it multiple TO addresses
	// Note: for more than 50 recipients, it's better to use the Batch API, but this is fine for now.
	sent, err := client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send emails via Resend: %w", err)
	}

	log.Printf("✅ Resend operation completed. ID: %s", sent.Id)
	return nil
}

// Removed manual sendEmail using net/smtp

func (s *Sender) formatSlipAsHTML(slip *models.DailySlip) string {
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        h1 { color: #2c3e50; border-bottom: 3px solid #3498db; padding-bottom: 10px; }
        h2 { color: #34495e; margin-top: 30px; }
        .summary { background: #ecf0f1; padding: 15px; border-radius: 5px; margin: 20px 0; }
        .bet { background: white; border: 1px solid #bdc3c7; padding: 15px; margin: 15px 0; border-radius: 5px; }
        .high { border-left: 5px solid #27ae60; }
        .medium { border-left: 5px solid #f39c12; }
        .low { border-left: 5px solid #e74c3c; }
        .label { font-weight: bold; color: #7f8c8d; }
        .value { color: #2c3e50; }
        .odds { font-size: 1.2em; color: #3498db; font-weight: bold; }
        .reasoning { font-style: italic; color: #555; margin-top: 10px; }
        .factors { margin-top: 10px; }
        .factor { color: #16a085; margin-left: 20px; }
    </style>
</head>
<body>
    <h1>🎯 Daily Betting Slip</h1>
    <div class="summary">
        <p><span class="label">Date:</span> <span class="value">%s</span></p>
        <p><span class="label">Total Bets:</span> <span class="value">%d</span></p>
        <p><span class="label">Accumulator Odds:</span> <span class="odds">%.2f</span></p>
        <p><span class="label">Generated:</span> <span class="value">%s</span></p>
    </div>
    <h2>📋 Recommendations</h2>
`, slip.Date.Format("Monday, January 2, 2006"), len(slip.Recommendations), slip.TotalOdds, slip.GeneratedAt.Format("15:04 MST"))

	for i, rec := range slip.Recommendations {
		confidenceClass := strings.ToLower(rec.Confidence)
		if strings.Contains(confidenceClass, "high") {
			confidenceClass = "high"
		} else if strings.Contains(confidenceClass, "medium") {
			confidenceClass = "medium"
		} else if strings.Contains(confidenceClass, "low") {
			confidenceClass = "low"
		} else {
			confidenceClass = "medium" // Default
		}

		html += fmt.Sprintf(`
    <div class="bet %s">
        <h3>%d. %s vs %s</h3>
        <p><span class="label">League:</span> <span class="value">%s</span> | <span class="label">Kickoff:</span> <span class="value">%s</span></p>
        <p><span class="label">Market:</span> <span class="value">%s</span></p>
        <p><span class="label">Selection:</span> <span class="value">%s</span> <span class="odds">@ %.2f</span></p>
        <p><span class="label">Confidence:</span> <span class="value">%s</span></p>
        <div class="reasoning">%s</div>
`, confidenceClass, i+1, rec.Game.HomeTeam, rec.Game.AwayTeam, rec.Game.League, rec.Game.KickoffTime,
			rec.Market, rec.Selection, rec.Odds, rec.Confidence, rec.Reasoning)

		if len(rec.KeyFactors) > 0 {
			html += `        <div class="factors"><strong>Key Factors:</strong>`
			for _, factor := range rec.KeyFactors {
				html += fmt.Sprintf(`<div class="factor">• %s</div>`, factor)
			}
			html += `        </div>`
		}

		html += `    </div>`
	}

	html += `
</body>
</html>`

	return html
}

func (s *Sender) formatOptimizationAsHTML(report *models.OptimizationReport) string {
	successColor := "#27ae60"
	if report.SuccessRate < 0.5 {
		successColor = "#e74c3c"
	} else if report.SuccessRate < 0.7 {
		successColor = "#f39c12"
	}

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        h1 { color: #2c3e50; border-bottom: 3px solid #3498db; padding-bottom: 10px; }
        .summary { background: #ecf0f1; padding: 20px; border-radius: 5px; margin: 20px 0; }
        .stat { font-size: 2em; font-weight: bold; margin: 10px 0; }
        .insights { background: white; border-left: 5px solid #3498db; padding: 15px; margin: 20px 0; }
        .improvements { background: #fff3cd; border-left: 5px solid #ffc107; padding: 15px; margin: 20px 0; }
        .label { font-weight: bold; color: #7f8c8d; }
        ul { line-height: 1.8; }
    </style>
</head>
<body>
    <h1>📊 Performance Analysis</h1>
    <div class="summary">
        <p><span class="label">Analysis Date:</span> %s</p>
        <p><span class="label">Total Bets:</span> %d</p>
        <p><span class="label">Wins:</span> %d | <span class="label">Losses:</span> %d</p>
        <p class="stat" style="color: %s;">Success Rate: %.1f%%</p>
    </div>
    <div class="insights">
        <h2>💡 Key Insights</h2>
        <p>%s</p>
        <p><strong>Winning Markets:</strong> %s</p>
        <p><strong>Losing Markets:</strong> %s</p>
    </div>
    <div class="improvements">
        <h2>🎯 Suggested Improvements</h2>
        <p>%s</p>
    </div>
</body>
</html>
`, report.Date.Format("Monday, January 2, 2006"), report.TotalBets, report.Wins, report.Losses,
		successColor, report.SuccessRate*100, report.Insights,
		strings.Join(report.WinningMarkets, ", "), strings.Join(report.LosingMarkets, ", "),
		report.PromptImprovements)

	return html
}
