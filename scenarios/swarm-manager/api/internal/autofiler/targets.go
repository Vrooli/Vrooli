package autofiler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"sort"
	"strings"
	"time"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/backlogstatus"
	"swarm-manager/internal/pathutil"
)

const scoreListBudget = 15 * time.Second

type Target struct {
	Scenario string
	Reason   string
	Priority float64
}

type TargetingStrategy interface {
	Candidates(ctx context.Context, limit int) ([]Target, error)
}

type FeaturePendingStrategy struct {
	BacklogReader    BacklogReader
	SelfScenarioName string
}

func (s FeaturePendingStrategy) Candidates(_ context.Context, limit int) ([]Target, error) {
	if s.BacklogReader == nil {
		return nil, fmt.Errorf("feature-pending strategy requires a backlog reader")
	}
	items, err := s.BacklogReader.LoadAll([]backlog.BacklogKind{backlog.KindExecute, backlog.KindFix, backlog.KindChore})
	if err != nil {
		return nil, err
	}
	openRemediation := scenariosWithOpenRemediation(items)
	best := map[string]Target{}
	for _, item := range items {
		if item.Kind != backlog.KindExecute || item.Status != backlog.StatusQueued || backlog.IsArchived(item) {
			continue
		}
		scenarios := pathutil.UniqueSortedStrings(pathutil.ScenariosFromGlobs(item.AcceptanceAllow))
		for _, scenario := range scenarios {
			if scenario == "" || scenario == s.SelfScenarioName {
				continue
			}
			if _, ok := openRemediation[scenario]; ok {
				continue
			}
			priority := float64(item.Priority)
			candidate := Target{
				Scenario: scenario,
				Reason:   "queued feature work has no open remediation item",
				Priority: priority,
			}
			if existing, ok := best[scenario]; !ok || candidate.Priority < existing.Priority {
				best[scenario] = candidate
			}
		}
	}
	out := make([]Target, 0, len(best))
	for _, candidate := range best {
		out = append(out, candidate)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Priority != out[j].Priority {
			return out[i].Priority < out[j].Priority
		}
		return out[i].Scenario < out[j].Scenario
	})
	return capTargets(out, limit), nil
}

func scenariosWithOpenRemediation(items []backlog.BacklogItem) map[string]struct{} {
	out := map[string]struct{}{}
	for _, item := range items {
		if item.Kind != backlog.KindFix && item.Kind != backlog.KindChore {
			continue
		}
		if !isOpenRemediationStatus(item.Status) || backlog.IsArchived(item) {
			continue
		}
		for _, scenario := range pathutil.ScenariosFromGlobs(item.AcceptanceAllow) {
			if scenario != "" {
				out[scenario] = struct{}{}
			}
		}
	}
	return out
}

func isOpenRemediationStatus(status backlog.BacklogStatus) bool {
	switch strings.ToLower(strings.TrimSpace(string(status))) {
	case backlogstatus.Completed, backlogstatus.Failed:
		return false
	default:
		return true
	}
}

type ScoreListRunner interface {
	PriorityScenarios(ctx context.Context, limit int) ([]Target, error)
}

type ImportanceStrategy struct {
	Runner ScoreListRunner
}

func (s ImportanceStrategy) Candidates(ctx context.Context, limit int) ([]Target, error) {
	runner := s.Runner
	if runner == nil {
		runner = CLIScoreListRunner{}
	}
	targets, err := runner.PriorityScenarios(ctx, limit)
	if err != nil {
		slog.Warn("autofiler: importance strategy unavailable", "err", err)
		return []Target{}, nil
	}
	return capTargets(targets, limit), nil
}

type CLIScoreListRunner struct{}

func (CLIScoreListRunner) PriorityScenarios(ctx context.Context, limit int) ([]Target, error) {
	path, err := exec.LookPath("scenario-completeness-scoring")
	if err != nil {
		return nil, fmt.Errorf("scenario-completeness-scoring CLI not found: %w", err)
	}
	if limit <= 0 {
		limit = 500
	}
	cctx, cancel := context.WithTimeout(ctx, scoreListBudget)
	defer cancel()
	out, err := exec.CommandContext(cctx, path, "score", "list",
		"--json", "--sort", "priority", "--order", "desc", "--limit", fmt.Sprintf("%d", limit)).Output()
	if err != nil {
		return nil, fmt.Errorf("score list failed: %w", err)
	}
	var payload struct {
		Scores []struct {
			Scenario string  `json:"scenario"`
			Priority float64 `json:"priority"`
		} `json:"scores"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return nil, fmt.Errorf("parse score list: %w", err)
	}
	targets := make([]Target, 0, len(payload.Scores))
	for _, score := range payload.Scores {
		scenario := strings.TrimSpace(score.Scenario)
		if scenario == "" {
			continue
		}
		targets = append(targets, Target{
			Scenario: scenario,
			Reason:   "scenario-completeness priority ranking",
			Priority: score.Priority,
		})
	}
	sort.SliceStable(targets, func(i, j int) bool {
		if targets[i].Priority != targets[j].Priority {
			return targets[i].Priority > targets[j].Priority
		}
		return targets[i].Scenario < targets[j].Scenario
	})
	return capTargets(targets, limit), nil
}

func capTargets(targets []Target, limit int) []Target {
	if limit > 0 && len(targets) > limit {
		targets = targets[:limit]
	}
	if targets == nil {
		return []Target{}
	}
	return targets
}
