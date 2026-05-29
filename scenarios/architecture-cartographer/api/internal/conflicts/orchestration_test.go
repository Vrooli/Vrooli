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
	// Locations differ so DetectConflicts' stable_id dedupe pass treats
	// them as two distinct drifts (identical drifts collapse to one row).
	det := &mocks.FakeDetector{NameValue: "cycle", Conflicts: []conflicts.Conflict{
		{Type: "cycle", Severity: conflicts.SeverityError, Locations: []string{"a"}},
		{Type: "cycle", Severity: conflicts.SeverityError, Locations: []string{"b"}},
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

func TestService_ValidateConflicts_ListsOutstandingExcludingSuppressed(t *testing.T) {
	repo := &mocks.FakeRepository{}
	svc := conflicts.NewService(repo, conflicts.NewRegistry(), conflicts.NewResolverRegistry())

	// Detection-only: every non-suppressed conflict is outstanding. A
	// suppressed finding is intentional and never counts.
	for _, c := range []conflicts.Conflict{
		{ID: "a", Scenario: "demo", Severity: conflicts.SeverityError, Suppressed: true, SuppressionReason: "intentional"},
		{ID: "b", Scenario: "demo", Severity: conflicts.SeverityError},
	} {
		if _, err := repo.UpsertConflict(context.Background(), c); err != nil {
			t.Fatalf("seed %s: %v", c.ID, err)
		}
	}

	out, clean, err := svc.ValidateConflicts(context.Background(), "demo")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(out) != 1 || out[0].ID != "b" {
		t.Fatalf("expected one outstanding (id=b, suppressed excluded), got %+v", out)
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
