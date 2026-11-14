package services

// DOC: docs/concepts/ARCHITECTURE.md#metrics-maintenance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

// AutoCompactionPolicy decides when the owner may run the one-time full
// VACUUM that converts a legacy auto_vacuum=NONE file to INCREMENTAL. After
// that conversion every retention pass returns freed pages online in bounded
// batches, so the full rewrite is needed at most once per database file.
type AutoCompactionPolicy struct {
	// WindowStartHour/WindowEndHour bound the quiet local-time window
	// [start, end). The rewrite holds the write lock for minutes.
	WindowStartHour int
	WindowEndHour   int
	// MinFreeBytes and MinFreeFraction: only rewrite when the freelist is
	// worth it (both must hold).
	MinFreeBytes    int64
	MinFreeFraction float64
	// SpaceMarginBytes stays free beyond VACUUM's estimated transient use.
	SpaceMarginBytes int64
}

// DefaultAutoCompactionPolicy rewrites between 02:00 and 05:00 local time when
// at least 1 GiB and 20% of the file are free pages, keeping 4 GiB of disk
// headroom.
func DefaultAutoCompactionPolicy() AutoCompactionPolicy {
	return AutoCompactionPolicy{WindowStartHour: 2, WindowEndHour: 5, MinFreeBytes: 1 << 30, MinFreeFraction: 0.20, SpaceMarginBytes: 4 << 30}
}

// AutoCompactionReceipt is the persisted evidence of one decision.
type AutoCompactionReceipt struct {
	DecidedAt      time.Time                 `json:"decided_at"`
	Decision       string                    `json:"decision"`
	Reason         string                    `json:"reason"`
	Before         repository.DatabaseStats  `json:"before"`
	After          *repository.DatabaseStats `json:"after,omitempty"`
	ReclaimedBytes int64                     `json:"reclaimed_bytes,omitempty"`
	DurationMillis int64                     `json:"duration_millis,omitempty"`
	FreeDiskBytes  uint64                    `json:"free_disk_bytes,omitempty"`
	NeedDiskBytes  int64                     `json:"need_disk_bytes,omitempty"`
	Error          string                    `json:"error,omitempty"`
}

// Decisions recorded in the receipt.
const (
	CompactionNotNeeded   = "not_needed"
	CompactionDeferred    = "deferred"
	CompactionPerformed   = "performed"
	CompactionFailed      = "failed"
	compactionReceiptName = "compaction-receipt.json"
)

// AutoCompactor runs the conversion VACUUM when the policy allows it.
type AutoCompactor struct {
	repo        repository.MaintenanceRepository
	policy      AutoCompactionPolicy
	clock       Clock
	freeSpace   func(dir string) (uint64, error)
	receiptPath string
}

// NewAutoCompactor builds a compactor; receiptPath may be empty to skip the
// persisted receipt (tests).
func NewAutoCompactor(repo repository.MaintenanceRepository, policy AutoCompactionPolicy, receiptPath string) *AutoCompactor {
	return &AutoCompactor{repo: repo, policy: policy, clock: RealClock{}, freeSpace: diskFreeBytes, receiptPath: receiptPath}
}

// AutoCompactionReceiptPath is the owner-scoped receipt location.
func AutoCompactionReceiptPath() (string, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{})
	if err != nil {
		return "", err
	}
	return resolver.Path(storage.Options{ScenarioID: "system-monitor"}, storage.ClassState, filepath.Join("retention", compactionReceiptName))
}

