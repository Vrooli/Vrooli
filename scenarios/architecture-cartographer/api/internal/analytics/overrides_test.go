package analytics_test

import (
	"context"
	"testing"

	"architecture-cartographer/internal/analytics"
	"architecture-cartographer/internal/analytics/mocks"
)

// TestOverrideRecording_RoundTrip — recording an override produces an
// Override row plus the recorder picks up Scenario/ChunkID validation.
func TestOverrideRecording_RoundTrip(t *testing.T) {
	repo := &mocks.FakeRepository{}
	svc := analytics.NewService(repo)

	ov, err := svc.RecordOverride(context.Background(), analytics.Override{
		Scenario:      "demo",
		ChunkID:       "chunk:f1",
		VerdictDomain: "graph",
		ChosenDomain:  "signals",
		Note:          "pathtoken weight too strong",
	})
	if err != nil {
		t.Fatalf("RecordOverride: %v", err)
	}
	if ov.Scenario != "demo" {
		t.Fatalf("Scenario lost: %+v", ov)
	}
	if repo.AppendOverrideCalls.Load() != 1 {
		t.Fatalf("AppendOverride should be called once, got %d", repo.AppendOverrideCalls.Load())
	}
}

func TestOverrideRecording_RejectsMissingChunkID(t *testing.T) {
	repo := &mocks.FakeRepository{}
	svc := analytics.NewService(repo)

	_, err := svc.RecordOverride(context.Background(), analytics.Override{
		Scenario:      "demo",
		VerdictDomain: "graph",
		ChosenDomain:  "signals",
	})
	if err == nil {
		t.Fatal("want error for missing chunk_id")
	}
	if repo.AppendOverrideCalls.Load() != 0 {
		t.Fatalf("repo must not be touched on validation error")
	}
}

func TestOverrideRecording_RejectsSameDomain(t *testing.T) {
	repo := &mocks.FakeRepository{}
	svc := analytics.NewService(repo)

	_, err := svc.RecordOverride(context.Background(), analytics.Override{
		Scenario:      "demo",
		ChunkID:       "chunk:f1",
		VerdictDomain: "graph",
		ChosenDomain:  "graph",
	})
	if err == nil {
		t.Fatal("want error when chosen == verdict")
	}
}
