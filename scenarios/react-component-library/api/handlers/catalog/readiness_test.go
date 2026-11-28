package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog"
	"react-component-library/internal/catalogcoverage"
)

func TestReadinessRunReadsLatestManifestAndFindings(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_STORAGE_ROOT", root)
	dir := filepath.Join(root, "state", "vrooli", "react-component-library", "gates", "latest")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{"run_id":"run-7","started_at":"2026-08-26T12:00:00Z","completed_at":"2026-08-26T12:01:00Z"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "findings.json"), []byte(`{"runId":"run-7","completedAt":"2026-08-26T12:01:00Z"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	run := readinessRun(root)
	if run.GetRunId() != "run-7" || run.GetStartedAt() != "2026-08-26T12:00:00Z" || !run.GetCompleted() {
		t.Fatalf("readiness run = %+v", run)
	}
	if run.GetEvidenceAge() == "missing" || run.GetEvidenceAge() == "unknown" {
		t.Fatalf("evidence age was not derived: %q", run.GetEvidenceAge())
	}
}

func TestReadinessRunPrefersManifestIdentityOverStaleFindings(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_STORAGE_ROOT", root)
	dir := filepath.Join(root, "state", "vrooli", "react-component-library", "gates", "latest")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{"run_id":"run-new","started_at":"2026-08-26T12:00:00Z","completed_at":"2026-08-26T12:01:00Z"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "findings.json"), []byte(`{"runId":"run-old","completedAt":"2026-08-25T12:01:00Z"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	run := readinessRun(root)
	if run.GetRunId() != "run-new" || run.GetCompletedAt() != "2026-08-26T12:01:00Z" || !run.GetCompleted() {
		t.Fatalf("readiness run = %+v, want manifest identity and completion", run)
	}
}

func TestReadinessRunUsesIndexWhenManifestOmitsStartTime(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_STORAGE_ROOT", root)
	dir := filepath.Join(root, "state", "vrooli", "react-component-library", "gates", "latest")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{"run_id":"run-current","completed_at":"2026-08-26T12:01:00Z"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	index := `[{"run_id":"run-previous","started_at":"2026-08-25T12:00:00Z"},{"run_id":"run-current","started_at":"2026-08-26T11:55:00Z"}]`
	if err := os.WriteFile(filepath.Join(root, "state", "vrooli", "react-component-library", "gates", "runs.index.json"), []byte(index), 0o644); err != nil {
		t.Fatal(err)
	}

	run := readinessRun(root)
	if run.GetRunId() != "run-current" || run.GetStartedAt() != "2026-08-26T11:55:00Z" {
		t.Fatalf("readiness run = %+v, want matching index start time", run)
	}
}

func TestReadinessUsesMaturityAndBlastRadius(t *testing.T) {
	report := &catalogcoverage.Report{
		Rows: []catalogcoverage.Row{
			{AssetID: "low", Name: "Low", FailedGates: []string{"a"}, BlocksDownstream: 1, Weight: 1},
			{AssetID: "high", Name: "High", FailedGates: []string{"b"}, BlocksDownstream: 4, Weight: 2},
		},
		Maturity: catalogcoverage.MaturityCoverage{
			CatalogCompletion:       catalogcoverage.CoverageMetric{Numerator: 1, Denominator: 1, Ratio: 1},
			MandatoryGateCoverage:   catalogcoverage.CoverageMetric{Numerator: 1, Denominator: 1, Ratio: 1},
			ProductionReadyCoverage: catalogcoverage.CoverageMetric{Numerator: 0, Denominator: 1, Ratio: 0},
		},
	}
	if got := achievedRung(report); got != "verified" {
		t.Fatalf("achieved rung = %q, want verified", got)
	}
	rows, _ := readinessTriageWithOmitted(report)
	if len(rows) != 2 || rows[0].GetAssetId() != "high" {
		t.Fatalf("triage ordering = %+v", rows)
	}
	if got := rungGap("production-ready", "verified"); got != 1 {
		t.Fatalf("rung gap = %d, want 1", got)
	}
}

func TestReadinessTriageReportsOmittedRows(t *testing.T) {
	report := &catalogcoverage.Report{Rows: make([]catalogcoverage.Row, 0, 51)}
	for i := 0; i < 51; i++ {
		report.Rows = append(report.Rows, catalogcoverage.Row{AssetID: fmt.Sprintf("asset-%02d", i), Name: "Asset", FailedGates: []string{"gate"}})
	}
	rows, omitted := readinessTriageWithOmitted(report)
	if len(rows) != 50 || omitted != 1 {
		t.Fatalf("triage rows=%d omitted=%d, want 50 and 1", len(rows), omitted)
	}
}

func TestReadinessNextStepsSeparateInspectionFromEvidenceWrite(t *testing.T) {
	steps := readinessNextSteps([]*catalogv1.ReadinessTriageRow{{Gate: "utility-class"}})
	if len(steps) != 3 {
		t.Fatalf("next steps = %v, want inspect, persist, then recheck", steps)
	}
	for _, want := range []string{"catalog gates utility-class", "without writing", "catalog record-gate utility-class", "Persist a measurement", "catalog readiness"} {
		found := false
		for _, step := range steps {
			if strings.Contains(step, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("next steps %v do not include %q", steps, want)
		}
	}
}
