package macos

import (
	"context"
	"errors"
	"strings"
	"time"

	"device-control/internal/native/host"
	"device-control/internal/sessions"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"google.golang.org/protobuf/encoding/protojson"
)

// HostBackend adapts the existing authenticated native user-session helper to
// the desktop-session provider. The helper remains the only owner of capture,
// input, and clipboard effects; this adapter only translates typed requests.
type HostBackend struct {
	native *host.Backend
}

func NewHostBackend(sessionID string) (*HostBackend, error) {
	native, err := host.NewForSession(sessionID)
	if err != nil {
		return nil, err
	}
	return &HostBackend{native: native}, nil
}

func (b *HostBackend) Probe(ctx context.Context, displayID string) (Probe, error) {
	if b == nil || b.native == nil {
		return Probe{}, errors.New("native companion unavailable")
	}
	observed, err := b.native.Probe(ctx)
	if err != nil {
		return Probe{}, err
	}
	if observed.PermissionCode != "" {
		return Probe{ActiveUser: observed.ActiveUser, SessionType: observed.SessionType, PermissionCode: observed.PermissionCode}, nil
	}
	if !observed.VP8EncoderAvailable {
		return Probe{ActiveUser: observed.ActiveUser, SessionType: observed.SessionType, PermissionCode: "vp8_encoder_unavailable"}, nil
	}
	if displayID != "" && displayID != observed.DisplayID {
		return Probe{ActiveUser: observed.ActiveUser, SessionType: observed.SessionType, PermissionCode: "display_not_selected"}, nil
	}
	return Probe{ActiveUser: observed.ActiveUser, SessionType: observed.SessionType, DisplayID: observed.DisplayID, DisplayName: observed.DisplayID, Width: observed.Width, Height: observed.Height, Scale: 1, CaptureAllowed: observed.PermissionCode == "", InputAllowed: observed.PermissionCode == "", ClipboardAllowed: observed.ClipboardAllowed, VP8EncoderAvailable: observed.VP8EncoderAvailable, PermissionCode: observed.PermissionCode, GeometryRevision: observed.GeometryRevision}, nil
}

func (b *HostBackend) Apply(ctx context.Context, req *desktopv1.InputRequest) error {
	if b == nil || b.native == nil || req == nil || req.GetAction() == nil {
		return errors.New("invalid native input request")
	}
	payload, err := protojson.Marshal(req.GetAction())
	if err != nil {
		return err
	}
	return b.native.Apply(ctx, sessions.DesktopCommand{ID: req.GetCommandId(), GeometryRevision: req.GetGeometryRevision(), Payload: payload})
}

func (b *HostBackend) ReadClipboard(ctx context.Context) (string, error) {
	if b == nil || b.native == nil {
		return "", errors.New("native companion unavailable")
	}
	return b.native.ReadClipboard(ctx)
}

func (b *HostBackend) WriteClipboard(ctx context.Context, text string) error {
	if b == nil || b.native == nil {
		return errors.New("native companion unavailable")
	}
	return b.native.WriteClipboard(ctx, text)
}

func (b *HostBackend) ReleaseHeld(ctx context.Context) error {
	if b == nil || b.native == nil {
		return errors.New("native companion unavailable")
	}
	return b.native.ReleaseHeld(ctx)
}

func (b *HostBackend) Close(context.Context) error {
	if b == nil || b.native == nil {
		return nil
	}
	return b.native.Close()
}

func (b *HostBackend) CaptureVP8(ctx context.Context, displayID string) ([]byte, time.Duration, error) {
	if b == nil || b.native == nil {
		return nil, 0, errors.New("native companion unavailable")
	}
	if strings.TrimSpace(displayID) != "main" {
		return nil, 0, errors.New("requested display is not the selected main display")
	}
	return b.native.CaptureVP8(ctx)
}

var _ Backend = (*HostBackend)(nil)
