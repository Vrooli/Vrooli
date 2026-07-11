package orchestration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// [REQ:REQ-P0-001] Scenario-owned portable profiles must use one strict,
// enabled Agent Manager dependency contract before reconciliation can mutate.
func TestReadScenarioProfileConfigRejectsUnknownAndDuplicateSources(t *testing.T) {
	write := func(t *testing.T, body string) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "service.json")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	for _, tc := range []struct{ name, body, want string }{
		{"unknown field", `{"dependencies":{"scenarios":{"agent-manager":{"config":{"profiles":{"reconcile":true,"mode":"update_if_unmodified","sources":[".vrooli/agent-profiles/default.json"],"unknown":true}}}}}}`, "failed to parse profile config"},
		{"duplicate source", `{"dependencies":{"scenarios":{"agent-manager":{"config":{"profiles":{"reconcile":true,"mode":"update_if_unmodified","sources":[".vrooli/agent-profiles/default.json",".vrooli/agent-profiles/default.json"]}}}}}}`, "duplicate profile source"},
		{"disabled dependency", `{"dependencies":{"scenarios":{"agent-manager":{"enabled":false}}}}`, "dependency must be enabled"},
		{"missing profiles contract", `{"dependencies":{"scenarios":{"agent-manager":{"config":{}}}}}`, "profiles"},
		{"missing reconcile mode", `{"dependencies":{"scenarios":{"agent-manager":{"config":{"profiles":{"mode":"update_if_unmodified","sources":[".vrooli/agent-profiles/default.json"]}}}}}}`, "profiles.reconcile"},
		{"missing sources", `{"dependencies":{"scenarios":{"agent-manager":{"config":{"profiles":{"reconcile":true,"mode":"update_if_unmodified","sources":[]}}}}}}`, "must declare at least one source"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := readScenarioProfileConfig(write(t, tc.body))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
		})
	}
}
