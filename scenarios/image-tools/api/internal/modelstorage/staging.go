package modelstorage

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"image-tools/internal/fetch"
)

// Staging states.
const (
	// StageActive: an acquisition in this process owns the directory.
	StageActive = "active"
	// StageFresh: unowned but written within the grace period; it may belong to
	// another process or be resumed shortly.
	StageFresh = "fresh"
	// StageAbandoned: unowned and silent for the grace period; reclaimable.
	StageAbandoned = "abandoned"
	// StageRecoverable: a replaced installation set aside whose destination is
	// missing (or unknown), so deleting it could destroy the only copy.
	StageRecoverable = "recoverable"
)

// DefaultStagingGrace is how long an unowned staging directory must go
// without any write before it counts as abandoned. A live download writes
// continuously, so two hours of silence is conclusive while still leaving a
// retried install time to resume the partial snapshot.
const DefaultStagingGrace = 2 * time.Hour

// DefaultStagingBootDelay lets durable jobs requeued by a restart claim and
// resume their partial staging before the boot sweep judges it.
const DefaultStagingBootDelay = 2 * time.Minute

// StagingEntry is one acquisition staging directory on disk.
type StagingEntry struct {
	Path      string
	Bytes     int64
	LastWrite time.Time
	State     string
	// Reason explains the state for logs and reports.
	Reason string
}

// StagingSummary totals staging directories for the storage report.
type StagingSummary struct {
	GraceSeconds   int64
	TotalBytes     int64
	AbandonedCount int
	AbandonedBytes int64
}

// stagingRoots are the directories scanned for acquisition staging: the
// dedicated staging directory, plus the weights directories where earlier
// releases created staging beside the weights they published.
func (s *Service) stagingRoots() []string {
	if len(s.deps.StagingRoots) > 0 {
		return s.deps.StagingRoots
	}
	root := s.deps.Root
	return []string{
		filepath.Join(root, fetch.StagingDirName, "models"), filepath.Join(root, fetch.StagingDirName, "adapters"),
		filepath.Join(root, "models"), filepath.Join(root, "adapters"),
	}
}

