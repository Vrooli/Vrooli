package x11

import (
	"context"
	"encoding/binary"
	"image/color"
	"os"
	"strings"
	"testing"

	"device-control/internal/sessions"
	"github.com/jezek/xgb/composite"

	"github.com/jezek/xgb/xproto"
	"github.com/stretchr/testify/require"
)

func TestActivationContextCapturesPriorWindowWithoutChangingFocus(t *testing.T) {
	display := isolatedDisplay(t)
	allowed := true
	b, err := newBackend(context.Background(), display, strings.Repeat("00", 16), func(context.Context, Peer) error {
		if !allowed {
			return ErrUnavailable
		}
		return nil
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = b.Close() })
	window, err := xproto.NewWindowId(b.conn)
	require.NoError(t, err)
	require.NoError(t, xproto.CreateWindowChecked(b.conn, 0, window, b.root, 20, 30, 200, 100, 0, xproto.WindowClassInputOutput, 0, 0, nil).Check())
	require.NoError(t, xproto.MapWindowChecked(b.conn, window).Check())
	require.NoError(t, b.VerifyWindowProcess(context.Background(), uint64(window), uint32(os.Getpid())))
	require.Error(t, b.VerifyWindowProcess(context.Background(), uint64(window), uint32(os.Getpid()+1)))
	require.Error(t, b.VerifyWindowProcess(context.Background(), uint64(b.root), uint32(os.Getpid())))

	require.NoError(t, xproto.SetInputFocusChecked(b.conn, xproto.InputFocusPointerRoot, window, xproto.TimeCurrentTime).Check())
	property := func(w xproto.Window, name string, kind xproto.Atom, value uint32) {
		a, e := xproto.InternAtom(b.conn, false, uint16(len(name)), name).Reply()
		require.NoError(t, e)
		data := make([]byte, 4)
		binary.LittleEndian.PutUint32(data, value)
		require.NoError(t, xproto.ChangePropertyChecked(b.conn, xproto.PropModeReplace, w, a.Atom, kind, 32, 1, data).Check())
	}
	// Missing window-manager evidence must not fall back to the root or a guess.
	_, err = b.CaptureActivation(context.Background())
	require.Error(t, err)
	property(b.root, "_NET_ACTIVE_WINDOW", xproto.AtomWindow, uint32(window))
	property(window, "_NET_WM_PID", xproto.AtomCardinal, 1234)
	// A client-controlled PID property cannot replace authenticated XRes ownership.
	require.Error(t, b.VerifyWindowProcess(context.Background(), uint64(window), 1234))
	require.NoError(t, b.VerifyWindowProcess(context.Background(), uint64(window), uint32(os.Getpid())))

	require.NoError(t, xproto.WarpPointerChecked(b.conn, 0, b.root, 0, 0, 0, 0, 50, 60).Check())
	result, err := b.CaptureActivation(context.Background())
	require.NoError(t, err)
	require.EqualValues(t, window, result.ActiveWindow)
	require.EqualValues(t, window, result.PointerWindow)
	require.EqualValues(t, 1234, result.ProcessID)
	require.EqualValues(t, 50, result.PointerX)
	require.EqualValues(t, 60, result.PointerY)
	require.NotEmpty(t, result.GeometryRevision)
	require.False(t, result.CapturedAt.IsZero())
	focus, err := xproto.GetInputFocus(b.conn).Reply()
	require.NoError(t, err)
	require.Equal(t, window, focus.Focus)
	require.EqualValues(t, 20, result.SourceBounds.X)
	require.EqualValues(t, 30, result.SourceBounds.Y)
	require.EqualValues(t, 200, result.SourceBounds.Width)
	require.EqualValues(t, 100, result.SourceBounds.Height)
	// Reparented clients report coordinates relative to a frame. Context bounds
	// must translate to the root instead of reusing GetGeometry's parent offsets.
	frame, err := xproto.NewWindowId(b.conn)
	require.NoError(t, err)
	require.NoError(t, xproto.CreateWindowChecked(b.conn, 0, frame, b.root, 100, 120, 400, 300, 0, xproto.WindowClassInputOutput, 0, 0, nil).Check())
	require.NoError(t, xproto.MapWindowChecked(b.conn, frame).Check())
	require.NoError(t, xproto.ReparentWindowChecked(b.conn, window, frame, 20, 30).Check())
	require.NoError(t, xproto.MapWindowChecked(b.conn, window).Check())
	moved, err := b.CaptureActivation(context.Background())
	require.NoError(t, err)
	require.EqualValues(t, 120, moved.SourceBounds.X)
	require.EqualValues(t, 150, moved.SourceBounds.Y)
	require.NoError(t, xproto.ConfigureWindowChecked(b.conn, frame, xproto.ConfigWindowX|xproto.ConfigWindowY, []uint32{0xffffffd8, 0xffffffe2}).Check())
	moved, err = b.CaptureActivation(context.Background())
	require.NoError(t, err)
	require.EqualValues(t, -20, moved.SourceBounds.X)
	require.Zero(t, moved.SourceBounds.Y)
	allowed = false
	require.Error(t, b.VerifyWindowProcess(context.Background(), uint64(window), uint32(os.Getpid())))
	denied, err := b.CaptureActivation(context.Background())
	require.Error(t, err)
	require.Equal(t, ActivationContext{}, denied)
	allowed = true
	// Permission may be revoked after the initial check, before evidence returns.
	checks := 0
	b.check = func(context.Context, Peer) error {
		checks++
		if checks > 1 {
			return ErrUnavailable
		}
		return nil
	}
	denied, err = b.CaptureActivation(context.Background())
	require.Error(t, err)
	require.Equal(t, ActivationContext{}, denied)
	b.check = func(context.Context, Peer) error { return nil }
	require.NoError(t, xproto.DestroyWindowChecked(b.conn, window).Check())
	_, err = b.CaptureActivation(context.Background())
	require.Error(t, err)
}

