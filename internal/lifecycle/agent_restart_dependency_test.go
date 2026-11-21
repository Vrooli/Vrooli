package lifecycle

import (
	"context"
	"errors"
	"io"
	"testing"

	internalcapacity "github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/resources"
	resourcecontrol "github.com/vrooli/vrooli/internal/resources/control"
	resourcemanifest "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/scenario"
	publiccapacity "github.com/vrooli/vrooli/packages/capacity"
)

func allowRestartFixtureBuilds(runner *Runner) {
	runner.buildAdmissionFactory = func(context.Context, string) (buildAdmissionBroker, error) {
		return &buildAdmissionFakeBroker{lease: &buildAdmissionFakeLease{}, verdict: publiccapacity.Verdict{Kind: internalcapacity.VerdictGrant}}, nil
	}
}

// A healthy optional model service must not pull models while its execution
// owner is offline. The resource check is made while the old owner still runs.
func TestAgentManagerRestartPreparesDependenciesBeforeStopping(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	manifest := lifecycleFixtureManifest("agent-manager")
	manifest.Dependencies.Resources = map[string]scenario.Dependency{
		"ollama": {Enabled: true, Required: false, StartupPolicy: scenario.DependencyStartupPolicyTryStart, Config: []byte(`{"model_roles":["code.local"]}`)},
	}
	writeLifecycleFixtureManifest(t, root, manifest)
	events := []string{}
	var ensureCalls, resourceStarts int
	var unhealthy bool
	runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.resourceStatus = func(name string, _ bool) (resourcecontrol.Status, error) {
			events = append(events, "resource-status")
			healthy := !unhealthy
			return resourcecontrol.Status{Resource: resources.Resource{Name: name}, Running: healthy, Healthy: &healthy}, nil
		}
		deps.resourceManifest = func(string) (resourcemanifest.ResourceManifest, error) {
			return resourcemanifest.ResourceManifest{Capabilities: resourcemanifest.ResourceManifestCapabilities{SupportsEnsure: true}}, nil
		}
		deps.runResourceCLI = func(string, []string, io.Writer, io.Writer) error { ensureCalls++; return nil }
		deps.runResource = func(string, []string, io.Writer, io.Writer) error { resourceStarts++; return nil }
	})
	allowRestartFixtureBuilds(runner)
	if _, err := runner.Start("agent-manager", StartOptions{}); err != nil {
		t.Fatal(err)
	}
	cleanupRunner(t, runner, "agent-manager", StopOptions{})
	if ensureCalls != 1 {
		t.Fatalf("initial start must ensure resource configuration, got %d", ensureCalls)
	}
	ensureCalls = 0
	events = nil
	runner.WithProgressSink(&restartOrderSink{events: &events})
	if _, err := runner.Restart("agent-manager", StartOptions{}); err != nil {
		t.Fatal(err)
	}
	if ensureCalls != 0 {
		t.Fatalf("restart needlessly re-ensured a healthy model resource %d times", ensureCalls)
	}
	if len(events) < 2 || events[0] != "resource-status" || events[1] != "stop" {
		t.Fatalf("dependency must be checked before stop: %v", events)
	}
	unhealthy = true
	events = nil
	result, err := runner.Restart("agent-manager", StartOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if resourceStarts != 0 || ensureCalls != 0 {
		t.Fatalf("optional resource was started or ensured: starts=%d ensures=%d", resourceStarts, ensureCalls)
	}
	if len(result.FailedResources) != 1 || result.FailedResources[0] != "ollama" {
		t.Fatalf("unavailable optional resource must be reported: %v", result.FailedResources)
	}
	if len(events) < 2 || events[0] != "resource-status" || events[1] != "stop" {
		t.Fatalf("unavailable optional resource must be assessed before stop: %v", events)
	}
}

func TestAgentManagerRestartLeavesOwnerServingWhenRequiredDependencyReadFails(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	manifest := lifecycleFixtureManifest("agent-manager")
	manifest.Dependencies.Resources = map[string]scenario.Dependency{
		"required": {Enabled: true, Required: true, StartupPolicy: scenario.DependencyStartupPolicyMustStart},
	}
	writeLifecycleFixtureManifest(t, root, manifest)
	unavailable := false
	runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.resourceStatus = func(name string, _ bool) (resourcecontrol.Status, error) {
			if unavailable {
				return resourcecontrol.Status{}, errors.New("resource observation failed")
			}
			healthy := true
			return resourcecontrol.Status{Resource: resources.Resource{Name: name}, Running: true, Healthy: &healthy}, nil
		}
	})
	allowRestartFixtureBuilds(runner)
	if _, err := runner.Start("agent-manager", StartOptions{}); err != nil {
		t.Fatal(err)
	}
	cleanupRunner(t, runner, "agent-manager", StopOptions{})
	before := liveScenarioPID(t, home, "agent-manager")
	unavailable = true
	if _, err := runner.Restart("agent-manager", StartOptions{}); err == nil {
		t.Fatal("required dependency failure must be reported")
	}
	if after := liveScenarioPID(t, home, "agent-manager"); after != before {
		t.Fatalf("dependency failure interrupted serving owner: %d -> %d", before, after)
	}
}