// Staging lists every staging directory with its ownership state.
func (s *Service) Staging(ctx context.Context) ([]StagingEntry, error) {
	var out []StagingEntry
	var errs []error
	for _, root := range s.stagingRoots() {
		entries, err := os.ReadDir(root)
		if err != nil {
			if !os.IsNotExist(err) {
				errs = append(errs, fmt.Errorf("modelstorage: list staging in %s: %w", root, err))
			}
			continue
		}
		for _, e := range entries {
			if ctx.Err() != nil {
				return out, ctx.Err()
			}
			if !e.IsDir() || !fetch.IsStageName(e.Name()) {
				continue
			}
			entry, err := s.inspectStage(filepath.Join(root, e.Name()), false)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			out = append(out, entry)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, errors.Join(errs...)
}

// inspectStage classifies path. heldBySweep is true when the caller already
// took the ownership hold to re-check the directory before removing it.
func (s *Service) inspectStage(path string, heldBySweep bool) (StagingEntry, error) {
	bytes, err := MeasureAllocated(path)
	if err != nil {
		return StagingEntry{}, fmt.Errorf("modelstorage: measure staging %s: %w", path, err)
	}
	last, err := lastWrite(path)
	if err != nil {
		return StagingEntry{}, fmt.Errorf("modelstorage: inspect staging %s: %w", path, err)
	}
	entry := StagingEntry{Path: path, Bytes: bytes, LastWrite: last}
	silent := s.deps.Now().Sub(last)
	switch {
	case !heldBySweep && fetch.StageHeld(ownerKey(path)):
		entry.State, entry.Reason = StageActive, "owned by an acquisition in this process"
	case silent < s.stagingGrace():
		entry.State, entry.Reason = StageFresh, fmt.Sprintf("written %s ago, within the %s grace period", silent.Round(time.Second), s.stagingGrace())
	case isPrevious(path) && !destinationExists(path):
		entry.State, entry.Reason = StageRecoverable, "replaced installation whose destination is missing or unknown; kept for recovery"
	default:
		entry.State, entry.Reason = StageAbandoned, fmt.Sprintf("no owner and no write for %s (grace %s)", silent.Round(time.Second), s.stagingGrace())
	}
	return entry, nil
}

// ownerKey is the registry key owning path: a set-aside belongs to the
// acquisition that holds its staging directory.
func ownerKey(path string) string { return strings.TrimSuffix(path, fetch.PreviousSuffix) }

func isPrevious(path string) bool { return fetch.IsPreviousName(filepath.Base(path)) }

// destinationExists is true only when the staged name maps to a destination
// that is currently present, so a replaced copy is never the sole one.
func destinationExists(path string) bool {
	dest, ok := fetch.StageDestination(path)
	if !ok {
		return false
	}
	_, err := os.Lstat(dest)
	return err == nil
}

// lastWrite is the newest modification time of the directory and everything
// in it, without following symlinks.
func lastWrite(root string) (time.Time, error) {
	var newest time.Time
	err := filepath.WalkDir(root, func(_ string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if errors.Is(walkErr, fs.ErrNotExist) {
				return nil // removed while walking: a live write path, skip it.
			}
			return walkErr
		}
		info, err := d.Info()
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if info.ModTime().After(newest) {
			newest = info.ModTime()
		}
		return nil
	})
	return newest, err
}

func (s *Service) stagingGrace() time.Duration {
	if s.cfg.StagingGrace > 0 {
		return s.cfg.StagingGrace
	}
	return DefaultStagingGrace
}

func summarizeStaging(entries []StagingEntry, grace time.Duration) StagingSummary {
	sum := StagingSummary{GraceSeconds: int64(grace / time.Second)}
	for _, e := range entries {
		sum.TotalBytes += e.Bytes
		if e.State == StageAbandoned {
			sum.AbandonedCount++
			sum.AbandonedBytes += e.Bytes
		}
	}
	return sum
}

// SweepStaging reclaims abandoned staging directories and returns the ones it
// removed (or, with dryRun, would remove). A directory is removed only while
// this process holds it in the ownership registry and a fresh inspection under
// that hold still finds it abandoned, so a resuming acquisition and a sweep
// can never act on the same directory at once.
func (s *Service) SweepStaging(ctx context.Context, trigger string, dryRun bool) ([]StagingEntry, error) {
	s.sweepMu.Lock()
	defer s.sweepMu.Unlock()
	entries, err := s.Staging(ctx)
	var reclaimed []StagingEntry
	var errs []error
	if err != nil {
		errs = append(errs, err)
	}
	for _, e := range entries {
		if e.State != StageAbandoned {
			continue
		}
		if dryRun {
			s.logReclaim(trigger, e, "would reclaim")
			reclaimed = append(reclaimed, e)
			continue
		}
		if !fetch.HoldStage(ownerKey(e.Path)) {
			continue // an acquisition claimed it after the listing
		}
		current, inspectErr := s.inspectStage(e.Path, true)
		switch {
		case inspectErr != nil:
			errs = append(errs, inspectErr)
		case current.State != StageAbandoned:
			// written or reclaimed since the listing
		default:
			if rmErr := os.RemoveAll(e.Path); rmErr != nil {
				errs = append(errs, fmt.Errorf("modelstorage: reclaim staging %s: %w", e.Path, rmErr))
			} else {
				s.logReclaim(trigger, current, "reclaimed")
				reclaimed = append(reclaimed, current)
			}
		}
		fetch.ReleaseStage(ownerKey(e.Path))
	}
	return reclaimed, errors.Join(errs...)
}

func (s *Service) logReclaim(trigger string, e StagingEntry, verb string) {
	s.deps.Logger.Printf("modelstorage: %s abandoned staging %s bytes=%d last_write=%s trigger=%s reason=%q",
		verb, e.Path, e.Bytes, e.LastWrite.UTC().Format(time.RFC3339), trigger, e.Reason)
}

// stagingText names a reclaim result for pass logs.
func stagingText(entries []StagingEntry) string {
	var total int64
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		total += e.Bytes
		names = append(names, filepath.Base(e.Path))
	}
	return fmt.Sprintf("count=%d bytes=%d [%s]", len(entries), total, strings.Join(names, " "))
}
