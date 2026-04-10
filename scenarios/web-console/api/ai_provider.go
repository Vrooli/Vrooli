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
	"time"
)

// defaultProviderTimeout is the fallback timeout for AI provider HTTP calls.
// CROSS-LANGUAGE COUPLING: ai_provider_config.go defaults also use 30s.
const defaultProviderTimeout = 30 * time.Second

// AIProvider abstracts a single AI provider for text generation.
// Providers are use-case-agnostic: the caller supplies both the system prompt
// (which sets the task context) and the user prompt.
type AIProvider interface {
	Name() string
	Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error)
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
func (c *AIProviderChain) Generate(ctx context.Context, systemPrompt, userPrompt string) (result string, provider string, err error) {
	var lastErr error
	for _, p := range c.providers {
		res, provErr := p.Generate(ctx, systemPrompt, userPrompt)
		if provErr != nil {
			log.Printf("ai-generate: provider %s failed: %v", p.Name(), provErr)
			lastErr = provErr
			continue
		}
		return res, p.Name(), nil
	}
	if lastErr != nil {
		return "", "", fmt.Errorf("all providers failed, last error: %w", lastErr)
	}
	return "", "", fmt.Errorf("no providers configured")
}

// OllamaProvider calls the local Ollama API using /api/chat for proper
// system/user message separation.
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

func (o *OllamaProvider) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	body := map[string]any{
		"model": o.Model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"stream": false,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.BaseURL+"/api/chat", bytes.NewReader(jsonBody))
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
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return result.Message.Content, nil
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

func (o *OpenRouterProvider) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if o.APIKey == "" {
		return "", fmt.Errorf("OPENROUTER_API_KEY not set")
	}

	body := map[string]any{
		"model": o.Model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
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

	return result.Choices[0].Message.Content, nil
}

// checkProviderResponse returns an error if the HTTP response from an AI
// provider indicates failure (non-200 status).
func checkProviderResponse(resp *http.Response, providerName string) error {
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("%s returned %d: %s", providerName, resp.StatusCode, string(body))
	}
	return nil
}

// executeAI wraps the AI provider chain with config-driven behavior:
// respects enabled/disabled, per-provider timeouts, and records health metrics.
// Returns raw LLM output with no post-processing.
func (s *Server) executeAI(ctx context.Context, systemPrompt, userPrompt string) (result, provider string, err error) {
	configs := s.aiConfig.GetConfigs()

	for _, cfg := range configs {
		if !cfg.Enabled {
			continue
		}

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
		res, pErr := p.Generate(pCtx, systemPrompt, userPrompt)
		elapsed := time.Since(start)
		cancel()

		if pErr != nil {
			s.aiConfig.RecordError(cfg.Name)
			err = pErr
			continue
		}

		s.aiConfig.RecordSuccess(cfg.Name, elapsed)
		return res, cfg.Name, nil
	}

	if err != nil {
		return "", "", err
	}
	return "", "", fmt.Errorf("no enabled providers configured")
}
