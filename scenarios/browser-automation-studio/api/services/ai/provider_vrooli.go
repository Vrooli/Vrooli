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

// VrooliProvider implements AIProvider using the LPBS AI Gateway.
// This provider charges credits through LPBS when executing prompts.
type VrooliProvider struct {
	log        *logrus.Logger
	apiURL     string // LPBS base URL (e.g., "https://vrooli.com" or "http://localhost:15000")
	model      string
	httpClient *http.Client
	authToken  string // User's JWT token for LPBS authentication

	// Availability cache to avoid repeated health checks
	availableMu   sync.RWMutex
	available     bool
	lastCheck     time.Time
	checkInterval time.Duration
}

// VrooliProviderOptions configures the Vrooli API provider.
type VrooliProviderOptions struct {
	Logger        *logrus.Logger
	APIURL        string // The LPBS base URL (e.g., "https://vrooli.com")
	Model         string
	AuthToken     string        // User's JWT token for authentication
	HTTPClient    *http.Client  // Optional custom HTTP client
	CheckInterval time.Duration // How often to re-check availability (default: 30s)
}

// vrooliChatRequest is the request body for LPBS AI chat endpoint.
type vrooliChatRequest struct {
	Model    string          `json:"model"`
	Messages []vrooliMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Metadata vrooliMetadata  `json:"metadata,omitempty"`
}

type vrooliMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type vrooliMetadata struct {
	AppBundleKey string `json:"app_bundle_key,omitempty"`
	Operation    string `json:"operation,omitempty"`
}

