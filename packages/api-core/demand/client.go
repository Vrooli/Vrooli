// Package demand provides the public, transport-neutral client used by
// scenarios that temporarily keep another scenario in demand.
package demand

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// CommandRunner is the narrow seam shared with discovery. It returns combined
// stdout and stderr so callers receive the control-plane diagnostic on error.
type CommandRunner func(context.Context, string, ...string) ([]byte, error)

// Control-plane round trips are bounded so a wedged control plane cannot stall
// a caller indefinitely.
//
// ControlPlaneTimeout must stay above the runtime registry's busy_timeout
// (10s, see buildDSN in internal/scenarioruntime) plus the cost of spawning the
// CLI. That store takes its write lock at BEGIN, so a writer under contention
// waits its turn rather than failing; a caller that gives up sooner than the
// store is willing to wait throws that wait away and turns an orderly queue
// back into a visible failure. Lower this only together with busy_timeout.
//
// Release is best effort and deliberately shorter: the lease TTL is the
// backstop when the control plane will not answer, and a caller should not
// block on cleanup for as long as it would block on acquisition.
const (
	ControlPlaneTimeout        = 15 * time.Second
	ControlPlaneReleaseTimeout = 5 * time.Second
)

const (
	KindExplicit   = "explicit"
	KindCore       = "core"
	KindDependency = "dependency"
	KindProgram    = "program"
	KindJob        = "job"
)

// Lease is the public representation of a runtime demand lease.
type Lease struct {
	LeaseID       string    `json:"lease_id"`
	Scenario      string    `json:"scenario"`
	Variant       string    `json:"variant,omitempty"`
	ConsumerID    string    `json:"consumer_id"`
	Kind          string    `json:"kind"`
	RequestID     string    `json:"request_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	LastRenewedAt time.Time `json:"last_renewed_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	Status        string    `json:"status"`
	StopReason    string    `json:"stop_reason,omitempty"`
	MetadataJSON  string    `json:"metadata_json,omitempty"`
}

type leaseEnvelope struct {
	Lease Lease `json:"lease"`
}

// AcquireRequest identifies one stable consumer hold. LeaseID is required so
// retries refresh the same row rather than creating unbounded history.
type AcquireRequest struct {
	LeaseID    string
	Scenario   string
	Variant    string
	ConsumerID string
	Kind       string
	RequestID  string
	Metadata   string
	TTL        time.Duration
}

// Client calls the control plane through its public CLI surface.
type Client struct {
	VrooliPath string
	Runner     CommandRunner
}

// LeaseClient is implemented by Client and by deterministic scenario test
// doubles. It is intentionally independent of the CLI transport.
type LeaseClient interface {
	Acquire(context.Context, AcquireRequest) (Lease, error)
	Renew(context.Context, string, time.Duration) (Lease, error)
	Release(context.Context, string, string) (Lease, error)
}

// AcquireScoped acquires one lease and returns a bounded cleanup closure. A
// nil client deliberately disables demand for static or injected callers.
// Cleanup uses a fresh context so cancellation of the work does not strand a
// lease until expiry.
func AcquireScoped(ctx context.Context, client LeaseClient, req AcquireRequest, reason string) (Lease, func(), error) {
	if client == nil {
		return Lease{}, nil, nil
	}
	lease, err := client.Acquire(ctx, req)
	if err != nil {
		return Lease{}, nil, err
	}
	var once sync.Once
	return lease, func() {
		once.Do(func() {
			releaseCtx, cancel := context.WithTimeout(context.Background(), ControlPlaneReleaseTimeout)
			defer cancel()
			_, _ = client.Release(releaseCtx, lease.LeaseID, reason)
		})
	}, nil
}

// ErrInvalidLease means the control plane did not establish a live lifetime.
var ErrInvalidLease = errors.New("invalid demand lease lifetime")

// ErrLeaseExpired cancels work when the last granted lifetime elapses.
var ErrLeaseExpired = errors.New("demand lease expired")

