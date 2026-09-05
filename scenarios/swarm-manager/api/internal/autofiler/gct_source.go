package autofiler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"swarm-manager/internal/execution"
)

const defaultReviewTimeout = 10 * time.Minute

type ReviewClient interface {
	TriggerReview(ctx context.Context, req execution.ReviewRequest) (string, error)
	PollReview(ctx context.Context, jobID string) (*execution.ReviewResult, bool, error)
}

type GCTFindingSource struct {
	Client        ReviewClient
	Freshness     *ReviewFreshnessStore
	FreshnessTime time.Duration
	PollInterval  time.Duration
	Timeout       time.Duration
}

func (s GCTFindingSource) Findings(ctx context.Context, target Target) ([]Finding, error) {
	scenario := strings.TrimSpace(target.Scenario)
	if scenario == "" {
		return nil, nil
	}
	if s.Client == nil {
		slog.Warn("autofiler: GCT finding source unavailable", "scenario", scenario, "err", "review client is nil")
		return []Finding{}, nil
	}
	now := time.Now().UTC()
	if s.Freshness != nil {
		fresh, err := s.Freshness.Fresh(scenario, s.FreshnessTime, now)
		if err != nil {
			slog.Warn("autofiler: review freshness check failed", "scenario", scenario, "err", err)
		} else if fresh {
			return []Finding{}, nil
		}
	}
	timeout := s.Timeout
	if timeout <= 0 {
		timeout = defaultReviewTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	jobID, err := s.Client.TriggerReview(runCtx, execution.ReviewRequest{
		ScenarioName:  scenario,
		ExpectedPaths: []string{"scenarios/" + slugPathSegment(scenario) + "/**"},
	})
	if err != nil {
		slog.Warn("autofiler: GCT review trigger failed", "scenario", scenario, "err", err)
		return []Finding{}, nil
	}

	interval := s.PollInterval
	if interval <= 0 {
		interval = 5 * time.Second
	}
	var result *execution.ReviewResult
	for {
		polled, done, err := s.Client.PollReview(runCtx, jobID)
		if err != nil {
			slog.Warn("autofiler: GCT review poll failed", "scenario", scenario, "err", err)
			return []Finding{}, nil
		}
		if done {
			result = polled
			break
		}
		select {
		case <-runCtx.Done():
			slog.Warn("autofiler: GCT review timed out", "scenario", scenario)
			return []Finding{}, nil
		case <-time.After(interval):
		}
	}
	if s.Freshness != nil {
		if err := s.Freshness.Mark(scenario, now); err != nil {
			slog.Warn("autofiler: mark review freshness failed", "scenario", scenario, "err", err)
		}
	}
	return FindingsFromGCTReview(scenario, result), nil
}

func FindingsFromGCTReview(scenario string, result *execution.ReviewResult) []Finding {
	if result == nil {
		return nil
	}
	scenario = strings.TrimSpace(scenario)
	findings := make([]Finding, 0, len(result.Dimensions))
	for _, dim := range result.Dimensions {
		severity, ok := severityFromReviewStatus(dim.Status)
		if !ok {
			continue
		}
		dimension := strings.TrimSpace(dim.Name)
		findingID := fmt.Sprintf("gct:%s:%s", slugPathSegment(scenario), slugify(dimension))
		findings = append(findings, Finding{
			ID:          findingID,
			Scenario:    scenario,
			Dimension:   dimension,
			Severity:    severity,
			Title:       fmt.Sprintf("[%s] readiness: %s is %s", scenario, dimension, severity),
			Description: "Git Control Tower readiness review reported a maintenance finding.",
			Details:     dim.Details,
		})
	}
	return findings
}

func severityFromReviewStatus(status string) (Severity, bool) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case string(SeverityRed):
		return SeverityRed, true
	case string(SeverityYellow):
		return SeverityYellow, true
	default:
		return "", false
	}
}
