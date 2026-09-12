package analytics_test

import (
	"context"
	"errors"
	"testing"

	"architecture-cartographer/internal/analytics"
	"architecture-cartographer/internal/analytics/mocks"
)

func TestService_RecordEvent_ValidatesScenario(t *testing.T) {
	svc := analytics.NewService(&mocks.FakeRepository{})
	_, err := svc.RecordEvent(context.Background(), analytics.Event{
		Kind: analytics.EventKindConflictDetected,
	})
	var typed analytics.ErrInvalidEvent
	if !errors.As(err, &typed) || typed.Field != "scenario" {
		t.Fatalf("want ErrInvalidEvent{scenario}, got %v", err)
	}
}

func TestService_RecordEvent_ValidatesKind(t *testing.T) {
	svc := analytics.NewService(&mocks.FakeRepository{})
	_, err := svc.RecordEvent(context.Background(), analytics.Event{
		Scenario: "demo",
		Kind:     analytics.EventKind("bogus"),
	})
	var typed analytics.ErrInvalidEvent
	if !errors.As(err, &typed) || typed.Field != "kind" {
		t.Fatalf("want ErrInvalidEvent{kind}, got %v", err)
	}
}

func TestService_RecordEvent_Persists(t *testing.T) {
	repo := &mocks.FakeRepository{}
	svc := analytics.NewService(repo)
	got, err := svc.RecordEvent(context.Background(), analytics.Event{
		Scenario: "demo",
		Kind:     analytics.EventKindConflictDetected,
	})
	if err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}
	if got.ID == "" {
		t.Fatal("expected generated ID")
	}
	if repo.AppendEventCalls.Load() != 1 {
		t.Fatalf("AppendEvent should be called once, got %d", repo.AppendEventCalls.Load())
	}
}

func TestService_RecordOverride_RejectsSelfDomain(t *testing.T) {
	svc := analytics.NewService(&mocks.FakeRepository{})
	_, err := svc.RecordOverride(context.Background(), analytics.Override{
		Scenario:      "demo",
		ChunkID:       "c1",
		VerdictDomain: "graph",
		ChosenDomain:  "graph",
	})
	var typed analytics.ErrInvalidOverride
	if !errors.As(err, &typed) || typed.Field != "chosen_domain" {
		t.Fatalf("want ErrInvalidOverride{chosen_domain}, got %v", err)
	}
}

func TestService_Stats_SuppressesLowN(t *testing.T) {
	repo := &mocks.FakeRepository{}
	svc := analytics.NewService(repo)
	for i := 0; i < 3; i++ {
		if _, err := svc.RecordEvent(context.Background(), analytics.Event{
			Scenario: "demo", Kind: analytics.EventKindVerdictProduced,
		}); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	stats, err := svc.Stats(context.Background(), "demo")
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if !stats.VerdictSuccessRateSuppressed {
		t.Fatalf("want suppressed, got %+v", stats)
	}
}

func TestService_Stats_ReportsRateAboveThreshold(t *testing.T) {
	repo := &mocks.FakeRepository{}
	svc := analytics.NewService(repo)
	for i := 0; i < analytics.MinVerdictObservations; i++ {
		if _, err := svc.RecordEvent(context.Background(), analytics.Event{
			Scenario: "demo", Kind: analytics.EventKindVerdictProduced,
		}); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	if _, err := svc.RecordOverride(context.Background(), analytics.Override{
		Scenario:      "demo",
		ChunkID:       "c1",
		VerdictDomain: "graph",
		ChosenDomain:  "manifest",
	}); err != nil {
		t.Fatalf("override: %v", err)
	}
	stats, err := svc.Stats(context.Background(), "demo")
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.VerdictSuccessRateSuppressed {
		t.Fatal("want unsuppressed")
	}
	want := 1.0 - 1.0/float64(analytics.MinVerdictObservations)
	if stats.VerdictSuccessRate < want-0.0001 || stats.VerdictSuccessRate > want+0.0001 {
		t.Fatalf("rate=%v want %v", stats.VerdictSuccessRate, want)
	}
}
