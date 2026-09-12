package conflicts_test

import (
	"context"
	"testing"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/conflicts/mocks"
)

func newSvc(detectors ...conflicts.Detector) (*mocks.FakeRepository, conflicts.Service) {
	repo := &mocks.FakeRepository{}
	reg := conflicts.NewRegistry(detectors...)
	resolvers := conflicts.NewResolverRegistry()
	return repo, conflicts.NewService(repo, reg, resolvers)
}

func TestService_UpsertConflicts_AssignsScenario(t *testing.T) {
	_, svc := newSvc()
	got, err := svc.UpsertConflicts(context.Background(), "demo", []conflicts.Conflict{{Type: "cycle"}})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if got[0].Scenario != "demo" {
		t.Fatalf("want scenario demo, got %q", got[0].Scenario)
	}
}

func TestService_ValidateConflicts_CleanWhenNoErrorSeverity(t *testing.T) {
	_, svc := newSvc()
	_, err := svc.UpsertConflicts(context.Background(), "demo", []conflicts.Conflict{
		{Type: "coupling_smell", Severity: conflicts.SeverityWarn, FindingClass: conflicts.FindingClassHeuristic},
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	outstanding, clean, err := svc.ValidateConflicts(context.Background(), "demo")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !clean {
		t.Fatalf("want clean (no error-severity), got dirty with %d outstanding", len(outstanding))
	}
	if len(outstanding) != 1 {
		t.Fatalf("warn-severity conflict is still outstanding (detection-only), got %d", len(outstanding))
	}
}

func TestService_ValidateConflicts_DirtyWhenErrorSeverity(t *testing.T) {
	_, svc := newSvc()
	if _, err := svc.UpsertConflicts(context.Background(), "demo", []conflicts.Conflict{
		{Type: "cycle", Severity: conflicts.SeverityError, FindingClass: conflicts.FindingClassDeterministic},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	_, clean, err := svc.ValidateConflicts(context.Background(), "demo")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if clean {
		t.Fatal("want dirty (an error-severity conflict is outstanding)")
	}
}

func TestService_ValidateConflicts_DirtyWhenBlockerSeverity(t *testing.T) {
	_, svc := newSvc()
	if _, err := svc.UpsertConflicts(context.Background(), "demo", []conflicts.Conflict{
		{Type: "layering", Severity: conflicts.SeverityBlocker, FindingClass: conflicts.FindingClassDeterministic},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	_, clean, err := svc.ValidateConflicts(context.Background(), "demo")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if clean {
		t.Fatal("want dirty (a blocker-severity conflict is outstanding)")
	}
}

func TestService_ValidateConflicts_ExcludesSuppressed(t *testing.T) {
	_, svc := newSvc()
	if _, err := svc.UpsertConflicts(context.Background(), "demo", []conflicts.Conflict{
		{Type: "cycle", Severity: conflicts.SeverityError, FindingClass: conflicts.FindingClassDeterministic, Suppressed: true, SuppressionReason: "intentional"},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	outstanding, clean, err := svc.ValidateConflicts(context.Background(), "demo")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !clean || len(outstanding) != 0 {
		t.Fatalf("a suppressed finding must not count as outstanding: clean=%v outstanding=%d", clean, len(outstanding))
	}
}

func TestRegistry_DetectAll_AlphabeticalOrder(t *testing.T) {
	zd := &mocks.FakeDetector{NameValue: "zeta", Conflicts: []conflicts.Conflict{{Type: "z"}}}
	ad := &mocks.FakeDetector{NameValue: "alpha", Conflicts: []conflicts.Conflict{{Type: "a"}}}
	reg := conflicts.NewRegistry(zd, ad)
	out, err := reg.DetectAll(context.Background(), conflicts.DetectInput{})
	if err != nil {
		t.Fatalf("DetectAll: %v", err)
	}
	if out[0].Type != "a" || out[1].Type != "z" {
		t.Fatalf("registry order broken: %+v", out)
	}
}

func TestRegistry_DetectAll_StampsClassAndCapsHeuristicSeverity(t *testing.T) {
	reg := conflicts.NewRegistry(&mocks.FakeDetector{
		NameValue: "heuristic",
		Conflicts: []conflicts.Conflict{{
			Type:         "mislocated_file",
			Severity:     conflicts.SeverityBlocker,
			FindingClass: conflicts.FindingClassHeuristic,
		}},
	})
	out, err := reg.DetectAll(context.Background(), conflicts.DetectInput{})
	if err != nil {
		t.Fatalf("DetectAll: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("conflicts = %d, want 1", len(out))
	}
	if out[0].FindingClass != conflicts.FindingClassHeuristic {
		t.Fatalf("class = %q, want heuristic", out[0].FindingClass)
	}
	if out[0].Severity != conflicts.SeverityWarn {
		t.Fatalf("heuristic severity = %q, want warn", out[0].Severity)
	}
}
