package ttschain

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"audio-tools/internal/controlplane"
)

// BackendEnsurer is the narrow lifecycle seam used only after a local TTS
// endpoint fails its liveness check.
type BackendEnsurer interface {
	EnsureRunning(context.Context, string) error
}

const (
	ttsStartTimeout  = 60 * time.Second
	ttsStartCooldown = 20 * time.Second
)

// CLIBackendEnsurer starts only resources declared by the auto TTS catalog.
// Starts are single-flight and cooldown guarded per resource.
type CLIBackendEnsurer struct {
	control  *controlplane.Client
	mu       sync.Mutex
	inflight map[string]*ensureCall
	last     map[string]ensureResult
	// startSlot bounds heavyweight model startup across different resources too.
	// Per-resource single-flight alone still permits Kale and Kokoro to load at once.
	startSlot chan struct{}
	allowed   map[string]struct{}
	now       func() time.Time
	start     func(context.Context, string) error
}

type ensureResult struct {
	at  time.Time
	err error
}

type ensureCall struct {
	done chan struct{}
	err  error
}

func NewCLIBackendEnsurer(allowedResources ...string) *CLIBackendEnsurer {
	c := controlplane.New()
	return &CLIBackendEnsurer{
		control: c, allowed: resourceSet(allowedResources), inflight: map[string]*ensureCall{}, last: map[string]ensureResult{}, startSlot: make(chan struct{}, 1), now: time.Now,
		start: func(ctx context.Context, resource string) error {
			_, err := c.Run(ctx, "resource", "start", resource)
			return err
		},
	}
}

// NewBackendEnsurerForTest injects the bounded start operation without a real
// control-plane process.
func NewBackendEnsurerForTest(start func(context.Context, string) error, allowedResources ...string) *CLIBackendEnsurer {
	return &CLIBackendEnsurer{
		control: controlplane.NewForTest("test-vrooli", nil), allowed: resourceSet(allowedResources), start: start,
		inflight: map[string]*ensureCall{}, last: map[string]ensureResult{}, startSlot: make(chan struct{}, 1), now: time.Now,
	}
}

func resourceSet(resources []string) map[string]struct{} {
	set := make(map[string]struct{}, len(resources))
	for _, resource := range resources {
		if resource != "" {
			set[resource] = struct{}{}
		}
	}
	return set
}

func (e *CLIBackendEnsurer) EnsureRunning(ctx context.Context, resource string) error {
	if e == nil {
		return fmt.Errorf("cannot start %q: TTS backend ensurer is not configured", resource)
	}
	if _, allowed := e.allowed[resource]; !allowed {
		return fmt.Errorf("TTS resource %q is not declared", resource)
	}
	if e.control == nil || !e.control.Available() {
		return fmt.Errorf("cannot start %q: vrooli binary not found on PATH", resource)
	}
	now := time.Now
	if e.now != nil {
		now = e.now
	}
	e.mu.Lock()
	if last, ok := e.last[resource]; ok && now().Sub(last.at) < ttsStartCooldown {
		e.mu.Unlock()
		return last.err
	}
	if call, ok := e.inflight[resource]; ok {
		e.mu.Unlock()
		select {
		case <-call.done:
			return call.err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	call := &ensureCall{done: make(chan struct{})}
	e.inflight[resource] = call
	e.mu.Unlock()
	var err error
	attempted := false
	select {
	case e.startSlot <- struct{}{}:
		// If cancellation and the available slot become ready together, select
		// may choose either case. Recheck cancellation before invoking the
		// control plane so canceled requests never start heavyweight runtimes.
		if err = ctx.Err(); err == nil {
			attempted = true
			cctx, cancel := context.WithTimeout(ctx, ttsStartTimeout)
			err = e.start(cctx, resource)
			cancel()
		}
		<-e.startSlot
	case <-ctx.Done():
		err = ctx.Err()
	}
	e.mu.Lock()
	call.err = err
	delete(e.inflight, resource)
	// A request canceled before it reached the start operation did not affect
	// resource state. Do not let that waiter suppress a later healthy request
	// through the per-resource cooldown.
	// Caller cancellation says nothing about backend health. In particular, a
	// canceled single-flight leader must not suppress a fresh request for the
	// full cooldown after it has canceled the resource start.
	if attempted && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		e.last[resource] = ensureResult{at: now(), err: err}
	}
	close(call.done)
	e.mu.Unlock()
	return err
}
