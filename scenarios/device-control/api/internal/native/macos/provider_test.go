package macos

import (
	"context"
	"testing"
	"time"

	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
)

type readinessBackend struct{}

func (readinessBackend) Probe(context.Context, string) (Probe, error) {
	return Probe{ActiveUser: "alice", SessionType: "Aqua", DisplayID: "main", CaptureAllowed: true, InputAllowed: true, ClipboardAllowed: true, GeometryRevision: "geometry-1", VP8EncoderAvailable: true}, nil
}
func (readinessBackend) Apply(context.Context, *desktopv1.InputRequest) error { return nil }
func (readinessBackend) ReadClipboard(context.Context) (string, error)        { return "", nil }
func (readinessBackend) WriteClipboard(context.Context, string) error         { return nil }
func (readinessBackend) CaptureVP8(context.Context, string) ([]byte, time.Duration, error) {
	return []byte{0x10, 0x00, 0x00}, 33 * time.Millisecond, nil
}
func (readinessBackend) ReleaseHeld(context.Context) error { return nil }
func (readinessBackend) Close(context.Context) error       { return nil }

func TestClassifyFailsClosedAtEachMacOSBoundary(t *testing.T) {
	tests := []struct {
		name  string
		probe Probe
		want  desktopv1.ReadinessState
	}{
		{"no session", Probe{}, desktopv1.ReadinessState_READINESS_STATE_NO_SESSION},
		{"console user without Aqua session", Probe{ActiveUser: "alice", SessionType: "LoginWindow"}, desktopv1.ReadinessState_READINESS_STATE_NO_SESSION},
		{"permission", Probe{ActiveUser: "alice", SessionType: "Aqua", PermissionCode: "screen_recording_required"}, desktopv1.ReadinessState_READINESS_STATE_PERMISSION_REQUIRED},
		{"display not selected", Probe{ActiveUser: "alice", SessionType: "Aqua", PermissionCode: "display_not_selected"}, desktopv1.ReadinessState_READINESS_STATE_NO_DISPLAY},
		{"clipboard unavailable", Probe{ActiveUser: "alice", SessionType: "Aqua", PermissionCode: "clipboard_unavailable"}, desktopv1.ReadinessState_READINESS_STATE_FAILED},
		{"no display", Probe{ActiveUser: "alice", SessionType: "Aqua"}, desktopv1.ReadinessState_READINESS_STATE_NO_DISPLAY},
		{"clipboard unavailable", Probe{ActiveUser: "alice", SessionType: "Aqua", DisplayID: "display-1", CaptureAllowed: true, InputAllowed: true}, desktopv1.ReadinessState_READINESS_STATE_FAILED},
		{"VP8 encoder unavailable", Probe{ActiveUser: "alice", SessionType: "Aqua", DisplayID: "display-1", CaptureAllowed: true, InputAllowed: true, ClipboardAllowed: true, PermissionCode: "vp8_encoder_unavailable"}, desktopv1.ReadinessState_READINESS_STATE_FAILED},
		{"ready", Probe{ActiveUser: "alice", SessionType: "Aqua", DisplayID: "display-1", CaptureAllowed: true, InputAllowed: true, ClipboardAllowed: true, VP8EncoderAvailable: true}, desktopv1.ReadinessState_READINESS_STATE_READY},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state, _, _ := classify(tt.probe)
			if state != tt.want {
				t.Fatalf("state=%v want=%v", state, tt.want)
			}
		})
	}
}

func TestClassifyUsesActionableRecoveryForTypedMacOSBoundaries(t *testing.T) {
	state, reason, recovery := classify(Probe{ActiveUser: "alice", SessionType: "Aqua", PermissionCode: "display_not_selected"})
	if state != desktopv1.ReadinessState_READINESS_STATE_NO_DISPLAY || reason != "display_not_selected" || recovery != "select_one_available_display" {
		t.Fatalf("display selection = state %v reason %q recovery %q", state, reason, recovery)
	}
	state, reason, recovery = classify(Probe{ActiveUser: "alice", SessionType: "Aqua", PermissionCode: "clipboard_unavailable"})
	if state != desktopv1.ReadinessState_READINESS_STATE_FAILED || reason != "clipboard_unavailable" || recovery != "inspect_companion_health" {
		t.Fatalf("clipboard = state %v reason %q recovery %q", state, reason, recovery)
	}
	state, reason, recovery = classify(Probe{ActiveUser: "alice", SessionType: "Aqua", PermissionCode: "vp8_encoder_unavailable"})
	if state != desktopv1.ReadinessState_READINESS_STATE_FAILED || reason != "vp8_encoder_unavailable" || recovery != "install_ffmpeg_with_libvpx" {
		t.Fatalf("encoder = state %v reason %q recovery %q", state, reason, recovery)
	}
}

func TestReadinessPublishesCompanionVersion(t *testing.T) {
	now := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	provider := Provider{Backend: readinessBackend{}, CompanionVersion: "0.1.0-darwin", Now: func() time.Time { return now }}
	readiness, err := provider.Readiness(context.Background(), "main")
	if err != nil {
		t.Fatal(err)
	}
	if readiness.GetCompanionVersion() != "0.1.0-darwin" || !readiness.GetObservedAt().AsTime().Equal(now) {
		t.Fatalf("readiness = %+v", readiness)
	}
}

func TestProviderExposesNativeVP8Capture(t *testing.T) {
	provider := Provider{Backend: readinessBackend{}}
	payload, duration, err := provider.CaptureVP8(context.Background(), "main")
	if err != nil || len(payload) == 0 || duration <= 0 {
		t.Fatalf("payload=%v duration=%v err=%v", payload, duration, err)
	}
}

func TestProviderExposesNativeLifecycle(t *testing.T) {
	provider := Provider{Backend: readinessBackend{}}
	if err := provider.ReleaseHeld(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := provider.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}
