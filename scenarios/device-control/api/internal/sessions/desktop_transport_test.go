package sessions

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"device-control/internal/testutil"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/localprincipal"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop/desktopv1connect"
)

type transportNative struct {
	effects      atomic.Int32
	releaseCalls atomic.Int32
	releaseErr   error
}

func (*transportNative) Validate(context.Context, DesktopCommand) error { return nil }
func (n *transportNative) Apply(context.Context, DesktopCommand) error  { n.effects.Add(1); return nil }
func (n *transportNative) ReleaseHeld(context.Context) error {
	n.releaseCalls.Add(1)
	return n.releaseErr
}

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
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: filepath.Join(testutil.SocketDir(t), "h.sock"), Net: "unix"})
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

func TestDesktopInvokeTransportValidation(t *testing.T) {
	require.True(t, validDesktopAction(&desktopv1.Action{Action: &desktopv1.Action_Invoke{Invoke: &desktopv1.InvokeAction{ElementId: "element", ObservationRevision: "revision"}}}))
	require.False(t, validDesktopAction(&desktopv1.Action{Action: &desktopv1.Action_Invoke{Invoke: &desktopv1.InvokeAction{ElementId: "element"}}}))
	require.False(t, validDesktopAction(&desktopv1.Action{Action: &desktopv1.Action_Invoke{Invoke: &desktopv1.InvokeAction{ElementId: string(make([]byte, 129)), ObservationRevision: "revision"}}}))
}

func TestDesktopUnixServeShutdownReleasesActiveLeaseOnce(t *testing.T) {
	native := &transportNative{}
	c, repo, lease, _, _, stop := startDesktopUnixHelperForShutdownTest(t, native)
	err := stop()
	require.NoError(t, err)
	require.EqualValues(t, 1, native.releaseCalls.Load(), "shutdown must own exactly one final native release")

	var hasLease, halted, pending bool
	require.NoError(t, repo.Update(context.Background(), func(s *DesktopState) error {
		hasLease = s.Lease != nil
		halted = s.Halted
		pending = s.CleanupPending
		return nil
	}))
	require.False(t, hasLease)
	require.True(t, halted)
	require.False(t, pending)
	receipt, err := repo.ReadCleanup(context.Background(), lease)
	require.NoError(t, err)
	require.True(t, receipt.Released)
	require.NotNil(t, c)
}

func TestDesktopUnixServeShutdownSurfacesFailedNativeRelease(t *testing.T) {
	releaseErr := errors.New("native held-input release failed")
	native := &transportNative{releaseErr: releaseErr}
	c, repo, lease, _, _, stop := startDesktopUnixHelperForShutdownTest(t, native)
	err := stop()
	require.ErrorIs(t, err, ErrDesktopCleanupPending)
	require.ErrorIs(t, err, releaseErr)
	require.EqualValues(t, 1, native.releaseCalls.Load(), "failed cleanup must not be hidden by a duplicate retry")

	var hasLease, halted, pending, receiptReleased bool
	require.NoError(t, repo.Update(context.Background(), func(s *DesktopState) error {
		hasLease = s.Lease != nil
		halted = s.Halted
		pending = s.CleanupPending
		receiptReleased = s.CleanupReceipts[lease.Epoch].Released
		return nil
	}))
	require.False(t, hasLease, "failed native release still revokes the lease")
	require.True(t, halted)
	require.True(t, pending, "failed cleanup must remain durable and uncertain")
	require.False(t, receiptReleased)
	require.NotNil(t, c)
}

func TestDesktopRequestTrackerStopsAdmissionAndReportsWaitTimeout(t *testing.T) {
	tracker := newDesktopRequestTracker()
	require.True(t, tracker.enter())
	tracker.stop()
	require.False(t, tracker.enter(), "new requests must not enter after shutdown starts")
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	require.ErrorIs(t, tracker.wait(ctx), context.DeadlineExceeded)
	tracker.leave()
	require.NoError(t, tracker.wait(context.Background()))
}

