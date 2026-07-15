package opsrunner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// OrphanSweepReport summarizes an orphan-snapshot reconciliation sweep.
type OrphanSweepReport struct {
	// DirsScanned is the number of agentops directories examined.
	DirsScanned int `json:"dirs_scanned"`
	// SnapshotsSeen is the total execution snapshot files found across those dirs.
	SnapshotsSeen int `json:"snapshots_seen"`
	// Reaped lists the execution snapshot file paths removed (orphans past the
	// grace period).
	Reaped []string `json:"reaped"`
	// SkippedTooRecent counts orphan snapshots left in place because they were
	// younger than the grace period (a concurrent in-flight Invoke may still be
	// between writing its snapshot and committing its workflow operation record).
	SkippedTooRecent int `json:"skipped_too_recent"`
}

// ReconcileOrphanSnapshots removes execution snapshots on disk that no workflow
// operation record references. Such an orphan is written by a concurrent Invoke
// that lost the workflow compare-and-swap: the snapshot is persisted (Invoke
// step 6) BEFORE the running operation record is appended (step 7), so a CAS
// conflict returns an error while leaving the snapshot file behind. The workflow
// state itself stays correct (exactly one winning running op); the orphan is pure
// disk waste.
//
// It is fail-honest and race-safe:
//   - A snapshot whose execution id appears in the sibling workflow.json is
//     preserved (it is a live or historical operation record).
//   - An orphan younger than minAge is SKIPPED, because an in-flight Invoke may be
//     between writing its snapshot and committing its operation record; only
//     orphans older than the grace period are reaped.
//   - A dir with no workflow.json treats every snapshot as unreferenced (an
//     abandoned scope root), still gated by minAge.
//
// Deleting only aged, unreferenced snapshots makes it safe to run at startup or
// live. A malformed workflow.json is treated conservatively (nothing reaped for
// that dir) rather than risking deletion of a referenced snapshot.
func ReconcileOrphanSnapshots(loc DomainLocator, now time.Time, minAge time.Duration) (OrphanSweepReport, error) {
	dirs, err := loc.Scan()
	if err != nil {
		return OrphanSweepReport{}, err
	}
	var report OrphanSweepReport
	for _, dir := range dirs {
		report.DirsScanned++
		execDir := filepath.Join(dir, executionsSubdir)
		entries, err := os.ReadDir(execDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return OrphanSweepReport{}, err
		}

		referenced, workflowReadable := referencedExecutionIDs(filepath.Join(dir, workflowFile))
		if !workflowReadable {
			// A workflow.json that exists but does not parse is a corruption signal;
			// skip the whole dir rather than risk deleting a referenced snapshot.
			continue
		}

		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
				continue
			}
			report.SnapshotsSeen++
			execID := strings.TrimSuffix(e.Name(), ".json")
			if referenced[execID] {
				continue
			}
			path := filepath.Join(execDir, e.Name())
			if snapshotAge(path, now) < minAge {
				report.SkippedTooRecent++
				continue
			}
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return OrphanSweepReport{}, err
			}
			report.Reaped = append(report.Reaped, path)
		}
	}
	return report, nil
}

// referencedExecutionIDs returns the set of execution ids the workflow at path
// references. workflowReadable is false only when the file exists but cannot be
// decoded (a corruption signal the caller treats conservatively); a missing file
// is readable-but-empty (every snapshot is unreferenced).
func referencedExecutionIDs(path string) (ids map[string]bool, workflowReadable bool) {
	ids = map[string]bool{}
	w, found, err := loadWorkflowFile(path)
	if err != nil {
		// The file exists but does not validate/decode: a corruption signal the
		// caller handles conservatively (reap nothing for this dir).
		return ids, false
	}
	if !found {
		// No workflow.json: readable-but-empty; every snapshot is unreferenced.
		return ids, true
	}
	for _, op := range w.Operations {
		if strings.TrimSpace(op.ExecutionID) != "" {
			ids[op.ExecutionID] = true
		}
	}
	return ids, true
}

// snapshotAge returns how long ago the snapshot at path was recorded, preferring
// its RecordedAt field and falling back to the file mtime. A snapshot that cannot
// be read/parsed is treated as maximally old so a corrupt orphan is reaped.
func snapshotAge(path string, now time.Time) time.Duration {
	raw, err := os.ReadFile(path)
	if err != nil {
		return maxAge
	}
	var snap struct {
		RecordedAt string `json:"recorded_at"`
	}
	if err := json.Unmarshal(raw, &snap); err == nil && strings.TrimSpace(snap.RecordedAt) != "" {
		if t, perr := time.Parse(time.RFC3339Nano, snap.RecordedAt); perr == nil {
			return now.Sub(t)
		}
	}
	if info, serr := os.Stat(path); serr == nil {
		return now.Sub(info.ModTime())
	}
	return maxAge
}

const maxAge = 1<<62 - 1 // effectively unbounded, so a corrupt/unreadable orphan is always reaped
