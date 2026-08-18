package gate

import (
	"context"
	"testing"
	"time"

	auditmocks "vrooli-bridge/internal/audit/mocks"
	"vrooli-bridge/internal/cpkeys"
	internalgate "vrooli-bridge/internal/gate"
	gatemocks "vrooli-bridge/internal/gate/mocks"
	"vrooli-bridge/internal/presence"
	internalqueue "vrooli-bridge/internal/queue"
	"vrooli-bridge/internal/registry"
	rmocks "vrooli-bridge/internal/registry/mocks"
	"vrooli-bridge/internal/runs"
	runsmocks "vrooli-bridge/internal/runs/mocks"

	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/api-core/scheduletest"

	dispatchH "vrooli-bridge/handlers/dispatch"
	queueH "vrooli-bridge/handlers/queue"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	gatev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/gate"
)

// dmHarness wires the gate handler over the SAME real services production wires
// (registry + presence + the shared dispatch service + runs), so the integration
// test drives the full cross-domain path a deployment-manager gate call takes:
// GateService → dispatch (allowlist + scopes + audit) → runs (durable lifecycle)
// → the node channel. Only the node-agent is simulated (we append the terminal
// run events a real agent would stream back).
type dmHarness struct {
	handler *connectHandler
	runs    runs.Service
	hub     *presence.Hub
}

func newDMHarness(t *testing.T) *dmHarness {
	t.Helper()
	clk := schedule.System()

	// One trusted node per target OS, each scoped to run the validation verb.
	repo := rmocks.NewFakeRepository()
	repo.Seed(registry.Node{ID: "ubuntu-1", Name: "ci-linux", OS: "linux", Arch: "amd64", Scopes: []string{"vrooli-bridge:write", "scenario test*"}})
	repo.Seed(registry.Node{ID: "mac-1", Name: "ci-darwin", OS: "darwin", Arch: "arm64", Scopes: []string{"vrooli-bridge:write", "scenario test*"}})
	repo.Seed(registry.Node{ID: "win-1", Name: "ci-windows", OS: "windows", Arch: "amd64", Scopes: []string{"vrooli-bridge:write", "scenario test*"}})
	registrySvc := registry.NewService(repo)

	runsSvc := runs.NewService(runsmocks.NewFakeRepository(), scheduletest.New(time.Now()))
	auditSink := &auditmocks.FakeSink{}

	hub := presence.NewHub(clk)
	// Each node dials out and holds a channel (online + protocol-compatible). We
	// drain the pushed JobPush frames in the background so the scheduler's push
	// never blocks on an unread channel.
	for _, id := range []string{"ubuntu-1", "mac-1", "win-1"} {
		conn := hub.Connect(id)
		t.Cleanup(conn.Close)
		go func() {
			for range conn.Out() {
			}
		}()
	}

	cpKey, err := cpkeys.LoadOrCreate(t.TempDir())
	require.NoError(t, err)
	scheduler := internalqueue.NewScheduler(queueH.NewChannelPusher(hub, cpKey), queueH.NewAborter(runsSvc), clk, 0)
	dispatchSvc := dispatchH.NewService(registrySvc, runsSvc, auditSink, hub, scheduler)

	svc := internalgate.NewService(
		gatemocks.NewFakeRepository(),
		nodeListerAdapter{svc: registrySvc},
		hub,
		runnerAdapter{dispatchSvc: dispatchSvc, runsSvc: runsSvc},
		clk,
	)
	return &dmHarness{handler: NewConnectHandler(Deps{Service: svc}), runs: runsSvc, hub: hub}
}

// settleRun simulates the node-agent streaming a terminal EXIT for a run.
func (h *dmHarness) settleRun(t *testing.T, runID string, exitCode int32) {
	t.Helper()
	_, err := h.runs.AppendEvent(context.Background(), runs.RunEvent{
		RunID: runID, Kind: runs.EventStatus, Sequence: 1, Status: "running",
	})
	require.NoError(t, err)
	_, err = h.runs.AppendEvent(context.Background(), runs.RunEvent{
		RunID: runID, Kind: runs.EventExit, Sequence: 2, ExitCode: exitCode,
	})
	require.NoError(t, err)
}

func dmRunReq() *connect.Request[gatev1.RunGateRequest] {
	return connect.NewRequest(&gatev1.RunGateRequest{
		Scenario:       "web-search",
		TargetRevision: "a1b2c3d",
		TargetOses:     []string{"linux", "darwin", "windows"},
	})
}

