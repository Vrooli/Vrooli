package lifecycle

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestDependencySourceDriftKeepsServingOwner(t *testing.T) {
	for _, policy := range []string{"optional", scenario.DependencyFreshnessPolicyReuseRunning} {
		t.Run(policy, func(t *testing.T) {
			root, home := t.TempDir(), t.TempDir()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, `{"status":"healthy","readiness":true}`)
			}))
			defer server.Close()
			manifest := scenario.ServiceManifest{
				Service:      scenario.ServiceMetadata{Name: "program-runtime"},
				Lifecycle:    scenario.Lifecycle{Health: &scenario.HealthConfig{Checks: []scenario.HealthCheck{{Name: "api", Type: "http", Target: server.URL, Critical: true}}}},
				Dependencies: scenario.Dependencies{Scenarios: map[string]scenario.Dependency{"agent-manager": {Enabled: true, StartupPolicy: scenario.DependencyStartupPolicyTryStart}}},
			}
			writeLifecycleFixtureManifest(t, root, manifest)
			runner := newLifecycleRunnerForTest(t, root, home, nil)
			item, err := runner.loadScenario("program-runtime", "")
			if err != nil {
				t.Fatal(err)
			}
			identity, err := scenarioBuildIdentity(item)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(item.Path, "PRD.md"), []byte("new authored documentation"), 0o600); err != nil {
				t.Fatal(err)
			}
			host, err := runner.runtimeDeps().hostSession(t.Context(), home)
			if err != nil {
				t.Fatal(err)
			}
			store, err := scenarioruntime.NewSQLiteStore(t.Context(), scenarioruntime.Config{HomeDir: home})
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			instance, err := store.CreateLease(t.Context(), scenarioruntime.Instance{Scenario: "program-runtime", Status: scenarioruntime.StatusRunning, BuildIdentity: identity, HostBootID: host.BootID}, scenarioruntime.DefaultHeartbeatTTL)
			if err != nil {
				t.Fatal(err)
			}
			dependency := scenario.Dependency{Enabled: true, StartupPolicy: scenario.DependencyStartupPolicyTryStart}
			if policy != "optional" {
				dependency.Required = true
				dependency.StartupPolicy = scenario.DependencyStartupPolicyMustStart
				dependency.FreshnessPolicy = policy
			}
			consumer := scenario.Scenario{Slug: "agent-manager", Manifest: scenario.ServiceManifest{Dependencies: scenario.Dependencies{Scenarios: map[string]scenario.Dependency{"program-runtime": dependency}}}}
			session := newStartSession(t.Context()).childStack("agent-manager")
			failed, err := runner.ensureDependency(t.Context(), consumer, StartOptions{}, session, 0, "program-runtime", 1)
			if err != nil || failed != "" || !session.isReady("program-runtime") {
				t.Fatalf("healthy serving dependency was churned into its bootstrap cycle: failed=%s err=%v", failed, err)
			}
			instances, err := store.ListInstances(t.Context(), scenarioruntime.InstanceFilter{Scenario: "program-runtime"})
			if err != nil || len(instances) != 1 || instances[0].InstanceID != instance.InstanceID || instances[0].Status != scenarioruntime.StatusRunning || instances[0].BuildIdentity != identity {
				t.Fatalf("reuse replaced the serving owner or fabricated its build identity: %+v %v", instances, err)
			}
		})
	}
}

func TestDependencyBootstrapDefersOptionalAncestor(t *testing.T) {
	for _, required := range []bool{false, true} {
		name := "optional"
		if required {
			name = "required"
		}
		t.Run(name, func(t *testing.T) {
			// No AM manifest, API or registry is needed to defer this edge.
			// Requiring them would deadlock bootstrap before AM can serve its gate.
			runner := &Runner{Out: io.Discard, Err: io.Discard}
			item := scenario.Scenario{Slug: "program-runtime", Manifest: scenario.ServiceManifest{Dependencies: scenario.Dependencies{Scenarios: map[string]scenario.Dependency{"agent-manager": {Enabled: true, Required: required}}}}}
			session := newStartSession(t.Context()).childStack("agent-manager").childStack("program-runtime")
			failed, err := runner.ensureDependency(t.Context(), item, StartOptions{}, session, 0, "agent-manager", 1)
			if required {
				if err == nil || !strings.Contains(err.Error(), "circular scenario dependency") {
					t.Fatalf("required cycle accepted: %v", err)
				}
			} else if err != nil || failed != "agent-manager" {
				t.Fatalf("optional ancestor must defer without maintaining its owner: failed=%s err=%v", failed, err)
			}
		})
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
