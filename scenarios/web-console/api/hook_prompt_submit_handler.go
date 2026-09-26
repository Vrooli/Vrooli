package main

import (
	"log"
	"net/http"
	"strings"
)

type hookPromptSubmitRequest struct {
	UserPrompt          string `json:"userPrompt"`
	WebConsoleSessionID string `json:"webConsoleSessionId"`
}

// handleHookPromptSubmit receives user prompts from Claude Code hooks
// and appends them to the conversation store.
// POST /api/v1/hooks/prompt-submit
func (s *Server) handleHookPromptSubmit(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Hook-Token")
	if token == "" || token != s.hookAuthToken {
		writeCatalogError(w, "unauthorized", "Missing or invalid X-Hook-Token header")
		return
	}

	var req hookPromptSubmitRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if strings.TrimSpace(req.UserPrompt) == "" {
		writeCatalogError(w, "invalid_body", "userPrompt is required")
		return
	}

	if strings.TrimSpace(req.WebConsoleSessionID) == "" {
		writeCatalogError(w, "invalid_body", "webConsoleSessionId is required")
		return
	}

	result := s.AppendUser(req.UserPrompt, req.WebConsoleSessionID, "prompt_submit_hook")
	log.Printf("hook-prompt-submit: session=%s appended=%v code=%s",
		sanitizeID(req.WebConsoleSessionID), result.Appended, result.Code)

	writeJSON(w, http.StatusOK, result)
}
