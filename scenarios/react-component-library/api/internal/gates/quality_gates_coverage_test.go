package gates

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDelegatedBrowserQualityGatesReportTheirExecutionBoundary(t *testing.T) {
	for _, test := range []struct {
		name string
		run  func(Scope) (Result, error)
		gate string
	}{
		{name: "accessibility", run: ValidateAccessibility, gate: "accessibility"},
		{name: "interaction", run: ValidateInteraction, gate: "interaction"},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.run(Scope{})
			if err != nil {
				t.Fatal(err)
			}
			if result.Status != "delegated" || result.Inspected != 0 || len(result.Findings) != 0 {
				t.Fatalf("result = %+v, want delegated and explicitly uninspected", result)
			}
			want := "catalog handler executes " + test.gate + " against a version-pinned browser report"
			if len(result.Skipped) != 1 || result.Skipped[0] != want {
				t.Fatalf("skipped = %v, want [%q]", result.Skipped, want)
			}
		})
	}
}

func TestValidateConsoleCleanUsesCurrentVersionEvidence(t *testing.T) {
	for _, test := range []struct {
		name      string
		evidence  string
		wantCode  string
		wantClean bool
	}{
		{
			name:      "clean browser capture",
			evidence:  `{"kind":"console","console":{"consoleErrors":[],"pageErrors":[],"failedRequests":[]}}`,
			wantClean: true,
		},
		{
			name:     "console error",
			evidence: `{"kind":"console","console":{"consoleErrors":["Warning: invalid prop"],"pageErrors":[],"failedRequests":[]}}`,
			wantCode: "catalog.console_clean",
		},
		{
			name:     "uncaught page error",
			evidence: `{"kind":"console","console":{"consoleErrors":[],"pageErrors":["TypeError: render failed"],"failedRequests":[]}}`,
			wantCode: "catalog.console_clean",
		},
		{
			name:     "failed request",
			evidence: `{"kind":"console","console":{"consoleErrors":[],"pageErrors":[],"failedRequests":["/asset.js"]}}`,
			wantCode: "catalog.console_clean",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeQualityFixtureFile(t, filepath.Join(root, "scenarios/react-component-library/library/components/Button/component.json"), `{"libraryId":"react-component-library:Button","catalogId":"controls.button","latest":"1.0.0"}`)
			writeQualityFixtureFile(t, filepath.Join(root, "scenarios/react-component-library/catalog/assets/components/button.json"), `{"asset":{"id":"controls.button","kind":"component","name":"Button"}}`)
			writeQualityFixtureFile(t, filepath.Join(root, "scenarios/react-component-library/library/components/Button/versions/1.0.0/story.json"), `{"kind":"component","stories":[]}`)
			db, err := openGateDB(context.Background(), filepath.Join(root, "reports.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if _, err := db.ExecContext(context.Background(), `CREATE TABLE component_test_reports (results_json TEXT, created_at TEXT)`); err != nil {
				t.Fatal(err)
			}
			report := `{"results":[{"stage":"declared_behavior","assetLibraryId":"react-component-library:Button","version":"1.0.0","subject":"default","evidence":[` + test.evidence + `]}]}`
			if _, err := db.ExecContext(context.Background(), `INSERT INTO component_test_reports(results_json, created_at) VALUES (?, ?)`, report, "2026-01-01T00:00:00Z"); err != nil {
				t.Fatal(err)
			}

			result, err := ValidateConsoleClean(Scope{Context: context.Background(), Root: root, DB: db.Primary()})
			if err != nil {
				t.Fatal(err)
			}
			if result.Inspected != 1 {
				t.Fatalf("inspected = %d, want one story specimen; result: %+v", result.Inspected, result)
			}
			if test.wantClean {
				if len(result.Findings) != 0 || len(result.RunnerError) != 0 {
					t.Fatalf("clean capture produced findings: %+v", result)
				}
				return
			}
			if len(result.Findings) != 1 || result.Findings[0].Code != test.wantCode || result.Findings[0].AssetID != "controls.button" {
				t.Fatalf("findings = %+v, want %s for controls.button", result.Findings, test.wantCode)
			}
		})
	}
}

func writeQualityFixtureFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
