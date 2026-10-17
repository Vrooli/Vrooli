package baseline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/encoding/protojson"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	safetyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/safety"
)

// recordedCall captures one shell-out so tests can assert the argv the
// engagement verbs build.
type recordedCall struct {
	name string
	args []string
}

// fakeRunner stubs the runCommand seam: it records every call and returns canned
// stdout keyed by "name arg0 arg1" prefix, or an error for a named failure.
type fakeRunner struct {
	calls  []recordedCall
	stdout map[string][]byte // keyed by a substring of "name args..."; first match wins
	failOn map[string]error  // keyed the same way
	// dynamic answers are keyed the same way and computed from the calls so far
	// (e.g. the floor's engagement list after a promote cleaned it); checked
	// before stdout.
	dynamic map[string]func() []byte
}

// newFakeRunner starts from a floor with no open engagements (an empty
// `recovery list`), the state `baseline start` checks before capturing.
func newFakeRunner(_ *testing.T) *fakeRunner {
	return &fakeRunner{
		stdout:  map[string][]byte{"recovery list": listJSON()},
		failOn:  map[string]error{},
		dynamic: map[string]func() []byte{},
	}
}

func (f *fakeRunner) run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, recordedCall{name: name, args: append([]string(nil), args...)})
	joined := name + " " + strings.Join(args, " ")
	for k, err := range f.failOn {
		if strings.Contains(joined, k) {
			return nil, err
		}
	}
	for k, answer := range f.dynamic {
		if strings.Contains(joined, k) {
			return answer(), nil
		}
	}
	for k, out := range f.stdout {
		if strings.Contains(joined, k) {
			return out, nil
		}
	}
	return nil, nil
}

func (f *fakeRunner) install() func() {
	prev := runCommand
	runCommand = f.run
	return func() { runCommand = prev }
}

func (f *fakeRunner) sawCommand(substr string) bool {
	return f.firstIndex(substr) >= 0
}

// firstIndex returns the index of the first recorded call matching substr, or -1.
func (f *fakeRunner) firstIndex(substr string) int {
	for i, c := range f.calls {
		if strings.Contains(c.name+" "+strings.Join(c.args, " "), substr) {
			return i
		}
	}
	return -1
}

// sawInOrder reports whether the first match of a precedes the first match of b
// (and both occurred).
func (f *fakeRunner) sawInOrder(a, b string) bool {
	ia, ib := f.firstIndex(a), f.firstIndex(b)
	return ia >= 0 && ib >= 0 && ia < ib
}

// ---- decision tree -------------------------------------------------------

func testCoreSet() coreSet {
	return coreSet{
		members:     map[string]bool{"test-genie": true, "agent-manager": true, "git-control-tower": true},
		trustedBase: map[string]bool{"git-control-tower": true, "test-genie": true, "data-backup-manager": true},
		source:      "test",
	}
}

func TestDecideModeTrustedBaseHardStop(t *testing.T) {
	cs := testCoreSet()
	// Even an explicit --mode shadow can't shadow a trusted-base scenario.
	d := decideMode("git-control-tower", modeShadow, modeSignals{}, cs)
	if d.Mode != modeLive {
		t.Fatalf("trusted base must route to live, got %q", d.Mode)
	}
	if !d.Reflexive {
		t.Errorf("trusted base must be reflexive")
	}
	if !d.NeedsOperator {
		t.Errorf("trusted base live without --operator-confirm needs an operator nod")
	}
	d2 := decideMode("git-control-tower", modeShadow, modeSignals{operatorConfirm: true}, cs)
	if d2.NeedsOperator {
		t.Errorf("operator-confirm should clear NeedsOperator")
	}
}

func TestDecideModeNamespaceabilityHardGate(t *testing.T) {
	cs := testCoreSet()
	// A non-reflexive scenario that writes an un-adopted store is forced to live
	// even when shadow is explicitly requested.
	d := decideMode("some-scenario", modeShadow, modeSignals{writesSharedStore: true}, cs)
	if d.Mode != modeLive {
		t.Fatalf("namespaceability hard gate must override explicit shadow, got %q", d.Mode)
	}
	if d.NeedsOperator {
		t.Errorf("non-reflexive scenario should not need an operator nod")
	}
}

func TestDecideModeExplicitShadowOverridesSoftGates(t *testing.T) {
	cs := testCoreSet()
	d := decideMode("some-scenario", modeShadow, modeSignals{modifiesLifecycle: true, singletonResource: true}, cs)
	if d.Mode != modeShadow {
		t.Fatalf("explicit shadow should override soft gates, got %q", d.Mode)
	}
	// The soft gates are surfaced as notes.
	joined := strings.Join(d.Reasons, "|")
	if !strings.Contains(joined, "overridden by explicit --mode shadow") {
		t.Errorf("soft gates should be noted, reasons=%v", d.Reasons)
	}
}

func TestDecideModeAutoSoftGateRoutesLive(t *testing.T) {
	cs := testCoreSet()
	d := decideMode("some-scenario", modeAuto, modeSignals{modifiesLifecycle: true}, cs)
	if d.Mode != modeLive {
		t.Fatalf("auto with a soft gate should route live, got %q", d.Mode)
	}
}

func TestDecideModeAutoDefaultsShadow(t *testing.T) {
	cs := testCoreSet()
	d := decideMode("some-scenario", modeAuto, modeSignals{}, cs)
	if d.Mode != modeShadow {
		t.Fatalf("auto with no gates should default shadow, got %q", d.Mode)
	}
	if d.NeedsOperator {
		t.Errorf("shadow should never need an operator nod")
	}
}

