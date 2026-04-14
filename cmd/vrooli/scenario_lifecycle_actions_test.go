package main

import (
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/orchestrator"
)

func TestParseScenarioRequirementsRequestTreatsHelpAsCommandHelp(t *testing.T) {
	_, err := scenariocli.ParseRequirementsRequest([]string{"--help"})
	if err == nil {
		t.Fatal("expected help-only error")
	}
	if !strings.Contains(err.Error(), scenariocli.RequirementsHelpText()) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseScenarioHealFromSandboxRequestUsesEnvDefault(t *testing.T) {
	t.Setenv("SANDBOX_MERGED_DIR", "/merged")

	req, err := scenariocli.ParseHealFromSandboxRequest(strings.TrimSpace("/merged"), nil)
	if err != nil {
		t.Fatalf("parseScenarioHealFromSandboxRequest: %v", err)
	}
	if req.MergedPath != "/merged" || req.DryRun {
		t.Fatalf("request = %+v", req)
	}
}

func TestFindSandboxAffectedScenariosReturnsSortedMatches(t *testing.T) {
	home := t.TempDir()
	startedAt := time.Now().Add(-1 * time.Minute)
	writeScenarioProcessRecordWithWorkingDir(t, home, "beta", "start-api", 1234, 18081, startedAt, "/merged/scenarios/beta")
	writeScenarioProcessRecordWithWorkingDir(t, home, "alpha", "start-api", 1235, 18082, startedAt, "/merged/scenarios/alpha")

	affected, err := orchestrator.SandboxAffectedScenarios(home, "/merged")
	if err != nil {
		t.Fatalf("findSandboxAffectedScenarios: %v", err)
	}
	if got := strings.Join(affected, ","); got != "alpha,beta" {
		t.Fatalf("affected = %q", got)
	}
}
