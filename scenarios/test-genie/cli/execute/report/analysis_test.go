package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	execTypes "test-genie/cli/internal/execute"
)

func TestAnalyzePhaseFailuresDoesNotMisclassifyTimeoutParameterAsPhaseTimeout(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "standards.log")
	if err := os.WriteFile(logPath, []byte("running standards audit via scenario-auditor (timeout=60s, fail_on=high)\n"), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	insights := AnalyzePhaseFailures([]execTypes.Phase{
		{
			Name:           "standards",
			Status:         "failed",
			LogPath:        logPath,
			Error:          "standards violations exceed fail_on=high (highest=critical)",
			Classification: "misconfiguration",
		},
	})
	if len(insights) != 1 {
		t.Fatalf("expected 1 insight, got %d", len(insights))
	}
	if insights[0].Cause == "Phase timeout" {
		t.Fatalf("expected non-timeout cause, got %q", insights[0].Cause)
	}
}

func TestAnalyzePhaseFailuresDoesNotLeakJSONFromStandardsLog(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "standards.log")
	logContent := strings.Join([]string{
		"running standards audit via scenario-auditor (timeout=60s, fail_on=high)",
		`{"security":null,"standards":{"summary":{"total":1,"highest_severity":"high","top_violations":[{"severity":"high","title":"Example","file_path":"PRD.md","line_number":1,"recommendation":"❌ example"}]}}}`,
	}, "\n")
	if err := os.WriteFile(logPath, []byte(logContent), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	insights := AnalyzePhaseFailures([]execTypes.Phase{
		{
			Name:        "standards",
			Status:      "failed",
			LogPath:     logPath,
			Error:       "standards violations exceed fail_on=high (highest=high)",
			Remediation: "Run `scenario-auditor audit demo --standards-only` and fix findings.",
		},
	})
	if len(insights) != 1 {
		t.Fatalf("expected 1 insight, got %d", len(insights))
	}
	if strings.Contains(insights[0].Detail, `{"security"`) {
		t.Fatalf("expected JSON to be omitted from insight detail, got %q", insights[0].Detail)
	}
	if insights[0].Cause == "" || !strings.Contains(insights[0].Cause, "standards violations") {
		t.Fatalf("expected standards error to be used as cause, got %q", insights[0].Cause)
	}
	if len(insights[0].Fixes) == 0 {
		t.Fatalf("expected remediation fix to be included")
	}
}