func TestDecideModeLiveOnReflexiveNeedsOperator(t *testing.T) {
	cs := testCoreSet()
	d := decideMode("agent-manager", modeLive, modeSignals{}, cs)
	if d.Mode != modeLive {
		t.Fatalf("live requested → live, got %q", d.Mode)
	}
	if !d.NeedsOperator {
		t.Errorf("live on a reflexive (non-trusted) scenario needs an operator nod")
	}
	d2 := decideMode("agent-manager", modeLive, modeSignals{operatorConfirm: true}, cs)
	if d2.NeedsOperator {
		t.Errorf("operator-confirm clears the nod")
	}
}

func TestDecideModeLiveOnNonReflexiveNoNod(t *testing.T) {
	cs := testCoreSet()
	d := decideMode("some-scenario", modeLive, modeSignals{}, cs)
	if d.Mode != modeLive || d.NeedsOperator {
		t.Fatalf("live on a plain scenario needs no nod, got %+v", d)
	}
}

// ---- loadCoreSet ---------------------------------------------------------

func TestLoadCoreSetFallbackWhenAnalyzerDown(t *testing.T) {
	f := newFakeRunner(t)
	f.failOn["scenario-dependency-analyzer core-set"] = fmt.Errorf("analyzer down")
	defer f.install()()

	cs := loadCoreSet(context.Background())
	// The SSOT constant must still seat the trusted base (hard-stop must hold).
	if !cs.isTrustedBase("git-control-tower") {
		t.Errorf("git-control-tower must be trusted base from the constant even with the analyzer down")
	}
	if cs.source != "fallback" {
		t.Errorf("source = %q, want fallback", cs.source)
	}
	if !cs.isMember("test-genie") {
		t.Errorf("seed members must be present from the constant")
	}
}

func TestLoadCoreSetAugmentsFromAnalyzer(t *testing.T) {
	f := newFakeRunner(t)
	resp := map[string]any{
		"source":       "computed",
		"core_set":     []string{"audio-tools"},
		"trusted_base": []string{},
	}
	b, _ := json.Marshal(resp)
	f.stdout["scenario-dependency-analyzer core-set"] = b
	defer f.install()()

	cs := loadCoreSet(context.Background())
	if !cs.isMember("audio-tools") {
		t.Errorf("closure member from the analyzer must be unioned in")
	}
	// The constant's trusted base is never shrunk by the analyzer.
	if !cs.isTrustedBase("git-control-tower") {
		t.Errorf("constant trusted base must survive an empty analyzer trusted_base")
	}
	if cs.source != "analyzer:computed" {
		t.Errorf("source = %q", cs.source)
	}
}

// ---- start ---------------------------------------------------------------

// fakeSnapshot stubs the anchor-snapshot seam.
func withFakeAnchors(t *testing.T, snapErr, diffVerdict string) func() {
	t.Helper()
	prevSnap, prevDiff := snapshotAnchor, diffAnchor
	snapshotAnchor = func(_ *cliapp.ScenarioApp, _ context.Context, _, _ string) error {
		if snapErr != "" {
			return fmt.Errorf("%s", snapErr)
		}
		return nil
	}
	diffAnchor = func(_ *cliapp.ScenarioApp, _ context.Context, _, _ string) (string, error) {
		return diffVerdict, nil
	}
	return func() { snapshotAnchor, diffAnchor = prevSnap, prevDiff }
}

func TestStartShadowHappyPath(t *testing.T) {
	f := newFakeRunner(t)
	// No analyzer → fallback constant; a non-reflexive scenario goes shadow.
	f.failOn["scenario-dependency-analyzer"] = fmt.Errorf("down")
	defer f.install()()
	defer withFakeAnchors(t, "", "clean")()

	res, err := startEngagement(nil, startParams{scenario: "demo-scenario", mode: modeAuto, slug: "wip"})
	if err != nil {
		t.Fatalf("startEngagement: %v", err)
	}
	if res.Decision.Mode != modeShadow {
		t.Fatalf("expected shadow, got %q", res.Decision.Mode)
	}
	if res.Variant != "shadow" || res.AmbientVar != "demo-scenario" {
		t.Errorf("shadow should set variant+ambient, got %+v", res)
	}
	if res.Anchor != "engagement-wip" {
		t.Errorf("anchor = %q", res.Anchor)
	}
	// The full floor sequence must have run, in order: capture → write → shadow start.
	if !f.sawCommand("recovery capture --scenario demo-scenario --slug wip") {
		t.Errorf("missing capture; calls=%v", f.calls)
	}
	if !f.sawCommand("recovery write --scenario demo-scenario --slug wip --mode shadow") {
		t.Errorf("missing manifest write; calls=%v", f.calls)
	}
	if !f.sawCommand("recovery write") || !f.sawCommand("--ambient-var demo-scenario") {
		t.Errorf("write should carry the ambient var; calls=%v", f.calls)
	}
	if !f.sawCommand("scenario start demo-scenario --instance shadow") {
		t.Errorf("missing shadow stand-up; calls=%v", f.calls)
	}
	// A shadow capture freezes the shared packages live builds from and fails
	// closed; only an explicit --allow-unfrozen relaxes it.
	if f.sawCommand("--allow-unfrozen") {
		t.Errorf("a shadow start must not relax the shared-package freeze by default; calls=%v", f.calls)
	}
}

