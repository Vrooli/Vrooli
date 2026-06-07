package baseline

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// quiesceJSON builds a protojson-shaped QuiesceScenarioResponse for the fake
// runner to return from `agent-manager run quiesce`.
func quiesceJSON(drained, aborted bool, reason string, inFlight ...string) []byte {
	type ref struct {
		ID string `json:"id"`
	}
	result := map[string]any{"drained": drained, "aborted": aborted, "reason": reason}
	if len(inFlight) > 0 {
		refs := make([]ref, 0, len(inFlight))
		for _, id := range inFlight {
			refs = append(refs, ref{ID: id})
		}
		result["inFlight"] = refs
	}
	b, _ := json.Marshal(map[string]any{"result": result})
	return b
}

// statusJSON builds a `vrooli scenario status --json` payload with the given
// lifecycle status under the scenario-wrapped shape.
func statusJSON(status string) []byte {
	b, _ := json.Marshal(map[string]any{"scenario": map[string]any{"status": status}})
	return b
}

// installShadowEngagement wires recovery show + a drained quiesce + a running
// probe so a promote happy-path can run end to end. Returns the fake runner.
func installShadowEngagement(t *testing.T) (*fakeRunner, func()) {
	t.Helper()
	f := newFakeRunner(t)
	f.stdout["recovery show"] = engagementJSON("shadow", "shadow", "engagement-wip")
	f.stdout["run quiesce"] = quiesceJSON(true, false, "live drained")
	f.stdout["recovery migrate"] = []byte(`{"fastPath":true}`)
	f.stdout["scenario status"] = statusJSON("running")
	return f, f.install()
}

func TestPromoteShadowHappyPath(t *testing.T) {
	f, restore := installShadowEngagement(t)
	defer restore()

	res, err := promoteEngagement(nil, promoteParams{scenario: "demo-scenario", slug: "wip"})
	if err != nil {
		t.Fatalf("promoteEngagement: %v", err)
	}
	if !res.Promoted || res.RolledBack {
		t.Fatalf("expected a clean promote, got %+v", res)
	}
	// The full §8 sequence must have run in order.
	want := []string{
		"agent-manager run quiesce --scenario demo-scenario",
		"data-backup-manager safety backup-now --scenario demo-scenario",
		"vrooli recovery migrate --scenario demo-scenario --slug wip --json",
		"vrooli recovery set-mode --scenario demo-scenario --slug wip --mode live",
		"vrooli scenario restart demo-scenario",
		"vrooli scenario status demo-scenario --json",
		"vrooli scenario stop demo-scenario --instance shadow",
		"vrooli recovery clean --scenario demo-scenario --slug wip",
	}
	for _, w := range want {
		if !f.sawCommand(w) {
			t.Errorf("missing step %q; calls=%v", w, f.calls)
		}
	}
	// The re-point (collapse the split) MUST happen before the restart, or the
	// restart would relaunch live from the frozen baseline copy.
	if !f.sawInOrder("recovery set-mode --scenario demo-scenario --slug wip --mode live", "scenario restart demo-scenario") {
		t.Errorf("re-point (set-mode live) must precede the live restart; calls=%v", f.calls)
	}
	// A clean promote never restores the working tree (the candidate is what we keep).
	if f.sawCommand("recovery restore") {
		t.Errorf("happy-path promote must NOT restore the working tree; calls=%v", f.calls)
	}
	// The drain must carry a timeout (default) and JSON.
	if !f.sawCommand("run quiesce --scenario demo-scenario --json --timeout 5m0s") {
		t.Errorf("quiesce should default the timeout + request JSON; calls=%v", f.calls)
	}
}

