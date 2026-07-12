package catalog

import (
	"context"
	"testing"
	"time"

	apidb "github.com/vrooli/api-core/database"

	testdb "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/testutil/db"
)

func TestSQLiteRepositorySaveValidationRunRoundTrip(t *testing.T) {
	repo := newTestRepository(t)
	now := time.Date(2026, 7, 9, 4, 30, 0, 0, time.UTC)
	run := ValidationRun{
		ID:         "run-1",
		TemplateID: "react-vite",
		Mode:       ModeShallow,
		Target:     "templates/scenarios/react-vite",
		Status:     "passed",
		StartedAt:  now,
		FinishedAt: now.Add(time.Second),
		PhaseResults: []PhaseResult{
			{Phase: "template-validate", Status: "passed", FindingCount: 1},
		},
		Findings: []ValidationFinding{
			{Key: "react-vite.example", Severity: "warning", Summary: "Example", Source: "test"},
		},
	}

	if err := repo.SaveValidationRun(context.Background(), run); err != nil {
		t.Fatalf("SaveValidationRun: %v", err)
	}
	got, err := repo.GetValidationRun(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("GetValidationRun: %v", err)
	}
	if got.ID != run.ID || got.TemplateID != run.TemplateID || got.Mode != run.Mode || len(got.Findings) != 1 {
		t.Fatalf("GetValidationRun() = %+v, want persisted run %+v", got, run)
	}
}

func TestSQLiteRepositoryUpsertDebtDeduplicatesByKey(t *testing.T) {
	repo := newTestRepository(t)
	first := time.Date(2026, 7, 9, 4, 30, 0, 0, time.UTC)
	entry := DebtEntry{
		Key:         "react-vite.duplicate",
		TemplateID:  "react-vite",
		Source:      "test",
		Severity:    "warning",
		Status:      "open",
		Title:       "Original",
		Detail:      "Original detail",
		FirstSeenAt: first,
		LastSeenAt:  first,
	}
	if err := repo.UpsertDebt(context.Background(), entry); err != nil {
		t.Fatalf("UpsertDebt first: %v", err)
	}
	entry.Title = "Updated"
	entry.Detail = "Updated detail"
	entry.LastSeenAt = first.Add(time.Hour)
	if err := repo.UpsertDebt(context.Background(), entry); err != nil {
		t.Fatalf("UpsertDebt second: %v", err)
	}
	entries, err := repo.ListDebt(context.Background(), "react-vite", "open")
	if err != nil {
		t.Fatalf("ListDebt: %v", err)
	}
	count := 0
	var got DebtEntry
	for _, candidate := range entries {
		if candidate.Key == entry.Key {
			count++
			got = candidate
		}
	}
	if count != 1 {
		t.Fatalf("debt key count = %d, want 1", count)
	}
	if got.Title != "Updated" || !got.FirstSeenAt.Equal(first) || !got.LastSeenAt.Equal(first.Add(time.Hour)) {
		t.Fatalf("debt entry = %+v, want updated title with preserved first_seen_at and new last_seen_at", got)
	}
}

func TestSQLiteRepositorySeededDebtIncludesPhase3Audit(t *testing.T) {
	repo := newTestRepository(t)
	entries, err := repo.ListDebt(context.Background(), "", "open")
	if err != nil {
		t.Fatalf("ListDebt: %v", err)
	}
	if len(entries) < 18 {
		t.Fatalf("seeded debt count = %d, want at least 18", len(entries))
	}
	if _, err := repo.GetDebt(context.Background(), "landing-page-react-vite.contract.missing-version"); err != nil {
		t.Fatalf("GetDebt landing page missing version: %v", err)
	}
}

func TestSQLiteRepositoryProjectsScenarioReleaseStatus(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 10, 18, 0, 0, 0, time.UTC)
	if err := repo.SyncScenarioTemplates(ctx, []ScenarioTemplate{{
		ID:           "candidate",
		Version:      "1.0.0",
		ManifestPath: "templates/scenarios/candidate/template.json",
		SourcePath:   "templates/scenarios/candidate",
		UpdatedAt:    now,
	}}); err != nil {
		t.Fatalf("SyncScenarioTemplates: %v", err)
	}

	assertStatus := func(want string) {
		t.Helper()
		record, err := repo.GetTemplate(ctx, "candidate")
		if err != nil {
			t.Fatalf("GetTemplate: %v", err)
		}
		if record.Status != want {
			t.Fatalf("status = %q, want %q", record.Status, want)
		}
	}
	assertStatus("quarantined")

	if err := repo.SaveValidationRun(ctx, ValidationRun{
		ID: "candidate-failed", TemplateID: "candidate", Mode: ModeDeep, Status: "failed", StartedAt: now, FinishedAt: now,
	}); err != nil {
		t.Fatalf("SaveValidationRun(failed): %v", err)
	}
	assertStatus("quarantined")

	if err := repo.SaveValidationRun(ctx, ValidationRun{
		ID: "candidate-passed", TemplateID: "candidate", Mode: ModeDeep, Status: "passed", StartedAt: now, FinishedAt: now.Add(time.Second),
	}); err != nil {
		t.Fatalf("SaveValidationRun(passed): %v", err)
	}
	if err := repo.UpsertDebt(ctx, DebtEntry{
		Key: "candidate.source", TemplateID: "candidate", Source: "template validation", Severity: "error", Status: "open", Title: "source defect", Detail: "source defect", FirstSeenAt: now, LastSeenAt: now,
	}); err != nil {
		t.Fatalf("UpsertDebt(source): %v", err)
	}
	assertStatus("debt")

	if err := repo.ResolveSourceDebt(ctx, "candidate", now.Add(2*time.Second)); err != nil {
		t.Fatalf("ResolveSourceDebt: %v", err)
	}
	if err := repo.UpsertDebt(ctx, DebtEntry{
		Key: "drift.candidate.downstream", TemplateID: "candidate", Source: "template drift", Severity: "warning", Status: "open", Title: "downstream drift", Detail: "downstream drift", FirstSeenAt: now, LastSeenAt: now,
	}); err != nil {
		t.Fatalf("UpsertDebt(drift): %v", err)
	}
	assertStatus("active")
}

func TestSQLiteRepositoryListTemplatesProjectsStatusWithSingleConnection(t *testing.T) {
	repo := newTestRepository(t)

	// Production deliberately caps SQLite at one connection. Listing the
	// registry must finish its row scan before deriving each record's status,
	// because status projection performs follow-up queries on that same pool.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	records, err := repo.ListTemplates(ctx, KindScenario)
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(records) == 0 {
		t.Fatal("ListTemplates returned no scenario templates")
	}
}

func newTestRepository(t *testing.T) Repository {
	t.Helper()
	db := testdb.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("EnsureSchemas: %v", err)
	}
	return NewSQLiteRepository(db)
}
