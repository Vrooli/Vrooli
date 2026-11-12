package resources

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

func nativeTransportFixture(t *testing.T) (openRGBNativeCapsule, openRGBNativeMaterial, platform.ServiceRef, openRGBExecWitness, openRGBNativeClaimCallbacks) {
	t.Helper()
	x := newOpenRGBExecFixture(t)
	p, catalog, config := nativeResourcePolicyFixture(t)
	p.ArtifactSHA256 = x.update.Target.Artifact.SHA256
	contract, e := x.update.Target.withConfigBytes(x.update.Target.ConfigFile, config)
	if e != nil {
		t.Fatal(e)
	}
	policy, e := freezeOpenRGBNativeExecPolicy(p, p.ArtifactSHA256, catalog, config)
	if e != nil {
		t.Fatal(e)
	}
	rt := x.fence.Intent.Runtime
	rt.NativeAccount = p.Account
	rt.OwnerPrincipal = p.Account
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", t.TempDir())
	t.Setenv("VROOLI_RESOURCE_ARTIFACT_DIR", t.TempDir())
	resourceRoot, err := OpenRGBNativeResourceRoot()
	if err != nil {
		t.Fatal(err)
	}
	rt.StateDir = filepath.Join(resourceRoot, "state")
	rt.Launcher = filepath.Join(t.TempDir(), "sealed-launcher", "vrooli")
	rt.Home = filepath.Join(t.TempDir(), "operator")
	inv, e := renderManagedUpdateInvocation(contract, rt, "fixture", "role", "private-token")
	if e != nil {
		t.Fatal(e)
	}
	i := platform.ServiceIntent(x.ref.Intent)
	i.NativeAccount = rt.NativeAccount
	unit, e := i.UnitName()
	if e != nil {
		t.Fatal(e)
	}
	label := "com.vrooli.owned." + strings.TrimSuffix(unit, ".service")
	ref := platform.ServiceRef{Intent: platform.OperationIntent(i), Unit: unit, Scope: platform.ScopeRef{Kind: platform.ScopeKindNative, Path: "user/501/" + label}, NativeInvocationID: i.InvocationID, RootPID: 123, RootStartTime: "456000007", Native: &platform.NativeWorkloadIdentity{Version: 2, Platform: "darwin", Manager: "launchd", Domain: "user/501", Name: label, InvocationID: i.InvocationID, Account: i.NativeAccount, ContainmentID: "user/501/" + label, ProcessOnly: true, RootUniqueID: "99", BootID: "12345678-1234-1234-1234-123456789abc", RootIncarnation: "darwin:123:456:7"}}
	if e = ref.ValidateIdentity(); e != nil {
		t.Fatal(e)
	}
	m := openRGBNativeMaterial{Contract: contract, Runtime: rt, Invocation: inv, StagePath: filepath.Join(t.TempDir(), "image"), NativePolicy: p, Catalog: catalog, Policy: policy, ClaimSHA256: strings.Repeat("c", 64), TokenHash: managedServiceTokenHash("private-token")}
	c := openRGBNativeCapsule{SchemaVersion: 1, ID: i.AttemptID, Intent: i, Account: i.NativeAccount, LauncherSHA256: rt.LauncherSHA256, RuntimeRoot: inv.RuntimeRoot, MaterialSHA256: m.digest(), Endpoint: filepath.Join(filepath.VolumeName(rt.Root)+string(filepath.Separator), "fixture", "owner.sock"), OwnerPID: 456, OwnerIncarnation: "owner-incarnation", OwnerPrincipal: "unix:501"}
	if e = m.validate(c); e != nil {
		t.Fatal(e)
	}
	w := openRGBExecWitness{DeviceGenerationSHA256: strings.Repeat("e", 64), Policy: policy, Incarnation: ref.Native.RootIncarnation, ClaimSHA256: m.ClaimSHA256, ContextSHA256: strings.Repeat("d", 64), SelectedNodes: []string{"IOService:/usb/port8/interface0"}}
	w.TransportSHA256, _ = hostreqkit.NativeOpenRGBTransportPolicySHA256(m.Policy.Platform)
	cb := openRGBNativeClaimCallbacks{Lookup: func(context.Context, platform.ServiceIntent) (platform.ServiceRef, error) { return ref, nil }, Inspect: func(context.Context, platform.ServiceRef) (platform.OperationObservation, error) {
		return platform.OperationObservation{Ref: platform.OperationRef(ref), State: platform.OperationRunning, Populated: true, RootAlive: true}, nil
	}, Incarnation: func(int) (string, error) { return ref.Native.RootIncarnation, nil }, ResolveAccount: func(context.Context, string) (platform.OwnedServiceAccount, error) {
		return platform.OwnedServiceAccount{Name: c.Account, Principal: "uid:501"}, nil
	}, Prepare: func(context.Context, platform.ServiceRef) (openRGBNativeMaterial, error) { return m, nil }, Claim: func(context.Context, platform.ServiceRef, openRGBExecWitness) error { return nil }}
	return c, m, ref, w, cb
}
func nativeTransportPipe(t *testing.T, s *openRGBNativeClaimServer, c openRGBNativeCapsule) *openRGBNativeClaimClient {
	t.Helper()
	a, b := net.Pipe()
	done := make(chan struct{})
	go func() { defer close(done); defer a.Close(); s.serve(a) }()
	cl := &openRGBNativeClaimClient{conn: b, capsule: c, ops: openRGBNativeTransportOps{peer: func(net.Conn) (int, string, error) { return c.OwnerPID, c.OwnerPrincipal, nil }, incarnation: func(int) (string, error) { return c.OwnerIncarnation, nil }}}
	t.Cleanup(func() {
		cl.Close()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("orphan exchange")
		}
	})
	return cl
}
func nativeTransportServer(c openRGBNativeCapsule, cb openRGBNativeClaimCallbacks) *openRGBNativeClaimServer {
	return newOpenRGBNativeClaimServer(context.Background(), c, cb, openRGBNativeTransportOps{peer: func(net.Conn) (int, string, error) { return 123, "unix:501", nil }})
}