func TestPromoteDrainAbortsLeavesEverythingUntouched(t *testing.T) {
	f := newFakeRunner(t)
	f.stdout["recovery show"] = engagementJSON("shadow", "shadow", "engagement-wip")
	f.stdout["run quiesce"] = quiesceJSON(false, true, "2 runs in-flight; retry or --force", "run-1", "run-2")
	defer f.install()()

	res, err := promoteEngagement(nil, promoteParams{scenario: "demo-scenario", slug: "wip"})
	if err == nil {
		t.Fatal("expected an abort error when the drain does not drain")
	}
	if res.Promoted {
		t.Errorf("promote must not succeed on an aborted drain, got %+v", res)
	}
	// Critically: nothing past the drain may have run — live is untouched.
	if f.sawCommand("scenario restart") {
		t.Errorf("aborted promote must NOT restart live; calls=%v", f.calls)
	}
	if f.sawCommand("safety backup-now") {
		t.Errorf("aborted promote must NOT snapshot; calls=%v", f.calls)
	}
	if f.sawCommand("recovery clean") {
		t.Errorf("aborted promote must NOT clean the engagement (retryable); calls=%v", f.calls)
	}
	if !strings.Contains(err.Error(), "in-flight") {
		t.Errorf("abort error should mention in-flight runs: %v", err)
	}
}

func TestPromoteProbeFailureAutoRollsBack(t *testing.T) {
	f := newFakeRunner(t)
	f.stdout["recovery show"] = engagementJSON("shadow", "shadow", "engagement-wip")
	f.stdout["run quiesce"] = quiesceJSON(true, false, "live drained")
	f.stdout["scenario status"] = statusJSON("stopped") // probe fails
	defer f.install()()

	res, err := promoteEngagement(nil, promoteParams{scenario: "demo-scenario", slug: "wip"})
	if err == nil {
		t.Fatal("expected a rollback error when the probe fails")
	}
	if res.Promoted {
		t.Errorf("a rolled-back promote is not promoted, got %+v", res)
	}
	if !res.RolledBack {
		t.Errorf("expected RolledBack=true, got %+v", res)
	}
	// Auto-rollback = re-open the shadow split (flip back to shadow mode) + restart;
	// the resolver then routes live back to the frozen baseline copy. No working-tree
	// restore is performed — the working tree legitimately keeps the candidate.
	if !f.sawCommand("recovery set-mode --scenario demo-scenario --slug wip --mode shadow") {
		t.Errorf("rollback must re-point live back to the baseline (set-mode shadow); calls=%v", f.calls)
	}
	if f.sawCommand("recovery restore") {
		t.Errorf("rollback must NOT restore the working tree (candidate is kept); calls=%v", f.calls)
	}
	// The shadow is left standing for diagnosis (no teardown on rollback).
	if f.sawCommand("scenario stop demo-scenario --instance shadow") {
		t.Errorf("rollback must NOT tear down the validated shadow; calls=%v", f.calls)
	}
	// The engagement stays open (no clean) so it can be retried.
	if f.sawCommand("recovery clean") {
		t.Errorf("rolled-back promote must NOT clean the engagement; calls=%v", f.calls)
	}
}

func TestPromoteMigrationBounceAbortsBeforeRestart(t *testing.T) {
	f := newFakeRunner(t)
	f.stdout["recovery show"] = engagementJSON("shadow", "shadow", "engagement-wip")
	f.stdout["run quiesce"] = quiesceJSON(true, false, "live drained")
	// The migration runner bounces (e.g. dry-run found the script incompatible
	// with the current live shape) — a non-zero exit.
	f.failOn["recovery migrate"] = errors.New("migration dry-run failed (live untouched)")
	defer f.install()()

	res, err := promoteEngagement(nil, promoteParams{scenario: "demo-scenario", slug: "wip"})
	if err == nil {
		t.Fatal("a migration bounce must hard-stop the promote")
	}
	if res.Promoted || res.RolledBack {
		t.Errorf("a bounced promote neither promotes nor rolls back (live never changed), got %+v", res)
	}
	// Live must be untouched: no restart, and the engagement stays open (no clean).
	if f.sawCommand("scenario restart") {
		t.Errorf("promote must NOT restart after a migration bounce; calls=%v", f.calls)
	}
	if f.sawCommand("recovery clean") {
		t.Errorf("a bounced promote must NOT clean the engagement (retryable); calls=%v", f.calls)
	}
	if !strings.Contains(err.Error(), "migration") {
		t.Errorf("the abort error should name the migration bounce: %v", err)
	}
}

