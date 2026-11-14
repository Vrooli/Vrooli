package sessions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/vrooli/api-core/targetmodel"
)

var ErrDesktopAdmission = errors.New("desktop admission refused")

// DesktopRepository serializes callbacks across all users of the destination.
// A successful return means the new state is durable. A callback error leaves
// the previous state intact. Implementations must not retry callbacks: they may
// contain an external native effect that cannot participate in a transaction.
type DesktopRepository interface {
	Update(context.Context, func(*DesktopState) error) error
}

type DesktopCleanupReceipt struct {
	Lease      DesktopLease `json:"lease"`
	Released   bool         `json:"released"`
	ObservedAt time.Time    `json:"observed_at"`
}

type DesktopObservationAdmission struct {
	Lease   DesktopLease `json:"lease"`
	GrantID string       `json:"grant_id"`
}

type DesktopState struct {
	Observers       map[uint64]DesktopObservationAdmission `json:"observers,omitempty"`
	grantRevoked    func(string) (bool, error)
	CleanupReceipts map[uint64]DesktopCleanupReceipt `json:"cleanup_receipts,omitempty"`
	FlowClaims      map[string]DesktopFlowClaim      `json:"flow_claims,omitempty"`
	GrantID         string                           `json:"grant_id,omitempty"`
	Halted          bool                             `json:"halted,omitempty"`
	HelperID        string                           `json:"helper_id,omitempty"`
	CleanupPending  bool                             `json:"cleanup_pending,omitempty"`
	Epoch           uint64                           `json:"epoch"`
	Lease           *DesktopLease                    `json:"lease,omitempty"`
	Receipts        map[string]DesktopReceipt        `json:"receipts,omitempty"`
}

type DesktopLease struct {
	Ref       targetmodel.SessionRef `json:"ref"`
	Actor     string                 `json:"actor"`
	HelperID  string                 `json:"helper_id"`
	Epoch     uint64                 `json:"epoch"`
	ExpiresAt time.Time              `json:"expires_at"`
	Control   bool                   `json:"control"`
}

type DesktopCommand struct {
	Lease            DesktopLease    `json:"lease"`
	ID               string          `json:"id"`
	GeometryRevision string          `json:"geometry_revision"`
	Payload          json.RawMessage `json:"payload"`
}

type DesktopReceipt struct {
	CommandID string `json:"command_id"`
	Digest    string `json:"digest"`
	Outcome   string `json:"outcome"`
}

// DesktopAuthority is provided by authenticated destination admission, never
// by the browser or the command payload. It checks the current caller, grant
// expiry/revocation, destination and requested operation on every invocation.
type DesktopAuthority interface {
	Authorize(context.Context, DesktopLease, string) error
}

// DesktopNative runs inside the authenticated OS user-session helper. Validate
// checks current lock/permission/geometry state without injecting input. Apply
// errors may follow an effect and therefore mean outcome_unknown, not retry.
type DesktopNative interface {
	Validate(context.Context, DesktopCommand) error
	Apply(context.Context, DesktopCommand) error
	ReleaseHeld(context.Context) error
}

type DesktopController struct {
	activationMu        sync.Mutex
	activation          *desktopActivationEntry
	nativeMu            sync.Mutex
	nativeCalls         map[*desktopNativeCall]struct{}
	nativeStops         map[*DesktopLease]struct{}
	repo                DesktopRepository
	authority           DesktopAuthority
	native              DesktopNative
	surface             targetmodel.SurfaceRef
	desktopID, helperID string
	now                 func() time.Time
}

func NewDesktopController(repo DesktopRepository, authority DesktopAuthority, native DesktopNative, surface targetmodel.SurfaceRef, desktopID, helperID string) (*DesktopController, error) {
	if repo == nil || authority == nil || native == nil || surface.Validate() != nil || desktopID == "" || helperID == "" {
		return nil, ErrDesktopAdmission
	}
	return &DesktopController{repo: repo, authority: authority, native: native, surface: surface, desktopID: desktopID, helperID: helperID, now: time.Now}, nil
}

