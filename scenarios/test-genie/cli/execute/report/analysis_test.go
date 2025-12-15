package report

import (
	"os"
	"path/filepath"
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
