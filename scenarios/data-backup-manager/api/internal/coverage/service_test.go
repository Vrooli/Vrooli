package coverage_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"data-backup-manager/internal/coverage"
	"data-backup-manager/internal/sources"
)

// --- fakes -----------------------------------------------------------------

type fakeSuggestions struct {
	list []coverage.Suggestion
	err  error
}

func (f *fakeSuggestions) ListTargetSuggestions(context.Context) ([]coverage.Suggestion, error) {
	return f.list, f.err
}

type fakeTargets struct {
	list       []coverage.CatalogTarget
	registered []coverage.RegisterInput
	failOn     string // owner/name to fail registration for
	nextID     int
}

func (f *fakeTargets) List(context.Context) ([]coverage.CatalogTarget, error) {
	return f.list, nil
}

func (f *fakeTargets) Register(_ context.Context, in coverage.RegisterInput) (coverage.CatalogTarget, error) {
	if in.Owner+"/"+in.Name == f.failOn {
		return coverage.CatalogTarget{}, errors.New("boom")
	}
	f.registered = append(f.registered, in)
	f.nextID++
	t := coverage.CatalogTarget{
		ID:         "tgt-new",
		Owner:      in.Owner,
		Name:       in.Name,
		SourceKind: in.SourceKind,
		Locator:    in.Locator,
	}
	f.list = append(f.list, t)
	return t, nil
}

type fakePlans struct{ planned map[string]struct{} }

func (f *fakePlans) PlannedTargetIDs(context.Context) (map[string]struct{}, error) {
	return f.planned, nil
}

type fakeRuns struct{ success map[string]time.Time }

func (f *fakeRuns) LastSuccessByTarget(_ context.Context, _ []string) (map[string]time.Time, error) {
	return f.success, nil
}

type fakeRestores struct{ verified map[string]time.Time }

func (f *fakeRestores) LastVerifiedByTarget(_ context.Context, _ []string) (map[string]time.Time, error) {
	return f.verified, nil
}

func suggestion(owner, name string, sensitive bool) coverage.Suggestion {
	return coverage.Suggestion{
		ID:         owner + "-" + name,
		Owner:      owner,
		Name:       name,
		SourceKind: sources.KindFilesystem,
		Locator:    "/data/" + owner + "/" + name,
		Sensitive:  sensitive,
	}
}

// --- tests -----------------------------------------------------------------

func TestReport_ClassifiesAndScores(t *testing.T) {
	ctx := context.Background()
	now := time.Unix(1700000000, 0).UTC()

	svc := coverage.NewService(coverage.Deps{
		Suggestions: &fakeSuggestions{list: []coverage.Suggestion{
			suggestion("vrooli", "plans", false),
			suggestion("vrooli", "config", false),
			suggestion("codex", "auth", true),
		}},
		Targets: &fakeTargets{list: []coverage.CatalogTarget{
			{ID: "t1", Owner: "swarm-manager", Name: "domain-data", SourceKind: sources.KindFilesystem, Locator: "/d1"},
			{ID: "t2", Owner: "vrooli", Name: "secrets", SourceKind: sources.KindFilesystem, Locator: "/d2"},
		}},
		Plans:    &fakePlans{planned: map[string]struct{}{"t1": {}}},
		Runs:     &fakeRuns{success: map[string]time.Time{"t1": now}},
		Restores: &fakeRestores{verified: map[string]time.Time{"t1": now}},
	})

	rep, err := svc.Report(ctx)
	if err != nil {
		t.Fatalf("Report: %v", err)
	}

	if got := rep.Summary.RecommendedCount; got != 2 {
		t.Fatalf("recommended_count = %d, want 2", got)
	}
	if got := rep.Summary.SensitiveCount; got != 1 {
		t.Fatalf("sensitive_count = %d, want 1", got)
	}
	if len(rep.Recommended) != 2 || len(rep.Sensitive) != 1 {
		t.Fatalf("split wrong: recommended=%d sensitive=%d", len(rep.Recommended), len(rep.Sensitive))
	}
	if rep.Summary.DefaultCoverageComplete {
		t.Fatal("default_coverage_complete should be false with recommendations outstanding")
	}
	if !rep.Summary.HasSensitiveUnreviewed {
		t.Fatal("has_sensitive_unreviewed should be true")
	}
	if rep.Summary.RegisteredCount != 2 {
		t.Fatalf("registered_count = %d, want 2", rep.Summary.RegisteredCount)
	}
	if rep.Summary.PlannedCount != 1 {
		t.Fatalf("planned_count = %d, want 1", rep.Summary.PlannedCount)
	}
	if !rep.Summary.HasUnplannedRegisteredTargets {
		t.Fatal("t2 is unplanned — has_unplanned_registered_targets should be true")
	}
	if rep.Summary.BackedUpCount != 1 || rep.Summary.VerifiedCount != 1 {
		t.Fatalf("backed_up=%d verified=%d, want 1/1", rep.Summary.BackedUpCount, rep.Summary.VerifiedCount)
	}
	if !rep.Summary.HasUnverifiedTargets {
		t.Fatal("t2 never verified — has_unverified_targets should be true")
	}

	// Per-row annotation.
	var t1, t2 coverage.RegisteredTarget
	for _, r := range rep.Registered {
		switch r.ID {
		case "t1":
			t1 = r
		case "t2":
			t2 = r
		}
	}
	if !t1.Planned || t1.LastSuccessAt.IsZero() || t1.LastVerifiedAt.IsZero() {
		t.Fatalf("t1 should be planned/backed-up/verified: %+v", t1)
	}
	if t2.Planned || !t2.LastSuccessAt.IsZero() || !t2.LastVerifiedAt.IsZero() {
		t.Fatalf("t2 should be unplanned/never-backed-up/never-verified: %+v", t2)
	}
}