func (c *DesktopController) valid(lease DesktopLease) bool {
	return lease.Ref.Validate() == nil && lease.Ref.Surface == c.surface && lease.Ref.DesktopSessionID == c.desktopID && lease.HelperID == c.helperID && lease.Actor != "" && c.now().Before(lease.ExpiresAt)
}

// Open also implements explicit takeover. expectedEpoch is the caller's last
// observed epoch; ordinary acquisition cannot replace a live lease. Epochs
// are never reused, including after restart or stop.
func (c *DesktopController) Open(ctx context.Context, lease DesktopLease, expectedEpoch uint64, takeover bool) (DesktopLease, error) {
	operation := "open"
	if takeover {
		operation = "transfer"
	}
	if !c.valid(lease) || lease.ExpiresAt.Sub(c.now()) > 10*time.Minute {
		return DesktopLease{}, ErrDesktopAdmission
	}
	if err := c.authority.Authorize(ctx, lease, operation); err != nil {
		return DesktopLease{}, ErrDesktopAdmission
	}
	var releaseErr error
	err := c.repo.Update(ctx, func(s *DesktopState) error {
		if s.Halted || (s.HelperID != "" && s.HelperID != c.helperID) {
			return ErrDesktopAdmission
		}
		if !c.valid(lease) || c.authority.Authorize(ctx, lease, operation) != nil || s.Epoch != expectedEpoch || s.Epoch == ^uint64(0) {
			return ErrDesktopAdmission
		}
		if lease.Control && s.Lease != nil && c.now().Before(s.Lease.ExpiresAt) && !takeover {
			return ErrDesktopAdmission
		}
		proposed := lease
		proposed.Epoch = s.Epoch + 1
		if c.authority.Authorize(ctx, proposed, operation) != nil {
			return ErrDesktopAdmission
		}
		grantID := ""
		if monitor, ok := c.authority.(DesktopGrantMonitor); ok {
			var err error
			grantID, err = monitor.BindGrant(ctx, proposed)
			if err != nil || grantID == "" {
				return ErrDesktopAdmission
			}
			if s.grantRevoked == nil {
				return ErrDesktopAdmission
			}
			revoked, err := s.grantRevoked(grantID)
			if err != nil || revoked {
				return ErrDesktopAdmission
			}

		}
		if !lease.Control {
			if takeover || len(s.Observers) >= 64 || s.CleanupPending {
				return ErrDesktopAdmission
			}
			if s.Lease != nil && s.Lease.Ref == proposed.Ref {
				return ErrDesktopAdmission
			}
			for _, observer := range s.Observers {
				if observer.Lease.Ref == proposed.Ref {
					return ErrDesktopAdmission
				}
			}
			if s.Observers == nil {
				s.Observers = make(map[uint64]DesktopObservationAdmission)
			}
			s.HelperID = c.helperID
			s.Epoch++
			lease = proposed
			s.Observers[lease.Epoch] = DesktopObservationAdmission{Lease: lease, GrantID: grantID}
			return nil
		}
		// Keep the old epoch until held input has been released. This call and all
		// actuation share the repository exclusion boundary.
		s.HelperID = c.helperID
		c.pruneActivation(s, true)
		releaseErr = c.native.ReleaseHeld(ctx)
		s.recordCleanup(c.now(), releaseErr)
		if releaseErr != nil {
			s.Lease = nil
			s.GrantID = ""
			return nil
		}
		s.Epoch++
		lease.Epoch = s.Epoch
		s.Lease = &lease
		s.GrantID = grantID
		s.Receipts = make(map[string]DesktopReceipt)
		s.FlowClaims = make(map[string]DesktopFlowClaim)
		return nil
	})
	if err != nil {
		return DesktopLease{}, err
	}
	if releaseErr != nil {
		return DesktopLease{}, releaseErr
	}
	return lease, nil
}

func sameDesktopLease(a, b DesktopLease) bool {
	return a.Ref == b.Ref && a.Actor == b.Actor && a.HelperID == b.HelperID && a.Epoch == b.Epoch && a.Control == b.Control && a.ExpiresAt.Equal(b.ExpiresAt)
}

