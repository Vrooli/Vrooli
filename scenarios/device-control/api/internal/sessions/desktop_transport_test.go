package sessions

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/localprincipal"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop/desktopv1connect"
)

type transportNative struct{ effects atomic.Int32 }

func (*transportNative) Validate(context.Context, DesktopCommand) error { return nil }
func (n *transportNative) Apply(context.Context, DesktopCommand) error  { n.effects.Add(1); return nil }
func (*transportNative) ReleaseHeld(context.Context) error              { return nil }

func (*transportNative) CaptureActivation(context.Context) (DesktopActivationContext, error) {
	return DesktopActivationContext{SourceBounds: DesktopBounds{X: 10, Y: 20, Width: 200, Height: 100}, ActiveWindow: 991, ProcessID: 992, DisplayID: "display-1", GeometryRevision: "geometry-1", PointerX: -20, PointerY: 30, CapturedAt: time.Now()}, nil
}

func TestDesktopUnixTransportAuthenticatesAndExecutesTypedActions(t *testing.T) { // [REQ:DEVICECONTROL-EVERYWHERE-AUTH-09]
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Unix peer transport")
	}
	c, _, _, _, lease := desktopFixture(t)
	public, private, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	var active atomic.Bool
	active.Store(true)
	authority, err := NewSignedDesktopAuthority(public, func(context.Context, string) (bool, error) { return active.Load(), nil })
	require.NoError(t, err)
	c.authority = authority
	native := &transportNative{}
	c.native = native
	helper, err := NewDesktopUnixHelper(c, authority)
	require.NoError(t, err)
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: filepath.Join(t.TempDir(), "h.sock"), Net: "unix"})
	require.NoError(t, err)
	require.NoError(t, os.Chmod(listener.Addr().String(), 0o600))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- helper.Serve(ctx, listener) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			require.NoError(t, err)
		case <-time.After(10 * time.Second):
			t.Error("helper failed to shut down")
		}
	})
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", listener.Addr().String())
	}}
	t.Cleanup(transport.CloseIdleConnections)
	client := desktopv1connect.NewDesktopHelperServiceClient(&http.Client{Transport: transport, Timeout: 5 * time.Second}, "http://desktop-helper")
	principal, err := localprincipal.Current()
	require.NoError(t, err)
	lease.Epoch++
	lease.Ref.SessionID = "wire-session"
	grant := DesktopGrant{ID: "wire-grant", Principal: principal, Lease: lease, Operations: []string{"open", "transfer", "act", "stop", "observe"}, IssuedAt: time.Now().Add(-time.Second), ExpiresAt: lease.ExpiresAt}
	token, err := SignDesktopGrant(private, grant, time.Now())
	require.NoError(t, err)
	open := connect.NewRequest(&desktopv1.OpenRequest{Lease: leaseToWire(lease), ExpectedEpoch: lease.Epoch - 1, Takeover: true})
	_, err = client.Open(context.Background(), open)
	require.Error(t, err)
	require.Zero(t, native.effects.Load())
	open.Header().Set("Authorization", "DesktopGrant "+token)
	opened, err := client.Open(context.Background(), open)
	require.NoError(t, err)
	require.Equal(t, lease.Epoch, opened.Msg.Lease.Epoch)
	capture := connect.NewRequest(&desktopv1.StopRequest{Lease: opened.Msg.Lease})
	_, err = client.CaptureActivation(context.Background(), capture)
	require.Error(t, err)
	capture.Header().Set("Authorization", "DesktopCleanup "+token)
	_, err = client.CaptureActivation(context.Background(), capture)
	require.Error(t, err)
	capture.Header().Set("Authorization", "DesktopGrant "+token)
	captured, err := client.CaptureActivation(context.Background(), capture)
	require.NoError(t, err)
	require.NotEmpty(t, captured.Msg.ContextId)
	require.EqualValues(t, -20, captured.Msg.PointerX)
	readContext := connect.NewRequest(&desktopv1.ReadActivationRequest{Lease: opened.Msg.Lease, ContextId: captured.Msg.ContextId})
	readContext.Header().Set("Authorization", "DesktopGrant "+token)
	confirmed, err := client.ReadActivation(context.Background(), readContext)
	require.NoError(t, err)
	require.Equal(t, captured.Msg.ContextId, confirmed.Msg.ContextId)
	active.Store(false)
	_, err = client.ReadActivation(context.Background(), readContext)
	require.Error(t, err)
	active.Store(true)
	require.Zero(t, native.effects.Load())

	missing := connect.NewRequest(&desktopv1.StopRequest{Lease: opened.Msg.Lease})
	missing.Header().Set("Authorization", "DesktopCleanup "+token)
	_, err = client.ReadCleanup(context.Background(), missing)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	request := connect.NewRequest(&desktopv1.ActRequest{Lease: opened.Msg.Lease, CommandId: "wire-command", GeometryRevision: "geometry-1", Action: &desktopv1.Action{Action: &desktopv1.Action_Pointer{Pointer: &desktopv1.PointerAction{Kind: desktopv1.PointerAction_KIND_CLICK, DisplayId: "display-1", X: 20, Y: 30, Button: desktopv1.PointerAction_BUTTON_PRIMARY}}}})
	request.Header().Set("Authorization", "DesktopGrant "+token)
	first, err := client.Act(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, "applied", first.Msg.Receipt.Outcome)
	second, err := client.Act(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, first.Msg.Receipt.Digest, second.Msg.Receipt.Digest)
	require.EqualValues(t, 1, native.effects.Load())
	active.Store(false)
	_, err = client.Act(context.Background(), request)
	require.Error(t, err)
	require.EqualValues(t, 1, native.effects.Load())
	active.Store(true)
	stop := connect.NewRequest(&desktopv1.StopRequest{Lease: opened.Msg.Lease})
	stop.Header().Set("Authorization", "DesktopGrant "+token)
	_, err = client.Stop(context.Background(), stop)
	require.NoError(t, err)
	_, err = client.Act(context.Background(), request)
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	require.EqualValues(t, 1, native.effects.Load())
	active.Store(false)
	cleanup := connect.NewRequest(&desktopv1.StopRequest{Lease: opened.Msg.Lease})
	cleanup.Header().Set("Authorization", "DesktopCleanup "+token)
	history, err := client.ReadCleanup(context.Background(), cleanup)
	require.NoError(t, err)
	require.True(t, history.Msg.Released)
	require.Equal(t, opened.Msg.Lease.Epoch, history.Msg.Lease.Epoch)
	request.Header().Set("Authorization", "DesktopCleanup "+token)
	_, err = client.Act(context.Background(), request)
	require.Error(t, err)
	stop.Header().Set("Authorization", "DesktopCleanup "+token)
	_, err = client.Stop(context.Background(), stop)
	require.Error(t, err)
	cleanup.Header().Set("Authorization", "DesktopGrant "+token)
	_, err = client.ReadCleanup(context.Background(), cleanup)
	require.Error(t, err)
	require.EqualValues(t, 1, native.effects.Load())
}

func TestDesktopWheelTransportValidation(t *testing.T) {
	for _, ticks := range []int32{-20, -1, 1, 20} {
		require.True(t, validDesktopAction(&desktopv1.Action{Action: &desktopv1.Action_Wheel{Wheel: &desktopv1.WheelAction{DisplayId: "display", VerticalTicks: ticks}}}))
	}
	for _, ticks := range []int32{0, 21, -21, -2147483648} {
		require.False(t, validDesktopAction(&desktopv1.Action{Action: &desktopv1.Action_Wheel{Wheel: &desktopv1.WheelAction{DisplayId: "display", VerticalTicks: ticks}}}))
	}
	require.False(t, validDesktopAction(&desktopv1.Action{Action: &desktopv1.Action_Wheel{}}))
}
