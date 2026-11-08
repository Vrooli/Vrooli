package validationrunner

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/catalog"

	apidb "github.com/vrooli/api-core/database"
	testdb "github.com/vrooli/api-core/databasetest"
)

func TestServiceRunValidationPersistsRunAndDebt(t *testing.T) {
	repo := newRepo(t)
	runner := &fakeRunner{
		validate: ValidateResult{
			Success:    false,
			TemplateID: "react-vite",
			Mode:       catalog.ModeShallow,
			Findings: []catalog.ValidationFinding{
				{Key: "react-vite.lint.problem", Severity: "warning", Summary: "Lint problem", Source: "template validate"},
			},
		},
	}
	service := NewService(repo, runner)
	now := time.Date(2026, 7, 9, 5, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	run, err := service.RunValidation(context.Background(), ValidateRequest{TemplateID: "react-vite", Mode: catalog.ModeShallow})
	if err != nil {
		t.Fatalf("RunValidation: %v", err)
	}
	if run.Status != "failed" || len(run.Findings) != 1 {
		t.Fatalf("run = %+v, want failed run with one finding", run)
	}
	if _, err := repo.GetValidationRun(context.Background(), run.ID); err != nil {
		t.Fatalf("GetValidationRun(%q): %v", run.ID, err)
	}
	entry, err := repo.GetDebt(context.Background(), "react-vite.lint.problem")
	if err != nil {
		t.Fatalf("GetDebt: %v", err)
	}
	if entry.Title != "Lint problem" || entry.Status != "open" {
		t.Fatalf("debt = %+v, want open lint problem", entry)
	}
}

func TestServiceRecordFleetDriftPersistsSnapshotAndDeduplicatesDebt(t *testing.T) {
	repo := newRepo(t)
	runner := &fakeRunner{
		drift: DriftResult{
			Success: true,
			Scenarios: []DriftScenario{
				{Scenario: "template-manager", TemplateID: "react-vite", Status: "ok"},
				{Scenario: "legacy-app", TemplateID: "react-vite", Status: "drifted", ManifestDrifted: true},
			},
		},
	}
	service := NewService(repo, runner)
	now := time.Date(2026, 7, 9, 5, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	first, err := service.RecordFleetDrift(context.Background())
	if err != nil {
		t.Fatalf("RecordFleetDrift first: %v", err)
	}
	second, err := service.RecordFleetDrift(context.Background())
	if err != nil {
		t.Fatalf("RecordFleetDrift second: %v", err)
	}
	if first.DriftCount != 1 || second.DriftCount != 1 {
		t.Fatalf("drift counts = %d/%d, want 1/1", first.DriftCount, second.DriftCount)
	}
	entries, err := repo.ListDebt(context.Background(), "react-vite", "open")
	if err != nil {
		t.Fatalf("ListDebt: %v", err)
	}
	count := 0
	for _, entry := range entries {
		if entry.Key == "drift.react-vite.legacy-app" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("drift debt count = %d, want 1", count)
	}
}

func TestServiceRecordFleetDriftResolvesOnlyFreshlyCleanScenarioDebt(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	old := time.Date(2026, 7, 8, 5, 0, 0, 0, time.UTC)
	for _, entry := range []catalog.DebtEntry{
		{Key: "drift.react-vite.fixed-app", TemplateID: "react-vite", Source: "template drift", Severity: "warning", Status: "open", Title: "old drift", Detail: "old drift", FirstSeenAt: old, LastSeenAt: old},
		{Key: "drift.react-vite.still-drifted", TemplateID: "react-vite", Source: "template drift", Severity: "warning", Status: "open", Title: "still drifted", Detail: "still drifted", FirstSeenAt: old, LastSeenAt: old},
		{Key: "drift.react-vite.hash-error", TemplateID: "react-vite", Source: "template drift", Severity: "warning", Status: "open", Title: "unknown", Detail: "unknown", FirstSeenAt: old, LastSeenAt: old},
		{Key: "drift.react-vite.flags-disagree", TemplateID: "react-vite", Source: "template drift", Severity: "warning", Status: "open", Title: "unknown", Detail: "unknown", FirstSeenAt: old, LastSeenAt: old},
		{Key: "drift.legacy-app", TemplateID: "react-vite", Source: "template drift", Severity: "warning", Status: "open", Title: "legacy key", Detail: "legacy key", FirstSeenAt: old, LastSeenAt: old},
		{Key: "react-vite.fixed-app.source", TemplateID: "react-vite", Source: "template validation", Severity: "error", Status: "open", Title: "source debt", Detail: "source debt", FirstSeenAt: old, LastSeenAt: old},
		{Key: "drift.react-vite.source-owned-app", TemplateID: "react-vite", Source: "template validation", Severity: "error", Status: "open", Title: "source debt", Detail: "source debt", FirstSeenAt: old, LastSeenAt: old},
	} {
		if err := repo.UpsertDebt(ctx, entry); err != nil {
			t.Fatalf("UpsertDebt(%q): %v", entry.Key, err)
		}
	}

	runner := &fakeRunner{drift: DriftResult{Success: true, Scenarios: []DriftScenario{
		{Scenario: "fixed-app", TemplateID: "react-vite", Status: "ok"},
		{Scenario: "still-drifted", TemplateID: "react-vite", Status: "drifted", ContentDrifted: true},
		{Scenario: "hash-error", TemplateID: "react-vite", Status: "hash_error"},
		{Scenario: "unverifiable-app", TemplateID: "react-vite", Status: "hash_error", Message: "hashing failed"},
		{Scenario: "contradictory-drift-app", TemplateID: "react-vite", Status: "drifted"},
		{Scenario: "flags-disagree", TemplateID: "react-vite", Status: "ok", ContentDrifted: true},
		{Scenario: "legacy-app", Status: "ok"},
		{Scenario: "source-owned-app", TemplateID: "react-vite", Status: "ok"},
	}}}
	service := NewService(repo, runner)
	now := old.Add(24 * time.Hour)
	service.now = func() time.Time { return now }

	snapshot, err := service.RecordFleetDrift(ctx)
	if err != nil {
		t.Fatalf("RecordFleetDrift: %v", err)
	}
	if snapshot.DriftCount != 2 {
		t.Fatalf("DriftCount = %d, want only the two scenarios with confirmed drift", snapshot.DriftCount)
	}
	for key, want := range map[string]string{
		"drift.react-vite.fixed-app":               "resolved",
		"drift.react-vite.still-drifted":           "open",
		"drift.react-vite.hash-error":              "open",
		"drift.react-vite.unverifiable-app":        "open",
		"drift.react-vite.contradictory-drift-app": "open",
		"drift.react-vite.flags-disagree":          "open",
		"drift.legacy-app":                         "resolved",
		"react-vite.fixed-app.source":              "open",
		"drift.react-vite.source-owned-app":        "open",
	} {
		entry, err := repo.GetDebt(ctx, key)
		if err != nil {
			t.Fatalf("GetDebt(%q): %v", key, err)
		}
		if entry.Status != want {
			t.Errorf("debt %q status = %q, want %q", key, entry.Status, want)
		}
		if want == "resolved" && !entry.LastSeenAt.Equal(now) {
			t.Errorf("debt %q last_seen_at = %s, want %s", key, entry.LastSeenAt, now)
		}
	}
	unverifiable, err := repo.GetDebt(ctx, "drift.react-vite.unverifiable-app")
	if err != nil {
		t.Fatalf("GetDebt(unverifiable-app): %v", err)
	}
	if unverifiable.Title != "unverifiable-app template drift could not be verified" || unverifiable.Detail != "hashing failed" {
		t.Fatalf("unverifiable debt = %+v, want an explicit uncertainty finding", unverifiable)
	}
	contradictory, err := repo.GetDebt(ctx, "drift.react-vite.contradictory-drift-app")
	if err != nil {
		t.Fatalf("GetDebt(contradictory-drift-app): %v", err)
	}
	if contradictory.Title != "contradictory-drift-app template drift could not be verified" {
		t.Fatalf("contradictory debt = %+v, want explicit uncertainty rather than confirmed drift", contradictory)
	}

	// A later drift observation reopens the exact debt key while retaining its
	// original first-seen time, while a clean observation resolves uncertainty.
	runner.drift = DriftResult{Success: true, Scenarios: []DriftScenario{
		{Scenario: "fixed-app", TemplateID: "react-vite", Status: "drifted", ManifestDrifted: true},
		{Scenario: "unverifiable-app", TemplateID: "react-vite", Status: "ok"},
	}}
	next := now.Add(time.Hour)
	service.now = func() time.Time { return next }
	if _, err := service.RecordFleetDrift(ctx); err != nil {
		t.Fatalf("RecordFleetDrift after regression: %v", err)
	}
	reopened, err := repo.GetDebt(ctx, "drift.react-vite.fixed-app")
	if err != nil {
		t.Fatalf("GetDebt after regression: %v", err)
	}
	if reopened.Status != "open" || !reopened.FirstSeenAt.Equal(old) || !reopened.LastSeenAt.Equal(next) {
		t.Errorf("reopened debt = %+v, want open with preserved first-seen and refreshed last-seen", reopened)
	}
	resolved, err := repo.GetDebt(ctx, "drift.react-vite.unverifiable-app")
	if err != nil {
		t.Fatalf("GetDebt after clean observation: %v", err)
	}
	if resolved.Status != "resolved" || !resolved.FirstSeenAt.Equal(now) || !resolved.LastSeenAt.Equal(next) {
		t.Errorf("uncertainty debt after clean observation = %+v, want resolved with first-seen retained", resolved)
	}
}

func TestServiceRecordFleetDriftSupersedesBootstrapPendingSnapshot(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	before, err := repo.ListDriftSnapshots(ctx, "react-vite")
	if err != nil {
		t.Fatalf("ListDriftSnapshots before: %v", err)
	}
	pendingBefore := 0
	for _, s := range before {
		if s.Status == "pending-live-run" {
			pendingBefore++
		}
	}
	if pendingBefore == 0 {
		t.Fatalf("expected a seeded pending-live-run snapshot before recording, got %+v", before)
	}

	runner := &fakeRunner{drift: DriftResult{
		Success:   true,
		Scenarios: []DriftScenario{{Scenario: "template-manager", TemplateID: "react-vite", Status: "ok"}},
	}}
	service := NewService(repo, runner)
	now := time.Date(2026, 7, 15, 5, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	if _, err := service.RecordFleetDrift(ctx); err != nil {
		t.Fatalf("RecordFleetDrift: %v", err)
	}

	after, err := repo.ListDriftSnapshots(ctx, "react-vite")
	if err != nil {
		t.Fatalf("ListDriftSnapshots after: %v", err)
	}
	for _, s := range after {
		if s.Status == "pending-live-run" {
			t.Fatalf("pending-live-run snapshot %q survived a live run", s.ID)
		}
	}
	superseded := 0
	for _, s := range after {
		if s.Status == "superseded" {
			superseded++
		}
	}
	if superseded != pendingBefore {
		t.Fatalf("superseded count = %d, want %d", superseded, pendingBefore)
	}
}

func TestServiceCleanDeepValidationResolvesSourceDebtButPreservesDrift(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 10, 18, 0, 0, 0, time.UTC)
	for _, entry := range []catalog.DebtEntry{
		{Key: "react-vite.source", TemplateID: "react-vite", Source: "template validation", Severity: "error", Status: "open", Title: "source", Detail: "source", FirstSeenAt: now, LastSeenAt: now},
		{Key: "drift.react-vite.downstream", TemplateID: "react-vite", Source: "template drift", Severity: "warning", Status: "open", Title: "drift", Detail: "drift", FirstSeenAt: now, LastSeenAt: now},
	} {
		if err := repo.UpsertDebt(ctx, entry); err != nil {
			t.Fatalf("UpsertDebt(%q): %v", entry.Key, err)
		}
	}
	service := NewService(repo, &fakeRunner{validate: ValidateResult{Success: true, Mode: catalog.ModeDeep, TemplateID: "react-vite"}})
	service.now = func() time.Time { return now }
	if _, err := service.RunValidation(ctx, ValidateRequest{TemplateID: "react-vite", Mode: catalog.ModeDeep}); err != nil {
		t.Fatalf("RunValidation: %v", err)
	}

	source, err := repo.GetDebt(ctx, "react-vite.source")
	if err != nil {
		t.Fatalf("GetDebt(source): %v", err)
	}
	if source.Status != "resolved" {
		t.Fatalf("source status = %q, want resolved", source.Status)
	}
	drift, err := repo.GetDebt(ctx, "drift.react-vite.downstream")
	if err != nil {
		t.Fatalf("GetDebt(drift): %v", err)
	}
	if drift.Status != "open" {
		t.Fatalf("drift status = %q, want open", drift.Status)
	}
}

func TestServiceFailedDeepValidationSupersedesPriorTestGenieDebt(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 11, 4, 30, 0, 0, time.UTC)
	for _, entry := range []catalog.DebtEntry{
		{Key: "react-vite.test-genie-deep-validation-failed-old-summary", TemplateID: "react-vite", Source: "react-vite", Severity: "warning", Status: "open", Title: "old deep failure", Detail: "old", FirstSeenAt: now, LastSeenAt: now},
		{Key: "react-vite.test-genie.deep-validation.phase-results", TemplateID: "react-vite", Source: "react-vite", Severity: "warning", Status: "open", Title: "old canonical deep failure", Detail: "old", FirstSeenAt: now, LastSeenAt: now},
		{Key: "react-vite.source.lint", TemplateID: "react-vite", Source: "template validation", Severity: "warning", Status: "open", Title: "source lint", Detail: "source", FirstSeenAt: now, LastSeenAt: now},
	} {
		if err := repo.UpsertDebt(ctx, entry); err != nil {
			t.Fatalf("UpsertDebt(%q): %v", entry.Key, err)
		}
	}
	service := NewService(repo, &fakeRunner{validate: ValidateResult{
		Success: false, Mode: catalog.ModeDeep, TemplateID: "react-vite",
		Findings: []catalog.ValidationFinding{{
			Key: "react-vite.test-genie.deep-validation.startup", Severity: "warning", Summary: "current startup failure", Source: "react-vite",
		}},
	}})
	service.now = func() time.Time { return now }
	if _, err := service.RunValidation(ctx, ValidateRequest{TemplateID: "react-vite", Mode: catalog.ModeDeep}); err != nil {
		t.Fatalf("RunValidation: %v", err)
	}

	for _, key := range []string{
		"react-vite.test-genie-deep-validation-failed-old-summary",
		"react-vite.test-genie.deep-validation.phase-results",
	} {
		entry, err := repo.GetDebt(ctx, key)
		if err != nil {
			t.Fatalf("GetDebt(%q): %v", key, err)
		}
		if entry.Status != "resolved" {
			t.Fatalf("debt %q status = %q, want resolved", key, entry.Status)
		}
	}
	current, err := repo.GetDebt(ctx, "react-vite.test-genie.deep-validation.startup")
	if err != nil {
		t.Fatalf("GetDebt(current): %v", err)
	}
	if current.Status != "open" {
		t.Fatalf("current debt status = %q, want open", current.Status)
	}
	source, err := repo.GetDebt(ctx, "react-vite.source.lint")
	if err != nil {
		t.Fatalf("GetDebt(source): %v", err)
	}
	if source.Status != "open" {
		t.Fatalf("unrelated source debt status = %q, want open", source.Status)
	}
}

func TestServicePersistsCompletedValidationAfterRequestCancellation(t *testing.T) {
	repo := newRepo(t)
	ctx, cancel := context.WithCancel(context.Background())
	runner := &cancelingRunner{
		cancel: cancel,
		result: ValidateResult{
			Success:    false,
			TemplateID: "react-vite",
			Mode:       catalog.ModeDeep,
			Findings: []catalog.ValidationFinding{{
				Key:      "react-vite.deep.failure",
				Severity: "warning",
				Summary:  "deep validation failed after execution",
				Source:   "test",
			}},
		},
	}

	run, err := NewService(repo, runner).RunValidation(ctx, ValidateRequest{TemplateID: "react-vite", Mode: catalog.ModeDeep})
	if err != nil {
		t.Fatalf("RunValidation: %v", err)
	}
	if _, err := repo.GetValidationRun(context.Background(), run.ID); err != nil {
		t.Fatalf("GetValidationRun(%q): %v", run.ID, err)
	}
	if _, err := repo.GetDebt(context.Background(), "react-vite.deep.failure"); err != nil {
		t.Fatalf("GetDebt: %v", err)
	}
	if runner.contextErr != nil {
		t.Fatalf("runner context canceled before terminal result: %v", runner.contextErr)
	}
}

func TestServiceShallowValidationHonorsRequestCancellation(t *testing.T) {
	repo := newRepo(t)
	ctx, cancel := context.WithCancel(context.Background())
	runner := &requestCancelingRunner{cancel: cancel}

	_, err := NewService(repo, runner).RunValidation(ctx, ValidateRequest{
		TemplateID: "react-vite",
		Mode:       catalog.ModeShallow,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunValidation error = %v, want context.Canceled", err)
	}
	if !errors.Is(runner.contextErr, context.Canceled) {
		t.Fatalf("runner context error = %v, want context.Canceled", runner.contextErr)
	}
	runs, err := repo.ListValidationRuns(context.Background(), "react-vite")
	if err != nil {
		t.Fatalf("ListValidationRuns: %v", err)
	}
	for _, run := range runs {
		if run.Trigger == "manual" {
			t.Fatalf("persisted canceled shallow run = %+v, want no terminal run", run)
		}
	}
}

// A scan that did not complete is not an observation: it must neither record
// a snapshot nor resolve or open drift debt.
func TestServiceRecordFleetDriftRefusesIncompleteScanWithoutTouchingDebt(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	old := time.Date(2026, 7, 8, 5, 0, 0, 0, time.UTC)
	open := catalog.DebtEntry{Key: "drift.react-vite.fixed-app", TemplateID: "react-vite", Source: "template drift", Severity: "warning", Status: "open", Title: "old drift", Detail: "old drift", FirstSeenAt: old, LastSeenAt: old}
	if err := repo.UpsertDebt(ctx, open); err != nil {
		t.Fatal(err)
	}
	before, err := repo.ListDriftSnapshots(ctx, "react-vite")
	if err != nil {
		t.Fatal(err)
	}

	runner := &fakeRunner{drift: DriftResult{Success: false, Scenarios: []DriftScenario{
		{Scenario: "fixed-app", TemplateID: "react-vite", Status: "ok"},
	}}}
	service := NewService(repo, runner)
	service.now = func() time.Time { return old.Add(24 * time.Hour) }

	if _, err := service.RecordFleetDrift(ctx); err == nil || !strings.Contains(err.Error(), "did not complete") {
		t.Fatalf("RecordFleetDrift error = %v, want incomplete-scan refusal", err)
	}
	entries, err := repo.ListDebt(ctx, "react-vite", "open")
	if err != nil {
		t.Fatal(err)
	}
	stillOpen := false
	for _, entry := range entries {
		if entry.Key == open.Key {
			stillOpen = entry.LastSeenAt.Equal(old)
		}
	}
	if !stillOpen {
		t.Fatalf("drift debt %q must stay open and unmodified after an incomplete scan", open.Key)
	}
	after, err := repo.ListDriftSnapshots(ctx, "react-vite")
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		t.Fatalf("snapshots %d -> %d; an incomplete scan must not record one", len(before), len(after))
	}
}

type fakeRunner struct {
	validate ValidateResult
	drift    DriftResult
}

func (f *fakeRunner) ValidateTemplate(context.Context, ValidateRequest) (ValidateResult, error) {
	return f.validate, nil
}

func (f *fakeRunner) RecordFleetDrift(context.Context) (DriftResult, error) {
	return f.drift, nil
}

type cancelingRunner struct {
	cancel     context.CancelFunc
	result     ValidateResult
	contextErr error
}

type requestCancelingRunner struct {
	cancel     context.CancelFunc
	contextErr error
}

func (r *requestCancelingRunner) ValidateTemplate(ctx context.Context, _ ValidateRequest) (ValidateResult, error) {
	r.cancel()
	r.contextErr = ctx.Err()
	return ValidateResult{}, r.contextErr
}

func (r *requestCancelingRunner) RecordFleetDrift(context.Context) (DriftResult, error) {
	return DriftResult{}, nil
}

func (r *cancelingRunner) ValidateTemplate(ctx context.Context, _ ValidateRequest) (ValidateResult, error) {
	r.cancel()
	r.contextErr = ctx.Err()
	return r.result, nil
}

func (r *cancelingRunner) RecordFleetDrift(context.Context) (DriftResult, error) {
	return DriftResult{}, nil
}

func newRepo(t *testing.T) catalog.Repository {
	t.Helper()
	db := testdb.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(catalog.Schema)); err != nil {
		t.Fatalf("EnsureSchemas: %v", err)
	}
	return catalog.NewSQLiteRepository(db)
}
