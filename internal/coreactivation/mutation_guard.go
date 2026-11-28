package coreactivation

import (
	"context"
	"fmt"
	"strings"

	platform "github.com/vrooli/platform-go"
)

// GuardOrdinaryMutation is read-only. Call it while holding the shared root
// install lock before ordinary image/unit replacement. It never acquires the
// operation lock (a worker may hold that while waiting for the install lock),
// starts recovery, or turns native absence into permission to replay work.
func (s *Store) GuardOrdinaryMutation(ctx context.Context) error {
	if ctx == nil {
		return ErrConflict
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	r, exists, err := s.Current()
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	if !r.Terminal() {
		return fmt.Errorf("%w: recorded root activation must finish before an ordinary update", ErrConflict)
	}
	if !r.DispatchCommitted {
		return nil
	}
	var absent bool
	if r.WorkerRef == nil {
		absent, err = platform.OwnedOperationIntentAbsent(ctx, r.WorkerIntent)
	} else {
		boot, _, _ := strings.Cut(r.PriorIncarnation, ":")
		absent, err = platform.OwnedOperationReleasedForIncarnation(ctx, *r.WorkerRef, boot+":"+r.WorkerRef.RootStartTime)
	}
	if err != nil || !absent {
		return fmt.Errorf("%w: root activation native retirement is unresolved", ErrConflict)
	}
	return nil
}

// ObserveVerifiedRetirement binds native absence to the exact verified record
// and rejects a concurrent replacement or revision of that record.
func (s *Store) ObserveVerifiedRetirement(ctx context.Context, id string) (bool, error) {
	return s.observeVerifiedRetirement(ctx, id, func(r Record) (bool, error) {
		if r.WorkerRef == nil {
			return platform.OwnedOperationIntentAbsent(ctx, r.WorkerIntent)
		}
		boot, _, _ := strings.Cut(r.PriorIncarnation, ":")
		return platform.OwnedOperationReleasedForIncarnation(ctx, *r.WorkerRef, boot+":"+r.WorkerRef.RootStartTime)
	})
}

func (s *Store) observeVerifiedRetirement(ctx context.Context, id string, observe func(Record) (bool, error)) (bool, error) {
	r, exists, err := s.Current()
	if err != nil {
		return false, err
	}
	if !exists || r.ID != id || r.Phase != Verified || !r.DispatchCommitted {
		return false, fmt.Errorf("root update identity or verified state changed before native retirement")
	}
	released, err := observe(r)
	if err != nil || !released {
		return false, err
	}
	after, exists, err := s.Current()
	if err != nil {
		return false, err
	}
	if !exists || after.ID != r.ID || after.Revision != r.Revision || after.Phase != Verified {
		return false, fmt.Errorf("root update identity or verified state changed during native retirement check")
	}
	return true, nil
}
