package baseline

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// installCycleFloor wires a floor holding eng (nil: none) that a promote's
// clean empties, plus a drained quiesce and a running live — the world a cycle
// promotes and re-creates in.
func installCycleFloor(t *testing.T, eng *engagementView) *fakeRunner {
	t.Helper()
	f := newFakeRunner(t)
	f.failOn["scenario-dependency-analyzer"] = fmt.Errorf("down")
	if eng != nil {
		f.stdout["recovery show"] = showJSON(*eng)
		open := *eng
		f.dynamic["recovery list"] = func() []byte {
			if f.sawCommand("recovery clean --scenario " + open.Scenario + " --slug " + open.Slug) {
				return listJSON()
			}
			return listJSON(open)
		}
	}
	f.stdout["run quiesce"] = quiesceJSON(true, false, "live drained")
	f.stdout["recovery migrate"] = []byte(`{"fast_path":true}`)
	f.stdout["scenario status"] = statusJSON("running")
	t.Cleanup(f.install())
	return f
}

func cycleParams(slug string) promoteParams {
	return promoteParams{scenario: "demo-scenario", slug: slug, force: true, probeCmd: probeCmd}
}

func TestCyclePromotesThenRecreatesTheSameName(t *testing.T) {
	eng := openGoalEngagement(false)
	eng.TTL = "3h0m0s"
	f := installCycleFloor(t, &eng)
	snapCalled := withSnapshotSpy(t)

	res, err := cycleEngagement(nil, cycleParams("bas-goal"))
	if err != nil {
		t.Fatalf("cycleEngagement: %v\nsteps: %v", err, res.Steps)
	}
	if !res.Promoted || !res.Recreated || res.RolledBack {
		t.Fatalf("expected promoted + recreated, got %+v", res)
	}
	// Promote (with its live probe) completes before anything is re-captured.
	if !f.sawInOrder("sh -c "+probeCmd, "recovery clean --scenario demo-scenario --slug bas-goal") ||
		!f.sawInOrder("recovery clean --scenario demo-scenario --slug bas-goal", "recovery capture --scenario demo-scenario --slug bas-goal") {
		t.Errorf("cycle must probe, close, then re-capture; calls=%v", f.calls)
	}
	// The fresh engagement keeps the name and TTL, is a shadow, and has no anchor run.
	if !f.sawCommand("recovery write --scenario demo-scenario --slug bas-goal --mode shadow --ttl 3h0m0s") {
		t.Errorf("re-create must write the same slug, shadow mode and TTL; calls=%v", f.calls)
	}
	if f.sawCommand("--anchor") || *snapCalled {
		t.Errorf("re-create must not capture an anchor; calls=%v", f.calls)
	}
	if !f.sawInOrder("recovery write --scenario demo-scenario --slug bas-goal", "scenario start demo-scenario --instance shadow") {
		t.Errorf("re-create must stand the shadow back up; calls=%v", f.calls)
	}
}

func TestCycleRollbackStartsNothing(t *testing.T) {
	eng := openGoalEngagement(false)
	f := installCycleFloor(t, &eng)
	f.failOn["sh -c"] = errors.New("exit status 1: journey J1 failed")

	res, err := cycleEngagement(nil, cycleParams("bas-goal"))
	if err == nil || !res.RolledBack || res.Promoted || res.Recreated {
		t.Fatalf("a failed probe must roll back and stop the cycle, got res=%+v err=%v", res, err)
	}
	if f.sawCommand("recovery capture") || f.sawCommand("recovery write") || f.sawCommand("scenario start") {
		t.Errorf("a rolled-back cycle must not re-create anything; calls=%v", f.calls)
	}
	if !strings.Contains(res.Next, "re-run `git-control-tower baseline cycle --scenario demo-scenario --name bas-goal`") {
		t.Errorf("the summary should say how to retry: %q", res.Next)
	}
}

func TestCycleRetryAfterPromoteOnlyRecreates(t *testing.T) {
	f := installCycleFloor(t, nil) // an earlier cycle promoted; the re-create did not finish

	res, err := cycleEngagement(nil, cycleParams("bas-goal"))
	if err != nil {
		t.Fatalf("cycleEngagement: %v", err)
	}
	if res.Promoted || !res.Recreated {
		t.Fatalf("a retry with nothing open must only re-create, got %+v", res)
	}
	for _, step := range []string{"run quiesce", "recovery set-mode", "scenario restart", "sh -c"} {
		if f.sawCommand(step) {
			t.Errorf("a retry must not promote again (%q ran); calls=%v", step, f.calls)
		}
	}
	if !f.sawCommand("recovery capture --scenario demo-scenario --slug bas-goal") {
		t.Errorf("a retry must re-create the engagement; calls=%v", f.calls)
	}
}

