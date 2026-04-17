package handlers

import (
	"agent-inbox/domain"
	"agent-inbox/testutil"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