// admittedLease matches independent observation membership or the exclusive
// controller. The global epoch allocates identities; it is not the live lease.
func (c *DesktopController) admittedLease(s *DesktopState, lease DesktopLease) (string, bool) {
	if s.HelperID != c.helperID || s.Halted || s.CleanupPending {
		return "", false
	}
	if !lease.Control {
		if observer, ok := s.Observers[lease.Epoch]; ok && sameDesktopLease(observer.Lease, lease) {
			return observer.GrantID, true
		}
	}
	// Preserve cleanup/admission of an older singleton observation lease.
	if s.Lease != nil && sameDesktopLease(*s.Lease, lease) {
		return s.GrantID, true
	}
	return "", false
}

func (c *DesktopController) admitted(s *DesktopState, lease DesktopLease) bool {
	grant, ok := c.admittedLease(s, lease)
	if !ok || !c.valid(lease) {
		return false
	}
	if grant != "" {
		if s.grantRevoked == nil {
			return false
		}
		revoked, err := s.grantRevoked(grant)
		if err != nil || revoked {
			return false
		}
	}
	return true
}

func (c *DesktopController) Act(ctx context.Context, command DesktopCommand) (DesktopReceipt, error) {
	if len(command.ID) == 0 || len(command.ID) > 128 || len(command.Payload) == 0 || len(command.Payload) > 64*1024 || !json.Valid(command.Payload) || command.GeometryRevision == "" {
		return DesktopReceipt{}, ErrDesktopAdmission
	}
	if !c.valid(command.Lease) || !command.Lease.Control || c.authority.Authorize(ctx, command.Lease, "act") != nil {
		return DesktopReceipt{}, ErrDesktopAdmission
	}
	ctx, cancel := c.nativeContext(ctx, command.Lease)
	defer cancel()
	encoded, err := json.Marshal(command)
	if err != nil {
		return DesktopReceipt{}, err
	}
	sum := sha256.Sum256(encoded)
	receipt := DesktopReceipt{CommandID: command.ID, Digest: hex.EncodeToString(sum[:]), Outcome: "outcome_unknown"}
	existing := false
	// Admission is durable before entering the native-effect boundary. A crash
	// after this commit leaves an uncertain receipt that is never blindly replayed.
	err = c.repo.Update(ctx, func(s *DesktopState) error {
		if !c.admitted(s, command.Lease) {
			return ErrDesktopAdmission
		}
		if prior, ok := s.Receipts[command.ID]; ok {
			if prior.Digest != receipt.Digest {
				return fmt.Errorf("conflicting command identity")
			}
			receipt, existing = prior, true
			return nil
		}
		if err := admitFlowCommand(s, command.ID); err != nil {
			return err
		}
		if len(s.Receipts) >= 1024 {
			return fmt.Errorf("session command limit reached")
		}
		if err := c.native.Validate(ctx, command); err != nil {
			return ErrDesktopAdmission
		}
		if s.Receipts == nil {
			s.Receipts = make(map[string]DesktopReceipt)
		}
		s.Receipts[command.ID] = receipt
		return nil
	})
	if err != nil || existing {
		return receipt, err
	}
	err = c.repo.Update(ctx, func(s *DesktopState) error {
		if !c.admitted(s, command.Lease) || c.authority.Authorize(ctx, command.Lease, "act") != nil {
			return ErrDesktopAdmission
		}
		if admitFlowCommand(s, command.ID) != nil || c.native.Validate(ctx, command) != nil {
			return ErrDesktopAdmission
		}
		effectCtx := context.WithValue(ctx, desktopMutationKey{}, func(check context.Context) error {
			if check.Err() != nil || !c.valid(command.Lease) || c.authority.Authorize(check, command.Lease, "act") != nil {
				return ErrDesktopAdmission
			}
			return nil
		})
		if CheckDesktopMutation(effectCtx) != nil {
			return ErrDesktopAdmission
		}
		if c.native.Apply(effectCtx, command) == nil {
			receipt.Outcome = "applied"
		}
		s.Receipts[command.ID] = receipt
		return nil
	})
	if err != nil {
		receipt.Outcome = "outcome_unknown"
	}
	return receipt, err
}

