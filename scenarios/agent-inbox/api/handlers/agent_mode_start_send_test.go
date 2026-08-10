package handlers

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"agent-inbox/domain"
	"agent-inbox/integrations"
)

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
