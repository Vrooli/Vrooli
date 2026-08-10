// Package handlers provides HTTP handlers for the Agent Inbox API.
// This file implements the skill suggestion endpoint.
package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"agent-inbox/services"
)

// SuggestSkills handles POST /api/v1/skills/suggest.
// Returns AI-powered skill suggestions based on conversation context.
// Gracefully degrades: returns empty suggestions on any error (never 500).
func (h *Handlers) SuggestSkills(w http.ResponseWriter, r *http.Request) {
	if h.SkillSuggest == nil {
		h.JSONResponse(w, &services.SuggestResponse{
			Suggestions: []services.SuggestedSkill{},
		}, http.StatusOK)
		return
	}

	var req services.SuggestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.JSONResponse(w, &services.SuggestResponse{
			Suggestions: []services.SuggestedSkill{},
		}, http.StatusOK)
		return
	}

	// Require at least one of chatId or inputText
	if req.ChatID == "" && req.InputText == "" {
		h.JSONResponse(w, &services.SuggestResponse{
			Suggestions: []services.SuggestedSkill{},
		}, http.StatusOK)
		return
	}

	resp, err := h.SkillSuggest.SuggestSkills(r.Context(), h.Repo, &req)
	if err != nil {
		log.Printf("skill suggest error: %v", err)
		h.JSONResponse(w, &services.SuggestResponse{
			Suggestions: []services.SuggestedSkill{},
		}, http.StatusOK)
		return
	}

	h.JSONResponse(w, resp, http.StatusOK)
}
