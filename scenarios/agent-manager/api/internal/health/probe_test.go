package health_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"agent-manager/internal/health"
)

type probeRegistry map[string][]string

func (r probeRegistry) IterModels(yield func(string, []string) bool) {
	for runnerType, models := range r {
		if !yield(runnerType, models) {
			return
		}
	}
}

type probeRunner struct {
	errors map[string]error
	calls  []string
}

func (r *probeRunner) ProbeModel(_ context.Context, modelID string) error {
	r.calls = append(r.calls, modelID)
	return r.errors[modelID]
}

func TestProbeRunOnceRecordsOutcomesAndSkipsUnavailableInputs(t *testing.T) {
	store := newTestStore(t)
	claude := &probeRunner{errors: map[string]error{"broken": errors.New("rate limit")}}
	probe := health.NewProbe(
		store,
		probeRegistry{
			"claude-code": {"healthy", " broken ", ""},
			"missing":     {"not-probed"},
		},
		func(runnerType string) health.ModelProber {
			if runnerType == "claude-code" {
				return claude
			}
			return nil
		},
		nil,
		health.ProbeConfig{},
	)

	probe.RunOnce(context.Background())
	if len(claude.calls) != 2 || claude.calls[0] != "healthy" || claude.calls[1] != "broken" {
		t.Fatalf("probe calls = %v, want trimmed non-empty models only", claude.calls)
	}
	snapshot, err := store.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if got := snapshot.Models["claude-code"]["healthy"]; got.Status != health.StatusOK {
		t.Fatalf("healthy observation = %+v", got)
	}
	if got := snapshot.Models["claude-code"]["broken"]; got.Status != health.StatusFailed || got.Message == "" {
		t.Fatalf("failed observation = %+v", got)
	}
	if _, ok := snapshot.Models["missing"]; ok {
		t.Fatalf("unregistered runner recorded a health result: %+v", snapshot.Models["missing"])
	}
}

func TestProbeRunOnceIsSafeWithMissingDependencies(t *testing.T) {
	health.NewProbe(nil, nil, nil, nil, health.ProbeConfig{}).RunOnce(context.Background())
}

func TestProbeDefaultsAndLifecycleAreSafeWithoutOptionalWorkers(t *testing.T) {
	config := health.DefaultProbeConfig()
	if config.Interval != 30*time.Minute || config.Retention != health.DefaultRetention || config.EvictionInterval != 24*time.Hour {
		t.Fatalf("default probe config=%+v", config)
	}
	ctx, cancel := context.WithCancel(context.Background())
	probe := health.NewProbe(nil, nil, nil, nil, health.ProbeConfig{})
	probe.Start(ctx)
	probe.Start(ctx)
	cancel()
	probe.Stop()
	probe.Stop()
	(*health.Probe)(nil).Start(context.Background())
	(*health.Probe)(nil).Stop()
}
