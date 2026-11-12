package resources

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/localprincipal"
	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	processowner "github.com/vrooli/vrooli/internal/process"
)

const openRGBNativeFrameLimit = 8 << 20

var errOpenRGBNativeRefused = errors.New("native OpenRGB owner refused exchange")

// The seam is per instance: tests never replace process-wide authentication.
type openRGBNativeTransportOps struct {
	peer        func(net.Conn) (int, string, error)
	incarnation func(int) (string, error)
}

func openRGBNativeProductionOps() openRGBNativeTransportOps {
	return openRGBNativeTransportOps{incarnation: processowner.Incarnation, peer: func(c net.Conn) (int, string, error) {
		u, ok := c.(*net.UnixConn)
		if !ok {
			return 0, "", errOpenRGBNativeRefused
		}
		p, e := localprincipal.Peer(u)
		if e != nil {
			return 0, "", e
		}
		pid, e := localprincipal.PeerPID(u)
		return int(pid), p.String(), e
	}}
}

type openRGBNativeClaimFrame struct {
	Operation string                 `json:"operation"`
	Ref       *platform.ServiceRef   `json:"ref,omitempty"`
	Witness   *openRGBExecWitness    `json:"witness,omitempty"`
	Material  *openRGBNativeMaterial `json:"material,omitempty"`
	OK        bool                   `json:"ok,omitempty"`
}

