package main

import (
	"agent-inbox/config"
	"agent-inbox/domain"
	"agent-inbox/handlers"
	"agent-inbox/integrations"
	"agent-inbox/middleware"
	"agent-inbox/persistence"
	"agent-inbox/services"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	_ "modernc.org/sqlite"
)

// TestServer provides a test harness for the API.
type TestServer struct {
	repo     *persistence.Repository
	router   *mux.Router
	handlers *handlers.Handlers
}

// setupTestServer creates a test server with an in-memory SQLite database.
func setupTestServer(t *testing.T) *TestServer {
	// New chats with no per-request model resolve their default from the
	// OpenRouter policy role. Set an explicit operator override so handler-driven
	// chat creation stays hermetic (no resource-openrouter binary required).
	t.Setenv("DEFAULT_AI_MODEL", "vendor/test-default-model")

	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("Failed to open in-memory SQLite: %v", err)
	}
	db.SetMaxOpenConns(1)

	repo := persistence.NewRepository(db)

	if err := repo.InitSchema(context.Background()); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	// Create storage service for tests
	storageCfg := config.GetStorageConfig()
	storage := services.NewLocalStorageService(storageCfg)

	// Create shared dependencies explicitly (no nil fallbacks)
	toolExecutor := integrations.NewToolExecutor()
	asyncTracker := services.NewAsyncTrackerService(toolExecutor, nil)

	h := handlers.New(repo, integrations.NewOllamaClient(), storage, asyncTracker, toolExecutor)
	router := mux.NewRouter()
	router.Use(middleware.Logging)
	router.Use(middleware.CORS)
	h.RegisterRoutes(router)

	return &TestServer{
		repo:     repo,
		router:   router,
		handlers: h,
	}
}

func (ts *TestServer) cleanup(t *testing.T) {
	// Clean up test data
	db := ts.repo.DB()
	_, _ = db.Exec("DELETE FROM messages")
	_, _ = db.Exec("DELETE FROM chat_labels")
	_, _ = db.Exec("DELETE FROM chats")
	_, _ = db.Exec("DELETE FROM labels")
	db.Close()
}

// Helper functions for creating test data

func createTestChat(t *testing.T, ts *TestServer) domain.Chat {
	body := bytes.NewBuffer([]byte(`{"name": "Test Chat"}`))
	req := httptest.NewRequest("POST", "/api/v1/chats", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create test chat: %s", w.Body.String())
	}

	var chat domain.Chat
	if err := json.Unmarshal(w.Body.Bytes(), &chat); err != nil {
		t.Fatalf("Failed to parse chat: %v", err)
	}

	return chat
}

func createTestChatWithModel(t *testing.T, ts *TestServer, name, model string) domain.Chat {
	body := bytes.NewBuffer([]byte(`{"name": "` + name + `", "model": "` + model + `"}`))
	req := httptest.NewRequest("POST", "/api/v1/chats", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create test chat: %s", w.Body.String())
	}

	var chat domain.Chat
	if err := json.Unmarshal(w.Body.Bytes(), &chat); err != nil {
		t.Fatalf("Failed to parse chat: %v", err)
	}

	return chat
}

func addTestMessage(t *testing.T, ts *TestServer, chatID, role, content string) domain.Message {
	body := bytes.NewBuffer([]byte(`{"role": "` + role + `", "content": "` + content + `"}`))
	req := httptest.NewRequest("POST", "/api/v1/chats/"+chatID+"/messages", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to add test message: %s", w.Body.String())
	}

	var msg domain.Message
	if err := json.Unmarshal(w.Body.Bytes(), &msg); err != nil {
		t.Fatalf("Failed to parse message: %v", err)
	}

	return msg
}

func createTestLabel(t *testing.T, ts *TestServer, name, color string) domain.Label {
	body := bytes.NewBuffer([]byte(`{"name": "` + name + `", "color": "` + color + `"}`))
	req := httptest.NewRequest("POST", "/api/v1/labels", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create test label: %s", w.Body.String())
	}

	var label domain.Label
	if err := json.Unmarshal(w.Body.Bytes(), &label); err != nil {
		t.Fatalf("Failed to parse label: %v", err)
	}

	return label
}
