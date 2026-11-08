package capacity

import (
	"context"
	"fmt"

	"github.com/vrooli/vrooli/internal/artifactlock"
)

// LockAdmission serializes strict no-reclaim sensing, decision and recording
// against the same canonical ledger across processes. It must not be held over
// external reclaim callbacks: those can invoke the resource lifecycle again.
// Ordinary advisory admissions retain their explicitly non-blocking policy.
func LockAdmission(ctx context.Context, store ClaimRepository) (func(), error) {
	owner, ok := store.(interface {
		LockAdmission(context.Context) (func(), error)
	})
	if !ok {
		return nil, fmt.Errorf("capacity store does not provide admission serialization")
	}
	return owner.LockAdmission(ctx)
}

func (s *SQLiteStore) LockAdmission(ctx context.Context) (func(), error) {
	if s.dbPath == ":memory:" {
		s.admissionOnce.Do(func() { s.admissionGate = make(chan struct{}, 1) })
		ch := s.admissionGate
		select {
		case ch <- struct{}{}:
			return func() { <-ch }, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return artifactlock.AcquireWithPathContext(ctx, "capacity-admission:"+s.dbPath, func(string) (string, error) {
		return s.dbPath + ".admission.lock", nil
	})
}