type blockingTransportNative struct {
	transportNative
	started     chan struct{}
	validateEnd chan error
	startOnce   sync.Once
}

func (n *blockingTransportNative) Validate(ctx context.Context, _ DesktopCommand) error {
	n.startOnce.Do(func() { close(n.started) })
	<-ctx.Done()
	err := ctx.Err()
	n.validateEnd <- err
	return err
}

func TestDesktopUnixServeCancelsRequestContextsBeforeCleanup(t *testing.T) {
	native := &blockingTransportNative{started: make(chan struct{}), validateEnd: make(chan error, 1)}
	_, _, lease, socketPath, private, stop := startDesktopUnixHelperForShutdownTest(t, native)
	principal, err := localprincipal.Current()
	require.NoError(t, err)
	grant := DesktopGrant{ID: "shutdown-grant", Principal: principal, Lease: lease, Operations: []string{"act"}, IssuedAt: time.Now().Add(-time.Second), ExpiresAt: lease.ExpiresAt}
	token, err := SignDesktopGrant(private, grant, time.Now())
	require.NoError(t, err)
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
	}}
	defer transport.CloseIdleConnections()
	client := desktopv1connect.NewDesktopHelperServiceClient(&http.Client{Transport: transport, Timeout: 5 * time.Second}, "http://desktop-helper")
	request := connect.NewRequest(&desktopv1.ActRequest{Lease: leaseToWire(lease), CommandId: "shutdown-command", GeometryRevision: "geometry", Action: &desktopv1.Action{Action: &desktopv1.Action_Pointer{Pointer: &desktopv1.PointerAction{Kind: desktopv1.PointerAction_KIND_CLICK, DisplayId: "display", X: 1, Y: 1, Button: desktopv1.PointerAction_BUTTON_PRIMARY}}}})
	request.Header().Set("Authorization", "DesktopGrant "+token)
	actDone := make(chan error, 1)
	go func() { _, err := client.Act(context.Background(), request); actDone <- err }()
	select {
	case <-native.started:
	case <-time.After(5 * time.Second):
		t.Fatal("native validation did not start")
	}

	require.NoError(t, stop())
	select {
	case err := <-native.validateEnd:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("helper shutdown did not cancel the in-flight RPC context")
	}
	select {
	case <-actDone:
	case <-time.After(time.Second):
		t.Fatal("in-flight RPC did not finish after cancellation")
	}
	require.EqualValues(t, 1, native.releaseCalls.Load())
}

type stubbornTransportNative struct {
	transportNative
	started chan struct{}
	resume  chan struct{}
	once    sync.Once
}

func (n *stubbornTransportNative) Validate(context.Context, DesktopCommand) error {
	n.once.Do(func() { close(n.started) })
	<-n.resume // Simulate a native call that does not return when its RPC context is canceled.
	return nil
}

