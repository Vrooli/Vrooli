package desktopwebrtc

import (
	"context"
	"errors"
	"testing"
	"time"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"google.golang.org/protobuf/proto"
)

type frameProvider struct {
	fakeProvider
	beforeCapture func()
}

func (p *frameProvider) CaptureJPEG(context.Context, string) ([]byte, error) {
	if p.beforeCapture != nil {
		p.beforeCapture()
	}
	return []byte("bounded test frame"), nil
}

func TestHTTPSFrameRequiresExactLiveLease(t *testing.T) {
	ctx := context.Background()
	provider := &frameProvider{fakeProvider: fakeProvider{ready: desktopv1.ReadinessState_READINESS_STATE_READY, selectedDisplay: "screen"}}
	m := NewManager(provider)
	now := time.Now()
	m.now = func() time.Time { return now }
	session, err := m.Open(ctx, &desktopv1.OpenSessionRequest{
		Surface: &commonv1.SurfaceRef{Target: &commonv1.TargetRef{ResourceId: "host"}}, DisplayId: "screen",
	})
	if err != nil {
		t.Fatal(err)
	}
	req := &desktopv1.ReadFrameRequest{Session: session.Ref, LeaseId: session.LeaseId, LeaseEpoch: session.LeaseEpoch}
	wrong := proto.Clone(req).(*desktopv1.ReadFrameRequest)
	wrong.Session.Surface.Target.ResourceId = "different-host"
	if _, err := m.ReadFrame(ctx, wrong); !errors.Is(err, ErrNotControl) {
		t.Fatalf("wrong surface = %v", err)
	}
	if frame, err := m.ReadFrame(ctx, req); err != nil || len(frame.GetJpeg()) == 0 {
		t.Fatalf("viewer frame = %v, %v", frame, err)
	}
	if _, err := m.ReadFrame(ctx, req); !errors.Is(err, ErrFrameBusy) {
		t.Fatalf("rapid poll = %v", err)
	}
	now = now.Add(time.Second)
	provider.beforeCapture = func() {
		_, closeErr := m.Close(ctx, &desktopv1.CloseSessionRequest{Session: session.Ref})
		if !errors.Is(closeErr, ErrCleanupPending) {
			t.Fatalf("close during capture = %v", closeErr)
		}
	}
	if _, err := m.ReadFrame(ctx, req); !errors.Is(err, ErrRevoked) {
		t.Fatalf("frame returned after close: %v", err)
	}
	if _, err := m.Close(ctx, &desktopv1.CloseSessionRequest{Session: session.Ref}); err != nil {
		t.Fatalf("closed capture cleanup did not settle: %v", err)
	}
}
