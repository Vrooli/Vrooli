// Package repositorymigration owns the resumable, fail-closed workflow for
// moving a plan to a fresh repository. It never chooses a backend or location:
// callers must provide an operator-selected destination and an owner that can
// prove capacity and credentials. Effects are idempotent by migration/target.
package repositorymigration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const RequiredTargetCount = 14

type Stage string

const (
	StageCapacity Stage = "capacity_proof"
	StageCapture  Stage = "capturing"
	StageDrill    Stage = "drilling"
	StageSwitch   Stage = "switching_plan_bindings"
	StageComplete Stage = "complete"
	StageFailed   Stage = "failed"
)

type CaptureReceipt struct {
	TargetID, DestinationID, SnapshotID string
	Bytes                               int64
	Verified                            bool
}
type DrillReceipt struct {
	TargetID, DestinationID, SnapshotID, RestoreID string
	Verified                                       bool
	ScratchReceiptID                               string
}
type CapacityProof struct {
	PlanID, OldDestinationID, DestinationID string
	TargetIDs                               []string
	AvailableBytes, RequiredBytes           int64
	// These are non-secret identity bindings. CredentialAuthority names the
	// authority/reference that proves access; it must never contain key material.
	DeviceIdentity, CredentialAuthority string
	MeasuredAt                          time.Time
	ExpiresAt                           time.Time
	// Receipt is an opaque owner evidence identifier, not credential material.
	Receipt string
}
type Item struct {
	TargetID string
	Capture  *CaptureReceipt `json:"capture,omitempty"`
	Drill    *DrillReceipt   `json:"drill,omitempty"`
}
type Migration struct {
	ID               string         `json:"id"`
	PlanID           string         `json:"plan_id"`
	OldDestinationID string         `json:"old_destination_id"`
	NewDestinationID string         `json:"new_destination_id"`
	Targets          []string       `json:"targets"`
	Stage            Stage          `json:"stage"`
	Capacity         *CapacityProof `json:"capacity,omitempty"`
	Items            []Item         `json:"items"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	Error            string         `json:"error,omitempty"`
}

type Store interface {
	Get(context.Context, string) (Migration, bool, error)
	Put(context.Context, Migration) error
}
type Effects interface {
	ProveCapacity(context.Context, string, string, string, []string, bool) (CapacityProof, error)
	Capture(context.Context, string, string, string, string) (CaptureReceipt, error)
	Drill(context.Context, string, string, string, string, string) (DrillReceipt, error)
	SwitchBindings(context.Context, string, string, string, []string, string) error
}
type (
	Clock       interface{ Now() time.Time }
	systemClock struct{}
)

func (systemClock) Now() time.Time { return time.Now() }

type Service struct {
	store   Store
	effects Effects
	clock   Clock
	mu      sync.Mutex
}

func New(store Store, effects Effects, clock Clock) *Service {
	if clock == nil {
		clock = systemClock{}
	}
	return &Service{store: store, effects: effects, clock: clock}
}

// Begin accepts only the explicit old/new repository identities and exact
// target set captured from the approved plan. It persists before performing any
// side effect; replay returns the same migration when its identity matches.
func (s *Service) Begin(ctx context.Context, id, planID, oldDestinationID, newDestinationID string, targets []string) (Migration, error) {
	if s == nil {
		return Migration{}, errors.New("migration owner is not configured")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id, planID, oldDestinationID, newDestinationID = strings.TrimSpace(id), strings.TrimSpace(planID), strings.TrimSpace(oldDestinationID), strings.TrimSpace(newDestinationID)
	targets = canonical(targets)
	if id == "" || planID == "" || oldDestinationID == "" || newDestinationID == "" || oldDestinationID == newDestinationID || len(targets) != RequiredTargetCount || !unique(targets) {
		return Migration{}, errors.New("migration requires explicit distinct repositories, plan, and exactly 14 unique targets")
	}
	if s == nil || s.store == nil || s.effects == nil {
		return Migration{}, errors.New("migration owner is not configured")
	}
	if got, ok, err := s.store.Get(ctx, id); err != nil {
		return Migration{}, err
	} else if ok {
		if got.PlanID != planID || got.OldDestinationID != oldDestinationID || got.NewDestinationID != newDestinationID || !equal(got.Targets, targets) {
			return Migration{}, errors.New("migration id already belongs to different inputs")
		}
		return got, nil
	}
	now := s.clock.Now().UTC()
	m := Migration{ID: id, PlanID: planID, OldDestinationID: oldDestinationID, NewDestinationID: newDestinationID, Targets: targets, Stage: StageCapacity, CreatedAt: now, UpdatedAt: now, Items: make([]Item, len(targets))}
	for i, t := range targets {
		m.Items[i].TargetID = t
	}
	if err := s.store.Put(ctx, m); err != nil {
		return Migration{}, err
	}
	return m, nil
}

// Resume runs only forward. Every external action has a stable idempotency key;
// the plan switch is last and requires all 14 exact target capture and drill
// receipts to be verified. The old destination is never written by this owner.
func (s *Service) Resume(ctx context.Context, id string) (Migration, error) {
	if s == nil {
		return Migration{}, errors.New("migration owner is not configured")
	}
	if s.store == nil || s.effects == nil {
		return Migration{}, errors.New("migration owner is not configured")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok, err := s.store.Get(ctx, id)
	if err != nil {
		return Migration{}, err
	}
	if !ok {
		return Migration{}, fmt.Errorf("migration %q not found", id)
	}
	if m.Stage == StageComplete {
		return m, nil
	}
	if m.Stage == StageFailed {
		return m, errors.New(m.Error)
	}
	fail := func(cause error) (Migration, error) {
		m.Error = cause.Error()
		m.UpdatedAt = s.clock.Now().UTC()
		if persistErr := s.store.Put(context.Background(), m); persistErr != nil {
			return m, errors.Join(cause, fmt.Errorf("persist migration failure state: %w", persistErr))
		}
		return m, cause
	}
	m.Error = ""
	if m.Stage == StageCapacity {
		proof, e := s.effects.ProveCapacity(ctx, m.PlanID, m.OldDestinationID, m.NewDestinationID, append([]string(nil), m.Targets...), false)
		if e != nil {
			return fail(fmt.Errorf("capacity proof: %w", e))
		}
		if !validCapacityProof(proof, m.PlanID, m.OldDestinationID, m.NewDestinationID, m.Targets, s.clock.Now()) {
			return fail(errors.New("capacity proof is incomplete, stale, or insufficient; no plan bindings changed"))
		}
		m.Capacity = &proof
		m.Stage = StageCapture
		if e = s.save(ctx, &m); e != nil {
			return m, e
		}
	}
	if m.Stage == StageCapture {
		for i := range m.Items {
			item := &m.Items[i]
			if item.Capture != nil && item.Capture.Verified {
				continue
			}
			receipt, e := s.effects.Capture(ctx, m.PlanID, item.TargetID, m.NewDestinationID, key(m.ID, "capture", item.TargetID))
			if e != nil {
				return fail(fmt.Errorf("capture target %s: %w", item.TargetID, e))
			}
			if receipt.TargetID != item.TargetID || receipt.DestinationID != m.NewDestinationID || strings.TrimSpace(receipt.SnapshotID) == "" || receipt.Bytes < 0 || !receipt.Verified {
				return fail(fmt.Errorf("capture target %s returned an invalid receipt", item.TargetID))
			}
			item.Capture = &receipt
			if e = s.save(ctx, &m); e != nil {
				return m, e
			}
		}
		m.Stage = StageDrill
		if err = s.save(ctx, &m); err != nil {
			return m, err
		}
	}
	if m.Stage == StageDrill {
		for i := range m.Items {
			item := &m.Items[i]
			if item.Drill != nil && item.Drill.Verified {
				continue
			}
			receipt, e := s.effects.Drill(ctx, m.PlanID, item.TargetID, m.NewDestinationID, item.Capture.SnapshotID, key(m.ID, "drill", item.TargetID))
			if e != nil {
				return fail(fmt.Errorf("drill target %s: %w", item.TargetID, e))
			}
			if receipt.TargetID != item.TargetID || receipt.DestinationID != m.NewDestinationID || receipt.SnapshotID != item.Capture.SnapshotID || strings.TrimSpace(receipt.RestoreID) == "" || strings.TrimSpace(receipt.ScratchReceiptID) == "" || !receipt.Verified {
				return fail(fmt.Errorf("drill target %s returned an invalid scratch-restore receipt", item.TargetID))
			}
			item.Drill = &receipt
			if e = s.save(ctx, &m); e != nil {
				return m, e
			}
		}
		m.Stage = StageSwitch
		if err = s.save(ctx, &m); err != nil {
			return m, err
		}
	}
	if m.Stage == StageSwitch {
		if !allVerified(m) {
			return fail(errors.New("plan switch refused: all 14 target capture and scratch-drill receipts are required"))
		}
		// Captures and drills may take hours. Re-establish capacity immediately
		// before changing plan bindings instead of relying on a pre-capture read.
		proof, proofErr := s.effects.ProveCapacity(ctx, m.PlanID, m.OldDestinationID, m.NewDestinationID, append([]string(nil), m.Targets...), true)
		if proofErr != nil {
			return fail(fmt.Errorf("final capacity proof: %w", proofErr))
		}
		if !validCapacityProof(proof, m.PlanID, m.OldDestinationID, m.NewDestinationID, m.Targets, s.clock.Now()) || m.Capacity == nil ||
			proof.DeviceIdentity != m.Capacity.DeviceIdentity || proof.CredentialAuthority != m.Capacity.CredentialAuthority {
			return fail(errors.New("final capacity proof is incomplete, stale, or insufficient; plan bindings remain unchanged"))
		}
		m.Capacity = &proof
		if err = s.save(ctx, &m); err != nil {
			return m, err
		}
		if err = s.effects.SwitchBindings(ctx, m.PlanID, m.OldDestinationID, m.NewDestinationID, append([]string(nil), m.Targets...), key(m.ID, "switch", m.PlanID)); err != nil {
			return fail(fmt.Errorf("switch plan bindings: %w", err))
		}
		m.Stage = StageComplete
		if err = s.save(ctx, &m); err != nil {
			return m, err
		}
	}
	return m, nil
}

func (s *Service) save(ctx context.Context, m *Migration) error {
	m.UpdatedAt = s.clock.Now().UTC()
	return s.store.Put(ctx, *m)
}

func validCapacityProof(proof CapacityProof, planID, oldDestinationID, destinationID string, targets []string, now time.Time) bool {
	return proof.PlanID == planID && proof.OldDestinationID == oldDestinationID && proof.DestinationID == destinationID && equal(canonical(proof.TargetIDs), canonical(targets)) && proof.AvailableBytes > 0 && proof.RequiredBytes > 0 &&
		proof.AvailableBytes >= proof.RequiredBytes && strings.TrimSpace(proof.DeviceIdentity) != "" &&
		strings.TrimSpace(proof.CredentialAuthority) != "" && strings.TrimSpace(proof.Receipt) != "" &&
		!proof.MeasuredAt.IsZero() && !proof.MeasuredAt.After(now) && proof.ExpiresAt.After(now) && proof.ExpiresAt.After(proof.MeasuredAt)
}

func allVerified(m Migration) bool {
	if len(m.Items) != RequiredTargetCount {
		return false
	}
	for _, i := range m.Items {
		if i.Capture == nil || !i.Capture.Verified || i.Drill == nil || !i.Drill.Verified || i.Drill.SnapshotID != i.Capture.SnapshotID {
			return false
		}
	}
	return true
}

func key(id, operation, target string) string {
	sum := sha256.Sum256([]byte(id + "\x00" + operation + "\x00" + target))
	return "db4-" + hex.EncodeToString(sum[:])
}

func canonical(xs []string) []string {
	out := append([]string(nil), xs...)
	for i := range out {
		out[i] = strings.TrimSpace(out[i])
	}
	sort.Strings(out)
	return out
}

func unique(xs []string) bool {
	for i, x := range xs {
		if x == "" || (i > 0 && x == xs[i-1]) {
			return false
		}
	}
	return true
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] == "" || a[i] != b[i] || (i > 0 && a[i] == a[i-1]) {
			return false
		}
	}
	return true
}

// FileStore stores independent migration journals below a DBM-owned state
// root. Each journal is atomically replaced so a crash cannot mix two
// migrations or truncate the prior receipt.
type FileStore struct {
	Root string
	mu   sync.Mutex
}

func (f *FileStore) path(id string) (string, error) {
	if f == nil || !filepath.IsAbs(f.Root) || strings.TrimSpace(id) == "" || strings.ContainsAny(id, `/\\`) {
		return "", errors.New("invalid migration store or id")
	}
	return filepath.Join(filepath.Clean(f.Root), id+".json"), nil
}

func (f *FileStore) Get(ctx context.Context, id string) (Migration, bool, error) {
	if err := ctx.Err(); err != nil {
		return Migration{}, false, err
	}
	p, err := f.path(id)
	if err != nil {
		return Migration{}, false, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	b, e := os.ReadFile(p)
	if errors.Is(e, os.ErrNotExist) {
		return Migration{}, false, nil
	}
	if e != nil {
		return Migration{}, false, e
	}
	var m Migration
	if e = json.Unmarshal(b, &m); e != nil {
		return Migration{}, false, e
	}
	if m.ID != id {
		return Migration{}, false, errors.New("migration journal identity mismatch")
	}
	return m, true, nil
}

func (f *FileStore) Put(ctx context.Context, m Migration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p, err := f.path(m.ID)
	if err != nil {
		return err
	}
	b, e := json.MarshalIndent(m, "", "  ")
	if e != nil {
		return e
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if e = os.MkdirAll(filepath.Dir(p), 0o700); e != nil {
		return e
	}
	tmp, e := os.CreateTemp(filepath.Dir(p), ".db4-migration-*")
	if e != nil {
		return e
	}
	name := tmp.Name()
	defer os.Remove(name)
	if e = tmp.Chmod(0o600); e == nil {
		_, e = tmp.Write(b)
	}
	if e == nil {
		e = tmp.Sync()
	}
	if closeErr := tmp.Close(); e == nil {
		e = closeErr
	}
	if e != nil {
		return e
	}
	if e = os.Rename(name, p); e != nil {
		return e
	}
	dir, e := os.Open(filepath.Dir(p))
	if e != nil {
		return e
	}
	defer dir.Close()
	return dir.Sync()
}