func TestCycleShadowStandUpFailureRemovesTheHalfCreatedEngagement(t *testing.T) {
	eng := openGoalEngagement(false)
	f := installCycleFloor(t, &eng)
	f.failOn["scenario start demo-scenario --instance shadow"] = errors.New("port allocation failed")

	res, err := cycleEngagement(nil, cycleParams("bas-goal"))
	if err == nil || !res.Promoted || res.Recreated {
		t.Fatalf("expected promoted but not re-created, got res=%+v err=%v", res, err)
	}
	// The engagement the failed start wrote is dropped, so the next cycle
	// re-creates it instead of promoting it again.
	writeAt := f.firstIndex("recovery write --scenario demo-scenario --slug bas-goal")
	cleaned := false
	for i := writeAt + 1; writeAt >= 0 && i < len(f.calls); i++ {
		c := f.calls[i]
		if c.name+" "+strings.Join(c.args, " ") == "vrooli recovery clean --scenario demo-scenario --slug bas-goal" {
			cleaned = true
		}
	}
	if !cleaned {
		t.Errorf("the half-created engagement must be cleaned after its write; calls=%v", f.calls)
	}
	if !strings.Contains(res.Next, "without promoting again") {
		t.Errorf("the summary should say a retry only re-creates: %q", res.Next)
	}
}

func TestCycleRefusesAnotherName(t *testing.T) {
	eng := openGoalEngagement(false)
	f := installCycleFloor(t, &eng)

	res, err := cycleEngagement(nil, cycleParams("wip"))
	if err == nil || !strings.Contains(res.Message, "did you mean --name bas-goal?") {
		t.Fatalf("cycle under another name must be refused with the open name, got res=%+v err=%v", res, err)
	}
	if f.sawCommand("run quiesce") || f.sawCommand("recovery capture") {
		t.Errorf("a refused cycle must not promote or capture; calls=%v", f.calls)
	}
}

func TestCycleRefusesALiveEngagement(t *testing.T) {
	eng := openGoalEngagement(false)
	eng.Mode, eng.Variant = modeLive, modeLive
	f := installCycleFloor(t, &eng)

	res, err := cycleEngagement(nil, cycleParams("bas-goal"))
	if err == nil || !strings.Contains(res.Message, "live engagement") {
		t.Fatalf("cycle must refuse a live engagement, got res=%+v err=%v", res, err)
	}
	if f.sawCommand("recovery clean") || f.sawCommand("recovery capture") {
		t.Errorf("a refused cycle must not touch the floor; calls=%v", f.calls)
	}
}

func TestCycleRequiresAnExplicitName(t *testing.T) {
	f := installCycleFloor(t, nil)
	err := runCycleCmd(nil, []string{"--scenario", "demo-scenario"})
	if err == nil || !strings.Contains(err.Error(), "--name is required") {
		t.Fatalf("cycle without --name must be refused (a default would open a second engagement), got %v", err)
	}
	if len(f.calls) != 0 {
		t.Errorf("a refused cycle must run nothing; calls=%v", f.calls)
	}
}

func TestCycleSummaryNamesTheOutcomeAndTheNextStep(t *testing.T) {
	eng := openGoalEngagement(false)
	f := installCycleFloor(t, &eng)
	f.failOn["sh -c"] = errors.New("exit status 1: journey J1 failed")
	f.stdout["safety backup-now"] = backupNowJSON("run-5")
	f.stdout["runs get"] = []byte(`{"run":{"status":"RUN_STATUS_COMPLETED"}}`)

	res, _ := cycleEngagement(nil, cycleParams("bas-goal"))
	var out bytes.Buffer
	writeCycleSummary(&out, res)
	for _, want := range []string{
		"✗ cycle demo-scenario/bas-goal: promote rolled back",
		"  · promote: ✗ live probe failed: exit status 1: journey J1 failed — auto-rolling back",
		"  data snapshot for manual restore: run-5",
		"  next: fix the cause, then re-run",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("summary should contain %q:\n%s", want, out.String())
		}
	}
}