func validateLifetimeLease(lease Lease, expectedID string) error {
	if strings.TrimSpace(lease.LeaseID) == "" || (expectedID != "" && lease.LeaseID != expectedID) {
		return fmt.Errorf("%w: lease identity mismatch", ErrInvalidLease)
	}
	if lease.Status != "active" || lease.ExpiresAt.IsZero() || !lease.ExpiresAt.After(time.Now()) {
		return fmt.Errorf("%w: active lease with future expiry required", ErrInvalidLease)
	}
	return nil
}

// Hold owns a renewable lease for a bounded caller lifetime. Work must use
// Context so loss of renewal cancels it. Close waits for bounded release and
// is safe to call more than once, including after parent cancellation.
type Hold struct {
	Lease      Lease
	ctx        context.Context
	cancel     context.CancelCauseFunc
	done       chan struct{}
	releaseErr error
}

func (h *Hold) Context() context.Context { return h.ctx }
func (h *Hold) Close() error {
	h.cancel(context.Canceled)
	<-h.done
	return h.releaseErr
}

// AcquireLifetime renews while ctx is live and releases on cancellation or
// Close. It never silently lets work continue after a failed renewal. Callers
// that need retry may acquire a new hold after stopping the previous work.
func AcquireLifetime(ctx context.Context, client LeaseClient, req AcquireRequest, reason string) (*Hold, error) {
	if client == nil {
		return nil, fmt.Errorf("demand lifetime requires a lease client")
	}
	if req.TTL <= 0 {
		req.TTL = 10 * time.Minute
	}
	acquireCtx, cancelAcquire := context.WithTimeout(ctx, ControlPlaneTimeout)
	lease, err := client.Acquire(acquireCtx, req)
	cancelAcquire()
	if err != nil {
		return nil, err
	}
	if err := validateLifetimeLease(lease, req.LeaseID); err != nil {
		// Only release an identity the caller requested or the acquisition returned;
		// a mismatched explicit identity must never authorize foreign cleanup.
		if lease.LeaseID != "" && (req.LeaseID == "" || lease.LeaseID == req.LeaseID) {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), ControlPlaneReleaseTimeout)
			_, cleanupErr := client.Release(cleanupCtx, lease.LeaseID, reason)
			cancel()
			err = errors.Join(err, cleanupErr)
		}
		return nil, err
	}
	holdCtx, cancel := context.WithCancelCause(ctx)
	hold := &Hold{Lease: lease, ctx: holdCtx, cancel: cancel, done: make(chan struct{})}
	// Expiry cancels work independently of a slow renewal transport. A renewal
	// response arriving after expiry cannot resurrect a cancelled hold.
	expiry := time.AfterFunc(time.Until(lease.ExpiresAt), func() { cancel(ErrLeaseExpired) })
	go func() {
		defer close(hold.done)
		defer expiry.Stop()
		defer func() {
			releaseCtx, cancel := context.WithTimeout(context.Background(), ControlPlaneReleaseTimeout)
			defer cancel()
			_, hold.releaseErr = client.Release(releaseCtx, lease.LeaseID, reason)
		}()
		granted := lease
		interval := func() time.Duration { return max(min(req.TTL/3, time.Until(granted.ExpiresAt)/3), time.Nanosecond) }
		timer := time.NewTimer(interval())
		defer timer.Stop()
		for {
			select {
			case <-holdCtx.Done():
				return
			case <-timer.C:
				renewCtx, cancelRenew := context.WithTimeout(holdCtx, min(ControlPlaneTimeout, time.Until(granted.ExpiresAt)))
				renewed, err := client.Renew(renewCtx, lease.LeaseID, req.TTL)
				cancelRenew()
				if err == nil {
					err = validateLifetimeLease(renewed, lease.LeaseID)
				}
				if err != nil {
					cancel(fmt.Errorf("renew demand lease %s: %w", lease.LeaseID, err))
					return
				}
				if holdCtx.Err() != nil {
					return
				}
				if !expiry.Stop() {
					cancel(ErrLeaseExpired)
					return
				}
				granted = renewed
				expiry.Reset(time.Until(granted.ExpiresAt))
				timer.Reset(interval())
			}
		}
	}()
	return hold, nil
}

