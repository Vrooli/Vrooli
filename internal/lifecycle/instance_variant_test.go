package lifecycle

import (
	"context"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"

	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
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
	cleanupRunner(t, runner, "alpha", StopOptions{})

	// "alpha@shadow" must be exactly equivalent to passing --instance shadow.
	shadowRes, err := runner.Start("alpha@shadow", StartOptions{})
	if err != nil {
		t.Fatalf("Start(alpha@shadow): %v", err)
	}
	shadowStopped := false
	t.Cleanup(func() {
		if !shadowStopped {
			if err := runner.Stop("alpha", StopOptions{Variant: "shadow"}); err != nil {
				t.Errorf("Stop(alpha@shadow) during cleanup: %v", err)
			}
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

// TestStopVariantLeavesLiveFixedPortListener is the stop-side twin of
// cleanupFixedPortOrphans' live-only rule. Stopping web-console@presentation
// killed live web-console's UI because the UI port is fixed in the manifest
// and the stop path cleaned every listener on it.
func TestStopVariantLeavesLiveFixedPortListener(t *testing.T) {
	const fixedUIPort = 36235
	const livePID = 4242

	for _, tc := range []struct {
		variant    string
		wantSignal bool
	}{
		{variant: "presentation", wantSignal: false},
		{variant: "", wantSignal: true},
	} {
		t.Run("variant="+tc.variant, func(t *testing.T) {
			root := t.TempDir()
			home := t.TempDir()
			manifest := lifecycleFixtureManifest("alpha")
			manifest.Ports["ui"] = scenario.Port{EnvVar: "UI_PORT", Port: intPtr(fixedUIPort)}
			writeLifecycleFixtureManifest(t, root, manifest)

			listening := true
			var signaled []int
			runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
				deps.readScenarioRecords = func(string, string) ([]process.Record, error) { return nil, nil }
				deps.listeningPIDs = func(port int) ([]int, error) {
					if port == fixedUIPort && listening {
						return []int{livePID}, nil
					}
					return nil, nil
				}
				deps.signalPID = func(pid int, _ bool) error {
					signaled = append(signaled, pid)
					listening = false
					return nil
				}
				deps.isPIDRunning = func(int) bool { return false }
			})

			if err := runner.cleanupScenarioRuntimeWithRegistryContext(context.Background(), "alpha", tc.variant, "", true, false, false); err != nil {
				t.Fatalf("cleanup(alpha@%s): %v", tc.variant, err)
			}
			if got := len(signaled) > 0; got != tc.wantSignal {
				t.Fatalf("alpha@%s stop signaled fixed-port listener = %v (%v), want %v", tc.variant, got, signaled, tc.wantSignal)
			}
		})
	}
}

// TestVariantBuildRefusedWhileLiveServesSharedOutputs covers the other half of
// the presentation-instance incident: starting web-console@presentation rebuilt
// the API binary and UI bundle in the working tree that live web-console serves.
func TestVariantBuildRefusedWhileLiveServesSharedOutputs(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixture(t, root, "alpha")
	runner := newLifecycleRunnerForTest(t, root, home, nil)

	if _, err := runner.Start("alpha", StartOptions{}); err != nil {
		t.Fatalf("Start(alpha live): %v", err)
	}
	cleanupRunner(t, runner, "alpha", StopOptions{})

	_, err := runner.Start("alpha@presentation", StartOptions{ForceSetup: true})
	if err == nil {
		_ = runner.Stop("alpha", StopOptions{Variant: "presentation"})
		t.Fatal("variant rebuild was admitted while live alpha serves the shared build outputs")
	}
	if !strings.Contains(err.Error(), "shares build outputs with running instance(s) alpha") {
		t.Fatalf("variant rebuild error = %v, want the shared-build-output refusal", err)
	}
	_ = runner.Stop("alpha", StopOptions{Variant: "presentation"})

	// With fresh outputs the variant needs no build and starts beside live.
	if _, err := runner.Start("alpha@presentation", StartOptions{}); err != nil {
		t.Fatalf("Start(alpha@presentation) with fresh outputs: %v", err)
	}
	if err := runner.Stop("alpha", StopOptions{Variant: "presentation"}); err != nil {
		t.Fatalf("Stop(alpha@presentation): %v", err)
	}
}

// TestApplyVariantDependenciesPublishesFollowList covers the #4 routing switch:
// a non-live instance publishes the list discovery reads, a live instance
// refuses it, and an undeclared dependency is rejected before start.
func TestApplyVariantDependenciesPublishesFollowList(t *testing.T) {
	item := scenario.Scenario{
		Slug:    "web-console",
		Variant: "presentation",
		Manifest: scenario.ServiceManifest{Dependencies: scenario.Dependencies{Scenarios: map[string]scenario.Dependency{
			"vrooli-bridge": {}, "audio-tools": {},
		}}},
	}

	envVars := map[string]string{}
	if err := applyVariantDependencies(item, []string{"audio-tools", "vrooli-bridge"}, envVars); err != nil {
		t.Fatalf("applyVariantDependencies: %v", err)
	}
	if got := envVars[cliutil.EnvVariantDependencies]; got != "audio-tools,vrooli-bridge" {
		t.Fatalf("published follow list = %q", got)
	}

	if err := applyVariantDependencies(item, []string{"integration-hub"}, map[string]string{}); err == nil {
		t.Error("undeclared dependency was accepted")
	}

	live := item
	live.Variant = "live"
	if err := applyVariantDependencies(live, []string{"audio-tools"}, map[string]string{}); err == nil {
		t.Error("live instance accepted a variant dependency list")
	}

	empty := map[string]string{}
	if err := applyVariantDependencies(item, nil, empty); err != nil || len(empty) != 0 {
		t.Errorf("no list should publish nothing: err=%v env=%v", err, empty)
	}
}
