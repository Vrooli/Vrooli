package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

// DOC: docs/concepts/ARCHITECTURE.md#ai-command-generation

// [REQ:P0-005a] AI Command Generation API
//
// Provider chain: Ollama (local) -> OpenRouter (cloud) with deterministic failover.
// The endpoint accepts a prompt plus optional terminal context and returns a
// generated command suggestion.

// commandSystemPrompt is the system instruction for command generation.
const commandSystemPrompt = "You are a command-line assistant. Given a natural language description, output ONLY the shell command. No explanation, no markdown, no backticks."

// AIGenerateRequest is the JSON body for the AI command generation endpoint.
type AIGenerateRequest struct {
	Prompt  string `json:"prompt"`
	Context string `json:"context,omitempty"` // terminal context (cwd, recent output, etc.)
}

// AIGenerateResponse is the JSON body returned by the AI command generation endpoint.
type AIGenerateResponse struct {
	Command  string `json:"command"`
	Provider string `json:"provider"` // "ollama" or "openrouter"
}

// knownCodeFences lists the markdown code fence prefixes that AI providers
// commonly wrap shell commands in. Order matters: longer/more-specific prefixes
// are tried first so that e.g. "```bash" is removed before the generic "```".
// Only shell-related fences are stripped; fences for other languages (python,
// ruby, etc.) are left intact so the caller sees that the AI returned the
// wrong type of output.
var knownCodeFences = []string{"```bash", "```sh", "```"}

// extractCommand cleans up AI output by stripping known markdown code fences,
// trimming whitespace, and selecting only the first line. This is the single
// place where the "raw AI text → executable command" decision is made.
func extractCommand(raw string) string {
	s := strings.TrimSpace(raw)
	for _, fence := range knownCodeFences {
		s = strings.TrimPrefix(s, fence)
	}
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	// Take only the first line if multiple lines returned
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		s = s[:idx]
	}
	return strings.TrimSpace(s)
}

// handleAIGenerate handles POST /api/v1/ai/generate
// [REQ:P0-005a] AI Command Generation API
func (s *Server) handleAIGenerate(w http.ResponseWriter, r *http.Request) {
	reqID := getRequestID(r)

	var req AIGenerateRequest
	if !decodeJSON(w, r, &req) {
		log.Printf("ai-generate [%s]: malformed JSON body", reqID)
		return
	}

	if strings.TrimSpace(req.Prompt) == "" {
		writeCatalogError(w, "invalid_body",
			"Prompt is required")
		return
	}

	userPrompt := req.Prompt
	if req.Context != "" {
		userPrompt = fmt.Sprintf("%s\n\nTerminal context: %s", req.Prompt, req.Context)
	}

	raw, provider, err := s.executeAI(r.Context(), commandSystemPrompt, userPrompt)
	if err != nil {
		log.Printf("ai-generate [%s]: all providers failed: %v", reqID, err)
		writeCatalogError(w, "ai_provider_unavailable",
			"AI command generation is currently unavailable. Check that Ollama is running or OPENROUTER_API_KEY is set.")
		return
	}

	command := extractCommand(raw)

	s.events.Emit(EventAIGenerate, "", map[string]string{
		"provider": provider,
		"prompt":   req.Prompt,
	})
	s.metrics.AIGenerations.Add(1)

	writeJSON(w, http.StatusOK, AIGenerateResponse{
		Command:  command,
		Provider: provider,
	})
}
