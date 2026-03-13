package server

import (
	"bet-agent/internal/config"
	"bet-agent/internal/db"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthEndpoints(t *testing.T) {
	cfg := &config.Config{
		DataDir: "/tmp/bet-agent-test",
	}

	// Create server with nil agent and empty database (no DB connection)
	database := &db.Database{}
	srv := NewServer(cfg, database, nil)

	t.Run("Static files handler exists", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		mux := http.NewServeMux()
		fs := http.FileServer(http.Dir("../../web/static"))
		mux.Handle("/", fs)
		mux.ServeHTTP(w, req)

		// We just verify the mux doesn't panic
	})

	t.Run("Subscribe rejects GET", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/subscribe", nil)
		w := httptest.NewRecorder()

		srv.handleSubscribe(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("Expected MethodNotAllowed, got %v", resp.Status)
		}
	})

	t.Run("Prompt history returns error without DB", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/public/analytics/prompt-history", nil)
		w := httptest.NewRecorder()

		srv.handlePromptHistory(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("Expected InternalServerError (no DB), got %v", resp.Status)
		}
	})
}
