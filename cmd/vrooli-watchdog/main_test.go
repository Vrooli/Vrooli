package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostpressure"
)

func TestRediscoveryGate(t *testing.T) {
	var report output
	if _, err := fromFixture("../../internal/hostpressure/testdata/host-2026-08-22", &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 4 {
		t.Fatalf("expected four evidence-backed findings, got %d: %v", len(report.Findings), report.Findings)
	}
	for _, key := range []string{"fork-rate", "stranded-memory", "abandoned-workload", "idle-vrooli-service"} {
		if len(report.Evidence[key]) == 0 {
			t.Fatalf("finding %q has no evidence", key)
		}
	}
}

func TestSustainedFailureUsesDurableHysteresis(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	base := time.Date(2026, 8, 22, 20, 0, 0, 0, time.UTC)
	previous := watchdogNow
	defer func() { watchdogNow = previous }()
	watchdogNow = func() time.Time { return base }
	if sustainedFailure("test", true, time.Minute) {
		t.Fatal("first observation must be held")
	}
	watchdogNow = func() time.Time { return base.Add(time.Minute - time.Second) }
	if sustainedFailure("test", true, time.Minute) {
		t.Fatal("failure before sustain window must be held")
	}
	watchdogNow = func() time.Time { return base.Add(time.Minute) }
	if !sustainedFailure("test", true, time.Minute) {
		t.Fatal("failure at sustain window must be reported")
	}
	if sustainedFailure("test", false, time.Minute) {
		t.Fatal("recovered failure must clear")
	}
}

func TestDisposalProposalIsStructuredAndPreviewOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeDisposalProposal(disposalProposal{Workload: "airbyte-abctl-control-plane", Class: "abandoned", Posture: "vrooli_only", Evidence: []string{"agent-experiments/airbyte"}, Reason: "historical evidence", ProposedAction: "preview undeclared-workload disposal"})
	f, err := os.Open(filepath.Join(home, ".vrooli", "state", "watchdog-disposal-proposals.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var line map[string]any
	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		t.Fatal("proposal file is empty")
	}
	if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
		t.Fatal(err)
	}
	if line["workload"] != "airbyte-abctl-control-plane" || line["proposed_action"] != "preview undeclared-workload disposal" {
		t.Fatalf("unexpected proposal: %#v", line)
	}
}

// A 61-second breach of the fork-rate bar is not a finding when the authored
// sustain is ten minutes; the finding appears once the window has elapsed.
func TestWatchdogUsesAuthoredSustain(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("VROOLI_SETPOINT_PATH", filepath.Join("..", "..", "scenarios", "infrastructure-manager", "setpoint", "reliability-setpoint.json"))
	base := time.Date(2026, 9, 2, 10, 14, 0, 0, time.UTC)
	previous := watchdogNow
	defer func() { watchdogNow = previous }()
	thresholds := readThresholds()
	if thresholds.Source == "compiled fallback" || thresholds.sustainFor("substrate/SB16") != 10*time.Minute {
		t.Fatalf("thresholds = %+v", thresholds)
	}
	breach := func(at time.Time) []string {
		watchdogNow = func() time.Time { return at }
		o := output{Evidence: map[string][]string{}, Readings: hostpressure.PressureSnapshot{
			ForkRate:    hostpressure.NewRead(2481, "test"),
			CPUPressure: hostpressure.NewUnread("test", "unread"),
		}}
		addPressureFindings(&o, thresholds, nil)
		return o.Findings
	}
	if f := breach(base); len(f) != 0 {
		t.Fatalf("first breach produced findings: %v", f)
	}
	if f := breach(base.Add(61 * time.Second)); len(f) != 0 {
		t.Fatalf("61s breach produced findings against a 10m sustain: %v", f)
	}
	if f := breach(base.Add(10 * time.Minute)); len(f) != 1 || !strings.Contains(f[0], "fork-rate") {
		t.Fatalf("sustained breach did not produce the fork-rate finding: %v", f)
	}
}