func (c *DesktopController) Stop(ctx context.Context, lease DesktopLease) error {
	if c.authority.Authorize(ctx, lease, "stop") != nil {
		return ErrDesktopAdmission
	}
	endStop := c.interruptNative(lease)
	defer endStop()
	var releaseErr error
	err := c.repo.Update(ctx, func(s *DesktopState) error {
		if observer, ok := s.Observers[lease.Epoch]; ok && sameDesktopLease(observer.Lease, lease) && lease.HelperID == c.helperID && lease.Ref.Surface == c.surface && lease.Ref.DesktopSessionID == c.desktopID {
			s.removeObserver(lease.Epoch, c.now())
			c.pruneActivation(s, false)
			return nil
		}
		// Expired leases can still release held input, but cannot stop a successor.
		if s.Lease == nil || !sameDesktopLease(*s.Lease, lease) || lease.HelperID != c.helperID || lease.Ref.Surface != c.surface || lease.Ref.DesktopSessionID != c.desktopID {
			return ErrDesktopAdmission
		}
		c.pruneActivation(s, true)
		releaseErr = c.native.ReleaseHeld(ctx)
		s.recordCleanup(c.now(), releaseErr)
		// Release failure must not preserve authority for further queued actions.
		s.Lease = nil
		s.GrantID = ""
		return nil
	})
	if err != nil {
		return err
	}
	return releaseErr
}

// Record release evidence before discarding the lease. A later destination-owned
// successful release reconciles prior failures without reviving any lease.
func (s *DesktopState) recordCleanup(now time.Time, releaseErr error) {
	s.CleanupPending = releaseErr != nil
	if s.Lease != nil {
		if s.CleanupReceipts == nil {
			s.CleanupReceipts = make(map[uint64]DesktopCleanupReceipt)
		}
		s.CleanupReceipts[s.Lease.Epoch] = DesktopCleanupReceipt{Lease: *s.Lease, Released: releaseErr == nil, ObservedAt: now}
	}
	if releaseErr == nil {
		for epoch, receipt := range s.CleanupReceipts {
			if !receipt.Released {
				receipt.Released = true
				receipt.ObservedAt = now
				s.CleanupReceipts[epoch] = receipt
			}
		}
	}
}

// ReadCleanup is distinct from live input admission and never performs native work.
func (c *DesktopController) ReadCleanup(ctx context.Context, lease DesktopLease) (DesktopCleanupReceipt, error) {
	authority, ok := c.authority.(interface {
		AuthorizeCleanupRead(context.Context, DesktopLease) error
	})
	if !ok || authority.AuthorizeCleanupRead(ctx, lease) != nil || lease.Ref.Surface != c.surface || lease.Ref.DesktopSessionID != c.desktopID {
		return DesktopCleanupReceipt{}, ErrDesktopAdmission
	}
	repository, ok := c.repo.(interface {
		ReadCleanup(context.Context, DesktopLease) (DesktopCleanupReceipt, error)
	})
	if !ok {
		return DesktopCleanupReceipt{}, ErrDesktopAdmission
	}
	receipt, err := repository.ReadCleanup(ctx, lease)
	if err == nil {
		return receipt, nil
	}
	proof, ok := c.authority.(interface {
		RevokedCleanupGrant(context.Context, DesktopLease) (string, error)
	})
	if !ok {
		return DesktopCleanupReceipt{}, err
	}
	id, proofErr := proof.RevokedCleanupGrant(ctx, lease)
	if proofErr != nil {
		return DesktopCleanupReceipt{}, err
	}
	fenced, ok := c.repo.(interface {
		ReadRevokedCleanup(context.Context, DesktopLease, string) (DesktopCleanupReceipt, error)
	})
	if !ok {
		return DesktopCleanupReceipt{}, err
	}
	return fenced.ReadRevokedCleanup(ctx, lease, id)
}
