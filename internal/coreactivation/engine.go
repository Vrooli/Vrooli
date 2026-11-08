package coreactivation

import (
	"context"
	"fmt"
	"reflect"
	"time"

	platform "github.com/vrooli/platform-go"
)

type Engine struct {
	Store   *Store
	Effects Effects
	Now     func() time.Time
	start   func(context.Context, platform.OperationIntent, platform.OperationSpec) (*platform.OwnedOperation, error)
	// Fixture seam for state-machine tests using marker images. Production
	// always derives compatibility from the actual frozen root executables.
	qualifyFrozen func(context.Context, Record) error
}

func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

// Begin admits already-qualified frozen inputs. It cannot turn a readiness or
// compatibility result into operator authorization. Its caller owns authority.
func (e *Engine) Begin(ctx context.Context, r Record, explicitRetry bool) (Record, error) {
	if e.Store == nil || e.Effects == nil {
		return Record{}, fmt.Errorf("core activation owner is incomplete")
	}
	release, err := e.Store.lock(ctx)
	if err != nil {
		return Record{}, err
	}
	defer release()
	current, exists, err := e.Store.Current()
	if err != nil {
		return Record{}, err
	}
	if exists && current.ID == r.ID {
		return current, fmt.Errorf("%w: request identity cannot be reused", ErrConflict)
	}
	r.FailedReleases = []string{}
	r.FailedBuilds = []string{}
	if exists {
		if !current.Terminal() {
			return Record{}, fmt.Errorf("%w: another activation remains unresolved", ErrConflict)
		}
		if err := e.Store.put(&current); err != nil {
			return current, err
		} // Confirm a previously ambiguous terminal commit.
		if err := e.Effects.ExcludeWorker(ctx, current); err != nil {
			return current, err
		}
		if err := e.Effects.RetireInputs(ctx, current); err != nil {
			return current, err
		}
		r.FailedReleases = append([]string{}, current.FailedReleases...)
		r.FailedBuilds = append([]string{}, current.FailedBuilds...)
	}
	failed := contains(r.FailedReleases, r.CandidateSHA) || contains(r.FailedBuilds, r.CandidateBuild)
	if failed && !explicitRetry {
		return current, ErrFailedRelease
	}
	if len(r.FailedReleases) >= 32 && !contains(r.FailedReleases, r.CandidateSHA) || len(r.FailedBuilds) >= 32 && !contains(r.FailedBuilds, r.CandidateBuild) {
		return Record{}, fmt.Errorf("%w: failed-release inventory requires review", ErrConflict)
	}
	r.SchemaVersion = recordSchemaVersion
	r.Revision = 1
	r.Phase = Prepared
	r.Fence = Ready
	r.WorkerRef = nil
	r.DispatchCommitted = false
	r.FailureClass = ""
	r.RestorationRequired = false
	if err := r.validate(e.Store.home); err != nil {
		return Record{}, err
	}
	if r.CreatedUnixNano > e.now().UnixNano() || !e.now().Before(time.Unix(0, r.DeadlineUnixNano)) {
		return Record{}, fmt.Errorf("%w: admission deadline elapsed", ErrConflict)
	}
	qualify := e.qualifyFrozen
	// Operation -> install is also the worker/recovery lock order. Retain
	// admission's install lock through qualification and durable commit.
	if locker, ok := e.Effects.(interface {
		LockAdmission(context.Context, Record) (func(), error)
	}); ok {
		release, err := locker.LockAdmission(ctx, r)
		if err != nil {
			return Record{}, err
		}
		defer release()
	}
	if qualify == nil {
		qualify = (Assets{Store: e.Store}).QualifyFrozenCompatibility
	}
	if err := qualify(ctx, r); err != nil {
		return Record{}, err
	}
	if err := e.Effects.QualifyAdmission(ctx, r); err != nil {
		return Record{}, err
	}
	if err := ctx.Err(); err != nil {
		return Record{}, err
	}
	if !e.now().Before(time.Unix(0, r.DeadlineUnixNano)) {
		return Record{}, fmt.Errorf("%w: qualification exceeded the admission deadline", ErrConflict)
	}
	if err := e.Store.put(&r); err != nil {
		return Record{}, err
	}
	return r, nil
}

