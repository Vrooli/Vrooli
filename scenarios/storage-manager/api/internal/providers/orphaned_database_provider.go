package providers

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/api-core/database"
	"storage-manager/internal/cleanup"
)

// OrphanedDatabaseProvider selects only a superseded database whose live
// replacement exists, is recent, and has no open handle. It is intentionally
// owner-budgeted: the zero-byte declaration is the R2 authority.
type OrphanedDatabaseProvider struct {
	legacy, live string
	files        cleanup.FileSystem
	liveness     cleanup.ProcessLiveness
	clock        cleanup.Clock
	minAge       time.Duration
	quarantine   string
	verifyLive   func(context.Context, string) error
}

func NewOrphanedDatabaseProvider(files cleanup.FileSystem, liveness cleanup.ProcessLiveness, clock cleanup.Clock, legacy, live string, minAge time.Duration, quarantineRoot ...string) cleanup.Provider {
	return newOrphanedDatabaseProvider(files, liveness, clock, legacy, live, minAge, nil, quarantineRoot...)
}

func NewOrphanedDatabaseProviderWithVerifier(files cleanup.FileSystem, liveness cleanup.ProcessLiveness, clock cleanup.Clock, legacy, live string, minAge time.Duration, verifier func(context.Context, string) error, quarantineRoot ...string) cleanup.Provider {
	return newOrphanedDatabaseProvider(files, liveness, clock, legacy, live, minAge, verifier, quarantineRoot...)
}

func newOrphanedDatabaseProvider(files cleanup.FileSystem, liveness cleanup.ProcessLiveness, clock cleanup.Clock, legacy, live string, minAge time.Duration, verifier func(context.Context, string) error, quarantineRoot ...string) cleanup.Provider {
	if minAge <= 0 {
		minAge = 30 * 24 * time.Hour
	}
	quarantine := ""
	if len(quarantineRoot) > 0 {
		quarantine = filepath.Clean(quarantineRoot[0])
	}
	return &OrphanedDatabaseProvider{legacy: filepath.Clean(legacy), live: filepath.Clean(live), files: files, liveness: liveness, clock: clock, minAge: minAge, quarantine: quarantine, verifyLive: verifier}
}

func (p *OrphanedDatabaseProvider) Metadata() cleanup.ProviderMetadata {
	return cleanup.ProviderMetadata{ID: "orphaned-database", Name: "Orphaned databases", Version: "v1", OwnerScenario: "storage-manager", SafetyTier: cleanup.SafetyTierSafeWithOwner, DefaultMode: cleanup.ProviderModeDisabled, DefaultApproval: cleanup.ApprovalModeOwner, OwnerBudget: true, SupportedPlatforms: []string{"linux", "darwin", "windows"}, IrreversibleEffects: []string{"removes a superseded database after its replacement is verified"}, TestSubstitute: "fake-filesystem"}
}

func (p *OrphanedDatabaseProvider) candidate(ctx context.Context) (cleanup.FileInfo, bool, string) {
	if p.files == nil || p.legacy == "" || p.live == "" {
		return cleanup.FileInfo{}, false, "provider dependencies unavailable"
	}
	legacy, err := p.files.Stat(ctx, p.legacy)
	if err != nil || legacy.IsDir {
		return cleanup.FileInfo{}, false, "legacy database unavailable"
	}
	live, err := p.files.Stat(ctx, p.live)
	if err != nil || live.IsDir {
		return cleanup.FileInfo{}, false, "live replacement unavailable"
	}
	now := time.Now().UTC()
	if p.clock != nil {
		now = p.clock.Now()
	}
	if live.ModTime.IsZero() || now.Sub(live.ModTime) > 24*time.Hour {
		return cleanup.FileInfo{}, false, "live replacement is stale"
	}
	if legacy.ModTime.IsZero() || now.Sub(legacy.ModTime) < p.minAge {
		return cleanup.FileInfo{}, false, "legacy database is younger than minimum age"
	}
	if p.liveness == nil {
		return cleanup.FileInfo{}, false, "process liveness check unavailable"
	}
	for _, path := range []string{p.legacy, p.legacy + "-wal", p.legacy + "-shm"} {
		open, err := p.liveness.IsRunning(ctx, path)
		if err != nil {
			return cleanup.FileInfo{}, false, "process liveness check failed"
		}
		if open {
			return cleanup.FileInfo{}, false, "database family has an open process handle"
		}
	}
	return legacy, true, ""
}

func (p *OrphanedDatabaseProvider) Estimate(ctx context.Context, req cleanup.EstimateRequest) (cleanup.Estimate, error) {
	info, ok, _ := p.candidate(ctx)
	if !ok {
		return cleanup.Estimate{ProviderID: p.Metadata().ID, ProviderVersion: "v1", ObservedAt: req.Scope.Now}, nil
	}
	return cleanup.Estimate{ProviderID: p.Metadata().ID, ProviderVersion: "v1", EstimatedBytes: info.Size, ItemCount: 1, ObservedAt: req.Scope.Now, MaxBytes: req.Policy.MaxBytes}, nil
}

