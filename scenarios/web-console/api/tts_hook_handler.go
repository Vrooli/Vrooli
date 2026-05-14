package main

import (
	"net/http"
	"time"

	"web-console/internal/sessionstore"
)

type hookStopRequest struct {
	AssistantResponse    string `json:"assistantResponse"`
	LastAssistantMessage string `json:"last_assistant_message"`
	HookEventName        string `json:"hook_event_name"`
	SessionIDSnake       string `json:"session_id"`
	WebConsoleSessionID  string `json:"web_console_session_id"`
	CWD                  string `json:"cwd"`
}

func (r hookStopRequest) assistantText() string {
	if r.LastAssistantMessage != "" {
		return r.LastAssistantMessage
	}
	return r.AssistantResponse
}

func (s *Server) handleHookStop(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Hook-Token")
	if token == "" || token != s.hookAuthToken {
		writeCatalogError(w, "unauthorized", "Invalid or missing hook token")
		return
	}
	var req hookStopRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	assistantText := req.assistantText()
	if assistantText == "" {
		writeCatalogError(w, "tts_input_required", "Hook payload did not include assistant response text")
		return
	}
	result := s.AppendAssistant(assistantText, req.WebConsoleSessionID, "claude_hook")
	// Phase 3 (recovery hardening): persist agent identity from the hook
	// payload so the recovery flow can later issue
	// `claude --resume <agent_session_id>` against the right project. The
	// payload's session_id is Claude's own session UUID.
	if result.Appended && req.WebConsoleSessionID != "" && req.SessionIDSnake != "" && s.sessionStore != nil {
		_ = s.sessionStore.UpdateAgentInfo(req.WebConsoleSessionID, sessionstore.AgentInfo{
			AgentType:      sessionstore.AgentClaude,
			AgentSessionID: req.SessionIDSnake,
			CWD:            req.CWD,
			LastActivityAt: time.Now(),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "routing": result, "routed": result.Appended})
}
