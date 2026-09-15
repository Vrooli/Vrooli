package host

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image"
	"image/png"
	"strings"
	"testing"

	"device-control/internal/sessions"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/targetmodel"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"google.golang.org/protobuf/encoding/protojson"
)

func framePNG(t *testing.T) []byte {
	t.Helper()
	var out bytes.Buffer
	require.NoError(t, png.Encode(&out, image.NewRGBA(image.Rect(0, 0, 4, 5))))
	return out.Bytes()
}

func actionPayload(t *testing.T, action *desktopv1.Action) []byte {
	t.Helper()
	payload, err := protojson.Marshal(action)
	require.NoError(t, err)
	return payload
}

func TestDarwinBackendCapturesAndAppliesPointerInBoundUserSession(t *testing.T) {
	pngData := framePNG(t)
	var calls [][]string
	b := newBackendForPlatform("darwin", func(_ context.Context, name string, args ...string) ([]byte, error) {
		require.Equal(t, "screencapture", name)
		return pngData, nil
	}, func(_ context.Context, name string, args ...string) error {
		calls = append(calls, append([]string{name}, args...))
		return nil
	})
	snapshot, err := b.Observe(context.Background())
	require.NoError(t, err)
	require.Equal(t, 4, snapshot.Image.Bounds().Dx())
	require.Equal(t, 5, snapshot.Image.Bounds().Dy())
	payload := actionPayload(t, &desktopv1.Action{Action: &desktopv1.Action_Pointer{Pointer: &desktopv1.PointerAction{Kind: desktopv1.PointerAction_KIND_CLICK, DisplayId: snapshot.DisplayID, X: 1, Y: 2, Button: desktopv1.PointerAction_BUTTON_PRIMARY}}})
	command := sessions.DesktopCommand{GeometryRevision: snapshot.GeometryRevision, Payload: payload}
	require.NoError(t, b.Validate(context.Background(), command))
	require.NoError(t, b.Apply(context.Background(), command))
	require.Equal(t, "osascript", calls[0][0])
}

func TestDarwinBackendPrefersScreenCaptureKitWhenConfigured(t *testing.T) {
	pngData := framePNG(t)
	commandCalled := false
	b := newBackendForPlatform("darwin", func(context.Context, string, ...string) ([]byte, error) {
		commandCalled = true
		return nil, errors.New("compatibility capture must not be selected")
	}, func(context.Context, string, ...string) error { return nil })
	b.nativeCapture = func(context.Context) ([]byte, error) { return pngData, nil }

	snapshot, err := b.Observe(context.Background())
	require.NoError(t, err)
	require.Equal(t, 4, snapshot.Image.Bounds().Dx())
	require.False(t, commandCalled, "configured ScreenCaptureKit capture should run before the compatibility command")
}

func TestDarwinBackendFallsBackWhenScreenCaptureKitUnavailable(t *testing.T) {
	pngData := framePNG(t)
	commandCalled := false
	b := newBackendForPlatform("darwin", func(_ context.Context, name string, args ...string) ([]byte, error) {
		commandCalled = true
		require.Equal(t, "screencapture", name)
		return pngData, nil
	}, func(context.Context, string, ...string) error { return nil })
	b.nativeCapture = func(context.Context) ([]byte, error) { return nil, errors.New("ScreenCaptureKit unavailable") }

	_, err := b.Observe(context.Background())
	require.NoError(t, err)
	require.True(t, commandCalled, "older packaged helpers retain the compatibility capture path")
}

func TestWindowsBackendDecodesBoundedCaptureAndAppliesKey(t *testing.T) {
	pngData := base64.StdEncoding.EncodeToString(framePNG(t))
	var calls [][]string
	b := newBackendForPlatform("windows", func(_ context.Context, name string, args ...string) ([]byte, error) {
		require.Equal(t, "powershell.exe", name)
		return []byte(pngData), nil
	}, func(_ context.Context, name string, args ...string) error {
		calls = append(calls, append([]string{name}, args...))
		return nil
	})
	snapshot, err := b.Observe(context.Background())
	require.NoError(t, err)
	payload := actionPayload(t, &desktopv1.Action{Action: &desktopv1.Action_Key{Key: &desktopv1.KeyAction{Kind: desktopv1.KeyAction_KIND_PRESS, Key: "CTRL+L"}}})
	command := sessions.DesktopCommand{GeometryRevision: snapshot.GeometryRevision, Payload: payload}
	require.NoError(t, b.Validate(context.Background(), command))
	require.NoError(t, b.Apply(context.Background(), command))
	require.Equal(t, "powershell.exe", calls[0][0])
	require.Contains(t, calls[0][len(calls[0])-1], "CTRL+L")
}

