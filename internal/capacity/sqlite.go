package capacity

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/clock"
	"github.com/vrooli/vrooli/internal/sqliteutil"
	"github.com/vrooli/vrooli/internal/tuning"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
	// Importing modernc.org/sqlite registers the pure-Go SQLite driver.
)

const capacityTxRetryAttempts = 5

var (
	capacityTxRetryBase = tuning.FastPersistenceRetryInterval()
	capacityTxRetryMax  = tuning.FastHealthPollInterval()
)

// Config configures the capacity ledger store.
type Config struct {
	HomeDir  string
	DBPath   string
	Clock    clock.Clock
	ReadOnly bool
	// MinimumDiskFloorBytes overrides the production 120 GiB post-peak floor.
	// Tests may set it to zero to avoid host-capacity assumptions.
	MinimumDiskFloorBytes *int64
	// MetadataDiskFloorBytes overrides the production floor for metadata-class
	// disk jobs (DiskJobClassMetadata). Tests may set it to zero.
	MetadataDiskFloorBytes *int64
}

// SQLiteStore is the capacity claim ledger backed by SQLite (modernc, pure-Go).
// It mirrors internal/scenarioruntime.SQLiteStore's patterns (single conn, WAL,
// busy_timeout, withRetryableTx) without importing it.
type SQLiteStore struct {
	db               *sql.DB
	clock            clock.Clock
	dbPath           string
	diskFloorMinimum int64
	// diskMetadataFloor is the post-peak floor of a metadata-class disk job.
	diskMetadataFloor int64
	admissionOnce     sync.Once
	admissionGate     chan struct{}
}

// DefaultDBPath resolves the capacity ledger SQLite path. It lives beside the
// scenarioruntime registry under the runtime-home `state` directory
// (state/capacity.db), resolved through the repo-contract authority. When
// homeDir is empty it uses the sudo-aware resolver in internal/config (never
// bare os.UserHomeDir, which would point a sudo'd process at /root).
func DefaultDBPath(homeDir string) (string, error) {
	if strings.TrimSpace(homeDir) == "" {
		dir, err := config.HomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		homeDir = dir
	}
	stateDir, err := repocontract.RuntimeHomeEntryPath(homeDir, repocontract.HomeKeyState)
	if err != nil {
		return "", fmt.Errorf("resolve runtime-home state dir: %w", err)
	}
	return filepath.Join(stateDir, "capacity.db"), nil
}

// ResolveConfiguredDBPath validates isolation before resolving a default ledger
// path. Constructors must call it before taking file locks or opening SQLite.
// Canonical file identity is established separately by NewSQLiteStore.
func ResolveConfiguredDBPath(cfg Config) (string, error) {
	// Hard test-isolation seam (plan §Phase 1): under `go test`, opening the
	// ledger without an explicit DBPath would silently resolve to (and write) the
	// LIVE ~/.vrooli/state/capacity.db — VROOLI_HOME does NOT isolate it, the very
	// gotcha that left `iy-grant-*` fixtures in the live ledger. Refuse, so every
	// capacity test must pass Config{DBPath: t.TempDir()...}. HomeDir-overridden
	// stores (cfg.HomeDir set) are still allowed for tests that fully redirect the
	// home root.
	if testing.Testing() && strings.TrimSpace(cfg.DBPath) == "" && strings.TrimSpace(cfg.HomeDir) == "" {
		return "", fmt.Errorf("capacity ledger: tests must pass an explicit Config.DBPath (or HomeDir); refusing to open the live ledger at the default path")
	}
	if strings.TrimSpace(cfg.DBPath) == "" {
		return DefaultDBPath(cfg.HomeDir)
	}
	return cfg.DBPath, nil
}

