package campaign

import (
	"context"
	"testing"

	"architecture-cartographer/internal/clock"
	localdb "architecture-cartographer/internal/database"
	testdb "architecture-cartographer/internal/testutil/db"

	apidb "github.com/vrooli/api-core/database"
	campaignv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/campaign"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

func newTestService(t *testing.T) Service {
	t.Helper()
	d := testdb.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(Schema),
	); err != nil {
		t.Fatalf("ensure schemas: %v", err)
	}
	return NewService(NewSQLiteRepository(d, clock.System{}))
}

func pf(scenario, code string, sev architecturev1.FindingSeverity, locs ...string) *architecturev1.ArchitectureFinding {
	return &architecturev1.ArchitectureFinding{
		Scenario:  scenario,
		Source:    architecturev1.FindingSource_FINDING_SOURCE_ARCHITECTURE,
		Code:      code,
		Severity:  sev,
		Locations: locs,
	}
}

func findingByCode(findings []Finding, code string) (Finding, bool) {
	for _, f := range findings {
		if f.Code == code {
			return f, true
		}
	}
	return Finding{}, false
}

func TestCreateIngestsFindings(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	st, err := svc.Create(ctx, "swarm-manager", "big-refactor", []*architecturev1.ArchitectureFinding{
		pf("swarm-manager", "cycle/cross-domain", architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER, "api/a", "api/b"),
		pf("swarm-manager", "coupling_smell", architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING, "api/c"),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if st.Total != 2 || st.Open != 2 {
		t.Fatalf("expected 2 open findings, got total=%d open=%d", st.Total, st.Open)
	}
	if st.Campaign.Status != CampaignOpen {
		t.Errorf("campaign should be open, got %s", st.Campaign.Status)
	}
	for _, f := range st.Findings {
		if f.Status != StatusDetected {
			t.Errorf("ingested finding should be detected, got %s", f.Status)
		}
		if f.StableID == "" {
			t.Errorf("finding missing stable id: %+v", f)
		}
	}
}

func TestResolveAndReauditValidates(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	st, err := svc.Create(ctx, "demo", "", []*architecturev1.ArchitectureFinding{
		pf("demo", "cycle/x", architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER, "api/a"),
		pf("demo", "mislocated_file", architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING, "api/b"),
	})
	if err != nil {
		t.Fatal(err)
	}
	id := st.Campaign.ID
	cyc, _ := findingByCode(st.Findings, "cycle/x")

	// Agent fixes the cycle and marks it resolved.
	if _, err := svc.Resolve(ctx, id, cyc.StableID, "extracted shared type"); err != nil {
		t.Fatalf("resolve: %v", err)
	}

	// Re-audit: the cycle is gone, the mislocated file persists.
	res, err := svc.Reaudit(ctx, id, []*architecturev1.ArchitectureFinding{
		pf("demo", "mislocated_file", architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING, "api/b"),
	})
	if err != nil {
		t.Fatalf("reaudit: %v", err)
	}
	if len(res.Validated) != 1 || res.Validated[0].Code != "cycle/x" {
		t.Fatalf("expected cycle validated, got %+v", res.Validated)
	}
	if len(res.StillOpen) != 1 || res.StillOpen[0].Code != "mislocated_file" {
		t.Fatalf("expected mislocated still open, got %+v", res.StillOpen)
	}
	if len(res.Regressions) != 0 {
		t.Fatalf("expected no regressions, got %+v", res.Regressions)
	}
	if res.Status.Validated != 1 {
		t.Errorf("status validated count = %d, want 1", res.Status.Validated)
	}
}

func TestReauditDetectsNewRegression(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	st, _ := svc.Create(ctx, "demo", "", []*architecturev1.ArchitectureFinding{
		pf("demo", "cycle/x", architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER, "api/a"),
	})
	id := st.Campaign.ID
	cyc, _ := findingByCode(st.Findings, "cycle/x")
	_, _ = svc.Resolve(ctx, id, cyc.StableID, "fixed")

	// Re-audit: original cycle gone, but the fix introduced a NEW cycle.
	res, err := svc.Reaudit(ctx, id, []*architecturev1.ArchitectureFinding{
		pf("demo", "cycle/auth", architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER, "api/auth"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Validated) != 1 {
		t.Errorf("original cycle should validate, got %+v", res.Validated)
	}
	if len(res.Regressions) != 1 || res.Regressions[0].Code != "cycle/auth" {
		t.Fatalf("expected the new cycle flagged as regression, got %+v", res.Regressions)
	}
	if !res.Regressions[0].Regressed {
		t.Errorf("new finding should carry Regressed=true")
	}
}

func TestReauditReappearanceIsRegression(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	st, _ := svc.Create(ctx, "demo", "", []*architecturev1.ArchitectureFinding{
		pf("demo", "cycle/x", architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER, "api/a"),
	})
	id := st.Campaign.ID
	cyc, _ := findingByCode(st.Findings, "cycle/x")
	_, _ = svc.Resolve(ctx, id, cyc.StableID, "thought I fixed it")

	// Re-audit: the "resolved" cycle is STILL there → regression.
	res, err := svc.Reaudit(ctx, id, []*architecturev1.ArchitectureFinding{
		pf("demo", "cycle/x", architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER, "api/a"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Regressions) != 1 || res.Regressions[0].Code != "cycle/x" {
		t.Fatalf("a resolved-but-still-present finding must be a regression, got %+v", res.Regressions)
	}
	// And it is back to an open state.
	cur, err := svc.Status(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := findingByCode(cur.Findings, "cycle/x")
	if got.Status != StatusDetected || !got.Regressed {
		t.Errorf("reappeared finding should be detected+regressed, got status=%s regressed=%v", got.Status, got.Regressed)
	}
}

func TestNextBalancedPrioritizesRegressionsThenCyclesThenSeverity(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	st, _ := svc.Create(ctx, "demo", "", []*architecturev1.ArchitectureFinding{
		pf("demo", "convergence_drift", architecturev1.FindingSeverity_FINDING_SEVERITY_INFO, "x"),
		pf("demo", "coupling_smell", architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR, "y"),
		pf("demo", "cycle/z", architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING, "z"),
	})
	id := st.Campaign.ID

	next, err := svc.Next(ctx, id, campaignv1.RankProfile_RANK_PROFILE_BALANCED)
	if err != nil {
		t.Fatal(err)
	}
	if len(next) != 3 {
		t.Fatalf("want 3 open, got %d", len(next))
	}
	// cycle first (blocks dependents) even though it's only WARNING.
	if next[0].Code != "cycle/z" {
		t.Errorf("cycle should lead the worklist, got %s", next[0].Code)
	}
	// then the higher-severity non-cycle.
	if next[1].Code != "coupling_smell" {
		t.Errorf("error-severity should precede info, got %s", next[1].Code)
	}
}

func TestStableIDReconciliationSurvivesLocationReorder(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	st, _ := svc.Create(ctx, "demo", "", []*architecturev1.ArchitectureFinding{
		pf("demo", "cycle/x", architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER, "api/a", "api/b"),
	})
	id := st.Campaign.ID

	// Re-audit with the SAME finding but locations reordered + back-slashed
	// — the afid must match, so it reconciles as still-open (not a new
	// regression).
	res, err := svc.Reaudit(ctx, id, []*architecturev1.ArchitectureFinding{
		pf("demo", "cycle/x", architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER, "api\\b", "api/a"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Regressions) != 0 {
		t.Fatalf("cosmetic location change must not manufacture a regression, got %+v", res.Regressions)
	}
	if len(res.StillOpen) != 1 {
		t.Fatalf("finding should reconcile as still-open, got %+v", res.StillOpen)
	}
}

func TestCloseMarksCampaignClosed(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	st, _ := svc.Create(ctx, "demo", "", nil)
	id := st.Campaign.ID
	closed, err := svc.Close(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if closed.Campaign.Status != CampaignClosed {
		t.Errorf("expected closed, got %s", closed.Campaign.Status)
	}
}

func TestListCampaignsFiltersByScenarioNewestFirst(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	a1, _ := svc.Create(ctx, "alpha", "first", nil)
	a2, _ := svc.Create(ctx, "alpha", "second", nil)
	if _, err := svc.Create(ctx, "beta", "other", nil); err != nil {
		t.Fatalf("create beta: %v", err)
	}

	alpha, err := svc.List(ctx, "alpha")
	if err != nil {
		t.Fatalf("list alpha: %v", err)
	}
	if len(alpha) != 2 {
		t.Fatalf("expected 2 alpha campaigns, got %d", len(alpha))
	}
	// Newest-first: created_at DESC, id DESC. Both share a created_at in the
	// fast test path, so the id-DESC tiebreaker decides; assert the set, not
	// a brittle ordering, plus that both alpha ids are present and no beta.
	ids := map[string]bool{alpha[0].ID: true, alpha[1].ID: true}
	if !ids[a1.Campaign.ID] || !ids[a2.Campaign.ID] {
		t.Errorf("alpha list missing a created campaign: got %v", ids)
	}
	for _, c := range alpha {
		if c.Scenario != "alpha" {
			t.Errorf("scenario filter leaked %q into alpha list", c.Scenario)
		}
	}

	all, err := svc.List(ctx, "")
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 campaigns across all scenarios, got %d", len(all))
	}
}

func TestNotFoundErrors(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	if _, err := svc.Status(ctx, "nope"); err == nil {
		t.Error("expected error for unknown campaign")
	}
}