func readOpenRGBNativeFrame(c net.Conn, out *openRGBNativeClaimFrame) error {
	var h [4]byte
	if _, e := io.ReadFull(c, h[:]); e != nil {
		return e
	}
	n := binary.BigEndian.Uint32(h[:])
	if n == 0 || n > openRGBNativeFrameLimit {
		return errOpenRGBNativeRefused
	}
	b := make([]byte, int(n))
	if _, e := io.ReadFull(c, b); e != nil {
		return e
	}
	if e := validateManagedUpdateJSON(b); e != nil {
		return e
	}
	if e := validateOpenRGBNativeFrameFields(b, out); e != nil {
		return e
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if e := d.Decode(out); e != nil {
		return e
	}
	var trailing any
	if d.Decode(&trailing) != io.EOF {
		return errOpenRGBNativeRefused
	}
	return nil
}
func writeOpenRGBNativeFrame(c net.Conn, f openRGBNativeClaimFrame) error {
	b, e := json.Marshal(f)
	if e != nil {
		return e
	}
	if len(b) > openRGBNativeFrameLimit {
		return errOpenRGBNativeRefused
	}
	var h [4]byte
	binary.BigEndian.PutUint32(h[:], uint32(len(b)))
	return writeOpenRGBNativeBytes(c, append(h[:], b...))
}
func writeOpenRGBNativeBytes(c net.Conn, b []byte) error {
	for len(b) > 0 {
		n, e := c.Write(b)
		if e != nil {
			return e
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		b = b[n:]
	}
	return nil
}

// Every exchange has a bounded deadline even when the caller has none.
func openRGBNativeDeadline(ctx context.Context, c net.Conn) func() {
	end := time.Now().Add(30 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(end) {
		end = d
	}
	_ = c.SetDeadline(end)
	stop := context.AfterFunc(ctx, func() { _ = c.Close() })
	return func() { stop() }
}

type openRGBNativeClaimServer struct {
	capsule       openRGBNativeCapsule
	callbacks     openRGBNativeClaimCallbacks
	ops           openRGBNativeTransportOps
	listener      net.Listener
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.Mutex
	admission     sync.Mutex
	conns         map[net.Conn]struct{}
	claimed       bool
	closed        bool
	wg            sync.WaitGroup
	closeOnce     sync.Once
	closeErr      error
	releasePeer   func() error
	endpointInfo  os.FileInfo
	peerOnce      sync.Once
	peerPrincipal string
	peerErr       error
}

func startOpenRGBNativeClaimServer(ctx context.Context, c openRGBNativeCapsule, cb openRGBNativeClaimCallbacks) (*openRGBNativeClaimServer, error) {
	if ctx == nil || c.validate() != nil || ctx.Err() != nil {
		return nil, errOpenRGBNativeRefused
	}
	if !openRGBNativeCallbacksComplete(cb) {
		return nil, errOpenRGBNativeRefused
	}
	if err := verifyOpenRGBNativeClaimOwner(ctx, c, openRGBNativeClaimOwnerOps{
		pid: os.Getpid, principal: func() (string, error) {
			p, err := localprincipal.Current()
			if err != nil {
				return "", err
			}
			return p.String(), nil
		},
		incarnation: processowner.Incarnation, security: platform.ProcessSecurity, account: platform.ResolveOwnedServiceAccount,
	}); err != nil {
		return nil, err
	}
	var e error
	if e = prepareOpenRGBNativeIPCRoot(ctx); e != nil {
		return nil, e
	}
	peerCleanup, e := platform.PrepareOwnedServicePeerObservation(ctx, c.Account)
	if e != nil {
		return nil, e
	}
	// Never unlink a prior owner's endpoint. Permissions are provisioned by the parent.
	l, e := net.ListenUnix("unix", &net.UnixAddr{Name: c.Endpoint, Net: "unix"})
	if e != nil {
		return nil, errors.Join(e, peerCleanup())
	}
	l.SetUnlinkOnClose(false)
	info, configureErr := configureOpenRGBNativeClaimSocket(ctx, c)
	if configureErr != nil {
		return nil, errors.Join(configureErr, l.Close(), removeOpenRGBNativeOwnedSocket(c.Endpoint, info), peerCleanup())
	}
	s := newOpenRGBNativeClaimServer(ctx, c, cb, openRGBNativeProductionOps())
	s.listener = l
	s.releasePeer = peerCleanup
	s.endpointInfo = info
	s.wg.Add(1)
	go s.accept()
	go func() { <-s.ctx.Done(); _ = s.Close() }()
	return s, nil
}
func openRGBNativeCallbacksComplete(c openRGBNativeClaimCallbacks) bool {
	return c.Lookup != nil && c.Inspect != nil && c.Incarnation != nil && c.ResolveAccount != nil && c.Prepare != nil && c.Claim != nil
}
func newOpenRGBNativeClaimServer(ctx context.Context, c openRGBNativeCapsule, cb openRGBNativeClaimCallbacks, ops openRGBNativeTransportOps) *openRGBNativeClaimServer {
	ctx, cancel := context.WithCancel(ctx)
	return &openRGBNativeClaimServer{capsule: c, callbacks: cb, ops: ops, ctx: ctx, cancel: cancel, conns: make(map[net.Conn]struct{})}
}
func (s *openRGBNativeClaimServer) accept() {
	defer s.wg.Done()
	for {
		c, e := s.listener.Accept()
		if e != nil {
			return
		}
		if s.preAuthenticate(c) != nil {
			_ = c.Close()
			continue
		}
		s.mu.Lock()
		if s.closed || len(s.conns) >= 16 {
			s.mu.Unlock()
			_ = c.Close()
			continue
		}
		s.conns[c] = struct{}{}
		s.wg.Add(1)
		s.mu.Unlock()
		go func() {
			defer s.wg.Done()
			defer c.Close()
			defer func() { s.mu.Lock(); delete(s.conns, c); s.mu.Unlock() }()
			s.serve(c)
		}()
	}
}
func (s *openRGBNativeClaimServer) Close() error {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.closed = true
		s.cancel()
		if s.listener != nil {
			s.closeErr = s.listener.Close()
		}
		for c := range s.conns {
			_ = c.Close()
		}
		s.mu.Unlock()
		s.wg.Wait()
		s.closeErr = errors.Join(s.closeErr, removeOpenRGBNativeOwnedSocket(s.capsule.Endpoint, s.endpointInfo))
		if s.releasePeer != nil {
			s.closeErr = errors.Join(s.closeErr, s.releasePeer())
		}
	})
	return s.closeErr
}
func openRGBNativeAccountPrincipal(p string) string {
	if strings.HasPrefix(p, "uid:") {
		return "unix:" + strings.TrimPrefix(p, "uid:")
	}
	if strings.HasPrefix(p, "sid:S-") {
		return "windows:" + strings.TrimPrefix(p, "sid:")
	}
	return ""
}
func (s *openRGBNativeClaimServer) observe(ctx context.Context, c net.Conn) (platform.ServiceRef, error) {
	var zero platform.ServiceRef
	pid, p, e := s.ops.peer(c)
	if e != nil || pid <= 0 {
		return zero, errOpenRGBNativeRefused
	}
	a, e := s.callbacks.ResolveAccount(ctx, s.capsule.Account)
	if e != nil || a.Name != s.capsule.Account || openRGBNativeAccountPrincipal(a.Principal) == "" || openRGBNativeAccountPrincipal(a.Principal) != p {
		return zero, errOpenRGBNativeRefused
	}
	r, e := s.callbacks.Lookup(ctx, s.capsule.Intent)
	if e != nil || platform.ServiceIntent(r.Intent) != s.capsule.Intent || r.Native == nil || !r.NativeOwnedIdentityComplete() || r.RootPID != pid {
		return zero, errOpenRGBNativeRefused
	}
	inc, e := s.callbacks.Incarnation(pid)
	if e != nil || !r.MatchesIncarnation(inc) {
		return zero, errOpenRGBNativeRefused
	}
	o, e := s.callbacks.Inspect(ctx, r)
	if e != nil || !o.Ref.Equal(platform.OperationRef(r)) || o.State != platform.OperationRunning || !o.Populated || !o.RootAlive {
		return zero, errOpenRGBNativeRefused
	}
	confirmed, e := s.callbacks.Lookup(ctx, s.capsule.Intent)
	if e != nil || !confirmed.Equal(r) {
		return zero, errOpenRGBNativeRefused
	}
	inc, e = s.callbacks.Incarnation(pid)
	if e != nil || !r.MatchesIncarnation(inc) {
		return zero, errOpenRGBNativeRefused
	}
	return r, nil
}
func (s *openRGBNativeClaimServer) prepare(ctx context.Context, c net.Conn, ref platform.ServiceRef) (openRGBNativeMaterial, error) {
	// Serialize disclosure against claim admission: a consumed claim can never
	// disclose its retained secret material to a later connection.
	s.admission.Lock()
	defer s.admission.Unlock()
	if s.claimed || ctx.Err() != nil {
		return openRGBNativeMaterial{}, errOpenRGBNativeRefused
	}
	m, e := s.callbacks.Prepare(ctx, ref)
	if e != nil || m.validate(s.capsule) != nil {
		return openRGBNativeMaterial{}, errOpenRGBNativeRefused
	}
	confirmed, e := s.observe(ctx, c)
	if e != nil || !confirmed.Equal(ref) {
		return openRGBNativeMaterial{}, errOpenRGBNativeRefused
	}
	if e = writeOpenRGBNativeFrame(c, openRGBNativeClaimFrame{Operation: "prepared", Material: &m, Ref: &ref, OK: true}); e != nil {
		return openRGBNativeMaterial{}, e
	}
	return m, nil
}
func (s *openRGBNativeClaimServer) serve(c net.Conn) {
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()
	stop := openRGBNativeDeadline(ctx, c)
	defer stop()
	if s.preAuthenticate(c) != nil {
		return
	}
	refuse := func() { _ = writeOpenRGBNativeFrame(c, openRGBNativeClaimFrame{Operation: "refused"}) }
	var request openRGBNativeClaimFrame
	if readOpenRGBNativeFrame(c, &request) != nil || request.Operation != "prepare" || request.Ref != nil || request.Witness != nil || request.Material != nil || request.OK {
		refuse()
		return
	}
	ref, e := s.observe(ctx, c)
	if e != nil {
		refuse()
		return
	}
	m, e := s.prepare(ctx, c, ref)
	if e != nil {
		refuse()
		return
	}
	request = openRGBNativeClaimFrame{}
	if readOpenRGBNativeFrame(c, &request) != nil || request.Operation != "claim" || request.Ref == nil || !request.Ref.Equal(ref) || request.Witness == nil || request.Material != nil || request.OK || validateOpenRGBNativeTransportWitness(m, ref, *request.Witness) != nil {
		refuse()
		return
	}
	confirmed, e := s.observe(ctx, c)
	if e != nil || !confirmed.Equal(ref) {
		refuse()
		return
	}
	s.admission.Lock()
	if s.claimed || ctx.Err() != nil {
		s.admission.Unlock()
		refuse()
		return
	}
	s.claimed = true
	s.admission.Unlock()
	// The durable fence callback owns persistence. Uncertain responses never permit retry.
	if s.callbacks.Claim(ctx, ref, *request.Witness) != nil {
		refuse()
		return
	}
	_ = writeOpenRGBNativeFrame(c, openRGBNativeClaimFrame{Operation: "claimed", OK: true})
}
func validateOpenRGBNativeTransportWitness(m openRGBNativeMaterial, r platform.ServiceRef, w openRGBExecWitness) error {
	expected, err := hostreqkit.NativeOpenRGBTransportPolicySHA256(m.Policy.Platform)
	if err != nil || w.TransportSHA256 != expected {
		return errOpenRGBNativeRefused
	}

	if w.Policy != m.Policy || w.ClaimSHA256 != m.ClaimSHA256 || !r.MatchesIncarnation(w.Incarnation) || r.Native == nil || r.Native.Platform != m.Policy.Platform || !updateDigest(w.ContextSHA256) || !updateDigest(w.DeviceGenerationSHA256) || len(w.SelectedNodes) == 0 || len(w.SelectedNodes) > 16 || len(w.SelectedNodes) != len(m.NativePolicy.Devices) {
		return errOpenRGBNativeRefused
	}
	for i, n := range w.SelectedNodes {
		if n == "" || len(n) > 4096 || strings.ContainsAny(n, "\x00\r\n") || i > 0 && w.SelectedNodes[i-1] >= n {
			return errOpenRGBNativeRefused
		}
	}
	if m.Policy.Platform == openRGBPlatformWindows {
		if w.ExecutionChild == nil || w.ExecutionChild.validateFor(r) != nil {
			return errOpenRGBNativeRefused
		}
	} else if w.ExecutionChild != nil {
		return errOpenRGBNativeRefused
	}
	return nil
}

type openRGBNativeClaimClient struct {
	conn      net.Conn
	capsule   openRGBNativeCapsule
	ops       openRGBNativeTransportOps
	mu        sync.Mutex
	prepared  bool
	attempted bool
	material  openRGBNativeMaterial
	ref       platform.ServiceRef
}

func dialOpenRGBNativeClaimOwner(ctx context.Context, c openRGBNativeCapsule) (*openRGBNativeClaimClient, error) {
	if ctx == nil || c.validate() != nil {
		return nil, errOpenRGBNativeRefused
	}
	conn, e := (&net.Dialer{}).DialContext(ctx, "unix", c.Endpoint)
	if e != nil {
		return nil, e
	}
	cl := &openRGBNativeClaimClient{conn: conn, capsule: c, ops: openRGBNativeProductionOps()}
	if e = cl.owner(); e != nil {
		conn.Close()
		return nil, e
	}
	return cl, nil
}
func (c *openRGBNativeClaimClient) owner() error {
	pid, p, e := c.ops.peer(c.conn)
	if e != nil || pid != c.capsule.OwnerPID || p != c.capsule.OwnerPrincipal {
		return errOpenRGBNativeRefused
	}
	inc, e := c.ops.incarnation(pid)
	if e != nil || inc != c.capsule.OwnerIncarnation {
		return errOpenRGBNativeRefused
	}
	return nil
}
func (c *openRGBNativeClaimClient) Prepare(ctx context.Context) (openRGBNativeMaterial, platform.ServiceRef, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var m openRGBNativeMaterial
	var r platform.ServiceRef
	if c.prepared || c.attempted || ctx == nil {
		return m, r, errOpenRGBNativeRefused
	}
	c.prepared = true
	stop := openRGBNativeDeadline(ctx, c.conn)
	defer stop()
	if e := c.owner(); e != nil {
		return m, r, e
	}
	if e := writeOpenRGBNativeFrame(c.conn, openRGBNativeClaimFrame{Operation: "prepare"}); e != nil {
		return m, r, e
	}
	var f openRGBNativeClaimFrame
	if e := readOpenRGBNativeFrame(c.conn, &f); e != nil {
		return m, r, e
	}
	if c.owner() != nil || f.Operation != "prepared" || !f.OK || f.Material == nil || f.Ref == nil || f.Witness != nil || f.Material.validate(c.capsule) != nil || !f.Ref.NativeOwnedIdentityComplete() || f.Ref.Native == nil || platform.ServiceIntent(f.Ref.Intent) != c.capsule.Intent {
		return m, r, errOpenRGBNativeRefused
	}
	c.material = *f.Material
	c.ref = *f.Ref
	return c.material, c.ref, nil
}
func (c *openRGBNativeClaimClient) Claim(ctx context.Context, r platform.ServiceRef, w openRGBExecWitness) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.attempted {
		return errOpenRGBNativeRefused
	}
	c.attempted = true
	if ctx == nil || !c.prepared || !r.Equal(c.ref) || validateOpenRGBNativeTransportWitness(c.material, r, w) != nil {
		return errOpenRGBNativeRefused
	}
	stop := openRGBNativeDeadline(ctx, c.conn)
	defer stop()
	if e := c.owner(); e != nil {
		return e
	}
	if e := writeOpenRGBNativeFrame(c.conn, openRGBNativeClaimFrame{Operation: "claim", Ref: &r, Witness: &w}); e != nil {
		return e
	}
	var f openRGBNativeClaimFrame
	if e := readOpenRGBNativeFrame(c.conn, &f); e != nil {
		return e
	}
	if c.owner() != nil || f.Operation != "claimed" || !f.OK || f.Ref != nil || f.Material != nil || f.Witness != nil {
		return errOpenRGBNativeRefused
	}
	return nil
}
func (c *openRGBNativeClaimClient) Close() error { return c.conn.Close() }