// NewSQLiteStore opens (and lazily creates + stamps) the capacity ledger.
func NewSQLiteStore(ctx context.Context, cfg Config) (*SQLiteStore, error) {
	clk := cfg.Clock
	if clk == nil {
		clk = clock.Real{}
	}
	dbPath, err := ResolveConfiguredDBPath(cfg)
	if err != nil {
		return nil, err
	}
	// Bind path interpretation to this open, rather than a later broker's cwd.
	// The non-file identity remains special and must not acquire a disk lock.
	if dbPath != ":memory:" {
		absolute, err := filepath.Abs(dbPath)
		if err != nil {
			return nil, fmt.Errorf("resolve capacity ledger absolute path: %w", err)
		}
		dbPath = absolute
	}
	readOnly := cfg.ReadOnly || strings.TrimSpace(os.Getenv("VROOLI_SANDBOX_MERGED")) != ""
	if !readOnly && dbPath != ":memory:" {
		if _, err := config.EnsureOwnedDir(filepath.Dir(dbPath)); err != nil {
			return nil, fmt.Errorf("prepare capacity ledger directory: %w", err)
		}
	}
	if dbPath != ":memory:" {
		if canonical, err := filepath.EvalSymlinks(dbPath); err == nil {
			dbPath = canonical
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("resolve capacity ledger file identity: %w", err)
		} else {
			// A new ledger has no leaf yet; its existing parent still has one
			// canonical identity, including runtime-home directory symlinks.
			parent, err := filepath.EvalSymlinks(filepath.Dir(dbPath))
			if err != nil {
				return nil, fmt.Errorf("resolve capacity ledger parent identity: %w", err)
			}
			dbPath = filepath.Join(parent, filepath.Base(dbPath))
		}
	}

	dsn := buildDSN(dbPath)
	if readOnly {
		dsn = buildReadOnlyDSN(dbPath)
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open capacity ledger sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)

	diskFloorMinimum := int64(120 << 30)
	if cfg.MinimumDiskFloorBytes != nil {
		if !testing.Testing() && *cfg.MinimumDiskFloorBytes < diskFloorMinimum {
			db.Close()
			return nil, fmt.Errorf("capacity ledger disk floor cannot be lowered below 120 GiB outside tests")
		}
		diskFloorMinimum = *cfg.MinimumDiskFloorBytes
	}
	diskMetadataFloor := DiskJobMetadataFloorBytes
	if cfg.MetadataDiskFloorBytes != nil {
		if !testing.Testing() && *cfg.MetadataDiskFloorBytes < DiskJobMetadataFloorBytes {
			db.Close()
			return nil, fmt.Errorf("capacity ledger metadata disk floor cannot be lowered below %d bytes outside tests", DiskJobMetadataFloorBytes)
		}
		diskMetadataFloor = *cfg.MetadataDiskFloorBytes
	}
	store := &SQLiteStore{db: db, clock: clk, dbPath: dbPath, diskFloorMinimum: diskFloorMinimum, diskMetadataFloor: diskMetadataFloor}
	if !readOnly {
		if err := store.ensureSchema(ctx); err != nil {
			db.Close()
			return nil, err
		}
	}
	return store, nil
}

// DBPath returns the path identity this store opened. It is exposed for
// adapters that must coordinate mutations across processes using the same
// ledger. In-memory SQLite stores return their configured in-memory identity;
// callers should not treat separate in-memory handles as shared storage.
func (s *SQLiteStore) DBPath() string {
	if s == nil {
		return ""
	}
	return s.dbPath
}

// Close releases the underlying database handle.
func (s *SQLiteStore) Close() error {
	if s == nil {
		return nil
	}
	return sqliteutil.Close(s.db)
}

func (s *SQLiteStore) now() time.Time { return s.clock.Now().UTC() }

func normalizeTTL(ttl, fallback time.Duration) time.Duration {
	if ttl <= 0 {
		if fallback <= 0 {
			return DefaultHeartbeatTTL
		}
		return fallback
	}
	return ttl
}