// Dispatch commits a once-only submission before contacting the native owner.
// Even when that commit or submission loses its acknowledgement, a caller may
// only inspect/recover the same intent. It must never submit that intent again.
func (e *Engine) Dispatch(ctx context.Context, id string, spec platform.OperationSpec) (*platform.OwnedOperation, error) {
	if e.Store == nil {
		return nil, fmt.Errorf("core activation store is required")
	}
	release, err := e.Store.lock(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	r, exists, err := e.Store.Current()
	if err != nil {
		return nil, err
	}
	if !exists || r.ID != id {
		return nil, ErrConflict
	}
	if r.Phase != Prepared || r.Fence != Ready || r.DispatchCommitted {
		return nil, ErrClaimed
	}
	if !e.now().Before(time.Unix(0, r.DeadlineUnixNano)) {
		return nil, fmt.Errorf("core activation dispatch deadline elapsed")
	}
	if err := r.WorkerIntent.ValidateSpec(spec); err != nil {
		return nil, err
	}
	if int64(spec.Limits.WallTime) > r.DeadlineUnixNano-r.CreatedUnixNano {
		return nil, fmt.Errorf("core activation native lifetime exceeds the admitted bound")
	}
	r.DispatchCommitted = true
	if err := e.Store.put(&r); err != nil {
		return nil, err
	}
	// The worker claims under this same lock. Release before dispatching so
	// acknowledgement does not depend on the worker waiting on its caller.
	release()
	start := e.start
	if start == nil {
		start = platform.StartOwnedOperation
	}
	return start(ctx, r.WorkerIntent, spec)
}

// RunWorker is invoked only by the bounded native adapter. The lock remains
// held through every effect and outcome, so an independent recovery owner
// cannot race a live worker. A claimed request is never replay permission.
func (e *Engine) RunWorker(ctx context.Context, id string, ref platform.OperationRef) (Record, error) {
	if e.Store == nil || e.Effects == nil {
		return Record{}, fmt.Errorf("core activation owner is incomplete")
	}
	release, err := e.Store.lock(ctx)
	if err != nil {
		return Record{}, err
	}
	defer release()
	r, exists, err := e.Store.Current()
	if err != nil {
		return Record{}, err
	}
	if !exists || r.ID != id {
		return Record{}, ErrConflict
	}
	if r.Fence != Ready || r.Phase != Prepared || !r.DispatchCommitted {
		return r, ErrClaimed
	}
	if !e.now().Before(time.Unix(0, r.DeadlineUnixNano)) {
		return r, fmt.Errorf("%w: worker deadline elapsed", ErrConflict)
	}
	if err := e.Effects.AuthorizeWorker(ctx, r, ref); err != nil {
		return r, err
	}
	if ref.Intent != r.WorkerIntent {
		return r, ErrConflict
	}
	ctx, cancel := context.WithDeadline(ctx, time.Unix(0, r.DeadlineUnixNano))
	defer cancel()
	r.WorkerRef = &ref
	r.Fence = Claimed
	if err := e.Store.put(&r); err != nil {
		return r, err
	}
	steps := []struct {
		phase   Phase
		failure string
		effect  func(context.Context, Record) error
	}{
		{Publishing, "publication", e.Effects.PublishCandidate},
		{Activating, "activation", e.Effects.ActivateCandidate},
		{Verifying, "verification", e.Effects.VerifyCandidate},
	}
	for _, step := range steps {
		if err := ctx.Err(); err != nil {
			return r, err
		}
		if !e.now().Before(time.Unix(0, r.DeadlineUnixNano)) {
			return r, fmt.Errorf("core activation deadline elapsed")
		}
		r.Phase = step.phase
		if err := e.Store.put(&r); err != nil {
			return r, err
		}
		if err := step.effect(ctx, r); err != nil {
			r.Phase = Restoring
			r.Fence = Revoked
			r.FailureClass = step.failure
			if commitErr := e.Store.put(&r); commitErr != nil {
				return r, commitErr
			}
			return r, fmt.Errorf("core activation %s failed; independent recovery required: %w", step.failure, err)
		}
	}
	if err := ctx.Err(); err != nil {
		return r, err
	}
	if !e.now().Before(time.Unix(0, r.DeadlineUnixNano)) {
		return r, fmt.Errorf("core activation deadline elapsed")
	}
	r.Phase = Verified
	r.Fence = Revoked
	if err := e.Store.put(&r); err != nil {
		return r, err
	}
	return r, nil
}

// Recover acts only on an admitted operation. It never builds, enrolls,
// redispatches a native intent or treats a missing process as an exit outcome.
func (e *Engine) Recover(ctx context.Context) (Record, bool, error) {
	if e.Store == nil || e.Effects == nil {
		return Record{}, false, fmt.Errorf("core activation owner is incomplete")
	}
	release, err := e.Store.lock(ctx)
	if err != nil {
		return Record{}, false, err
	}
	defer release()
	r, exists, err := e.Store.Current()
	if err != nil || !exists {
		return r, exists, err
	}
	if r.Terminal() {
		if err := e.Store.put(&r); err != nil {
			return r, true, err
		}
		if err := e.Effects.ExcludeWorker(ctx, r); err != nil {
			return r, true, err
		}
		return r, true, e.Effects.RetireInputs(ctx, r)
	}
	if r.Fence == Ready && e.now().Before(time.Unix(0, r.DeadlineUnixNano)) {
		return r, true, nil
	}
	if r.FailureClass == "" {
		r.FailureClass = "worker_lost"
		if !e.now().Before(time.Unix(0, r.DeadlineUnixNano)) {
			r.FailureClass = "deadline"
		}
	}
	r.Phase = Restoring
	r.Fence = Revoked
	// Even after a lost acknowledgement, confirm this exact revocation
	// durably before excluding a worker or changing the serving image.
	if err := e.Store.put(&r); err != nil {
		return r, true, err
	}
	if err := e.Effects.ExcludeWorker(ctx, r); err != nil {
		return r, true, err
	}
	if err := e.Effects.VerifyPrior(ctx, r); err == nil && !r.RestorationRequired {
		r.Phase = FailedPreserved
	} else {
		// Persist restoration intent before its first effect. After a lost
		// effect response or terminal commit, replay the same idempotent
		// restoration rather than relabelling it as an untouched prior image.
		if !r.RestorationRequired {
			r.RestorationRequired = true
			if err := e.Store.put(&r); err != nil {
				return r, true, err
			}
		}
		if err := e.Effects.RestorePrior(ctx, r); err != nil {
			return r, true, err
		}
		if err := e.Effects.VerifyPrior(ctx, r); err != nil {
			return r, true, err
		}
		r.Phase = FailedRestored
	}
	if !contains(r.FailedReleases, r.CandidateSHA) {
		r.FailedReleases = append(r.FailedReleases, r.CandidateSHA)
	}
	if !contains(r.FailedBuilds, r.CandidateBuild) {
		r.FailedBuilds = append(r.FailedBuilds, r.CandidateBuild)
	}
	if err := e.Store.put(&r); err != nil {
		return r, true, err
	}
	return r, true, e.Effects.RetireInputs(ctx, r)
}

// GuardActivation is read-only and suitable for a child convergence adapter
// while the native worker owns the operation lock. The adapter must hold the
// separate root-install lock, verify native membership and compare this frozen
// unit digest before touching its scheduler definition.
func (s *Store) GuardActivation(id, unitSHA string, ref platform.OperationRef) error {
	r, exists, err := s.Current()
	if err != nil {
		return err
	}
	if !exists {
		return ErrConflict
	}
	if r.ID != id || r.Phase != Activating || r.Fence != Claimed || r.CandidateUnitSHA != unitSHA || r.WorkerRef == nil || !reflect.DeepEqual(*r.WorkerRef, ref) {
		return ErrClaimed
	}
	return nil
}

func contains(values []string, v string) bool {
	for _, value := range values {
		if value == v {
			return true
		}
	}
	return false
}
