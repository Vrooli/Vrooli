package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"agent-inbox/domain"
	"agent-inbox/integrations"
	"agent-inbox/persistence"
	"agent-inbox/testutil"

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
	router.HandleFunc("/api/v1/chats/{id}/agent-mode/start", h.StartAgentMode).Methods("POST")
	router.HandleFunc("/api/v1/chats/{id}/agent-mode/message", h.SendAgentMessage).Methods("POST")
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

// setAgentMode puts a chat into agent mode with the given task/run IDs.
func (env *agentModeTestEnv) setAgentMode(t *testing.T, chatID, taskID, runID string) {
	t.Helper()
	if err := env.repo.SetAgentMode(context.Background(), chatID, taskID, runID); err != nil {
		t.Fatalf("failed to set agent mode: %v", err)
	}
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

// =============================================================================
// Validation Tests (no DB required)
// =============================================================================

func TestStartAgentMode_InvalidUUID(t *testing.T) {
	h := &Handlers{}
	req := httptest.NewRequest("POST", "/api/v1/chats/not-a-uuid/agent-mode/start", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "not-a-uuid"})
	w := httptest.NewRecorder()

	h.StartAgentMode(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStartAgentMode_InvalidJSON(t *testing.T) {
	h := &Handlers{}
	body := bytes.NewBufferString("{invalid json")
	req := httptest.NewRequest("POST", "/api/v1/chats/550e8400-e29b-41d4-a716-446655440000/agent-mode/start", body)
	req = mux.SetURLVars(req, map[string]string{"id": "550e8400-e29b-41d4-a716-446655440000"})
	w := httptest.NewRecorder()

	h.StartAgentMode(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStartAgentMode_MissingMessage(t *testing.T) {
	h := &Handlers{}
	body := bytes.NewBufferString(`{"project_path": "/tmp"}`)
	req := httptest.NewRequest("POST", "/api/v1/chats/550e8400-e29b-41d4-a716-446655440000/agent-mode/start", body)
	req = mux.SetURLVars(req, map[string]string{"id": "550e8400-e29b-41d4-a716-446655440000"})
	w := httptest.NewRecorder()

	h.StartAgentMode(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStartAgentMode_MissingProjectPath(t *testing.T) {
	h := &Handlers{}
	body := bytes.NewBufferString(`{"message": "hello"}`)
	req := httptest.NewRequest("POST", "/api/v1/chats/550e8400-e29b-41d4-a716-446655440000/agent-mode/start", body)
	req = mux.SetURLVars(req, map[string]string{"id": "550e8400-e29b-41d4-a716-446655440000"})
	w := httptest.NewRecorder()

	h.StartAgentMode(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStartAgentMode_NonexistentProjectPath(t *testing.T) {
	h := &Handlers{}
	body := bytes.NewBufferString(`{"message": "hello", "project_path": "/nonexistent/path/xyz"}`)
	req := httptest.NewRequest("POST", "/api/v1/chats/550e8400-e29b-41d4-a716-446655440000/agent-mode/start", body)
	req = mux.SetURLVars(req, map[string]string{"id": "550e8400-e29b-41d4-a716-446655440000"})
	w := httptest.NewRecorder()

	h.StartAgentMode(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("does not exist")) {
		t.Errorf("expected 'does not exist' in response, got: %s", w.Body.String())
	}
}

func TestStartAgentMode_FileNotDirectory(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(tmpFile, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := &Handlers{}
	body := bytes.NewBufferString(`{"message": "hello", "project_path": "` + tmpFile + `"}`)
	req := httptest.NewRequest("POST", "/api/v1/chats/550e8400-e29b-41d4-a716-446655440000/agent-mode/start", body)
	req = mux.SetURLVars(req, map[string]string{"id": "550e8400-e29b-41d4-a716-446655440000"})
	w := httptest.NewRecorder()

	h.StartAgentMode(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("not a directory")) {
		t.Errorf("expected 'not a directory' in response, got: %s", w.Body.String())
	}
}

func TestSendAgentMessage_InvalidUUID(t *testing.T) {
	h := &Handlers{}
	req := httptest.NewRequest("POST", "/api/v1/chats/bad/agent-mode/message", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "bad"})
	w := httptest.NewRecorder()

	h.SendAgentMessage(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSendAgentMessage_EmptyMessage(t *testing.T) {
	h := &Handlers{}
	body := bytes.NewBufferString(`{"message": ""}`)
	req := httptest.NewRequest("POST", "/api/v1/chats/550e8400-e29b-41d4-a716-446655440000/agent-mode/message", body)
	req = mux.SetURLVars(req, map[string]string{"id": "550e8400-e29b-41d4-a716-446655440000"})
	w := httptest.NewRecorder()

	h.SendAgentMessage(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetAgentEvents_InvalidUUID(t *testing.T) {
	h := &Handlers{}
	req := httptest.NewRequest("GET", "/api/v1/chats/xyz/agent-mode/events", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "xyz"})
	w := httptest.NewRecorder()

	h.GetAgentEvents(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestStopAgentMode_InvalidUUID(t *testing.T) {
	h := &Handlers{}
	req := httptest.NewRequest("POST", "/api/v1/chats/xyz/agent-mode/stop", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "xyz"})
	w := httptest.NewRecorder()

	h.StopAgentMode(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetAgentStatus_InvalidUUID(t *testing.T) {
	h := &Handlers{}
	req := httptest.NewRequest("GET", "/api/v1/chats/xyz/agent-mode/status", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "xyz"})
	w := httptest.NewRecorder()

	h.GetAgentStatus(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestClearAgentMode_InvalidUUID(t *testing.T) {
	h := &Handlers{}
	req := httptest.NewRequest("POST", "/api/v1/chats/xyz/agent-mode/clear", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "xyz"})
	w := httptest.NewRecorder()

	h.ClearAgentMode(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// =============================================================================
// StartAgentMode Handler Tests (DB required)
// =============================================================================

func TestStartAgentMode_Success(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)
	projectDir := t.TempDir()

	env.mockAgent.StartResult = &integrations.AgentChatSession{
		TaskID:    "task-123",
		RunID:     "run-456",
		SessionID: "sess-789",
	}

	w := env.doRequest("POST", "/api/v1/chats/"+chatID+"/agent-mode/start", map[string]interface{}{
		"message":      "Fix the bug",
		"project_path": projectDir,
		"runner_type":  "claude-code",
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp AgentModeResponse
	decodeBody(t, w, &resp)

	if resp.ChatID != chatID {
		t.Errorf("expected chat_id %s, got %s", chatID, resp.ChatID)
	}
	if resp.TaskID != "task-123" {
		t.Errorf("expected task_id task-123, got %s", resp.TaskID)
	}
	if resp.RunID != "run-456" {
		t.Errorf("expected run_id run-456, got %s", resp.RunID)
	}
	if resp.SessionID != "sess-789" {
		t.Errorf("expected session_id sess-789, got %s", resp.SessionID)
	}

	// Verify mock was called correctly
	if len(env.mockAgent.StartCalls) != 1 {
		t.Fatalf("expected 1 StartAgentChat call, got %d", len(env.mockAgent.StartCalls))
	}
	call := env.mockAgent.StartCalls[0]
	if call.Message != "Fix the bug" {
		t.Errorf("expected message 'Fix the bug', got %s", call.Message)
	}
	if call.Config.ProjectPath != projectDir {
		t.Errorf("expected project_path %s, got %s", projectDir, call.Config.ProjectPath)
	}

	// Verify chat is now in agent mode in DB
	chatMode, _, runID, err := env.repo.GetAgentMode(context.Background(), chatID)
	if err != nil {
		t.Fatalf("GetAgentMode failed: %v", err)
	}
	if chatMode != domain.ChatModeAgent {
		t.Errorf("expected chat_mode agent, got %s", chatMode)
	}
	if runID != "run-456" {
		t.Errorf("expected run_id run-456 in DB, got %s", runID)
	}
}

func TestStartAgentMode_DefaultRunnerType(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)

	env.mockAgent.StartResult = &integrations.AgentChatSession{
		TaskID: "task-1",
		RunID:  "run-1",
	}

	w := env.doRequest("POST", "/api/v1/chats/"+chatID+"/agent-mode/start", map[string]interface{}{
		"message":      "hello",
		"project_path": "/tmp",
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify default runner type was applied
	if len(env.mockAgent.StartCalls) != 1 {
		t.Fatal("expected 1 call")
	}
	if env.mockAgent.StartCalls[0].Config.RunnerType != integrations.RunnerTypeClaudeCode {
		t.Errorf("expected default runner type claude-code, got %s", env.mockAgent.StartCalls[0].Config.RunnerType)
	}
}

func TestStartAgentMode_ChatNotFound(t *testing.T) {
	env := setupAgentModeTest(t)

	w := env.doRequest("POST", "/api/v1/chats/550e8400-e29b-41d4-a716-446655440000/agent-mode/start", map[string]interface{}{
		"message":      "hello",
		"project_path": "/tmp",
	})

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStartAgentMode_AlreadyInAgentMode(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)
	env.setAgentMode(t, chatID, "task-1", "run-1")

	w := env.doRequest("POST", "/api/v1/chats/"+chatID+"/agent-mode/start", map[string]interface{}{
		"message":      "hello",
		"project_path": "/tmp",
	})

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
	assertErrorCode(t, w, "V014")
}

func TestStartAgentMode_AgentManagerFails(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)

	env.mockAgent.StartError = fmt.Errorf("connection refused")

	w := env.doRequest("POST", "/api/v1/chats/"+chatID+"/agent-mode/start", map[string]interface{}{
		"message":      "hello",
		"project_path": "/tmp",
	})

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStartAgentMode_AgentClientNil(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)
	env.handler.AgentClient = nil // Simulate agent-manager not available

	w := env.doRequest("POST", "/api/v1/chats/"+chatID+"/agent-mode/start", map[string]interface{}{
		"message":      "hello",
		"project_path": "/tmp",
	})

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502 for nil agent client, got %d: %s", w.Code, w.Body.String())
	}
}

// =============================================================================
// SendAgentMessage Handler Tests (DB required)
// =============================================================================

func TestSendAgentMessage_Success(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)
	env.setAgentMode(t, chatID, "task-1", "run-1")

	w := env.doRequest("POST", "/api/v1/chats/"+chatID+"/agent-mode/message", map[string]interface{}{
		"message": "Please continue",
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	decodeBody(t, w, &resp)

	if resp["success"] != true {
		t.Error("expected success=true")
	}
	if resp["run_id"] != "run-1" {
		t.Errorf("expected run_id run-1, got %v", resp["run_id"])
	}

	// Verify mock was called
	if len(env.mockAgent.ContinueCalls) != 1 {
		t.Fatalf("expected 1 ContinueChat call, got %d", len(env.mockAgent.ContinueCalls))
	}
	if env.mockAgent.ContinueCalls[0].RunID != "run-1" {
		t.Errorf("expected runID run-1, got %s", env.mockAgent.ContinueCalls[0].RunID)
	}
	if env.mockAgent.ContinueCalls[0].Message != "Please continue" {
		t.Errorf("expected message 'Please continue', got %s", env.mockAgent.ContinueCalls[0].Message)
	}
}

func TestSendAgentMessage_NotInAgentMode(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t) // Default is LLM mode

	w := env.doRequest("POST", "/api/v1/chats/"+chatID+"/agent-mode/message", map[string]interface{}{
		"message": "hello",
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSendAgentMessage_NoActiveRun(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)
	env.setAgentMode(t, chatID, "task-1", "") // Agent mode but no run ID

	w := env.doRequest("POST", "/api/v1/chats/"+chatID+"/agent-mode/message", map[string]interface{}{
		"message": "hello",
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSendAgentMessage_ContinueFails(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)
	env.setAgentMode(t, chatID, "task-1", "run-1")
	env.mockAgent.ContinueErr = fmt.Errorf("agent busy")

	w := env.doRequest("POST", "/api/v1/chats/"+chatID+"/agent-mode/message", map[string]interface{}{
		"message": "hello",
	})

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d: %s", w.Code, w.Body.String())
	}
}

// =============================================================================
// GetAgentEvents Handler Tests (DB required)
// =============================================================================

func TestGetAgentEvents_Success(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)
	env.setAgentMode(t, chatID, "task-1", "run-1")

	env.mockAgent.EventsResult = []*integrations.TranslatedEvent{
		{ID: "evt-1", Type: "message", Role: "assistant", Content: "Working on it"},
		{ID: "evt-2", Type: "tool_call", Role: "assistant", ToolName: "read_file"},
	}

	w := env.doRequest("GET", "/api/v1/chats/"+chatID+"/agent-mode/events", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	decodeBody(t, w, &resp)

	events, ok := resp["events"].([]interface{})
	if !ok {
		t.Fatal("expected events array in response")
	}
	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}
}

func TestGetAgentEvents_WithAfterSequence(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)
	env.setAgentMode(t, chatID, "task-1", "run-1")

	env.mockAgent.EventsResult = []*integrations.TranslatedEvent{}

	w := env.doRequest("GET", "/api/v1/chats/"+chatID+"/agent-mode/events?after_sequence=42", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify the after_sequence was passed through
	if len(env.mockAgent.EventsCalls) != 1 {
		t.Fatalf("expected 1 GetEvents call, got %d", len(env.mockAgent.EventsCalls))
	}
	if env.mockAgent.EventsCalls[0].AfterSequence != 42 {
		t.Errorf("expected after_sequence 42, got %d", env.mockAgent.EventsCalls[0].AfterSequence)
	}
}

func TestGetAgentEvents_NotInAgentMode(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)

	w := env.doRequest("GET", "/api/v1/chats/"+chatID+"/agent-mode/events", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetAgentEvents_GetEventsFails(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)
	env.setAgentMode(t, chatID, "task-1", "run-1")
	env.mockAgent.EventsError = fmt.Errorf("timeout")

	w := env.doRequest("GET", "/api/v1/chats/"+chatID+"/agent-mode/events", nil)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d: %s", w.Code, w.Body.String())
	}
}

// =============================================================================
// GetAgentStatus Handler Tests (DB required)
// =============================================================================

func TestGetAgentStatus_FullStatus(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)
	env.setAgentMode(t, chatID, "task-1", "run-1")

	env.mockAgent.StatusResult = &integrations.AgentRunStatus{
		RunID:           "run-1",
		Status:          integrations.RunStatusRunning,
		Phase:           "coding",
		ProgressPercent: 50,
		SessionID:       "sess-abc",
	}

	w := env.doRequest("GET", "/api/v1/chats/"+chatID+"/agent-mode/status", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	decodeBody(t, w, &resp)

	if resp["is_agent"] != true {
		t.Error("expected is_agent=true")
	}
	if resp["status"] != "running" {
		t.Errorf("expected status running, got %v", resp["status"])
	}
	if resp["phase"] != "coding" {
		t.Errorf("expected phase coding, got %v", resp["phase"])
	}
	if resp["progress_percent"] != float64(50) {
		t.Errorf("expected progress_percent 50, got %v", resp["progress_percent"])
	}
}

func TestGetAgentStatus_NotAgentMode(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t) // Default LLM mode

	w := env.doRequest("GET", "/api/v1/chats/"+chatID+"/agent-mode/status", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	decodeBody(t, w, &resp)

	if resp["is_agent"] != false {
		t.Error("expected is_agent=false for LLM mode")
	}
}

func TestGetAgentStatus_NoRunID(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)
	env.setAgentMode(t, chatID, "task-1", "") // Agent mode but no run

	w := env.doRequest("GET", "/api/v1/chats/"+chatID+"/agent-mode/status", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	decodeBody(t, w, &resp)

	if resp["is_agent"] != true {
		t.Error("expected is_agent=true")
	}
	if resp["task_id"] != "task-1" {
		t.Errorf("expected task_id task-1, got %v", resp["task_id"])
	}
	if resp["run_id"] != nil {
		t.Errorf("expected run_id nil, got %v", resp["run_id"])
	}

	// Should NOT have called GetRunStatus since no run ID
	if len(env.mockAgent.StatusCalls) != 0 {
		t.Error("expected no GetRunStatus calls when no run ID")
	}
}

func TestGetAgentStatus_StatusFetchFails(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)
	env.setAgentMode(t, chatID, "task-1", "run-1")
	env.mockAgent.StatusError = fmt.Errorf("connection refused")

	w := env.doRequest("GET", "/api/v1/chats/"+chatID+"/agent-mode/status", nil)

	// Should still return 200 with partial info (graceful degradation)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 (graceful degradation), got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	decodeBody(t, w, &resp)

	if resp["is_agent"] != true {
		t.Error("expected is_agent=true")
	}
	if resp["error"] != "unable to fetch live status" {
		t.Errorf("expected error message, got %v", resp["error"])
	}
}

// =============================================================================
// StopAgentMode Handler Tests (DB required)
// =============================================================================

func TestStopAgentMode_Success(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)
	env.setAgentMode(t, chatID, "task-1", "run-1")

	w := env.doRequest("POST", "/api/v1/chats/"+chatID+"/agent-mode/stop", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	decodeBody(t, w, &resp)

	if resp["success"] != true {
		t.Error("expected success=true")
	}

	// Verify mock was called
	if len(env.mockAgent.StopCalls) != 1 {
		t.Fatalf("expected 1 StopRun call, got %d", len(env.mockAgent.StopCalls))
	}
	if env.mockAgent.StopCalls[0] != "run-1" {
		t.Errorf("expected runID run-1, got %s", env.mockAgent.StopCalls[0])
	}
}

func TestStopAgentMode_NotInAgentMode(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)

	w := env.doRequest("POST", "/api/v1/chats/"+chatID+"/agent-mode/stop", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStopAgentMode_StopFails(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)
	env.setAgentMode(t, chatID, "task-1", "run-1")
	env.mockAgent.StopError = fmt.Errorf("run already stopped")

	w := env.doRequest("POST", "/api/v1/chats/"+chatID+"/agent-mode/stop", nil)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d: %s", w.Code, w.Body.String())
	}
}

// =============================================================================
// ClearAgentMode Handler Tests (DB required)
// =============================================================================

func TestClearAgentMode_Success(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)
	env.setAgentMode(t, chatID, "task-1", "run-1")

	w := env.doRequest("POST", "/api/v1/chats/"+chatID+"/agent-mode/clear", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	decodeBody(t, w, &resp)

	if resp["success"] != true {
		t.Error("expected success=true")
	}
	if resp["chat_mode"] != domain.ChatModeLLM {
		t.Errorf("expected chat_mode llm, got %v", resp["chat_mode"])
	}

	// Verify DB state
	chatMode, _, _, err := env.repo.GetAgentMode(context.Background(), chatID)
	if err != nil {
		t.Fatalf("GetAgentMode failed: %v", err)
	}
	if chatMode != domain.ChatModeLLM {
		t.Errorf("expected chat_mode llm in DB, got %s", chatMode)
	}

	// Should have tried to stop the run
	if len(env.mockAgent.StopCalls) != 1 {
		t.Errorf("expected 1 StopRun call, got %d", len(env.mockAgent.StopCalls))
	}
}

func TestClearAgentMode_WithActiveRun(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)
	env.setAgentMode(t, chatID, "task-1", "run-active")

	w := env.doRequest("POST", "/api/v1/chats/"+chatID+"/agent-mode/clear", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify the stop was attempted with the correct run ID
	if len(env.mockAgent.StopCalls) != 1 {
		t.Fatalf("expected 1 StopRun call, got %d", len(env.mockAgent.StopCalls))
	}
	if env.mockAgent.StopCalls[0] != "run-active" {
		t.Errorf("expected runID run-active, got %s", env.mockAgent.StopCalls[0])
	}

	// Verify mode was cleared
	chatMode, _, _, _ := env.repo.GetAgentMode(context.Background(), chatID)
	if chatMode != domain.ChatModeLLM {
		t.Errorf("expected llm mode after clear, got %s", chatMode)
	}
}

func TestClearAgentMode_StopFails_StillClears(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)
	env.setAgentMode(t, chatID, "task-1", "run-1")
	env.mockAgent.StopError = fmt.Errorf("already stopped")

	w := env.doRequest("POST", "/api/v1/chats/"+chatID+"/agent-mode/clear", nil)

	// Should still succeed — stop failure is non-fatal
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify mode was still cleared despite stop failure
	chatMode, _, _, _ := env.repo.GetAgentMode(context.Background(), chatID)
	if chatMode != domain.ChatModeLLM {
		t.Errorf("expected llm mode after clear (even with stop failure), got %s", chatMode)
	}
}

func TestClearAgentMode_NotInAgentMode(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t) // Default LLM mode

	w := env.doRequest("POST", "/api/v1/chats/"+chatID+"/agent-mode/clear", nil)

	// Should still succeed - clearing LLM mode is a no-op but valid
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Should NOT have tried to stop any run
	if len(env.mockAgent.StopCalls) != 0 {
		t.Error("expected no StopRun calls when not in agent mode")
	}
}

func TestClearAgentMode_AgentClientNil(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)
	env.setAgentMode(t, chatID, "task-1", "run-1")
	env.handler.AgentClient = nil // Simulate agent-manager not available

	w := env.doRequest("POST", "/api/v1/chats/"+chatID+"/agent-mode/clear", nil)

	// Should still succeed - nil client means we skip the stop attempt
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify mode was cleared
	chatMode, _, _, _ := env.repo.GetAgentMode(context.Background(), chatID)
	if chatMode != domain.ChatModeLLM {
		t.Errorf("expected llm mode, got %s", chatMode)
	}
}

// =============================================================================
// getAgentClient Helper Tests
// =============================================================================

func TestGetAgentClient_Nil(t *testing.T) {
	h := &Handlers{AgentClient: nil}
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	client := h.getAgentClient(w, req)

	if client != nil {
		t.Error("expected nil client")
	}
	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
	assertErrorCode(t, w, "D008")
}

func TestGetAgentClient_Set(t *testing.T) {
	mock := &testutil.MockAgentManagerClient{}
	h := &Handlers{AgentClient: mock}
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	client := h.getAgentClient(w, req)

	if client == nil {
		t.Error("expected non-nil client")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 (default), got %d", w.Code)
	}
}

// =============================================================================
// Agent Error Code Regression Tests
// =============================================================================

func TestAgentErrorCodes_NotInAgentMode(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)
	// Chat is in LLM mode (default), not agent mode

	tests := []struct {
		name   string
		method string
		url    string
	}{
		{"SendMessage", "POST", "/api/v1/chats/" + chatID + "/agent-mode/message"},
		{"GetEvents", "GET", "/api/v1/chats/" + chatID + "/agent-mode/events"},
		{"Stop", "POST", "/api/v1/chats/" + chatID + "/agent-mode/stop"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var w *httptest.ResponseRecorder
			if tt.method == "POST" {
				w = env.doRequest(tt.method, tt.url, map[string]interface{}{"message": "hello"})
			} else {
				w = env.doRequest(tt.method, tt.url, nil)
			}

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
			}
			assertErrorCode(t, w, "V012")
		})
	}
}

func TestAgentErrorCodes_NoActiveRun(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)
	// Set agent mode with task ID but empty run ID
	env.setAgentMode(t, chatID, "task-1", "")

	tests := []struct {
		name   string
		method string
		url    string
	}{
		{"SendMessage", "POST", "/api/v1/chats/" + chatID + "/agent-mode/message"},
		{"GetEvents", "GET", "/api/v1/chats/" + chatID + "/agent-mode/events"},
		{"Stop", "POST", "/api/v1/chats/" + chatID + "/agent-mode/stop"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var w *httptest.ResponseRecorder
			if tt.method == "POST" {
				w = env.doRequest(tt.method, tt.url, map[string]interface{}{"message": "hello"})
			} else {
				w = env.doRequest(tt.method, tt.url, nil)
			}

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
			}
			assertErrorCode(t, w, "V013")
		})
	}
}

func TestAgentErrorCodes_AlreadyActive(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)
	env.setAgentMode(t, chatID, "task-1", "run-1")

	w := env.doRequest("POST", "/api/v1/chats/"+chatID+"/agent-mode/start", map[string]interface{}{
		"message":      "hello",
		"project_path": "/tmp",
	})

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
	assertErrorCode(t, w, "V014")
}

func TestAgentErrorCodes_ManagerUnavailable(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)
	env.handler.AgentClient = nil

	w := env.doRequest("POST", "/api/v1/chats/"+chatID+"/agent-mode/start", map[string]interface{}{
		"message":      "hello",
		"project_path": "/tmp",
	})

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d: %s", w.Code, w.Body.String())
	}
	assertErrorCode(t, w, "D008")

	// Verify user_message is in details
	var resp APIErrorResponse
	decodeBody(t, w, &resp)
	if resp.Error.Details == nil {
		t.Fatal("expected details in error response")
	}
	if _, ok := resp.Error.Details["user_message"]; !ok {
		t.Error("expected user_message in error details")
	}
}