func removeOpenRGBNativeOwnedSocket(path string, expected os.FileInfo) error {
	if expected == nil {
		return nil
	}
	actual, e := os.Lstat(path)
	if errors.Is(e, os.ErrNotExist) {
		return nil
	}
	if e != nil {
		return e
	}
	if !os.SameFile(expected, actual) {
		return errors.New("native OpenRGB endpoint changed; refusing successor unlink")
	}
	return os.Remove(path)
}

// Resolve the fixed local account once, then reject foreign kernel principals
// before reading a frame or consuming an authenticated connection slot.
func (s *openRGBNativeClaimServer) preAuthenticate(c net.Conn) error {
	s.peerOnce.Do(func() {
		a, e := s.callbacks.ResolveAccount(s.ctx, s.capsule.Account)
		if e != nil || a.Name != s.capsule.Account {
			s.peerErr = errOpenRGBNativeRefused
			return
		}
		s.peerPrincipal = openRGBNativeAccountPrincipal(a.Principal)
		if s.peerPrincipal == "" {
			s.peerErr = errOpenRGBNativeRefused
		}
	})
	if s.peerErr != nil {
		return s.peerErr
	}
	pid, principal, e := s.ops.peer(c)
	if e != nil || pid <= 0 || principal != s.peerPrincipal {
		return errOpenRGBNativeRefused
	}
	return s.ctx.Err()
}
