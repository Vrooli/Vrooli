package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	openRouterAPIURL      = "https://openrouter.ai/api/v1/chat/completions"
	openRouterKeyCheckURL = "https://openrouter.ai/api/v1/auth/key"
	byokDefaultModel      = "openai/gpt-4o-mini"
	byokRequestTimeout    = 120 * time.Second
	byokKeyValidationTTL  = 5 * time.Minute
)

// BYOKProvider implements AIProvider using the user's OpenRouter API key.
// This provider does not charge credits since the user is using their own key.
type BYOKProvider struct {
	log    *logrus.Logger
	apiKey string
	model  string
	client *http.Client

	// Key validation cache
	cacheMu       sync.RWMutex
	keyValid      bool
	keyValidUntil time.Time
}

// BYOKProviderOptions configures the BYOK provider.
type BYOKProviderOptions struct {
	Logger *logrus.Logger
	APIKey string
	Model  string
}

// NewBYOKProvider creates a new BYOK provider with the user's OpenRouter key.
func NewBYOKProvider(opts BYOKProviderOptions) *BYOKProvider {
	model := opts.Model
	if model == "" {
		model = byokDefaultModel
	}

	return &BYOKProvider{
		log:    opts.Logger,
		apiKey: opts.APIKey,
		model:  model,
		client: &http.Client{
			Timeout: byokRequestTimeout,
		},
	}
}

// Type implements AIProvider.
func (p *BYOKProvider) Type() ProviderType {
	return ProviderTypeBYOK
}

// IsAvailable implements AIProvider.
// Checks if the API key is valid by making a lightweight validation request.
func (p *BYOKProvider) IsAvailable(ctx context.Context) bool {
	if p.apiKey == "" {
		return false
	}

	// Check cache first
	p.cacheMu.RLock()
	if time.Now().Before(p.keyValidUntil) {
		valid := p.keyValid
		p.cacheMu.RUnlock()
		return valid
	}
	p.cacheMu.RUnlock()

	// Validate key with OpenRouter
	valid := p.validateKey(ctx)

	// Update cache
	p.cacheMu.Lock()
	p.keyValid = valid
	p.keyValidUntil = time.Now().Add(byokKeyValidationTTL)
	p.cacheMu.Unlock()

	return valid
}

// validateKey checks if the API key is valid with OpenRouter.
func (p *BYOKProvider) validateKey(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, openRouterKeyCheckURL, nil)
	if err != nil {
		p.log.WithError(err).Debug("Failed to create key validation request")
		return false
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		p.log.WithError(err).Debug("BYOK key validation request failed")
		return false
	}
	defer resp.Body.Close()

	// OpenRouter returns 200 for valid keys
	return resp.StatusCode == http.StatusOK
}

// ExecutePrompt implements AIProvider.
func (p *BYOKProvider) ExecutePrompt(ctx context.Context, prompt string) (string, error) {
	if p.apiKey == "" {
		return "", ErrInvalidBYOKKey
	}

	reqBody := openRouterRequest{
		Model: p.model,
		Messages: []openRouterMessage{
			{Role: "user", Content: prompt},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openRouterAPIURL, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://vrooli.com")
	req.Header.Set("X-Title", "Browser Automation Studio")

	start := time.Now()
	resp, err := p.client.Do(req)
	duration := time.Since(start)

	if err != nil {
		p.log.WithError(err).WithFields(logrus.Fields{
			"model":    p.model,
			"duration": duration.Milliseconds(),
		}).Error("BYOK OpenRouter request failed")
		return "", fmt.Errorf("openrouter request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		// Invalidate cached key
		p.cacheMu.Lock()
		p.keyValid = false
		p.keyValidUntil = time.Time{}
		p.cacheMu.Unlock()
		return "", ErrInvalidBYOKKey
	}

	if resp.StatusCode != http.StatusOK {
		p.log.WithFields(logrus.Fields{
			"status": resp.StatusCode,
			"body":   string(body),
		}).Error("BYOK OpenRouter request returned error")
		return "", fmt.Errorf("openrouter returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp openRouterResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return "", errors.New("openrouter returned no choices")
	}

	content := apiResp.Choices[0].Message.Content
	if content == "" {
		return "", errors.New("openrouter returned empty response")
	}

	p.log.WithFields(logrus.Fields{
		"model":    p.model,
		"duration": duration.Milliseconds(),
		"provider": "byok",
	}).Debug("BYOK OpenRouter request completed")

	return content, nil
}

// Model implements AIProvider.
func (p *BYOKProvider) Model() string {
	return p.model
}

// OpenRouter API types
type openRouterRequest struct {
	Model    string              `json:"model"`
	Messages []openRouterMessage `json:"messages"`
}

type openRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openRouterResponse struct {
	Choices []openRouterChoice `json:"choices"`
}

type openRouterChoice struct {
	Message openRouterMessage `json:"message"`
}

// Compile-time interface check
var _ AIProvider = (*BYOKProvider)(nil)
