//go:build linux

package localworkbridge

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

// SourceReleaseBoundary holds the SAME serialization as source revocation and
// launch invalidation during fn. It must call fn once, synchronously, with a
// context bounded by ctx and any earlier authority expiry. It must permit fresh
// reads by SourcePermit.Recheck (no recursive non-reentrant lock acquisition).
// This is a trusted owner contract, NOT a worker attestation or isolation proof.
// No production implementation is installed.
type SourceReleaseBoundary interface {
	WithinSourceRelease(context.Context, SourceBrokerScope, func(context.Context) error) error
}

// SourceTransportConfig accepts only launcher-owned anonymous pipe ends. On
// successful enabled construction the transport owns these ends exclusively.
// It never opens a path, listener, socket, process, or credential channel.
// The lifecycle owner must prove the worker's descriptor allowlist, confinement,
// immutable image, and stop/join before supplying this unregistered component.
type SourceTransportConfig struct {
	Enabled             bool
	Broker              *SourceBroker
	Requests, Responses *os.File
	Boundary            SourceReleaseBoundary
	MaxOperations       int
	Lifetime, IOTimeout time.Duration
}

type SourceTransport struct {
	enabled           bool
	cfg               SourceTransportConfig
	mu                sync.Mutex
	started, stopping bool
	cancel            context.CancelFunc
	done              chan struct{}
	closeOnce         sync.Once
}

func NewSourceTransport(c SourceTransportConfig) (*SourceTransport, error) {
	t := &SourceTransport{enabled: c.Enabled}
	if !c.Enabled {
		return t, nil
	}
	if c.Broker == nil || !c.Broker.enabled || c.Boundary == nil || c.MaxOperations < 1 || c.MaxOperations > 64 || c.Lifetime <= 0 || c.Lifetime > 180*time.Second || c.IOTimeout <= 0 || c.IOTimeout > 5*time.Second || !sourcePipe(c.Requests, unix.O_RDONLY) || !sourcePipe(c.Responses, unix.O_WRONLY) {
		return nil, ErrSourceAuthority
	}
	a, err := c.Requests.Stat()
	if err != nil {
		return nil, ErrSourceAuthority
	}
	b, err := c.Responses.Stat()
	if err != nil || os.SameFile(a, b) {
		return nil, ErrSourceAuthority
	}
	// Deadline support is mandatory; a blocking descriptor cannot be owned safely.
	if c.Requests.SetReadDeadline(time.Time{}) != nil || c.Responses.SetWriteDeadline(time.Time{}) != nil {
		return nil, ErrSourceAuthority
	}
	t.cfg = c
	t.done = make(chan struct{})
	return t, nil
}

func sourcePipe(f *os.File, mode int) bool {
	if f == nil {
		return false
	}
	st, err := f.Stat()
	if err != nil || st.Mode()&os.ModeNamedPipe == 0 {
		return false
	}
	raw, err := f.SyscallConn()
	if err != nil {
		return false
	}
	valid := false
	err = raw.Control(func(fd uintptr) {
		name, err := os.Readlink(fmt.Sprintf("/proc/self/fd/%d", fd))
		if err != nil || !strings.HasPrefix(name, "pipe:[") || !strings.HasSuffix(name, "]") {
			return
		}
		flags, err := unix.FcntlInt(fd, unix.F_GETFL, 0)
		valid = err == nil && flags&unix.O_ACCMODE == mode
	})
	return err == nil && valid
}

func (t *SourceTransport) closePipes() {
	t.closeOnce.Do(func() { t.cfg.Requests.Close(); t.cfg.Responses.Close() })
}

// Serve is one finite lifetime, never a listener or a restart loop. A protocol,
// authority, timeout or partial-write failure closes the channel. There is no
// automatic retry, success/error frame on failure, or new admission identity.
// Stop closes blocked IO but still waits for cooperative owner work to return.
func (t *SourceTransport) Serve(ctx context.Context) error {
	if t == nil || !t.enabled {
		return ErrDisabled
	}
	if ctx == nil {
		return ErrSourceAuthority
	}
	t.mu.Lock()
	if t.started || t.stopping {
		t.mu.Unlock()
		return ErrStopped
	}
	t.started = true
	ctx, t.cancel = context.WithTimeout(ctx, t.cfg.Lifetime)
	t.mu.Unlock()
	watcher := make(chan struct{})
	go func() { defer close(watcher); <-ctx.Done(); t.closePipes() }()
	defer func() { t.cancel(); t.closePipes(); <-watcher; close(t.done) }()
	for i := 0; i < t.cfg.MaxOperations; i++ {
		if ctx.Err() != nil {
			return ErrSourceAuthority
		}
		if t.cfg.Requests.SetReadDeadline(sourceIODeadline(ctx, t.cfg.IOTimeout)) != nil {
			return ErrSourceAuthority
		}
		var header [4]byte
		n, err := io.ReadFull(t.cfg.Requests, header[:])
		if err == io.EOF && n == 0 && ctx.Err() == nil {
			return nil
		}
		if err != nil {
			return ErrSourceAuthority
		}
		size := binary.BigEndian.Uint32(header[:])
		if size == 0 || size > 4096 {
			return ErrSourceAuthority
		}
		raw := make([]byte, size)
		if _, err = io.ReadFull(t.cfg.Requests, raw); err != nil {
			return ErrSourceAuthority
		}
		_, err = t.cfg.Broker.execute(ctx, raw, t.release)
		clear(raw)
		if err != nil {
			return ErrSourceAuthority
		}
	}
	return nil
}

