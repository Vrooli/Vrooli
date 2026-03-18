package main

import (
	"net/http"
)

type hookStopRequest struct {
	AssistantResponse    string `json:"assistantResponse"`
	LastAssistantMessage string `json:"last_assistant_message"`
	HookEventName        string `json:"hook_event_name"`
	SessionIDSnake       string `json:"session_id"`
	WebConsoleSessionID  string `json:"web_console_session_id"`
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
	result := s.routeTTSCandidate(assistantText, req.WebConsoleSessionID, "claude_hook")
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "routing": result, "routed": result.Routed})
}
