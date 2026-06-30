package health

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"resource-openrouter/cli/internal/auth"
	"resource-openrouter/cli/internal/config"

	resourceenv "resource-openrouter/cli/internal/env"
)

// HTTPClient is the narrow HTTP client contract used for safe OpenRouter probes.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// Result summarizes a safe OpenRouter connectivity probe.
type Result struct {
	Status        string
	Message       string
	Endpoint      string
	HTTPStatus    int
	Authenticated bool
}

// Probe performs a provider-safe GET against the OpenRouter models endpoint.
func Probe(ctx context.Context, client HTTPClient, runtime resourceenv.Runtime, creds auth.Credentials) (Result, error) {
	if client == nil {
		client = http.DefaultClient
	}
	endpoint := config.ModelsEndpoint(runtime.APIBaseURL)
	if endpoint == "" {
		return Result{}, fmt.Errorf("OpenRouter models endpoint is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Result{}, err
	}
	if creds.Valid() {
		req.Header.Set("Authorization", "Bearer "+creds.APIKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return Result{
			Status:        "unreachable",
			Message:       err.Error(),
			Endpoint:      endpoint,
			Authenticated: creds.Valid(),
		}, err
	}
	defer resp.Body.Close()

	result := Result{
		Endpoint:      endpoint,
		HTTPStatus:    resp.StatusCode,
		Authenticated: creds.Valid(),
	}
	switch resp.StatusCode {
	case http.StatusOK, http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden:
		result.Status = "reachable"
		result.Message = "OpenRouter API responded"
	default:
		result.Status = "degraded"
		result.Message = fmt.Sprintf("unexpected OpenRouter status: %d", resp.StatusCode)
	}
	return result, nil
}

// Generate performs a direct OpenRouter chat completion request.
func Generate(ctx context.Context, client HTTPClient, runtime resourceenv.Runtime, creds auth.Credentials, model, prompt string, temperature float64, maxTokens int) ([]byte, error) {
	if client == nil {
		client = http.DefaultClient
	}
	endpoint := config.ChatCompletionsEndpoint(runtime.APIBaseURL)
	if endpoint == "" {
		return nil, fmt.Errorf("OpenRouter chat completions endpoint is required")
	}
	payload := struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		Temperature float64 `json:"temperature"`
		MaxTokens   int     `json:"max_tokens,omitempty"`
	}{
		Model:       model,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}
	payload.Messages = []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}{{Role: "user", Content: prompt}}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+creds.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		var failure struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(responseBody, &failure) == nil && strings.TrimSpace(failure.Error.Message) != "" {
			return nil, fmt.Errorf("OpenRouter generate failed: %s", failure.Error.Message)
		}
		return nil, fmt.Errorf("OpenRouter generate failed with status %d", resp.StatusCode)
	}
	return responseBody, nil
}
