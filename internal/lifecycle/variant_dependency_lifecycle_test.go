package lifecycle

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/runtimesupervisor"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func TestShadowVariantDependencyOverrideDoesNotLeakIntoStaleLiveDependency(t *testing.T) {
	ctx := context.Background()
	root, home := t.TempDir(), t.TempDir()
	marker := t.TempDir() + "/parent-variant-dependencies"

	writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "parent"},
		Dependencies: scenario.Dependencies{Scenarios: map[string]scenario.Dependency{
			"dep": {Enabled: true, Required: true},
		}},
		Lifecycle: scenario.Lifecycle{Develop: scenario.Phase{Steps: []scenario.PhaseStep{{
			Name: "capture-variant-dependencies",
			Exec: []string{"bash", "-c", "printf '%s' \"$VROOLI_VARIANT_DEPENDENCIES\" > " + marker},
		}}}},
	})
	writeLifecycleFixture(t, root, "dep")

	t.Setenv(runtimesupervisor.ModeEnv, runtimesupervisor.ModeOff)
	runner := newLifecycleRunnerForTest(t, root, home, nil)
	if _, err := runner.StartContext(ctx, "dep", StartOptions{}); err != nil {
		t.Fatalf("start initial live dependency: %v", err)
	}
	cleanupRunner(t, runner, "dep", StopOptions{})
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("open runtime registry: %v", err)
	}
	oldDependency := mustSingleRunningInstance(t, store, "dep", "live")
	if err := store.Close(); err != nil {
		t.Fatalf("close runtime registry: %v", err)
	}

	result, err := runner.StartContext(ctx, "parent", StartOptions{
		Variant:             "shadow",
		VariantDependencies: []string{"dep"},
		ForceSetup:          true,
		ForceSetupScenario:  "dep",
	})
	if err != nil {
		t.Fatalf("start shadow consumer after stale live dependency replacement: %v", err)
	}
	cleanupRunner(t, runner, "parent", StopOptions{Variant: "shadow"})
	if result.Scenario.Variant != "shadow" {
		t.Fatalf("consumer variant = %q, want shadow", result.Scenario.Variant)
	}

	got, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read consumer's routed dependency list: %v", err)
	}
	if string(got) != "dep" {
		t.Fatalf("consumer VROOLI_VARIANT_DEPENDENCIES = %q, want dep", got)
	}
	store, err = scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("reopen runtime registry: %v", err)
	}
	defer store.Close()
	newDependency := mustSingleRunningInstance(t, store, "dep", "live")
	if newDependency.InstanceID == oldDependency.InstanceID {
		t.Fatalf("dependency instance %q was not replaced despite scoped forced setup", newDependency.InstanceID)
	}
}

func TestLiveVariantDependencyOverrideRefusesBeforeStartingDependencies(t *testing.T) {
	ctx := context.Background()
	root, home := t.TempDir(), t.TempDir()
	writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "parent"},
		Dependencies: scenario.Dependencies{Scenarios: map[string]scenario.Dependency{
			"dep": {Enabled: true, Required: true},
		}},
	})
	writeLifecycleFixture(t, root, "dep")
	t.Setenv(runtimesupervisor.ModeEnv, runtimesupervisor.ModeOff)
	runner := newLifecycleRunnerForTest(t, root, home, nil)

	_, err := runner.StartContext(ctx, "parent", StartOptions{VariantDependencies: []string{"dep"}})
	if err == nil || !strings.Contains(err.Error(), "require a non-live instance") {
		t.Fatalf("live start with variant dependency override error = %v, want non-live-instance refusal", err)
	}
	store, storeErr := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if storeErr != nil {
		t.Fatalf("open runtime registry: %v", storeErr)
	}
	defer store.Close()
	instances, listErr := store.ListInstances(ctx, scenarioruntime.InstanceFilter{Scenario: "dep"})
	if listErr != nil {
		t.Fatalf("list dependency instances: %v", listErr)
	}
	if len(instances) != 0 {
		t.Fatalf("invalid live override created dependency runtime state before refusal: %+v", instances)
	}
}

func TestLiveVariantDependencyOverrideRefusesBeforeRestartFenceAndStop(t *testing.T) {
	ctx := context.Background()
	runner, service, events := restartFenceTestRunner(t, "allow")
	runner.WithProgressSink(restartFenceProgressSink{service: service})
	marker := t.TempDir() + "/dependency-started"
	writeLifecycleFixtureManifest(t, runner.Root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "alpha"},
		Ports: map[string]scenario.Port{
			"api": {EnvVar: "API_PORT", Range: "22000-22010"},
		},
		Dependencies: scenario.Dependencies{Scenarios: map[string]scenario.Dependency{
			"dep": {Enabled: true, Required: true},
		}},
	})
	writeLifecycleFixtureManifest(t, runner.Root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "dep"},
		Lifecycle: scenario.Lifecycle{Develop: scenario.Phase{Steps: []scenario.PhaseStep{{
			Name: "mark-start", Exec: []string{"bash", "-c", "touch " + marker},
		}}}},
	})

	store, err := runner.runtimeDeps().runtimeRegistry(ctx, runner.Home)
	if err != nil {
		t.Fatalf("open runtime registry: %v", err)
	}
	before, err := store.ListInstances(ctx, scenarioruntime.InstanceFilter{Scenario: "alpha", Variant: "live", Statuses: scenarioruntime.ActiveInstanceStatuses()})
	if err != nil || len(before) != 1 {
		_ = store.Close()
		t.Fatalf("initial active target = %+v, err=%v; want one serving instance", before, err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close runtime registry: %v", err)
	}

	_, err = runner.RestartContext(ctx, "alpha", StartOptions{VariantDependencies: []string{"dep"}})
	if err == nil || !strings.Contains(err.Error(), "require a non-live instance") {
		t.Fatalf("restart with live variant dependency override error = %v, want non-live-instance refusal", err)
	}
	if got := snapshotRestartFenceEvents(service, events); len(got) != 0 {
		t.Fatalf("provider/stop events before invalid override refusal = %v, want none", got)
	}
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Fatalf("dependency side effect occurred before invalid restart refusal: stat error=%v", statErr)
	}

	store, err = runner.runtimeDeps().runtimeRegistry(ctx, runner.Home)
	if err != nil {
		t.Fatalf("reopen runtime registry: %v", err)
	}
	defer store.Close()
	after, err := store.ListInstances(ctx, scenarioruntime.InstanceFilter{Scenario: "alpha", Variant: "live", Statuses: scenarioruntime.ActiveInstanceStatuses()})
	if err != nil || len(after) != 1 || after[0].InstanceID != before[0].InstanceID || after[0].Status != scenarioruntime.StatusRunning {
		t.Fatalf("active target after refused restart = %+v, err=%v; want the same running owner %q", after, err, before[0].InstanceID)
	}
	dependencies, err := store.ListInstances(ctx, scenarioruntime.InstanceFilter{Scenario: "dep"})
	if err != nil || len(dependencies) != 0 {
		t.Fatalf("dependency runtime state after refused restart = %+v, err=%v; want none", dependencies, err)
	}
}
