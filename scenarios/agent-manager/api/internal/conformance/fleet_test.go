package conformance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRepositoryCodingAgentFleetConforms(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	report, err := FleetConforms(repoRoot)
	if err != nil {
		t.Fatalf("fleet conformance failed: %v (%#v)", err, report.Findings)
	}
	if len(report.Entries) != 6 {
		t.Fatalf("entries = %d, want five coding agents plus install-only Ollama: %#v", len(report.Entries), report.Entries)
	}
}

func TestFleetConformanceFailsWhenRunnerResourceDeclarationIsRemoved(t *testing.T) {
	repoRoot := t.TempDir()
	for _, runner := range []struct {
		resource string
		runner   string
		codec    string
		signal   string
	}{
		{"claude-code", "claude-code", "claude", "CLAUDECODE=1"},
		{"codex", "codex", "codex", "CODEX_CI=1 AND CODEX_THREAD_ID non-empty"},
		{"grok", "grok", "grok", "GROK_AGENT=1"},
		{"opencode", "opencode", "opencode", "OPENCODE_PID matches a calling-process ancestor"},
		// Antigravity is deliberately absent: the gate must report the missing
		// resource instead of allowing the runner registry to drift silently.
	} {
		writeFleetManifest(t, repoRoot, runner.resource, map[string]any{
			"runner_type":      runner.runner,
			"codec":            runner.codec,
			"detection_signal": runner.signal,
			"hook_surface":     "native-permissions",
		})
	}
	report, err := ValidateFleet(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	if !hasFleetFinding(report, "agent_fleet.resource_missing") {
		t.Fatalf("findings = %#v, want missing antigravity resource finding", report.Findings)
	}
}

func writeFleetManifest(t *testing.T, root, resource string, execution map[string]any) {
	t.Helper()
	path := filepath.Join(root, "resources", resource, "resource.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	payload := map[string]any{"name": resource, "execution": execution}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func hasFleetFinding(report FleetReport, code string) bool {
	for _, finding := range report.Findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}