func TestStartShadowAllowUnfrozenPassesTheOverride(t *testing.T) {
	f := newFakeRunner(t)
	f.failOn["scenario-dependency-analyzer"] = fmt.Errorf("down")
	defer f.install()()
	defer withFakeAnchors(t, "", "clean")()

	if _, err := startEngagement(nil, startParams{scenario: "demo-scenario", mode: modeShadow, slug: "wip", noAnchor: true, allowUnfrozen: true}); err != nil {
		t.Fatalf("startEngagement: %v", err)
	}
	if !f.sawCommand("recovery capture --scenario demo-scenario --slug wip --allow-unfrozen") {
		t.Errorf("--allow-unfrozen must reach the capture; calls=%v", f.calls)
	}
}

func TestStartLiveDoesNotStandUpShadow(t *testing.T) {
	f := newFakeRunner(t)
	f.failOn["scenario-dependency-analyzer"] = fmt.Errorf("down")
	defer f.install()()
	defer withFakeAnchors(t, "", "clean")()

	res, err := startEngagement(nil, startParams{scenario: "demo-scenario", mode: modeLive, slug: "wip"})
	if err != nil {
		t.Fatalf("startEngagement: %v", err)
	}
	if res.Decision.Mode != modeLive || res.Variant != "live" {
		t.Fatalf("expected live, got %+v", res)
	}
	if f.sawCommand("scenario start") {
		t.Errorf("live mode must not stand up a shadow; calls=%v", f.calls)
	}
	if f.sawCommand("--ambient-var") {
		t.Errorf("live mode must not set an ambient var; calls=%v", f.calls)
	}
	// Live runs the working tree in place and never builds from a freeze, so a
	// failed freeze must not block a live engagement.
	if !f.sawCommand("recovery capture --scenario demo-scenario --slug wip --allow-unfrozen") {
		t.Errorf("a live capture must not fail closed on the shared-package freeze; calls=%v", f.calls)
	}
}

func TestStartReflexiveLiveRequiresOperatorConfirm(t *testing.T) {
	f := newFakeRunner(t)
	// Analyzer down → constant seeds the reflexive set incl. agent-manager.
	f.failOn["scenario-dependency-analyzer"] = fmt.Errorf("down")
	defer f.install()()
	defer withFakeAnchors(t, "", "clean")()

	_, err := startEngagement(nil, startParams{scenario: "agent-manager", mode: modeLive, slug: "wip"})
	if err == nil || !strings.Contains(err.Error(), "operator nod") {
		t.Fatalf("expected operator-nod error, got %v", err)
	}
	// Nothing should have been captured/written when the nod is missing.
	if f.sawCommand("recovery capture") {
		t.Errorf("must not capture before the operator nod; calls=%v", f.calls)
	}

	// With the nod it proceeds (live, no shadow).
	res, err := startEngagement(nil, startParams{scenario: "agent-manager", mode: modeLive, slug: "wip", signals: modeSignals{operatorConfirm: true}})
	if err != nil {
		t.Fatalf("with --operator-confirm: %v", err)
	}
	if res.Decision.Mode != modeLive {
		t.Errorf("expected live, got %q", res.Decision.Mode)
	}
}

func TestStartNoAnchorSkipsSnapshot(t *testing.T) {
	f := newFakeRunner(t)
	f.failOn["scenario-dependency-analyzer"] = fmt.Errorf("down")
	defer f.install()()
	snapCalled := false
	prevSnap := snapshotAnchor
	snapshotAnchor = func(_ *cliapp.ScenarioApp, _ context.Context, _, _ string) error { snapCalled = true; return nil }
	defer func() { snapshotAnchor = prevSnap }()

	res, err := startEngagement(nil, startParams{scenario: "demo-scenario", mode: modeShadow, slug: "wip", noAnchor: true})
	if err != nil {
		t.Fatalf("startEngagement: %v", err)
	}
	if snapCalled {
		t.Errorf("--no-anchor must skip the anchor snapshot")
	}
	if res.Anchor != "" {
		t.Errorf("anchor should be empty, got %q", res.Anchor)
	}
	if f.sawCommand("--anchor") {
		t.Errorf("write must not carry --anchor when skipped; calls=%v", f.calls)
	}
}

func TestStartInvalidModeRejected(t *testing.T) {
	if _, err := startEngagement(nil, startParams{scenario: "x", mode: "bogus"}); err == nil {
		t.Fatal("expected invalid-mode error")
	}
	if _, err := startEngagement(nil, startParams{scenario: ""}); err == nil {
		t.Fatal("expected missing-scenario error")
	}
	if _, err := startEngagement(nil, startParams{scenario: "x", ttl: "notaduration"}); err == nil {
		t.Fatal("expected invalid-ttl error")
	}
}

func TestStartTTLThreadedToWrite(t *testing.T) {
	f := newFakeRunner(t)
	f.failOn["scenario-dependency-analyzer"] = fmt.Errorf("down")
	defer f.install()()
	defer withFakeAnchors(t, "", "clean")()

	res, err := startEngagement(nil, startParams{scenario: "demo-scenario", mode: modeShadow, slug: "wip", ttl: "3h"})
	if err != nil {
		t.Fatalf("startEngagement: %v", err)
	}
	if res.TTL != "3h0m0s" {
		t.Errorf("ttl = %q", res.TTL)
	}
	if !f.sawCommand("--ttl 3h0m0s") {
		t.Errorf("write should carry --ttl; calls=%v", f.calls)
	}
}

// ---- open-engagement guard (O15) -----------------------------------------

