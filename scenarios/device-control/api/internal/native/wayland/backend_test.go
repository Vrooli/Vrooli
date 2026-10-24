package wayland

import (
	"context"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"device-control/internal/sessions"
	"github.com/godbus/dbus/v5"
	"github.com/stretchr/testify/require"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"google.golang.org/protobuf/encoding/protojson"
)

func testPNG(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "portal.png")
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, png.Encode(f, image.NewRGBA(image.Rect(0, 0, 8, 6))))
	require.NoError(t, f.Close())
	return path
}

func testAction(t *testing.T, action *desktopv1.Action) []byte {
	t.Helper()
	data, err := protojson.Marshal(action)
	require.NoError(t, err)
	return data
}

func exercisePortalFlow(t *testing.T, compositor string) {
	path := testPNG(t)
	var requests []string
	var calls []string
	b := &Backend{displayID: "wayland-" + compositor + "-portal"}
	b.requestFn = func(_ context.Context, iface, method string, _ ...any) (map[string]dbus.Variant, error) {
		requests = append(requests, iface+"."+method)
		switch iface + "." + method {
		case "org.freedesktop.portal.Screenshot.Screenshot":
			return map[string]dbus.Variant{"uri": dbus.MakeVariant("file://" + path)}, nil
		case "org.freedesktop.portal.RemoteDesktop.CreateSession":
			return map[string]dbus.Variant{"session_handle": dbus.MakeVariant(dbus.ObjectPath("/org/freedesktop/portal/desktop/session/1"))}, nil
		case "org.freedesktop.portal.RemoteDesktop.SelectDevices":
			return map[string]dbus.Variant{}, nil
		case "org.freedesktop.portal.ScreenCast.SelectSources":
			return map[string]dbus.Variant{}, nil
		case "org.freedesktop.portal.RemoteDesktop.Start":
			streams := []struct {
				ID         uint32
				Properties map[string]dbus.Variant
			}{{ID: 17}}
			return map[string]dbus.Variant{"streams": dbus.MakeVariant(streams)}, nil
		default:
			t.Fatalf("unexpected portal request %s.%s", iface, method)
			return nil, ErrUnavailable
		}
	}
	b.callFn = func(_ context.Context, method string, _ ...any) error {
		calls = append(calls, method)
		return nil
	}
	snapshot, err := b.Observe(context.Background())
	require.NoError(t, err)
	require.Equal(t, 8, snapshot.Image.Bounds().Dx())
	require.Equal(t, "wayland-"+compositor+"-portal", snapshot.DisplayID)
	action := &desktopv1.Action{Action: &desktopv1.Action_Key{Key: &desktopv1.KeyAction{Kind: desktopv1.KeyAction_KIND_PRESS, Key: "A"}}}
	command := sessions.DesktopCommand{GeometryRevision: snapshot.GeometryRevision, Payload: testAction(t, action)}
	require.NoError(t, b.Validate(context.Background(), command))
	require.NoError(t, b.Apply(context.Background(), command))
	require.Equal(t, []string{"NotifyKeyboardKeysym", "NotifyKeyboardKeysym"}, calls)
	wheel := &desktopv1.Action{Action: &desktopv1.Action_Wheel{Wheel: &desktopv1.WheelAction{DisplayId: snapshot.DisplayID, X: 2, Y: 3, VerticalTicks: 1}}}
	wheelCommand := sessions.DesktopCommand{GeometryRevision: snapshot.GeometryRevision, Payload: testAction(t, wheel)}
	require.NoError(t, b.Validate(context.Background(), wheelCommand))
	require.NoError(t, b.Apply(context.Background(), wheelCommand))
	require.Equal(t, []string{"NotifyKeyboardKeysym", "NotifyKeyboardKeysym", "NotifyPointerMotionAbsolute", "NotifyPointerAxis"}, calls)
	require.Equal(t, []string{"org.freedesktop.portal.Screenshot.Screenshot", "org.freedesktop.portal.RemoteDesktop.CreateSession", "org.freedesktop.portal.RemoteDesktop.SelectDevices", "org.freedesktop.portal.ScreenCast.SelectSources", "org.freedesktop.portal.RemoteDesktop.Start"}, requests)
}

func TestPortalBackendCapturesAndUsesGrantedRemoteDesktop(t *testing.T) {
	exercisePortalFlow(t, "gnome")
}

func TestAcceptanceNAT06GNOMEWaylandPortalCaptureAndInput(t *testing.T) {
	exercisePortalFlow(t, "gnome")
}

