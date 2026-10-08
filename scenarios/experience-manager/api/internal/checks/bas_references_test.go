package checks

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"experience-manager/internal/spec"
)

func TestBASReferenceCheckReportsMissingAndUncoveredRefs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "bas/cases/generated/orphan.json", `{
  "metadata": {"labels": {"spec_entry_id": "missing-page"}}
}`)
	report := spec.Report{
		Scenario:   "demo",
		TargetPath: root,
		Spec: &spec.ScenarioSpec{
			Index: spec.IndexDocument{Pages: []spec.DocumentRef{
				{ID: "home", Path: "pages/home.json", Status: "active"},
			}},
			Pages: map[string]spec.PageDocument{"home": {Page: spec.PageIdentity{ID: "home"}}},
		},
	}

	findings := BASReferenceCheck{}.Run(context.Background(), report)
	if len(findings) != 2 {
		t.Fatalf("findings = %d, want missing case ref + uncovered page: %+v", len(findings), findings)
	}
	if findings[0].Code != spec.CodeRefUnresolved || findings[1].Code != spec.CodeRouteUnspecced {
		t.Fatalf("unexpected findings: %+v", findings)
	}
}

func TestBASReferenceCheckAcceptsSpecEntryLabels(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "bas/cases/generated/home.json", `{
  "metadata": {"labels": {"spec_entry_id": "home"}}
}`)
	report := spec.Report{
		Scenario:   "demo",
		TargetPath: root,
		Spec: &spec.ScenarioSpec{
			Index: spec.IndexDocument{Pages: []spec.DocumentRef{
				{ID: "home", Path: "pages/home.json", Status: "active"},
			}},
			Pages: map[string]spec.PageDocument{"home": {Page: spec.PageIdentity{ID: "home"}}},
		},
	}
	if findings := (BASReferenceCheck{}).Run(context.Background(), report); len(findings) != 0 {
		t.Fatalf("expected no findings, got %+v", findings)
	}
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
