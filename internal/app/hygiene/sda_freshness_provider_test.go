package hygiene

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

type stubDependencyFreshnessRunner struct {
	report sdaFreshnessReport
	err    error
}

func (s stubDependencyFreshnessRunner) CheckDependencyFreshness(_ context.Context, _ string) (sdaFreshnessReport, error) {
	return s.report, s.err
}

func TestSDAFreshnessProviderMapsStaleSurfacesToHygieneFindings(t *testing.T) {
	root := t.TempDir()
	report := Report{Root: root, Success: true}
	provider := sdaFreshnessProvider{
		root: root,
		runner: stubDependencyFreshnessRunner{report: sdaFreshnessReport{
			Clean:   false,
			Root:    root,
			Mode:    "touched",
			Touched: []string{"packages/shared/shared.go"},
			Summary: sdaFreshnessSummary{Checked: 2, Stale: 1, Errors: 1},
			Surfaces: []sdaFreshnessSurface{
				{
					Scenario:  "demo",
					Surface:   "api",
					GoModPath: "scenarios/demo/api/go.mod",
					Status:    "stale",
					DiffPaths: []string{"scenarios/demo/api/go.mod", "scenarios/demo/api/go.sum"},
				},
				{
					Scenario:  "broken",
					Surface:   "cli",
					GoModPath: "scenarios/broken/cli/go.mod",
					Status:    "error",
					Error:     "go mod tidy failed",
				},
			},
			NextActions: []sdaFreshnessAction{{
				Code:    "reconcile_go_modules",
				Message: "Run tidy",
				Command: "scenario-dependency-analyzer deps reconcile --all-modules --apply",
			}, {
				Code:    "preview_missing_local_replaces",
				Message: "Preview replaces",
				Command: "scenario-dependency-analyzer deps reconcile --all --json",
			}},
		}},
	}

	if err := provider.Run(context.Background(), Request{}, &report); err != nil {
		t.Fatalf("Run: %v", err)
	}

	check := findCheck(t, report, dependencyFreshnessProviderID)
	if check.Passed || check.Severity != SeverityError {
		t.Fatalf("dependency freshness check = %+v, want failing error", check)
	}
	if report.SharedDrift == nil {
		t.Fatalf("expected shared_drift compatibility report")
	}
	if report.SharedDrift.Clean {
		t.Fatalf("compat shared_drift must be dirty")
	}
	if len(report.SharedDrift.Scenarios) != 2 {
		t.Fatalf("compat scenarios = %#v, want two surfaces", report.SharedDrift.Scenarios)
	}
	if got := report.SharedDrift.Scenarios[0].APIDir; got != "scenarios/demo/api" {
		t.Fatalf("compat APIDir = %q, want surface directory", got)
	}
	finding := findFinding(t, report, "dependency_freshness")
	if len(finding.Locations) == 0 || finding.Locations[0] != "scenarios/demo/api/go.mod" {
		t.Fatalf("finding locations = %#v, want stale diff path first", finding.Locations)
	}
	if len(report.Actions) != 2 {
		t.Fatalf("actions = %#v, want SDA apply and reconcile actions", report.Actions)
	}
	if report.Actions[0].Command != "scenario-dependency-analyzer deps reconcile --all-modules --apply" || report.Actions[0].Fixability != FixabilityAutomatic {
		t.Fatalf("first action = %#v, want automatic SDA apply action", report.Actions[0])
	}
	if report.Actions[1].Command != "scenario-dependency-analyzer deps reconcile --all --json" || report.Actions[1].Fixability != FixabilityGuided {
		t.Fatalf("second action = %#v, want guided reconcile preview", report.Actions[1])
	}
	if finding.Fixability != FixabilityGuided {
		t.Fatalf("finding fixability = %q, want guided when errors are present", finding.Fixability)
	}
}