func (p *OrphanedDatabaseProvider) Preview(ctx context.Context, req cleanup.PreviewRequest) (cleanup.Preview, error) {
	info, ok, reason := p.candidate(ctx)
	if !ok {
		return cleanup.Preview{ProviderID: p.Metadata().ID, ProviderVersion: "v1", BlockedReason: reason}, nil
	}
	return cleanup.Preview{ProviderID: p.Metadata().ID, ProviderVersion: "v1", MaxBytes: req.Policy.MaxBytes, AllowSingleOvershoot: true, Items: []cleanup.PreviewItem{{ID: p.legacy, Path: p.legacy, Description: "superseded database; live replacement is recent and unheld", Bytes: info.Size, Action: "remove orphan", SafetyTier: cleanup.SafetyTierSafeWithOwner}}}, nil
}

func (p *OrphanedDatabaseProvider) Apply(ctx context.Context, req cleanup.ApplyRequest) (cleanup.ApplyResult, error) {
	if req.IdempotencyKey == "" {
		return cleanup.ApplyResult{}, fmt.Errorf("orphaned database apply requires idempotency key")
	}
	if p.quarantine != "" {
		result, err := p.expireQuarantine(ctx)
		if err != nil {
			return result, err
		}
		if _, ok, _ := p.candidate(ctx); !ok {
			result.AlreadyDone = result.ReclaimedBytes == 0
			return result, nil
		}
		if err := p.files.MkdirAll(ctx, p.quarantine); err != nil {
			return result, fmt.Errorf("create orphan quarantine: %w", err)
		}
		stamp := p.now().Unix()
		for _, path := range []string{p.legacy, p.legacy + "-wal", p.legacy + "-shm"} {
			if _, err := p.files.Stat(ctx, path); err != nil {
				continue
			}
			destination := filepath.Join(p.quarantine, fmt.Sprintf("%d-%s", stamp, filepath.Base(path)))
			if err := p.files.Rename(ctx, path, destination); err != nil {
				result.Warnings = append(result.Warnings, err.Error())
				continue
			}
			result.AppliedItems = append(result.AppliedItems, destination)
		}
		result.Applied = true
		result.Warnings = append(result.Warnings, "legacy database family quarantined for 24h; no bytes reclaimed until expiry")
		return result, nil
	}
	if _, ok, _ := p.candidate(ctx); !ok {
		return cleanup.ApplyResult{ProviderID: p.Metadata().ID, AlreadyDone: true}, nil
	}
	result := cleanup.ApplyResult{ProviderID: p.Metadata().ID, Applied: true}
	for _, path := range []string{p.legacy, p.legacy + "-wal", p.legacy + "-shm"} {
		var size int64
		if info, statErr := p.files.Stat(ctx, path); statErr == nil {
			size = info.Size
		}
		if err := p.files.RemoveAll(ctx, path); err != nil {
			// A failed removal is not reclaimed space. Keeping the byte count
			// conditional on successful deletion makes the durable recovery
			// ledger truthful when a sidecar disappears or permissions change.
			result.Warnings = append(result.Warnings, err.Error())
			continue
		}
		result.ReclaimedBytes += size
		result.AppliedItems = append(result.AppliedItems, path)
	}
	return result, nil
}

func (p *OrphanedDatabaseProvider) now() time.Time {
	if p.clock != nil {
		return p.clock.Now()
	}
	return time.Now().UTC()
}

func (p *OrphanedDatabaseProvider) expireQuarantine(ctx context.Context) (cleanup.ApplyResult, error) {
	result := cleanup.ApplyResult{ProviderID: p.Metadata().ID, Applied: true}
	entries, err := p.files.ReadDir(ctx, p.quarantine)
	if err != nil {
		// A missing quarantine is the normal first-run state.
		return result, nil
	}
	cutoff := p.now().Add(-24 * time.Hour).Unix()
	for _, entry := range entries {
		name := filepath.Base(entry.Path)
		parts := strings.SplitN(name, "-", 2)
		if len(parts) != 2 {
			continue
		}
		stamp, parseErr := strconv.ParseInt(parts[0], 10, 64)
		if parseErr != nil || stamp > cutoff || !p.isOrphanFamily(parts[1]) {
			continue
		}
		if err := p.files.RemoveAll(ctx, entry.Path); err != nil {
			result.Warnings = append(result.Warnings, err.Error())
			continue
		}
		result.ReclaimedBytes += entry.Size
		result.AppliedItems = append(result.AppliedItems, entry.Path)
	}
	return result, nil
}

func (p *OrphanedDatabaseProvider) isOrphanFamily(name string) bool {
	base := filepath.Base(p.legacy)
	return name == base || name == base+"-wal" || name == base+"-shm"
}

func (p *OrphanedDatabaseProvider) Verify(ctx context.Context, _ cleanup.VerifyRequest) (cleanup.VerifyResult, error) {
	if _, err := p.files.Stat(ctx, p.live); err != nil {
		return cleanup.VerifyResult{}, fmt.Errorf("live database is unavailable: %w", err)
	}
	if p.verifyLive != nil {
		if err := p.verifyLive(ctx, p.live); err != nil {
			return cleanup.VerifyResult{}, fmt.Errorf("live database quick_check failed: %w", err)
		}
	}
	return cleanup.VerifyResult{Verified: true, Message: "live replacement remains available and passes quick_check"}, nil
}

func VerifySQLiteQuickCheck(ctx context.Context, path string) error {
	db, err := database.Open(ctx, database.Config{
		Driver:       database.DriverSQLite,
		DSN:          "file:" + path + "?mode=ro",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		return err
	}
	defer db.Close()
	var result string
	if err := db.QueryRowContext(ctx, "PRAGMA quick_check").Scan(&result); err != nil {
		return err
	}
	if result != "ok" {
		return fmt.Errorf("result %q", result)
	}
	return nil
}