// openGoalEngagement is the 2026-10-07 incident's floor state: a goal's shadow
// engagement, open (TTL-less) since the goal started.
func openGoalEngagement(expired bool) engagementView {
	opened := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	return engagementView{
		Scenario: "demo-scenario", Slug: "bas-goal", Mode: "shadow", Variant: "shadow",
		AnchorBaselineName: "engagement-bas-goal", AmbientVar: "demo-scenario", TTL: "0s",
		CreatedAt: &opened, LastTouchedAt: &opened, Expired: expired,
	}
}

// assertNothingCaptured fails when start touched the restore point, the anchor
// record, the manifest or the shadow instance.
func assertNothingCaptured(t *testing.T, f *fakeRunner, snapCalled bool) {
	t.Helper()
	for _, step := range []string{"recovery capture", "recovery write", "scenario start", "safety backup-now"} {
		if f.sawCommand(step) {
			t.Errorf("a refused start must not run %q; calls=%v", step, f.calls)
		}
	}
	if snapCalled {
		t.Errorf("a refused start must not capture an anchor snapshot")
	}
}

func withSnapshotSpy(t *testing.T) *bool {
	t.Helper()
	called := false
	prev := snapshotAnchor
	snapshotAnchor = func(_ *cliapp.ScenarioApp, _ context.Context, _, _ string) error { called = true; return nil }
	t.Cleanup(func() { snapshotAnchor = prev })
	return &called
}

func TestStartRefusesOpenEngagementBeforeCapture(t *testing.T) {
	for _, expired := range []bool{false, true} {
		t.Run(fmt.Sprintf("expired=%v", expired), func(t *testing.T) {
			f := newFakeRunner(t)
			f.failOn["scenario-dependency-analyzer"] = fmt.Errorf("down")
			f.stdout["recovery list"] = listJSON(openGoalEngagement(expired))
			defer f.install()()
			snapCalled := withSnapshotSpy(t)

			_, err := startEngagement(nil, startParams{scenario: "demo-scenario", mode: modeShadow, slug: "bas-goal"})
			if err == nil {
				t.Fatal("start over an open engagement must be refused")
			}
			assertNothingCaptured(t, f, *snapCalled)
			for _, want := range []string{
				"did not capture or copy anything",
				"baseline check --scenario demo-scenario --name bas-goal",
				"baseline promote --scenario demo-scenario --name bas-goal",
				"baseline cycle --scenario demo-scenario --name bas-goal",
				"baseline abandon --scenario demo-scenario --name bas-goal",
				"--replace (keeps its restore point)",
				"opened 2026-09-29T12:00:00Z",
			} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("refusal should contain %q:\n%v", want, err)
				}
			}
			if strings.Contains(err.Error(), "baseline gc") != expired {
				t.Errorf("the gc hint belongs only to an expired engagement (expired=%v):\n%v", expired, err)
			}
		})
	}
}

func TestStartRefusesSecondEngagementUnderAnotherName(t *testing.T) {
	f := newFakeRunner(t)
	f.failOn["scenario-dependency-analyzer"] = fmt.Errorf("down")
	f.stdout["recovery list"] = listJSON(
		engagementView{Scenario: "other-scenario", Slug: "wip", Mode: "shadow", Variant: "shadow"},
		openGoalEngagement(false),
	)
	defer f.install()()
	snapCalled := withSnapshotSpy(t)

	// The bare command TARGETS.md drifted to: no --name, so slug wip.
	_, err := startEngagement(nil, startParams{scenario: "demo-scenario", mode: modeShadow, slug: "wip", replace: true})
	if err == nil {
		t.Fatal("a second engagement for the scenario must be refused, even with --replace")
	}
	assertNothingCaptured(t, f, *snapCalled)
	if !strings.Contains(err.Error(), "did you mean --name bas-goal?") {
		t.Errorf("refusal should name the open engagement's slug:\n%v", err)
	}
}

func TestStartReplaceTakesOverWithoutRecapturing(t *testing.T) {
	f := newFakeRunner(t)
	f.failOn["scenario-dependency-analyzer"] = fmt.Errorf("down")
	f.stdout["recovery list"] = listJSON(openGoalEngagement(false))
	defer f.install()()
	snapCalled := withSnapshotSpy(t)

	res, err := startEngagement(nil, startParams{scenario: "demo-scenario", mode: modeShadow, slug: "bas-goal", replace: true})
	if err != nil {
		t.Fatalf("--replace takeover: %v", err)
	}
	if f.sawCommand("recovery capture") {
		t.Errorf("a takeover must keep the open engagement's restore point; calls=%v", f.calls)
	}
	if *snapCalled {
		t.Errorf("a takeover must keep the open engagement's anchor, not snapshot the candidate")
	}
	if !f.sawCommand("recovery write --scenario demo-scenario --slug bas-goal --mode shadow") ||
		!f.sawCommand("--anchor engagement-bas-goal") || !f.sawCommand("--replace") {
		t.Errorf("the takeover must re-write the manifest with --replace and the kept anchor; calls=%v", f.calls)
	}
	if res.RestorePoint != "preserved" || res.Anchor != "engagement-bas-goal" {
		t.Errorf("result should report the preserved restore point and kept anchor, got %+v", res)
	}
	if f.sawCommand("safety backup-now") || f.sawCommand("safety populate-shadow") {
		t.Errorf("a takeover must keep the shadow's data, not re-seed it from live; calls=%v", f.calls)
	}
}