func TestOpenRGBNativeTransportClaimIsGlobalAndUncertainIsConsumed(t *testing.T) {
	for _, uncertain := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "uncertain"}[uncertain], func(t *testing.T) {
			c, _, r, w, cb := nativeTransportFixture(t)
			var claims atomic.Int32
			cb.Claim = func(context.Context, platform.ServiceRef, openRGBExecWitness) error {
				claims.Add(1)
				if uncertain {
					return errors.New("secret internal failure")
				}
				return nil
			}
			s := nativeTransportServer(c, cb)
			defer s.Close()
			cl := nativeTransportPipe(t, s, c)
			if _, _, e := cl.Prepare(context.Background()); e != nil {
				t.Fatal(e)
			}
			e := cl.Claim(context.Background(), r, w)
			if (e != nil) != uncertain {
				t.Fatalf("claim: %v", e)
			}
			if cl.Claim(context.Background(), r, w) == nil {
				t.Fatal("client retried")
			}
			other := nativeTransportPipe(t, s, c)
			if _, _, e = other.Prepare(context.Background()); e == nil {
				t.Fatal("consumed claim disclosed material")
			}
			if other.Claim(context.Background(), r, w) == nil {
				t.Fatal("global claim retried")
			}
			if claims.Load() != 1 {
				t.Fatal("claim callback count", claims.Load())
			}
		})
	}
}

