// Package testing provides LLM-based prompt testing via Ollama.
package testing

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OllamaClient handles communication with the Ollama API.
// This is a testing seam: inject a mock client to test without a real Ollama instance.
type OllamaClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewOllamaClient creates a new Ollama client.
// If baseURL is empty, the client is considered disabled.
func NewOllamaClient(baseURL string) *OllamaClient {
	return &OllamaClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// IsEnabled returns true if Ollama is configured.
func (c *OllamaClient) IsEnabled() bool {
	return c.baseURL != ""
}

// Generate runs a prompt through Ollama and returns the response.
func (c *OllamaClient) Generate(model, prompt string, maxTokens int, temperature float64) (*OllamaResponse, float64, error) {
	if !c.IsEnabled() {
		return nil, 0, fmt.Errorf("ollama not configured")
	}

	req := OllamaRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
		Options: map[string]interface{}{
			"num_predict": maxTokens,
			"temperature": temperature,
		},
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	startTime := time.Now()

	resp, err := c.httpClient.Post(c.baseURL+"/api/generate", "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to call Ollama: %w", err)
	}
	defer resp.Body.Close()

	responseTime := float64(time.Since(startTime).Milliseconds())

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, responseTime, fmt.Errorf("ollama error: %s", string(body))
	}

	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, responseTime, fmt.Errorf("failed to parse response: %w", err)
	}

	return &ollamaResp, responseTime, nil
}
