package capacity

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"
)

func withResourceAdmissionTicks(ctx context.Context, opts AdmitOptions, ticks <-chan time.Time, run func(context.Context) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	runOwner := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		return run(ctx)
	}
	result, err := AdmitResource(ctx, opts)
	if err != nil {
		admissionLog(opts, "admission: %v", err)
		if opts.NoReclaim {
			return err
		}
		return runOwner()
	}
	for _, warning := range result.Verdict.Warnings {
		admissionLog(opts, "admission: %s", warning)
	}
	if result.Skipped || result.ClaimID == "" {
		if opts.NoReclaim {
			return fmt.Errorf("no-reclaim admission unavailable: %s", result.Reason)
		}
		return runOwner()
	}
	admissionLog(opts, "admission claim=%s verdict=%s granted_bytes=%d no_reclaim=%t", result.ClaimID, result.Verdict.Kind, result.Verdict.GrantedBytes, opts.NoReclaim)
	store, err := openAdmitStore(ctx, opts)
	if err != nil {
		admissionLog(opts, "open startup lease: %v", err)
		if opts.NoReclaim {
			return err
		}
		return runOwner()
	}
	defer store.Close()
	ttl := result.ttl

	interval := ttl / 3
	if interval <= 0 {
		interval = time.Nanosecond
	}
	if ticks == nil {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		ticks = ticker.C
	}
	// ended records that the startup claim left the active statuses: the
	// resource's capacity companion released it (Ollama holds no reservation
	// while no model is loaded) or the broker expired or preempted it. The
	// lease then has nothing to keep alive; renewing would only report a stale
	// generation on every tick. renew runs on one goroutine and once more after
	// it is joined, so the flag needs no lock.
	ended := false
	renew := func(parent context.Context) {
		if ended {
			return
		}
		callCtx, cancel := context.WithTimeout(parent, interval)
		defer cancel()
		current, getErr := store.GetClaim(callCtx, result.ClaimID)
		if getErr == nil && (current.OwnerKind != OwnerKindResource || current.OwnerID != opts.ResourceName) {
			getErr = fmt.Errorf("claim identity no longer matches resource")
		}
		if getErr == nil && !IsActiveClaimStatus(current.Status) {
			ended = true
			admissionLog(opts, "startup claim %s is %s; startup lease ended", current.ClaimID, current.Status)
			return
		}
		if getErr == nil {
			_, getErr = store.HeartbeatClaim(callCtx, current.ClaimID, current.Generation, ttl)
		}
		if getErr != nil && parent.Err() == nil {
			admissionLog(opts, "renew startup claim %s: %v", result.ClaimID, getErr)
		}
	}
	leaseCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-leaseCtx.Done():
				return
			case <-ticks:
				renew(leaseCtx)
			}
		}
	}()
	// Join before closing the store; no stranded renewal goroutine after return.
	var once sync.Once
	stop := func() { once.Do(func() { cancel(); <-done }) }
	defer stop()
	runErr := runOwner()
	stop()
	if ctx.Err() == nil {
		renew(ctx)
	}
	return runErr
}

// ReleaseResourceAdmissionAfterStop captures this owner's active claims before
// a stop performed under the caller's lifecycle lock. Only a successful verified
// terminal stop releases those IDs. Restart and uncertain stop never call this.
func ReleaseResourceAdmissionAfterStop(ctx context.Context, opts AdmitOptions, stop func() error) error {
	_, declared, err := LoadResourceClaimSpec(opts.Root, opts.ResourceName)
	if err != nil {
		admissionLog(opts, "read stop claims: %v", err)
	}
	if !declared || err != nil {
		return stop()
	}
	store, err := openAdmitStore(ctx, opts)
	if err != nil {
		admissionLog(opts, "open stop claims: %v", err)
		return stop()
	}
	defer store.Close()
	claims, err := store.ListClaims(ctx, ClaimFilter{OwnerKind: OwnerKindResource, OwnerID: opts.ResourceName, Statuses: ActiveClaimStatuses()})
	if err != nil {
		admissionLog(opts, "capture stop claims: %v", err)
	}
	if stopErr := stop(); stopErr != nil {
		return stopErr
	}
	// A successful stop may finish escalation after caller cancellation. Ledger
	// cleanup is bounded and retains normal stop authority, not a new model task.
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), tuning.CredentialServiceTimeout())
	defer cancel()
	for _, claim := range claims {
		current, getErr := store.GetClaim(cleanupCtx, claim.ClaimID)
		if getErr == nil && (current.OwnerKind != OwnerKindResource || current.OwnerID != opts.ResourceName) {
			getErr = fmt.Errorf("claim identity no longer matches resource")
		}
		if getErr == nil {
			_, getErr = store.ReleaseClaim(cleanupCtx, claim.ClaimID)
		}
		if getErr != nil {
			admissionLog(opts, "release stopped claim %s: %v", claim.ClaimID, getErr)
		}
	}
	return nil
}

func admissionLog(opts AdmitOptions, format string, args ...any) {
	if opts.Log != nil {
		fmt.Fprintf(opts.Log, "capacity (%s): "+format+"\n", append([]any{opts.ResourceName}, args...)...)
	}
}
