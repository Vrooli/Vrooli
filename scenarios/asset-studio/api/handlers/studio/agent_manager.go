package studio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// agentManagerCommissioner creates an ordinary Agent Manager task. Asset
// Studio never starts a run, selects a profile, or treats agent output as a
// verdict; those remain explicit operator decisions in Agent Manager.
type agentManagerCommissioner struct {
	BaseURL string
	Client  *http.Client
}

func (c *agentManagerCommissioner) CreateTask(ctx context.Context, request, identity string) (string, error) {
	if strings.TrimSpace(c.BaseURL) == "" {
		return "", fmt.Errorf("Agent Manager commissioner URL is not configured")
	}
	payload := map[string]any{"task": map[string]any{"title": "Asset Studio proposal", "description": request, "scopePath": "scenarios/asset-studio", "projectRoot": ".", "createdBy": "asset-studio:" + identity, "contextAttachments": []map[string]any{{"type": "note", "key": "asset-studio-boundary", "content": "This is an editable, untrusted proposal. It cannot judge conformance or release assets.", "format": "text"}}}}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+"/agent_manager.v1.AgentManagerService/CreateTask", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("create Agent Manager task: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("Agent Manager task creation returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var decoded struct {
		Task struct {
			Id string `json:"id"`
		} `json:"task"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return "", fmt.Errorf("decode Agent Manager task: %w", err)
	}
	if strings.TrimSpace(decoded.Task.Id) == "" {
		return "", fmt.Errorf("Agent Manager returned task without id")
	}
	return decoded.Task.Id, nil
}
