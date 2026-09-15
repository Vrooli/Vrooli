package x11

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"image/png"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"device-control/internal/sessions"
	"github.com/jezek/xgb/xproto"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/localprincipal"
	"github.com/vrooli/api-core/targetmodel"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop/desktopv1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
	_ "modernc.org/sqlite"
)

func isolatedDisplay(t *testing.T) uint16 {
	t.Helper()
	path, err := exec.LookPath("Xvfb")
	if err != nil {
		t.Skip("Xvfb fixture unavailable")
	}
	read, write, err := os.Pipe()
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	command := exec.CommandContext(ctx, path, "-displayfd", "3", "-screen", "0", "640x480x24", "-nolisten", "tcp", "-ac")
	command.ExtraFiles = []*os.File{write}
	require.NoError(t, command.Start())
	write.Close()
	t.Cleanup(func() { cancel(); _ = command.Wait(); read.Close() })
	require.NoError(t, read.SetReadDeadline(time.Now().Add(5*time.Second)))
	line, err := bufio.NewReader(read).ReadString('\n')
	require.NoError(t, err)
	number, err := strconv.ParseUint(strings.TrimSpace(line), 10, 16)
	require.NoError(t, err)
	return uint16(number)
}

func TestNativeX11CapturePointerAndRelease(t *testing.T) { // [REQ:DEVICECONTROL-EVERYWHERE-NAT-01] [REQ:DEVICECONTROL-EVERYWHERE-NAT-09]
	display := isolatedDisplay(t)
	allowed := true
	backend, err := newBackend(context.Background(), display, strings.Repeat("00", 16), func(context.Context, Peer) error {
		if !allowed {
			return ErrUnavailable
		}
		return nil
	})
	require.NoError(t, err)
	defer backend.Close()
	require.NoError(t, xproto.ChangeWindowAttributesChecked(backend.conn, backend.root, xproto.CwBackPixel, []uint32{0xff0000}).Check())
	require.NoError(t, xproto.ClearAreaChecked(backend.conn, false, backend.root, 0, 0, 0, 0).Check())
	snapshot, err := backend.Observe(context.Background())
	require.NoError(t, err)
	require.Equal(t, 640, snapshot.Image.Bounds().Dx())
	require.Equal(t, 480, snapshot.Image.Bounds().Dy())
	r, g, b, a := snapshot.Image.At(10, 10).RGBA()
	require.EqualValues(t, 65535, r)
	require.Zero(t, g)
	require.Zero(t, b)
	require.EqualValues(t, 65535, a)
	action := &desktopv1.Action{Action: &desktopv1.Action_Pointer{Pointer: &desktopv1.PointerAction{Kind: desktopv1.PointerAction_KIND_DOWN, Button: desktopv1.PointerAction_BUTTON_PRIMARY, DisplayId: snapshot.DisplayID, X: 120, Y: 80}}}
	payload, err := protojson.Marshal(action)
	require.NoError(t, err)
	cmd := sessions.DesktopCommand{GeometryRevision: snapshot.GeometryRevision, Payload: payload}
	require.NoError(t, backend.Validate(context.Background(), cmd))
	require.NoError(t, backend.Apply(context.Background(), cmd))
	pointer, err := xproto.QueryPointer(backend.conn, backend.root).Reply()
	require.NoError(t, err)
	require.EqualValues(t, 120, pointer.RootX)
	require.EqualValues(t, 80, pointer.RootY)
	require.NotZero(t, pointer.Mask&xproto.ButtonMask1)
	require.NoError(t, backend.ReleaseHeld(context.Background()))
	pointer, err = xproto.QueryPointer(backend.conn, backend.root).Reply()
	require.NoError(t, err)
	require.Zero(t, pointer.Mask&xproto.ButtonMask1)
	// Preserve a button pressed outside this helper's ownership.
	require.NoError(t, backend.event(xproto.ButtonPress, 1, 0, 0))
	require.Error(t, backend.Validate(context.Background(), cmd))
	require.Error(t, backend.Apply(context.Background(), cmd))
	action.GetPointer().Kind = desktopv1.PointerAction_KIND_UP
	cmd.Payload, err = protojson.Marshal(action)
	require.NoError(t, err)
	require.Error(t, backend.Apply(context.Background(), cmd))
	require.NoError(t, backend.ReleaseHeld(context.Background()))
	pointer, err = xproto.QueryPointer(backend.conn, backend.root).Reply()
	require.NoError(t, err)
	require.NotZero(t, pointer.Mask&xproto.ButtonMask1)
	require.NoError(t, backend.event(xproto.ButtonRelease, 1, 0, 0))
	action.GetPointer().Kind = desktopv1.PointerAction_KIND_DOWN
	cmd.Payload, err = protojson.Marshal(action)
	require.NoError(t, err)
	cmd.GeometryRevision = "stale"
	require.Error(t, backend.Apply(context.Background(), cmd))
	allowed = false
	_, err = backend.Observe(context.Background())
	require.Error(t, err)
	require.Error(t, backend.Validate(context.Background(), sessions.DesktopCommand{GeometryRevision: snapshot.GeometryRevision, Payload: payload}))
}

