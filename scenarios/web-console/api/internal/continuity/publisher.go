package continuity

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

// AgentManagerPublisher adapts the durable Web Console publication queue to
// Agent Manager's existing transcript-import authority. It never sends raw
// event text or a Web Console catalog; Agent Manager reads the native
// transcript through its own parser and owns the resulting search projection.
type AgentManagerPublisher struct {
	BaseURL string
	Client  *http.Client
}

func agentManagerRunnerType(agentType string) string {
	switch strings.TrimSpace(strings.ToLower(agentType)) {
	case "claude":
		return "claude-code"
	default:
		return agentType
	}
}

func (p *AgentManagerPublisher) Publish(ctx context.Context, record PublicationRecord) error {
	if p == nil || strings.TrimSpace(p.BaseURL) == "" {
		return fmt.Errorf("agent manager publication endpoint is unavailable")
	}
	if strings.TrimSpace(record.RolloutRef) == "" {
		return fmt.Errorf("session %q has no native transcript source", record.SessionID)
	}
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	payload := struct {
		Path            string `json:"path"`
		RunnerType      string `json:"runnerType"`
		Label           string `json:"label,omitempty"`
		SourceHarness   string `json:"sourceHarness"`
		SourceSessionID string `json:"sourceSessionId"`
	}{
		Path: record.RolloutRef, RunnerType: agentManagerRunnerType(record.AgentType), Label: record.CurrentTitle,
		SourceHarness: "web-console", SourceSessionID: record.SessionID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	endpoint := strings.TrimRight(p.BaseURL, "/") + "/api/v1/runs/import-transcript"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("publish session %q: %w", record.SessionID, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		detail, readErr := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		if readErr == nil && strings.TrimSpace(string(detail)) != "" {
			return fmt.Errorf("publish session %q: agent manager returned %s: %s", record.SessionID, resp.Status, strings.TrimSpace(string(detail)))
		}
		return fmt.Errorf("publish session %q: agent manager returned %s", record.SessionID, resp.Status)
	}
	return nil
}

func (p *AgentManagerPublisher) Tombstone(ctx context.Context, record PublicationRecord) error {
	if p == nil || strings.TrimSpace(p.BaseURL) == "" {
		return fmt.Errorf("agent manager publication endpoint is unavailable")
	}
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	body, err := json.Marshal(struct {
		SourceHarness   string `json:"sourceHarness"`
		SourceSessionID string `json:"sourceSessionId"`
	}{SourceHarness: "web-console", SourceSessionID: record.SessionID})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(p.BaseURL, "/")+"/api/v1/runs/external/tombstone", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// A tombstone is an idempotent delete. Agent Manager returns 404 when the
	// source was never imported (or was already removed); that state already
	// satisfies the deletion contract and must converge the durable queue.
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("tombstone session %q: agent manager returned %s", record.SessionID, resp.Status)
	}
	return nil
}