func TestStartReplaceRefusesModeChange(t *testing.T) {
	f := newFakeRunner(t)
	f.failOn["scenario-dependency-analyzer"] = fmt.Errorf("down")
	f.stdout["recovery list"] = listJSON(openGoalEngagement(false))
	defer f.install()()

	_, err := startEngagement(nil, startParams{scenario: "demo-scenario", mode: modeLive, slug: "bas-goal", replace: true})
	if err == nil || !strings.Contains(err.Error(), "a takeover keeps the mode") {
		t.Fatalf("a takeover that flips shadow to live must be refused, got %v", err)
	}
	if f.sawCommand("recovery write") || f.sawCommand("recovery capture") {
		t.Errorf("a refused takeover must not touch the floor; calls=%v", f.calls)
	}
}

func TestStartRefusesWhenOpenEngagementsUnreadable(t *testing.T) {
	f := newFakeRunner(t)
	f.failOn["recovery list"] = fmt.Errorf("floor unavailable")
	defer f.install()()
	snapCalled := withSnapshotSpy(t)

	_, err := startEngagement(nil, startParams{scenario: "demo-scenario", mode: modeShadow, slug: "wip"})
	if err == nil || !strings.Contains(err.Error(), "could not check for an open engagement") {
		t.Fatalf("start must fail closed when it cannot read the floor, got %v", err)
	}
	assertNothingCaptured(t, f, *snapCalled)
}

// ---- safety backup (O17) -------------------------------------------------

func TestSafetyBackupNowReadsTheCLIRunID(t *testing.T) {
	cases := map[string][]byte{
		"cli output (run_id)": backupNowJSON("run-7"),
		"lowerCamel (runId)":  []byte(`{"runId":"run-7"}`),
	}
	for name, out := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFakeRunner(t)
			f.stdout["safety backup-now"] = out
			defer f.install()()

			runID, note := safetyBackupNow(context.Background(), "demo-scenario")
			if runID != "run-7" || note != "" {
				t.Fatalf("safetyBackupNow = (%q, %q), want (run-7, \"\")", runID, note)
			}
		})
	}
}

func TestSafetyBackupNowSaysWhyNoRunStarted(t *testing.T) {
	cases := []struct {
		name string
		out  []byte
		err  error
		want string
	}{
		{"code-only", nil, fmt.Errorf("exit status 1: scenario has no registered targets: demo-scenario"), "code-only"},
		{"substrate down", nil, fmt.Errorf("connection refused"), "safety backup unavailable: "},
		{"no run id", []byte(`{"status":"RUN_STATUS_PENDING"}`), nil, "no run id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeRunner(t)
			if tc.err != nil {
				f.failOn["safety backup-now"] = tc.err
			} else {
				f.stdout["safety backup-now"] = tc.out
			}
			defer f.install()()

			runID, note := safetyBackupNow(context.Background(), "demo-scenario")
			if runID != "" || !strings.Contains(note, tc.want) {
				t.Fatalf("safetyBackupNow = (%q, %q), want no run and a note containing %q", runID, note, tc.want)
			}
		})
	}
}

// ---- shadow data population ----------------------------------------------

// withNoSleep stubs the poll-delay seam so the wait loop runs without real time.
func withNoSleep(t *testing.T) func() {
	t.Helper()
	prev := sleepFn
	sleepFn = func(time.Duration) {}
	return func() { sleepFn = prev }
}

// seedPopulationStdout primes the fakeRunner with the canned data-substrate +
// floor responses a successful shadow data population reads, in the shapes the
// producers print (generated messages with proto field names).
func seedPopulationStdout(f *fakeRunner, registered, runStatus, postgresDB, dataDir string) {
	f.stdout["safety register-targets"] = []byte(`{"registered":[` + registered + `]}`)
	f.stdout["safety backup-now"] = backupNowJSON("run-123")
	f.stdout["runs get"] = []byte(`{"run":{"status":"` + runStatus + `"}}`)
	f.stdout["recovery namespace"], _ = protojson.MarshalOptions{UseProtoNames: true}.Marshal(&cliv1.RecoveryNamespaceOutput{
		PostgresDb: postgresDB, DataDir: dataDir,
	})
	f.stdout["safety populate-shadow"] = []byte(`{}`)
}

// backupNowJSON renders `data-backup-manager safety backup-now --json` exactly
// as the CLI prints it: cli-core's PrintProtoJSON of the generated response
// ({"run_id": …}).
func backupNowJSON(runID string) []byte {
	var buf bytes.Buffer
	_ = cliapp.PrintProtoJSON(&buf, &safetyv1.BackupScenarioNowResponse{
		RunId: runID, PlanId: "plan-1", DestinationId: "dest-1", TargetCount: 1, Status: "RUN_STATUS_PENDING",
	})
	return buf.Bytes()
}

