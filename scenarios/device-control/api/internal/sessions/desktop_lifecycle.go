package sessions

import (
	"context"
	"errors"
	"time"
)

// ActivateHelper is a trusted helper bootstrap operation, never a client RPC.
// The lifecycle coordinator supplies the previously registered generation and
// epoch after authenticating the OS session. CAS prevents a delayed bootstrap
// from overwriting a successor. A new generation fences all previous grants.
func (c *DesktopController) ActivateHelper(ctx context.Context, previousHelperID string, expectedEpoch uint64) error {
	var releaseErr error
	err := c.repo.Update(ctx, func(s *DesktopState) error {
		if s.HelperID != previousHelperID || s.Epoch != expectedEpoch || s.Epoch == ^uint64(0) || s.HelperID == c.helperID {
			return ErrDesktopAdmission
		}
		for epoch := range s.Observers {
			s.removeObserver(epoch, c.now())
		}
		s.HelperID = c.helperID
		s.Halted = false
		s.Epoch++
		// Preserve prior receipts for reconciliation. A subsequent authorized Open
		// starts its own receipt namespace only after releasing held input.
		c.pruneActivation(s, true)
		releaseErr = c.native.ReleaseHeld(ctx)
		s.recordCleanup(c.now(), releaseErr)
		s.Lease = nil
		s.GrantID = ""
		return nil
	})
	if err != nil {
		return err
	}
	return releaseErr
}

// ReapExpired checks expiry, grant revocation and native session state in the
// helper lifecycle loop. It does not require a
// still-live caller grant: the destination owns expiry and release. A stale
// helper generation cannot release inputs owned by its successor.
func (c *DesktopController) ReapExpired(ctx context.Context) error {
	return c.cleanup(ctx, false)
}

// Shutdown revokes this helper generation's lease before its listener closes.
// The caller supplies a fresh bounded cleanup context, not an already-cancelled
// request context. Failed release remains durable and is retried at bootstrap.
func (c *DesktopController) Shutdown(ctx context.Context) error {
	return c.cleanup(ctx, true)
}

func (c *DesktopController) cleanup(ctx context.Context, force bool) error {
	var releaseErr error
	err := c.repo.Update(ctx, func(s *DesktopState) error {
		c.pruneActivation(s, force)
		if s.HelperID != c.helperID {
			return ErrDesktopAdmission
		}
		if force {
			s.Halted = true
		}
		mustRelease := force || s.CleanupPending
		if s.GrantID != "" && s.grantRevoked != nil {
			revoked, err := s.grantRevoked(s.GrantID)
			if err != nil {
				return err
			}
			mustRelease = mustRelease || revoked
		}
		if s.Lease != nil && !mustRelease {
			mustRelease = !c.now().Before(s.Lease.ExpiresAt)
			if !mustRelease {
				if monitor, ok := c.authority.(DesktopGrantMonitor); ok {
					mustRelease = monitor.CheckGrant(ctx, s.GrantID) != nil
				}
			}
			if !mustRelease {
				if native, ok := c.native.(DesktopSessionMonitor); ok {
					mustRelease = native.CheckSession(ctx) != nil
				}
			}
		}
		for epoch, observer := range s.Observers {
			remove := force || observer.Lease.HelperID != c.helperID || !c.now().Before(observer.Lease.ExpiresAt)
			if !remove && observer.GrantID != "" {
				if s.grantRevoked == nil {
					remove = true
				} else {
					revoked, err := s.grantRevoked(observer.GrantID)
					remove = err != nil || revoked
				}
				if monitor, ok := c.authority.(DesktopGrantMonitor); ok && !remove {
					remove = monitor.CheckGrant(ctx, observer.GrantID) != nil
				}
			}
			if !remove {
				if native, ok := c.native.(DesktopSessionMonitor); ok {
					remove = native.CheckSession(ctx) != nil
				}
			}
			if remove {
				s.removeObserver(epoch, c.now())
			}
		}
		c.pruneActivation(s, false)
		if !mustRelease {
			return nil
		}
		c.pruneActivation(s, true)
		releaseErr = c.native.ReleaseHeld(ctx)
		s.recordCleanup(c.now(), releaseErr)
		s.Lease = nil
		s.GrantID = ""
		return nil
	})
	if err != nil {
		return err
	}
	return releaseErr
}

// MaintainLease runs inside the helper process. It owns one bounded expiry
// check per tick and a final shutdown attempt. Errors are returned to the
// lifecycle supervisor so failed cleanup is visible and can be retried; they
// must never be treated as successful input release.
func (c *DesktopController) MaintainLease(ctx context.Context) (result error) {
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		result = errors.Join(result, c.Shutdown(cleanupCtx))
	}()
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			cleanupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := c.ReapExpired(cleanupCtx)
			cancel()
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				return err
			}
		}
	}
}

// DesktopSessionMonitor checks native login/lock/permission state without input.
// It supplements grant expiry/revocation; process liveness alone is insufficient.
type DesktopSessionMonitor interface{ CheckSession(context.Context) error }

// Observation membership never owned held input. Its cleanup proof therefore
// records authority removal without releasing another session's input.
func (s *DesktopState) removeObserver(epoch uint64, now time.Time) {
	observer, ok := s.Observers[epoch]
	if !ok {
		return
	}
	if s.CleanupReceipts == nil {
		s.CleanupReceipts = make(map[uint64]DesktopCleanupReceipt)
	}
	s.CleanupReceipts[epoch] = DesktopCleanupReceipt{Lease: observer.Lease, Released: true, ObservedAt: now}
	delete(s.Observers, epoch)
}