func TestOpenRGBNativeTransportRejectsPeerAndObservationChanges(t *testing.T) {
	for _, name := range []string{"principal", "pid", "incarnation", "intent", "inspection", "changed-during-inspect", "changed-after-prepare"} {
		t.Run(name, func(t *testing.T) {
			c, m, r, _, cb := nativeTransportFixture(t)
			prepared := 0
			cb.Prepare = func(context.Context, platform.ServiceRef) (openRGBNativeMaterial, error) { prepared++; return m, nil }
			s := nativeTransportServer(c, cb)
			defer s.Close()
			switch name {
			case "principal":
				s.ops.peer = func(net.Conn) (int, string, error) { return 123, "unix:999", nil }
			case "pid":
				s.ops.peer = func(net.Conn) (int, string, error) { return 124, "unix:501", nil }
			case "incarnation":
				s.callbacks.Incarnation = func(int) (string, error) { return "darwin:123:457:7", nil }
			case "intent":
				s.callbacks.Lookup = func(context.Context, platform.ServiceIntent) (platform.ServiceRef, error) {
					r.Intent.AttemptID = "other"
					return r, nil
				}
			case "inspection":
				s.callbacks.Inspect = func(context.Context, platform.ServiceRef) (platform.OperationObservation, error) {
					return platform.OperationObservation{}, nil
				}
			case "changed-during-inspect":
				inspected := false
				inspect := s.callbacks.Inspect
				s.callbacks.Inspect = func(ctx context.Context, ref platform.ServiceRef) (platform.OperationObservation, error) {
					inspected = true
					return inspect(ctx, ref)
				}
				s.callbacks.Incarnation = func(int) (string, error) {
					if inspected {
						return "reused", nil
					}
					return r.Native.RootIncarnation, nil
				}
			case "changed-after-prepare":
				s.callbacks.Incarnation = func(int) (string, error) {
					if prepared > 0 {
						return "reused", nil
					}
					return r.Native.RootIncarnation, nil
				}
			}
			cl := nativeTransportPipe(t, s, c)
			got, _, e := cl.Prepare(context.Background())
			if e == nil || got.TokenHash != "" {
				t.Fatal("untrusted peer received material")
			}
			if name != "changed-after-prepare" && prepared != 0 {
				t.Fatal("prepared before authentication")
			}
		})
	}
}

func TestOpenRGBNativeTransportOwnerDeathAndLostAcknowledgment(t *testing.T) {
	c, _, r, w, cb := nativeTransportFixture(t)
	s := nativeTransportServer(c, cb)
	defer s.Close()
	cl := nativeTransportPipe(t, s, c)
	if _, _, e := cl.Prepare(context.Background()); e != nil {
		t.Fatal(e)
	}
	cl.ops.incarnation = func(int) (string, error) { return "reused", nil }
	if cl.Claim(context.Background(), r, w) == nil {
		t.Fatal("owner death permitted claim")
	}
	if cl.Claim(context.Background(), r, w) == nil {
		t.Fatal("retried death")
	}
}

func TestOpenRGBNativeTransportLostACKCannotRetry(t *testing.T) {
	c, _, r, w, cb := nativeTransportFixture(t)
	var calls atomic.Int32
	entered, release := make(chan struct{}), make(chan struct{})
	cb.Claim = func(context.Context, platform.ServiceRef, openRGBExecWitness) error {
		calls.Add(1)
		close(entered)
		<-release
		return nil
	}
	s := nativeTransportServer(c, cb)
	defer s.Close()
	cl := nativeTransportPipe(t, s, c)
	if _, _, e := cl.Prepare(context.Background()); e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	go func() { done <- cl.Claim(context.Background(), r, w) }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("claim did not reach durable owner")
	}
	cl.Close()
	close(release)
	if e := <-done; e == nil {
		t.Fatal("lost ACK reported success")
	}
	if cl.Claim(context.Background(), r, w) == nil {
		t.Fatal("lost ACK retried")
	}
	other := nativeTransportPipe(t, s, c)
	if _, _, e := other.Prepare(context.Background()); e == nil {
		t.Fatal("lost ACK reopened disclosure")
	}
	if calls.Load() != 1 {
		t.Fatal("claim count", calls.Load())
	}
}

func TestOpenRGBNativeTransportWitnessBindings(t *testing.T) {
	_, m, r, w, _ := nativeTransportFixture(t)
	for _, name := range []string{"policy", "claim", "incarnation", "context", "nodes", "child", "generation", "count"} {
		t.Run(name, func(t *testing.T) {
			bad := w
			switch name {
			case "policy":
				bad.Policy.ID = strings.Repeat("e", 64)
			case "claim":
				bad.ClaimSHA256 = strings.Repeat("e", 64)
			case "incarnation":
				bad.Incarnation = "reused"
			case "context":
				bad.ContextSHA256 = "invalid"
			case "nodes":
				bad.SelectedNodes = []string{strings.Repeat("x", 4097)}
			case "generation":
				bad.DeviceGenerationSHA256 = ""
			case "count":
				bad.SelectedNodes = append(append([]string{}, bad.SelectedNodes...), "IOService:/usb/port9/interface0")
			case "child":
				bad.ExecutionChild = &openRGBNativeExecutionChild{PID: 42}
			}
			if validateOpenRGBNativeTransportWitness(m, r, bad) == nil {
				t.Fatal("changed witness accepted")
			}
		})
	}
}