// MaybeCompact evaluates the policy and performs the conversion VACUUM when
// every gate holds. Only "performed" and "failed" outcomes are persisted, so
// the receipt keeps the last real action rather than hourly deferrals.
func (c *AutoCompactor) MaybeCompact(ctx context.Context) (AutoCompactionReceipt, error) {
	now := c.clock.Now()
	receipt := AutoCompactionReceipt{DecidedAt: now.UTC()}
	before, err := c.repo.SQLiteStats(ctx)
	if errors.Is(err, repository.ErrNotSupported) {
		receipt.Decision, receipt.Reason = CompactionNotNeeded, "repository has no page storage"
		return receipt, nil
	}
	if err != nil {
		return receipt, err
	}
	receipt.Before = before
	freeBytes := before.FreelistCount * before.PageSize

	switch {
	case before.AutoVacuum == repository.AutoVacuumIncremental:
		receipt.Decision, receipt.Reason = CompactionNotNeeded, "already incremental; retention returns free pages online"
		return receipt, nil
	case freeBytes < c.policy.MinFreeBytes || float64(freeBytes) < c.policy.MinFreeFraction*float64(before.SizeBytes):
		receipt.Decision, receipt.Reason = CompactionDeferred, fmt.Sprintf("only %d free bytes of %d; below the rewrite threshold", freeBytes, before.SizeBytes)
		return receipt, nil
	case !inWindow(now.Hour(), c.policy.WindowStartHour, c.policy.WindowEndHour):
		receipt.Decision, receipt.Reason = CompactionDeferred, fmt.Sprintf("outside the quiet window %02d:00-%02d:00", c.policy.WindowStartHour, c.policy.WindowEndHour)
		return receipt, nil
	}

	// VACUUM writes a temp copy of the live pages and passes the rewritten
	// database through the WAL; both land on the database's filesystem here
	// (temp_store=FILE in the system temp dir, normally the same device).
	live := before.SizeBytes - freeBytes
	need := 2*live + before.WALBytes + c.policy.SpaceMarginBytes
	receipt.NeedDiskBytes = need
	if before.Path != "" {
		free, ferr := c.freeSpace(filepath.Dir(before.Path))
		if ferr != nil {
			receipt.Decision, receipt.Reason = CompactionDeferred, "cannot measure free disk: "+ferr.Error()
			return receipt, nil
		}
		receipt.FreeDiskBytes = free
		if int64(free) < need {
			receipt.Decision, receipt.Reason = CompactionDeferred, fmt.Sprintf("insufficient disk: %d free, need %d", free, need)
			return receipt, nil
		}
	}

	start := c.clock.Now()
	result, err := c.repo.Compact(ctx)
	receipt.DurationMillis = c.clock.Since(start).Milliseconds()
	if err != nil {
		receipt.Decision, receipt.Error = CompactionFailed, err.Error()
		c.persist(receipt)
		return receipt, err
	}
	after := result.StatsAfter
	receipt.After = &after
	receipt.ReclaimedBytes = result.ReclaimedBytes
	receipt.Decision, receipt.Reason = CompactionPerformed, "converted to auto_vacuum=INCREMENTAL"
	if after.AutoVacuum != repository.AutoVacuumIncremental {
		receipt.Decision = CompactionFailed
		receipt.Error = fmt.Sprintf("auto_vacuum is %d after compaction, want %d", after.AutoVacuum, repository.AutoVacuumIncremental)
		c.persist(receipt)
		return receipt, errors.New(receipt.Error)
	}
	c.persist(receipt)
	return receipt, nil
}

func (c *AutoCompactor) persist(receipt AutoCompactionReceipt) {
	if c.receiptPath == "" {
		return
	}
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return
	}
	_ = storage.WriteFileAtomic(c.receiptPath, append(data, '\n'), 0o644)
}

func inWindow(hour, start, end int) bool {
	if start == end {
		return true
	}
	if start < end {
		return hour >= start && hour < end
	}
	return hour >= start || hour < end // wraps midnight
}

// ParseCompactionWindow parses "HH-HH" (SYSTEM_MONITOR_COMPACTION_WINDOW).
func ParseCompactionWindow(raw string, fallback AutoCompactionPolicy) AutoCompactionPolicy {
	var start, end int
	if _, err := fmt.Sscanf(raw, "%d-%d", &start, &end); err != nil || start < 0 || start > 23 || end < 0 || end > 23 {
		return fallback
	}
	fallback.WindowStartHour, fallback.WindowEndHour = start, end
	return fallback
}
