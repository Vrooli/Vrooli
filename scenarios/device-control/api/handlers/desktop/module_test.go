package desktop

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"device-control/internal/desktopwebrtc"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
)

type readyProvider struct{}

func (readyProvider) Readiness(context.Context, string) (*desktopv1.DesktopReadiness, error) {
	return &desktopv1.DesktopReadiness{State: desktopv1.ReadinessState_READINESS_STATE_READY, SelectedDisplayId: "display-1", GeometryRevision: "geometry-1"}, nil
}
func (readyProvider) ApplyInput(context.Context, *desktopv1.InputRequest) error { return nil }
func (readyProvider) ReadClipboard(context.Context) (string, error)             { return "", nil }
func (readyProvider) WriteClipboard(context.Context, string) error              { return nil }

func TestDesktopSessionServiceDelegatesTypedSessionLifecycle(t *testing.T) {
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
