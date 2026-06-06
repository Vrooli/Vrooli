package lifecycle

import (
	"context"
	"testing"

	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// TestRunnerStartShadowInstanceIsolatedFromLive is the unit-level form of the
// P1 spike gate: a live instance and a `@shadow` instance of the same scenario
// run concurrently on distinct ports as distinct registry instances, and
// stopping the shadow leaves the live instance and its port claims fully intact
// (the reap-sibling regression). It also proves the `name@variant` argument is
// equivalent to the `--instance` flag and that the variant-keyed advisory lock
// lets the two variants start without serializing into ErrScenarioBusy.
func TestRunnerStartShadowInstanceIsolatedFromLive(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixture(t, root, "alpha")

	runner := newLifecycleRunnerForTest(t, root, home, nil)
	ctx := context.Background()

	liveRes, err := runner.Start("alpha", StartOptions{})
	if err != nil {
		t.Fatalf("Start(alpha live): %v", err)
	}
	t.Cleanup(func() { _ = runner.Stop("alpha", StopOptions{}) })

	// "alpha@shadow" must be exactly equivalent to passing --instance shadow.
	shadowRes, err := runner.Start("alpha@shadow", StartOptions{})
	if err != nil {
		t.Fatalf("Start(alpha@shadow): %v", err)
	}
	shadowStopped := false
	t.Cleanup(func() {
		if !shadowStopped {
			_ = runner.Stop("alpha", StopOptions{Variant: "shadow"})
		}
	})

	// Distinct first-choice ports from the variant-aware CRC seed.
	if liveRes.AllocatedPorts["api"] == 0 || shadowRes.AllocatedPorts["api"] == 0 {
		t.Fatalf("missing api ports: live=%v shadow=%v", liveRes.AllocatedPorts, shadowRes.AllocatedPorts)
	}
	if liveRes.AllocatedPorts["api"] == shadowRes.AllocatedPorts["api"] {
		t.Fatalf("live and shadow share api port %d; expected distinct", liveRes.AllocatedPorts["api"])
	}

	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer store.Close()

	liveInst := mustSingleRunningInstance(t, store, "alpha", scenarioruntime.DefaultVariant)
	shadowInst := mustSingleRunningInstance(t, store, "alpha", "shadow")
	if liveInst.InstanceID == shadowInst.InstanceID {
		t.Fatal("live and shadow resolved to the same registry instance")
	}
	if liveInst.Variant != scenarioruntime.DefaultVariant || shadowInst.Variant != "shadow" {
		t.Fatalf("variants = (%q, %q), want (live, shadow)", liveInst.Variant, shadowInst.Variant)
	}

	liveClaimsBefore := activeClaimCount(t, store, liveInst.InstanceID)
	if liveClaimsBefore == 0 {
		t.Fatal("live instance has no active port claims before shadow stop")
	}

	// The reap-sibling regression: stopping the shadow must NOT mark or release
	// the live instance's rows.
	if err := runner.Stop("alpha", StopOptions{Variant: "shadow"}); err != nil {
		t.Fatalf("Stop(alpha shadow): %v", err)
	}
	shadowStopped = true

	stillLive, err := store.GetInstance(ctx, liveInst.InstanceID)
	if err != nil {
		t.Fatalf("GetInstance(live): %v", err)
	}
	if stillLive.Status != scenarioruntime.StatusRunning {
		t.Fatalf("live status = %q after shadow stop, want running (reap-sibling)", stillLive.Status)
	}
	if got := activeClaimCount(t, store, liveInst.InstanceID); got != liveClaimsBefore {
		t.Fatalf("live active claims dropped from %d to %d after shadow stop (reap-sibling)", liveClaimsBefore, got)
	}

	stoppedShadow, err := store.GetInstance(ctx, shadowInst.InstanceID)
	if err != nil {
		t.Fatalf("GetInstance(shadow): %v", err)
	}
	if stoppedShadow.Status == scenarioruntime.StatusRunning {
		t.Fatalf("shadow status = %q after stop, want stopped", stoppedShadow.Status)
	}
}

func mustSingleRunningInstance(t *testing.T, store *scenarioruntime.SQLiteStore, scenario, variant string) scenarioruntime.Instance {
	t.Helper()
	instances, err := store.ListInstances(context.Background(), scenarioruntime.InstanceFilter{
		Scenario: scenario,
		Variant:  variant,
		Statuses: []string{scenarioruntime.StatusRunning},
	})
	if err != nil {
		t.Fatalf("ListInstances(%s@%s): %v", scenario, variant, err)
	}
	if len(instances) != 1 {
		t.Fatalf("running instances for %s@%s = %d, want exactly one", scenario, variant, len(instances))
	}
	return instances[0]
}

func activeClaimCount(t *testing.T, store *scenarioruntime.SQLiteStore, instanceID string) int {
	t.Helper()
	claims, err := store.ListPortClaims(context.Background(), scenarioruntime.PortClaimFilter{
		InstanceID: instanceID,
		Statuses:   scenarioruntime.ActivePortClaimStatuses(),
	})
	if err != nil {
		t.Fatalf("ListPortClaims(%s): %v", instanceID, err)
	}
	return len(claims)
}