func TestReport_CompleteWhenNoRecommendations(t *testing.T) {
	svc := coverage.NewService(coverage.Deps{
		Suggestions: &fakeSuggestions{list: []coverage.Suggestion{suggestion("codex", "auth", true)}},
		Targets:     &fakeTargets{},
		Plans:       &fakePlans{},
		Runs:        &fakeRuns{},
		Restores:    &fakeRestores{},
	})
	rep, err := svc.Report(context.Background())
	if err != nil {
		t.Fatalf("Report: %v", err)
	}
	if !rep.Summary.DefaultCoverageComplete {
		t.Fatal("only a sensitive suggestion remains — default coverage is complete")
	}
	if !rep.Summary.HasSensitiveUnreviewed {
		t.Fatal("sensitive suggestion should still be flagged for review")
	}
}

func TestAcceptDefaults_RegistersNonSensitiveSkipsSensitive(t *testing.T) {
	targets := &fakeTargets{}
	svc := coverage.NewService(coverage.Deps{
		Suggestions: &fakeSuggestions{list: []coverage.Suggestion{
			suggestion("vrooli", "plans", false),
			suggestion("codex", "auth", true),
		}},
		Targets: targets,
	})

	res, err := svc.AcceptDefaults(context.Background(), coverage.AcceptOptions{})
	if err != nil {
		t.Fatalf("AcceptDefaults: %v", err)
	}
	if len(res.Accepted) != 1 || res.Accepted[0].Owner != "vrooli" {
		t.Fatalf("accepted wrong: %+v", res.Accepted)
	}
	if res.Accepted[0].TargetID == "" {
		t.Fatal("non-dry-run accept should carry a target id")
	}
	if len(res.SkippedSensitive) != 1 || res.SkippedSensitive[0].Owner != "codex" {
		t.Fatalf("sensitive not skipped: %+v", res.SkippedSensitive)
	}
	if len(targets.registered) != 1 {
		t.Fatalf("expected one registration, got %d", len(targets.registered))
	}
}

func TestAcceptDefaults_IncludeSensitive(t *testing.T) {
	targets := &fakeTargets{}
	svc := coverage.NewService(coverage.Deps{
		Suggestions: &fakeSuggestions{list: []coverage.Suggestion{suggestion("codex", "auth", true)}},
		Targets:     targets,
	})
	res, err := svc.AcceptDefaults(context.Background(), coverage.AcceptOptions{IncludeSensitive: true})
	if err != nil {
		t.Fatalf("AcceptDefaults: %v", err)
	}
	if len(res.Accepted) != 1 || !res.Accepted[0].Sensitive {
		t.Fatalf("sensitive should be accepted with opt-in: %+v", res.Accepted)
	}
	if len(res.SkippedSensitive) != 0 {
		t.Fatalf("nothing should be skipped: %+v", res.SkippedSensitive)
	}
}

func TestAcceptDefaults_DryRunMutatesNothing(t *testing.T) {
	targets := &fakeTargets{}
	svc := coverage.NewService(coverage.Deps{
		Suggestions: &fakeSuggestions{list: []coverage.Suggestion{suggestion("vrooli", "plans", false)}},
		Targets:     targets,
	})
	res, err := svc.AcceptDefaults(context.Background(), coverage.AcceptOptions{DryRun: true})
	if err != nil {
		t.Fatalf("AcceptDefaults: %v", err)
	}
	if !res.DryRun {
		t.Fatal("response should echo dry_run")
	}
	if len(res.Accepted) != 1 || res.Accepted[0].TargetID != "" {
		t.Fatalf("dry-run accept should plan without a target id: %+v", res.Accepted)
	}
	if len(targets.registered) != 0 {
		t.Fatalf("dry-run must not register anything, got %d", len(targets.registered))
	}
}

func TestAcceptDefaults_PartialFailureReported(t *testing.T) {
	targets := &fakeTargets{failOn: "vrooli/config"}
	svc := coverage.NewService(coverage.Deps{
		Suggestions: &fakeSuggestions{list: []coverage.Suggestion{
			suggestion("vrooli", "plans", false),
			suggestion("vrooli", "config", false),
		}},
		Targets: targets,
	})
	res, err := svc.AcceptDefaults(context.Background(), coverage.AcceptOptions{})
	if err != nil {
		t.Fatalf("AcceptDefaults should not fail wholesale: %v", err)
	}
	if len(res.Accepted) != 1 {
		t.Fatalf("one should succeed: %+v", res.Accepted)
	}
	if len(res.Failed) != 1 || res.Failed[0].Name != "config" {
		t.Fatalf("failure not reported per item: %+v", res.Failed)
	}
}

func TestUnregisteredDefaultTargets_NonSensitiveOnly(t *testing.T) {
	svc := coverage.NewService(coverage.Deps{
		Suggestions: &fakeSuggestions{list: []coverage.Suggestion{
			suggestion("vrooli", "plans", false),
			suggestion("codex", "auth", true),
		}},
	})
	recs, err := svc.UnregisteredDefaultTargets(context.Background())
	if err != nil {
		t.Fatalf("UnregisteredDefaultTargets: %v", err)
	}
	if len(recs) != 1 || recs[0].Sensitive {
		t.Fatalf("only non-sensitive recommendations expected: %+v", recs)
	}
}
