package execution

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestFinalizationStatusTransitionGraph(t *testing.T) {
	statuses := []FinalizationStatus{
		FinalizationStatusPending,
		FinalizationStatusRunning,
		FinalizationStatusCompleted,
		FinalizationStatusSkipped,
		FinalizationStatusFailed,
	}
	for _, from := range statuses {
		for _, to := range statuses {
			t.Run(string(from)+"-to-"+string(to), func(t *testing.T) {
				finalization := &Finalization{Status: from}
				_, allowed := finalizationStatusTransitions[from][to]
				want := from == to || allowed
				err := finalization.TransitionStatus(to)
				if want {
					if err != nil || finalization.Status != to {
						t.Fatalf("transition %q -> %q: status=%q err=%v", from, to, finalization.Status, err)
					}
					return
				}
				if !errors.Is(err, ErrInvalidFinalizationStatusTransition) || finalization.Status != from {
					t.Fatalf("illegal transition %q -> %q changed status or lost typed error: status=%q err=%v", from, to, finalization.Status, err)
				}
			})
		}
	}
}

func TestFinalizationStatusRejectsUnknownAndTerminalResurrection(t *testing.T) {
	for _, test := range []struct {
		name string
		from FinalizationStatus
		to   FinalizationStatus
	}{
		{name: "unknown current", from: "corrupt", to: FinalizationStatusFailed},
		{name: "unknown next", from: FinalizationStatusPending, to: "corrupt"},
		{name: "completed resurrection", from: FinalizationStatusCompleted, to: FinalizationStatusRunning},
		{name: "skipped resurrection", from: FinalizationStatusSkipped, to: FinalizationStatusRunning},
		{name: "failed resurrection", from: FinalizationStatusFailed, to: FinalizationStatusRunning},
	} {
		t.Run(test.name, func(t *testing.T) {
			finalization := &Finalization{Status: test.from}
			if err := finalization.TransitionStatus(test.to); !errors.Is(err, ErrInvalidFinalizationStatusTransition) {
				t.Fatalf("TransitionStatus(%q) error = %v", test.to, err)
			}
			if finalization.Status != test.from {
				t.Fatalf("invalid transition mutated status to %q", finalization.Status)
			}
		})
	}
	var nilFinalization *Finalization
	if err := nilFinalization.TransitionStatus(FinalizationStatusRunning); !errors.Is(err, ErrInvalidFinalizationStatusTransition) {
		t.Fatalf("nil finalization transition error = %v", err)
	}
}

func TestFinalizationStatusSurvivesDurableRecordRecovery(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "execution.json"))
	finalization := &Finalization{Status: FinalizationStatusPending}
	if err := finalization.TransitionStatus(FinalizationStatusRunning); err != nil {
		t.Fatal(err)
	}
	if err := finalization.TransitionStatus(FinalizationStatusCompleted); err != nil {
		t.Fatal(err)
	}
	record := Record{ExecutionID: "finalization-recovery", Status: StatusValidating, Finalization: finalization}
	record.Finalization.Scenarios = []ScenarioFinalization{{
		ScenarioName: "recover-me",
		Restart:      RestartResult{Status: FinalizationStatusCompleted, Attempts: 1},
		Health:       HealthCheckResult{Status: FinalizationStatusCompleted, HealthStatus: "healthy"},
		Review:       ScenarioReviewStep{Status: FinalizationStatusSkipped, SkipReason: "review disabled"},
	}}
	if err := store.Save([]Record{record}); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 || loaded[0].Finalization == nil || loaded[0].Finalization.Status != FinalizationStatusCompleted {
		t.Fatalf("durable finalization state = %+v", loaded)
	}
	if got := loaded[0].Finalization.Scenarios[0]; got.Restart.Status != FinalizationStatusCompleted || got.Health.Status != FinalizationStatusCompleted || got.Review.Status != FinalizationStatusSkipped {
		t.Fatalf("durable scenario-step states = %+v", got)
	}
}

