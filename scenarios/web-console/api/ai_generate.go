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

// suggestSystemPrompt is the system instruction for multi-command suggestion.
const suggestSystemPrompt = "You are a command-line assistant. Given a natural language description, output 1 to 3 shell commands (one per line) that accomplish the task. If there is only one reasonable command, output just that one. No explanation, no numbering, no markdown, no backticks."

// AISuggestRequest is the JSON body for the AI suggest endpoint.
type AISuggestRequest struct {
	Prompt  string `json:"prompt"`
	Context string `json:"context,omitempty"`
}

// AISuggestResponse is the JSON body returned by the AI suggest endpoint.
type AISuggestResponse struct {
	Commands []string `json:"commands"`
	Provider string   `json:"provider"`
}

// extractCommands splits raw AI output into 1–3 individual commands.
// Each line is cleaned of markdown fences and whitespace; empty lines are dropped.
func extractCommands(raw string) []string {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	var commands []string
	for _, line := range lines {
		cmd := extractCommand(line)
		if cmd != "" {
			commands = append(commands, cmd)
		}
		if len(commands) >= 3 {
			break
		}
	}
	return commands
}

// handleAISuggest handles POST /api/v1/ai/suggest
func (s *Server) handleAISuggest(w http.ResponseWriter, r *http.Request) {
	reqID := getRequestID(r)

	var req AISuggestRequest
	if !decodeJSON(w, r, &req) {
		log.Printf("ai-suggest [%s]: malformed JSON body", reqID)
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

	raw, provider, err := s.executeAI(r.Context(), buildSuggestSystemPrompt(s.systemContext), userPrompt)
	if err != nil {
		log.Printf("ai-suggest [%s]: all providers failed: %v", reqID, err)
		writeCatalogError(w, "ai_provider_unavailable",
			"AI suggestion is currently unavailable. Check that Ollama is running or OPENROUTER_API_KEY is set.")
		return
	}

	commands := extractCommands(raw)

	s.events.Emit(EventAISuggest, "", map[string]string{
		"provider": provider,
		"prompt":   req.Prompt,
		"count":    fmt.Sprintf("%d", len(commands)),
	})
	s.metrics.AISuggestions.Add(1)

	writeJSON(w, http.StatusOK, AISuggestResponse{
		Commands: commands,
		Provider: provider,
	})
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

	raw, provider, err := s.executeAI(r.Context(), buildCommandSystemPrompt(s.systemContext), userPrompt)
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