func sourceIODeadline(ctx context.Context, timeout time.Duration) time.Time {
	until := time.Now().Add(timeout)
	if deadline, ok := ctx.Deadline(); ok && deadline.Before(until) {
		return deadline
	}
	return until
}

func (t *SourceTransport) release(ctx context.Context, p *SourcePermit, raw []byte) error {
	if len(raw) == 0 || len(raw) > sourceBrokerResponseLimit {
		return ErrSourceAuthority
	}
	// Establish a write budget BEFORE the boundary can block. The boundary must
	// clip it further to current authority expiry; every chunk rechecks authority.
	ctx, cancel := context.WithTimeout(ctx, t.cfg.IOTimeout)
	defer cancel()
	calls := 0
	var callbackErr error
	err := t.cfg.Boundary.WithinSourceRelease(ctx, t.cfg.Broker.scope, func(inner context.Context) (err error) {
		defer func() {
			if err != nil {
				callbackErr = ErrSourceAuthority
			}
		}()
		calls++
		if calls != 1 || inner == nil || ctx.Err() != nil {
			return ErrSourceAuthority
		}
		// Never trust a boundary to lengthen or replace the caller's cancellation.
		inner, stop := context.WithCancel(inner)
		defer stop()
		cancelInner := sourceAfterFunc(ctx, stop)
		defer cancelInner()
		interruptIO := sourceAfterFunc(inner, t.closePipes)
		defer interruptIO()
		deadline := sourceIODeadline(ctx, t.cfg.IOTimeout)
		if d, ok := inner.Deadline(); ok && d.Before(deadline) {
			deadline = d
		}
		if t.cfg.Responses.SetWriteDeadline(deadline) != nil {
			return ErrSourceAuthority
		}
		frame := make([]byte, 4+len(raw))
		defer clear(frame)
		binary.BigEndian.PutUint32(frame[:4], uint32(len(raw)))
		copy(frame[4:], raw)
		for len(frame) > 0 {
			if ctx.Err() != nil || inner.Err() != nil || !time.Now().Before(deadline) || p.Recheck(inner) != nil {
				return ErrSourceAuthority
			}
			// Bound each kernel write. Partial frames are terminal and must never be
			// interpreted by a worker as a completed response or safe fresh admission.
			end := min(len(frame), 4096)
			n, err := t.cfg.Responses.Write(frame[:end])
			if err != nil || n != end {
				return ErrSourceAuthority
			}
			frame = frame[n:]
		}
		if ctx.Err() != nil || inner.Err() != nil || p.Recheck(inner) != nil {
			return ErrSourceAuthority
		}
		return nil
	})
	if err != nil || callbackErr != nil || calls != 1 || ctx.Err() != nil {
		return ErrSourceAuthority
	}
	return nil
}

// sourceAfterFunc returns a stop-and-join function, not just a stop request.
// AfterFunc's stop alone does not wait for an already-started callback.
func sourceAfterFunc(ctx context.Context, fn func()) func() {
	done := make(chan struct{})
	stop := context.AfterFunc(ctx, func() { defer close(done); fn() })
	return func() {
		if !stop() {
			<-done
		}
	}
}

// Stop may time out while an uncooperative trusted owner remains active. Such a
// timeout is NOT a stopped/joined proof and never authorizes replacement work.
func (t *SourceTransport) Stop(ctx context.Context) error {
	if t == nil || !t.enabled {
		return nil
	}
	if ctx == nil {
		return ErrSourceAuthority
	}
	t.mu.Lock()
	t.stopping = true
	if !t.started {
		t.started = true
		t.closePipes()
		close(t.done)
	}
	if t.cancel != nil {
		t.cancel()
	}
	t.mu.Unlock()
	t.closePipes()
	select {
	case <-t.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
