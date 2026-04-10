package handlers

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"agent-inbox/domain"
	"agent-inbox/integrations"
)

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

func TestSendAgentMessage_RunStillActive(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)
	env.setAgentMode(t, chatID, "task-1", "run-1")
	env.mockAgent.StatusResult = &integrations.AgentRunStatus{Status: "running"}

	w := env.doRequest("POST", "/api/v1/chats/"+chatID+"/agent-mode/message", map[string]interface{}{
		"message": "follow up",
	})

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
	assertErrorCode(t, w, string(domain.ErrCodeAgentRunBusy))

	// Verify ContinueChat was NOT called
	if len(env.mockAgent.ContinueCalls) != 0 {
		t.Errorf("expected 0 ContinueChat calls, got %d", len(env.mockAgent.ContinueCalls))
	}
}

func TestSendAgentMessage_Success(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)
	env.setAgentMode(t, chatID, "task-1", "run-1")
	env.mockAgent.StatusResult = &integrations.AgentRunStatus{Status: "complete"}

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

func TestSendAgentMessage_AutoNamesDefaultChat(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChatWithName(t, "New Chat")
	env.setAgentMode(t, chatID, "task-1", "run-1")
	env.mockAgent.StatusResult = &integrations.AgentRunStatus{Status: "complete"}
	configureOllamaNaming(t, env.handler, "Follow Up Debug Session")

	w := env.doRequest("POST", "/api/v1/chats/"+chatID+"/agent-mode/message", map[string]interface{}{
		"message": "Continue with fixing flaky tests",
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	chat, err := env.repo.GetChat(context.Background(), chatID)
	if err != nil {
		t.Fatalf("GetChat failed: %v", err)
	}
	if chat == nil {
		t.Fatal("expected chat to exist")
	}
	if chat.Name != "Follow Up Debug Session" {
		t.Fatalf("expected auto-named chat, got %q", chat.Name)
	}
}