func TestSDAFreshnessProviderMarksStaleOnlyReportAutomatic(t *testing.T) {
	root := t.TempDir()
	report := Report{Root: root, Success: true}
	provider := sdaFreshnessProvider{
		root: root,
		runner: stubDependencyFreshnessRunner{report: sdaFreshnessReport{
			Clean:   false,
			Root:    root,
			Mode:    "touched",
			Summary: sdaFreshnessSummary{Checked: 1, Stale: 1},
			Surfaces: []sdaFreshnessSurface{{
				Scenario:  "demo",
				Surface:   "api",
				GoModPath: "scenarios/demo/api/go.mod",
				Status:    "stale",
				DiffPaths: []string{"scenarios/demo/api/go.mod"},
			}},
			NextActions: []sdaFreshnessAction{{
				Code:    "reconcile_go_modules",
				Message: "Run tidy",
				Command: "scenario-dependency-analyzer deps reconcile --all-modules --apply",
			}},
		}},
	}

	if err := provider.Run(context.Background(), Request{}, &report); err != nil {
		t.Fatalf("Run: %v", err)
	}

	finding := findFinding(t, report, "dependency_freshness")
	if finding.Fixability != FixabilityAutomatic {
		t.Fatalf("finding fixability = %q, want automatic", finding.Fixability)
	}
	if len(finding.NextActions) != 1 || finding.NextActions[0].Fixability != FixabilityAutomatic {
		t.Fatalf("next actions = %#v, want automatic tidy action", finding.NextActions)
	}
}

func TestSDAFreshnessProviderKeepsColdCacheUnqualified(t *testing.T) {
	root := t.TempDir()
	report := Report{Root: root, Success: true}
	provider := sdaFreshnessProvider{
		root: root,
		runner: stubDependencyFreshnessRunner{report: sdaFreshnessReport{
			Clean: false,
			Root:  root,
			Mode:  "touched",
			Summary: sdaFreshnessSummary{
				Checked: 1, NeedsDownload: 1,
			},
			Surfaces: []sdaFreshnessSurface{{
				Scenario: "demo", Surface: "api", GoModPath: "scenarios/demo/api/go.mod", Status: "needs_download",
			}},
		}},
	}
	if err := provider.Run(context.Background(), Request{}, &report); err != nil {
		t.Fatalf("Run: %v", err)
	}
	check := findCheck(t, report, dependencyFreshnessProviderID)
	if check.Passed || check.Severity != SeverityError {
		t.Fatalf("dependency freshness check = %+v, want blocking incomplete coverage", check)
	}
	if report.SharedDrift == nil || report.SharedDrift.Clean {
		t.Fatalf("shared drift = %#v, want cold-cache validation to remain unqualified", report.SharedDrift)
	}
	if len(report.Findings) != 1 || report.Findings[0].Code != "dependency_freshness" {
		t.Fatalf("findings = %#v, want one dependency-freshness blocker", report.Findings)
	}
}

func TestSDAFreshnessProviderReportsUnavailableWithoutInterpretingDrift(t *testing.T) {
	root := t.TempDir()
	report := Report{Root: root, Success: true}
	provider := sdaFreshnessProvider{
		root:   root,
		runner: stubDependencyFreshnessRunner{err: errors.New("executable not found")},
	}

	if err := provider.Run(context.Background(), Request{}, &report); err != nil {
		t.Fatalf("Run: %v", err)
	}

	check := findCheck(t, report, dependencyFreshnessProviderID)
	if check.Passed || check.Severity != SeverityError {
		t.Fatalf("dependency freshness check = %+v, want provider error", check)
	}
	if report.SharedDrift != nil {
		t.Fatalf("unavailable provider must not fabricate shared_drift: %#v", report.SharedDrift)
	}
	finding := findFinding(t, report, "dependency_freshness_provider")
	if finding.NextActions[0].Command != "scenario-dependency-analyzer freshness --touched --json" {
		t.Fatalf("next action = %#v, want SDA inspection command", finding.NextActions)
	}
}