func TestStartShadowPopulatesData(t *testing.T) {
	f := newFakeRunner(t)
	f.failOn["scenario-dependency-analyzer"] = fmt.Errorf("down")
	seedPopulationStdout(f, `{"name":"postgres"},{"name":"data"}`, "RUN_STATUS_COMPLETED",
		"vrooli_demo_scenario_shadow", "/data/vrooli/demo-scenario@shadow")
	defer f.install()()
	defer withFakeAnchors(t, "", "clean")()
	defer withNoSleep(t)()

	res, err := startEngagement(nil, startParams{scenario: "demo-scenario", mode: modeAuto, slug: "wip"})
	if err != nil {
		t.Fatalf("startEngagement: %v", err)
	}
	// The data half must run after the shadow stand-up: register → backup → poll →
	// resolve namespaces → populate.
	if !f.sawCommand("safety register-targets --scenario demo-scenario") {
		t.Errorf("missing register-targets; calls=%v", f.calls)
	}
	if !f.sawCommand("safety backup-now --scenario demo-scenario") {
		t.Errorf("missing backup-now; calls=%v", f.calls)
	}
	if !f.sawCommand("runs get run-123") {
		t.Errorf("missing run poll; calls=%v", f.calls)
	}
	if !f.sawCommand("recovery namespace --scenario demo-scenario --variant shadow") {
		t.Errorf("missing namespace query; calls=%v", f.calls)
	}
	// Both registered targets must map to their shadow locations.
	if !f.sawCommand("safety populate-shadow --scenario demo-scenario --run-id run-123 --mappings postgres=vrooli_demo_scenario_shadow,data=/data/vrooli/demo-scenario@shadow") {
		t.Errorf("populate-shadow mappings wrong; calls=%v", f.calls)
	}
	if len(res.DataPopulation) != 1 || !strings.Contains(res.DataPopulation[0], "populated from safety run run-123") {
		t.Errorf("DataPopulation = %v, want success note", res.DataPopulation)
	}
}

func TestStartShadowCodeOnlySkipsPopulation(t *testing.T) {
	f := newFakeRunner(t)
	f.failOn["scenario-dependency-analyzer"] = fmt.Errorf("down")
	// No registered targets → code-only; nothing to copy.
	f.stdout["safety register-targets"] = []byte(`{"registered":[]}`)
	defer f.install()()
	defer withFakeAnchors(t, "", "clean")()
	defer withNoSleep(t)()

	res, err := startEngagement(nil, startParams{scenario: "demo-scenario", mode: modeShadow, slug: "wip"})
	if err != nil {
		t.Fatalf("startEngagement: %v", err)
	}
	if f.sawCommand("safety backup-now") || f.sawCommand("safety populate-shadow") {
		t.Errorf("code-only scenario must not back up or populate; calls=%v", f.calls)
	}
	if len(res.DataPopulation) != 1 || !strings.Contains(res.DataPopulation[0], "code-only") {
		t.Errorf("DataPopulation = %v, want code-only skip note", res.DataPopulation)
	}
}

func TestStartShadowBackupTimeoutSkips(t *testing.T) {
	f := newFakeRunner(t)
	f.failOn["scenario-dependency-analyzer"] = fmt.Errorf("down")
	// The backup is enqueued but never reaches terminal — population must give up
	// without failing the engagement, and never populate from an unfinished run.
	seedPopulationStdout(f, `{"name":"postgres"}`, "RUN_STATUS_CAPTURING",
		"vrooli_demo_scenario_shadow", "")
	defer f.install()()
	defer withFakeAnchors(t, "", "clean")()
	defer withNoSleep(t)()

	res, err := startEngagement(nil, startParams{scenario: "demo-scenario", mode: modeShadow, slug: "wip"})
	if err != nil {
		t.Fatalf("startEngagement must not fail on a stuck backup: %v", err)
	}
	if f.sawCommand("safety populate-shadow") {
		t.Errorf("must not populate from an unfinished run; calls=%v", f.calls)
	}
	if len(res.DataPopulation) != 1 || !strings.Contains(res.DataPopulation[0], "did not finish") {
		t.Errorf("DataPopulation = %v, want poll-budget skip note", res.DataPopulation)
	}
}

func TestShadowTargetMappingsDropsUnmappableAndEmpty(t *testing.T) {
	f := newFakeRunner(t)
	// "qdrant" has no shadow location (not derivable here) and an empty dataDir
	// must be dropped — only "postgres" survives.
	f.stdout["recovery namespace"] = []byte(`{"postgresDb":"vrooli_x_shadow","dataDir":""}`)
	defer f.install()()

	got := shadowTargetMappings(context.Background(), "x", "shadow", []string{"postgres", "data", "qdrant"})
	if got != "postgres=vrooli_x_shadow" {
		t.Fatalf("mappings = %q, want only the postgres pair", got)
	}
}

func TestShadowTargetMappingsNamespaceQueryFailureIsEmpty(t *testing.T) {
	f := newFakeRunner(t)
	f.failOn["recovery namespace"] = fmt.Errorf("floor down")
	defer f.install()()

	if got := shadowTargetMappings(context.Background(), "x", "shadow", []string{"postgres"}); got != "" {
		t.Fatalf("a failed namespace query should yield no mappings, got %q", got)
	}
}

func TestSafetyRunTerminal(t *testing.T) {
	terminal := []string{"RUN_STATUS_COMPLETED", "RUN_STATUS_PARTIAL_FAILED", "RUN_STATUS_FAILED"}
	for _, s := range terminal {
		if !safetyRunTerminal(s) {
			t.Errorf("%s should be terminal", s)
		}
	}
	nonTerminal := []string{"RUN_STATUS_PENDING", "RUN_STATUS_CAPTURING", "RUN_STATUS_SNAPSHOTTING", "", "garbage"}
	for _, s := range nonTerminal {
		if safetyRunTerminal(s) {
			t.Errorf("%s should not be terminal", s)
		}
	}
}

// ---- check ---------------------------------------------------------------

// engagementJSON renders a `recovery show/list` fixture as the producer does:
// the typed vrooli.cli.v1 contract marshaled with UseProtoNames (snake_case).
func engagementJSON(mode, variant, anchor string) []byte {
	b, _ := protojson.MarshalOptions{UseProtoNames: true}.Marshal(&cliv1.RecoveryEngagementView{
		Scenario: "demo-scenario", Slug: "wip", Mode: mode, Variant: variant, AnchorBaselineName: anchor,
	})
	return b
}

