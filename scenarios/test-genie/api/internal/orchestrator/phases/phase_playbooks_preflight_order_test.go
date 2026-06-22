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

// succeedingRoutingClient satisfies routing_v1connect.RoutingServiceClient and
// returns empty-but-valid responses so the routed path can run end-to-end in
// tests. ClearTestPool returns no stats so the lease-stats gate is skipped.
type succeedingRoutingClient struct{}

func (succeedingRoutingClient) InstallTestPool(context.Context, *connect.Request[routingv1.InstallTestPoolRequest]) (*connect.Response[routingv1.InstallTestPoolResponse], error) {
	return connect.NewResponse(&routingv1.InstallTestPoolResponse{}), nil
}

func (succeedingRoutingClient) ClearTestPool(context.Context, *connect.Request[routingv1.ClearTestPoolRequest]) (*connect.Response[routingv1.ClearTestPoolResponse], error) {
	return connect.NewResponse(&routingv1.ClearTestPoolResponse{}), nil
}

func (succeedingRoutingClient) HeartbeatTestPool(context.Context, *connect.Request[routingv1.HeartbeatTestPoolRequest]) (*connect.Response[routingv1.HeartbeatTestPoolResponse], error) {
	return connect.NewResponse(&routingv1.HeartbeatTestPoolResponse{}), nil
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

// assertRefusedSkip asserts the report is a fail-closed refusal: no hard error,
// and a SKIP observation announcing the refusal. The restart-based fallback was
// deleted, so any non-routed outcome is a refusal — never a restart.
func assertRefusedSkip(t *testing.T, report RunReport) {
	t.Helper()
	if report.Err != nil {
		t.Fatalf("refusal must be a skip, not a hard error; got: %v", report.Err)
	}
	for _, obs := range report.Observations {
		if obs.Prefix == "SKIP" && strings.Contains(strings.ToLower(obs.Text), "refused") {
			return
		}
	}
	t.Fatalf("expected a SKIP observation announcing the refusal; got observations=%+v", report.Observations)
}

// TestRunPlaybooksPhase_ProductionMode_RefusesBeforeIsolation verifies that when
// the scenario reports production-mode (RoutingService probe returns
// errRoutingServiceDisabled) the phase refuses fail-closed BEFORE calling
// isoManager.Prepare() — there is no restart-based fallback to spend a
// testcontainer on.
//
// The fakeIsolation is wired to fail if Prepare is called, so a passing test
// means Prepare was never reached.
func TestRunPlaybooksPhase_ProductionMode_RefusesBeforeIsolation(t *testing.T) {
	env, iso := minimalRoutedScenario(t)

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
		t.Fatal("isoManager.Prepare was called — production-mode refusal should short-circuit before isolation")
	}
	assertRefusedSkip(t, report)
}

// TestRunPlaybooksPhase_RoutingUnreachable_RefusesBeforeIsolation verifies that
// when the routing client/probe is unreachable, the phase refuses fail-closed
// before isolation — the routed path is the only path, so an unreachable
// RoutingService cannot be worked around with a restart.
func TestRunPlaybooksPhase_RoutingUnreachable_RefusesBeforeIsolation(t *testing.T) {
	env, iso := minimalRoutedScenario(t)

	restoreIso := overrideIsolationManager(iso)
	defer restoreIso()

	restoreChecker := overrideRoutingChecker(stubEligibilityChecker{
		elig: eligibility.Eligibility{Routed: true},
	})
	defer restoreChecker()

	// Probe fails generically — classified as routing_unreachable → refuse.
	restoreProbes := overrideRoutingProbes(stubRoutingClient{}, errors.New("probe boom"))
	defer restoreProbes()

	report := runPlaybooksPhase(context.Background(), env, io.Discard)

	if iso.called {
		t.Fatal("isoManager.Prepare was called — routing-unreachable refusal should short-circuit before isolation")
	}
	assertRefusedSkip(t, report)
}
