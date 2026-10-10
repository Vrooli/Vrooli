package vroolicli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/agentscope"
	"github.com/vrooli/vrooli/internal/agentsession"
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
	if err := renderAgentList(context.Background(), &out, func() ([]scenarioruntime.EditorLease, error) { return leases, nil }, agentsession.Reconciler{Census: emptyCensus, Alive: alwaysAlive, Frozen: frozen, Now: fixedClock(now)}, false); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	for _, want := range []string{"run-1", "/home/op/Vrooli", "vrooli-agent-run-1.scope", "4242", "1m30s", "/home/op/Vrooli/internal", "yes", "run-2", "1h0m0s", "n/a"} {
		if !strings.Contains(text, want) {
			t.Fatalf("list lacks %q:\n%s", want, text)
		}
	}
	out.Reset()
	if err := renderAgentList(context.Background(), &out, func() ([]scenarioruntime.EditorLease, error) { return nil, nil }, agentsession.Reconciler{Census: emptyCensus, Alive: alwaysAlive, Frozen: frozen, Now: fixedClock(now)}, false); err != nil || !strings.Contains(out.String(), "No live agent sessions") {
		t.Fatalf("empty list = %q, %v", out.String(), err)
	}
	out.Reset()
	if err := renderAgentList(context.Background(), &out, func() ([]scenarioruntime.EditorLease, error) { return leases, nil }, agentsession.Reconciler{Census: emptyCensus, Alive: alwaysAlive, Frozen: frozen, Now: fixedClock(now)}, true); err != nil || !strings.Contains(out.String(), `"sessions"`) {
		t.Fatalf("json list = %q, %v", out.String(), err)
	}
	if got := absoluteClaims([]string{"internal/setpoint", "/abs/path", " "}, "/home/op/Vrooli"); len(got) != 2 || got[0] != "/home/op/Vrooli/internal/setpoint" || got[1] != "/abs/path" {
		t.Fatalf("absoluteClaims = %v", got)
	}
}

// emptyCensus is a kernel holding no scopes; alwaysAlive is a host on which
// every lease's process is running. Together they isolate the rendering from
// the reconciliation the dedicated tests below exercise.
func emptyCensus() ([]agentscope.Entry, error) { return nil, nil }

// frozenUnknown is a host whose freeze state is not being exercised.
func frozenUnknown(string) string { return "n/a" }

func alwaysAlive(int) bool { return true }

// fixedClock pins the reconciliation's clock to a case's own instant.
func fixedClock(at time.Time) func() time.Time { return func() time.Time { return at } }

// A lease whose process is gone is a ghost: a row waiting for the registry's
// next sweep, not a session. On 2026-09-04 735 of 773 "active" leases were
// ghosts and this command presented every one of them as a live session.
func TestAgentListMarksALeaseWithADeadProcessAsAGhost(t *testing.T) {
	now := time.Now().UTC()
	leases := []scenarioruntime.EditorLease{{
		SessionID: "codex-1", Harness: "codex", Agent: "codex", PID: 4242,
		Scope: "vrooli-agent-codex-1", CreatedAt: now.Add(-time.Minute),
	}}
	var out bytes.Buffer
	dead := func(int) bool { return false }
	if err := renderAgentList(context.Background(), &out, func() ([]scenarioruntime.EditorLease, error) { return leases, nil }, agentsession.Reconciler{Census: emptyCensus, Alive: dead, Frozen: frozenUnknown, Now: fixedClock(now)}, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "ghost") {
		t.Fatalf("a lease with a dead process must be marked a ghost:\n%s", out.String())
	}
}

// A scope the kernel holds that no lease names must be shown, not hidden.
// Seven such scopes were holding the scenario fleet and this command listed
// none of them.
func TestAgentListShowsScopesNoLeaseNames(t *testing.T) {
	now := time.Now().UTC()
	census := func() ([]agentscope.Entry, error) {
		return []agentscope.Entry{
			{Name: "vrooli-agent-codex-eb0c.scope", Tasks: 787, Agentless: true},
			{Name: "vrooli-agent-grok-x.scope", Tasks: 3},
		}, nil
	}
	var out bytes.Buffer
	if err := renderAgentList(context.Background(), &out, func() ([]scenarioruntime.EditorLease, error) { return nil, nil }, agentsession.Reconciler{Census: census, Alive: alwaysAlive, Frozen: frozenUnknown, Now: fixedClock(now)}, false); err != nil {
		t.Fatal(err)
	}
	rendered := out.String()
	if !strings.Contains(rendered, "vrooli-agent-codex-eb0c.scope") || !strings.Contains(rendered, "agentless") {
		t.Fatalf("a scope whose agent has exited must be listed as agentless:\n%s", rendered)
	}
	if !strings.Contains(rendered, "unrecorded") {
		t.Fatalf("a scope with no lease must be listed as unrecorded:\n%s", rendered)
	}
}

