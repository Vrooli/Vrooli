package handlers

import (
	"net/http"
	"testing"
	"time"

	"agent-inbox/integrations"
)

// =============================================================================
// ListAgentRuns Handler Tests
// =============================================================================

func TestListAgentRuns_Success(t *testing.T) {
	env := setupAgentModeTest(t)

	now := time.Now().UTC().Truncate(time.Second)
	env.mockAgent.ListRunsResult = &integrations.ListRunsResult{
		Runs: []integrations.AgentRunSummary{
			{RunID: "run-1", TaskID: "task-1", Tag: "inbox-session", Status: integrations.RunStatusRunning, ProgressPercent: 50, CreatedAt: now},
			{RunID: "run-2", TaskID: "task-2", Tag: "cli-task", Status: integrations.RunStatusComplete, CreatedAt: now},
		},
		Total:   2,
		HasMore: false,
	}

	w := env.doRequest("GET", "/api/v1/agent-runs", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp integrations.ListRunsResult
	decodeBody(t, w, &resp)

	if len(resp.Runs) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(resp.Runs))
	}
	if resp.Total != 2 {
		t.Errorf("expected total 2, got %d", resp.Total)
	}
	if resp.Runs[0].RunID != "run-1" {
		t.Errorf("expected run-1, got %s", resp.Runs[0].RunID)
	}
}

func TestListAgentRuns_WithFilters(t *testing.T) {
	env := setupAgentModeTest(t)

	env.mockAgent.ListRunsResult = &integrations.ListRunsResult{
		Runs: []integrations.AgentRunSummary{},
	}

	w := env.doRequest("GET", "/api/v1/agent-runs?status=running&tag_prefix=inbox&limit=10&offset=5", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify filters were passed through
	if len(env.mockAgent.ListRunsCalls) != 1 {
		t.Fatalf("expected 1 ListRuns call, got %d", len(env.mockAgent.ListRunsCalls))
	}
	opts := env.mockAgent.ListRunsCalls[0].Options
	if opts.Status != "running" {
		t.Errorf("expected status 'running', got %q", opts.Status)
	}
	if opts.TagPrefix != "inbox" {
		t.Errorf("expected tag_prefix 'inbox', got %q", opts.TagPrefix)
	}
	if opts.Limit != 10 {
		t.Errorf("expected limit 10, got %d", opts.Limit)
	}
	if opts.Offset != 5 {
		t.Errorf("expected offset 5, got %d", opts.Offset)
	}
}