func TestWindowsHeldKeyIsReleasedAndDuplicateDownIsIdempotent(t *testing.T) {
	pngData := base64.StdEncoding.EncodeToString(framePNG(t))
	var calls [][]string
	b := newBackendForPlatform("windows", func(context.Context, string, ...string) ([]byte, error) {
		return []byte(pngData), nil
	}, func(_ context.Context, name string, args ...string) error {
		calls = append(calls, append([]string{name}, args...))
		return nil
	})
	snapshot, err := b.Observe(context.Background())
	require.NoError(t, err)
	command := sessions.DesktopCommand{GeometryRevision: snapshot.GeometryRevision, Payload: actionPayload(t, &desktopv1.Action{Action: &desktopv1.Action_Key{Key: &desktopv1.KeyAction{Kind: desktopv1.KeyAction_KIND_DOWN, Key: "ShiftLeft"}}})}
	require.NoError(t, b.Validate(context.Background(), command))
	require.NoError(t, b.Apply(context.Background(), command))
	require.NoError(t, b.Validate(context.Background(), command))
	require.NoError(t, b.Apply(context.Background(), command))
	require.Equal(t, uint16(0xa0), b.heldKeys["SHIFTLEFT"])
	require.Len(t, calls, 1, "duplicate DOWN must not send a second native effect")
	require.NoError(t, b.ReleaseHeld(context.Background()))
	require.Empty(t, b.heldKeys)
	require.Len(t, calls, 2)
	require.Contains(t, calls[0][len(calls[0])-1], "keybd_event")
	require.Contains(t, calls[1][len(calls[1])-1], "0x0002")
}

func TestMacOSHeldKeyUsesQuartzRelease(t *testing.T) {
	var calls [][]string
	b := newBackendForPlatform("darwin", func(context.Context, string, ...string) ([]byte, error) { return framePNG(t), nil }, func(_ context.Context, name string, args ...string) error {
		calls = append(calls, append([]string{name}, args...))
		return nil
	})
	snapshot, err := b.Observe(context.Background())
	require.NoError(t, err)
	command := sessions.DesktopCommand{GeometryRevision: snapshot.GeometryRevision, Payload: actionPayload(t, &desktopv1.Action{Action: &desktopv1.Action_Key{Key: &desktopv1.KeyAction{Kind: desktopv1.KeyAction_KIND_DOWN, Key: "ShiftLeft"}}})}
	require.NoError(t, b.Apply(context.Background(), command))
	require.NoError(t, b.ReleaseHeld(context.Background()))
	require.Empty(t, b.heldKeys)
	require.Len(t, calls, 2)
	require.Contains(t, calls[0][len(calls[0])-1], "CGEventCreateKeyboardEvent")
	require.Contains(t, calls[0][len(calls[0])-1], ",true)")
	require.Contains(t, calls[1][len(calls[1])-1], ",false)")
}

func TestWindowsHeldPointerIsReleasedWithoutMovingCursor(t *testing.T) {
	pngData := base64.StdEncoding.EncodeToString(framePNG(t))
	var calls [][]string
	b := newBackendForPlatform("windows", func(context.Context, string, ...string) ([]byte, error) { return []byte(pngData), nil }, func(_ context.Context, name string, args ...string) error {
		calls = append(calls, append([]string{name}, args...))
		return nil
	})
	snapshot, err := b.Observe(context.Background())
	require.NoError(t, err)
	action := &desktopv1.Action{Action: &desktopv1.Action_Pointer{Pointer: &desktopv1.PointerAction{Kind: desktopv1.PointerAction_KIND_DOWN, DisplayId: snapshot.DisplayID, X: 1, Y: 2, Button: desktopv1.PointerAction_BUTTON_PRIMARY}}}
	command := sessions.DesktopCommand{GeometryRevision: snapshot.GeometryRevision, Payload: actionPayload(t, action)}
	require.NoError(t, b.Apply(context.Background(), command))
	require.True(t, b.heldButtons[desktopv1.PointerAction_BUTTON_PRIMARY])
	require.NoError(t, b.ReleaseHeld(context.Background()))
	require.Empty(t, b.heldButtons)
	require.Len(t, calls, 2)
	require.Contains(t, calls[0][len(calls[0])-1], "SetCursorPos(1,2)")
	require.NotContains(t, calls[1][len(calls[1])-1], "SetCursorPos")
	require.Contains(t, calls[1][len(calls[1])-1], "0x0004")
}

func TestMacOSHeldPointerUsesQuartzAndReleasesAtCurrentLocation(t *testing.T) {
	var calls [][]string
	b := newBackendForPlatform("darwin", func(context.Context, string, ...string) ([]byte, error) { return framePNG(t), nil }, func(_ context.Context, name string, args ...string) error {
		calls = append(calls, append([]string{name}, args...))
		return nil
	})
	snapshot, err := b.Observe(context.Background())
	require.NoError(t, err)
	action := &desktopv1.Action{Action: &desktopv1.Action_Pointer{Pointer: &desktopv1.PointerAction{Kind: desktopv1.PointerAction_KIND_DOWN, DisplayId: snapshot.DisplayID, X: 1, Y: 2, Button: desktopv1.PointerAction_BUTTON_PRIMARY}}}
	command := sessions.DesktopCommand{GeometryRevision: snapshot.GeometryRevision, Payload: actionPayload(t, action)}
	require.NoError(t, b.Apply(context.Background(), command))
	require.True(t, b.heldButtons[desktopv1.PointerAction_BUTTON_PRIMARY])
	require.NoError(t, b.ReleaseHeld(context.Background()))
	require.Empty(t, b.heldButtons)
	require.Len(t, calls, 2)
	require.Contains(t, calls[0][len(calls[0])-1], "CGEventCreateMouseEvent")
	require.Contains(t, calls[1][len(calls[1])-1], "CGEventGetLocation")
}

