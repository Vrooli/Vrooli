package runs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	sharedartifacts "test-genie/internal/shared/artifacts"
)

// PinLease is an expiring ownership claim over durable run evidence. Unlike a
// historical index pin, it always names an owner and expiry so retention cannot
// be bypassed indefinitely by abandoned baseline references.
type PinLease struct {
	RunID     string    `json:"runId"`
	Owner     string    `json:"owner"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// DefaultPinLeaseTTL is intentionally finite. Consumers that need longer
// protection renew their lease through the existing PinRun RPC.
const DefaultPinLeaseTTL = 30 * 24 * time.Hour

func (l PinLease) ActiveAt(now time.Time) bool {
	return l.RunID != "" && l.Owner != "" && !l.ExpiresAt.IsZero() && l.ExpiresAt.After(now)
}

// PinLeaseStore is the sole persistence owner for pin leases. It uses the same
// advisory-lock + atomic-replace discipline as the run index without coupling
// mutable retention policy to immutable run evidence.
type PinLeaseStore struct {
	path     string
	lockPath string
}

func NewPinLeaseStore(scenarioDir string) *PinLeaseStore {
	path := filepath.Join(scenarioDir, sharedartifacts.CoverageRoot, "pin-leases.json")
	return &PinLeaseStore{path: path, lockPath: path + ".lock"}
}

func (s *PinLeaseStore) Grant(runID, owner, reason string, ttl time.Duration, now time.Time) (PinLease, error) {
	if runID == "" || owner == "" || ttl <= 0 {
		return PinLease{}, fmt.Errorf("run id, owner, and positive lease duration are required")
	}
	lease := PinLease{RunID: runID, Owner: owner, Reason: reason, CreatedAt: now.UTC(), ExpiresAt: now.UTC().Add(ttl)}
	err := s.withLock(func() error {
		leases, err := s.readUnlocked()
		if err != nil {
			return err
		}
		filtered := leases[:0]
		for _, current := range leases {
			if current.RunID != runID || current.Owner != owner {
				filtered = append(filtered, current)
			}
		}
		return s.writeUnlocked(append(filtered, lease))
	})
	return lease, err
}

func (s *PinLeaseStore) Revoke(runID, owner string) error {
	return s.withLock(func() error {
		leases, err := s.readUnlocked()
		if err != nil {
			return err
		}
		filtered := leases[:0]
		for _, current := range leases {
			if current.RunID != runID || current.Owner != owner {
				filtered = append(filtered, current)
			}
		}
		return s.writeUnlocked(filtered)
	})
}

func (s *PinLeaseStore) Active(runID string, now time.Time) (bool, error) {
	leases, err := s.ActiveForRun(runID, now)
	return len(leases) > 0, err
}

// ActiveForRun returns the active leases for one run and prunes expired leases
// under the same lock. Read projections use this instead of historical index
// pin metadata, ensuring the UI cannot advertise expired protection.
func (s *PinLeaseStore) ActiveForRun(runID string, now time.Time) ([]PinLease, error) {
	// No lease file means no run here was ever pinned: there is nothing to read
	// and nothing to prune. Short-circuit before withLock, which would otherwise
	// create coverage/ just to answer a query — the way phantom scenario
	// directories were being created.
	if _, err := os.Stat(s.path); os.IsNotExist(err) {
		return nil, nil
	}
	var active []PinLease
	err := s.withLock(func() error {
		leases, err := s.readUnlocked()
		if err != nil {
			return err
		}
		kept := leases[:0]
		for _, lease := range leases {
			if lease.ActiveAt(now) {
				kept = append(kept, lease)
				if lease.RunID == runID {
					active = append(active, lease)
				}
			}
		}
		if len(kept) != len(leases) {
			return s.writeUnlocked(kept)
		}
		return nil
	})
	return active, err
}

func (s *PinLeaseStore) withLock(fn func() error) error {
	if err := os.MkdirAll(filepath.Dir(s.lockPath), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(s.lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	unlock, err := lockFile(file)
	if err != nil {
		return err
	}
	defer unlock()
	return fn()
}

func (s *PinLeaseStore) readUnlocked() ([]PinLease, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) || len(data) == 0 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var leases []PinLease
	if err := json.Unmarshal(data, &leases); err != nil {
		return nil, fmt.Errorf("parse pin leases: %w", err)
	}
	return leases, nil
}

func (s *PinLeaseStore) writeUnlocked(leases []PinLease) error {
	data, err := json.MarshalIndent(leases, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
