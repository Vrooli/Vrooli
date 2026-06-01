package phases

import (
	"os"
	"path/filepath"
	"testing"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

func TestParseGoCoverProfile(t *testing.T) {
	cases := []struct {
		name             string
		content          string
		wantCov, wantTot int
	}{
		{
			name: "mixed covered/uncovered",
			content: "mode: atomic\n" +
				"api/x.go:1.1,3.2 4 1\n" + // 4 covered
				"api/x.go:5.1,6.2 2 0\n" + // 2 uncovered
				"api/y.go:1.1,2.2 4 3\n", // 4 covered
			wantCov: 8, wantTot: 10,
		},
		{
			name:    "empty profile (mode only)",
			content: "mode: set\n",
			wantCov: 0, wantTot: 0,
		},
		{
			name:    "malformed lines skipped",
			content: "mode: count\nnot a real line\napi/z.go:1.1,2.2 3 1\n",
			wantCov: 3, wantTot: 3,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cov, tot := parseGoCoverProfile(tc.content)
			if cov != tc.wantCov || tot != tc.wantTot {
				t.Fatalf("parseGoCoverProfile = (%d,%d), want (%d,%d)", cov, tot, tc.wantCov, tc.wantTot)
			}
		})
	}
}

func TestParseLCOV(t *testing.T) {
	content := "TN:\nSF:src/a.ts\nLF:10\nLH:7\nend_of_record\n" +
		"SF:src/b.ts\nLF:20\nLH:18\nend_of_record\n"
	hit, found := parseLCOV(content)
	if hit != 25 || found != 30 {
		t.Fatalf("parseLCOV = (%d,%d), want (25,30)", hit, found)
	}
}

func TestCoverageFindingsSeverityBands(t *testing.T) {
	targets := []coverageTarget{
		{Name: "go:low", Path: "coverage/go/low.out", Percent: 40},      // error
		{Name: "go:mid", Path: "coverage/go/mid.out", Percent: 60},      // warning
		{Name: "node:high", Path: "coverage/ui/lcov.info", Percent: 85}, // no finding
	}
	findings := coverageFindings("demo", targets)
	if len(findings) != 2 {
		t.Fatalf("want 2 findings (low+mid), got %d", len(findings))
	}
	bySeverity := map[architecturev1.FindingSeverity]int{}
	for _, f := range findings {
		if f.Source != architecturev1.FindingSource_FINDING_SOURCE_COVERAGE {
			t.Errorf("finding %q source = %v, want COVERAGE", f.Code, f.Source)
		}
		bySeverity[f.Severity]++
	}
	if bySeverity[architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR] != 1 {
		t.Errorf("want 1 error finding, got %d", bySeverity[architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR])
	}
	if bySeverity[architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING] != 1 {
		t.Errorf("want 1 warning finding, got %d", bySeverity[architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING])
	}
}

func TestCollectCoverageTargets(t *testing.T) {
	dir := t.TempDir()
	goDir := filepath.Join(dir, "coverage", "go")
	if err := os.MkdirAll(goDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(goDir, "go-api.coverage.out"),
		[]byte("mode: atomic\napi/x.go:1.1,2.2 10 1\napi/x.go:3.1,4.2 10 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	uiDir := filepath.Join(dir, "coverage", "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "lcov.info"),
		[]byte("SF:ui/a.ts\nLF:100\nLH:90\nend_of_record\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	targets := collectCoverageTargets(dir)
	if len(targets) != 2 {
		t.Fatalf("want 2 targets, got %d: %+v", len(targets), targets)
	}
	got := map[string]float64{}
	for _, tgt := range targets {
		got[tgt.Name] = tgt.Percent
	}
	if got["go:api"] != 50 {
		t.Errorf("go:api = %.1f, want 50", got["go:api"])
	}
	if got["node:ui"] != 90 {
		t.Errorf("node:ui = %.1f, want 90", got["node:ui"])
	}
}

func TestCollectCoverageTargetsEmpty(t *testing.T) {
	if got := collectCoverageTargets(t.TempDir()); len(got) != 0 {
		t.Fatalf("want no targets for empty scenario, got %d", len(got))
	}
}
