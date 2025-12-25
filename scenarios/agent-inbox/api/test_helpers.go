package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestServer wraps Server for testing
type TestServer struct {
	*Server
}

// setupTestServer creates a test server with a real test database
func setupTestServer(t *testing.T) *TestServer {
	// Skip if DATABASE_URL is not set (unit tests without DB)
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping database tests")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	srv := &Server{
		config: &Config{Port: "0"},
		db:     db,
		router: mux.NewRouter(),
	}

	if err := srv.initSchema(); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	srv.setupRoutes()

	return &TestServer{srv}
}

func (ts *TestServer) cleanup(t *testing.T) {
	// Clean up test data
	_, _ = ts.db.Exec("DELETE FROM messages")
	_, _ = ts.db.Exec("DELETE FROM chat_labels")
	_, _ = ts.db.Exec("DELETE FROM chats")
	_, _ = ts.db.Exec("DELETE FROM labels")
	ts.db.Close()
}

// Helper functions for creating test data

func createTestChat(t *testing.T, ts *TestServer) Chat {
	body := bytes.NewBuffer([]byte(`{"name": "Test Chat"}`))
	req := httptest.NewRequest("POST", "/api/v1/chats", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create test chat: %s", w.Body.String())
	}

	var chat Chat
	if err := json.Unmarshal(w.Body.Bytes(), &chat); err != nil {
		t.Fatalf("Failed to parse chat: %v", err)
	}

	return chat
}

func createTestChatWithModel(t *testing.T, ts *TestServer, name, model string) Chat {
	body := bytes.NewBuffer([]byte(`{"name": "` + name + `", "model": "` + model + `"}`))
	req := httptest.NewRequest("POST", "/api/v1/chats", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create test chat: %s", w.Body.String())
	}

	var chat Chat
	if err := json.Unmarshal(w.Body.Bytes(), &chat); err != nil {
		t.Fatalf("Failed to parse chat: %v", err)
	}

	return chat
}

func addTestMessage(t *testing.T, ts *TestServer, chatID, role, content string) Message {
	body := bytes.NewBuffer([]byte(`{"role": "` + role + `", "content": "` + content + `"}`))
	req := httptest.NewRequest("POST", "/api/v1/chats/"+chatID+"/messages", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to add test message: %s", w.Body.String())
	}

	var msg Message
	if err := json.Unmarshal(w.Body.Bytes(), &msg); err != nil {
		t.Fatalf("Failed to parse message: %v", err)
	}

	return msg
}

func createTestLabel(t *testing.T, ts *TestServer, name, color string) Label {
	body := bytes.NewBuffer([]byte(`{"name": "` + name + `", "color": "` + color + `"}`))
	req := httptest.NewRequest("POST", "/api/v1/labels", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create test label: %s", w.Body.String())
	}

	var label Label
	if err := json.Unmarshal(w.Body.Bytes(), &label); err != nil {
		t.Fatalf("Failed to parse label: %v", err)
	}

	return label
}
