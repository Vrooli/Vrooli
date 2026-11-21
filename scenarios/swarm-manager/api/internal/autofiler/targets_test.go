package autofiler

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"swarm-manager/internal/backlog"
)

func TestFeaturePendingStrategyCandidates(t *testing.T) {
	strategy := FeaturePendingStrategy{
		SelfScenarioName: "swarm-manager",
		BacklogReader: fakeBacklogReader{items: []backlog.BacklogItem{
			{
				Kind:            backlog.KindExecute,
				Name:            "queued-alpha",
				Status:          backlog.StatusQueued,
				Priority:        4,
				AcceptanceAllow: []string{"scenarios/alpha/**"},
			},
			{
				Kind:            backlog.KindExecute,
				Name:            "higher-priority-beta",
				Status:          backlog.StatusQueued,
				Priority:        1,
				AcceptanceAllow: []string{"scenarios/beta/**"},
			},
			{
				Kind:            backlog.KindFix,
				Name:            "beta-remediation",
				Status:          backlog.StatusBacklog,
				AcceptanceAllow: []string{"scenarios/beta/**"},
			},
			{
				Kind:            backlog.KindExecute,
				Name:            "self",
				Status:          backlog.StatusQueued,
				Priority:        1,
				AcceptanceAllow: []string{"scenarios/swarm-manager/**"},
			},
			{
				Kind:            backlog.KindExecute,
				Name:            "not-queued",
				Status:          backlog.StatusReady,
				Priority:        1,
				AcceptanceAllow: []string{"scenarios/gamma/**"},
			},
		}},
	}

	targets, err := strategy.Candidates(context.Background(), 10)
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}
	if got, want := targetScenarios(targets), []string{"alpha"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("scenarios = %#v, want %#v", got, want)
	}
}

func TestFeaturePendingStrategyOrdersByFeaturePriority(t *testing.T) {
	strategy := FeaturePendingStrategy{
		BacklogReader: fakeBacklogReader{items: []backlog.BacklogItem{
			{Kind: backlog.KindExecute, Status: backlog.StatusQueued, Priority: 5, AcceptanceAllow: []string{"scenarios/alpha/**"}},
			{Kind: backlog.KindExecute, Status: backlog.StatusQueued, Priority: 2, AcceptanceAllow: []string{"scenarios/beta/**"}},
			{Kind: backlog.KindExecute, Status: backlog.StatusQueued, Priority: 2, AcceptanceAllow: []string{"scenarios/charlie/**"}},
		}},
	}
	targets, err := strategy.Candidates(context.Background(), 2)
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}
	if got, want := targetScenarios(targets), []string{"beta", "charlie"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("scenarios = %#v, want %#v", got, want)
	}
}

type fakeScoreRunner struct {
	targets []Target
	err     error
}

func (r fakeScoreRunner) PriorityScenarios(_ context.Context, _ int) ([]Target, error) {
	return append([]Target(nil), r.targets...), r.err
}

func TestImportanceStrategyCapsRunnerResults(t *testing.T) {
	strategy := ImportanceStrategy{Runner: fakeScoreRunner{targets: []Target{
		{Scenario: "alpha", Priority: 9},
		{Scenario: "beta", Priority: 8},
	}}}
	targets, err := strategy.Candidates(context.Background(), 1)
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}
	if got, want := targetScenarios(targets), []string{"alpha"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("scenarios = %#v, want %#v", got, want)
	}
}

func TestImportanceStrategyDegradesToEmptyOnRunnerError(t *testing.T) {
	strategy := ImportanceStrategy{Runner: fakeScoreRunner{err: errors.New("scoring stopped")}}
	targets, err := strategy.Candidates(context.Background(), 10)
	if err != nil {
		t.Fatalf("Candidates error = %v, want nil", err)
	}
	if len(targets) != 0 {
		t.Fatalf("targets = %#v, want empty", targets)
	}
}

func targetScenarios(targets []Target) []string {
	out := make([]string, 0, len(targets))
	for _, target := range targets {
		out = append(out, target.Scenario)
	}
	return out
}