func TestPerScenarioFinalizationStepTransitionGraphs(t *testing.T) {
	tests := []struct {
		name    string
		new     func(FinalizationStatus) (func() FinalizationStatus, func(FinalizationStatus) error)
		allowed map[FinalizationStatus]map[FinalizationStatus]struct{}
	}{
		{
			name: "restart",
			new: func(status FinalizationStatus) (func() FinalizationStatus, func(FinalizationStatus) error) {
				step := &RestartResult{Status: status}
				return func() FinalizationStatus { return step.Status }, step.TransitionStatus
			},
			allowed: map[FinalizationStatus]map[FinalizationStatus]struct{}{
				FinalizationStatusPending: {FinalizationStatusRunning: {}, FinalizationStatusSkipped: {}, FinalizationStatusFailed: {}},
				FinalizationStatusRunning: {FinalizationStatusCompleted: {}, FinalizationStatusFailed: {}},
				FinalizationStatusFailed:  {FinalizationStatusRunning: {}},
			},
		},
		{
			name: "health",
			new: func(status FinalizationStatus) (func() FinalizationStatus, func(FinalizationStatus) error) {
				step := &HealthCheckResult{Status: status}
				return func() FinalizationStatus { return step.Status }, step.TransitionStatus
			},
			allowed: map[FinalizationStatus]map[FinalizationStatus]struct{}{
				FinalizationStatusPending: {FinalizationStatusRunning: {}, FinalizationStatusSkipped: {}, FinalizationStatusFailed: {}},
				FinalizationStatusRunning: {FinalizationStatusCompleted: {}, FinalizationStatusFailed: {}},
				FinalizationStatusFailed:  {FinalizationStatusRunning: {}},
			},
		},
		{
			name: "review",
			new: func(status FinalizationStatus) (func() FinalizationStatus, func(FinalizationStatus) error) {
				step := &ScenarioReviewStep{Status: status}
				return func() FinalizationStatus { return step.Status }, step.TransitionStatus
			},
			allowed: map[FinalizationStatus]map[FinalizationStatus]struct{}{
				FinalizationStatusPending: {FinalizationStatusRunning: {}, FinalizationStatusSkipped: {}, FinalizationStatusFailed: {}},
				FinalizationStatusRunning: {FinalizationStatusCompleted: {}, FinalizationStatusFailed: {}},
			},
		},
	}
	statuses := []FinalizationStatus{FinalizationStatusPending, FinalizationStatusRunning, FinalizationStatusCompleted, FinalizationStatusSkipped, FinalizationStatusFailed}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, from := range statuses {
				for _, to := range statuses {
					name := string(from) + "-to-" + string(to)
					t.Run(name, func(t *testing.T) {
						current, transition := test.new(from)
						_, allowed := test.allowed[from][to]
						want := from == to || allowed
						err := transition(to)
						if want && err == nil && current() == to {
							return
						}
						if !want && errors.Is(err, ErrInvalidFinalizationStatusTransition) && current() == from {
							return
						}
						t.Fatalf("transition %q -> %q: err=%v status-after=%q", from, to, err, current())
					})
				}
			}
			for _, invalid := range []struct {
				from, to FinalizationStatus
			}{{"unknown", FinalizationStatusFailed}, {FinalizationStatusPending, "unknown"}} {
				_, transition := test.new(invalid.from)
				if err := transition(invalid.to); !errors.Is(err, ErrInvalidFinalizationStatusTransition) {
					t.Fatalf("invalid transition %q -> %q error = %v", invalid.from, invalid.to, err)
				}
			}
		})
	}
	var nilRestart *RestartResult
	var nilHealth *HealthCheckResult
	var nilReview *ScenarioReviewStep
	for name, err := range map[string]error{
		"restart": nilRestart.TransitionStatus(FinalizationStatusRunning),
		"health":  nilHealth.TransitionStatus(FinalizationStatusRunning),
		"review":  nilReview.TransitionStatus(FinalizationStatusRunning),
	} {
		if !errors.Is(err, ErrInvalidFinalizationStatusTransition) {
			t.Fatalf("nil %s transition error = %v", name, err)
		}
	}
}

