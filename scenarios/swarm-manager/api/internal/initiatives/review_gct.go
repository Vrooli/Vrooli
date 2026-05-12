package initiativereview

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/initiatives"
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

// freshGCTConcurrency caps parallel TriggerReview calls into
// git-control-tower. GCT serializes per-scenario runs internally, so
// bumping this higher than the typical scenario count doesn't help;
// keeping it modest avoids thundering-herd on initiatives that span many
// scenarios.
const freshGCTConcurrency = 4

// collectAffectedScenarios walks the initiative's member items and
// returns the sorted union of affected scenarios across their latest
// finalizations. Returns nil when no execution lookup is wired or when
// no item has finalization data — the review still runs but without
// integration evidence (degraded mode).
func (s *Service) collectAffectedScenarios(init *initiatives.Initiative) []string {
	if s.executionLookup == nil {
		return nil
	}
	set := make(map[string]struct{})
	refs := append([]string(nil), init.Items...)
	sort.Strings(refs)
	for _, ref := range refs {
		parts := strings.SplitN(ref, "/", 2)
		if len(parts) != 2 {
			continue
		}
		kind, err := backlog.ParseBacklogKind(parts[0])
		if err != nil {
			continue
		}
		fin, err := s.executionLookup.LatestFinalizationFor(kind, parts[1])
		if err != nil || fin == nil {
			continue
		}
		for _, scenario := range fin.AffectedScenarios {
			scenario = strings.TrimSpace(scenario)
			if scenario == "" {
				continue
			}
			set[scenario] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for scenario := range set {
		out = append(out, scenario)
	}
	sort.Strings(out)
	return out
}

// runFreshGCT triggers and polls a GCT review for each scenario in
// parallel (bounded by freshGCTConcurrency) and returns verdicts keyed
// by scenario name. Returns nil when no client is wired or no scenarios
// are in scope — callers treat nil as "no fresh GCT evidence" and fall
// through. Per-scenario failures are captured on GCTResult.Error rather
// than aborting the fan-out, so a single unreachable scenario never
// blocks an initiative review from landing.
func (s *Service) runFreshGCT(ctx context.Context, scenarios []string) map[string]*GCTResult {
	if s.gctClient == nil || len(scenarios) == 0 {
		return nil
	}

	concurrency := freshGCTConcurrency
	if len(scenarios) < concurrency {
		concurrency = len(scenarios)
	}

	work := make(chan string, len(scenarios))
	for _, scenario := range scenarios {
		work <- scenario
	}
	close(work)

	var mu sync.Mutex
	results := make(map[string]*GCTResult, len(scenarios))
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for scenario := range work {
				res := s.runFreshGCTOne(ctx, scenario)
				mu.Lock()
				results[scenario] = res
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return results
}

// runFreshGCTOne performs one scenario's trigger-then-poll loop. Never
// returns an error: a failed trigger or poll is surfaced on
// GCTResult.Error so the caller can include the failure alongside
// healthy verdicts in the attachment payload.
func (s *Service) runFreshGCTOne(ctx context.Context, scenario string) *GCTResult {
	res := &GCTResult{ScenarioName: scenario}

	jobID, err := s.gctClient.TriggerReview(ctx, scenario)
	if err != nil {
		slog.Warn("initiative review: fresh GCT trigger failed",
			"scenario", scenario, "err", err)
		res.Error = fmt.Sprintf("trigger: %s", err.Error())
		return res
	}
	res.JobID = jobID

	interval := s.gctPollInterval
	if interval <= 0 {
		interval = 3 * time.Second
	}
	timeout := s.gctPollTimeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	deadline := s.clock().Add(timeout)

	for {
		result, done, pollErr := s.gctClient.PollReview(ctx, jobID)
		if pollErr != nil {
			slog.Warn("initiative review: fresh GCT poll failed",
				"scenario", scenario, "job_id", jobID, "err", pollErr)
			res.Error = fmt.Sprintf("poll: %s", pollErr.Error())
			return res
		}
		if done {
			if result != nil {
				res.Classification = result.Classification
				res.Summary = result.Summary
				res.RawDimensions = result.RawDimensions
				res.ReviewedAt = result.ReviewedAt
			}
			return res
		}
		if s.clock().After(deadline) {
			slog.Warn("initiative review: fresh GCT timed out",
				"scenario", scenario, "job_id", jobID, "timeout", timeout)
			res.Error = fmt.Sprintf("timed out after %s", timeout)
			return res
		}
		select {
		case <-ctx.Done():
			res.Error = ctx.Err().Error()
			return res
		case <-time.After(interval):
		}
	}
}
