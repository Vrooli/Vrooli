package handlers

import (
	"agent-inbox/domain"
	"agent-inbox/integrations"
	"context"
	"fmt"
	"net/http"
	"testing"
)

// =============================================================================
// StartAgentMode Handler Tests (DB required)
// =============================================================================

func TestAddMessage_AutoNamesDefaultChat(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChatWithName(t, "New Chat")
	configureOllamaNaming(t, env.handler, "Repository Scaffolding")

	w := env.doRequest("POST", "/api/v1/chats/"+chatID+"/messages", map[string]interface{}{
		"role":    "user",
		"content": "Please scaffold a starter repository layout",
	})

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	chat, err := env.repo.GetChat(context.Background(), chatID)
	if err != nil {
		t.Fatalf("GetChat failed: %v", err)
	}
	if chat == nil {
		t.Fatal("expected chat to exist")
	}
	if chat.Name != "Repository Scaffolding" {
		t.Fatalf("expected auto-named chat, got %q", chat.Name)
	}
}

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

func TestStartAgentMode_AutoNamesDefaultChat(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChatWithName(t, "New Chat")
	projectDir := t.TempDir()
	configureOllamaNaming(t, env.handler, "Agent Task Planning")

	env.mockAgent.StartResult = &integrations.AgentChatSession{
		TaskID: "task-auto-1",
		RunID:  "run-auto-1",
	}

	w := env.doRequest("POST", "/api/v1/chats/"+chatID+"/agent-mode/start", map[string]interface{}{
		"message":      "Build me a release checklist",
		"project_path": projectDir,
		"runner_type":  "claude-code",
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
	if chat.Name != "Agent Task Planning" {
		t.Fatalf("expected auto-named chat, got %q", chat.Name)
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

func TestStartAgentMode_RejectsUnsupportedRunnerType(t *testing.T) {
	env := setupAgentModeTest(t)
	chatID := env.createChat(t)

	w := env.doRequest("POST", "/api/v1/chats/"+chatID+"/agent-mode/start", map[string]interface{}{
		"message":      "hello",
		"project_path": "/tmp",
		"runner_type":  "grok",
	})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	if len(env.mockAgent.StartCalls) != 0 {
		t.Fatalf("unsupported runner should not start agent-manager call")
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