func TestWindowsBackendAppliesWheelWithBoundedNativeDeltas(t *testing.T) {
	pngData := base64.StdEncoding.EncodeToString(framePNG(t))
	var calls [][]string
	b := newBackendForPlatform("windows", func(_ context.Context, name string, args ...string) ([]byte, error) {
		return []byte(pngData), nil
	}, func(_ context.Context, name string, args ...string) error {
		calls = append(calls, append([]string{name}, args...))
		return nil
	})
	snapshot, err := b.Observe(context.Background())
	require.NoError(t, err)
	action := &desktopv1.Action{Action: &desktopv1.Action_Wheel{Wheel: &desktopv1.WheelAction{
		DisplayId: snapshot.DisplayID, X: 1, Y: 2, VerticalTicks: 2, HorizontalTicks: -1,
	}}}
	command := sessions.DesktopCommand{GeometryRevision: snapshot.GeometryRevision, Payload: actionPayload(t, action)}
	require.NoError(t, b.Validate(context.Background(), command))
	require.NoError(t, b.Apply(context.Background(), command))
	require.Len(t, calls, 1)
	require.Contains(t, calls[0][len(calls[0])-1], "0x0800")
	require.Contains(t, calls[0][len(calls[0])-1], "0x1000")
}

func TestDarwinBackendAppliesWheelDirections(t *testing.T) {
	pngData := framePNG(t)
	var calls [][]string
	b := newBackendForPlatform("darwin", func(_ context.Context, name string, args ...string) ([]byte, error) {
		return pngData, nil
	}, func(_ context.Context, name string, args ...string) error {
		calls = append(calls, append([]string{name}, args...))
		return nil
	})
	snapshot, err := b.Observe(context.Background())
	require.NoError(t, err)
	action := &desktopv1.Action{Action: &desktopv1.Action_Wheel{Wheel: &desktopv1.WheelAction{
		DisplayId: snapshot.DisplayID, X: 1, Y: 2, VerticalTicks: -1, HorizontalTicks: 1,
	}}}
	command := sessions.DesktopCommand{GeometryRevision: snapshot.GeometryRevision, Payload: actionPayload(t, action)}
	require.NoError(t, b.Apply(context.Background(), command))
	require.Len(t, calls, 1)
	require.Contains(t, calls[0][len(calls[0])-1], "scroll down")
	require.Contains(t, calls[0][len(calls[0])-1], "scroll right")
}

func TestBackendRejectsStaleGeometryAndInvalidCapture(t *testing.T) {
	b := newBackendForPlatform("darwin", func(context.Context, string, ...string) ([]byte, error) { return []byte("not png"), nil }, func(context.Context, string, ...string) error { return nil })
	_, err := b.Observe(context.Background())
	require.Error(t, err)
	require.Error(t, b.Validate(context.Background(), sessions.DesktopCommand{GeometryRevision: "stale", Payload: []byte(`{"key":{"key":"A"}}`)}))
}

func TestDarwinCaptureFailureSurfacesPermissionEvidence(t *testing.T) {
	b := newBackendForPlatform("darwin", func(context.Context, string, ...string) ([]byte, error) {
		return nil, errors.New("screen recording denied")
	}, func(context.Context, string, ...string) error { return nil })
	_, err := b.Observe(context.Background())
	require.Error(t, err)
	require.ErrorIs(t, err, ErrPermissionDenied)
}

func TestDarwinEmptyCaptureSurfacesPermissionEvidence(t *testing.T) {
	b := newBackendForPlatform("darwin", func(context.Context, string, ...string) ([]byte, error) {
		return nil, nil
	}, func(context.Context, string, ...string) error { return nil })
	_, err := b.Observe(context.Background())
	require.ErrorIs(t, err, ErrPermissionDenied)
}

func TestAcceptanceNAT04MacOSRevokeDoesNotReuseStaleCapture(t *testing.T) {
	pngData := framePNG(t)
	active := true
	b := newBackendForPlatform("darwin", func(context.Context, string, ...string) ([]byte, error) {
		if !active {
			return nil, errors.New("screen recording permission revoked")
		}
		return pngData, nil
	}, func(context.Context, string, ...string) error { return nil })
	first, err := b.Observe(context.Background())
	require.NoError(t, err)
	require.NotNil(t, first.Image)
	active = false
	second, err := b.Observe(context.Background())
	require.ErrorIs(t, err, ErrPermissionDenied)
	require.Nil(t, second.Image, "a revoked grant must not return the previous image")
}

func TestBackendRefusesSemanticTextWithoutAnAccessibilityAdapter(t *testing.T) {
	pngData := framePNG(t)
	b := newBackendForPlatform("windows", func(context.Context, string, ...string) ([]byte, error) {
		return []byte(base64.StdEncoding.EncodeToString(pngData)), nil
	}, func(context.Context, string, ...string) error { return nil })
	snapshot, err := b.Observe(context.Background())
	require.NoError(t, err)
	payload := actionPayload(t, &desktopv1.Action{Action: &desktopv1.Action_Text{Text: &desktopv1.TextAction{Text: "hello", ElementId: "element", ObservationRevision: "observation"}}})
	require.Error(t, b.Validate(context.Background(), sessions.DesktopCommand{GeometryRevision: snapshot.GeometryRevision, Payload: payload}))
}

