// This file wakes a parked parent run when one of its child runs finishes.
package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// ProducerChildren parks a parent run until any of its direct child runs
// reaches a terminal state. The await key is the parked parent's own run ID.
// The park deadline doubles as the timer wake, and `run wake --key` is the
// explicit wake; together they are the orchestrator's three wake sources.
const ProducerChildren = "children"

// defaultChildrenPollInterval bounds how quickly a child's terminal state
// reaches the parked parent. The wait itself costs no tokens; only the
// resumed turn does.
const defaultChildrenPollInterval = 10 * time.Second

// childRunReader is the narrow run surface the children waiter reads.
type childRunReader interface {
	GetRun(ctx context.Context, id uuid.UUID) (*domain.Run, error)
	ListRuns(ctx context.Context, opts RunListOptions) ([]*domain.Run, error)
}

type childrenWaiter struct {
	runs     childRunReader
	interval time.Duration
	now      func() time.Time
}

// NewChildrenWaiter builds the waiter for ProducerChildren. A zero interval
// uses the default poll interval.
func NewChildrenWaiter(runs childRunReader, interval time.Duration) Waiter {
	if interval <= 0 {
		interval = defaultChildrenPollInterval
	}
	return &childrenWaiter{runs: runs, interval: interval, now: time.Now}
}

func (w *childrenWaiter) Producer() string { return ProducerChildren }

// childRunSummary is one child row in the wake payload.
type childRunSummary struct {
	RunID   string `json:"run_id"`
	Status  string `json:"status"`
	EndedAt string `json:"ended_at,omitempty"`
}

// Wait returns once a child run has ended that the parent has not yet been
// told about: one that ended after the parent's previous wake, or after the
// parent started when it has never been woken. A child that finishes while the
// parent is still deciding to park therefore wakes it immediately, and an
// agent-manager restart neither misses nor repeats a child. On deadline it
// returns the current child status with the context error, so the timer wake
// still carries the evidence.
func (w *childrenWaiter) Wait(ctx context.Context, key string) (string, error) {
	parentID, err := uuid.Parse(strings.TrimSpace(key))
	if err != nil {
		return "", domain.NewValidationError("key", fmt.Sprintf("children await key %q must be the parked parent run ID", key))
	}
	since := w.now()
	if parent, err := w.runs.GetRun(ctx, parentID); err == nil && parent != nil {
		since = unreportedSince(parent, since)
	}
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		children, err := w.runs.ListRuns(ctx, RunListOptions{ParentRunID: &parentID})
		if err != nil && ctx.Err() == nil {
			return "", fmt.Errorf("list child runs of %s: %w", parentID, err)
		}
		if ended := endedSince(children, since); len(ended) > 0 {
			return childrenPayload(parentID, "child_run_ended", ended, children)
		}
		select {
		case <-ctx.Done():
			payload, _ := childrenPayload(parentID, "timer", nil, children)
			return payload, ctx.Err()
		case <-ticker.C:
		}
	}
}

// unreportedSince is the instant after which a child's end has not been
// delivered to the parent: its previous wake, else its start, else creation.
func unreportedSince(parent *domain.Run, fallback time.Time) time.Time {
	switch {
	case parent.LastAwaitResolvedAt != nil:
		return *parent.LastAwaitResolvedAt
	case parent.StartedAt != nil:
		return *parent.StartedAt
	case !parent.CreatedAt.IsZero():
		return parent.CreatedAt
	default:
		return fallback
	}
}

func endedSince(children []*domain.Run, since time.Time) []*domain.Run {
	var ended []*domain.Run
	for _, child := range children {
		if child != nil && child.Status.IsTerminal() && child.EndedAt != nil && !child.EndedAt.Before(since) {
			ended = append(ended, child)
		}
	}
	return ended
}

func childrenPayload(parentID uuid.UUID, reason string, ended, all []*domain.Run) (string, error) {
	summarize := func(runs []*domain.Run) []childRunSummary {
		rows := make([]childRunSummary, 0, len(runs))
		for _, run := range runs {
			if run == nil {
				continue
			}
			row := childRunSummary{RunID: run.ID.String(), Status: string(run.Status)}
			if run.EndedAt != nil {
				row.EndedAt = run.EndedAt.UTC().Format(time.RFC3339)
			}
			rows = append(rows, row)
		}
		return rows
	}
	active := 0
	for _, run := range all {
		if run != nil && !run.Status.IsTerminal() {
			active++
		}
	}
	data, err := json.Marshal(map[string]any{
		"kind": "child_runs", "parent_run_id": parentID.String(), "wake_reason": reason,
		"ended": summarize(ended), "active_children": active, "children": summarize(all),
	})
	return string(data), err
}
