package health

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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

// ImageInput is the provider-neutral image payload accepted by the resource
// command surface. DataB64 is converted into a provider content part only at
// the HTTP boundary and is never included in command output.
type ImageInput struct {
	MediaType string
	DataB64   string
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
func Generate(ctx context.Context, client HTTPClient, runtime resourceenv.Runtime, creds auth.Credentials, model, prompt string, temperature float64, maxTokens int, responseFormat json.RawMessage, images []ImageInput) ([]byte, error) {
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
			Content any    `json:"content"`
		} `json:"messages"`
		Temperature    float64         `json:"temperature"`
		MaxTokens      int             `json:"max_tokens,omitempty"`
		ResponseFormat json.RawMessage `json:"response_format,omitempty"`
	}{
		Model:          model,
		Temperature:    temperature,
		MaxTokens:      maxTokens,
		ResponseFormat: responseFormat,
	}
	content := any(prompt)
	if len(images) > 0 {
		parts := make([]map[string]any, 0, len(images)+1)
		parts = append(parts, map[string]any{"type": "text", "text": prompt})
		for _, image := range images {
			parts = append(parts, map[string]any{
				"type": "image_url",
				"image_url": map[string]string{
					"url": "data:" + image.MediaType + ";base64," + image.DataB64,
				},
			})
		}
		content = parts
	}
	payload.Messages = []struct {
		Role    string `json:"role"`
		Content any    `json:"content"`
	}{{Role: "user", Content: content}}

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

// GenerateImage performs the resource-owned OpenRouter image request. Callers
// provide a policy-resolved model and prompt; credentials, base URL, transport,
// and provider error handling remain inside this resource.
func GenerateImage(ctx context.Context, client HTTPClient, runtime resourceenv.Runtime, creds auth.Credentials, model, prompt, inputFile string, outputCount int) ([]byte, error) {
	if client == nil {
		client = http.DefaultClient
	}
	endpoint := config.ImagesEndpoint(runtime.APIBaseURL)
	if endpoint == "" {
		return nil, fmt.Errorf("OpenRouter images endpoint is required")
	}
	if strings.TrimSpace(model) == "" || strings.TrimSpace(prompt) == "" {
		return nil, fmt.Errorf("image model and prompt are required")
	}
	payload := map[string]any{"model": model, "prompt": prompt}
	if outputCount > 1 {
		payload["n"] = outputCount
	}
	if strings.TrimSpace(inputFile) != "" {
		image, err := os.ReadFile(inputFile)
		if err != nil {
			return nil, fmt.Errorf("read image input: %w", err)
		}
		if len(image) == 0 {
			return nil, fmt.Errorf("image input is empty")
		}
		// OpenRouter accepts provider-neutral input references. Keeping the
		// encoding here makes image-to-image and instructed edits a resource
		// capability instead of leaking provider transport into callers.
		payload["input_references"] = []any{map[string]any{
			"type": "image_url",
			"image_url": map[string]string{
				"url": "data:image/png;base64," + base64.StdEncoding.EncodeToString(image),
			},
		}}
	}
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
		return nil, fmt.Errorf("OpenRouter image generation failed with status %d", resp.StatusCode)
	}
	return responseBody, nil
}
