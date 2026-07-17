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
	// PinRecord exists only to decode historical index records. New retention
	// ownership is represented by PinLease in pin_leases.go.
	PinRecord = runindex.PinRecord
	// PhaseRecord is a compact per-phase summary stored in the index for fast
	// enumeration without reading per-run phase-results files.
	PhaseRecord = runindex.PhaseRecord
	// RunRecord is the index entry for a single test-genie execution.
	RunRecord = runindex.RunRecord
)

// Run status values recorded in the index.
const (
	StatusQueued     = runindex.StatusQueued
	StatusInProgress = runindex.StatusInProgress
	StatusPassed     = runindex.StatusPassed
	StatusFailed     = runindex.StatusFailed
	StatusAborted    = runindex.StatusAborted
)

// ErrRunNotFound is returned when a run ID is absent from the index.
var ErrRunNotFound = errors.New("run not found in index")

// ErrRunPinned is returned when a delete is attempted on a pinned run without force.
var ErrRunPinned = errors.New("run is pinned; unpin or use force to delete")

// Terminal snapshot errors are typed so callers can distinguish a legacy run
// from corrupt or future-version evidence without inventing empty fields.
var (
	ErrSnapshotNotFound           = errors.New("terminal run snapshot not found")
	ErrInvalidTerminalSnapshot    = errors.New("invalid terminal run snapshot")
	ErrUnsupportedSnapshotVersion = errors.New("unsupported terminal run snapshot version")
)

// TerminalSnapshotSchemaVersion is the first canonical durable terminal run
// contract. Additive evolution may retain this version; incompatible changes
// require a new version and an explicit reader migration.
const TerminalSnapshotSchemaVersion = 1

// TerminalSnapshot is the heavy, immutable terminal truth stored beside a
// run's artifacts. Run is the compact enumeration projection; Result preserves
// the complete orchestrator result without making the shared index package
// depend on the orchestrator package.
type TerminalSnapshot struct {
	SchemaVersion int             `json:"schema_version"`
	Run           RunRecord       `json:"run"`
	Result        json.RawMessage `json:"result"`
}

// Index is the append-only run index backed by coverage/runs.index.json. All
// mutations serialize through an advisory flock on a sibling lock file so
// concurrent test-genie processes (multiple agents on one box) cannot corrupt
// it.
type Index struct {
	scenarioDir string
	path        string
	lockPath    string
}

// NewIndex returns the Index for a scenario directory.
func NewIndex(scenarioDir string) *Index {
	path := sharedartifacts.RunsIndexPath(scenarioDir)
	return &Index{scenarioDir: scenarioDir, path: path, lockPath: path + ".lock"}
}

// Path returns the absolute path to the index file.
func (i *Index) Path() string { return i.path }

// ScenarioDir returns the scenario root owning this index and its run artifacts.
func (i *Index) ScenarioDir() string { return i.scenarioDir }

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

// Finalize atomically publishes the heavy terminal snapshot before flipping
// the compact index entry terminal, all under the same index lock. If snapshot
// serialization or persistence fails, the index remains non-terminal so no
// reader can mistake a partial finalization for complete evidence.
func (i *Index) Finalize(runID string, result any, mutate func(*RunRecord) error) error {
	return i.withLock(func() error {
		records, err := i.readUnlocked()
		if err != nil {
			return err
		}
		for idx := range records {
			if records[idx].RunID != runID {
				continue
			}
			if err := mutate(&records[idx]); err != nil {
				return err
			}
			if !isTerminalRecord(records[idx].Status) {
				return fmt.Errorf("%w: status %q is not terminal", ErrInvalidTerminalSnapshot, records[idx].Status)
			}
			if err := writeTerminalSnapshot(i.scenarioDir, records[idx], result); err != nil {
				return err
			}
			return i.writeUnlocked(records)
		}
		return ErrRunNotFound
	})
}

// ReadTerminalSnapshot loads and validates the canonical terminal snapshot for
// runID. A missing file identifies a pre-snapshot legacy run and is returned as
// ErrSnapshotNotFound; corrupt and future-version files remain distinct errors.
func (i *Index) ReadTerminalSnapshot(runID string) (TerminalSnapshot, error) {
	path := sharedartifacts.RunSnapshotPath(i.scenarioDir, runID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return TerminalSnapshot{}, ErrSnapshotNotFound
		}
		return TerminalSnapshot{}, fmt.Errorf("read terminal snapshot: %w", err)
	}
	var snapshot TerminalSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return TerminalSnapshot{}, fmt.Errorf("%w: decode: %v", ErrInvalidTerminalSnapshot, err)
	}
	if snapshot.SchemaVersion != TerminalSnapshotSchemaVersion {
		return TerminalSnapshot{}, fmt.Errorf("%w: got %d", ErrUnsupportedSnapshotVersion, snapshot.SchemaVersion)
	}
	if snapshot.Run.RunID != runID || !isTerminalRecord(snapshot.Run.Status) || len(snapshot.Result) == 0 || string(snapshot.Result) == "null" {
		return TerminalSnapshot{}, fmt.Errorf("%w: identity, status, or result is incomplete", ErrInvalidTerminalSnapshot)
	}
	return snapshot, nil
}

func writeTerminalSnapshot(scenarioDir string, record RunRecord, result any) error {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal terminal result: %w", err)
	}
	if string(resultJSON) == "null" {
		return fmt.Errorf("%w: result is nil", ErrInvalidTerminalSnapshot)
	}
	snapshot := TerminalSnapshot{
		SchemaVersion: TerminalSnapshotSchemaVersion,
		Run:           record,
		Result:        resultJSON,
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal terminal snapshot: %w", err)
	}
	path := sharedartifacts.RunSnapshotPath(scenarioDir, record.RunID)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create terminal snapshot dir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".run-snapshot-*.tmp")
	if err != nil {
		return fmt.Errorf("create terminal snapshot tmp: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write terminal snapshot tmp: %w", err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod terminal snapshot tmp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync terminal snapshot tmp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close terminal snapshot tmp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace terminal snapshot: %w", err)
	}
	if dirHandle, err := os.Open(dir); err == nil {
		_ = dirHandle.Sync()
		_ = dirHandle.Close()
	}
	return nil
}

func isTerminalRecord(status string) bool {
	switch status {
	case StatusPassed, StatusFailed, StatusAborted:
		return true
	default:
		return false
	}
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
