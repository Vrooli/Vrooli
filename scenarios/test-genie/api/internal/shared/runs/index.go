package runs

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	sharedartifacts "test-genie/internal/shared/artifacts"

	"github.com/vrooli/freshness-go/runindex"
)

// The record types and status vocabulary stored in the index are the shared
// read contract (github.com/vrooli/freshness-go/runindex) so freshness
// verdicts and cached status readers decode exactly what this write side
// stores. This package keeps the write/locking side internal and aliases the
// shared types.
type (
	// DiagnosticsConfig is the serialized diagnostics profile a run was
	// executed with. It is immutable once the run completes (Decision 4 in
	// Plan A).
	DiagnosticsConfig = runindex.DiagnosticsConfig
	// PinRecord protects a run from retention GC while an external consumer
	// (e.g. a git-control-tower baseline) references it.
	PinRecord = runindex.PinRecord
	// PhaseRecord is a compact per-phase summary stored in the index for fast
	// enumeration without reading per-run phase-results files.
	PhaseRecord = runindex.PhaseRecord
	// RunRecord is the index entry for a single test-genie execution.
	RunRecord = runindex.RunRecord
)

// Run status values recorded in the index.
const (
	StatusInProgress = runindex.StatusInProgress
	StatusPassed     = runindex.StatusPassed
	StatusFailed     = runindex.StatusFailed
	StatusAborted    = runindex.StatusAborted
)

// ErrRunNotFound is returned when a run ID is absent from the index.
var ErrRunNotFound = errors.New("run not found in index")

// ErrRunPinned is returned when a delete is attempted on a pinned run without force.
var ErrRunPinned = errors.New("run is pinned; unpin or use force to delete")

// Index is the append-only run index backed by coverage/runs.index.json. All
// mutations serialize through an advisory flock on a sibling lock file so
// concurrent test-genie processes (multiple agents on one box) cannot corrupt
// it.
type Index struct {
	path     string
	lockPath string
}

// NewIndex returns the Index for a scenario directory.
func NewIndex(scenarioDir string) *Index {
	path := sharedartifacts.RunsIndexPath(scenarioDir)
	return &Index{path: path, lockPath: path + ".lock"}
}

// Path returns the absolute path to the index file.
func (i *Index) Path() string { return i.path }

// withLock runs fn while holding an exclusive advisory lock on the index.
func (i *Index) withLock(fn func() error) error {
	if err := os.MkdirAll(filepath.Dir(i.lockPath), 0o755); err != nil {
		return fmt.Errorf("create index dir: %w", err)
	}
	lf, err := os.OpenFile(i.lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("open index lock: %w", err)
	}
	defer lf.Close()

	if err := syscall.Flock(int(lf.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("acquire index lock: %w", err)
	}
	defer func() { _ = syscall.Flock(int(lf.Fd()), syscall.LOCK_UN) }()

	return fn()
}

// readUnlocked loads the records without locking (callers hold the lock).
func (i *Index) readUnlocked() ([]RunRecord, error) {
	data, err := os.ReadFile(i.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read index: %w", err)
	}
	if len(data) == 0 {
		return nil, nil
	}
	var records []RunRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("parse index: %w", err)
	}
	return records, nil
}

// writeUnlocked atomically replaces the index file (callers hold the lock).
func (i *Index) writeUnlocked(records []RunRecord) error {
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal index: %w", err)
	}
	tmp := i.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write index tmp: %w", err)
	}
	if err := os.Rename(tmp, i.path); err != nil {
		return fmt.Errorf("replace index: %w", err)
	}
	return nil
}

// List returns all run records sorted newest-first (by StartedAt, then RunID).
func (i *Index) List() ([]RunRecord, error) {
	var records []RunRecord
	err := i.withLock(func() error {
		loaded, err := i.readUnlocked()
		if err != nil {
			return err
		}
		records = loaded
		return nil
	})
	if err != nil {
		return nil, err
	}
	runindex.SortNewestFirst(records)
	return records, nil
}

// Find returns the record for a run ID, or ErrRunNotFound.
func (i *Index) Find(runID string) (RunRecord, error) {
	var found RunRecord
	var ok bool
	err := i.withLock(func() error {
		records, err := i.readUnlocked()
		if err != nil {
			return err
		}
		for _, r := range records {
			if r.RunID == runID {
				found, ok = r, true
				return nil
			}
		}
		return nil
	})
	if err != nil {
		return RunRecord{}, err
	}
	if !ok {
		return RunRecord{}, ErrRunNotFound
	}
	return found, nil
}

// Append adds a new record. If a record with the same RunID already exists it
// is replaced (idempotent upsert for the start-then-finalize lifecycle).
func (i *Index) Append(rec RunRecord) error {
	return i.withLock(func() error {
		records, err := i.readUnlocked()
		if err != nil {
			return err
		}
		for idx, r := range records {
			if r.RunID == rec.RunID {
				records[idx] = rec
				return i.writeUnlocked(records)
			}
		}
		records = append(records, rec)
		return i.writeUnlocked(records)
	})
}

// Update mutates an existing record under lock. Returns ErrRunNotFound if the
// run is absent.
func (i *Index) Update(runID string, mutate func(*RunRecord) error) error {
	return i.withLock(func() error {
		records, err := i.readUnlocked()
		if err != nil {
			return err
		}
		for idx := range records {
			if records[idx].RunID == runID {
				if err := mutate(&records[idx]); err != nil {
					return err
				}
				return i.writeUnlocked(records)
			}
		}
		return ErrRunNotFound
	})
}

// Remove deletes a record by run ID. Missing runs are a no-op.
func (i *Index) Remove(runID string) error {
	return i.withLock(func() error {
		records, err := i.readUnlocked()
		if err != nil {
			return err
		}
		filtered := records[:0]
		for _, r := range records {
			if r.RunID != runID {
				filtered = append(filtered, r)
			}
		}
		return i.writeUnlocked(filtered)
	})
}

// Pin adds a pin to a run, protecting it from retention GC. Re-pinning by the
// same PinnedBy is idempotent (updates the reason/timestamp).
func (i *Index) Pin(runID string, pin PinRecord) error {
	return i.Update(runID, func(r *RunRecord) error {
		for idx := range r.Pins {
			if r.Pins[idx].PinnedBy == pin.PinnedBy {
				r.Pins[idx] = pin
				return nil
			}
		}
		r.Pins = append(r.Pins, pin)
		return nil
	})
}

// Unpin removes the pin owned by pinnedBy. Absent pins are a no-op.
func (i *Index) Unpin(runID, pinnedBy string) error {
	return i.Update(runID, func(r *RunRecord) error {
		filtered := r.Pins[:0]
		for _, p := range r.Pins {
			if p.PinnedBy != pinnedBy {
				filtered = append(filtered, p)
			}
		}
		r.Pins = filtered
		return nil
	})
}