func TestBackendBoundsPointerWheelAndCaptureOutput(t *testing.T) {
	var output boundedOutput
	_, err := output.Write(make([]byte, maxCaptureBytes+1))
	require.Error(t, err)
	b := newBackendForPlatform("darwin", func(context.Context, string, ...string) ([]byte, error) { return framePNG(t), nil }, func(context.Context, string, ...string) error { return nil })
	snapshot, err := b.Observe(context.Background())
	require.NoError(t, err)
	invalid := []*desktopv1.Action{
		{Action: &desktopv1.Action_Pointer{Pointer: &desktopv1.PointerAction{Kind: desktopv1.PointerAction_KIND_CLICK, DisplayId: snapshot.DisplayID, X: 1, Y: 1}}},
		{Action: &desktopv1.Action_Wheel{Wheel: &desktopv1.WheelAction{DisplayId: snapshot.DisplayID, X: 1, Y: 1, VerticalTicks: 21}}},
	}
	for _, action := range invalid {
		require.Error(t, b.Validate(context.Background(), sessions.DesktopCommand{GeometryRevision: snapshot.GeometryRevision, Payload: actionPayload(t, action)}))
	}
}

func TestSessionMonitorRejectsWrongWindowsSession(t *testing.T) {
	b := newBackendForPlatform("windows", func(_ context.Context, name string, args ...string) ([]byte, error) {
		require.Equal(t, "powershell.exe", name)
		return []byte("7\n"), nil
	}, func(context.Context, string, ...string) error { return nil })
	b.session = "8"
	require.Error(t, b.CheckSession(context.Background()))
	b.session = "7"
	require.NoError(t, b.CheckSession(context.Background()))
}

func TestSessionBoundObserveRechecksWindowsSessionAroundCapture(t *testing.T) {
	pngData := base64.StdEncoding.EncodeToString(framePNG(t))
	checks := 0
	b := newBackendForPlatform("windows", func(_ context.Context, name string, args ...string) ([]byte, error) {
		require.Equal(t, "powershell.exe", name)
		script := args[len(args)-1]
		if strings.Contains(script, "GetCurrentProcess().SessionId") {
			checks++
			return []byte("7\n"), nil
		}
		require.Contains(t, script, "SystemInformation")
		return []byte(pngData), nil
	}, func(context.Context, string, ...string) error { return nil })
	b.session = "7"
	_, err := b.Observe(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, checks, "a session-bound capture must be bracketed by identity checks")
}

func TestWindowsUIAutomationEnumeratesResolvesAndSetsText(t *testing.T) {
	pngData := base64.StdEncoding.EncodeToString(framePNG(t))
	var commands [][]string
	command := func(_ context.Context, name string, args ...string) ([]byte, error) {
		commands = append(commands, append([]string{name}, args...))
		script := args[len(args)-1]
		switch {
		case strings.Contains(script, "GetCurrentProcess().SessionId"):
			return []byte("7\n"), nil
		case strings.Contains(script, "SystemInformation"):
			return []byte(pngData), nil
		case strings.Contains(script, "parent_index"):
			return []byte(`[ {"name":"Editor","runtime_id":"1,2","editable":false,"value":"","parent_index":-1,"window_index":0}, {"name":"Title","runtime_id":"1,3","editable":true,"value":"before","parent_index":0,"window_index":0} ]`), nil
		case strings.Contains(script, "FromRuntimeId") && strings.Contains(script, "ValuePattern") && strings.Contains(script, "ConvertTo-Json"):
			return []byte(`"before"`), nil
		case strings.Contains(script, "SetValue"):
			return nil, nil
		case strings.Contains(script, "process_id"):
			return []byte(`{"name":"Editor","runtime_id":"1,2","process_id":42}`), nil
		default:
			t.Fatalf("unexpected UI Automation command: %s", script)
			return nil, errors.New("unexpected UI Automation command")
		}
	}
	var runs [][]string
	b := newBackendForPlatform("windows", command, func(_ context.Context, name string, args ...string) error {
		runs = append(runs, append([]string{name}, args...))
		return nil
	})
	catalog, err := b.Applications(context.Background(), "session-1")
	require.NoError(t, err)
	require.Len(t, catalog.Applications, 1)
	snapshot, err := b.ObserveApplication(context.Background(), catalog.Applications[0].ID, catalog.Revision, "session-1")
	require.NoError(t, err)
	require.NotNil(t, snapshot.Semantic)
	require.Len(t, snapshot.Semantic.Elements, 2)
	windowID := snapshot.Semantic.Elements[0].ID
	resolution, err := b.Resolve(context.Background(), sessions.DesktopSelector{Revision: snapshot.Semantic.Revision, WindowID: windowID, Name: "Title", EditableOnly: true}, "session-1")
	require.NoError(t, err)
	require.Equal(t, "unique", resolution.Disposition)
	require.Len(t, resolution.ElementIDs, 1)
	textAction := &desktopv1.Action{Action: &desktopv1.Action_Text{Text: &desktopv1.TextAction{Text: "after", ElementId: resolution.ElementIDs[0], ObservationRevision: snapshot.Semantic.Revision}}}
	lease := sessions.DesktopLease{Ref: targetmodel.SessionRef{SessionID: "session-1"}}
	commandPayload := sessions.DesktopCommand{Lease: lease, GeometryRevision: snapshot.GeometryRevision, Payload: actionPayload(t, textAction)}
	require.NoError(t, b.Validate(context.Background(), commandPayload))
	require.NoError(t, b.Apply(context.Background(), commandPayload))
	require.Len(t, runs, 1)
	require.Contains(t, runs[0][len(runs[0])-1], "SetValue")
	require.NotEmpty(t, commands)
}

