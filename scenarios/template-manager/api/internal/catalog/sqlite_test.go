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

func newTestRepository(t *testing.T) Repository {
	t.Helper()
	db := testdb.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("EnsureSchemas: %v", err)
	}
	return NewSQLiteRepository(db)
}
