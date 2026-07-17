package runs

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"

	sharedartifacts "test-genie/internal/shared/artifacts"
)

// RetentionPolicy names every durable-evidence budget controlled by the
// lifecycle owner. Retention protection is lease-based: a lease must name an
// owner and expires automatically, so abandoned baseline references cannot
// retain evidence forever.
type RetentionPolicy struct {
	KeepMostRecent int // keep at least this many most-recent unleased runs (0 = unlimited)
	KeepMaxAgeDays int // drop unleased runs older than this (0 = no age limit)
	KeepMaxSizeMB  int // drop oldest unleased runs once total size exceeds this (0 = no size limit)
}

func DefaultRetentionPolicy() RetentionPolicy {
	return RetentionPolicy{KeepMostRecent: 20, KeepMaxAgeDays: 30, KeepMaxSizeMB: 5000}
}

// RetentionReport makes decisions inspectable before or after cleanup. It
// deliberately reports complete lifecycle footprints (artifact tree + log
// tree) rather than only the coverage/runs directory.
type RetentionReport struct {
	Eligible []string
	Deleted  []string
	Kept     []string
}

// DetailStore is the persistence side of the evidence lifecycle. The shared
// retention domain owns ordering and recovery; the execution package supplies
// the concrete SQLite implementation without creating an import cycle.
type DetailStore interface {
	DeleteByRunID(ctx context.Context, runID string) error
}

// RetentionService is the single authority for run-scoped filesystem lifecycle
// decisions. It writes a tombstone before deletion and reconciles unfinished
// tombstones before making a new decision, so a process crash converges safely
// on the next run without a separate cleanup script.
type RetentionService struct {
	scenarioDir string
	policy      RetentionPolicy
	index       *Index
	leases      *PinLeaseStore
	details     DetailStore
	now         func() time.Time
}

// WithDetailStore attaches the compact execution-history store to this
// lifecycle authority. It is intentionally a construction-time seam; callers
// must not delete execution rows independently of retention.
func (s *RetentionService) WithDetailStore(store DetailStore) *RetentionService {
	if s != nil {
		s.details = store
	}
	return s
}

func NewRetentionService(scenarioDir string, policy RetentionPolicy) *RetentionService {
	return &RetentionService{scenarioDir: scenarioDir, policy: policy, index: NewIndex(scenarioDir), leases: NewPinLeaseStore(scenarioDir), now: time.Now}
}

// Collect reconciles interrupted lifecycle work, computes retention eligibility,
// then deletes only terminal run footprints without an active protection lease.
func (s *RetentionService) Collect(ctx context.Context) (RetentionReport, error) {
	if s == nil || s.index == nil {
		return RetentionReport{}, fmt.Errorf("retention service is not configured")
	}
	if err := s.Reconcile(ctx); err != nil {
		return RetentionReport{}, err
	}
	runs, err := s.index.List()
	if err != nil {
		return RetentionReport{}, err
	}

	report := RetentionReport{}
	type keptRun struct {
		id   string
		size int64
	}
	kept := make([]keptRun, 0, len(runs))
	var keptSize int64
	keptUnpinned := 0
	now := s.now().UTC()

	for _, run := range runs {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		size := runFootprintBytes(s.scenarioDir, run.RunID)
		leased, err := s.leases.Active(run.RunID, now)
		if err != nil {
			return report, fmt.Errorf("check pin lease for %s: %w", run.RunID, err)
		}
		if leased {
			report.Kept = append(report.Kept, run.RunID)
			keptSize += size
			continue
		}
		keptUnpinned++
		tooOld := s.policy.KeepMaxAgeDays > 0 && now.Sub(run.StartedAt) > time.Duration(s.policy.KeepMaxAgeDays)*24*time.Hour
		overCount := s.policy.KeepMostRecent > 0 && keptUnpinned > s.policy.KeepMostRecent
		if tooOld || overCount {
			report.Eligible = append(report.Eligible, run.RunID)
			if err := s.delete(ctx, run.RunID); err != nil {
				return report, err
			}
			report.Deleted = append(report.Deleted, run.RunID)
			continue
		}
		report.Kept = append(report.Kept, run.RunID)
		keptSize += size
		kept = append(kept, keptRun{id: run.RunID, size: size})
	}

	if s.policy.KeepMaxSizeMB > 0 {
		capBytes := int64(s.policy.KeepMaxSizeMB) * 1024 * 1024
		for i := len(kept) - 1; i >= 0 && keptSize > capBytes; i-- {
			report.Eligible = append(report.Eligible, kept[i].id)
			if err := s.delete(ctx, kept[i].id); err != nil {
				return report, err
			}
			report.Deleted = append(report.Deleted, kept[i].id)
			keptSize -= kept[i].size
		}
	}
	return report, nil
}

