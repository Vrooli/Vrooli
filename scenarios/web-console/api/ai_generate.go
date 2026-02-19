package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// DOC: docs/concepts/ARCHITECTURE.md#ai-command-generation

// [REQ:P0-005a] AI Command Generation API
//
// Provider chain: Ollama (local) -> OpenRouter (cloud) with deterministic failover.
// The endpoint accepts a prompt plus optional terminal context and returns a
// generated command suggestion.

// defaultProviderTimeout is the fallback timeout for AI provider HTTP calls.
// CROSS-LANGUAGE COUPLING: ai_provider_config.go defaults also use 30s.
const defaultProviderTimeout = 30 * time.Second

// systemPromptPrefix is the system instruction shared by all providers.
const systemPromptPrefix = "You are a command-line assistant. Given a natural language description, output ONLY the shell command. No explanation, no markdown, no backticks."

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

// AIProvider abstracts a single AI provider for command generation.
type AIProvider interface {
	Name() string
	Generate(ctx context.Context, prompt string) (string, error)
}

// AIProviderChain tries each provider in order until one succeeds.
type AIProviderChain struct {
	providers []AIProvider
}

// NewAIProviderChain creates a chain with the given providers.
func NewAIProviderChain(providers ...AIProvider) *AIProviderChain {
	return &AIProviderChain{providers: providers}
}

// Generate tries each provider in order, returning the first successful result.
func (c *AIProviderChain) Generate(ctx context.Context, prompt string) (command string, provider string, err error) {
	var lastErr error
	for _, p := range c.providers {
		result, provErr := p.Generate(ctx, prompt)
		if provErr != nil {
			log.Printf("ai-generate: provider %s failed: %v", p.Name(), provErr)
			lastErr = provErr
			continue
		}
		return result, p.Name(), nil
	}
	if lastErr != nil {
		return "", "", fmt.Errorf("all providers failed, last error: %w", lastErr)
	}
	return "", "", fmt.Errorf("no providers configured")
}

// OllamaProvider calls the local Ollama API.
type OllamaProvider struct {
	BaseURL string
	Model   string
	Client  *http.Client
}

// NewOllamaProvider creates an Ollama provider with env-configurable settings.
func NewOllamaProvider() *OllamaProvider {
	baseURL := os.Getenv("OLLAMA_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	model := os.Getenv("WC_OLLAMA_MODEL")
	if model == "" {
		model = "llama3.2"
	}
	return &OllamaProvider{
		BaseURL: baseURL,
		Model:   model,
		Client:  &http.Client{Timeout: defaultProviderTimeout},
	}
}

func (o *OllamaProvider) Name() string { return "ollama" }

func (o *OllamaProvider) Generate(ctx context.Context, prompt string) (string, error) {
	body := map[string]any{
		"model":  o.Model,
		"prompt": buildSystemPrompt(prompt),
		"stream": false,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.BaseURL+"/api/generate", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := checkProviderResponse(resp, "ollama"); err != nil {
		return "", err
	}

	var result struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return extractCommand(result.Response), nil
}

// OpenRouterProvider calls the OpenRouter API as a fallback.
type OpenRouterProvider struct {
	APIKey string
	Model  string
	Client *http.Client
}

// NewOpenRouterProvider creates an OpenRouter provider with env-configurable settings.
func NewOpenRouterProvider() *OpenRouterProvider {
	model := os.Getenv("WC_OPENROUTER_MODEL")
	if model == "" {
		model = "meta-llama/llama-3-8b-instruct"
	}
	return &OpenRouterProvider{
		APIKey: os.Getenv("OPENROUTER_API_KEY"),
		Model:  model,
		Client: &http.Client{Timeout: defaultProviderTimeout},
	}
}

func (o *OpenRouterProvider) Name() string { return "openrouter" }

func (o *OpenRouterProvider) Generate(ctx context.Context, prompt string) (string, error) {
	if o.APIKey == "" {
		return "", fmt.Errorf("OPENROUTER_API_KEY not set")
	}

	body := map[string]any{
		"model": o.Model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPromptPrefix},
			{"role": "user", "content": prompt},
		},
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.APIKey)

	resp, err := o.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := checkProviderResponse(resp, "openrouter"); err != nil {
		return "", err
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return extractCommand(result.Choices[0].Message.Content), nil
}

// checkProviderResponse returns an error if the HTTP response from an AI
// provider indicates failure (non-200 status). This is the single place
// where the "did the upstream provider accept our request?" decision is made.
func checkProviderResponse(resp *http.Response, providerName string) error {
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("%s returned %d: %s", providerName, resp.StatusCode, string(body))
	}
	return nil
}

// buildSystemPrompt wraps the user prompt with command-generation context.
func buildSystemPrompt(userPrompt string) string {
	return fmt.Sprintf(
		"%s\n\nDescription: %s",
		systemPromptPrefix,
		userPrompt,
	)
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

// generateWithConfig wraps the AI provider chain with config-driven behavior:
// respects enabled/disabled, per-provider timeouts, and records health metrics.
func (s *Server) generateWithConfig(ctx context.Context, prompt string) (command, provider string, err error) {
	configs := s.aiConfig.GetConfigs()

	for _, cfg := range configs {
		if !cfg.Enabled {
			continue
		}

		// Find the matching provider in the chain
		var p AIProvider
		for _, cp := range s.aiChain.providers {
			if cp.Name() == cfg.Name {
				p = cp
				break
			}
		}
		if p == nil {
			continue
		}

		timeout := time.Duration(cfg.TimeoutSec) * time.Second
		pCtx, cancel := context.WithTimeout(ctx, timeout)

		start := time.Now()
		result, pErr := p.Generate(pCtx, prompt)
		elapsed := time.Since(start)
		cancel()

		if pErr != nil {
			s.aiConfig.RecordError(cfg.Name)
			err = pErr
			continue
		}

		s.aiConfig.RecordSuccess(cfg.Name, elapsed)
		return result, cfg.Name, nil
	}

	if err != nil {
		return "", "", err
	}
	return "", "", fmt.Errorf("no enabled providers configured")
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

	prompt := req.Prompt
	if req.Context != "" {
		prompt = fmt.Sprintf("%s\n\nTerminal context: %s", req.Prompt, req.Context)
	}

	command, provider, err := s.generateWithConfig(r.Context(), prompt)
	if err != nil {
		log.Printf("ai-generate [%s]: all providers failed: %v", reqID, err)
		writeCatalogError(w, "ai_provider_unavailable",
			"AI command generation is currently unavailable. Check that Ollama is running or OPENROUTER_API_KEY is set.")
		return
	}

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
