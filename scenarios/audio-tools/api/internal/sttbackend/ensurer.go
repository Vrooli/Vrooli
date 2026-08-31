// Package sttbackend provides on-demand recovery of a down STT backend resource
// (plan L1 — the 2026-06-29 incident fix). audio-tools is the only consumer of
// its local STT backends, so when whisper (and its :8090 activity-edge companion)
// is torn down, a restart of audio-tools does NOT re-ensure it — the optional
// dependency defaults to skip. The next transcribe then hits connection-refused
// and, before this package, returned the raw transport error with no recovery.
//
// Ensurer closes that gap: on a backend-down transcribe, the STT pipeline asks
// the Ensurer to bring the backing resource up (the sanctioned
// `vrooli resource start <resource>`), then retries the request once. It is:
//
//   - single-flighted per resource — N concurrent transcribes that all hit the
//     down backend trigger at most ONE `resource start`;
//   - cooldown-guarded — repeated failures within a short window return the
//     cached result instead of thrashing the lifecycle;
//   - allowlisted — only resources audio-tools declares as STT backends
//     (whisper, kyutai-stt, sherpa-onnx) are ever started; never an arbitrary name.
//
// It is best-effort and bounded: a missing vrooli binary or a start timeout
// surfaces as an error the caller maps to a typed, user-actionable message — it
// never blocks the request unboundedly.
package sttbackend

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"audio-tools/internal/controlplane"
)

// ErrResourceNotAllowed is returned when EnsureRunning is asked to start a
// resource outside the STT-backend allowlist (defence-in-depth).
var ErrResourceNotAllowed = errors.New("resource is not a declared STT backend")

// allowedResources is the whitelist of backing STT resources this package will
// ever start. Mirrors sttcapacity.allowedResources plus whisper (which IS a
// startable backend here even though its capacity activity is reported edge-side).
var allowedResources = map[string]struct{}{
	"whisper":     {},
	"kyutai-stt":  {},
	"sherpa-onnx": {},
}

// DefaultEnsureTimeout bounds a single `vrooli resource start` so a hung
// lifecycle never blocks a transcribe indefinitely; on timeout the caller maps
// the failure to a typed "backend could not be started" error.
const DefaultEnsureTimeout = 60 * time.Second

// DefaultCooldown suppresses repeated start attempts for a resource within this
// window after an attempt completes — so a backend that is genuinely down (or
// slow to come up) does not get a fresh `resource start` on every queued
// transcribe.
const DefaultCooldown = 20 * time.Second

// Ensurer brings a down STT backend resource back up on demand. The nil Ensurer
// is never used in production; callers gate on a non-nil seam.
type Ensurer interface {
	// EnsureRunning starts the named backend resource if it is not already
	// running, returning nil once the start command succeeds (or was recently
	// attempted successfully). Returns a non-nil error when the resource is not
	// allowlisted or the start failed/timed out.
	EnsureRunning(ctx context.Context, resource string) error
}

// CLIEnsurer is the production Ensurer. It shells the sanctioned
// `vrooli resource start <resource>` (consistent with how sttcapacity shells
// `vrooli capacity …`) so audio-tools never couples to lifecycle internals.
type CLIEnsurer struct {
	controlPlane *controlplane.Client
	// now is the clock seam (tests inject a fake to exercise the cooldown).
	now func() time.Time

	timeout  time.Duration
	cooldown time.Duration

	mu       sync.Mutex
	inflight map[string]*ensureCall
	last     map[string]ensureOutcome
}

type ensureCall struct {
	done chan struct{}
	err  error
}

type ensureOutcome struct {
	at  time.Time
	err error
}

// NewCLIEnsurer resolves the vrooli binary once at startup. The returned ensurer
// is safe even when the binary is missing (EnsureRunning then returns an error
// the caller maps to a typed message).
func NewCLIEnsurer() *CLIEnsurer {
	return &CLIEnsurer{
		controlPlane: controlplane.New(),
		now:          time.Now,
		timeout:      DefaultEnsureTimeout,
		cooldown:     DefaultCooldown,
		inflight:     map[string]*ensureCall{},
		last:         map[string]ensureOutcome{},
	}
}

// EnsureRunning implements Ensurer with single-flight + cooldown + allowlist.
func (e *CLIEnsurer) EnsureRunning(ctx context.Context, resource string) error {
	if _, ok := allowedResources[resource]; !ok {
		return fmt.Errorf("%w: %q", ErrResourceNotAllowed, resource)
	}
	if e.controlPlane == nil || !e.controlPlane.Available() {
		return fmt.Errorf("cannot start %q: vrooli binary not found on PATH", resource)
	}

	e.mu.Lock()
	// Cooldown: within the window after the last attempt, return its cached result
	// rather than re-shelling. A recent SUCCESS means the resource is (being)
	// started; a recent FAILURE means stop hammering the lifecycle.
	if out, ok := e.last[resource]; ok && e.since(out.at) < e.cooldown {
		e.mu.Unlock()
		return out.err
	}
	// Single-flight: join an in-flight start for the same resource.
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

	err := e.runStart(ctx, resource)

	e.mu.Lock()
	call.err = err
	delete(e.inflight, resource)
	e.last[resource] = ensureOutcome{at: e.now(), err: err}
	close(call.done)
	e.mu.Unlock()
	return err
}

// runStart shells `vrooli resource start <resource>` under a bounded timeout that
// is the smaller of the caller's deadline and e.timeout.
func (e *CLIEnsurer) runStart(ctx context.Context, resource string) error {
	cctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()
	if _, err := e.controlPlane.Run(cctx, "resource", "start", resource); err != nil {
		return fmt.Errorf("vrooli resource start %s: %w", resource, err)
	}
	return nil
}

func (e *CLIEnsurer) since(at time.Time) time.Duration {
	now := time.Now
	if e.now != nil {
		now = e.now
	}
	return now().Sub(at)
}
