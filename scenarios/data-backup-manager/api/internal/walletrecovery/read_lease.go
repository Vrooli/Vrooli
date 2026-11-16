package walletrecovery

import (
	"context"
	"errors"
	"strings"
	"time"
)

var ErrSnapshotSuperseded = errors.New("requested recovery snapshot is no longer newest")

type SnapshotState struct {
	ID         string
	TargetID   string
	Generation int64
}

// SnapshotCatalog reads the validated DBM catalog and mutation generation.
// Implementations must bracket catalog enumeration with generation reads so
// an in-flight or completed mutation cannot be mistaken for a stable head.
type SnapshotCatalog interface {
	NewestRecoverySnapshot(context.Context, string, string) (SnapshotState, error)
}

type ReadLeaseStore interface {
	ReserveRead(context.Context, Reservation, time.Time) error
	RenewRead(context.Context, string, string, time.Time, time.Time) error
	ReleaseRead(context.Context, string, string) error
}

type ReadLease struct {
	LeaseID       string    `json:"lease_id"`
	TargetID      string    `json:"target_id"`
	DestinationID string    `json:"destination_id"`
	SnapshotID    string    `json:"snapshot_id"`
	Generation    int64     `json:"generation"`
	ExpiresAt     time.Time `json:"expires_at"`
}

type ReadLeaseService struct {
	Catalog SnapshotCatalog
	Store   ReadLeaseStore
	Now     func() time.Time
	TTL     time.Duration
}

const maxReadLeaseTTL = 15 * time.Minute

func (s ReadLeaseService) Acquire(ctx context.Context, leaseID, targetName, destinationID, snapshotID string) (ReadLease, error) {
	leaseID, targetName, destinationID, snapshotID = strings.TrimSpace(leaseID), strings.TrimSpace(targetName), strings.TrimSpace(destinationID), strings.TrimSpace(snapshotID)
	if leaseID == "" || targetName == "" || destinationID == "" || snapshotID == "" {
		return ReadLease{}, errors.New("lease, target name, destination and snapshot identities are required")
	}
	if s.Catalog == nil || s.Store == nil {
		return ReadLease{}, errors.New("snapshot catalog and reservation store are not configured")
	}
	latest, err := s.Catalog.NewestRecoverySnapshot(ctx, targetName, destinationID)
	if err != nil {
		return ReadLease{}, err
	}
	if latest.ID != snapshotID {
		return ReadLease{}, ErrSnapshotSuperseded
	}
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	ttl := s.TTL
	if ttl <= 0 {
		ttl = maxReadLeaseTTL
	} else if ttl > maxReadLeaseTTL {
		ttl = maxReadLeaseTTL
	}
	if strings.TrimSpace(latest.TargetID) == "" {
		return ReadLease{}, errors.New("DBM catalog omitted durable target ID for read reservation")
	}
	lease := ReadLease{LeaseID: leaseID, TargetID: latest.TargetID, DestinationID: destinationID, SnapshotID: snapshotID, Generation: latest.Generation, ExpiresAt: now.Add(ttl)}
	if err := s.Store.ReserveRead(ctx, Reservation{
		RequestID: leaseID, TargetID: latest.TargetID, DestinationID: destinationID,
		SnapshotID: snapshotID, SnapshotGeneration: latest.Generation, ExpiresAt: lease.ExpiresAt,
	}, now); err != nil {
		return ReadLease{}, err
	}
	return lease, nil
}

func (s ReadLeaseService) Renew(ctx context.Context, leaseID, targetID string) (ReadLease, error) {
	leaseID, targetID = strings.TrimSpace(leaseID), strings.TrimSpace(targetID)
	if leaseID == "" || targetID == "" || s.Store == nil {
		return ReadLease{}, errors.New("lease, target and reservation store are required")
	}
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	ttl := s.TTL
	if ttl <= 0 {
		ttl = maxReadLeaseTTL
	} else if ttl > maxReadLeaseTTL {
		ttl = maxReadLeaseTTL
	}
	expiresAt := now.Add(ttl)
	if err := s.Store.RenewRead(ctx, leaseID, targetID, expiresAt, now); err != nil {
		return ReadLease{}, err
	}
	return ReadLease{LeaseID: leaseID, TargetID: targetID, ExpiresAt: expiresAt}, nil
}

func (s ReadLeaseService) Release(ctx context.Context, leaseID, targetID string) error {
	if s.Store == nil {
		return errors.New("reservation store is not configured")
	}
	return s.Store.ReleaseRead(ctx, strings.TrimSpace(leaseID), strings.TrimSpace(targetID))
}