func TestCheckCleanShadowGuidance(t *testing.T) {
	f := newFakeRunner(t)
	f.stdout["recovery show"] = engagementJSON("shadow", "shadow", "engagement-wip")
	defer f.install()()
	defer withFakeAnchors(t, "", "clean")()

	res, err := checkEngagement(nil, "demo-scenario", "wip")
	if err != nil {
		t.Fatalf("checkEngagement: %v", err)
	}
	if res.Verdict != "clean" {
		t.Errorf("verdict = %q", res.Verdict)
	}
	if !strings.Contains(res.Guidance, "shadow") {
		t.Errorf("shadow-clean guidance unexpected: %q", res.Guidance)
	}
	// The lease must be renewed.
	if !f.sawCommand("recovery touch --scenario demo-scenario --slug wip") {
		t.Errorf("check must renew the lease; calls=%v", f.calls)
	}
}

func TestCheckRegressionGuidance(t *testing.T) {
	f := newFakeRunner(t)
	f.stdout["recovery show"] = engagementJSON("shadow", "shadow", "engagement-wip")
	defer f.install()()
	defer withFakeAnchors(t, "", "regression")()

	res, err := checkEngagement(nil, "demo-scenario", "wip")
	if err != nil {
		t.Fatalf("checkEngagement: %v", err)
	}
	if exitCodeForVerdict(res.Verdict) != exitRegression {
		t.Errorf("regression should map to a non-zero exit, verdict=%q", res.Verdict)
	}
	if !strings.Contains(res.Guidance, "fix") {
		t.Errorf("regression guidance should prompt a fix: %q", res.Guidance)
	}
}

func TestCheckNoAnchorIsError(t *testing.T) {
	f := newFakeRunner(t)
	f.stdout["recovery show"] = engagementJSON("shadow", "shadow", "")
	defer f.install()()
	defer withFakeAnchors(t, "", "clean")()

	if _, err := checkEngagement(nil, "demo-scenario", "wip"); err == nil || !strings.Contains(err.Error(), "no anchor") {
		t.Fatalf("expected no-anchor error, got %v", err)
	}
}

// ---- abandon -------------------------------------------------------------

func TestAbandonShadowDiscardsCandidateAndRestartsLiveBeforeClean(t *testing.T) {
	f := newFakeRunner(t)
	f.stdout["recovery show"] = engagementJSON("shadow", "shadow", "engagement-wip")
	defer f.install()()

	res, err := abandonEngagement(nil, "demo-scenario", "wip")
	if err != nil {
		t.Fatalf("abandonEngagement: %v", err)
	}
	if !f.sawCommand("scenario stop demo-scenario --instance shadow") {
		t.Errorf("shadow abandon must stop the shadow; calls=%v", f.calls)
	}
	// In the live-from-copy model the working tree holds the candidate, so abandon
	// must discard it by restoring the baseline over the working tree — after the
	// shadow (which runs from that tree) is stopped.
	if !f.sawCommand("recovery restore --scenario demo-scenario --slug wip") {
		t.Errorf("shadow abandon must restore the baseline over the working tree; calls=%v", f.calls)
	}
	if !f.sawInOrder("scenario stop demo-scenario --instance shadow", "recovery restore --scenario demo-scenario --slug wip") {
		t.Errorf("shadow must be stopped before the working tree is overwritten; calls=%v", f.calls)
	}
	// Live served the baseline from the engagement's serving tree; the clean
	// restarts it from the restored working tree before deleting that tree.
	if !f.sawInOrder("recovery restore --scenario demo-scenario --slug wip", "recovery clean --scenario demo-scenario --slug wip --restart-live") {
		t.Errorf("shadow abandon must restart live from the restored working tree while cleaning; calls=%v", f.calls)
	}
	if !strings.Contains(res.Action, "live restarted") {
		t.Errorf("action = %q", res.Action)
	}
}

func TestAbandonShadowKeepsEngagementWhenLiveRestartFails(t *testing.T) {
	f := newFakeRunner(t)
	f.stdout["recovery show"] = engagementJSON("shadow", "shadow", "engagement-wip")
	f.failOn["recovery clean"] = fmt.Errorf("recovery: restart browser-automation-studio from the working tree: build failed; the engagement is kept")
	defer f.install()()

	if _, err := abandonEngagement(nil, "demo-scenario", "wip"); err == nil || !strings.Contains(err.Error(), "engagement is kept") {
		t.Fatalf("a failed live restart must fail abandon and keep the engagement, got %v", err)
	}
}

func TestAbandonLiveRestoresAndRestarts(t *testing.T) {
	f := newFakeRunner(t)
	f.stdout["recovery show"] = engagementJSON("live", "live", "engagement-wip")
	defer f.install()()

	res, err := abandonEngagement(nil, "demo-scenario", "wip")
	if err != nil {
		t.Fatalf("abandonEngagement: %v", err)
	}
	if !f.sawCommand("recovery restore --scenario demo-scenario --slug wip") {
		t.Errorf("live abandon must restore the working tree; calls=%v", f.calls)
	}
	if !f.sawCommand("scenario restart demo-scenario") {
		t.Errorf("live abandon must restart; calls=%v", f.calls)
	}
	if f.sawCommand("scenario stop") {
		t.Errorf("live abandon must not stop a shadow; calls=%v", f.calls)
	}
	if !strings.Contains(res.Action, "restored") {
		t.Errorf("action = %q", res.Action)
	}
}

