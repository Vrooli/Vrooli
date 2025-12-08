package report

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	execTypes "test-genie/cli/internal/execute"
	"test-genie/cli/internal/repo"
)

func TestPrinterIncludesGuidesAndInsights(t *testing.T) {
	tmp := t.TempDir()

	unitLog := filepath.Join(tmp, "unit.log")
	if err := os.WriteFile(unitLog, []byte("❌ test failed: sample\n"), 0o644); err != nil {
		t.Fatalf("write unit log: %v", err)
	}
	integrationLog := filepath.Join(tmp, "integration.log")
	if err := os.WriteFile(integrationLog, []byte("UI bundle is stale and must be rebuilt\n"), 0o644); err != nil {
		t.Fatalf("write integration log: %v", err)
	}

	resp := execTypes.Response{
		Success:     true,
		StartedAt:   time.Now().UTC().Format(time.RFC3339),
		CompletedAt: time.Now().Add(30 * time.Second).UTC().Format(time.RFC3339),
		PhaseSummary: execTypes.PhaseSummary{
			Total:           2,
			Failed:          2,
			DurationSeconds: 30,
		},
		Phases: []execTypes.Phase{
			{Name: "unit", Status: "failed", DurationSeconds: 8, LogPath: unitLog},
			{Name: "integration", Status: "failed", DurationSeconds: 12, LogPath: integrationLog},
		},
	}

	var buf bytes.Buffer
	pr := New(&buf, "demo-scenario", "", nil, nil, false, nil)
	pr.Print(resp)

	out := buf.String()
	for _, token := range []string{
		"QUICK FIX GUIDE:",
		"PHASE-SPECIFIC DEBUG GUIDES:",
		"UI bundle",
		"artifact roots:",
	} {
		if !strings.Contains(out, token) {
			t.Fatalf("expected output to contain %q\n----\n%s\n----", token, out)
		}
	}
}

func TestPrintResultsCondensedReplay(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "structure.log")
	if err := os.WriteFile(logPath, []byte("ERROR: lighthouse.json missing property 'version'\n"), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	resp := execTypes.Response{
		PhaseSummary: execTypes.PhaseSummary{
			Total:           1,
			Failed:          1,
			DurationSeconds: 2,
		},
		Phases: []execTypes.Phase{
			{
				Name:            "structure",
				Status:          "failed",
				DurationSeconds: 2,
				LogPath:         logPath,
				Remediation:     "Add the missing version to .vrooli/lighthouse.json",
			},
		},
	}

	var buf bytes.Buffer
	pr := New(&buf, "demo", "", nil, nil, false, nil)
	pr.SetStreamedObservations(true)
	pr.PrintResults(resp)

	out := buf.String()
	if strings.Contains(out, "[PHASE_START") {
		t.Fatalf("expected condensed replay without phase banners, got:\n%s", out)
	}
	if !strings.Contains(out, "structure") || !strings.Contains(out, "ERROR: lighthouse.json missing property 'version'") {
		t.Fatalf("expected headline from log to be included:\n%s", out)
	}
	if !strings.Contains(out, "log: "+logPath) {
		t.Fatalf("expected log path to be included:\n%s", out)
	}
	if !strings.Contains(out, "fix: Add the missing version") {
		t.Fatalf("expected remediation hint to be included:\n%s", out)
	}
	expectedDoc := repo.AbsPath("scenarios/test-genie/docs/phases/structure/README.md")
	if !strings.Contains(out, expectedDoc) {
		t.Fatalf("expected structure doc hint to be included:\n%s", out)
	}
}

func TestPrintPrePlanShowsDocs(t *testing.T) {
	var buf bytes.Buffer
	pr := New(&buf, "demo", "", nil, nil, false, nil)
	pr.PrintPreExecution([]string{"lint"})

	out := buf.String()
	expectedDoc := repo.AbsPath("scenarios/test-genie/docs/phases/lint/README.md")
	if !strings.Contains(out, "docs: "+expectedDoc) {
		t.Fatalf("expected lint doc link in plan, got:\n%s", out)
	}
}

func TestPassedPhaseWarningsAreSurfaced(t *testing.T) {
	resp := execTypes.Response{
		Success:     false,
		StartedAt:   time.Now().UTC().Format(time.RFC3339),
		CompletedAt: time.Now().Add(2 * time.Second).UTC().Format(time.RFC3339),
		PhaseSummary: execTypes.PhaseSummary{
			Total:           1,
			Passed:          1,
			Failed:          0,
			DurationSeconds: 2,
		},
		Phases: []execTypes.Phase{
			{
				Name:            "dependencies",
				Status:          "passed",
				DurationSeconds: 2,
				Observations: execTypes.ObservationList{
					{Prefix: "WARNING", Text: "resource telemetry unavailable"},
					{Prefix: "SUCCESS", Text: "command available: curl"},
				},
			},
		},
	}

	var buf bytes.Buffer
	pr := New(&buf, "demo", "", nil, nil, false, nil)
	pr.Print(resp)

	out := buf.String()
	if !strings.Contains(out, "warnings: resource telemetry unavailable") {
		t.Fatalf("expected warning details to be surfaced for passed phase:\n%s", out)
	}
}
