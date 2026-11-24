package vrooli

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/testutil"
)

func TestProbeComponentSkipsWhenNoPortAllocated(t *testing.T) {
	got := probeComponent(context.Background(), nil, roleUIPort, 0, false)
	if got.State != ComponentProbeSkipped {
		t.Fatalf("absent port must be skipped, got %v", got.State)
	}
	if got.Failed() {
		t.Error("a skipped probe must never count as unreachable")
	}
}

func TestProbeComponentReportsUnreachable(t *testing.T) {
	probe := func(context.Context, int32) ComponentProbeState { return ComponentProbeUnreachable }
	got := probeComponent(context.Background(), probe, roleUIPort, 4321, true)
	if !got.Failed() {
		t.Fatalf("want unreachable, got %v", got.State)
	}
	if got.Detail == "" {
		t.Error("unreachable probe should explain itself")
	}
}

// serveUI serves handler on a loopback port for the strict probe tests.
func serveUI(t *testing.T, handler http.HandlerFunc) int32 {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: handler}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	return int32(listener.Addr().(*net.TCPAddr).Port)
}

const goodPage = `<!doctype html><html><body><div id="root"></div><script type="module" src="/assets/app.js"></script></body></html>`

// The strict form of the probe: 200, a non-blank page, and a bundle that loads.
func TestProbeHTTPPortIsStrict(t *testing.T) {
	cases := []struct {
		name    string
		handler http.HandlerFunc
		want    ComponentProbeState
	}{
		{"good page and bundle", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/assets/app.js" {
				_, _ = w.Write([]byte("console.log(1)"))
				return
			}
			_, _ = w.Write([]byte(goodPage))
		}, ComponentProbeOK},
		{"blank 200 page", func(w http.ResponseWriter, _ *http.Request) {}, ComponentProbeDegraded},
		{"root but bundle 404", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/assets/app.js" {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write([]byte(goodPage))
		}, ComponentProbeDegraded},
		{"root but empty bundle", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/assets/app.js" {
				_, _ = w.Write([]byte(goodPage))
			}
		}, ComponentProbeDegraded},
		{"404 index", func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) }, ComponentProbeUnreachable},
		{"500 index", func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "boom", 500) }, ComponentProbeUnreachable},
		{"static page with text and no script", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("<html><body><h1>Status</h1><p>All systems are operating normally.</p></body></html>"))
		}, ComponentProbeOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := probeHTTPPort(context.Background(), serveUI(t, tc.handler)); got != tc.want {
				t.Fatalf("probe = %v, want %v", got, tc.want)
			}
		})
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

// A manifest that declares a UI port while status returned none is a skipped
// probe with a warning, never a silent green.
func TestScenarioCheckWarnsWhenManifestDeclaresUIButStatusHasNone(t *testing.T) {
	const output = `{"success":true,"scenario":{"name":"ui-less","status":"running",` +
		`"health_status":"healthy","ports":{"API_PORT":18000}}}`
	mockExecutor := testutil.NewMockExecutor()
	mockExecutor.Responses["vrooli scenario status ui-less --json"] = testutil.MockResponse{Output: []byte(output)}
	check := NewScenarioCheck("ui-less", true,
		WithScenarioExecutor(mockExecutor),
		WithScenarioDeclaresUI(func() bool { return true }),
	)
	result := check.Run(context.Background())
	if result.Status != checks.StatusWarning || result.Details["uiProbe"] != string(ComponentProbeSkipped) {
		t.Fatalf("result = %v uiProbe=%v, want warning with a skipped probe", result.Status, result.Details["uiProbe"])
	}
	if healable, _ := result.Details["healableDegradation"].(bool); healable {
		t.Error("a skipped probe is a signal, not a restart trigger")
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

// A restart still draining its provider fence after five minutes is a loud
// critical naming the initiator and the repair, never a silent wait, and it is
// not heal-eligible: another restart would join the same fence.
func TestScenarioCheckReportsRestartStuckInProviderFence(t *testing.T) {
	status := func(elapsed int) string {
		return `{"success":true,"scenario":{"name":"vrooli-events","status":"running","health_status":"healthy",` +
			`"ports":{"API_PORT":18000},"start_operation":{"operation":"restart","status":"running","current_step":"provider_fence",` +
			`"elapsed_seconds":` + strconv.Itoa(elapsed) + `,"initiator_pid":4242,"fence_operation_id":"restart-inst-1"}}}`
	}
	run := func(elapsed int) checks.Result {
		mockExecutor := testutil.NewMockExecutor()
		mockExecutor.Responses["vrooli scenario status vrooli-events --json"] = testutil.MockResponse{Output: []byte(status(elapsed))}
		return NewScenarioCheck("vrooli-events", true, WithScenarioExecutor(mockExecutor),
			WithScenarioDeclaresUI(func() bool { return false })).Run(context.Background())
	}

	if got := run(120); got.Status != checks.StatusOK {
		t.Fatalf("a young restart must not be flagged: %v %q", got.Status, got.Message)
	}
	got := run(301)
	if got.Status != checks.StatusCritical || got.Details["rootCause"] != "lifecycle-provider-fence-stuck" {
		t.Fatalf("stuck restart = %v %q, want critical", got.Status, got.Message)
	}
	for _, want := range []string{"4242", "--force-lifecycle", "provider_fence"} {
		if !strings.Contains(got.Message, want) {
			t.Errorf("message %q does not name %q", got.Message, want)
		}
	}
	if eligible, _ := got.Details["autoHealEligible"].(bool); eligible {
		t.Error("a stuck restart must not trigger another restart")
	}
}