func TestNativeX11KeyboardOwnershipAndRelease(t *testing.T) { // [REQ:DEVICECONTROL-EVERYWHERE-NAT-09]
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
	snapshot, err := b.Observe(context.Background())
	require.NoError(t, err)
	command := func(name string, kind desktopv1.KeyAction_Kind) sessions.DesktopCommand {
		payload, err := protojson.Marshal(&desktopv1.Action{Action: &desktopv1.Action_Key{Key: &desktopv1.KeyAction{Key: name, Kind: kind}}})
		require.NoError(t, err)
		return sessions.DesktopCommand{GeometryRevision: snapshot.GeometryRevision, Payload: payload}
	}
	down := command("ShiftLeft", desktopv1.KeyAction_KIND_DOWN)
	require.NoError(t, b.Validate(context.Background(), down))
	require.NoError(t, b.Apply(context.Background(), down))
	code := b.heldKeys["ShiftLeft"]
	require.NotZero(t, code)
	state, err := xproto.QueryKeymap(b.conn).Reply()
	require.NoError(t, err)
	require.NotZero(t, state.Keys[code/8]&(1<<(code%8)))
	// A press may not consume a modifier held by an earlier command.
	require.Error(t, b.Apply(context.Background(), command("ShiftLeft", desktopv1.KeyAction_KIND_PRESS)))
	allowed = false
	require.Error(t, b.Apply(context.Background(), command("Enter", desktopv1.KeyAction_KIND_PRESS)))
	require.NoError(t, b.ReleaseHeld(context.Background()))
	state, err = xproto.QueryKeymap(b.conn).Reply()
	require.NoError(t, err)
	require.Zero(t, state.Keys[code/8]&(1<<(code%8)))
	allowed = true
	// A key held outside this helper must not be acquired or released by it.
	require.NoError(t, b.event(xproto.KeyPress, code, 0, 0))
	require.Error(t, b.Apply(context.Background(), down))
	require.Error(t, b.Apply(context.Background(), command("ShiftLeft", desktopv1.KeyAction_KIND_UP)))
	require.NoError(t, b.ReleaseHeld(context.Background()))
	state, err = xproto.QueryKeymap(b.conn).Reply()
	require.NoError(t, err)
	require.NotZero(t, state.Keys[code/8]&(1<<(code%8)))
	require.NoError(t, b.event(xproto.KeyRelease, code, 0, 0))
	for _, key := range []string{"Enter", "ArrowLeft", "F1", "a", "7"} {
		require.NoError(t, b.Apply(context.Background(), command(key, desktopv1.KeyAction_KIND_PRESS)))
	}
	require.Empty(t, b.heldKeys)
	require.Error(t, b.Apply(context.Background(), command("unsupported", desktopv1.KeyAction_KIND_PRESS)))
	down.GeometryRevision = "old-layout"
	require.Error(t, b.Apply(context.Background(), down))
}