func TestAcceptanceNAT03WindowsSemanticInvoke(t *testing.T) {
	pngData := base64.StdEncoding.EncodeToString(framePNG(t))
	var runs [][]string
	command := func(_ context.Context, name string, args ...string) ([]byte, error) {
		script := args[len(args)-1]
		switch {
		case strings.Contains(script, "GetCurrentProcess().SessionId"):
			return []byte("7\n"), nil
		case strings.Contains(script, "SystemInformation"):
			return []byte(pngData), nil
		case strings.Contains(script, "parent_index"):
			return []byte(`[{"name":"Editor","runtime_id":"1,2","editable":false,"value":"","parent_index":-1,"window_index":0},{"name":"Save","runtime_id":"1,3","editable":false,"value":"","parent_index":0,"window_index":0}]`), nil
		case strings.Contains(script, "process_id"):
			return []byte(`{"name":"Editor","runtime_id":"1,2","process_id":42}`), nil
		default:
			t.Fatalf("unexpected changing-text UI Automation command: %s", script)
			return nil, errors.New("unexpected UI Automation command")
		}
	}
	b := newBackendForPlatform("windows", command, func(_ context.Context, name string, args ...string) error {
		runs = append(runs, append([]string{name}, args...))
		return nil
	})
	catalog, err := b.Applications(context.Background(), "session-1")
	require.NoError(t, err)
	snapshot, err := b.ObserveApplication(context.Background(), catalog.Applications[0].ID, catalog.Revision, "session-1")
	require.NoError(t, err)
	action := &desktopv1.Action{Action: &desktopv1.Action_Invoke{Invoke: &desktopv1.InvokeAction{ElementId: snapshot.Semantic.Elements[1].ID, ObservationRevision: snapshot.Semantic.Revision}}}
	commandPayload := sessions.DesktopCommand{Lease: sessions.DesktopLease{Ref: targetmodel.SessionRef{SessionID: "session-1"}}, GeometryRevision: snapshot.GeometryRevision, Payload: actionPayload(t, action)}
	require.NoError(t, b.Validate(context.Background(), commandPayload))
	require.NoError(t, b.Apply(context.Background(), commandPayload))
	require.Len(t, runs, 1)
	require.Contains(t, runs[0][len(runs[0])-1], "InvokePattern")
}

func TestSemanticInvokeRefusesReplacementProcess(t *testing.T) {
	pngData := base64.StdEncoding.EncodeToString(framePNG(t))
	processChecks := 0
	runs := 0
	command := func(_ context.Context, _ string, args ...string) ([]byte, error) {
		script := args[len(args)-1]
		switch {
		case strings.Contains(script, "SystemInformation"):
			return []byte(pngData), nil
		case strings.Contains(script, "parent_index"):
			return []byte(`[{"name":"Editor","runtime_id":"1,2","editable":false,"value":"","parent_index":-1,"window_index":0},{"name":"Export","runtime_id":"1,3","editable":false,"value":"","parent_index":0,"window_index":0}]`), nil
		case strings.Contains(script, "GetCurrentProcess().SessionId"):
			return []byte("7\n"), nil
		case strings.Contains(script, "process_id"):
			processChecks++
			if processChecks == 1 {
				return []byte(`{"name":"Editor","runtime_id":"1,2","process_id":42}`), nil
			}
			return []byte(`[]`), nil
		default:
			t.Fatalf("unexpected changing-text UI Automation command: %s", script)
			return nil, errors.New("unexpected UI Automation command")
		}
	}
	b := newBackendForPlatform("windows", command, func(context.Context, string, ...string) error { runs++; return nil })
	catalog, err := b.Applications(context.Background(), "session-1")
	require.NoError(t, err)
	snapshot, err := b.ObserveApplication(context.Background(), catalog.Applications[0].ID, catalog.Revision, "session-1")
	require.NoError(t, err)
	action := &desktopv1.Action{Action: &desktopv1.Action_Invoke{Invoke: &desktopv1.InvokeAction{ElementId: snapshot.Semantic.Elements[1].ID, ObservationRevision: snapshot.Semantic.Revision}}}
	payload := actionPayload(t, action)
	lease := sessions.DesktopLease{Ref: targetmodel.SessionRef{SessionID: "session-1"}}
	err = b.Apply(context.Background(), sessions.DesktopCommand{Lease: lease, GeometryRevision: snapshot.GeometryRevision, Payload: payload})
	require.ErrorIs(t, err, ErrUnavailable)
	require.Zero(t, runs, "a replacement process must be refused before invoking the cached element")
}