func TestOpenRGBNativeTransportWitnessRequiresSortedUniqueRoutes(t *testing.T) {
	_, m, r, w, _ := nativeTransportFixture(t)
	second := m.NativePolicy.Devices[0]
	second.StableID = "IOService:/usb/port9/interface0"
	m.NativePolicy.Devices = append(m.NativePolicy.Devices, second)
	binding, err := freezeOpenRGBNativeExecPolicy(m.NativePolicy, m.Contract.Artifact.SHA256, m.Catalog, m.Contract.ConfigBytes)
	if err != nil {
		t.Fatal(err)
	}
	m.Policy = binding
	w.Policy = binding
	w.SelectedNodes = []string{"IOService:/usb/port8/interface0", "IOService:/usb/port9/interface0"}
	if e := validateOpenRGBNativeTransportWitness(m, r, w); e != nil {
		t.Fatal("sorted bounded routes refused", e)
	}
	for _, nodes := range [][]string{{w.SelectedNodes[1], w.SelectedNodes[0]}, {w.SelectedNodes[0], w.SelectedNodes[0]}} {
		bad := w
		bad.SelectedNodes = nodes
		if validateOpenRGBNativeTransportWitness(m, r, bad) == nil {
			t.Fatal("duplicate or unsorted routes admitted", nodes)
		}
	}
}

func TestOpenRGBNativeTransportStrictFrames(t *testing.T) {
	for _, body := range []string{`{"operation":"prepare","Operation":"claim"}`, `{"Operation":"prepare"}`, `{"operation":"claim","ref":{"root_pid":123,"Root_PID":456}}`, `{"operation":"claim","witness":{"policy":{"id":"a","ID":"b"}}}`, `{"operation":"prepare","unknown":1}`, `{"operation":"prepare","operation":"claim"}`, `{"operation":"claim","ref":{"root_pid":123,"root_pid":456}}`, `{"operation":"prepare"} {}`, `{"operation":"prepare"}null`, ""} {
		t.Run(body, func(t *testing.T) {
			a, b := net.Pipe()
			defer a.Close()
			defer b.Close()
			go func() {
				var h [4]byte
				binary.BigEndian.PutUint32(h[:], uint32(len(body)))
				_ = writeOpenRGBNativeBytes(a, append(h[:], body...))
			}()
			var f openRGBNativeClaimFrame
			if readOpenRGBNativeFrame(b, &f) == nil {
				t.Fatal("malformed frame accepted")
			}
		})
	}
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()
	go func() { var h [4]byte; binary.BigEndian.PutUint32(h[:], openRGBNativeFrameLimit+1); a.Write(h[:]) }()
	var f openRGBNativeClaimFrame
	if readOpenRGBNativeFrame(b, &f) == nil {
		t.Fatal("oversized frame accepted")
	}
}

func TestOpenRGBNativeTransportRejectsClaimBeforePrepareAndCancellation(t *testing.T) {
	c, _, r, w, cb := nativeTransportFixture(t)
	s := nativeTransportServer(c, cb)
	defer s.Close()
	cl := nativeTransportPipe(t, s, c)
	if cl.Claim(context.Background(), r, w) == nil {
		t.Fatal("claim before prepare accepted")
	}
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()
	ctx, cancel := context.WithCancel(context.Background())
	stop := openRGBNativeDeadline(ctx, a)
	defer stop()
	cancel()
	var f openRGBNativeClaimFrame
	done := make(chan error, 1)
	go func() { done <- readOpenRGBNativeFrame(a, &f) }()
	select {
	case e := <-done:
		if e == nil {
			t.Fatal("canceled exchange succeeded")
		}
	case <-time.After(time.Second):
		t.Fatal("canceled exchange leaked")
	}
}

type nativeTransportFixtureListener struct {
	incoming chan net.Conn
	done     chan struct{}
	once     sync.Once
}

func (l *nativeTransportFixtureListener) Accept() (net.Conn, error) {
	select {
	case c := <-l.incoming:
		return c, nil
	case <-l.done:
		return nil, net.ErrClosed
	}
}
func (l *nativeTransportFixtureListener) Close() error {
	l.once.Do(func() { close(l.done) })
	return nil
}
func (l *nativeTransportFixtureListener) Addr() net.Addr {
	return &net.UnixAddr{Name: "fixture", Net: "unix"}
}

