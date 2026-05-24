package rewrite

import (
	"context"
	"testing"
)

func TestMemoryStoreRoundTrip(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	p := Plan{
		ID:           "abc",
		ScenarioPath: "/tmp/x",
		Operations:   []Operation{FileMove{From: "a.go", To: "b.go"}},
	}
	if err := s.Save(context.Background(), p); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load(context.Background(), "abc")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.ID != p.ID || got.ScenarioPath != p.ScenarioPath || len(got.Operations) != 1 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

func TestMemoryStoreLoadMissingIsPlanNotFound(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	_, err := s.Load(context.Background(), "nope")
	if err == nil {
		t.Fatal("expected error for missing plan")
	}
	rerr, ok := err.(RewriteError)
	if !ok || rerr.Kind != RewriteErrorPlanNotFound {
		t.Fatalf("want PlanNotFound, got %v", err)
	}
}
