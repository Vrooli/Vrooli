package main

import "net/http"

type hookStopRequest struct {
	AssistantResponse    string `json:"assistantResponse"`
	SessionID            string `json:"sessionId"`
	LastAssistantMessage string `json:"last_assistant_message"`
	HookEventName        string `json:"hook_event_name"`
	SessionIDSnake       string `json:"session_id"`
}

func (r hookStopRequest) assistantText() string {
	if r.LastAssistantMessage != "" {
		return r.LastAssistantMessage
	}
	return r.AssistantResponse
}

func (r hookStopRequest) targetSessionID() string {
	if r.SessionIDSnake != "" {
		return r.SessionIDSnake
	}
	return r.SessionID
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
	result := s.deliverTTS(assistantText, req.targetSessionID(), "claude_hook")
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "delivery": result, "delivered": result.Delivered})
}
