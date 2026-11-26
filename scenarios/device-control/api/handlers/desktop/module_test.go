package desktop

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"device-control/internal/desktopwebrtc"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	desktopconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop/desktopv1connect"
	interactivev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/interactive"
	interactiveconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/interactive/interactive_v1connect"
)

type readyProvider struct{}

func (readyProvider) Readiness(context.Context, string) (*desktopv1.DesktopReadiness, error) {
	return &desktopv1.DesktopReadiness{State: desktopv1.ReadinessState_READINESS_STATE_READY, SelectedDisplayId: "display-1", GeometryRevision: "geometry-1"}, nil
}
func (readyProvider) ApplyInput(context.Context, *desktopv1.InputRequest) error { return nil }
func (readyProvider) ReadClipboard(context.Context) (string, error)             { return "", nil }
func (readyProvider) WriteClipboard(context.Context, string) error              { return nil }
func (readyProvider) ReleaseHeld(context.Context) error                         { return nil }

// SOFTWARE SIMULATION: this exercises the Device Control owner with an
// in-process fake display provider. It does not prove a physical desktop,
// capture device, native input, or network transport.
func TestSoftwareSimulationDesktopSessionServiceLifecycle(t *testing.T) {
	t.Log("SOFTWARE SIMULATION ONLY: in-process Device Control session owner and fake display provider")
	h := &handler{manager: desktopwebrtc.NewManager(readyProvider{})}
	ctx := context.Background()
	surface := &commonv1.SurfaceRef{Target: &commonv1.TargetRef{ResourceId: "mac-node"}}

	opened, err := h.OpenSession(ctx, connect.NewRequest(&desktopv1.OpenSessionRequest{Surface: surface, DisplayId: "display-1", Control: true, Clipboard: true}))
	if err != nil {
		t.Fatalf("open session: %v", err)
	}
	if opened.Msg.GetCodec() != "VP8" || !opened.Msg.GetController() {
		t.Fatalf("unexpected session: %+v", opened.Msg)
	}

	input, err := h.Input(ctx, connect.NewRequest(&desktopv1.InputRequest{Session: opened.Msg.GetRef(), LeaseId: opened.Msg.GetLeaseId(), LeaseEpoch: opened.Msg.GetLeaseEpoch(), DisplayId: opened.Msg.GetSelectedDisplayId(), GeometryRevision: opened.Msg.GetGeometryRevision(), CommandId: "click-1", Action: &desktopv1.Action{}}))
	if err != nil {
		t.Fatalf("input: %v", err)
	}
	if input.Msg.GetOutcome() != desktopv1.InputOutcome_INPUT_OUTCOME_ACCEPTED {
		t.Fatalf("unexpected input receipt: %+v", input.Msg)
	}

	released, err := h.ReleaseControl(ctx, connect.NewRequest(&desktopv1.ReleaseControlRequest{Session: opened.Msg.GetRef(), LeaseId: opened.Msg.GetLeaseId(), LeaseEpoch: opened.Msg.GetLeaseEpoch()}))
	if err != nil {
		t.Fatalf("release control: %v", err)
	}
	if released.Msg.GetController() || released.Msg.GetLeaseEpoch() != opened.Msg.GetLeaseEpoch()+1 {
		t.Fatalf("unexpected hand-back state: %+v", released.Msg)
	}

	closed, err := h.CloseSession(ctx, connect.NewRequest(&desktopv1.CloseSessionRequest{Session: opened.Msg.GetRef()}))
	if err != nil || !closed.Msg.GetClosed() {
		t.Fatalf("close: response=%v err=%v", closed.Msg, err)
	}
}

