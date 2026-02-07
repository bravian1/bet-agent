package server

import (
	"bet-agent/internal/config"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestHandleSubscribe(t *testing.T) {
	// Setup temporary data directory
	tmpDir, err := os.MkdirTemp("", "bet-agent-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		DataDir: tmpDir,
	}

	srv := NewServer(cfg)

	// Test 1: Subscribe new email
	t.Run("Subscribe New Email", func(t *testing.T) {
		body := []byte(`{"email": "test@example.com"}`)
		req := httptest.NewRequest("POST", "/api/subscribe", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		srv.handleSubscribe(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK, got %v", resp.Status)
		}

		var result map[string]string
		json.NewDecoder(resp.Body).Decode(&result)
		if result["status"] != "success" {
			t.Errorf("Expected status success, got %v", result["status"])
		}

		// Verify file content
		recipients, _ := srv.loadEmails()
		if len(recipients) != 1 || recipients[0].Email != "test@example.com" {
			t.Errorf("Expected 1 recipient with email test@example.com")
		}
	})

	// Test 2: Duplicate email
	t.Run("Duplicate Email", func(t *testing.T) {
		// First subscription (already done in previous test, but state persists in file)
		// Wait, state persists in file, so we expect "exists" now.

		body := []byte(`{"email": "test@example.com"}`)
		req := httptest.NewRequest("POST", "/api/subscribe", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		srv.handleSubscribe(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK, got %v", resp.Status)
		}

		var result map[string]string
		json.NewDecoder(resp.Body).Decode(&result)
		if result["status"] != "exists" {
			t.Errorf("Expected status exists, got %v", result["status"])
		}
	})

	// Test 3: Invalid Email
	t.Run("Empty Email", func(t *testing.T) {
		body := []byte(`{"email": ""}`)
		req := httptest.NewRequest("POST", "/api/subscribe", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		srv.handleSubscribe(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status BadRequest, got %v", resp.Status)
		}
	})
}
