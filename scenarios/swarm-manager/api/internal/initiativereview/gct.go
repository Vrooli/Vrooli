package initiativereview

import (
	"context"
	"encoding/json"
)

// GCTClient runs a fresh git-control-tower review for a single scenario.
// Narrow by design: initiative review only needs scenarioName → verdict.
// Kept as a local interface so this package doesn't import
// internal/execution (which owns the execution-record store, an unrelated
// concern); main.go adapts execution.HTTPReviewClient to this surface.
//
// Implementations must be safe to call concurrently — initiative review
// fans out one TriggerReview per affected scenario.
type GCTClient interface {
	TriggerReview(ctx context.Context, scenarioName string) (jobID string, err error)
	PollReview(ctx context.Context, jobID string) (result *GCTResult, done bool, err error)
}

// GCTResult is the per-scenario verdict collected from a fresh GCT run.
// Field shape mirrors what the review agent already parses from the
// gct-review-results attachment so skill prompts stay stable: the
// backlog-review flow and initiative-review flow serialize an identical
// vocabulary into the attachment.
//
// Error is populated when trigger or poll fails for this specific
// scenario — callers serialize it alongside the healthy verdicts so the
// agent can reason about partial signal rather than treating one flaky
// scenario as a review-wide failure.
type GCTResult struct {
	ScenarioName   string          `json:"scenario_name"`
	JobID          string          `json:"job_id,omitempty"`
	Classification string          `json:"classification,omitempty"`
	Summary        string          `json:"summary,omitempty"`
	RawDimensions  json.RawMessage `json:"raw_dimensions,omitempty"`
	ReviewedAt     string          `json:"reviewed_at,omitempty"`
	Error          string          `json:"error,omitempty"`
}
