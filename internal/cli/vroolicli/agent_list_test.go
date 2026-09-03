package vroolicli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// [REQ:STORM-002] vrooli agent list shows every live session with its tree,
// scope, pid, age, claims and frozen state.
func TestAgentListRendersTreeScopePid(t *testing.T) {
	now := time.Date(2026, 9, 2, 16, 0, 0, 0, time.UTC)
	leases := []scenarioruntime.EditorLease{
		{SessionID: "run-1", Harness: "claude", Agent: "claude", PID: 4242, WorkingDir: "/home/op/Vrooli", Scope: "cgroup:/user.slice/vrooli-agents.slice/vrooli-agent-run-1.scope", Claims: []string{"/home/op/Vrooli/internal"}, CreatedAt: now.Add(-90 * time.Second), LastHeartbeatAt: now},
		{SessionID: "run-2", Harness: "codex", Agent: "codex", PID: 4343, WorkingDir: "/home/op/other", Scope: "none", CreatedAt: now.Add(-time.Hour), LastHeartbeatAt: now},
	}
	var out bytes.Buffer
	frozen := func(scope string) string {
		if strings.Contains(scope, "run-1") {
			return "yes"
		}
		return "n/a"
	}
	if err := renderAgentList(&out, func() ([]scenarioruntime.EditorLease, error) { return leases, nil }, frozen, now, false); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	for _, want := range []string{"run-1", "/home/op/Vrooli", "vrooli-agent-run-1.scope", "4242", "1m30s", "/home/op/Vrooli/internal", "yes", "run-2", "1h0m0s", "n/a"} {
		if !strings.Contains(text, want) {
			t.Fatalf("list lacks %q:\n%s", want, text)
		}
	}
	out.Reset()
	if err := renderAgentList(&out, func() ([]scenarioruntime.EditorLease, error) { return nil, nil }, frozen, now, false); err != nil || !strings.Contains(out.String(), "No live agent sessions") {
		t.Fatalf("empty list = %q, %v", out.String(), err)
	}
	out.Reset()
	if err := renderAgentList(&out, func() ([]scenarioruntime.EditorLease, error) { return leases, nil }, frozen, now, true); err != nil || !strings.Contains(out.String(), `"sessions"`) {
		t.Fatalf("json list = %q, %v", out.String(), err)
	}
	if got := absoluteClaims([]string{"internal/setpoint", "/abs/path", " "}, "/home/op/Vrooli"); len(got) != 2 || got[0] != "/home/op/Vrooli/internal/setpoint" || got[1] != "/abs/path" {
		t.Fatalf("absoluteClaims = %v", got)
	}
}