func TestNativeX11WheelDeliveryAndBounds(t *testing.T) {
	display := isolatedDisplay(t)
	b, err := newBackend(context.Background(), display, strings.Repeat("00", 16), func(context.Context, Peer) error { return nil })
	require.NoError(t, err)
	t.Cleanup(func() { _ = b.Close() })
	require.NoError(t, xproto.ChangeWindowAttributesChecked(b.conn, b.root, xproto.CwEventMask, []uint32{xproto.EventMaskButtonPress | xproto.EventMaskButtonRelease}).Check())
	snapshot, err := b.Observe(context.Background())
	require.NoError(t, err)
	makeCommand := func(h, v int32) sessions.DesktopCommand {
		payload, err := protojson.Marshal(&desktopv1.Action{Action: &desktopv1.Action_Wheel{Wheel: &desktopv1.WheelAction{DisplayId: snapshot.DisplayID, X: 50, Y: 60, HorizontalTicks: h, VerticalTicks: v}}})
		require.NoError(t, err)
		return sessions.DesktopCommand{GeometryRevision: snapshot.GeometryRevision, Payload: payload}
	}
	for _, ticks := range [][2]int32{{0, 0}, {21, 0}, {10, 11}, {-2147483648, 0}} {
		command := makeCommand(ticks[0], ticks[1])
		require.Error(t, b.Validate(context.Background(), command))
		require.Error(t, b.Apply(context.Background(), command))
	}
	command := makeCommand(1, -2)
	require.NoError(t, b.Validate(context.Background(), command))
	require.NoError(t, b.Apply(context.Background(), command))
	require.NoError(t, b.Apply(context.Background(), makeCommand(-1, 2)))
	var presses, releases []byte
	for {
		event, err := b.conn.PollForEvent()
		require.NoError(t, err)
		if event == nil {
			break
		}
		switch e := event.(type) {
		case xproto.ButtonPressEvent:
			presses = append(presses, byte(e.Detail))
			require.EqualValues(t, 50, e.RootX)
			require.EqualValues(t, 60, e.RootY)
		case xproto.ButtonReleaseEvent:
			releases = append(releases, byte(e.Detail))
		}
	}
	require.Equal(t, []byte{4, 4, 7, 5, 5, 6}, presses)
	require.Equal(t, presses, releases)
	require.Empty(t, b.held)
	command.GeometryRevision = "old-layout"
	require.Error(t, b.Apply(context.Background(), command))
}