func TestSemanticTextRefusesValueChangeAfterValidation(t *testing.T) {
	pngData := base64.StdEncoding.EncodeToString(framePNG(t))
	reads := 0
	runs := 0
	command := func(_ context.Context, _ string, args ...string) ([]byte, error) {
		script := args[len(args)-1]
		switch {
		case strings.Contains(script, "GetCurrentProcess().SessionId"):
			return []byte("7\n"), nil
		case strings.Contains(script, "SystemInformation"):
			return []byte(pngData), nil
		case strings.Contains(script, "parent_index"):
			return []byte(`[{"name":"Editor","runtime_id":"1,2","editable":false,"value":"","parent_index":-1,"window_index":0},{"name":"Title","runtime_id":"1,3","editable":true,"value":"before","parent_index":0,"window_index":0}]`), nil
		case strings.Contains(script, "ValuePattern") && strings.Contains(script, "ConvertTo-Json"):
			reads++
			if reads == 1 {
				return []byte(`"before"`), nil
			}
			return []byte(`"changed"`), nil
		case strings.Contains(script, "process_id"):
			return []byte(`{"name":"Editor","runtime_id":"1,2","process_id":42}`), nil
		default:
			return nil, errors.New("unexpected UI Automation command")
		}
	}
	b := newBackendForPlatform("windows", command, func(context.Context, string, ...string) error { runs++; return nil })
	snapshot, err := b.ObserveProcess(context.Background(), 42, "session-1")
	require.NoError(t, err)
	action := &desktopv1.Action{Action: &desktopv1.Action_Text{Text: &desktopv1.TextAction{Text: "after", ElementId: snapshot.Semantic.Elements[1].ID, ObservationRevision: snapshot.Semantic.Revision}}}
	err = b.Apply(context.Background(), sessions.DesktopCommand{Lease: sessions.DesktopLease{Ref: targetmodel.SessionRef{SessionID: "session-1"}}, GeometryRevision: snapshot.GeometryRevision, Payload: actionPayload(t, action)})
	require.ErrorIs(t, err, ErrUnavailable)
	require.Zero(t, runs, "a value change after validation must refuse the setter")
}

func TestWindowsActivationProbeAndWindowVerification(t *testing.T) {
	var runs [][]string
	b := newBackendForPlatform("windows", func(_ context.Context, name string, args ...string) ([]byte, error) {
		require.Equal(t, "powershell.exe", name)
		script := args[len(args)-1]
		switch {
		case strings.Contains(script, "active_window"):
			return []byte(`{"active_window":42,"process_id":7,"pointer_window":42,"pointer_x":10,"pointer_y":20,"x":1,"y":2,"width":800,"height":600}`), nil
		default:
			return nil, errors.New("unexpected activation command")
		}
	}, func(_ context.Context, name string, args ...string) error {
		runs = append(runs, append([]string{name}, args...))
		return nil
	})
	metadata, err := b.CaptureActivation(context.Background())
	require.NoError(t, err)
	require.Equal(t, uint64(42), metadata.ActiveWindow)
	require.Equal(t, uint32(7), metadata.ProcessID)
	require.Equal(t, uint32(800), metadata.SourceBounds.Width)
	require.NoError(t, b.VerifyWindowProcess(context.Background(), 99, 7))
	require.Len(t, runs, 1)
	require.Contains(t, runs[0][len(runs[0])-1], "window process mismatch")
}

func TestMacOSWindowVerificationRequiresFrontmostWindow(t *testing.T) {
	var script string
	b := newBackendForPlatform("darwin", func(context.Context, string, ...string) ([]byte, error) {
		return nil, errors.New("unexpected macOS command")
	}, func(_ context.Context, name string, args ...string) error {
		require.Equal(t, "osascript", name)
		script = args[len(args)-1]
		return nil
	})
	require.NoError(t, b.VerifyWindowProcess(context.Background(), 42, 42))
	require.Contains(t, script, "frontmost()!==true")
	require.Contains(t, script, "ws && ws.length>0")
}

func TestMacOSWindowVerificationRefusesNonFrontmostProcess(t *testing.T) {
	b := newBackendForPlatform("darwin", func(context.Context, string, ...string) ([]byte, error) {
		return nil, errors.New("unexpected macOS command")
	}, func(_ context.Context, name string, args ...string) error {
		require.Equal(t, "osascript", name)
		require.Contains(t, args[len(args)-1], "frontmost()!==true")
		return errors.New("frontmost window process unavailable")
	})
	require.ErrorIs(t, b.VerifyWindowProcess(context.Background(), 42, 42), ErrUnavailable)
}

func TestMacOSActivationProbeAndImageRemainBounded(t *testing.T) {
	pngData := framePNG(t)
	commands := 0
	b := newBackendForPlatform("darwin", func(_ context.Context, name string, args ...string) ([]byte, error) {
		commands++
		script := args[len(args)-1]
		if strings.Contains(script, "frontmost process unavailable") {
			return []byte(`{"active_window":42,"process_id":42,"pointer_window":42,"pointer_x":0,"pointer_y":0,"x":0,"y":0,"width":4,"height":5,"root_x":0,"root_y":0}`), nil
		}
		if name == "screencapture" {
			return pngData, nil
		}
		return nil, errors.New("unexpected macOS activation command")
	}, func(_ context.Context, _ string, _ ...string) error { return nil })
	imageEvidence, err := b.CaptureActivationImage(context.Background())
	require.NoError(t, err)
	require.Equal(t, uint64(42), imageEvidence.Context.ActiveWindow)
	require.Equal(t, 4, imageEvidence.Image.Bounds().Dx())
	require.Equal(t, 5, imageEvidence.Image.Bounds().Dy())
	require.Equal(t, 3, commands, "two identity reads and one bounded image capture")
}

