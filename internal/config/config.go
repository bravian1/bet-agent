package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

// ============================================================================
// CONFIGURATION
// ============================================================================

type Config struct {
	GeminiAPIKey      string
	DiscoveryModel    string
	AnalysisModel     string
	OptimizationModel string
	SMTPHost          string
	SMTPPort          string
	SMTPUser          string
	SMTPPassword      string
	EmailFrom         string
	EmailTo           string
	MainCron          string
	OptimizationCron  string
	RateLimitSeconds  int
	DataDir           string
	PromptsDir        string
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	config := &Config{
		GeminiAPIKey:      os.Getenv("GEMINI_API_KEY"),
		DiscoveryModel:    getEnvOrDefault("DISCOVERY_MODEL", "gemini-2.0-flash-exp"),
		AnalysisModel:     getEnvOrDefault("ANALYSIS_MODEL", "gemini-2.0-flash-thinking-exp"),
		OptimizationModel: getEnvOrDefault("OPTIMIZATION_MODEL", "gemini-2.0-flash-thinking-exp"),
		SMTPHost:          os.Getenv("SMTP_HOST"),
		SMTPPort:          getEnvOrDefault("SMTP_PORT", "587"),
		SMTPUser:          os.Getenv("SMTP_USER"),
		SMTPPassword:      os.Getenv("SMTP_PASSWORD"),
		EmailFrom:         os.Getenv("EMAIL_FROM"),
		EmailTo:           os.Getenv("EMAIL_TO"),
		MainCron:          getEnvOrDefault("MAIN_CRON", "0 9 * * *"),
		OptimizationCron:  getEnvOrDefault("OPTIMIZATION_CRON", "0 22 * * *"),
		RateLimitSeconds:  getEnvOrDefaultInt("RATE_LIMIT_SECONDS", 3),
		DataDir:           getEnvOrDefault("DATA_DIR", "data"),
		PromptsDir:        getEnvOrDefault("PROMPTS_DIR", "prompts"),
	}

	if config.GeminiAPIKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is required")
	}

	return config, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvOrDefaultInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		fmt.Sscanf(value, "%d", &intValue)
		return intValue
	}
	return defaultValue
}