func TestAuthenticatedHelperObservesAndControlsNativeX11(t *testing.T) {
	display := isolatedDisplay(t)
	backend, err := newBackend(context.Background(), display, strings.Repeat("00", 16), func(context.Context, Peer) error { return nil })
	require.NoError(t, err)
	t.Cleanup(func() { _ = backend.Close() })
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "helper.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	repo, err := sessions.NewSQLiteDesktopRepository(context.Background(), db, "fixture-desktop")
	require.NoError(t, err)
	public, private, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	var active atomic.Bool
	active.Store(true)
	authority, err := sessions.NewSignedDesktopAuthority(public, func(context.Context, string) (bool, error) { return active.Load(), nil })
	require.NoError(t, err)
	surface := targetmodel.SurfaceRef{Target: targetmodel.TargetRef{OwnerScenario: "vrooli-bridge", ResourceID: "fixture-host", HostNodeID: "fixture-host"}, OwnerScenario: "device-control", SurfaceID: "fixture-desktop"}
	controller, err := sessions.NewDesktopController(repo, authority, backend, surface, "fixture-login", "fixture-helper")
	require.NoError(t, err)
	require.NoError(t, controller.ActivateHelper(context.Background(), "", 0))
	helper, err := sessions.NewDesktopUnixHelper(controller, authority)
	require.NoError(t, err)
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: filepath.Join(t.TempDir(), "h.sock"), Net: "unix"})
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- helper.Serve(ctx, listener) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			require.NoError(t, err)
		case <-time.After(10 * time.Second):
			t.Error("helper shutdown timeout")
		}
	})
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", listener.Addr().String())
	}}
	t.Cleanup(transport.CloseIdleConnections)
	client := desktopv1connect.NewDesktopHelperServiceClient(&http.Client{Transport: transport, Timeout: 5 * time.Second}, "http://fixture-helper")
	lease := sessions.DesktopLease{Ref: targetmodel.SessionRef{Surface: surface, SessionID: "fixture-session", DesktopSessionID: "fixture-login"}, Actor: "fixture-operator", HelperID: "fixture-helper", Epoch: 2, ExpiresAt: time.Now().Add(time.Minute), Control: true}
	principal, err := localprincipal.Current()
	require.NoError(t, err)
	token, err := sessions.SignDesktopGrant(private, sessions.DesktopGrant{ID: "fixture-grant", Principal: principal, Lease: lease, Operations: []string{"open", "observe", "act", "stop"}, IssuedAt: time.Now(), ExpiresAt: lease.ExpiresAt}, time.Now())
	require.NoError(t, err)
	wireLease := &desktopv1.Lease{Ref: lease.Ref.Proto(), Actor: lease.Actor, HelperId: lease.HelperID, Epoch: lease.Epoch, ExpiresAt: timestamppb.New(lease.ExpiresAt), Control: true}
	open := connect.NewRequest(&desktopv1.OpenRequest{Lease: wireLease, ExpectedEpoch: 1})
	open.Header().Set("Authorization", "DesktopGrant "+token)
	_, err = client.Open(context.Background(), open)
	require.NoError(t, err)
	observe := connect.NewRequest(&desktopv1.ObserveRequest{Lease: wireLease})
	observe.Header().Set("Authorization", "DesktopGrant "+token)
	captured, err := client.Observe(context.Background(), observe)
	require.NoError(t, err)
	pixels, err := png.Decode(bytes.NewReader(captured.Msg.Png))
	require.NoError(t, err)
	require.Equal(t, 640, pixels.Bounds().Dx())
	require.Equal(t, 480, pixels.Bounds().Dy())
	act := connect.NewRequest(&desktopv1.ActRequest{Lease: wireLease, CommandId: "native-wire-click", GeometryRevision: captured.Msg.GeometryRevision, Action: &desktopv1.Action{Action: &desktopv1.Action_Pointer{Pointer: &desktopv1.PointerAction{Kind: desktopv1.PointerAction_KIND_CLICK, Button: desktopv1.PointerAction_BUTTON_PRIMARY, DisplayId: captured.Msg.DisplayId, X: 210, Y: 130}}}})
	act.Header().Set("Authorization", "DesktopGrant "+token)
	receipt, err := client.Act(context.Background(), act)
	require.NoError(t, err)
	require.Equal(t, "applied", receipt.Msg.Receipt.Outcome)
	pointer, err := xproto.QueryPointer(backend.conn, backend.root).Reply()
	require.NoError(t, err)
	require.EqualValues(t, 210, pointer.RootX)
	require.EqualValues(t, 130, pointer.RootY)
	require.Zero(t, pointer.Mask&xproto.ButtonMask1)
	// Revoke after a native press, without sending any further helper request.
	// The maintenance loop must release it while the original lease is unexpired.
	act.Msg.CommandId = "native-held-button"
	act.Msg.Action.GetPointer().Kind = desktopv1.PointerAction_KIND_DOWN
	_, err = client.Act(context.Background(), act)
	require.NoError(t, err)
	pointer, err = xproto.QueryPointer(backend.conn, backend.root).Reply()
	require.NoError(t, err)
	require.NotZero(t, pointer.Mask&xproto.ButtonMask1)
	active.Store(false)
	require.Eventually(t, func() bool {
		pointer, err := xproto.QueryPointer(backend.conn, backend.root).Reply()
		return err == nil && pointer != nil && pointer.Mask&xproto.ButtonMask1 == 0
	}, 2*time.Second, 10*time.Millisecond)

}
