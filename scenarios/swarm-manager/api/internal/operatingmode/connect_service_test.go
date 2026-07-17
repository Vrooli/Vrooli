package operatingmode

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
)

// newConnectTestClient mounts the real OperatingModeService Connect handler over
// a service built from the data-backed registry and returns a generated client
// pointed at it — exercising the full transport + projection-mapper path, not
// just the mappers in isolation.
func newConnectTestClient(t *testing.T, opts serviceOptions) apiconnect.OperatingModeServiceClient {
	t.Helper()
	svc := newTestServiceWithOptions(t, t.TempDir(), opts)
	path, handler := apiconnect.NewOperatingModeServiceHandler(NewConnectService(svc))
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return apiconnect.NewOperatingModeServiceClient(server.Client(), server.URL)
}

func TestConnectCatalogProjectsRegisteredModes(t *testing.T) {
	client := newConnectTestClient(t, serviceOptions{})

	resp, err := client.Catalog(context.Background(), connect.NewRequest(&apipb.OperatingModeCatalogRequest{}))
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}
	byMode := map[string]*apipb.OperatingModeCatalogEntry{}
	for _, e := range resp.Msg.GetModes() {
		byMode[e.GetMode()] = e
	}
	for _, want := range []string{"holistic-loop", "phased-plan-drain"} {
		if _, ok := byMode[want]; !ok {
			t.Fatalf("catalog missing mode %q; got %v", want, byMode)
		}
	}
	// The member-item strategy sentinel is not a mode and never appears in
	// the catalog.
	if _, ok := byMode["item-level"]; ok {
		t.Fatalf("catalog contains the retired item-level pseudo-mode")
	}

	// A phase-based mode carries a populated capabilities block and phase graph.
	holistic := byMode["holistic-loop"]
	if holistic.GetCapabilities() == nil || !holistic.GetSupportsPhases() {
		t.Fatalf("holistic-loop should support phases: %+v", holistic)
	}
	if holistic.GetPhaseGraph() == nil || holistic.GetPhaseGraph().GetStartPhase() == "" {
		t.Fatalf("holistic-loop should carry a phase graph with a start phase: %+v", holistic.GetPhaseGraph())
	}
	if len(holistic.GetPhases()) == 0 {
		t.Fatalf("holistic-loop should carry projected phases")
	}
}

func TestConnectGetModeUnknownIsNotFound(t *testing.T) {
	client := newConnectTestClient(t, serviceOptions{})

	_, err := client.GetMode(context.Background(), connect.NewRequest(&apipb.OperatingModeGetRequest{Mode: "does-not-exist"}))
	if err == nil {
		t.Fatalf("expected error for unknown mode")
	}
	if got := connect.CodeOf(err); got != connect.CodeNotFound {
		t.Fatalf("code = %v, want NotFound; err=%v", got, err)
	}
}

func TestConnectGetModeMissingModeIsInvalidArgument(t *testing.T) {
	client := newConnectTestClient(t, serviceOptions{})

	_, err := client.GetMode(context.Background(), connect.NewRequest(&apipb.OperatingModeGetRequest{Mode: ""}))
	if err == nil {
		t.Fatalf("expected error for empty mode")
	}
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want InvalidArgument; err=%v", got, err)
	}
}

func TestConnectSimulateWalksRealGuards(t *testing.T) {
	client := newConnectTestClient(t, serviceOptions{})

	resp, err := client.SimulateMode(context.Background(), connect.NewRequest(&apipb.OperatingModeSimulateRequest{
		Mode: "phased-plan-drain",
	}))
	if err != nil {
		t.Fatalf("SimulateMode: %v", err)
	}
	if resp.Msg.GetMode() != "phased-plan-drain" {
		t.Fatalf("mode = %q, want phased-plan-drain", resp.Msg.GetMode())
	}
	if len(resp.Msg.GetTrace()) == 0 {
		t.Fatalf("simulation trace should be non-empty")
	}
	// Every non-terminal step renders a generic guard-derived transition; the
	// first step must carry a phase and a round projection.
	first := resp.Msg.GetTrace()[0]
	if first.GetPhase() == "" {
		t.Fatalf("first step missing phase: %+v", first)
	}
	if first.GetRound() == nil {
		t.Fatalf("first step missing round projection: %+v", first)
	}
}