func TestPromoteLiveAcceptsInPlace(t *testing.T) {
	f := newFakeRunner(t)
	f.stdout["recovery show"] = engagementJSON("live", "live", "engagement-wip")
	defer f.install()()

	res, err := promoteEngagement(nil, promoteParams{scenario: "demo-scenario", slug: "wip"})
	if err != nil {
		t.Fatalf("promoteEngagement (live): %v", err)
	}
	if !res.Promoted {
		t.Errorf("live promote should succeed, got %+v", res)
	}
	// Live accept = drop the safety net only. No drain, no restart, no shadow.
	if f.sawCommand("run quiesce") || f.sawCommand("scenario restart") || f.sawCommand("scenario stop") {
		t.Errorf("live accept must not drain/restart/stop; calls=%v", f.calls)
	}
	if !f.sawCommand("recovery clean --scenario demo-scenario --slug wip") {
		t.Errorf("live accept must drop the restore point + manifest; calls=%v", f.calls)
	}
}

func TestPromoteNoDrainSkipsQuiesce(t *testing.T) {
	f := newFakeRunner(t)
	f.stdout["recovery show"] = engagementJSON("shadow", "shadow", "engagement-wip")
	f.stdout["scenario status"] = statusJSON("running")
	defer f.install()()

	res, err := promoteEngagement(nil, promoteParams{scenario: "demo-scenario", slug: "wip", noDrain: true})
	if err != nil {
		t.Fatalf("promoteEngagement (--no-drain): %v", err)
	}
	if !res.Promoted {
		t.Errorf("expected promote with --no-drain, got %+v", res)
	}
	if f.sawCommand("run quiesce") {
		t.Errorf("--no-drain must skip the quiesce; calls=%v", f.calls)
	}
	if res.Drained {
		t.Errorf("Drained should be false when the drain is skipped, got %+v", res)
	}
}

func TestPromoteThreadsDrainFlags(t *testing.T) {
	f, restore := installShadowEngagement(t)
	defer restore()

	if _, err := promoteEngagement(nil, promoteParams{
		scenario: "demo-scenario", slug: "wip",
		excludeRun: "run-self", tagPrefix: "ecosystem-", scopePrefix: "scenarios/demo-scenario", force: true,
	}); err != nil {
		t.Fatalf("promoteEngagement: %v", err)
	}
	for _, frag := range []string{
		"--exclude-run run-self", "--tag-prefix ecosystem-", "--scope-prefix scenarios/demo-scenario", "--force",
	} {
		if !f.sawCommand(frag) {
			t.Errorf("drain should carry %q; calls=%v", frag, f.calls)
		}
	}
}

func TestPromoteDrainHardErrorStopsPromote(t *testing.T) {
	f := newFakeRunner(t)
	f.stdout["recovery show"] = engagementJSON("shadow", "shadow", "engagement-wip")
	// The self-deadlock guard rejects with a non-zero exit — a hard error.
	f.failOn["run quiesce"] = context.DeadlineExceeded
	defer f.install()()

	_, err := promoteEngagement(nil, promoteParams{scenario: "demo-scenario", slug: "wip"})
	if err == nil {
		t.Fatal("a failed drain command must hard-stop the promote")
	}
	if f.sawCommand("scenario restart") {
		t.Errorf("promote must not proceed after a drain command failure; calls=%v", f.calls)
	}
}

func TestPromoteRequiresScenario(t *testing.T) {
	if _, err := promoteEngagement(nil, promoteParams{}); err == nil {
		t.Fatal("expected a missing-scenario error")
	}
}

func TestParseQuiesce(t *testing.T) {
	q := parseQuiesce(quiesceJSON(true, false, "ok"))
	if !q.drained || q.aborted {
		t.Errorf("drained quiesce mis-parsed: %+v", q)
	}
	q2 := parseQuiesce(quiesceJSON(false, true, "stuck", "a", "b"))
	if q2.drained || !q2.aborted || len(q2.inFlight) != 2 {
		t.Errorf("aborted quiesce mis-parsed: %+v", q2)
	}
	if parseQuiesce([]byte("not json")).drained {
		t.Errorf("garbage must not parse as drained")
	}
}

func TestExtractStatus(t *testing.T) {
	cases := map[string]string{
		`{"scenario":{"status":"running"}}`: "running",
		`{"details":{"status":"stopped"}}`:  "stopped",
		`{"status":"running"}`:              "running",
		`not json`:                          "",
		`{}`:                                "",
	}
	for in, want := range cases {
		if got := extractStatus([]byte(in)); got != want {
			t.Errorf("extractStatus(%s) = %q, want %q", in, got, want)
		}
	}
}
