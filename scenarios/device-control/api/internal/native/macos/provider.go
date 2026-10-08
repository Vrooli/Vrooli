// Package macos adapts the native companion probes to the browser-session
// provider. OS-specific capture and CGEvent code stays in native/host; this
// package owns the readiness vocabulary and fail-closed mapping consumed by
// Device Control.
package macos

import (
	"context"
	"errors"
	"strings"
	"time"

	"device-control/internal/desktopwebrtc"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Probe struct {
	ActiveUser          string
	SessionType         string
	DisplayID           string
	DisplayName         string
	Width, Height       uint32
	Scale               float32
	CaptureAllowed      bool
	InputAllowed        bool
	ClipboardAllowed    bool
	VP8EncoderAvailable bool
	PermissionCode      string
	GeometryRevision    string
}

type Backend interface {
	Probe(context.Context, string) (Probe, error)
	Apply(context.Context, *desktopv1.InputRequest) error
	ReadClipboard(context.Context) (string, error)
	WriteClipboard(context.Context, string) error
}

type vp8Backend interface {
	CaptureVP8(context.Context, string) ([]byte, time.Duration, error)
}

type lifecycleBackend interface {
	ReleaseHeld(context.Context) error
	Close(context.Context) error
}

type Provider struct {
	Backend          Backend
	CompanionVersion string
	Now              func() time.Time
}

func (p Provider) Readiness(ctx context.Context, displayID string) (*desktopv1.DesktopReadiness, error) {
	if p.Backend == nil {
		return nil, errors.New("macOS companion backend unavailable")
	}
	probe, err := p.Backend.Probe(ctx, displayID)
	if err != nil {
		return nil, err
	}
	state, reason, recovery := classify(probe)
	now := time.Now().UTC()
	if p.Now != nil {
		now = p.Now()
	}
	readiness := &desktopv1.DesktopReadiness{State: state, ReasonCode: reason, Recovery: recovery, ActiveUser: probe.ActiveUser, SessionType: probe.SessionType, SelectedDisplayId: probe.DisplayID, CaptureAllowed: probe.CaptureAllowed, InputAllowed: probe.InputAllowed, ClipboardAllowed: probe.ClipboardAllowed, CompanionVersion: p.CompanionVersion, GeometryRevision: probe.GeometryRevision, ObservedAt: timestamppb.New(now), ExpiresAt: timestamppb.New(now.Add(10 * time.Second))}
	if probe.DisplayID != "" {
		readiness.Displays = []*desktopv1.Display{{Id: probe.DisplayID, Name: probe.DisplayName, Width: probe.Width, Height: probe.Height, Scale: probe.Scale, Selected: true}}
	}
	return readiness, nil
}
func (p Provider) ApplyInput(ctx context.Context, req *desktopv1.InputRequest) error {
	if p.Backend == nil {
		return errors.New("macOS companion backend unavailable")
	}
	return p.Backend.Apply(ctx, req)
}
func (p Provider) ReadClipboard(ctx context.Context) (string, error) {
	if p.Backend == nil {
		return "", errors.New("macOS companion backend unavailable")
	}
	return p.Backend.ReadClipboard(ctx)
}
func (p Provider) WriteClipboard(ctx context.Context, text string) error {
	if p.Backend == nil {
		return errors.New("macOS companion backend unavailable")
	}
	return p.Backend.WriteClipboard(ctx, text)
}

// CaptureVP8 exposes the native capture/encode seam to Device Control's
// ephemeral WebRTC session manager without widening the general provider
// interface used for readiness and input policy.
func (p Provider) CaptureVP8(ctx context.Context, displayID string) ([]byte, time.Duration, error) {
	source, ok := p.Backend.(vp8Backend)
	if !ok {
		return nil, 0, errors.New("macOS VP8 capture backend unavailable")
	}
	return source.CaptureVP8(ctx, displayID)
}

func (p Provider) ReleaseHeld(ctx context.Context) error {
	lifecycle, ok := p.Backend.(lifecycleBackend)
	if !ok {
		return errors.New("macOS input lifecycle backend unavailable")
	}
	return lifecycle.ReleaseHeld(ctx)
}

func (p Provider) Close(ctx context.Context) error {
	lifecycle, ok := p.Backend.(lifecycleBackend)
	if !ok {
		return errors.New("macOS provider lifecycle backend unavailable")
	}
	return lifecycle.Close(ctx)
}

func classify(p Probe) (desktopv1.ReadinessState, string, string) {
	if strings.TrimSpace(p.ActiveUser) == "" || strings.TrimSpace(p.SessionType) != "Aqua" {
		return desktopv1.ReadinessState_READINESS_STATE_NO_SESSION, "no_aqua_session", "log_in_to_the_approved_gui_user"
	}
	if p.PermissionCode == "display_not_selected" {
		return desktopv1.ReadinessState_READINESS_STATE_NO_DISPLAY, "display_not_selected", "select_one_available_display"
	}
	if p.PermissionCode == "clipboard_unavailable" {
		return desktopv1.ReadinessState_READINESS_STATE_FAILED, "clipboard_unavailable", "inspect_companion_health"
	}
	if p.PermissionCode == "vp8_encoder_unavailable" {
		return desktopv1.ReadinessState_READINESS_STATE_FAILED, "vp8_encoder_unavailable", "install_ffmpeg_with_libvpx"
	}
	if p.PermissionCode != "" {
		return desktopv1.ReadinessState_READINESS_STATE_PERMISSION_REQUIRED, p.PermissionCode, "approve_screen_recording_and_accessibility_for_the_companion"
	}
	if p.DisplayID == "" {
		return desktopv1.ReadinessState_READINESS_STATE_NO_DISPLAY, "no_display", "attach_or_select_one_display"
	}
	if !p.VP8EncoderAvailable {
		return desktopv1.ReadinessState_READINESS_STATE_FAILED, "vp8_encoder_unavailable", "install_ffmpeg_with_libvpx"
	}
	if !p.CaptureAllowed || !p.InputAllowed || !p.ClipboardAllowed {
		return desktopv1.ReadinessState_READINESS_STATE_FAILED, "native_capability_unavailable", "inspect_companion_health"
	}
	return desktopv1.ReadinessState_READINESS_STATE_READY, "ready", ""
}

var _ desktopwebrtc.Provider = Provider{}
var _ desktopwebrtc.VP8Source = Provider{}
var _ interface{ ReleaseHeld(context.Context) error } = Provider{}
var _ interface{ Close(context.Context) error } = Provider{}
