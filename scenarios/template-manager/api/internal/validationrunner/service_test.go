package validationrunner

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/catalog"

	apidb "github.com/vrooli/api-core/database"
	testdb "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/testutil/db"
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

func newRepo(t *testing.T) catalog.Repository {
	t.Helper()
	db := testdb.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(catalog.Schema)); err != nil {
		t.Fatalf("EnsureSchemas: %v", err)
	}
	return catalog.NewSQLiteRepository(db)
}
