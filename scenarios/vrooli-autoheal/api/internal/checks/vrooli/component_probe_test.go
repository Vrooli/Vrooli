package vrooli

import (
	"context"
	"net"
	"net/http"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/testutil"
)

func TestProbeComponentSkipsWhenNoPortAllocated(t *testing.T) {
	got := probeComponent(context.Background(), nil, roleUIPort, 0, false)
	if got.State != ComponentProbeSkipped {
		t.Fatalf("absent port must be skipped, got %v", got.State)
	}
	if got.Unreachable() {
		t.Error("a skipped probe must never count as unreachable")
	}
}

func TestProbeComponentReportsUnreachable(t *testing.T) {
	probe := func(context.Context, int32) ComponentProbeState { return ComponentProbeUnreachable }
	got := probeComponent(context.Background(), probe, roleUIPort, 4321, true)
	if !got.Unreachable() {
		t.Fatalf("want unreachable, got %v", got.State)
	}
	if got.Detail == "" {
		t.Error("unreachable probe should explain itself")
	}
}

// A live server that answers 404 is still a reachable UI. Treating a 404 as
// unreachable would restart every scenario whose dev server has no index route.
func TestProbeHTTPPortTreats404AsReachable(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })

	port := int32(listener.Addr().(*net.TCPAddr).Port)
	if got := probeHTTPPort(context.Background(), port); got != ComponentProbeOK {
		t.Fatalf("404 from a live server must be reachable, got %v", got)
	}
}

func TestProbeHTTPPortDetectsClosedPort(t *testing.T) {
	// Bind then immediately release, so the port is almost certainly free.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := int32(listener.Addr().(*net.TCPAddr).Port)
	_ = listener.Close()

	if got := probeHTTPPort(context.Background(), port); got != ComponentProbeUnreachable {
		t.Fatalf("closed port must be unreachable, got %v", got)
	}
}

func TestProbeHTTPPortSkipsZeroPort(t *testing.T) {
	if got := probeHTTPPort(context.Background(), 0); got != ComponentProbeSkipped {
		t.Fatalf("zero port must be skipped, got %v", got)
	}
}

// The core regression: `vrooli scenario status` reports health_status from the
// API probe only, so a scenario whose UI is dead reports "healthy" and used to
// produce StatusOK. It must now be a healable warning instead.
func TestScenarioCheckEscalatesDeadUIOnHealthyScenario(t *testing.T) {
	const output = `{"success":true,"scenario":{"name":"web-console","status":"running",` +
		`"health_status":"healthy","ports":{"API_PORT":18000,"UI_PORT":21000}}}`

	mockExecutor := testutil.NewMockExecutor()
	mockExecutor.Responses["vrooli scenario status web-console --json"] = testutil.MockResponse{
		Output: []byte(output),
	}

	check := NewScenarioCheck("web-console", true,
		WithScenarioExecutor(mockExecutor),
		WithScenarioComponentProbe(func(_ context.Context, port int32) ComponentProbeState {
			if port != 21000 {
				t.Errorf("probe should target UI_PORT, got %d", port)
			}
			return ComponentProbeUnreachable
		}),
	)

	result := check.Run(context.Background())

	if result.Status != checks.StatusWarning {
		t.Fatalf("dead UI must warn, got %v (message %q)", result.Status, result.Message)
	}
	if result.Details["rootCause"] != rootCauseUIUnreachable {
		t.Errorf("rootCause = %v, want %s", result.Details["rootCause"], rootCauseUIUnreachable)
	}
	if result.Details["recommendedAction"] != recommendedActionRestart {
		t.Errorf("recommendedAction = %v, want %s", result.Details["recommendedAction"], recommendedActionRestart)
	}
	// Without this marker the default policy would never act on the warning.
	if healable, _ := result.Details["healableDegradation"].(bool); !healable {
		t.Error("dead UI must be marked as a healable degradation")
	}
	if result.Details["uiProbe"] != string(ComponentProbeUnreachable) {
		t.Errorf("uiProbe = %v, want unreachable", result.Details["uiProbe"])
	}
}

func TestScenarioCheckKeepsHealthyWhenUIAnswers(t *testing.T) {
	const output = `{"success":true,"scenario":{"name":"web-console","status":"running",` +
		`"health_status":"healthy","ports":{"API_PORT":18000,"UI_PORT":21000}}}`

	mockExecutor := testutil.NewMockExecutor()
	mockExecutor.Responses["vrooli scenario status web-console --json"] = testutil.MockResponse{
		Output: []byte(output),
	}

	check := NewScenarioCheck("web-console", true,
		WithScenarioExecutor(mockExecutor),
		WithScenarioComponentProbe(func(context.Context, int32) ComponentProbeState {
			return ComponentProbeOK
		}),
	)

	result := check.Run(context.Background())
	if result.Status != checks.StatusOK {
		t.Fatalf("reachable UI must stay OK, got %v", result.Status)
	}
	if _, ok := result.Details["healableDegradation"]; ok {
		t.Error("healthy scenario must not be marked as a healable degradation")
	}
}

// A scenario with no UI at all must never be escalated for a missing UI.
func TestScenarioCheckIgnoresScenarioWithoutUIPort(t *testing.T) {
	const output = `{"success":true,"scenario":{"name":"api-only","status":"running",` +
		`"health_status":"healthy","ports":{"API_PORT":18000}}}`

	mockExecutor := testutil.NewMockExecutor()
	mockExecutor.Responses["vrooli scenario status api-only --json"] = testutil.MockResponse{
		Output: []byte(output),
	}

	probed := false
	check := NewScenarioCheck("api-only", true,
		WithScenarioExecutor(mockExecutor),
		WithScenarioComponentProbe(func(context.Context, int32) ComponentProbeState {
			probed = true
			return ComponentProbeUnreachable
		}),
	)

	result := check.Run(context.Background())
	if probed {
		t.Error("a scenario with no UI_PORT must not be probed")
	}
	if result.Status != checks.StatusOK {
		t.Fatalf("api-only scenario must stay OK, got %v", result.Status)
	}
	if result.Details["uiProbe"] != string(ComponentProbeSkipped) {
		t.Errorf("uiProbe = %v, want skipped", result.Details["uiProbe"])
	}
}

// A scenario the orchestrator already calls degraded should also gain the UI
// root cause, so the degradation becomes actionable rather than just noted.
func TestScenarioCheckAddsRootCauseToDegradedScenario(t *testing.T) {
	const output = `{"success":true,"scenario":{"name":"git-control-tower","status":"running",` +
		`"health_status":"degraded","ports":{"API_PORT":18710,"UI_PORT":21400}}}`

	mockExecutor := testutil.NewMockExecutor()
	mockExecutor.Responses["vrooli scenario status git-control-tower --json"] = testutil.MockResponse{
		Output: []byte(output),
	}

	check := NewScenarioCheck("git-control-tower", true,
		WithScenarioExecutor(mockExecutor),
		WithScenarioComponentProbe(func(context.Context, int32) ComponentProbeState {
			return ComponentProbeUnreachable
		}),
	)

	result := check.Run(context.Background())
	if result.Status != checks.StatusWarning {
		t.Fatalf("want warning, got %v", result.Status)
	}
	if result.Details["rootCause"] != rootCauseUIUnreachable {
		t.Errorf("degraded scenario with dead UI should name the UI root cause, got %v",
			result.Details["rootCause"])
	}
}