// ---- gc ------------------------------------------------------------------

// listJSON renders a `recovery list` fixture the way the producer does: the
// typed vrooli.cli.v1 contract marshaled with UseProtoNames (snake_case).
func listJSON(views ...engagementView) []byte {
	out := &cliv1.RecoveryListOutput{}
	for _, v := range views {
		out.Engagements = append(out.Engagements, viewProto(v))
	}
	b, _ := protojson.MarshalOptions{UseProtoNames: true}.Marshal(out)
	return b
}

// showJSON renders a `recovery show` fixture for one engagement view.
func showJSON(v engagementView) []byte {
	b, _ := protojson.MarshalOptions{UseProtoNames: true}.Marshal(viewProto(v))
	return b
}

func viewProto(v engagementView) *cliv1.RecoveryEngagementView {
	eng := &cliv1.RecoveryEngagementView{
		Scenario:           v.Scenario,
		Slug:               v.Slug,
		Mode:               v.Mode,
		Variant:            v.Variant,
		ShadowInstanceKey:  v.ShadowInstanceKey,
		AnchorBaselineName: v.AnchorBaselineName,
		AmbientVar:         v.AmbientVar,
		Ttl:                v.TTL,
		Expired:            v.Expired,
	}
	if v.ExpiresAt != nil {
		eng.ExpiresAt = v.ExpiresAt.Format(time.RFC3339Nano)
	}
	if v.CreatedAt != nil {
		eng.CreatedAt = v.CreatedAt.Format(time.RFC3339Nano)
	}
	if v.LastTouchedAt != nil {
		eng.LastTouchedAt = v.LastTouchedAt.Format(time.RFC3339Nano)
	}
	return eng
}

func TestGCReapsOnlyExpiredByDefault(t *testing.T) {
	f := newFakeRunner(t)
	f.stdout["recovery list"] = listJSON(
		engagementView{Scenario: "a", Slug: "wip", Mode: "shadow", Variant: "shadow", Expired: true},
		engagementView{Scenario: "b", Slug: "wip", Mode: "shadow", Variant: "shadow", Expired: false},
	)
	defer f.install()()

	res, err := gcEngagements(context.Background(), false)
	if err != nil {
		t.Fatalf("gcEngagements: %v", err)
	}
	if len(res.Reaped) != 1 || res.Reaped[0] != "a/wip" {
		t.Fatalf("reaped = %v, want [a/wip]", res.Reaped)
	}
	if len(res.Skipped) != 1 || res.Skipped[0] != "b/wip" {
		t.Fatalf("skipped = %v, want [b/wip]", res.Skipped)
	}
	if !f.sawCommand("scenario stop a --instance shadow") {
		t.Errorf("gc must stop the expired shadow; calls=%v", f.calls)
	}
	if f.sawCommand("scenario stop b") {
		t.Errorf("gc must not touch the live (non-expired) engagement; calls=%v", f.calls)
	}
}

func TestGCRefusesALiveServingSplitWithoutForce(t *testing.T) {
	f := newFakeRunner(t)
	f.stdout["recovery list"] = listJSON(
		engagementView{Scenario: "a", Slug: "wip", Mode: "shadow", Variant: "shadow", Expired: true},
		engagementView{Scenario: "b", Slug: "wip", Mode: "shadow", Variant: "shadow", Expired: true},
	)
	f.failOn["recovery clean --scenario a --slug wip"] = fmt.Errorf("vrooli recovery clean: exit status 1: recovery: an instance runs from the engagement: a runs from /cache/a/baseline-wip/serving")
	defer f.install()()

	res, err := gcEngagements(context.Background(), false)
	if err != nil {
		t.Fatalf("gcEngagements: %v", err)
	}
	if len(res.Refused) != 1 || res.Refused[0] != "a/wip" {
		t.Fatalf("refused = %v, want [a/wip]", res.Refused)
	}
	if len(res.Reaped) != 1 || res.Reaped[0] != "b/wip" {
		t.Fatalf("reaped = %v, want [b/wip]", res.Reaped)
	}
	if f.sawCommand("--force") {
		t.Errorf("gc without --force must not pass the override; calls=%v", f.calls)
	}
}

func TestGCForceReapsAll(t *testing.T) {
	f := newFakeRunner(t)
	f.stdout["recovery list"] = listJSON(
		engagementView{Scenario: "a", Slug: "wip", Mode: "live", Variant: "live", Expired: false},
		engagementView{Scenario: "b", Slug: "wip", Mode: "shadow", Variant: "shadow", Expired: false},
	)
	defer f.install()()

	res, err := gcEngagements(context.Background(), true)
	if err != nil {
		t.Fatalf("gcEngagements: %v", err)
	}
	if len(res.Reaped) != 2 {
		t.Fatalf("force should reap all, got %v", res.Reaped)
	}
	// Only the shadow gets a scenario stop; the live engagement is clean-only.
	if f.sawCommand("scenario stop a") {
		t.Errorf("live engagement should not be stopped; calls=%v", f.calls)
	}
	if !f.sawCommand("scenario stop b --instance shadow") {
		t.Errorf("shadow engagement should be stopped; calls=%v", f.calls)
	}
	if !f.sawCommand("recovery clean --scenario b --slug wip --force") {
		t.Errorf("gc --force must pass the override that moves live off a serving split; calls=%v", f.calls)
	}
}