// ScenarioStarter starts a scenario while the caller retains its demand lease.
// It is separate from LeaseClient so lease-only callers and test doubles stay
// intentionally small.
type ScenarioStarter interface {
	StartScenario(context.Context, string, string) error
}

func (c Client) runner() CommandRunner {
	if c.Runner != nil {
		return c.Runner
	}
	path := strings.TrimSpace(c.VrooliPath)
	if path == "" {
		path = "vrooli"
	}
	return func(ctx context.Context, _ string, args ...string) ([]byte, error) {
		return exec.CommandContext(ctx, path, args...).CombinedOutput()
	}
}

// StableLeaseID derives a bounded, retry-safe identity when a caller does not
// need to expose its own human-readable ID.
func StableLeaseID(consumerID, scenario, requestID string) string {
	h := sha256.Sum256([]byte(strings.Join([]string{consumerID, scenario, requestID}, "\x00")))
	return "demand_" + hex.EncodeToString(h[:16])
}

// StableVariantLeaseID isolates variant holds while preserving the established
// identity of the default (live) instance.
func StableVariantLeaseID(consumerID, scenario, variant, requestID string) string {
	variant = strings.TrimSpace(variant)
	if variant != "" && variant != "live" {
		scenario += "@" + variant
	}
	return StableLeaseID(consumerID, scenario, requestID)
}

func (c Client) Acquire(ctx context.Context, req AcquireRequest) (Lease, error) {
	if strings.TrimSpace(req.LeaseID) == "" {
		req.LeaseID = StableVariantLeaseID(req.ConsumerID, req.Scenario, req.Variant, req.RequestID)
	}
	args := []string{"scenario", "demand", "acquire", "--scenario", req.Scenario, "--consumer", req.ConsumerID, "--kind", req.Kind, "--lease-id", req.LeaseID, "--json"}
	if req.Variant != "" {
		args = append(args, "--variant", req.Variant)
	}
	if req.RequestID != "" {
		args = append(args, "--request-id", req.RequestID)
	}
	if req.Metadata != "" {
		args = append(args, "--metadata", req.Metadata)
	}
	if req.TTL > 0 {
		args = append(args, "--ttl", req.TTL.String())
	}
	return c.call(ctx, args...)
}

func (c Client) Renew(ctx context.Context, leaseID string, ttl time.Duration) (Lease, error) {
	args := []string{"scenario", "demand", "renew", "--lease-id", leaseID, "--json"}
	if ttl > 0 {
		args = append(args, "--ttl", ttl.String())
	}
	return c.call(ctx, args...)
}

func (c Client) Release(ctx context.Context, leaseID, reason string) (Lease, error) {
	args := []string{"scenario", "demand", "release", "--lease-id", leaseID, "--json"}
	if reason != "" {
		args = append(args, "--reason", reason)
	}
	return c.call(ctx, args...)
}

// StartScenario marks the resulting instance demand-managed. The caller must
// acquire and retain a renewable lease before invoking this method.
func (c Client) StartScenario(ctx context.Context, scenario, variant string) error {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return fmt.Errorf("scenario is required")
	}
	if variant = strings.TrimSpace(variant); variant != "" {
		scenario += "@" + variant
	}
	args := []string{"scenario", "start", scenario, "--demand-managed"}
	if output, err := c.runner()(ctx, "vrooli", args...); err != nil {
		return fmt.Errorf("start demand-managed scenario %s: %w: %s", scenario, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (c Client) call(ctx context.Context, args ...string) (Lease, error) {
	output, err := c.runner()(ctx, "vrooli", args...)
	if err != nil {
		return Lease{}, fmt.Errorf("demand control-plane command: %w: %s", err, strings.TrimSpace(string(output)))
	}
	var envelope leaseEnvelope
	if err := json.Unmarshal(output, &envelope); err != nil {
		return Lease{}, fmt.Errorf("decode demand lease response: %w", err)
	}
	if strings.TrimSpace(envelope.Lease.LeaseID) == "" {
		return Lease{}, fmt.Errorf("demand lease response did not contain lease_id")
	}
	return envelope.Lease, nil
}