func TestActivationImageUsesSelectedWindowBackingStorage(t *testing.T) {
	display := isolatedDisplay(t)
	allowed := true
	b, err := newBackend(context.Background(), display, strings.Repeat("00", 16), func(context.Context, Peer) error {
		if !allowed {
			return ErrUnavailable
		}
		return nil
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = b.Close() })
	create := func(pixel uint32, border uint16) xproto.Window {
		window, e := xproto.NewWindowId(b.conn)
		require.NoError(t, e)
		require.NoError(t, xproto.CreateWindowChecked(b.conn, 0, window, b.root, 20, 30, 80, 60, border, xproto.WindowClassInputOutput, 0, xproto.CwBackPixel|xproto.CwBorderPixel, []uint32{pixel, 0x0000ff}).Check())
		require.NoError(t, xproto.MapWindowChecked(b.conn, window).Check())
		return window
	}
	window := create(0xff0000, 3)
	property := func(w xproto.Window, name string, kind xproto.Atom, value uint32) {
		atom, e := xproto.InternAtom(b.conn, false, uint16(len(name)), name).Reply()
		require.NoError(t, e)
		data := make([]byte, 4)
		binary.LittleEndian.PutUint32(data, value)
		require.NoError(t, xproto.ChangePropertyChecked(b.conn, xproto.PropModeReplace, w, atom.Atom, kind, 32, 1, data).Check())
	}
	property(b.root, "_NET_ACTIVE_WINDOW", xproto.AtomWindow, uint32(window))
	property(window, "_NET_WM_PID", xproto.AtomCardinal, uint32(os.Getpid()))
	require.NoError(t, xproto.SetInputFocusChecked(b.conn, xproto.InputFocusPointerRoot, window, xproto.TimeCurrentTime).Check())
	// No compositor: image capture must refuse, while metadata remains usable.
	absent, err := b.CaptureActivationImage(context.Background())
	require.Error(t, err)
	require.Nil(t, absent.Image)
	_, err = b.CaptureActivation(context.Background())
	require.NoError(t, err)
	// Only the fixture redirects its window; production capture never does.
	require.NoError(t, composite.Init(b.conn))
	_, err = composite.QueryVersion(b.conn, 0, 4).Reply()
	require.NoError(t, err)
	require.NoError(t, composite.RedirectWindowChecked(b.conn, window, composite.RedirectAutomatic).Check())
	require.NoError(t, xproto.ClearAreaChecked(b.conn, false, window, 0, 0, 0, 0).Check())
	_ = create(0x00ff00, 0) // Fully overlaps the selected client's upper-left content.
	captured, err := b.CaptureActivationImage(context.Background())
	require.NoError(t, err)
	require.EqualValues(t, window, captured.Context.ActiveWindow)
	require.Equal(t, 80, captured.Image.Bounds().Dx())
	require.Equal(t, 60, captured.Image.Bounds().Dy())
	for y := 0; y < 60; y++ {
		for x := 0; x < 80; x++ {
			require.Equal(t, color.RGBA{R: 255, A: 255}, captured.Image.RGBAAt(x, y))
		}
	}
	focus, err := xproto.GetInputFocus(b.conn).Reply()
	require.NoError(t, err)
	require.Equal(t, window, focus.Focus)
	require.NoError(t, xproto.ChangeWindowAttributesChecked(b.conn, window, xproto.CwBackPixel, []uint32{0x0000ff}).Check())
	require.NoError(t, xproto.ClearAreaChecked(b.conn, false, window, 0, 0, 0, 0).Check())
	fresh, err := b.CaptureActivationImage(context.Background())
	require.NoError(t, err)
	require.Equal(t, color.RGBA{B: 255, A: 255}, fresh.Image.RGBAAt(0, 0))
	require.Equal(t, color.RGBA{R: 255, A: 255}, captured.Image.RGBAAt(0, 0))
	allowed = false
	denied, err := b.CaptureActivationImage(context.Background())
	require.Error(t, err)
	require.Equal(t, sessions.DesktopActivationImage{}, denied)
	allowed = true
	checks := 0
	b.check = func(context.Context, Peer) error {
		checks++
		if checks > 1 {
			return ErrUnavailable
		}
		return nil
	}
	denied, err = b.CaptureActivationImage(context.Background())
	require.Error(t, err)
	require.Equal(t, sessions.DesktopActivationImage{}, denied)
	b.check = func(context.Context, Peer) error { return nil }
	require.NoError(t, xproto.UnmapWindowChecked(b.conn, window).Check())
	denied, err = b.CaptureActivationImage(context.Background())
	require.Error(t, err)
	require.Equal(t, sessions.DesktopActivationImage{}, denied)
	pixels, err := b.activationPixels(window, sessions.DesktopBounds{Width: 65535, Height: 65535})
	require.Error(t, err)
	require.Nil(t, pixels)
}