func TestDesktopUnixServeKeepsResourcesAlivePastHandlerDrainTimeout(t *testing.T) {
	native := &stubbornTransportNative{started: make(chan struct{}), resume: make(chan struct{})}
	type lifecycleObservation struct {
		status string
		active int
		code   string
	}
	var lifecycleMu sync.Mutex
	var lifecycle []lifecycleObservation
	reporter := func(status string, active int, code string) error {
		lifecycleMu.Lock()
		defer lifecycleMu.Unlock()
		lifecycle = append(lifecycle, lifecycleObservation{status: status, active: active, code: code})
		return nil
	}
	_, _, lease, socketPath, private, stop := startDesktopUnixHelperForShutdownTest(t, native, reporter)
	principal, err := localprincipal.Current()
	require.NoError(t, err)
	grant := DesktopGrant{ID: "stubborn-shutdown-grant", Principal: principal, Lease: lease, Operations: []string{"act"}, IssuedAt: time.Now().Add(-time.Second), ExpiresAt: lease.ExpiresAt}
	token, err := SignDesktopGrant(private, grant, time.Now())
	require.NoError(t, err)
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
	}}
	defer transport.CloseIdleConnections()
	client := desktopv1connect.NewDesktopHelperServiceClient(&http.Client{Transport: transport, Timeout: 15 * time.Second}, "http://desktop-helper")
	request := connect.NewRequest(&desktopv1.ActRequest{Lease: leaseToWire(lease), CommandId: "stubborn-shutdown-command", GeometryRevision: "geometry", Action: &desktopv1.Action{Action: &desktopv1.Action_Pointer{Pointer: &desktopv1.PointerAction{Kind: desktopv1.PointerAction_KIND_CLICK, DisplayId: "display", X: 1, Y: 1, Button: desktopv1.PointerAction_BUTTON_PRIMARY}}}})
	request.Header().Set("Authorization", "DesktopGrant "+token)
	actDone := make(chan error, 1)
	go func() { _, err := client.Act(context.Background(), request); actDone <- err }()
	select {
	case <-native.started:
	case <-time.After(5 * time.Second):
		t.Fatal("native validation did not start")
	}

	stopDone := make(chan error, 1)
	go func() { stopDone <- stop() }()
	select {
	case err := <-stopDone:
		t.Fatalf("helper returned while an accepted handler still used its resources: %v", err)
	case <-time.After(desktopHelperShutdownTimeout + 100*time.Millisecond):
	}
	require.Zero(t, native.releaseCalls.Load(), "native cleanup must wait until the accepted handler stops")
	lifecycleMu.Lock()
	require.Contains(t, lifecycle, lifecycleObservation{status: "stopping", active: 1, code: "rpc_drain_timeout"}, "the bounded drain result must be reported before the handler exits")
	lifecycleMu.Unlock()

	close(native.resume)
	select {
	case err := <-stopDone:
		require.ErrorIs(t, err, context.DeadlineExceeded, "the exceeded drain timeout remains visible after safe cleanup")
	case <-time.After(5 * time.Second):
		t.Fatal("helper did not finish cleanup after the accepted handler exited")
	}
	select {
	case <-actDone:
	case <-time.After(time.Second):
		t.Fatal("accepted RPC did not finish after native validation resumed")
	}
	require.EqualValues(t, 1, native.releaseCalls.Load())
	lifecycleMu.Lock()
	require.Contains(t, lifecycle, lifecycleObservation{status: "controller-cleanup-completed", active: 0}, "controller cleanup receipt follows handler exit")
	lifecycleMu.Unlock()
}

func startDesktopUnixHelperForShutdownTest(t *testing.T, native DesktopNative, reporters ...func(string, int, string) error) (*DesktopController, *SQLiteDesktopRepository, DesktopLease, string, ed25519.PrivateKey, func() error) {
	t.Helper()
	controller, repo, _, _, lease := desktopFixture(t)
	public, private, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	authority, err := NewSignedDesktopAuthority(public, func(context.Context, string) (bool, error) { return true, nil })
	require.NoError(t, err)
	controller.authority = authority
	controller.native = native
	helper, err := NewDesktopUnixHelper(controller, authority)
	require.NoError(t, err)
	if len(reporters) > 0 {
		helper.SetLifecycleReporter(reporters[0])
	}
	socketPath := filepath.Join(testutil.SocketDir(t), "shutdown.sock")
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: socketPath, Net: "unix"})
	require.NoError(t, err)
	require.NoError(t, os.Chmod(listener.Addr().String(), 0o600))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- helper.Serve(ctx, listener) }()
	var once sync.Once
	var serveErr error
	stop := func() error {
		once.Do(func() {
			cancel()
			select {
			case serveErr = <-done:
			case <-time.After(10 * time.Second):
				serveErr = errors.New("helper did not shut down")
			}
		})
		return serveErr
	}
	t.Cleanup(func() { _ = stop() })
	return controller, repo, lease, socketPath, private, stop
}