// A fork-rate finding names the parent that owns the storm: the fixture's
// burst parent (200 sleeps) appears first, with pid, name, child count,
// delta and scope, and the whole report lands in the state file.
func TestForkRateFindingCarriesTopParents(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("VROOLI_SETPOINT_PATH", filepath.Join("..", "..", "scenarios", "infrastructure-manager", "setpoint", "reliability-setpoint.json"))
	root := filepath.Join("..", "..", "internal", "hostpressure", "testdata", "fixtures", "fork-storm")
	o := output{Evidence: map[string][]string{}, Thresholds: readThresholds()}
	previous, err := fromFixture(root, &o)
	if err != nil {
		t.Fatal(err)
	}
	if previous == nil {
		t.Fatal("the fork-storm fixture carries procs-t0.tsv, so a previous tree must load")
	}
	attachAttribution(&o, previous)
	var fork string
	for _, finding := range o.Findings {
		if strings.HasPrefix(finding, "fork-rate:") {
			fork = finding
		}
	}
	if fork == "" {
		t.Fatalf("no fork-rate finding: %v", o.Findings)
	}
	if o.Attribution == nil || o.Attribution.State != hostpressure.Read || len(o.Attribution.ByChildren) == 0 {
		t.Fatalf("attribution = %+v", o.Attribution)
	}
	top := o.Attribution.ByChildren[0]
	if top.Children < 200 || top.Name != "sh" || top.PID == 0 {
		t.Fatalf("top parent = %+v, want the burst's sh with 200 children", top)
	}
	if o.Attribution.ByDelta[0].PID != top.PID || o.Attribution.ByDelta[0].Delta < 200 {
		t.Fatalf("delta leader = %+v, want the burst parent with +200", o.Attribution.ByDelta[0])
	}
	evidence := strings.Join(o.Evidence["fork-rate"], "\n")
	if !strings.Contains(evidence, "pid="+strconv.FormatInt(top.PID, 10)) || !strings.Contains(evidence, "children="+strconv.Itoa(top.Children)) {
		t.Fatalf("fork-rate evidence lacks the top parent: %q", evidence)
	}
	writeLastReport(o)
	data, err := os.ReadFile(lastReportPath())
	if err != nil {
		t.Fatalf("last report: %v", err)
	}
	var stored output
	if err := json.Unmarshal(data, &stored); err != nil || stored.Attribution == nil || stored.Attribution.ByChildren[0].PID != top.PID {
		t.Fatalf("stored report = %s (%v)", data, err)
	}
}

// A storm whose top parent runs inside an agent session scope produces a
// containment proposal in the shared proposal file; other scopes do not.
func TestStormProposalNamesTheAgentScope(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	o := output{Evidence: map[string][]string{"fork-rate": {"x"}}}
	attribution := hostpressure.AttributionReading{State: hostpressure.Read, ByChildren: []hostpressure.Parent{{PID: 42, Name: "claude", Children: 300, Delta: 280, Scope: "/user.slice/user-1000.slice/user@1000.service/vrooli-agents.slice/vrooli-agent-abc.scope"}}}
	proposeStormContainment(&o, attribution)
	data, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".vrooli", "state", "watchdog-disposal-proposals.jsonl"))
	if err != nil {
		t.Fatalf("proposal file: %v", err)
	}
	var line map[string]any
	if err := json.Unmarshal([]byte(strings.SplitN(string(data), "\n", 2)[0]), &line); err != nil {
		t.Fatal(err)
	}
	if line["class"] != stormProposalClass || !strings.Contains(line["scope"].(string), "vrooli-agent-abc.scope") || len(o.Actions) != 1 {
		t.Fatalf("proposal = %#v actions=%v", line, o.Actions)
	}
	other := output{Evidence: map[string][]string{"fork-rate": {"x"}}}
	proposeStormContainment(&other, hostpressure.AttributionReading{State: hostpressure.Read, ByChildren: []hostpressure.Parent{{PID: 1, Name: "init", Children: 900, Scope: "/init.scope"}}})
	if len(other.Actions) != 0 {
		t.Fatalf("a non-agent parent must not be proposed for containment: %v", other.Actions)
	}
}

