package runs

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	sharedartifacts "test-genie/internal/shared/artifacts"
)

// RetentionPolicy bounds how many runs and how much disk the run history keeps.
// Pinned runs are always retained regardless of these limits.
type RetentionPolicy struct {
	KeepMostRecent   int  // keep at least this many most-recent unpinned runs (0 = unlimited)
	KeepMaxAgeDays   int  // drop unpinned runs older than this (0 = no age limit)
	KeepMaxSizeMB    int  // drop oldest unpinned runs once total size exceeds this (0 = no size limit)
	AlwaysKeepPinned bool // always true; pinned runs are never GC'd
}

// DefaultRetentionPolicy returns the standard retention bounds (Plan A §1.5).
func DefaultRetentionPolicy() RetentionPolicy {
	return RetentionPolicy{
		KeepMostRecent:   20,
		KeepMaxAgeDays:   30,
		KeepMaxSizeMB:    5000,
		AlwaysKeepPinned: true,
	}
}

// GC enforces the retention policy against the run index and on-disk run
// directories under scenarioDir. It never deletes pinned runs. It returns the
// run IDs that were removed. GC is safe to call concurrently with executions
// because all index mutations serialize through the index flock.
func GC(ctx context.Context, scenarioDir string, policy RetentionPolicy) ([]string, error) {
	idx := NewIndex(scenarioDir)
	runs, err := idx.List() // newest-first
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var deleted []string

	// kept tracks the unpinned runs that survived the count/age pass, in
	// newest-first order, so the size-cap pass can drop the oldest first.
	type keptRun struct {
		id   string
		size int64
	}
	var kept []keptRun
	var keptSize int64
	keptUnpinned := 0

	for _, r := range runs {
		if err := ctx.Err(); err != nil {
			return deleted, err
		}
		size := dirSizeBytes(sharedartifacts.RunDir(scenarioDir, r.RunID))

		if r.IsPinned() {
			keptSize += size
			continue
		}

		keptUnpinned++
		tooOld := policy.KeepMaxAgeDays > 0 && now.Sub(r.StartedAt) > time.Duration(policy.KeepMaxAgeDays)*24*time.Hour
		overCount := policy.KeepMostRecent > 0 && keptUnpinned > policy.KeepMostRecent
		if tooOld || overCount {
			if err := deleteRun(idx, scenarioDir, r.RunID); err == nil {
				deleted = append(deleted, r.RunID)
			}
			continue
		}

		keptSize += size
		kept = append(kept, keptRun{id: r.RunID, size: size})
	}

	// Size cap: drop oldest surviving unpinned runs until under the cap.
	if policy.KeepMaxSizeMB > 0 {
		capBytes := int64(policy.KeepMaxSizeMB) * 1024 * 1024
		for i := len(kept) - 1; i >= 0 && keptSize > capBytes; i-- {
			if err := deleteRun(idx, scenarioDir, kept[i].id); err == nil {
				deleted = append(deleted, kept[i].id)
				keptSize -= kept[i].size
			}
		}
	}

	return deleted, nil
}

// DeleteRun removes a single run's artifacts and index entry. It refuses to
// delete a pinned run unless force is true.
func DeleteRun(scenarioDir, runID string, force bool) error {
	idx := NewIndex(scenarioDir)
	rec, err := idx.Find(runID)
	if err != nil {
		return err
	}
	if rec.IsPinned() && !force {
		return ErrRunPinned
	}
	return deleteRun(idx, scenarioDir, runID)
}

// deleteRun removes the run directory and its index entry.
func deleteRun(idx *Index, scenarioDir, runID string) error {
	if err := os.RemoveAll(sharedartifacts.RunDir(scenarioDir, runID)); err != nil {
		return err
	}
	return idx.Remove(runID)
}

// dirSizeBytes returns the total size of a directory tree, or 0 if missing.
func dirSizeBytes(dir string) int64 {
	var total int64
	_ = filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if info, infoErr := d.Info(); infoErr == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}
