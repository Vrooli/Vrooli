package phases

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing"
	"github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing/routing_v1connect"

	"test-genie/internal/eligibility"
	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/playbooksclaims"
	playbooksclaimsmocks "test-genie/internal/playbooksclaims/mocks"
)

// stubEligibilityChecker reports a fixed eligibility outcome.
type stubEligibilityChecker struct {
	elig eligibility.Eligibility
	err  error
}

func (s stubEligibilityChecker) Check(ctx context.Context, scenario string, mapping workspace.Mapping) (eligibility.Eligibility, error) {
	return s.elig, s.err
}

func (s stubEligibilityChecker) Invalidate(string) {}

// stubRoutingClient satisfies routing_v1connect.RoutingServiceClient but
// none of its methods are expected to be called in tests that bail out at
// the preflight stage. Any unexpected call surfaces loudly.
type stubRoutingClient struct{}

func (stubRoutingClient) InstallTestPool(context.Context, *connect.Request[routingv1.InstallTestPoolRequest]) (*connect.Response[routingv1.InstallTestPoolResponse], error) {
	return nil, errors.New("stub: InstallTestPool not expected")
}

func (stubRoutingClient) ClearTestPool(context.Context, *connect.Request[routingv1.ClearTestPoolRequest]) (*connect.Response[routingv1.ClearTestPoolResponse], error) {
	return nil, errors.New("stub: ClearTestPool not expected")
}

func (stubRoutingClient) HeartbeatTestPool(context.Context, *connect.Request[routingv1.HeartbeatTestPoolRequest]) (*connect.Response[routingv1.HeartbeatTestPoolResponse], error) {
	return nil, errors.New("stub: HeartbeatTestPool not expected")
}

func overrideRoutingChecker(stub eligibilityChecker) func() {
	prev := routingChecker
	routingChecker = stub
	return func() { routingChecker = prev }
}

func overrideRoutingProbes(client routing_v1connect.RoutingServiceClient, probeErr error) func() {
	prevClient := resolveScenarioRoutingClient
	prevProbe := probeRoutingServiceEnabled
	resolveScenarioRoutingClient = func(ctx context.Context, scenarioName string) (routing_v1connect.RoutingServiceClient, error) {
		return client, nil
	}
	probeRoutingServiceEnabled = func(ctx context.Context, scenarioName string) error {
		return probeErr
	}
	return func() {
		resolveScenarioRoutingClient = prevClient
		probeRoutingServiceEnabled = prevProbe
	}
}

// minimalRoutedScenario writes the bits of disk state runPlaybooksPhase
// requires (registry with one playbook, a workflow file) so it doesn't bail
// out early on missing files.
func minimalRoutedScenario(t *testing.T) (workspace.Environment, *fakeIsolation) {
	t.Helper()
	appRoot := t.TempDir()
	scenarioName := "fake-routed-scenario"
	scenarioDir := filepath.Join(appRoot, "scenarios", scenarioName)
	for _, sub := range []string{"ui", "bas/cases", ".vrooli", "test"} {
		if err := os.MkdirAll(filepath.Join(scenarioDir, sub), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", sub, err)
		}
	}
	registry := `{"playbooks":[{"file":"bas/cases/smoke.json"}]}`
	if err := os.WriteFile(filepath.Join(scenarioDir, "bas", "registry.json"), []byte(registry), 0o644); err != nil {
		t.Fatalf("write registry: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "bas", "cases", "smoke.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write smoke playbook: %v", err)
	}

	claimsSvc := playbooksclaims.NewService(playbooksclaims.Config{Repo: playbooksclaimsmocks.NewFakeRepository()})
	env := workspace.Environment{
		ScenarioName:  scenarioName,
		ScenarioDir:   scenarioDir,
		TestDir:       filepath.Join(scenarioDir, "test"),
		AppRoot:       appRoot,
		TargetRuntime: &fakeTargetRuntime{},
		Claims:        claimsSvc,
	}
	return env, &fakeIsolation{err: fmt.Errorf("prepare-not-expected")}
}

// TestRunPlaybooksPhase_ProductionMode_ShortCircuitsBeforeIsolation verifies
// gap #1 from the routed-test-db investigation: when the scenario reports
// production-mode (RoutingService probe returns errRoutingServiceDisabled)
// AND the fallback path can't run because TargetRuntime is nil, the phase
// must bail BEFORE calling isoManager.Prepare().
//
// The fakeIsolation is wired to fail if Prepare is called, so the assertion
// is implicit: a passing test means Prepare was never reached.
func TestRunPlaybooksPhase_ProductionMode_ShortCircuitsBeforeIsolation(t *testing.T) {
	env, iso := minimalRoutedScenario(t)
	env.TargetRuntime = nil // force the fallback fail-fast path

	restoreIso := overrideIsolationManager(iso)
	defer restoreIso()

	restoreChecker := overrideRoutingChecker(stubEligibilityChecker{
		elig: eligibility.Eligibility{Routed: true},
	})
	defer restoreChecker()

	restoreProbes := overrideRoutingProbes(stubRoutingClient{}, errRoutingServiceDisabled)
	defer restoreProbes()

	report := runPlaybooksPhase(context.Background(), env, io.Discard)

	if iso.called {
		t.Fatal("isoManager.Prepare was called — preflight should have short-circuited before isolation")
	}
	if report.Err == nil {
		t.Fatal("expected a hard failure when fallback is impossible")
	}
	if !strings.Contains(report.Err.Error(), "target runtime") {
		t.Errorf("error should mention target runtime, got: %v", report.Err)
	}
}

// TestRunPlaybooksPhase_RoutingUnreachable_StillTriesFallback verifies that
// when the routing client is unreachable but TargetRuntime is wired, we
// still proceed to isolation+fallback (since fallback doesn't need routing).
func TestRunPlaybooksPhase_RoutingUnreachable_StillTriesFallback(t *testing.T) {
	env, iso := minimalRoutedScenario(t)
	// Prepare should be called this time — but make it fail with a
	// well-known sentinel so the phase terminates without progressing to
	// the actual scenario lifecycle (which we don't have wired in this
	// test). The fact that Prepare WAS reached is the assertion.
	iso.err = errors.New("sentinel-prepare-failure")

	restoreIso := overrideIsolationManager(iso)
	defer restoreIso()

	restoreChecker := overrideRoutingChecker(stubEligibilityChecker{
		elig: eligibility.Eligibility{Routed: true},
	})
	defer restoreChecker()

	// Probe fails generically — should be classified as routing_unreachable,
	// downgrade to fallback, but Prepare must still be reached.
	restoreProbes := overrideRoutingProbes(stubRoutingClient{}, errors.New("probe boom"))
	defer restoreProbes()

	report := runPlaybooksPhase(context.Background(), env, io.Discard)

	if !iso.called {
		t.Fatal("isoManager.Prepare should have been called (fallback path was viable)")
	}
	if report.Err == nil || !strings.Contains(report.Err.Error(), "sentinel-prepare-failure") {
		t.Errorf("expected the sentinel Prepare error to surface, got: %v", report.Err)
	}
}
