package main

import (
	"encoding/json"
	"net/http"
)

// handleAIChat handles AI chat requests
func (s *Server) handleAIChat(w http.ResponseWriter, r *http.Request) {
	var input AIRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// For now, return a mock response
	// In production, this would integrate with resource-openrouter or similar AI service
	response := AIResponse{
		Response: "I'm a mock AI response. In production, I would integrate with OpenRouter to provide real AI assistance.",
		Suggestions: []string{
			"Add TypeScript types",
			"Improve accessibility",
			"Add error handling",
		},
	}

	s.respondJSON(w, http.StatusOK, response)
}

// handleAIRefactor handles code refactoring requests
func (s *Server) handleAIRefactor(w http.ResponseWriter, r *http.Request) {
	var input AIRefactorRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// For now, return a mock response
	// In production, this would integrate with resource-openrouter or similar AI service
	response := AIRefactorResponse{
		RefactoredCode: input.Code + "\n// Refactored by AI (mock)",
		Diff:           "- Original code\n+ Refactored code (mock)",
	}

	s.respondJSON(w, http.StatusOK, response)
}
