package main

import "net/http"

type hookStopRequest struct {
	AssistantResponse string `json:"assistantResponse"`
	SessionID         string `json:"sessionId"`
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
	result := s.deliverTTS(req.AssistantResponse, req.SessionID, "claude_hook")
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "delivery": result, "delivered": result.Delivered})
}
