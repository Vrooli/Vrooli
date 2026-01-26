// Package promptmanager provides an HTTP client for the prompt-manager service.
package promptmanager

// DOC: docs/concepts/ARCHITECTURE.md#integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// Client is an HTTP client for prompt-manager.
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new prompt-manager client.
func NewClient(timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
	}
}

// GetSkill loads a skill by ID and returns its content.
func (c *Client) GetSkill(ctx context.Context, id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("skill id is required")
	}
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/v1/skills/%s", baseURL, id), nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("prompt-manager error: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	return payload.Content, nil
}

func (c *Client) resolveBaseURL(ctx context.Context) (string, error) {
	url, err := discovery.ResolveScenarioURLDefault(ctx, "prompt-manager")
	if err != nil {
		return "", fmt.Errorf("resolve prompt-manager url: %w", err)
	}
	return strings.TrimRight(url, "/"), nil
}
