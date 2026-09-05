package x11

import (
	"context"
	"encoding/binary"
	"os"
	"strings"
	"testing"

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