// TestConnectAuthoringRoundTrip drives scaffold → validate → draft-simulate
// through the generated client, proving the self-serve authoring surface is
// reachable over Connect end to end and that a scaffolded, unregistered mode is
// simulatable straight from disk.
func TestConnectAuthoringRoundTrip(t *testing.T) {
	client := newConnectTestClient(t, serviceOptions{})
	ctx := context.Background()

	scaffold, err := client.ScaffoldMode(ctx, connect.NewRequest(&apipb.OperatingModeScaffoldRequest{
		Id:    "wire-demo",
		Label: "Wire Demo",
	}))
	if err != nil {
		t.Fatalf("ScaffoldMode: %v", err)
	}
	if scaffold.Msg.GetMode() != "wire-demo" || len(scaffold.Msg.GetCreatedFiles()) == 0 {
		t.Fatalf("unexpected scaffold response: %+v", scaffold.Msg)
	}

	validate, err := client.ValidateMode(ctx, connect.NewRequest(&apipb.OperatingModeValidateRequest{Mode: "wire-demo"}))
	if err != nil {
		t.Fatalf("ValidateMode: %v", err)
	}
	if !validate.Msg.GetOk() {
		t.Fatalf("scaffolded mode failed validation: %+v", validate.Msg)
	}

	sim, err := client.SimulateMode(ctx, connect.NewRequest(&apipb.OperatingModeSimulateRequest{
		Mode:  "wire-demo",
		Draft: true,
	}))
	if err != nil {
		t.Fatalf("draft SimulateMode: %v", err)
	}
	if len(sim.Msg.GetTrace()) != 3 {
		t.Fatalf("draft simulation trace = %d steps, want 3: %+v", len(sim.Msg.GetTrace()), sim.Msg.GetTrace())
	}

	// A non-draft simulate of the same id must 404 — it is not registered.
	if _, err := client.SimulateMode(ctx, connect.NewRequest(&apipb.OperatingModeSimulateRequest{Mode: "wire-demo"})); err == nil {
		t.Fatalf("expected non-draft simulate of unregistered mode to fail")
	}
}

// TestConnectValidateMissingReportsNotOK confirms an absent mode is a typed
// ok=false report, not a transport error.
func TestConnectValidateMissingReportsNotOK(t *testing.T) {
	client := newConnectTestClient(t, serviceOptions{})
	resp, err := client.ValidateMode(context.Background(), connect.NewRequest(&apipb.OperatingModeValidateRequest{Mode: "ghost-mode"}))
	if err != nil {
		t.Fatalf("ValidateMode: %v", err)
	}
	if resp.Msg.GetOk() || len(resp.Msg.GetErrors()) == 0 {
		t.Fatalf("expected not-ok report with errors, got %+v", resp.Msg)
	}
}

func TestConnectUpdateModeNoChangesIsInvalidArgument(t *testing.T) {
	client := newConnectTestClient(t, serviceOptions{})

	_, err := client.UpdateMode(context.Background(), connect.NewRequest(&apipb.OperatingModeUpdateRequest{
		Mode: "holistic-loop",
	}))
	if err == nil {
		t.Fatalf("expected error when no overlay fields provided")
	}
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want InvalidArgument; err=%v", got, err)
	}
}

// TestConnectSwitchModeActiveItemExecutionsCarriesConflictDetail proves the
// active-item-executions conflict travels over Connect as a FailedPrecondition
// carrying the structured OperatingModeActiveItemExecutionsConflict detail — the
// wire contract the UI's parseActiveItemExecutionsConflict decodes to list the
// affected executions before re-submitting the switch with cancellation.
func TestConnectSwitchModeActiveItemExecutionsCarriesConflictDetail(t *testing.T) {
	active := &fakeItemExecutions{active: []ActiveItemExecution{{
		ItemRef:     "execute/do-thing",
		ExecutionID: "exec-1",
		RunID:       "run-1",
		Status:      "running",
	}}}
	updater := &fakeModeUpdater{items: map[string]InitiativeSnapshot{
		"init-item": {
			Name:  "init-item",
			Title: "Init Item",
			Mode:  string(ModeItemLevel),
			Items: []string{"execute/do-thing"},
		},
	}}
	client := newConnectTestClient(t, serviceOptions{
		initiatives: fakeInitiatives{items: map[string]InitiativeSnapshot{
			"init-item": updater.items["init-item"],
		}},
		modeUpdater:    updater,
		itemExecutions: active,
	})

	_, err := client.SwitchMode(context.Background(), connect.NewRequest(&apipb.OperatingModeSwitchRequest{
		InitiativeName: "init-item",
		Mode:           string(ModeHolisticLoop),
	}))
	if err == nil {
		t.Fatalf("expected active-item-executions conflict")
	}
	if got := connect.CodeOf(err); got != connect.CodeFailedPrecondition {
		t.Fatalf("code = %v, want FailedPrecondition; err=%v", got, err)
	}
	connErr := new(connect.Error)
	if !errors.As(err, &connErr) {
		t.Fatalf("error is not a *connect.Error: %v", err)
	}
	var conflict *apipb.OperatingModeActiveItemExecutionsConflict
	for _, d := range connErr.Details() {
		msg, verr := d.Value()
		if verr != nil {
			continue
		}
		if c, ok := msg.(*apipb.OperatingModeActiveItemExecutionsConflict); ok {
			conflict = c
			break
		}
	}
	if conflict == nil {
		t.Fatalf("conflict detail not attached to Connect error")
	}
	if conflict.GetInitiativeName() != "init-item" || conflict.GetToMode() != string(ModeHolisticLoop) {
		t.Fatalf("conflict detail = %+v", conflict)
	}
	if len(conflict.GetExecutions()) != 1 || conflict.GetExecutions()[0].GetExecutionId() != "exec-1" {
		t.Fatalf("conflict executions = %+v", conflict.GetExecutions())
	}
}