// Delete removes an operator-selected run through the same tombstone and
// reconciliation flow as policy GC. Actively leased evidence needs force explicitly.
func (s *RetentionService) Delete(ctx context.Context, runID string, force bool) error {
	if s == nil || s.index == nil {
		return fmt.Errorf("retention service is not configured")
	}
	if _, err := s.index.Find(runID); err != nil {
		return err
	}
	leased, err := s.leases.Active(runID, s.now().UTC())
	if err != nil {
		return fmt.Errorf("check pin lease for %s: %w", runID, err)
	}
	if leased && !force {
		return ErrRunPinned
	}
	return s.delete(ctx, runID)
}

// Reconcile completes any deletion that wrote its tombstone but did not reach
// the index update. It is idempotent: missing directories and index entries
// are treated as already-completed work.
func (s *RetentionService) Reconcile(ctx context.Context) error {
	entries, err := os.ReadDir(s.tombstoneDir())
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read retention tombstones: %w", err)
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.tombstoneDir(), entry.Name()))
		if err != nil {
			return err
		}
		var tombstone retentionTombstone
		if err := json.Unmarshal(data, &tombstone); err != nil || tombstone.RunID == "" {
			return fmt.Errorf("invalid retention tombstone %s", entry.Name())
		}
		if err := s.deleteFootprint(tombstone.RunID); err != nil {
			return err
		}
		if err := s.deleteDetails(ctx, tombstone.RunID); err != nil {
			return err
		}
		if err := s.index.Remove(tombstone.RunID); err != nil {
			return err
		}
		if err := os.Remove(filepath.Join(s.tombstoneDir(), entry.Name())); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

type retentionTombstone struct {
	RunID     string    `json:"runId"`
	CreatedAt time.Time `json:"createdAt"`
}

func (s *RetentionService) delete(ctx context.Context, runID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(s.tombstoneDir(), 0o755); err != nil {
		return err
	}
	tombstonePath := filepath.Join(s.tombstoneDir(), sharedartifacts.SanitizeFilename(runID)+".json")
	payload, err := json.Marshal(retentionTombstone{RunID: runID, CreatedAt: s.now().UTC()})
	if err != nil {
		return err
	}
	if err := os.WriteFile(tombstonePath, payload, 0o644); err != nil {
		return err
	}
	if err := s.deleteFootprint(runID); err != nil {
		return err
	}
	if err := s.deleteDetails(ctx, runID); err != nil {
		return err
	}
	if err := s.index.Remove(runID); err != nil {
		return err
	}
	if err := os.Remove(tombstonePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *RetentionService) deleteFootprint(runID string) error {
	if err := os.RemoveAll(sharedartifacts.RunDir(s.scenarioDir, runID)); err != nil {
		return err
	}
	return os.RemoveAll(sharedartifacts.RunLogsDir(s.scenarioDir, runID))
}

func (s *RetentionService) deleteDetails(ctx context.Context, runID string) error {
	if s.details == nil {
		return nil
	}
	if err := s.details.DeleteByRunID(ctx, runID); err != nil {
		return fmt.Errorf("delete execution detail for %s: %w", runID, err)
	}
	return nil
}

func (s *RetentionService) tombstoneDir() string {
	return filepath.Join(s.scenarioDir, sharedartifacts.CoverageRoot, "retention-tombstones")
}

// DeleteRun remains the public run API but delegates to the lifecycle owner.
func DeleteRun(scenarioDir, runID string, force bool) error {
	return NewRetentionService(scenarioDir, DefaultRetentionPolicy()).Delete(context.Background(), runID, force)
}

func runFootprintBytes(scenarioDir, runID string) int64 {
	return dirSizeBytes(sharedartifacts.RunDir(scenarioDir, runID)) + dirSizeBytes(sharedartifacts.RunLogsDir(scenarioDir, runID))
}

func dirSizeBytes(dir string) int64 {
	var total int64
	_ = filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if info, infoErr := d.Info(); infoErr == nil {
				total += info.Size()
			}
		}
		return nil
	})
	return total
}

// Ensure the report stays deterministic for operator output if a future source
// supplies candidates out of index order.
func (r *RetentionReport) Sort() {
	sort.Strings(r.Eligible)
	sort.Strings(r.Deleted)
	sort.Strings(r.Kept)
}
