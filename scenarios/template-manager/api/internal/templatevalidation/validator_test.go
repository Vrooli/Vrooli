package templatevalidation

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/catalog"
)

type fakeRepo struct {
	template catalog.TemplateRecord
	debt     []catalog.DebtEntry
}

func (f fakeRepo) ListTemplates(context.Context, catalog.TemplateKind) ([]catalog.TemplateRecord, error) {
	return nil, nil
}

func (f fakeRepo) GetTemplate(context.Context, string) (catalog.TemplateRecord, error) {
	if f.template.ID == "" {
		return catalog.TemplateRecord{}, catalog.ErrNotFound{Kind: "template", ID: "missing"}
	}
	return f.template, nil
}

func (f fakeRepo) SyncScenarioTemplates(context.Context, []catalog.ScenarioTemplate) error {
	return nil
}
func (f fakeRepo) SaveValidationRun(context.Context, catalog.ValidationRun) error { return nil }
func (f fakeRepo) ListValidationRuns(context.Context, string) ([]catalog.ValidationRun, error) {
	return nil, nil
}

func (f fakeRepo) GetValidationRun(context.Context, string) (catalog.ValidationRun, error) {
	return catalog.ValidationRun{}, nil
}
func (f fakeRepo) SaveDriftSnapshot(context.Context, catalog.DriftSnapshot) error { return nil }
func (f fakeRepo) SupersedePendingDriftSnapshots(context.Context, time.Time) error {
	return nil
}

func (f fakeRepo) ListDriftSnapshots(context.Context, string) ([]catalog.DriftSnapshot, error) {
	return nil, nil
}
func (f fakeRepo) UpsertDebt(context.Context, catalog.DebtEntry) error        { return nil }
func (f fakeRepo) ResolveSourceDebt(context.Context, string, time.Time) error { return nil }
func (f fakeRepo) ResolveSupersededDeepValidationDebt(context.Context, string, time.Time) error {
	return nil
}

func (f fakeRepo) ListDebt(context.Context, string, string) ([]catalog.DebtEntry, error) {
	return f.debt, nil
}

func (f fakeRepo) GetDebt(context.Context, string) (catalog.DebtEntry, error) {
	return catalog.DebtEntry{}, nil
}

func TestValidateScenarioReportsMissingProvenance(t *testing.T) {
	root := t.TempDir()
	writeService(t, root, map[string]any{"service": map[string]any{"name": "legacy"}})

	validator := NewValidator("", fakeRepo{})
	report, err := validator.ValidateScenario(context.Background(), "legacy", root)
	if err != nil {
		t.Fatalf("ValidateScenario() error = %v", err)
	}
	if report.Scenario != "legacy" {
		t.Fatalf("scenario = %q, want legacy", report.Scenario)
	}
	if len(report.Findings) != 1 || report.Findings[0].Code != CodeProvenanceMissing {
		t.Fatalf("findings = %#v, want %s", report.Findings, CodeProvenanceMissing)
	}
	if !report.Findings[0].Autofix {
		t.Fatal("missing provenance finding should advertise autofix")
	}
}

func TestValidateScenarioReadsProvenanceAndLedgerDebt(t *testing.T) {
	root := t.TempDir()
	writeService(t, root, map[string]any{
		"generation": map[string]any{
			"template": map[string]any{"id": "react-vite", "version": "1.5.0"},
		},
	})
	if err := os.WriteFile(filepath.Join(root, "docs", "START-HERE.md"), []byte("start"), 0o644); err == nil {
		t.Fatal("expected missing docs dir write to fail")
	}
	validator := NewValidator("", fakeRepo{
		template: catalog.TemplateRecord{ID: "react-vite", LatestVersion: "1.6.0"},
		debt:     []catalog.DebtEntry{{Key: "react-vite.test", TemplateID: "react-vite"}},
	})
	validator.Now = func() time.Time { return time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC) }

	report, err := validator.ValidateScenario(context.Background(), "demo", root)
	if err != nil {
		t.Fatalf("ValidateScenario() error = %v", err)
	}
	gotCodes := map[string]bool{}
	for _, finding := range report.Findings {
		gotCodes[finding.Code] = true
	}
	if !gotCodes[CodeTemplateVersionLag] {
		t.Fatalf("missing %s in %#v", CodeTemplateVersionLag, report.Findings)
	}
	if !gotCodes[CodeInheritedDebtOutstanding] {
		t.Fatalf("missing %s in %#v", CodeInheritedDebtOutstanding, report.Findings)
	}
}

func TestProvenanceFixPreviewAndApply(t *testing.T) {
	repoRoot := t.TempDir()
	templateDir := filepath.Join(repoRoot, "templates", "scenarios", "react-vite")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "template.json"), []byte(`{"version":"1.6.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	writeService(t, root, map[string]any{"service": map[string]any{"name": "legacy"}})

	registry := NewFixRegistry(repoRoot)
	preview, err := registry.Preview(root, []string{CodeProvenanceMissing})
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}
	if len(preview) != 1 {
		t.Fatalf("preview count = %d, want 1", len(preview))
	}
	applied, err := registry.Apply(root, []string{CodeProvenanceMissing})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if len(applied) != 1 {
		t.Fatalf("applied count = %d, want 1", len(applied))
	}
	prov, found, err := readProvenance(root)
	if err != nil {
		t.Fatalf("readProvenance() error = %v", err)
	}
	if !found || prov.TemplateID != "react-vite" || prov.TemplateVersion != "1.6.0" || !prov.Adopted {
		t.Fatalf("provenance = %#v, found=%v", prov, found)
	}
}

func writeService(t *testing.T, root string, doc map[string]any) {
	t.Helper()
	dir := filepath.Join(root, ".vrooli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "service.json"), append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}
