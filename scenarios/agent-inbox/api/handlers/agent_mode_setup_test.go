package handlers

import (
	"agent-inbox/config"
	"agent-inbox/integrations"
	"agent-inbox/persistence"
	"agent-inbox/testutil"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	_ "modernc.org/sqlite"
)

// =============================================================================
// Test Environment
// =============================================================================

type agentModeTestEnv struct {
	handler   *Handlers
	mockAgent *testutil.MockAgentManagerClient
	router    *mux.Router
	repo      *persistence.Repository
	db        *sql.DB
}

// setupAgentModeTest creates a test environment with an in-memory SQLite DB and mock agent client.
func setupAgentModeTest(t *testing.T) *agentModeTestEnv {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("failed to open in-memory SQLite: %v", err)
	}
	db.SetMaxOpenConns(1)

	repo := persistence.NewRepository(db)
	if err := repo.InitSchema(context.Background()); err != nil {
		t.Fatalf("failed to initialize schema: %v", err)
	}

	mockAgent := &testutil.MockAgentManagerClient{}

	h := &Handlers{
		Repo:        repo,
		AgentClient: mockAgent,
	}

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/agent-runs", h.ListAgentRuns).Methods("GET")
	router.HandleFunc("/api/v1/agent-runs/{run_id}/events", h.GetRunEvents).Methods("GET")
	router.HandleFunc("/api/v1/chats/{id}/agent-mode/attach", h.AttachAgentRun).Methods("POST")
	router.HandleFunc("/api/v1/chats/{id}/agent-mode/start", h.StartAgentMode).Methods("POST")
	router.HandleFunc("/api/v1/chats/{id}/agent-mode/message", h.SendAgentMessage).Methods("POST")
	router.HandleFunc("/api/v1/chats/{id}/messages", h.AddMessage).Methods("POST")
	router.HandleFunc("/api/v1/chats/{id}/agent-mode/events", h.GetAgentEvents).Methods("GET")
	router.HandleFunc("/api/v1/chats/{id}/agent-mode/status", h.GetAgentStatus).Methods("GET")
	router.HandleFunc("/api/v1/chats/{id}/agent-mode/stop", h.StopAgentMode).Methods("POST")
	router.HandleFunc("/api/v1/chats/{id}/agent-mode/clear", h.ClearAgentMode).Methods("POST")

	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM messages")
		_, _ = db.Exec("DELETE FROM chat_labels")
		_, _ = db.Exec("DELETE FROM chats")
		db.Close()
	})

	return &agentModeTestEnv{
		handler:   h,
		mockAgent: mockAgent,
		router:    router,
		repo:      repo,
		db:        db,
	}
}

// createChat creates a test chat in the database and returns its ID.
func (env *agentModeTestEnv) createChat(t *testing.T) string {
	t.Helper()
	chat, err := env.repo.CreateChat(context.Background(), "Test Chat", "", "", "")
	if err != nil {
		t.Fatalf("failed to create test chat: %v", err)
	}
	return chat.ID
}

func (env *agentModeTestEnv) createChatWithName(t *testing.T, name string) string {
	t.Helper()
	chat, err := env.repo.CreateChat(context.Background(), name, "", "", "")
	if err != nil {
		t.Fatalf("failed to create test chat: %v", err)
	}
	return chat.ID
}

// setAgentMode puts a chat into agent mode with the given task/run IDs.
func (env *agentModeTestEnv) setAgentMode(t *testing.T, chatID, taskID, runID string) {
	t.Helper()
	if err := env.repo.SetAgentMode(context.Background(), chatID, taskID, runID); err != nil {
		t.Fatalf("failed to set agent mode: %v", err)
	}
}

func configureOllamaNaming(t *testing.T, h *Handlers, generatedName string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fmt.Sprintf(`{"response":"%s","done":true}`, generatedName)))
	}))
	t.Cleanup(server.Close)

	namingCfg := config.Default().Integration.Naming
	namingCfg.Timeout = 500 * time.Millisecond
	h.OllamaClient = integrations.NewOllamaClientWithConfig(server.URL, namingCfg)
}

// doRequest makes an HTTP request through the test router.
func (env *agentModeTestEnv) doRequest(method, url string, body interface{}) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, url, reqBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)
	return w
}

// decodeBody decodes a JSON response body into the target.
func decodeBody(t *testing.T, w *httptest.ResponseRecorder, target interface{}) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), target); err != nil {
		t.Fatalf("failed to decode response body: %v\nbody: %s", err, w.Body.String())
	}
}

// assertErrorCode verifies that the response body contains the expected error code.
func assertErrorCode(t *testing.T, w *httptest.ResponseRecorder, expectedCode string) {
	t.Helper()
	var resp APIErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode error response: %v\nbody: %s", err, w.Body.String())
	}
	if resp.Error.Code != expectedCode {
		t.Errorf("expected error code %s, got %s", expectedCode, resp.Error.Code)
	}
}
