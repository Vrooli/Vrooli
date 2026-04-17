package handlers

import (
	"agent-inbox/domain"
	"agent-inbox/integrations"
	"context"
	"fmt"
	"net/http"
	"testing"

	_ "modernc.org/sqlite"
)

func TestListAgentRuns_AgentManagerUnavailable(t *testing.T) {
	env := setupAgentModeTest(t)
	env.handler.AgentClient = nil

	w := env.doRequest("GET", "/api/v1/agent-runs", nil)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d: %s", w.Code, w.Body.String())
	}
	assertErrorCode(t, w, "D008")
}

// =============================================================================
// GetRunEvents Handler Tests
// =============================================================================

func TestGetRunEvents_Success(t *testing.T) {
	env := setupAgentModeTest(t)

	env.mockAgent.EventsResult = []*integrations.TranslatedEvent{
		{ID: "evt-1", Type: "message", Role: "assistant", Content: "Hello!", Sequence: 1},
		{ID: "evt-2", Type: "tool_call", Role: "assistant", Content: "", ToolName: "bash", Sequence: 2},
	}

	w := env.doRequest("GET", "/api/v1/agent-runs/run-abc/events", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Events []integrations.TranslatedEvent `json:"events"`
		RunID  string                         `json:"run_id"`
	}
	decodeBody(t, w, &resp)

	if resp.RunID != "run-abc" {
		t.Errorf("expected run_id run-abc, got %s", resp.RunID)
	}
	if len(resp.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(resp.Events))
	}
}

func TestGetRunEvents_WithAfterSequence(t *testing.T) {
	env := setupAgentModeTest(t)

	env.mockAgent.EventsResult = []*integrations.TranslatedEvent{}

	w := env.doRequest("GET", "/api/v1/agent-runs/run-abc/events?after_sequence=5", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if len(env.mockAgent.EventsCalls) != 1 {
		t.Fatalf("expected 1 GetEvents call, got %d", len(env.mockAgent.EventsCalls))
	}
	if env.mockAgent.EventsCalls[0].AfterSequence != 5 {
		t.Errorf("expected after_sequence 5, got %d", env.mockAgent.EventsCalls[0].AfterSequence)
	}
}

func TestGetRunEvents_AgentManagerUnavailable(t *testing.T) {
	env := setupAgentModeTest(t)
	env.handler.AgentClient = nil

	w := env.doRequest("GET", "/api/v1/agent-runs/run-abc/events", nil)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d: %s", w.Code, w.Body.String())
	}
	assertErrorCode(t, w, "D008")
}

// =============================================================================
// AttachAgentRun Handler Tests
// =============================================================================

func TestAttachAgentRun_Success(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)

	env.mockAgent.StatusResult = &integrations.AgentRunStatus{
		RunID:  "run-ext-1",
		Status: integrations.RunStatusRunning,
	}

	w := env.doRequest("POST", "/api/v1/chats/"+chatID+"/agent-mode/attach", map[string]interface{}{
		"run_id":  "run-ext-1",
		"task_id": "task-ext-1",
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp AgentModeResponse
	decodeBody(t, w, &resp)

	if resp.ChatID != chatID {
		t.Errorf("expected chat_id %s, got %s", chatID, resp.ChatID)
	}
	if resp.RunID != "run-ext-1" {
		t.Errorf("expected run_id run-ext-1, got %s", resp.RunID)
	}
	if resp.TaskID != "task-ext-1" {
		t.Errorf("expected task_id task-ext-1, got %s", resp.TaskID)
	}

	// Verify DB state
	chatMode, taskID, runID, err := env.repo.GetAgentMode(context.Background(), chatID)
	if err != nil {
		t.Fatalf("GetAgentMode failed: %v", err)
	}
	if chatMode != domain.ChatModeAgent {
		t.Errorf("expected chat_mode agent, got %s", chatMode)
	}
	if taskID != "task-ext-1" {
		t.Errorf("expected task_id task-ext-1 in DB, got %s", taskID)
	}
	if runID != "run-ext-1" {
		t.Errorf("expected run_id run-ext-1 in DB, got %s", runID)
	}

	// Verify GetRunStatus was called to validate the run
	if len(env.mockAgent.StatusCalls) != 1 {
		t.Fatalf("expected 1 GetRunStatus call, got %d", len(env.mockAgent.StatusCalls))
	}
	if env.mockAgent.StatusCalls[0] != "run-ext-1" {
		t.Errorf("expected GetRunStatus for run-ext-1, got %s", env.mockAgent.StatusCalls[0])
	}
}

func TestAttachAgentRun_ChatNotFound(t *testing.T) {
	env := setupAgentModeTest(t)

	w := env.doRequest("POST", "/api/v1/chats/550e8400-e29b-41d4-a716-446655440000/agent-mode/attach", map[string]interface{}{
		"run_id":  "run-1",
		"task_id": "task-1",
	})

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAttachAgentRun_AlreadyActive(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)
	env.setAgentMode(t, chatID, "task-existing", "run-existing")

	w := env.doRequest("POST", "/api/v1/chats/"+chatID+"/agent-mode/attach", map[string]interface{}{
		"run_id":  "run-new",
		"task_id": "task-new",
	})

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
	assertErrorCode(t, w, "V014")
}

func TestAttachAgentRun_RunNotFound(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)

	env.mockAgent.StatusError = fmt.Errorf("HTTP 404: run not found")

	w := env.doRequest("POST", "/api/v1/chats/"+chatID+"/agent-mode/attach", map[string]interface{}{
		"run_id":  "run-nonexistent",
		"task_id": "task-1",
	})

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAttachAgentRun_MissingFields(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)

	tests := []struct {
		name string
		body map[string]interface{}
	}{
		{"missing run_id", map[string]interface{}{"task_id": "task-1"}},
		{"missing task_id", map[string]interface{}{"run_id": "run-1"}},
		{"empty run_id", map[string]interface{}{"run_id": "", "task_id": "task-1"}},
		{"empty task_id", map[string]interface{}{"run_id": "run-1", "task_id": ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := env.doRequest("POST", "/api/v1/chats/"+chatID+"/agent-mode/attach", tt.body)
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}