func TestActivationImageCropsVirtualScreenOrigin(t *testing.T) {
	pngData := framePNG(t)
	b := newBackendForPlatform("windows", func(_ context.Context, name string, args ...string) ([]byte, error) {
		require.Equal(t, "powershell.exe", name)
		script := args[len(args)-1]
		if strings.Contains(script, "active_window") {
			return []byte(`{"active_window":42,"process_id":7,"pointer_window":42,"pointer_x":0,"pointer_y":0,"x":-99,"y":-49,"width":2,"height":3,"root_x":-100,"root_y":-50}`), nil
		}
		if strings.Contains(script, "SystemInformation") {
			return []byte(base64.StdEncoding.EncodeToString(pngData)), nil
		}
		return nil, errors.New("unexpected activation command")
	}, func(context.Context, string, ...string) error { return nil })
	evidence, err := b.CaptureActivationImage(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, evidence.Image.Bounds().Dx())
	require.Equal(t, 3, evidence.Image.Bounds().Dy())
}

func TestActivationImageRejectsSessionChangeAfterCapture(t *testing.T) {
	pngData := base64.StdEncoding.EncodeToString(framePNG(t))
	sessionChecks := 0
	b := newBackendForPlatform("windows", func(_ context.Context, name string, args ...string) ([]byte, error) {
		require.Equal(t, "powershell.exe", name)
		script := args[len(args)-1]
		switch {
		case strings.Contains(script, "GetCurrentProcess().SessionId"):
			sessionChecks++
			if sessionChecks >= 3 {
				return []byte("8\n"), nil
			}
			return []byte("7\n"), nil
		case strings.Contains(script, "active_window"):
			return []byte(`{"active_window":42,"process_id":7,"pointer_window":42,"pointer_x":0,"pointer_y":0,"x":0,"y":0,"width":4,"height":5,"root_x":0,"root_y":0}`), nil
		case strings.Contains(script, "SystemInformation"):
			return []byte(pngData), nil
		default:
			return nil, errors.New("unexpected activation command")
		}
	}, func(context.Context, string, ...string) error { return nil })
	b.session = "7"
	_, err := b.CaptureActivationImage(context.Background())
	require.ErrorIs(t, err, ErrUnavailable)
	require.Equal(t, 3, sessionChecks, "session is checked before probing, after probing, and after capture")
}

func TestSessionBoundSemanticInvokeRechecksBeforeEffect(t *testing.T) {
	pngData := base64.StdEncoding.EncodeToString(framePNG(t))
	checks := 0
	var runs [][]string
	command := func(_ context.Context, name string, args ...string) ([]byte, error) {
		script := args[len(args)-1]
		switch {
		case strings.Contains(script, "GetCurrentProcess().SessionId"):
			checks++
			return []byte("7\n"), nil
		case strings.Contains(script, "SystemInformation"):
			return []byte(pngData), nil
		case strings.Contains(script, "parent_index"):
			return []byte(`[{"name":"Editor","runtime_id":"1,2","editable":false,"value":"","parent_index":-1,"window_index":0},{"name":"Export","runtime_id":"1,3","editable":false,"value":"","parent_index":0,"window_index":0}]`), nil
		case strings.Contains(script, "process_id"):
			return []byte(`{"name":"Editor","runtime_id":"1,2","process_id":42}`), nil
		default:
			return nil, errors.New("unexpected UI Automation command")
		}
	}
	b := newBackendForPlatform("windows", command, func(_ context.Context, name string, args ...string) error {
		runs = append(runs, append([]string{name}, args...))
		return nil
	})
	b.session = "7"
	catalog, err := b.Applications(context.Background(), "session-1")
	require.NoError(t, err)
	snapshot, err := b.ObserveApplication(context.Background(), catalog.Applications[0].ID, catalog.Revision, "session-1")
	require.NoError(t, err)
	action := &desktopv1.Action{Action: &desktopv1.Action_Invoke{Invoke: &desktopv1.InvokeAction{ElementId: snapshot.Semantic.Elements[1].ID, ObservationRevision: snapshot.Semantic.Revision}}}
	commandPayload := sessions.DesktopCommand{Lease: sessions.DesktopLease{Ref: targetmodel.SessionRef{SessionID: "session-1"}}, GeometryRevision: snapshot.GeometryRevision, Payload: actionPayload(t, action)}
	before := checks
	require.NoError(t, b.Apply(context.Background(), commandPayload))
	require.GreaterOrEqual(t, checks-before, 2, "semantic invocation must recheck the bound session immediately before the effect")
	require.Len(t, runs, 1)
}