func TestParseSDAFreshnessReportRequiresMatchingCompleteTouchedCoverage(t *testing.T) {
	tests := []struct {
		name string
		json string
		want bool
	}{
		{
			name: "complete matching root",
			json: `{"root":"/repo","mode":"touched","clean":false,"summary":{"checked":2,"clean":1,"stale":1,"errors":0,"needs_download":0,"skipped":0},"surfaces":[{"scenario":"demo","surface":"api","go_mod_path":"scenarios/demo/api/go.mod","status":"clean"},{"scenario":"demo","surface":"cli","go_mod_path":"scenarios/demo/cli/go.mod","status":"stale"}]}`,
			want: true,
		},
		{
			name: "complete cold-cache report",
			json: `{"root":"/repo","mode":"touched","clean":false,"summary":{"checked":1,"clean":0,"stale":0,"errors":0,"needs_download":1,"skipped":0},"surfaces":[{"scenario":"demo","surface":"api","go_mod_path":"scenarios/demo/api/go.mod","status":"needs_download"}]}`,
			want: true,
		},
		{
			name: "different checkout",
			json: `{"root":"/other","mode":"touched","clean":true,"summary":{"checked":0,"clean":0},"surfaces":[]}`,
		},
		{
			name: "missing root identity",
			json: `{"mode":"touched","clean":true,"summary":{"checked":0}}`,
		},
		{
			name: "wrong coverage mode",
			json: `{"root":"/repo","mode":"all","clean":true,"summary":{"checked":0},"surfaces":[]}`,
		},
		{
			name: "checked surface omitted from evidence",
			json: `{"root":"/repo","mode":"touched","clean":true,"summary":{"checked":1,"clean":1},"surfaces":[]}`,
		},
		{
			name: "clean contradicts stale count",
			json: `{"root":"/repo","mode":"touched","clean":true,"summary":{"checked":1,"clean":0,"stale":1},"surfaces":[{"scenario":"demo","surface":"api","go_mod_path":"scenarios/demo/api/go.mod","status":"stale"}]}`,
		},
		{
			name: "dirty without a blocker",
			json: `{"root":"/repo","mode":"touched","clean":false,"summary":{"checked":1,"clean":1},"surfaces":[{"scenario":"demo","surface":"api","go_mod_path":"scenarios/demo/api/go.mod","status":"clean"}]}`,
		},
		{
			name: "unknown surface status",
			json: `{"root":"/repo","mode":"touched","clean":false,"summary":{"checked":1,"skipped":1},"surfaces":[{"scenario":"demo","surface":"api","go_mod_path":"scenarios/demo/api/go.mod","status":"future"}]}`,
		},
		{
			name: "surface statuses contradict summary",
			json: `{"root":"/repo","mode":"touched","clean":true,"summary":{"checked":1,"clean":1},"surfaces":[{"scenario":"demo","surface":"api","go_mod_path":"scenarios/demo/api/go.mod","status":"stale"}]}`,
		},
		{
			name: "surface identity missing",
			json: `{"root":"/repo","mode":"touched","clean":true,"summary":{"checked":1,"clean":1},"surfaces":[{"status":"clean"}]}`,
		},
		{
			name: "surface identity duplicated",
			json: `{"root":"/repo","mode":"touched","clean":true,"summary":{"checked":2,"clean":2},"surfaces":[{"scenario":"demo","surface":"api","go_mod_path":"scenarios/demo/api/go.mod","status":"clean"},{"scenario":"demo","surface":"api","go_mod_path":"scenarios/demo/api/go.mod","status":"clean"}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSDAFreshnessReport("/repo", []byte(tt.json))
			if tt.want && err != nil {
				t.Fatalf("parse: %v", err)
			}
			if !tt.want && err == nil {
				t.Fatalf("parse accepted malformed report: %+v", got)
			}
		})
	}
}

func TestFreshnessLocationsDeduplicatesAndTruncates(t *testing.T) {
	report := sdaFreshnessReport{Surfaces: []sdaFreshnessSurface{
		{Status: "stale", DiffPaths: []string{"a/go.mod", "a/go.mod"}},
		{Status: "stale", DiffPaths: []string{"b/go.mod"}},
		{Status: "error", GoModPath: "c/go.mod"},
		{Status: "error", GoModPath: "d/go.mod"},
		{Status: "error", GoModPath: "e/go.mod"},
		{Status: "error", GoModPath: "f/go.mod"},
	}}
	got := freshnessLocations(report)
	if len(got) != 6 {
		t.Fatalf("locations = %#v, want 5 plus truncation note", got)
	}
	if got[0] != "a/go.mod" || got[5] != "... 1 more (see dependency_freshness summary)" {
		t.Fatalf("locations = %#v, want dedupe and truncation", got)
	}
}

func TestSurfaceDirUsesSlashPaths(t *testing.T) {
	got := surfaceDir(filepath.Join("scenarios", "demo", "cli", "go.mod"))
	if got != "scenarios/demo/cli" {
		t.Fatalf("surfaceDir = %q, want slash path", got)
	}
}

func findCheck(t *testing.T, report Report, name string) Check {
	t.Helper()
	for _, check := range report.Checks {
		if check.Name == name {
			return check
		}
	}
	t.Fatalf("missing check %q in %#v", name, report.Checks)
	return Check{}
}

func findFinding(t *testing.T, report Report, code string) Finding {
	t.Helper()
	for _, finding := range report.Findings {
		if finding.Code == code {
			return finding
		}
	}
	t.Fatalf("missing finding %q in %#v", code, report.Findings)
	return Finding{}
}