// vrooliChatResponse is the response from LPBS AI chat endpoint.
type vrooliChatResponse struct {
	ID               string `json:"id"`
	Model            string `json:"model"`
	Content          string `json:"content"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
	CreditsCharged   int64  `json:"credits_charged"`
	FinishReason     string `json:"finish_reason,omitempty"`
}

// vrooliErrorResponse represents an error from LPBS.
type vrooliErrorResponse struct {
	Error     string `json:"error"`
	ErrorType string `json:"error_type"`
}

// NewVrooliProvider creates a new Vrooli API provider.
func NewVrooliProvider(opts VrooliProviderOptions) *VrooliProvider {
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 120 * time.Second}
	}

	checkInterval := opts.CheckInterval
	if checkInterval == 0 {
		checkInterval = 30 * time.Second
	}

	return &VrooliProvider{
		log:           opts.Logger,
		apiURL:        opts.APIURL,
		model:         opts.Model,
		httpClient:    httpClient,
		authToken:     opts.AuthToken,
		checkInterval: checkInterval,
	}
}

// Type implements AIProvider.
func (p *VrooliProvider) Type() ProviderType {
	return ProviderTypeVrooli
}

// IsAvailable implements AIProvider.
// Checks if the LPBS AI gateway is configured and reachable.
func (p *VrooliProvider) IsAvailable(ctx context.Context) bool {
	// Check if API URL is configured
	if p.apiURL == "" {
		return false
	}

	// Check if auth token is provided
	if p.authToken == "" {
		return false
	}

	// Use cached availability if recent enough
	p.availableMu.RLock()
	if time.Since(p.lastCheck) < p.checkInterval && !p.lastCheck.IsZero() {
		avail := p.available
		p.availableMu.RUnlock()
		return avail
	}
	p.availableMu.RUnlock()

	// Perform health check
	p.availableMu.Lock()
	defer p.availableMu.Unlock()

	// Double-check after acquiring write lock
	if time.Since(p.lastCheck) < p.checkInterval && !p.lastCheck.IsZero() {
		return p.available
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", p.apiURL+"/api/v1/ai/health", nil)
	if err != nil {
		p.available = false
		p.lastCheck = time.Now()
		return false
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		if p.log != nil {
			p.log.WithError(err).Debug("LPBS health check failed")
		}
		p.available = false
		p.lastCheck = time.Now()
		return false
	}
	defer resp.Body.Close()

	p.available = resp.StatusCode == http.StatusOK
	p.lastCheck = time.Now()

	if p.log != nil {
		p.log.WithFields(logrus.Fields{
			"available": p.available,
			"status":    resp.StatusCode,
		}).Debug("LPBS availability check completed")
	}

	return p.available
}

// ExecutePrompt implements AIProvider.
// Sends the prompt to LPBS AI gateway and returns the response.
func (p *VrooliProvider) ExecutePrompt(ctx context.Context, prompt string) (string, error) {
	if p.apiURL == "" {
		return "", ErrVrooliAPIUnavailable
	}

	if p.authToken == "" {
		return "", errors.New("LPBS authentication token not provided")
	}

	// Build the request
	reqBody := vrooliChatRequest{
		Model: p.model,
		Messages: []vrooliMessage{
			{Role: "user", Content: prompt},
		},
		Stream: false,
		Metadata: vrooliMetadata{
			AppBundleKey: "browser-automation-studio",
			Operation:    "ai.prompt",
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.apiURL+"/api/v1/ai/chat", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.authToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("LPBS request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)

		// Try to parse as error response
		var errResp vrooliErrorResponse
		if json.Unmarshal(bodyBytes, &errResp) == nil && errResp.Error != "" {
			// Map specific error types
			switch errResp.ErrorType {
			case "insufficient_credits":
				return "", ErrInsufficientCredits
			case "unauthorized":
				return "", errors.New("LPBS authentication failed")
			case "rate_limited":
				return "", errors.New("LPBS rate limit exceeded")
			}
			return "", errors.New(errResp.Error)
		}

		return "", fmt.Errorf("LPBS returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse successful response
	var result vrooliChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if p.log != nil {
		p.log.WithFields(logrus.Fields{
			"model":             result.Model,
			"prompt_tokens":     result.PromptTokens,
			"completion_tokens": result.CompletionTokens,
			"credits_charged":   result.CreditsCharged,
		}).Debug("LPBS request completed")
	}

	return result.Content, nil
}

// ExecutePromptWithMetadata executes a prompt with custom metadata for tracking.
func (p *VrooliProvider) ExecutePromptWithMetadata(ctx context.Context, prompt string, operation string) (string, error) {
	if p.apiURL == "" {
		return "", ErrVrooliAPIUnavailable
	}

	if p.authToken == "" {
		return "", errors.New("LPBS authentication token not provided")
	}

	reqBody := vrooliChatRequest{
		Model: p.model,
		Messages: []vrooliMessage{
			{Role: "user", Content: prompt},
		},
		Stream: false,
		Metadata: vrooliMetadata{
			AppBundleKey: "browser-automation-studio",
			Operation:    operation,
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.apiURL+"/api/v1/ai/chat", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.authToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("LPBS request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		var errResp vrooliErrorResponse
		if json.Unmarshal(bodyBytes, &errResp) == nil && errResp.Error != "" {
			if errResp.ErrorType == "insufficient_credits" {
				return "", ErrInsufficientCredits
			}
			return "", errors.New(errResp.Error)
		}
		return "", fmt.Errorf("LPBS returned status %d", resp.StatusCode)
	}

	var result vrooliChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return result.Content, nil
}

// Model implements AIProvider.
//
// Returns the model the provider was configured with (resolved by the chain
// through the OpenRouter resource policy, or an explicit override). No concrete
// default slug is baked in.
func (p *VrooliProvider) Model() string {
	return p.model
}

// SetAuthToken updates the authentication token.
// This is useful when the token is obtained after provider creation.
func (p *VrooliProvider) SetAuthToken(token string) {
	p.authToken = token
}

// ClearAvailabilityCache forces the next IsAvailable call to re-check.
func (p *VrooliProvider) ClearAvailabilityCache() {
	p.availableMu.Lock()
	p.lastCheck = time.Time{}
	p.availableMu.Unlock()
}

// ErrInsufficientCredits is returned when the user doesn't have enough credits.
var ErrInsufficientCredits = errors.New("insufficient credits for this operation")

// Compile-time interface check
var _ AIProvider = (*VrooliProvider)(nil)
