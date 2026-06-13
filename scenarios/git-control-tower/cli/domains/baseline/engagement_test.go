package baseline

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/encoding/protojson"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
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
}

func newFakeRunner(_ *testing.T) *fakeRunner {
	return &fakeRunner{stdout: map[string][]byte{}, failOn: map[string]error{}}
}

func (f *fakeRunner) run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, recordedCall{name: name, args: append([]string(nil), args...)})
	joined := name + " " + strings.Join(args, " ")
	for k, err := range f.failOn {
		if strings.Contains(joined, k) {
			return nil, err
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
	diffAnchor = func(_ *cliapp.ScenarioApp, _ context.Context, _, _, _ string) (string, error) {
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

// ---- shadow data population ----------------------------------------------

// withNoSleep stubs the poll-delay seam so the wait loop runs without real time.
func withNoSleep(t *testing.T) func() {
	t.Helper()
	prev := sleepFn
	sleepFn = func(time.Duration) {}
	return func() { sleepFn = prev }
}

// seedPopulationStdout primes the fakeRunner with the canned data-substrate +
// floor responses a successful shadow data population reads.
func seedPopulationStdout(f *fakeRunner, registered, runStatus, postgresDB, dataDir string) {
	f.stdout["safety register-targets"] = []byte(`{"registered":[` + registered + `]}`)
	f.stdout["safety backup-now"] = []byte(`{"runId":"run-123","status":"RUN_STATUS_PENDING"}`)
	f.stdout["runs get"] = []byte(`{"run":{"status":"` + runStatus + `"}}`)
	f.stdout["recovery namespace"] = []byte(`{"postgresDb":"` + postgresDB + `","dataDir":"` + dataDir + `"}`)
	f.stdout["safety populate-shadow"] = []byte(`{}`)
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

	res, err := checkEngagement(nil, "demo-scenario", "wip", "")
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

	res, err := checkEngagement(nil, "demo-scenario", "wip", "")
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

	if _, err := checkEngagement(nil, "demo-scenario", "wip", ""); err == nil || !strings.Contains(err.Error(), "no anchor") {
		t.Fatalf("expected no-anchor error, got %v", err)
	}
}

// ---- abandon -------------------------------------------------------------

func TestAbandonShadowDiscardsCandidateLeavesLiveUntouched(t *testing.T) {
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
	// Live served the baseline from the copy throughout — it is never restarted.
	if f.sawCommand("scenario restart") {
		t.Errorf("shadow abandon must NOT restart live; calls=%v", f.calls)
	}
	if !f.sawCommand("recovery clean --scenario demo-scenario --slug wip") {
		t.Errorf("abandon must clean the engagement; calls=%v", f.calls)
	}
	if !strings.Contains(res.Action, "live untouched") {
		t.Errorf("action = %q", res.Action)
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
		out.Engagements = append(out.Engagements, eng)
	}
	b, _ := protojson.MarshalOptions{UseProtoNames: true}.Marshal(out)
	return b
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
}
