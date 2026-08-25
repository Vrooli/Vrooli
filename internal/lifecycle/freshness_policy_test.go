package lifecycle

import (
	"context"
	"io"
	"testing"

	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func freshnessPolicyDecision(policy string) dependencyDecision {
	return dependencyDecision{
		policy:          scenario.DependencyStartupPolicyMustStart,
		freshnessPolicy: policy,
	}
}

func TestApplyFreshnessPolicyReuseRunningKeepsProcess(t *testing.T) {
	r := &Runner{Out: io.Discard, Err: io.Discard, Verbosity: VerbosityQuiet}
	dep := scenario.Scenario{Slug: "beta"}
	handled, err := r.applyDependencyFreshnessPolicy("alpha", dep,
		freshnessPolicyDecision(scenario.DependencyFreshnessPolicyReuseRunning),
		registryRuntimeView{}, []string{"main.go content changed"})
	if err != nil {
		t.Fatalf("applyDependencyFreshnessPolicy: %v", err)
	}
	if !handled {
		t.Fatal("reuse_running must keep the running process (handled=true)")
	}
}

func TestApplyFreshnessPolicyRestartWhenStaleNoConsumersProceeds(t *testing.T) {
	home := t.TempDir()
	r := &Runner{Root: t.TempDir(), Home: home, Out: io.Discard, Err: io.Discard, Verbosity: VerbosityQuiet}
	dep := scenario.Scenario{Slug: "beta"}
	// Empty registry: no other live consumers, so restart_when_stale is NOT
	// degraded and the caller is told to proceed with the restart (handled=false).
	handled, err := r.applyDependencyFreshnessPolicy("alpha", dep,
		freshnessPolicyDecision(scenario.DependencyFreshnessPolicyRestartWhenStale),
		registryRuntimeView{Instance: scenarioruntime.Instance{InstanceID: "beta-1"}}, []string{"main.go content changed"})
	if err != nil {
		t.Fatalf("applyDependencyFreshnessPolicy: %v", err)
	}
	if handled {
		t.Fatal("restart_when_stale with no other consumers must proceed to restart (handled=false)")
	}
}

func TestApplyFreshnessPolicyArbitrationDegradesToRebuildOnly(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	// A live consumer (gamma) depends on beta. The dep (beta) itself has no
	// setup phase, so the degraded rebuild_only path is a no-op build that keeps
	// the process — we only assert it did NOT restart (handled=true).
	gamma := lifecycleFixtureManifest("gamma")
	gamma.Dependencies.Scenarios = map[string]scenario.Dependency{
		"beta": {Required: true, StartupPolicy: scenario.DependencyStartupPolicyMustStart},
	}
	writeLifecycleFixtureManifest(t, root, gamma)

	ctx := context.Background()
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	if _, err := store.CreateInstance(ctx, scenarioruntime.Instance{Scenario: "gamma"}); err != nil {
		t.Fatalf("CreateInstance(gamma): %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close freshness test store: %v", err)
	}

	r := &Runner{Root: root, Home: home, Out: io.Discard, Err: io.Discard, Verbosity: VerbosityQuiet}
	// beta dep with an empty setup phase so rebuildDependencyArtifacts is a no-op.
	dep := scenario.Scenario{Slug: "beta", Manifest: scenario.ServiceManifest{Service: scenario.ServiceMetadata{Name: "beta"}}}

	handled, err := r.applyDependencyFreshnessPolicy("alpha", dep,
		freshnessPolicyDecision(scenario.DependencyFreshnessPolicyRestartWhenStale),
		registryRuntimeView{Instance: scenarioruntime.Instance{InstanceID: "beta-1"}}, []string{"main.go content changed"})
	if err != nil {
		t.Fatalf("applyDependencyFreshnessPolicy: %v", err)
	}
	if !handled {
		t.Fatal("arbitration must degrade restart_when_stale to rebuild_only when a live consumer exists (handled=true)")
	}
}