// CreateClaim persists a new claim. The verdict/sizing decision is made by
// Decide; CreateClaim only records the granted result. Status defaults to
// granted, activity to idle, generation to 1.
func (s *SQLiteStore) CreateClaim(ctx context.Context, claim CapacityClaim, ttl time.Duration) (CapacityClaim, error) {
	if strings.TrimSpace(claim.OwnerID) == "" {
		return CapacityClaim{}, fmt.Errorf("%w: owner_id is required", ErrInvalidClaim)
	}
	if claim.ResourceKind == "" {
		return CapacityClaim{}, fmt.Errorf("%w: resource_kind is required", ErrInvalidClaim)
	}
	if strings.TrimSpace(claim.ClaimID) == "" {
		claim.ClaimID = newID("clm")
	}
	if claim.OwnerKind == "" {
		claim.OwnerKind = OwnerKindResource
	}
	if claim.Status == "" {
		claim.Status = StatusGranted
	}
	if claim.ActivityState == "" {
		claim.ActivityState = ActivityIdle
	}
	if claim.Priority == 0 {
		claim.Priority = PriorityBatch
	}
	if claim.Generation <= 0 {
		claim.Generation = 1
	}
	now := s.now()
	deadline := now.Add(normalizeTTL(ttl, DefaultHeartbeatTTL))
	claim.CreatedAt = now
	claim.UpdatedAt = now
	claim.LastHeartbeatAt = &now
	claim.HeartbeatDeadlineAt = &deadline
	if claim.ActivityState == ActivityActive {
		claim.LastActiveAt = &now
		if claim.Priority >= PriorityInteractive {
			claim.Protected = true
		}
	}

	profileJSON, err := marshalProfile(claim.DegradeProfile)
	if err != nil {
		return CapacityClaim{}, err
	}

	err = s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		_, execErr := tx.ExecContext(ctx, `
INSERT INTO capacity_claims (
  claim_id, owner_kind, owner_id, instance_id, resource_kind, gpu_index,
  amount_bytes, preferred_bytes, floor_bytes, priority, protected, yield_when_idle, status,
  activity_state, generation, created_at, updated_at, last_heartbeat_at,
  heartbeat_deadline_at, last_active_at, degrade_profile,
  observed_bytes, observed_peak_bytes, observed_at, idle_unload_ttl_seconds, idle_grace_seconds
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			claim.ClaimID, claim.OwnerKind, claim.OwnerID, claim.InstanceID, claim.ResourceKind,
			sqliteutil.OptionalIntValue(claim.GPUIndex), claim.AmountBytes, claim.PreferredBytes, claim.FloorBytes,
			claim.Priority, boolToInt(claim.Protected), boolToInt(claim.YieldWhenIdle), claim.Status, claim.ActivityState, claim.Generation,
			formatTime(claim.CreatedAt), formatTime(claim.UpdatedAt), formatOptionalTime(claim.LastHeartbeatAt),
			formatOptionalTime(claim.HeartbeatDeadlineAt), formatOptionalTime(claim.LastActiveAt), profileJSON,
			claim.ObservedBytes, claim.ObservedPeakBytes, formatOptionalTime(claim.ObservedAt), int64(claim.IdleUnloadTTL/time.Second), int64(claim.IdleGrace/time.Second))
		if execErr != nil {
			return fmt.Errorf("insert capacity claim: %w", execErr)
		}
		return nil
	})
	if err != nil {
		return CapacityClaim{}, err
	}
	return claim, nil
}

// HeartbeatClaim renews liveness without changing activity or generation. A
// heartbeat with a stale generation fails (the claim was mutated by the broker,
// e.g. degraded), signaling the owner to re-read.
func (s *SQLiteStore) HeartbeatClaim(ctx context.Context, claimID string, generation int64, ttl time.Duration) (CapacityClaim, error) {
	if strings.TrimSpace(claimID) == "" {
		return CapacityClaim{}, fmt.Errorf("%w: claim_id is required", ErrInvalidClaim)
	}
	var out CapacityClaim
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		if err := acquireClaimWriteAuthority(ctx, tx); err != nil {
			return err
		}
		now := s.now()
		deadline := now.Add(normalizeTTL(ttl, DefaultHeartbeatTTL))
		// Heartbeat proves liveness; it is not a resize. Keeping updated_at on
		// the last real mutation is what makes the degrade/upshift debounce
		// meaningful: a parked claim is heartbeated every poll, and bumping this
		// column made every upshift permanently look "just resized".
		result, execErr := tx.ExecContext(ctx, `
UPDATE capacity_claims
SET last_heartbeat_at = ?, heartbeat_deadline_at = ?
WHERE claim_id = ? AND generation = ? AND status IN (?, ?, ?)
  AND (owner_kind != ? OR heartbeat_deadline_at > ?)`,
			formatTime(now), formatTime(deadline),
			claimID, generation, StatusReserved, StatusGranted, StatusDegraded, OwnerKindOp, formatTime(now))
		if execErr != nil {
			return fmt.Errorf("heartbeat capacity claim: %w", execErr)
		}
		return finishMutation(ctx, tx, result, claimID, &out)
	})
	if err != nil {
		return CapacityClaim{}, err
	}
	return out, nil
}

// ReportActivity sets the work-owner-reported activity state. It bumps the
// generation: an activity change invalidates any pending arbitration decision
// (so the broker can't degrade a claim that just went active). For
// interactive-tier claims, active auto-sets protected and idle clears it.
func (s *SQLiteStore) ReportActivity(ctx context.Context, claimID string, generation int64, state string) (CapacityClaim, error) {
	if strings.TrimSpace(claimID) == "" {
		return CapacityClaim{}, fmt.Errorf("%w: claim_id is required", ErrInvalidClaim)
	}
	if state != ActivityActive && state != ActivityIdle {
		return CapacityClaim{}, fmt.Errorf("%w: activity state must be active|idle", ErrInvalidClaim)
	}
	now := s.now()
	var out CapacityClaim
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		// Serialize the demand report against transition creation on this exact
		// claim. BeginResidencyTransition takes the same SQLite write authority
		// before it checks idleness; whichever operation wins is therefore
		// authoritative, and a late active signal cannot race an open unload.
		if _, err := tx.ExecContext(ctx, `UPDATE capacity_claims SET claim_id=claim_id WHERE claim_id=?`, claimID); err != nil {
			return fmt.Errorf("acquire activity claim write authority: %w", err)
		}
		unresolved, fenceErr := hasUnresolvedResidencyTransitionTx(ctx, tx, claimID, s.now())
		if fenceErr != nil {
			return fenceErr
		}
		if unresolved {
			return ErrResidencyTransitionOpen
		}
		prior, getErr := getClaimTx(ctx, tx, claimID)
		if getErr != nil {
			return getErr
		}
		if prior.Generation != generation {
			return staleGenerationError(prior)
		}
		if !IsActiveClaimStatus(prior.Status) {
			return fmt.Errorf("%w: claim %s is %s, not active", ErrInvalidClaim, claimID, prior.Status)
		}
		lastActive := formatOptionalTime(prior.LastActiveAt)
		protected := prior.Protected
		if state == ActivityActive || prior.ActivityState == ActivityActive {
			lastActive = formatTime(now)
		}
		if state == ActivityActive {
			if prior.Priority >= PriorityInteractive {
				protected = true
			}
		} else if prior.Priority >= PriorityInteractive {
			protected = false
		}
		result, execErr := tx.ExecContext(ctx, `
UPDATE capacity_claims
SET activity_state = ?, protected = ?, last_active_at = ?, generation = generation + 1, updated_at = ?
WHERE claim_id = ? AND generation = ?`,
			state, boolToInt(protected), lastActive, formatTime(now), claimID, generation)
		if execErr != nil {
			return fmt.Errorf("report capacity activity: %w", execErr)
		}
		return finishMutation(ctx, tx, result, claimID, &out)
	})
	if err != nil {
		return CapacityClaim{}, err
	}
	return out, nil
}

// DegradeClaim moves a claim to a lower profile step (the broker->adopter
// callback target's persisted result). It bumps the generation.
func (s *SQLiteStore) DegradeClaim(ctx context.Context, claimID string, generation int64, step string, amountBytes int64) (CapacityClaim, error) {
	if strings.TrimSpace(claimID) == "" {
		return CapacityClaim{}, fmt.Errorf("%w: claim_id is required", ErrInvalidClaim)
	}
	if amountBytes < 0 {
		return CapacityClaim{}, fmt.Errorf("%w: degrade amount must be non-negative", ErrInvalidClaim)
	}
	now := s.now()
	var out CapacityClaim
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		unresolved, fenceErr := hasUnresolvedResidencyTransitionTx(ctx, tx, claimID, s.now())
		if fenceErr != nil {
			return fenceErr
		}
		if unresolved {
			return ErrResidencyTransitionOpen
		}
		result, execErr := tx.ExecContext(ctx, `
UPDATE capacity_claims
SET status = ?, amount_bytes = ?, generation = generation + 1, updated_at = ?
WHERE claim_id = ? AND generation = ? AND status IN (?, ?, ?)`,
			StatusDegraded, amountBytes, formatTime(now),
			claimID, generation, StatusReserved, StatusGranted, StatusDegraded)
		if execErr != nil {
			return fmt.Errorf("degrade capacity claim: %w", execErr)
		}
		_ = step // the label is the adopter's concern; the ledger records the resulting amount.
		return finishMutation(ctx, tx, result, claimID, &out)
	})
	if err != nil {
		return CapacityClaim{}, err
	}
	return out, nil
}

// ResizeClaim changes an existing claim's amount in place, keeping its identity
// and its observed-usage history.
//
// A resize is not a degrade and not an upshift: those two move a claim between
// declared profile rungs and mean "the broker asked for less" or "the broker
// gave back more". A resize means "what this owner actually needs has changed",
// which is an owner-side fact. It changes the amount only: preferred_bytes is
// the declaration (the manifest's reservation, which lifecycle admission
// compares against and right-sizing measures against), not the current size.
// Modelling it as release-and-reclaim, as the
// ollama companion did, creates a new ledger row per model load and throws away
// the observed peak that right-sizing depends on.
func (s *SQLiteStore) ResizeClaim(ctx context.Context, claimID string, generation int64, amountBytes int64) (CapacityClaim, error) {
	if strings.TrimSpace(claimID) == "" {
		return CapacityClaim{}, fmt.Errorf("%w: claim_id is required", ErrInvalidClaim)
	}
	if amountBytes <= 0 {
		return CapacityClaim{}, fmt.Errorf("%w: resize amount must be positive; release the claim to give the capacity back", ErrInvalidClaim)
	}
	now := s.now()
	var out CapacityClaim
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		unresolved, fenceErr := hasUnresolvedResidencyTransitionTx(ctx, tx, claimID, s.now())
		if fenceErr != nil {
			return fenceErr
		}
		if unresolved {
			return ErrResidencyTransitionOpen
		}
		result, execErr := tx.ExecContext(ctx, `
UPDATE capacity_claims
SET status = ?, amount_bytes = ?, generation = generation + 1, updated_at = ?
WHERE claim_id = ? AND generation = ? AND status IN (?, ?, ?)`,
			StatusGranted, amountBytes, formatTime(now),
			claimID, generation, StatusReserved, StatusGranted, StatusDegraded)
		if execErr != nil {
			return fmt.Errorf("resize capacity claim: %w", execErr)
		}
		return finishMutation(ctx, tx, result, claimID, &out)
	})
	if err != nil {
		return CapacityClaim{}, err
	}
	return out, nil
}