func TestPerScenarioFinalizationUpdatesRejectIllegalDurableEdges(t *testing.T) {
	path := filepath.Join(t.TempDir(), "execution.json")
	store := NewStore(path)
	record := Record{ExecutionID: "invalid-step-edge", Status: StatusValidating, Finalization: &Finalization{
		Status: FinalizationStatusRunning,
		Scenarios: []ScenarioFinalization{{
			ScenarioName: "example",
			Restart:      RestartResult{Status: FinalizationStatusPending},
			Health:       HealthCheckResult{Status: FinalizationStatusPending},
			Review:       ScenarioReviewStep{Status: FinalizationStatusPending},
		}},
	}}
	if err := store.Save([]Record{record}); err != nil {
		t.Fatal(err)
	}
	service := NewService(ServiceConfig{DataRoot: t.TempDir(), StorePath: path})
	updates := []struct {
		name string
		call func() error
	}{
		{name: "restart", call: func() error {
			return service.updateScenarioRestartState("invalid-step-edge", "example", RestartResult{Status: FinalizationStatusCompleted})
		}},
		{name: "health", call: func() error {
			return service.updateScenarioHealthState("invalid-step-edge", "example", HealthCheckResult{Status: FinalizationStatusCompleted})
		}},
		{name: "review", call: func() error {
			return service.updateScenarioReviewState("invalid-step-edge", "example", ScenarioReviewStep{Status: FinalizationStatusCompleted})
		}},
	}
	for _, update := range updates {
		t.Run(update.name, func(t *testing.T) {
			if err := update.call(); !errors.Is(err, ErrInvalidFinalizationStatusTransition) {
				t.Fatalf("illegal durable update error = %v", err)
			}
			loaded, err := store.Load()
			if err != nil {
				t.Fatal(err)
			}
			steps := loaded[0].Finalization.Scenarios[0]
			var got FinalizationStatus
			switch update.name {
			case "restart":
				got = steps.Restart.Status
			case "health":
				got = steps.Health.Status
			case "review":
				got = steps.Review.Status
			}
			if got != FinalizationStatusPending {
				t.Fatalf("invalid edge persisted status %q", got)
			}
		})
	}
}

func TestFailFinalizationClosesActiveScenarioSteps(t *testing.T) {
	path := filepath.Join(t.TempDir(), "execution.json")
	store := NewStore(path)
	record := Record{ExecutionID: "failed-step-recovery", Status: StatusValidating, Finalization: &Finalization{
		Status: FinalizationStatusRunning,
		Scenarios: []ScenarioFinalization{{
			ScenarioName: "example",
			Restart:      RestartResult{Status: FinalizationStatusRunning},
			Health:       HealthCheckResult{Status: FinalizationStatusRunning},
			Review:       ScenarioReviewStep{Status: FinalizationStatusRunning},
		}},
	}}
	if err := store.Save([]Record{record}); err != nil {
		t.Fatal(err)
	}
	service := NewService(ServiceConfig{DataRoot: t.TempDir(), StorePath: path})
	if err := service.failFinalization("failed-step-recovery", "example", "poll timed out"); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	got := loaded[0].Finalization.Scenarios[0]
	if loaded[0].Finalization.Status != FinalizationStatusFailed || got.Restart.Status != FinalizationStatusFailed || got.Health.Status != FinalizationStatusFailed || got.Review.Status != FinalizationStatusFailed {
		t.Fatalf("failed parent/child states = finalization %q, restart %q, health %q, review %q", loaded[0].Finalization.Status, got.Restart.Status, got.Health.Status, got.Review.Status)
	}
}