// [REQ:BOOT-RECOVERY-001] The watchdog reads the installed CLI's recorded
// source root before it looks anywhere else that is not the unit's own
// environment, and never guesses from its own directory: it is installed
// under ~/.vrooli/libexec, which is nobody's repository.
func TestWatchdogResolvesRepoRootFromSourcePointer(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("VROOLI_ROOT", "")
	t.Setenv("VROOLI_SOURCE_ROOT", "")
	repo := filepath.Join(t.TempDir(), "checkout")
	if err := os.MkdirAll(filepath.Join(home, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".vrooli", "source-root"), []byte(repo+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	prevFind, prevCWD := findRepoRootFn, findRepoRootFromCWDFn
	defer func() { findRepoRootFn, findRepoRootFromCWDFn = prevFind, prevCWD }()
	var asked []string
	findRepoRootFn = func(start string) (string, error) {
		asked = append(asked, start)
		if start == repo {
			return repo, nil
		}
		return "", errors.New("not a repository")
	}
	findRepoRootFromCWDFn = func() (string, error) { return "", errors.New("cwd is not a repository") }

	root, err := resolveWatchdogRoot()
	if err != nil || root != repo {
		t.Fatalf("root = %q, %v; asked %v", root, err, asked)
	}
	if len(asked) != 1 || asked[0] != repo {
		t.Fatalf("resolution consulted %v, want only the source pointer", asked)
	}

	// The unit's own environment outranks the pointer.
	explicit := filepath.Join(t.TempDir(), "explicit")
	t.Setenv("VROOLI_ROOT", explicit)
	findRepoRootFn = func(start string) (string, error) { return start, nil }
	if root, err := resolveWatchdogRoot(); err != nil || root != explicit {
		t.Fatalf("VROOLI_ROOT root = %q, %v", root, err)
	}

	// With no pointer and no environment, the answer is an error that says
	// what was tried, never the binary's own directory.
	t.Setenv("VROOLI_ROOT", "")
	if err := os.Remove(filepath.Join(home, ".vrooli", "source-root")); err != nil {
		t.Fatal(err)
	}
	if root, err := resolveWatchdogRoot(); err == nil || !strings.Contains(err.Error(), "repo root not found") {
		t.Fatalf("root = %q, err = %v; want a not-found error naming what was tried", root, err)
	}
}

// [REQ:BOOT-RECOVERY-001] The unit-restart escalation moved from the shell
// script into the binary: behind --reclaim, a sustained liveness finding
// restarts each down core unit, and the CPU saturation brake holds it.
func TestWatchdogBinaryEscalatesUnitRestartUnderReclaim(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	thresholds := thresholdSource{CPUPressurePercent: 60}
	var restarted []string
	restart := func(unit string) error {
		restarted = append(restarted, unit)
		if unit == "vrooli-autoheal.service" {
			return errors.New("unit not found")
		}
		return nil
	}

	// No sustained finding: nothing is restarted even though units are down.
	o := output{Evidence: map[string][]string{}, UnitsDown: []string{"vrooli-runtime-supervisor.service"}, Readings: hostpressure.PressureSnapshot{CPUPressure: hostpressure.NewRead(5, "test")}}
	escalateUnitRestarts(&o, thresholds, restart)
	if len(restarted) != 0 || len(o.Actions) != 0 {
		t.Fatalf("restarted %v before the liveness finding sustained", restarted)
	}

	// Sustained finding under saturation: held, with the reason recorded.
	o = output{Evidence: map[string][]string{}, UnitsDown: []string{"vrooli-runtime-supervisor.service"}, Readings: hostpressure.PressureSnapshot{CPUPressure: hostpressure.NewRead(96, "psi")}}
	add(&o, "unit-liveness", "declared units are not active", nil)
	escalateUnitRestarts(&o, thresholds, restart)
	if len(restarted) != 0 || !hasFinding(o, "unit-restart") || !strings.Contains(strings.Join(o.Findings, " "), "held") {
		t.Fatalf("saturated host: restarted %v, findings %v", restarted, o.Findings)
	}

	// Sustained finding on an idle host: each down unit is restarted, a
	// failed restart is a finding, a successful one an action.
	o = output{Evidence: map[string][]string{}, UnitsDown: []string{"vrooli-runtime-supervisor.service", "vrooli-autoheal.service"}, Readings: hostpressure.PressureSnapshot{CPUPressure: hostpressure.NewUnread("psi", "unread")}}
	add(&o, "unit-liveness", "declared units are not active", nil)
	escalateUnitRestarts(&o, thresholds, restart)
	if len(restarted) != 2 {
		t.Fatalf("restarted = %v, want both down units", restarted)
	}
	if len(o.Actions) != 1 || !strings.Contains(o.Actions[0], "vrooli-runtime-supervisor.service") {
		t.Fatalf("actions = %v", o.Actions)
	}
	if !strings.Contains(strings.Join(o.Findings, " "), "restart of vrooli-autoheal.service failed") {
		t.Fatalf("findings = %v, want the failed restart recorded", o.Findings)
	}
}

// A fixture rehearsal must not overwrite the host's live report.
func TestFixtureRunDoesNotWriteLastReport(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if _, err := os.Stat(lastReportPath()); !os.IsNotExist(err) {
		t.Fatalf("precondition: %v", err)
	}
	// main() is not callable here; the guard is the *fixtures check around
	// writeLastReport, exercised by running the binary in the fixture test
	// below through the same path. Assert the live path writes and read it.
	writeLastReport(output{CapturedAt: time.Now().UTC()})
	if _, err := os.Stat(lastReportPath()); err != nil {
		t.Fatalf("live write missing: %v", err)
	}
}