func TestOpenRGBNativeTransportConnectionLimitAndClose(t *testing.T) {
	c, _, _, _, cb := nativeTransportFixture(t)
	s := nativeTransportServer(c, cb)
	l := &nativeTransportFixtureListener{incoming: make(chan net.Conn), done: make(chan struct{})}
	s.listener = l
	s.wg.Add(1)
	go s.accept()
	defer s.Close()
	var clients []net.Conn
	for i := 0; i < 17; i++ {
		a, b := net.Pipe()
		clients = append(clients, b)
		l.incoming <- a
	}
	defer func() {
		for _, c := range clients {
			c.Close()
		}
	}()
	_ = clients[16].SetReadDeadline(time.Now().Add(time.Second))
	var b [1]byte
	if _, e := clients[16].Read(b[:]); !errors.Is(e, io.EOF) {
		t.Fatalf("seventeenth connection was not refused: %v", e)
	}
	if e := s.Close(); e != nil {
		t.Fatal(e)
	}
	if e := s.Close(); e != nil {
		t.Fatal("Close not idempotent", e)
	}
	s.mu.Lock()
	remaining := len(s.conns)
	s.mu.Unlock()
	if remaining != 0 {
		t.Fatal("accepted workers leaked", remaining)
	}
}

func TestOpenRGBNativeTransportCloseRestoresPeerGrantOnce(t *testing.T) {
	c, _, _, _, cb := nativeTransportFixture(t)
	s := nativeTransportServer(c, cb)
	calls := 0
	sentinel := errors.New("external ACL changed; refused overwrite")
	s.releasePeer = func() error { calls++; return sentinel }
	if !errors.Is(s.Close(), sentinel) || !errors.Is(s.Close(), sentinel) {
		t.Fatal("peer grant cleanup error lost")
	}
	if calls != 1 {
		t.Fatalf("peer grant released %d times", calls)
	}
}

func TestOpenRGBNativeTransportCleanupPreservesSuccessor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "socket-fixture")
	if e := os.WriteFile(path, []byte("old"), 0600); e != nil {
		t.Fatal(e)
	}
	prior, e := os.Lstat(path)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.Rename(path, path+".retired"); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(path, []byte("new"), 0600); e != nil {
		t.Fatal(e)
	}
	if e = removeOpenRGBNativeOwnedSocket(path, prior); e == nil {
		t.Fatal("successor endpoint cleanup accepted")
	}
	if _, e = os.Stat(path); e != nil {
		t.Fatal("successor removed", e)
	}
	current, e := os.Lstat(path)
	if e != nil {
		t.Fatal(e)
	}
	if e = removeOpenRGBNativeOwnedSocket(path, current); e != nil {
		t.Fatal(e)
	}
}

func TestOpenRGBNativeTransportIdleForeignPeersCannotConsumeSlots(t *testing.T) {
	c, _, _, _, cb := nativeTransportFixture(t)
	s := nativeTransportServer(c, cb)
	legitServer, legitClient := net.Pipe()
	defer legitClient.Close()
	s.ops.peer = func(conn net.Conn) (int, string, error) {
		if conn == legitServer {
			return 123, "unix:501", nil
		}
		return 999, "unix:999", nil
	}
	listener := &nativeTransportFixtureListener{incoming: make(chan net.Conn), done: make(chan struct{})}
	s.listener = listener
	s.wg.Add(1)
	go s.accept()
	defer s.Close()
	for i := 0; i < 20; i++ {
		a, b := net.Pipe()
		listener.incoming <- a
		_ = b.SetReadDeadline(time.Now().Add(time.Second))
		var one [1]byte
		if _, e := b.Read(one[:]); !errors.Is(e, io.EOF) {
			b.Close()
			t.Fatalf("idle foreign peer retained before any frame: %v", e)
		}
		b.Close()
	}
	listener.incoming <- legitServer
	client := &openRGBNativeClaimClient{conn: legitClient, capsule: c, ops: openRGBNativeTransportOps{peer: func(net.Conn) (int, string, error) { return c.OwnerPID, c.OwnerPrincipal, nil }, incarnation: func(int) (string, error) { return c.OwnerIncarnation, nil }}}
	if _, _, e := client.Prepare(context.Background()); e != nil {
		t.Fatalf("admitted peer capacity consumed by idle foreign peers: %v", e)
	}
}