// [REQ:BRG-P1-002] deployment-manager invokes the cross-OS gate over the API
// contract: bridge dispatches the validation on a real node per OS, and once all
// three finish green the gate's structured result is PASSED — "production-ready
// on Ubuntu + macOS + Windows". Bridge supplies the capability; the verdict that
// gates promotion is deployment-manager's to own.
func TestDeploymentManagerGate_AllGreenIsProductionReady(t *testing.T) {
	h := newDMHarness(t)

	resp, err := h.handler.RunGate(ownerCtx(), dmRunReq())
	require.NoError(t, err)
	require.Len(t, resp.Msg.Results, 3, "one validation run dispatched per target OS")
	require.Equal(t, gatev1.GateVerdict_GATE_VERDICT_PENDING, resp.Msg.Verdict)

	// Each node's validation suite finishes green.
	for _, r := range resp.Msg.Results {
		require.NotEmpty(t, r.RunId, "OS %s was dispatched a durable run", r.Os)
		h.settleRun(t, r.RunId, 0)
	}

	// deployment-manager blocks once on the gate and reads the structured verdict.
	got, err := h.handler.WaitGate(ownerCtx(), connect.NewRequest(&gatev1.WaitGateRequest{Id: resp.Msg.GateId, TimeoutSeconds: 30}))
	require.NoError(t, err)
	require.False(t, got.Msg.TimedOut)
	require.Equal(t, gatev1.GateVerdict_GATE_VERDICT_PASSED, got.Msg.Gate.Verdict)
	require.Equal(t, int32(3), got.Msg.Gate.Passed)
	require.Equal(t, int32(0), got.Msg.Gate.Failed)
	for _, r := range got.Msg.Results {
		require.Equal(t, gatev1.OSDisposition_OS_DISPOSITION_PASSED, r.Disposition, "OS %s green", r.Os)
	}
}

// [REQ:BRG-P1-002] One OS failing its native validation fails the whole gate, and
// the structured result surfaces the offending OS's run id + exit code so
// deployment-manager can drill into the logs and block promotion.
func TestDeploymentManagerGate_OneOSFailsGate(t *testing.T) {
	h := newDMHarness(t)

	resp, err := h.handler.RunGate(ownerCtx(), dmRunReq())
	require.NoError(t, err)

	var winRun string
	for _, r := range resp.Msg.Results {
		if r.Os == "windows" {
			winRun = r.RunId
			h.settleRun(t, r.RunId, 1) // windows validation fails
		} else {
			h.settleRun(t, r.RunId, 0)
		}
	}
	require.NotEmpty(t, winRun)

	got, err := h.handler.WaitGate(ownerCtx(), connect.NewRequest(&gatev1.WaitGateRequest{Id: resp.Msg.GateId, TimeoutSeconds: 30}))
	require.NoError(t, err)
	require.Equal(t, gatev1.GateVerdict_GATE_VERDICT_FAILED, got.Msg.Gate.Verdict)
	require.Equal(t, int32(1), got.Msg.Gate.Failed)

	var win *gatev1.OSResult
	for _, r := range got.Msg.Results {
		if r.Os == "windows" {
			win = r
		}
	}
	require.NotNil(t, win)
	require.Equal(t, gatev1.OSDisposition_OS_DISPOSITION_FAILED, win.Disposition)
	require.Equal(t, int32(1), win.ExitCode)
	require.Equal(t, winRun, win.RunId, "the failing OS's run id is surfaced for log drill-in")
}

// [REQ:BRG-P1-002] When the fleet has no node for a target OS, the gate fails
// that OS as NO_NODE — deployment-manager learns the scenario cannot be proven
// production-ready there, rather than getting a false green.
func TestDeploymentManagerGate_MissingOSNodeFails(t *testing.T) {
	h := newDMHarness(t)
	// Revoke the windows node so no eligible node runs that OS.
	_, err := h.handler.RunGate(ownerCtx(), connect.NewRequest(&gatev1.RunGateRequest{
		Scenario:       "web-search",
		TargetRevision: "a1b2c3d",
		TargetOses:     []string{"linux", "freebsd"}, // freebsd has no node
	}))
	require.NoError(t, err)

	// The linux run is dispatched; freebsd has no node.
	gates, err := h.handler.ListGates(ownerCtx(), connect.NewRequest(&gatev1.ListGatesRequest{}))
	require.NoError(t, err)
	require.NotEmpty(t, gates.Msg.Gates)
	got, err := h.handler.GetGate(ownerCtx(), connect.NewRequest(&gatev1.GetGateRequest{Id: gates.Msg.Gates[0].Id}))
	require.NoError(t, err)
	require.Equal(t, gatev1.GateVerdict_GATE_VERDICT_FAILED, got.Msg.Gate.Verdict)

	byOS := map[string]gatev1.OSDisposition{}
	for _, r := range got.Msg.Results {
		byOS[r.Os] = r.Disposition
	}
	require.Equal(t, gatev1.OSDisposition_OS_DISPOSITION_NO_NODE, byOS["freebsd"])
}
