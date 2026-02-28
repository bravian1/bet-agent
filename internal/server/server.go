package server

import (
	"bet-agent/internal/agent"
	"bet-agent/internal/config"
	"bet-agent/internal/db"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
)

type Server struct {
	config   *config.Config
	database *db.Database
	agent    *agent.BetAgent
}

func NewServer(cfg *config.Config, database *db.Database, betAgent *agent.BetAgent) *Server {
	return &Server{
		config:   cfg,
		database: database,
		agent:    betAgent,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Serve static files
	fs := http.FileServer(http.Dir("web/static"))
	mux.Handle("/", fs)

	// API endpoints
	mux.HandleFunc("/api/subscribe", s.handleSubscribe)

	// Internal Endpoints for cron (Serverless execution)
	mux.HandleFunc("/api/internal/run-main", s.withAuth(s.handleRunMain))
	mux.HandleFunc("/api/internal/run-optimization", s.withAuth(s.handleRunOptimization))

	// Determine port
	port := "8080" // Default port
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}
	// Note: We'll update the main function to handle standard ENV port if needed
	log.Printf("🌐 Starting web server on http://localhost:%s", port)
	return http.ListenAndServe(":"+port, mux)
}

// withAuth is a middleware that requires the INTERNAL_API_KEY
func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.config.InternalAPIKey != "" {
			authHeader := r.Header.Get("Authorization")
			expected := "Bearer " + s.config.InternalAPIKey
			if authHeader != expected {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next(w, r)
	}
}

func (s *Server) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	if err := s.database.AddSubscriber(req.Email); err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "UNIQUE constraint") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"status":  "exists",
				"message": "Email already subscribed",
			})
			return
		}
		log.Printf("Failed to save email: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ New subscriber: %s", req.Email)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Successfully subscribed!",
	})
}

func (s *Server) handleRunMain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Run in a goroutine so the webhook returns immediately
	go func() {
		log.Println("\n🔔 Main workflow triggered via HTTP")
		if err := s.agent.RunMainWorkflow(context.Background()); err != nil {
			log.Printf("❌ Main workflow error: %v\n", err)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Main workflow started",
	})
}

func (s *Server) handleRunOptimization(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Run in a goroutine so the webhook returns immediately
	go func() {
		log.Println("\n🔔 Optimization workflow triggered via HTTP")
		if err := s.agent.RunOptimizationWorkflow(context.Background()); err != nil {
			log.Printf("❌ Optimization workflow error: %v\n", err)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Optimization workflow started",
	})
}