func TestDesktopSessionServiceMapsReadinessAndInvalidCalls(t *testing.T) {
	h := &handler{manager: desktopwebrtc.NewManager(readyProvider{})}
	readiness, err := h.GetReadiness(context.Background(), connect.NewRequest(&desktopv1.GetReadinessRequest{Surface: &commonv1.SurfaceRef{}, DisplayId: "display-1"}))
	if err != nil || readiness.Msg.GetState() != desktopv1.ReadinessState_READINESS_STATE_READY {
		t.Fatalf("readiness: response=%v err=%v", readiness.Msg, err)
	}

	_, err = h.OpenSession(context.Background(), connect.NewRequest(&desktopv1.OpenSessionRequest{Surface: &commonv1.SurfaceRef{}, DisplayId: ""}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("invalid open code = %v, want invalid argument", connect.CodeOf(err))
	}

}

type composedStandInProvider struct {
	state         desktopv1.ReadinessState
	clipboard     string
	acceptedInput int
}

func (p *composedStandInProvider) Readiness(context.Context, string) (*desktopv1.DesktopReadiness, error) {
	return &desktopv1.DesktopReadiness{
		State: p.state, SelectedDisplayId: "simulation-display", GeometryRevision: "geometry-simulation",
		CaptureAllowed:   p.state == desktopv1.ReadinessState_READINESS_STATE_READY,
		InputAllowed:     p.state == desktopv1.ReadinessState_READINESS_STATE_READY,
		ClipboardAllowed: p.state == desktopv1.ReadinessState_READINESS_STATE_READY,
		Displays:         []*desktopv1.Display{{Id: "simulation-display", Width: 1920, Height: 1080, Selected: true}},
	}, nil
}
func (p *composedStandInProvider) ApplyInput(context.Context, *desktopv1.InputRequest) error {
	p.acceptedInput++
	return nil
}
func (p *composedStandInProvider) ReadClipboard(context.Context) (string, error) {
	return p.clipboard, nil
}
func (p *composedStandInProvider) WriteClipboard(_ context.Context, text string) error {
	p.clipboard = text
	return nil
}
func (p *composedStandInProvider) ReleaseHeld(context.Context) error { return nil }

func TestSoftwareSimulationDesktopSessionHTTPContract(t *testing.T) {
	t.Log("SOFTWARE SIMULATION ONLY: in-process Device Control HTTP owner over a fake display provider")
	provider := &composedStandInProvider{state: desktopv1.ReadinessState_READINESS_STATE_PERMISSION_REQUIRED, clipboard: "synthetic clipboard"}
	servicePath, service := desktopconnect.NewDesktopSessionServiceHandler(&handler{manager: desktopwebrtc.NewManager(provider)})
	router := mux.NewRouter()
	connectx.RegisterServices(router, connectx.ServiceMount{Path: servicePath, Handler: service})
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	client := desktopconnect.NewDesktopSessionServiceClient(server.Client(), server.URL)
	ctx := context.Background()
	surface := &commonv1.SurfaceRef{Target: &commonv1.TargetRef{ResourceId: "software-simulation-node"}}

	blocked, err := client.GetReadiness(ctx, connect.NewRequest(&desktopv1.GetReadinessRequest{Surface: surface, DisplayId: "simulation-display"}))
	if err != nil || blocked.Msg.GetState() != desktopv1.ReadinessState_READINESS_STATE_PERMISSION_REQUIRED {
		t.Fatalf("permission readiness: response=%v err=%v", blocked, err)
	}
	_, err = client.OpenSession(ctx, connect.NewRequest(&desktopv1.OpenSessionRequest{Surface: surface, DisplayId: "simulation-display", Control: true, Clipboard: true, RequestId: "simulation-blocked-open"}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("blocked open code=%v err=%v, want failed precondition", connect.CodeOf(err), err)
	}

	provider.state = desktopv1.ReadinessState_READINESS_STATE_READY
	opened, err := client.OpenSession(ctx, connect.NewRequest(&desktopv1.OpenSessionRequest{Surface: surface, DisplayId: "simulation-display", Control: false, Clipboard: true, RequestId: "simulation-viewer-open"}))
	if err != nil || opened.Msg.GetController() {
		t.Fatalf("viewer open: response=%v err=%v", opened, err)
	}
	attached, err := client.AttachViewer(ctx, connect.NewRequest(&desktopv1.AttachViewerRequest{Session: opened.Msg.GetRef(), RequestId: "simulation-watch"}))
	if err != nil || attached.Msg.GetViewerCount() < 1 {
		t.Fatalf("attach viewer: response=%v err=%v", attached, err)
	}
	controlled, err := client.TakeControl(ctx, connect.NewRequest(&desktopv1.TakeControlRequest{Session: opened.Msg.GetRef(), Takeover: true}))
	if err != nil || !controlled.Msg.GetController() || controlled.Msg.GetLeaseEpoch() == 0 {
		t.Fatalf("take control: response=%v err=%v", controlled, err)
	}
	input, err := client.Input(ctx, connect.NewRequest(&desktopv1.InputRequest{
		Session: controlled.Msg.GetRef(), LeaseId: controlled.Msg.GetLeaseId(), LeaseEpoch: controlled.Msg.GetLeaseEpoch(),
		DisplayId: controlled.Msg.GetSelectedDisplayId(), GeometryRevision: controlled.Msg.GetGeometryRevision(), CommandId: "simulation-pointer",
		Action: &desktopv1.Action{Action: &desktopv1.Action_Pointer{Pointer: &desktopv1.PointerAction{Kind: desktopv1.PointerAction_KIND_CLICK, Button: desktopv1.PointerAction_BUTTON_PRIMARY, DisplayId: "simulation-display", X: 900, Y: 500}}},
	}))
	if err != nil || input.Msg.GetOutcome() != desktopv1.InputOutcome_INPUT_OUTCOME_ACCEPTED || provider.acceptedInput != 1 {
		t.Fatalf("bounded owner input: receipt=%v accepted=%d err=%v", input, provider.acceptedInput, err)
	}
	write, err := client.Clipboard(ctx, connect.NewRequest(&desktopv1.ClipboardRequest{
		Session: controlled.Msg.GetRef(), LeaseId: controlled.Msg.GetLeaseId(), LeaseEpoch: controlled.Msg.GetLeaseEpoch(),
		Direction: desktopv1.ClipboardDirection_CLIPBOARD_DIRECTION_TO_DESKTOP, Text: "synthetic clipboard input", CommandId: "simulation-clipboard-write",
	}))
	if err != nil || write.Msg.GetOutcome() != desktopv1.InputOutcome_INPUT_OUTCOME_ACCEPTED || provider.clipboard != "synthetic clipboard input" {
		t.Fatalf("clipboard write receipt=%v err=%v", write, err)
	}
	read, err := client.Clipboard(ctx, connect.NewRequest(&desktopv1.ClipboardRequest{
		Session: controlled.Msg.GetRef(), LeaseId: controlled.Msg.GetLeaseId(), LeaseEpoch: controlled.Msg.GetLeaseEpoch(),
		Direction: desktopv1.ClipboardDirection_CLIPBOARD_DIRECTION_FROM_DESKTOP, CommandId: "simulation-clipboard-read",
	}))
	if err != nil || read.Msg.GetText() != "synthetic clipboard input" {
		t.Fatalf("clipboard read receipt=%v err=%v", read, err)
	}
	released, err := client.ReleaseControl(ctx, connect.NewRequest(&desktopv1.ReleaseControlRequest{Session: controlled.Msg.GetRef(), LeaseId: controlled.Msg.GetLeaseId(), LeaseEpoch: controlled.Msg.GetLeaseEpoch()}))
	if err != nil || released.Msg.GetController() || released.Msg.GetLeaseEpoch() <= controlled.Msg.GetLeaseEpoch() {
		t.Fatalf("hand-back: response=%v err=%v", released, err)
	}
	closed, err := client.CloseSession(ctx, connect.NewRequest(&desktopv1.CloseSessionRequest{Session: opened.Msg.GetRef(), Reason: "software_simulation_complete"}))
	if err != nil || !closed.Msg.GetClosed() {
		t.Fatalf("close: response=%v err=%v", closed, err)
	}
	reconnected, err := client.OpenSession(ctx, connect.NewRequest(&desktopv1.OpenSessionRequest{Surface: surface, DisplayId: "simulation-display", Control: false, Clipboard: true, RequestId: "simulation-fresh-session"}))
	if err != nil || reconnected.Msg.GetRef().GetSessionId() == opened.Msg.GetRef().GetSessionId() {
		t.Fatalf("fresh session after close: response=%v err=%v", reconnected, err)
	}
	if _, err = client.CloseSession(ctx, connect.NewRequest(&desktopv1.CloseSessionRequest{Session: reconnected.Msg.GetRef(), Reason: "software_simulation_complete"})); err != nil {
		t.Fatalf("fresh session close: %v", err)
	}
}

// TestSoftwareSimulationCrossProcessBridgeComposition opens a real
// Device Control owner session, then uses its exact session/lease identity in
// a separate Bridge owner process over the public Connect HTTP contracts.
// Both the display provider and Bridge actor/node trust are synthetic.
func TestSoftwareSimulationCrossProcessBridgeComposition(t *testing.T) {
	t.Log("SOFTWARE SIMULATION ONLY: Device Control HTTP owner composed with Bridge HTTP owner in separate OS processes")
	bridgeDir := os.Getenv("MB4_BRIDGE_API_DIR")
	if bridgeDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		bridgeDir = filepath.Clean(filepath.Join(cwd, "../../../../vrooli-bridge/api"))
	}
	temp := t.TempDir()
	readyFile := filepath.Join(temp, "bridge-ready")
	shutdownFile := filepath.Join(temp, "bridge-shutdown")
	cmd := exec.Command("go", "test", "./handlers/interactive", "-run", "^TestSoftwareSimulationBridgeOwnerProcess$", "-count=1", "-v")
	cmd.Dir = bridgeDir
	cmd.Env = append(os.Environ(), "MB4_BRIDGE_READY_FILE="+readyFile, "MB4_BRIDGE_SHUTDOWN_FILE="+shutdownFile)
	var childLog strings.Builder
	cmd.Stdout, cmd.Stderr = &childLog, &childLog
	if err := cmd.Start(); err != nil {
		t.Fatalf("start Bridge owner test process: %v", err)
	}
	var childWaitErr error
	t.Cleanup(func() {
		_ = os.WriteFile(shutdownFile, []byte("stop"), 0600)
		if childWaitErr == nil {
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			select {
			case childWaitErr = <-done:
			case <-time.After(3 * time.Second):
				_ = cmd.Process.Kill()
				childWaitErr = <-done
			}
		}
		if childWaitErr != nil {
			t.Errorf("Bridge owner process: %v\n%s", childWaitErr, childLog.String())
		} else {
			t.Logf("Bridge owner child process receipt:\n%s", childLog.String())
		}
	})
	var bridgeURL string
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		body, err := os.ReadFile(readyFile)
		if err == nil {
			bridgeURL = strings.TrimSpace(string(body))
			break
		}
		if cmd.ProcessState != nil {
			t.Fatalf("Bridge owner exited before ready: %v\n%s", cmd.ProcessState, childLog.String())
		}
		time.Sleep(50 * time.Millisecond)
	}
	if bridgeURL == "" {
		t.Fatalf("Bridge owner did not become ready\n%s", childLog.String())
	}

	provider := &composedStandInProvider{state: desktopv1.ReadinessState_READINESS_STATE_READY, clipboard: "synthetic clipboard"}
	servicePath, service := desktopconnect.NewDesktopSessionServiceHandler(&handler{manager: desktopwebrtc.NewManager(provider)})
	router := mux.NewRouter()
	connectx.RegisterServices(router, connectx.ServiceMount{Path: servicePath, Handler: service})
	deviceServer := httptest.NewServer(router)
	t.Cleanup(deviceServer.Close)
	ctx := context.Background()
	device := desktopconnect.NewDesktopSessionServiceClient(deviceServer.Client(), deviceServer.URL)
	bridge := interactiveconnect.NewInteractiveDesktopServiceClient(&http.Client{Timeout: 5 * time.Second}, bridgeURL)
	const nodeID = "software-simulation-node"
	surface := &commonv1.SurfaceRef{SurfaceId: "desktop", OwnerScenario: "device-control", Target: &commonv1.TargetRef{ResourceId: nodeID, HostNodeId: nodeID}}
	opened, err := device.OpenSession(ctx, connect.NewRequest(&desktopv1.OpenSessionRequest{Surface: surface, DisplayId: "simulation-display", Control: true, Clipboard: true, RequestId: "cross-process-open"}))
	if err != nil || !opened.Msg.GetController() {
		t.Fatalf("Device Control open: response=%v err=%v", opened, err)
	}
	deviceRef := opened.Msg.GetRef()
	channelReq := &interactivev1.OpenChannelRequest{NodeId: nodeID, Surface: surface, SessionId: deviceRef.GetSessionId(), LeaseId: opened.Msg.GetLeaseId(), LeaseEpoch: opened.Msg.GetLeaseEpoch(), Protocol: interactivev1.ChannelProtocol_CHANNEL_PROTOCOL_WEBRTC_VP8, Role: interactivev1.ChannelRole_CHANNEL_ROLE_CONTROLLER, RequestId: "cross-process-channel"}
	channel, err := bridge.OpenChannel(ctx, connect.NewRequest(channelReq))
	if err != nil || channel.Msg.GetGrant().GetSessionId() != deviceRef.GetSessionId() || channel.Msg.GetGrant().GetLeaseId() != opened.Msg.GetLeaseId() || channel.Msg.GetGrant().GetLeaseEpoch() != opened.Msg.GetLeaseEpoch() {
		t.Fatalf("Bridge failed to admit exact Device Control session/lease identity: response=%v err=%v", channel, err)
	}
	signal, err := bridge.Signal(ctx, connect.NewRequest(&interactivev1.SignalRequest{Grant: channel.Msg.GetGrant(), Kind: interactivev1.SignalKind_SIGNAL_KIND_OFFER, Generation: "simulation-generation-1", RequestId: "cross-process-offer", Payload: []byte("synthetic-offer")}))
	if err != nil || !signal.Msg.GetAccepted() {
		t.Fatalf("Bridge signal: response=%v err=%v", signal, err)
	}
	input, err := device.Input(ctx, connect.NewRequest(&desktopv1.InputRequest{Session: deviceRef, LeaseId: opened.Msg.GetLeaseId(), LeaseEpoch: opened.Msg.GetLeaseEpoch(), DisplayId: opened.Msg.GetSelectedDisplayId(), GeometryRevision: opened.Msg.GetGeometryRevision(), CommandId: "cross-process-input", Action: &desktopv1.Action{}}))
	if err != nil || input.Msg.GetOutcome() != desktopv1.InputOutcome_INPUT_OUTCOME_ACCEPTED {
		t.Fatalf("Device Control input: response=%v err=%v", input, err)
	}
	clip, err := device.Clipboard(ctx, connect.NewRequest(&desktopv1.ClipboardRequest{Session: deviceRef, LeaseId: opened.Msg.GetLeaseId(), LeaseEpoch: opened.Msg.GetLeaseEpoch(), Direction: desktopv1.ClipboardDirection_CLIPBOARD_DIRECTION_FROM_DESKTOP, CommandId: "cross-process-clipboard"}))
	if err != nil || clip.Msg.GetText() != "synthetic clipboard" {
		t.Fatalf("Device Control clipboard: response=%v err=%v", clip, err)
	}

	// A network drop closes the Bridge channel while the Device Control
	// session remains owned. Reconnect reuses the same lease identity.
	closed, err := bridge.CloseChannel(ctx, connect.NewRequest(&interactivev1.CloseChannelRequest{Grant: channel.Msg.GetGrant(), Reason: "software_simulation_drop"}))
	if err != nil || !closed.Msg.GetClosed() {
		t.Fatalf("Bridge drop close: response=%v err=%v", closed, err)
	}
	if _, err = bridge.Signal(ctx, connect.NewRequest(&interactivev1.SignalRequest{Grant: channel.Msg.GetGrant(), Kind: interactivev1.SignalKind_SIGNAL_KIND_ICE, Generation: "simulation-generation-1", RequestId: "stale-after-drop", Payload: []byte("synthetic-ice")})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("stale post-drop signal code=%v err=%v", connect.CodeOf(err), err)
	}
	channelReq.RequestId = "cross-process-reconnect"
	channelReq.Takeover = true
	reconnected, err := bridge.OpenChannel(ctx, connect.NewRequest(channelReq))
	if err != nil || reconnected.Msg.GetGrant().GetChannelId() == channel.Msg.GetGrant().GetChannelId() {
		t.Fatalf("Bridge reconnect: response=%v err=%v", reconnected, err)
	}

	released, err := device.ReleaseControl(ctx, connect.NewRequest(&desktopv1.ReleaseControlRequest{Session: deviceRef, LeaseId: opened.Msg.GetLeaseId(), LeaseEpoch: opened.Msg.GetLeaseEpoch()}))
	if err != nil || released.Msg.GetController() || released.Msg.GetLeaseEpoch() != opened.Msg.GetLeaseEpoch()+1 {
		t.Fatalf("Device Control handback: response=%v err=%v", released, err)
	}
	revoked, err := bridge.RevokeChannel(ctx, connect.NewRequest(&interactivev1.RevokeChannelRequest{ChannelId: reconnected.Msg.GetGrant().GetChannelId(), Reason: interactivev1.RevokeReason_REVOKE_REASON_OPERATOR, Detail: "software_simulation_handback"}))
	if err != nil || !revoked.Msg.GetRevoked() {
		t.Fatalf("Bridge revoke on handback: response=%v err=%v", revoked, err)
	}
	if _, err = bridge.Signal(ctx, connect.NewRequest(&interactivev1.SignalRequest{Grant: reconnected.Msg.GetGrant(), Kind: interactivev1.SignalKind_SIGNAL_KIND_ICE, Generation: "simulation-generation-2", RequestId: "stale-after-handback", Payload: []byte("synthetic-ice")})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("revoked Bridge signal code=%v err=%v", connect.CodeOf(err), err)
	}
	deviceClosed, err := device.CloseSession(ctx, connect.NewRequest(&desktopv1.CloseSessionRequest{Session: deviceRef}))
	if err != nil || !deviceClosed.Msg.GetClosed() {
		t.Fatalf("Device Control cleanup: response=%v err=%v", deviceClosed, err)
	}
}