func TestAcceptanceNAT07KDEWaylandPortalCaptureAndInput(t *testing.T) {
	exercisePortalFlow(t, "kde")
}

func TestPortalBackendAbsolutePointerRequiresAStream(t *testing.T) {
	b := &Backend{displayID: "wayland-kde-portal", width: 10, height: 10, revision: "r", requestFn: func(context.Context, string, string, ...any) (map[string]dbus.Variant, error) {
		return map[string]dbus.Variant{}, nil
	}, callFn: func(context.Context, string, ...any) error { return nil }}
	b.started = true
	b.session = dbus.ObjectPath("/session")
	action := &desktopv1.Action{Action: &desktopv1.Action_Pointer{Pointer: &desktopv1.PointerAction{Kind: desktopv1.PointerAction_KIND_MOVE, DisplayId: b.displayID, X: 1, Y: 2}}}
	command := sessions.DesktopCommand{GeometryRevision: "r", Payload: testAction(t, action)}
	require.ErrorIs(t, b.Validate(context.Background(), command), ErrUnavailable)
	err := b.Apply(context.Background(), command)
	require.ErrorIs(t, err, ErrUnavailable)
}

func TestRemoteDesktopUsesSharedSessionForScreenCastStream(t *testing.T) {
	var requests []string
	var selectedPath dbus.ObjectPath
	b := &Backend{displayID: "wayland-kde-portal", requestFn: func(_ context.Context, iface, method string, args ...any) (map[string]dbus.Variant, error) {
		requests = append(requests, iface+"."+method)
		switch iface + "." + method {
		case "org.freedesktop.portal.RemoteDesktop.CreateSession":
			options, ok := args[0].(map[string]dbus.Variant)
			require.True(t, ok)
			token, ok := options["session_handle_token"].Value().(string)
			require.True(t, ok)
			require.Regexp(t, `^vrooli[0-9a-f]{32}$`, token)
			return map[string]dbus.Variant{"session_handle": dbus.MakeVariant("/session/1")}, nil
		case "org.freedesktop.portal.RemoteDesktop.SelectDevices":
			selectedPath, _ = args[0].(dbus.ObjectPath)
			options, ok := args[1].(map[string]dbus.Variant)
			require.True(t, ok)
			require.Equal(t, uint32(3), options["types"].Value())
			return map[string]dbus.Variant{}, nil
		case "org.freedesktop.portal.ScreenCast.SelectSources":
			require.Equal(t, selectedPath, args[0])
			options, ok := args[1].(map[string]dbus.Variant)
			require.True(t, ok)
			require.Equal(t, uint32(1), options["types"].Value())
			return map[string]dbus.Variant{}, nil
		case "org.freedesktop.portal.RemoteDesktop.Start":
			require.Equal(t, selectedPath, args[0])
			streams := []struct {
				ID         uint32
				Properties map[string]dbus.Variant
			}{{ID: 22}}
			return map[string]dbus.Variant{"streams": dbus.MakeVariant(streams)}, nil
		default:
			return nil, ErrUnavailable
		}
	}, callFn: func(context.Context, string, ...any) error { return nil }}
	require.NoError(t, b.ensureRemoteDesktop(context.Background()))
	require.Equal(t, dbus.ObjectPath("/session/1"), b.session)
	require.Equal(t, uint32(22), b.stream)
	require.Equal(t, []string{"org.freedesktop.portal.RemoteDesktop.CreateSession", "org.freedesktop.portal.RemoteDesktop.SelectDevices", "org.freedesktop.portal.ScreenCast.SelectSources", "org.freedesktop.portal.RemoteDesktop.Start"}, requests)
}

func TestKeySymAcceptsUnicodeAndNamedKeys(t *testing.T) {
	v, ok := keySym("é")
	require.True(t, ok)
	require.Equal(t, int32('é'), v)
	v, ok = keySym("ENTER")
	require.True(t, ok)
	require.Equal(t, int32(0xff0d), v)
	v, ok = keySym("ShiftLeft")
	require.True(t, ok)
	require.Equal(t, int32(0xffe1), v)
	v, ok = keySym("F12")
	require.True(t, ok)
	require.Equal(t, int32(0xffc9), v)
	_, ok = keySym("F13")
	require.False(t, ok)
}

func TestPortalBackendStopsGrantedSessionOnClose(t *testing.T) {
	var calls []string
	b := &Backend{session: dbus.ObjectPath("/session"), started: true, callFn: func(_ context.Context, method string, _ ...any) error {
		calls = append(calls, method)
		return nil
	}}
	require.NoError(t, b.Close())
	require.Equal(t, []string{"Session.Close"}, calls)
}