// A lease naming a scope the kernel does not hold is a phantom: three such
// rows were shown as live sessions.
func TestAgentListMarksALeaseNamingNoLiveScope(t *testing.T) {
	now := time.Now().UTC()
	leases := []scenarioruntime.EditorLease{{
		SessionID: "claude-code-b4b", Harness: "claude-code", Agent: "claude", PID: 916011,
		Scope: "vrooli-agent-claude-code-3fb16182", CreatedAt: now.Add(-time.Minute),
	}}
	var out bytes.Buffer
	if err := renderAgentList(context.Background(), &out, func() ([]scenarioruntime.EditorLease, error) { return leases, nil }, agentsession.Reconciler{Census: emptyCensus, Alive: alwaysAlive, Frozen: frozenUnknown, Now: fixedClock(now)}, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "unrecorded") {
		t.Fatalf("a lease naming a scope the kernel does not hold must say so:\n%s", out.String())
	}
}

// A census that cannot run must never make a session look healthy.
func TestAgentListReportsUndeterminedWhenTheKernelCannotBeRead(t *testing.T) {
	now := time.Now().UTC()
	leases := []scenarioruntime.EditorLease{{
		SessionID: "codex-1", Harness: "codex", Agent: "codex", PID: 4242,
		Scope: "vrooli-agent-codex-1", CreatedAt: now.Add(-time.Minute),
	}}
	blind := func() ([]agentscope.Entry, error) { return nil, errKernelUnreadable }
	var out bytes.Buffer
	if err := renderAgentList(context.Background(), &out, func() ([]scenarioruntime.EditorLease, error) { return leases, nil }, agentsession.Reconciler{Census: blind, Alive: alwaysAlive, Frozen: frozenUnknown, Now: fixedClock(now)}, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "undetermined") {
		t.Fatalf("an unreadable kernel is undetermined, never live:\n%s", out.String())
	}
}

var errKernelUnreadable = errors.New("cgroup unreadable")

// The list carries the reap policy's verdict and explicit-unknown usage per
// row, and a summary that never folds an unknown figure in as 0.
func TestAgentListJSONCarriesUsageAndReapVerdict(t *testing.T) {
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	old := now.Add(-20 * 24 * time.Hour)
	leases := []scenarioruntime.EditorLease{
		{
			SessionID: "desktop", PID: 1, Scope: "vrooli-agent-desktop", CreatedAt: old, LastHeartbeatAt: now,
			Owner: scenarioruntime.EditorLeaseOwner{Kind: agentsession.KindDesktopApp, Ref: "/data", Source: "launch"},
		},
		{
			SessionID: "ui", PID: 2, Scope: "vrooli-agent-ui", CreatedAt: old, LastHeartbeatAt: now,
			Owner: scenarioruntime.EditorLeaseOwner{Kind: agentsession.KindWebConsoleSession, Ref: "wc", Source: "launch", EvidenceJSON: `{"web_console_origin":"ui"}`},
		},
	}
	census := func() ([]agentscope.Entry, error) {
		return []agentscope.Entry{{Name: "vrooli-agent-desktop.scope"}, {Name: "vrooli-agent-ui.scope"}}, nil
	}
	reconciler := agentsession.Reconciler{
		Census: census, Alive: alwaysAlive, Frozen: frozenUnknown, Now: fixedClock(now),
		Probes: agentsession.Probes{
			DesktopAppProcesses: func(string) ([]int, error) { return nil, nil },
			WebConsoleSession: func(context.Context, string, string) (agentsession.WebConsoleSession, error) {
				return agentsession.WebConsoleSession{}, agentsession.ErrNotFound
			},
		},
	}
	var out bytes.Buffer
	if err := renderAgentList(context.Background(), &out, func() ([]scenarioruntime.EditorLease, error) { return leases, nil }, reconciler, true); err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Sessions []map[string]any `json:"sessions"`
		Summary  AgentListSummary `json:"summary"`
	}
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatalf("decode %s: %v", out.String(), err)
	}
	rows := map[string]map[string]any{}
	for _, row := range doc.Sessions {
		rows[row["session"].(string)] = row
	}
	desktop, ui := rows["desktop"], rows["ui"]
	if desktop["reapable"] != true || ui["reapable"] != false || ui["reap_reason"] != "web-console session origin ui: an operator terminal is never auto-reaped" {
		t.Fatalf("verdicts: desktop %v %v, ui %v %v", desktop["reapable"], desktop["reap_reason"], ui["reapable"], ui["reap_reason"])
	}
	// No host reader in this fixture: the figures are unknown, so they are
	// null with a status, never 0.
	for _, key := range []string{"memory_bytes", "swap_bytes"} {
		if value, present := desktop[key]; !present || value != nil {
			t.Fatalf("%s = %v (present %v), want an explicit null", key, value, present)
		}
	}
	if desktop["memory_status"] != agentsession.UsageUnavailable || desktop["swap_status"] != agentsession.UsageUnavailable {
		t.Fatalf("statuses = %v / %v", desktop["memory_status"], desktop["swap_status"])
	}
	if s := doc.Summary; s.Sessions != 2 || s.Reapable != 1 || s.MemoryUnknown != 2 || s.SwapUnknown != 2 || s.MemoryBytes != 0 ||
		s.ByOwnerKind[agentsession.KindDesktopApp][agentsession.StateOrphaned] != 1 {
		t.Fatalf("summary = %+v", s)
	}
}
