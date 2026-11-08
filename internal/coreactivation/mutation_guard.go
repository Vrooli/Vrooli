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
