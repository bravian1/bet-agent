package server

import (
	"bet-agent/internal/config"
	"bet-agent/internal/models"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Server struct {
	config *config.Config
	mu     sync.Mutex // Protects file writes
}

func NewServer(cfg *config.Config) *Server {
	return &Server{
		config: cfg,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Serve static files
	fs := http.FileServer(http.Dir("web/static"))
	mux.Handle("/", fs)

	// API endpoints
	mux.HandleFunc("/api/subscribe", s.handleSubscribe)

	port := "8080" // Could be configurable
	log.Printf("🌐 Starting web server on http://localhost:%s", port)
	return http.ListenAndServe(":"+port, mux)
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

	s.mu.Lock()
	defer s.mu.Unlock()

	recipients, err := s.loadEmails()
	if err != nil {
		log.Printf("Failed to load emails: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Check for duplicates
	for _, r := range recipients {
		if r.Email == req.Email {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"status":  "exists",
				"message": "Email already subscribed",
			})
			return
		}
	}

	// Add new recipient
	recipients = append(recipients, models.Recipient{
		Email:     req.Email,
		CreatedAt: time.Now(),
	})

	if err := s.saveEmails(recipients); err != nil {
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

func (s *Server) loadEmails() ([]models.Recipient, error) {
	path := filepath.Join(s.config.DataDir, "mailing_list.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []models.Recipient{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var recipients []models.Recipient
	if err := json.Unmarshal(data, &recipients); err != nil {
		return nil, err
	}

	return recipients, nil
}

func (s *Server) saveEmails(recipients []models.Recipient) error {
	path := filepath.Join(s.config.DataDir, "mailing_list.json")
	data, err := json.MarshalIndent(recipients, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
