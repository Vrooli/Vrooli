package conflicts_test

import (
	"context"
	"sync"
	"testing"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/conflicts/mocks"
)

// fakeRecorder captures every Record call so tests can assert
// analytics emission without reaching the real analytics package.
type fakeRecorder struct {
	mu     sync.Mutex
	events []struct {
		Scenario, Kind, ConflictID string
		Payload                    map[string]any
	}
}

func (f *fakeRecorder) Record(_ context.Context, scenario, kind, conflictID string, payload map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, struct {
		Scenario, Kind, ConflictID string
		Payload                    map[string]any
	}{scenario, kind, conflictID, payload})
}

func (f *fakeRecorder) kinds() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.events))
	for i, e := range f.events {
		out[i] = e.Kind
	}
	return out
}

func TestService_DetectConflicts_EmitsAnalyticsPerConflict(t *testing.T) {
	repo := &mocks.FakeRepository{}
	det := &mocks.FakeDetector{NameValue: "cycle", Conflicts: []conflicts.Conflict{
		{Type: "cycle", Severity: conflicts.SeverityError},
		{Type: "cycle", Severity: conflicts.SeverityError},
	}}
	rec := &fakeRecorder{}
	svc := conflicts.NewServiceWithAnalytics(repo, conflicts.NewRegistry(det), conflicts.NewResolverRegistry(), rec)

	got, err := svc.DetectConflicts(context.Background(), conflicts.DetectOrchestrationInput{Scenario: "demo"})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 conflicts persisted, got %d", len(got))
	}
	kinds := rec.kinds()
	if len(kinds) != 2 || kinds[0] != "conflict_detected" {
		t.Fatalf("want 2 conflict_detected events, got %v", kinds)
	}
}

func TestService_DetectConflicts_RejectsMissingScenario(t *testing.T) {
	_, svc := newSvc()
	_, err := svc.DetectConflicts(context.Background(), conflicts.DetectOrchestrationInput{})
	if err == nil {
		t.Fatal("want error for missing scenario")
	}
}

func TestService_AssignAndResolve_EmitAnalytics(t *testing.T) {
	repo := &mocks.FakeRepository{}
	rec := &fakeRecorder{}
	svc := conflicts.NewServiceWithAnalytics(repo, conflicts.NewRegistry(), conflicts.NewResolverRegistry(), rec)

	seeded, err := svc.UpsertConflicts(context.Background(), "demo", []conflicts.Conflict{{Type: "cycle"}})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	id := seeded[0].ID

	if _, _, err := svc.AssignConflict(context.Background(), id, "graph", "ok", false); err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if _, _, _, err := svc.ResolveConflict(context.Background(), id, "fixed", false, false); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if _, _, err := svc.ReopenConflict(context.Background(), id, "needs rework", false); err != nil {
		t.Fatalf("Reopen: %v", err)
	}

	kinds := rec.kinds()
	want := []string{"conflict_assigned", "conflict_resolved", "conflict_reopened"}
	for i, w := range want {
		if i >= len(kinds) || kinds[i] != w {
			t.Fatalf("kinds[%d]=%q want %q (all=%v)", i, kinds[i], w, kinds)
		}
	}
}

func TestService_ValidateConflicts_ReturnsOutstandingOnly(t *testing.T) {
	repo := &mocks.FakeRepository{}
	svc := conflicts.NewService(repo, conflicts.NewRegistry(), conflicts.NewResolverRegistry())

	// Seed three conflicts: one resolved, one force-resolved, one open.
	for _, c := range []conflicts.Conflict{
		{ID: "a", Scenario: "demo", Severity: conflicts.SeverityError, Status: conflicts.ResolutionStatusResolved},
		{ID: "b", Scenario: "demo", Severity: conflicts.SeverityError, Status: conflicts.ResolutionStatusForceResolved},
		{ID: "c", Scenario: "demo", Severity: conflicts.SeverityError, Status: conflicts.ResolutionStatusDetected},
	} {
		if _, err := repo.UpsertConflict(context.Background(), c); err != nil {
			t.Fatalf("seed %s: %v", c.ID, err)
		}
	}

	out, clean, err := svc.ValidateConflicts(context.Background(), "demo")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(out) != 1 || out[0].ID != "c" {
		t.Fatalf("expected one outstanding (id=c), got %+v", out)
	}
	if clean {
		t.Fatalf("expected clean=false when an error-severity conflict outstanding")
	}
}

func TestService_ValidateConflicts_CleanWhenNoneOutstanding(t *testing.T) {
	repo := &mocks.FakeRepository{}
	svc := conflicts.NewService(repo, conflicts.NewRegistry(), conflicts.NewResolverRegistry())

	out, clean, err := svc.ValidateConflicts(context.Background(), "demo")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(out) != 0 || !clean {
		t.Fatalf("expected clean+empty, got out=%v clean=%v", out, clean)
	}
}