func TestPortalBackendReleasesHeldKeysAndButtons(t *testing.T) {
	var calls []string
	b := &Backend{
		displayID: "wayland-gnome-portal",
		width:     100,
		height:    100,
		stream:    7,
		started:   true,
		session:   dbus.ObjectPath("/session/1"),
		callFn: func(_ context.Context, method string, _ ...any) error {
			calls = append(calls, method)
			return nil
		},
	}
	require.NoError(t, b.key(context.Background(), &desktopv1.KeyAction{Kind: desktopv1.KeyAction_KIND_DOWN, Key: "A"}))
	require.NoError(t, b.pointer(context.Background(), &desktopv1.PointerAction{Kind: desktopv1.PointerAction_KIND_DOWN, DisplayId: b.displayID, X: 10, Y: 20, Button: desktopv1.PointerAction_BUTTON_PRIMARY}))
	require.True(t, b.heldKeys[int32('A')])
	require.True(t, b.heldButtons[1])
	require.NoError(t, b.ReleaseHeld(context.Background()))
	require.Empty(t, b.heldKeys)
	require.Empty(t, b.heldButtons)
	require.Equal(t, []string{"NotifyKeyboardKeysym", "NotifyPointerMotionAbsolute", "NotifyPointerButton", "NotifyKeyboardKeysym", "NotifyPointerButton"}, calls)
}

func TestPortalBackendRetainsHeldStateWhenReleaseFails(t *testing.T) {
	failRelease := true
	b := &Backend{
		started: true,
		session: dbus.ObjectPath("/session/1"),
		callFn: func(_ context.Context, method string, args ...any) error {
			if method == "NotifyKeyboardKeysym" && len(args) > 0 && args[len(args)-1] == uint32(0) && failRelease {
				return errors.New("release unavailable")
			}
			return nil
		},
	}
	b.heldKeys = map[int32]bool{65: true}
	require.Error(t, b.ReleaseHeld(context.Background()))
	require.True(t, b.heldKeys[65], "failed release must remain pending for a later cleanup attempt")
	failRelease = false
	require.NoError(t, b.ReleaseHeld(context.Background()))
	require.Empty(t, b.heldKeys)
}

func TestPortalBackendCloseReleasesHeldInputBeforeSessionClose(t *testing.T) {
	var calls []string
	b := &Backend{
		started:     true,
		session:     dbus.ObjectPath("/session/1"),
		heldKeys:    map[int32]bool{65: true},
		heldButtons: map[uint32]bool{1: true},
		callFn: func(_ context.Context, method string, _ ...any) error {
			calls = append(calls, method)
			return nil
		},
	}
	require.NoError(t, b.Close())
	require.Equal(t, []string{"NotifyKeyboardKeysym", "NotifyPointerButton", "Session.Close"}, calls)
}

func TestPortalOperationsRecheckUserSessionBeforeUse(t *testing.T) {
	sessionChanged := errors.New("portal user session changed")
	requests := 0
	b := &Backend{
		displayID: "wayland-gnome-portal",
		width:     8,
		height:    6,
		revision:  "r1",
		requestFn: func(context.Context, string, string, ...any) (map[string]dbus.Variant, error) {
			requests++
			return nil, nil
		},
		sessionCheckFn: func(context.Context) error { return sessionChanged },
	}
	_, err := b.Observe(context.Background())
	require.ErrorIs(t, err, sessionChanged)
	action := &desktopv1.Action{Action: &desktopv1.Action_Key{Key: &desktopv1.KeyAction{Kind: desktopv1.KeyAction_KIND_PRESS, Key: "A"}}}
	command := sessions.DesktopCommand{GeometryRevision: "r1", Payload: testAction(t, action)}
	require.ErrorIs(t, b.Validate(context.Background(), command), sessionChanged)
	require.ErrorIs(t, b.Apply(context.Background(), command), sessionChanged)
	require.Zero(t, requests, "a changed user session must be rejected before portal use")
}

func TestSessionPathRejectsMalformedPortalObjectPaths(t *testing.T) {
	for _, value := range []dbus.Variant{
		dbus.MakeVariant("relative/session"),
		dbus.MakeVariant("/session//child"),
		dbus.MakeVariant("/session/child/é"),
	} {
		_, ok := sessionPath(value)
		require.False(t, ok)
	}
	path, ok := sessionPath(dbus.MakeVariant("/session/child"))
	require.True(t, ok)
	require.Equal(t, dbus.ObjectPath("/session/child"), path)
}