func TestMacOSAccessibilityEnumeratesResolvesAndSetsText(t *testing.T) {
	pngData := framePNG(t)
	var runs [][]string
	command := func(_ context.Context, name string, args ...string) ([]byte, error) {
		switch name {
		case "stat":
			return []byte("1000\n"), nil
		case "screencapture":
			return pngData, nil
		case "osascript":
			script := args[len(args)-1]
			switch {
			case strings.Contains(script, "JSON.stringify(value)"):
				return []byte(`"before"`), nil
			case strings.Contains(script, "entireContents") && strings.Contains(script, "JSON.stringify(out)"):
				return []byte(`[{"name":"Editor","runtime_id":"pid:42:0:0","editable":false,"value":"","parent_index":-1,"window_index":0},{"name":"Title","runtime_id":"pid:42:0:1","editable":true,"value":"before","parent_index":0,"window_index":0}]`), nil
			case strings.Contains(script, "applicationProcesses"):
				return []byte(`[{"name":"Editor","runtime_id":"pid:42","process_id":42}]`), nil
			default:
				return nil, errors.New("unexpected macOS accessibility script")
			}
		default:
			return nil, errors.New("unexpected macOS command")
		}
	}
	b := newBackendForPlatform("darwin", command, func(_ context.Context, name string, args ...string) error {
		runs = append(runs, append([]string{name}, args...))
		return nil
	})
	catalog, err := b.Applications(context.Background(), "lease-mac")
	require.NoError(t, err)
	require.Len(t, catalog.Applications, 1)
	snapshot, err := b.ObserveApplication(context.Background(), catalog.Applications[0].ID, catalog.Revision, "lease-mac")
	require.NoError(t, err)
	require.NotNil(t, snapshot.Semantic)
	require.Len(t, snapshot.Semantic.Elements, 2)
	resolution, err := b.Resolve(context.Background(), sessions.DesktopSelector{Revision: snapshot.Semantic.Revision, WindowID: snapshot.Semantic.Elements[0].ID, Name: "Title", EditableOnly: true}, "lease-mac")
	require.NoError(t, err)
	require.Equal(t, "unique", resolution.Disposition)
	textAction := &desktopv1.Action{Action: &desktopv1.Action_Text{Text: &desktopv1.TextAction{Text: "after", ElementId: resolution.ElementIDs[0], ObservationRevision: snapshot.Semantic.Revision}}}
	commandPayload := sessions.DesktopCommand{Lease: sessions.DesktopLease{Ref: targetmodel.SessionRef{SessionID: "lease-mac"}}, GeometryRevision: snapshot.GeometryRevision, Payload: actionPayload(t, textAction)}
	require.NoError(t, b.Validate(context.Background(), commandPayload))
	require.NoError(t, b.Apply(context.Background(), commandPayload))
	require.Len(t, runs, 1)
	require.Equal(t, "osascript", runs[0][0])
	require.Contains(t, runs[0][len(runs[0])-1], "target.value")
}

func TestMacOSSemanticInvokeRefusesReplacedElement(t *testing.T) {
	pngData := framePNG(t)
	semanticCalls := 0
	runs := 0
	command := func(_ context.Context, name string, args ...string) ([]byte, error) {
		switch name {
		case "stat":
			return []byte("1000\n"), nil
		case "screencapture":
			return pngData, nil
		case "osascript":
			script := args[len(args)-1]
			switch {
			case strings.Contains(script, "entireContents") && strings.Contains(script, "JSON.stringify(out)"):
				semanticCalls++
				if semanticCalls == 1 {
					return []byte(`[{"name":"Editor","runtime_id":"pid:42:0:0","editable":false,"value":"","parent_index":-1,"window_index":0},{"name":"Export","runtime_id":"pid:42:0:1","editable":false,"value":"","parent_index":0,"window_index":0}]`), nil
				}
				return []byte(`[{"name":"Editor","runtime_id":"pid:42:0:0","editable":false,"value":"","parent_index":-1,"window_index":0},{"name":"Different action","runtime_id":"pid:42:0:1","editable":false,"value":"","parent_index":0,"window_index":0}]`), nil
			case strings.Contains(script, "applicationProcesses"):
				return []byte(`[{"name":"Editor","runtime_id":"pid:42","process_id":42}]`), nil
			default:
				return nil, errors.New("unexpected macOS semantic command")
			}
		default:
			return nil, errors.New("unexpected macOS command")
		}
	}
	b := newBackendForPlatform("darwin", command, func(_ context.Context, _ string, _ ...string) error {
		runs++
		return nil
	})
	catalog, err := b.Applications(context.Background(), "lease-mac")
	require.NoError(t, err)
	snapshot, err := b.ObserveApplication(context.Background(), catalog.Applications[0].ID, catalog.Revision, "lease-mac")
	require.NoError(t, err)
	action := &desktopv1.Action{Action: &desktopv1.Action_Invoke{Invoke: &desktopv1.InvokeAction{ElementId: snapshot.Semantic.Elements[1].ID, ObservationRevision: snapshot.Semantic.Revision}}}
	err = b.Validate(context.Background(), sessions.DesktopCommand{Lease: sessions.DesktopLease{Ref: targetmodel.SessionRef{SessionID: "lease-mac"}}, GeometryRevision: snapshot.GeometryRevision, Payload: actionPayload(t, action)})
	require.ErrorIs(t, err, ErrUnavailable)
	require.Zero(t, runs, "a replaced macOS element must be refused before invoking it")
}

func TestMacOSSemanticCommandSeparatesWindowOrdinalFromRootIndex(t *testing.T) {
	command := macOSSemanticCommand("pid:42")
	script := command[len(command)-1]
	require.Contains(t, script, "function add(e,parent,windowNumber,id,rootIndex)")
	require.Contains(t, script, "runtime_id:\"pid:\"+pid+\":\"+windowNumber+\":\"+id")
	require.Contains(t, script, "window_index:rootIndex")
	require.Contains(t, script, "const rootIndex=out.length; add(windows[wi],-1,wi,0,rootIndex)")
	require.Contains(t, script, "add(children[ei],rootIndex,wi,ei+1,rootIndex)")
}

func TestWindowsRuntimeIDRejectsNegativeComponent(t *testing.T) {
	require.False(t, validRuntimeID("1,-2"))
	require.True(t, validRuntimeID("1,2"))
}
