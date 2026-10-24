package capacity

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/clock"
	"github.com/vrooli/vrooli/internal/storagetime"
	"github.com/vrooli/vrooli/internal/tuning"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"

	// Importing modernc.org/sqlite registers the pure-Go SQLite driver.
	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
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
}

// SQLiteStore is the capacity claim ledger backed by SQLite (modernc, pure-Go).
// It mirrors internal/scenarioruntime.SQLiteStore's patterns (single conn, WAL,
// busy_timeout, withRetryableTx) without importing it.
type SQLiteStore struct {
	db     *sql.DB
	clock  clock.Clock
	dbPath string
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

	store := &SQLiteStore{db: db, clock: clk, dbPath: dbPath}
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
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
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
			optionalIntValue(claim.GPUIndex), claim.AmountBytes, claim.PreferredBytes, claim.FloorBytes,
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
		result, execErr := tx.ExecContext(ctx, `
UPDATE capacity_claims
SET last_heartbeat_at = ?, heartbeat_deadline_at = ?, updated_at = ?
WHERE claim_id = ? AND generation = ? AND status IN (?, ?, ?)
  AND (owner_kind != ? OR heartbeat_deadline_at > ?)`,
			formatTime(now), formatTime(deadline), formatTime(now),
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
		prior, getErr := getClaimTx(ctx, tx, claimID)
		if getErr != nil {
			return getErr
		}
		if prior.Generation != generation {
			return ErrStaleGeneration
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
// which is an owner-side fact. Modelling it as release-and-reclaim, as the
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
		result, execErr := tx.ExecContext(ctx, `
UPDATE capacity_claims
SET status = ?, amount_bytes = ?, preferred_bytes = ?, generation = generation + 1, updated_at = ?
WHERE claim_id = ? AND generation = ? AND status IN (?, ?, ?)`,
			StatusGranted, amountBytes, amountBytes, formatTime(now),
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

// UpshiftClaim steps a claim UP to a larger profile rung (the symmetric
// counterpart of DegradeClaim): it raises amount_bytes and restores status to
// granted (the claim is no longer running below its preferred size), bumping the
// generation under the optimistic-concurrency guard. The label is the adopter's
// concern; the ledger records the resulting amount.
func (s *SQLiteStore) UpshiftClaim(ctx context.Context, claimID string, generation int64, step string, amountBytes int64) (CapacityClaim, error) {
	if strings.TrimSpace(claimID) == "" {
		return CapacityClaim{}, fmt.Errorf("%w: claim_id is required", ErrInvalidClaim)
	}
	if amountBytes < 0 {
		return CapacityClaim{}, fmt.Errorf("%w: upshift amount must be non-negative", ErrInvalidClaim)
	}
	now := s.now()
	var out CapacityClaim
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		result, execErr := tx.ExecContext(ctx, `
UPDATE capacity_claims
SET status = ?, amount_bytes = ?, generation = generation + 1, updated_at = ?
WHERE claim_id = ? AND generation = ? AND status IN (?, ?, ?)`,
			StatusGranted, amountBytes, formatTime(now),
			claimID, generation, StatusReserved, StatusGranted, StatusDegraded)
		if execErr != nil {
			return fmt.Errorf("upshift capacity claim: %w", execErr)
		}
		_ = step // the label is the adopter's concern; the ledger records the resulting amount.
		return finishMutation(ctx, tx, result, claimID, &out)
	})
	if err != nil {
		return CapacityClaim{}, err
	}
	return out, nil
}

// ReleaseClaim terminates a claim cleanly (by claim ID, no generation guard —
// the owner is done with it).
func (s *SQLiteStore) ReleaseClaim(ctx context.Context, claimID string) (CapacityClaim, error) {
	return s.terminate(ctx, claimID, StatusReleased)
}

// PreemptClaim records preemption intent only for the planned idle generation.
// It does not stop the owner or establish that physical memory was released.
func (s *SQLiteStore) PreemptClaim(ctx context.Context, claimID string, generation int64, reason string) (CapacityClaim, error) {
	_ = reason
	var out CapacityClaim
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE capacity_claims SET status = ?, generation = generation + 1, updated_at = ?
WHERE claim_id = ? AND generation = ? AND status IN (?, ?, ?) AND activity_state = ? AND protected = 0`,
			StatusPreempted, formatTime(s.now()), claimID, generation, StatusReserved, StatusGranted, StatusDegraded, ActivityIdle)
		if err != nil {
			return fmt.Errorf("preempt idle capacity claim: %w", err)
		}
		return finishMutation(ctx, tx, result, claimID, &out)
	})
	return out, err
}

func (s *SQLiteStore) terminate(ctx context.Context, claimID, status string) (CapacityClaim, error) {
	if strings.TrimSpace(claimID) == "" {
		return CapacityClaim{}, fmt.Errorf("%w: claim_id is required", ErrInvalidClaim)
	}
	now := s.now()
	var out CapacityClaim
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		result, execErr := tx.ExecContext(ctx, `
UPDATE capacity_claims
SET status = ?, generation = generation + 1, updated_at = ?
WHERE claim_id = ? AND status IN (?, ?, ?)`,
			status, formatTime(now), claimID, StatusReserved, StatusGranted, StatusDegraded)
		if execErr != nil {
			return fmt.Errorf("terminate capacity claim: %w", execErr)
		}
		affected, raErr := result.RowsAffected()
		if raErr != nil {
			return fmt.Errorf("inspect capacity claim termination: %w", raErr)
		}
		if affected == 0 {
			// Already terminal or absent — surface NotFound vs idempotent return.
			existing, getErr := getClaimTx(ctx, tx, claimID)
			if getErr != nil {
				return getErr
			}
			out = existing
			return nil
		}
		out, execErr = getClaimTx(ctx, tx, claimID)
		return execErr
	})
	if err != nil {
		return CapacityClaim{}, err
	}
	return out, nil
}

// ExpireStaleClaims sweeps active claims whose heartbeat deadline has elapsed to
// status=expired, returning the expired set.
func (s *SQLiteStore) ExpireStaleClaims(ctx context.Context, at time.Time, proofs AbsenceProofs) ([]CapacityClaim, error) {
	var expired []CapacityClaim
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		if err := acquireClaimWriteAuthority(ctx, tx); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, claimSelectSQL+`
WHERE status IN (?, ?, ?) AND heartbeat_deadline_at IS NOT NULL AND heartbeat_deadline_at <= ?
ORDER BY created_at ASC`, StatusReserved, StatusGranted, StatusDegraded, formatTime(at.UTC()))
		if err != nil {
			return fmt.Errorf("list stale capacity claims: %w", err)
		}
		candidates, scanErr := scanClaims(rows)
		if closeErr := rows.Close(); closeErr != nil && scanErr == nil {
			scanErr = closeErr
		}
		if scanErr != nil {
			return scanErr
		}
		expired = nil
		for _, claim := range candidates {
			// The scan or earlier rows can age an absence witness. Validate at this
			// row's mutation boundary while retaining exclusive SQLite write authority.
			now := s.now()
			if claim.ResourceKind == ResourceKindVRAM && claim.OwnerKind != OwnerKindOp && !proofs[claim.ClaimID].validFor(claim, now) {
				continue
			}
			if _, err := tx.ExecContext(ctx, `
UPDATE capacity_claims
SET status = ?, generation = generation + 1, updated_at = ?
WHERE claim_id = ? AND status IN (?, ?, ?)`, StatusExpired, formatTime(now), claim.ClaimID, StatusReserved, StatusGranted, StatusDegraded); err != nil {
				return fmt.Errorf("expire capacity claim %s: %w", claim.ClaimID, err)
			}
			claim.Status = StatusExpired
			claim.Generation++
			claim.UpdatedAt = now
			expired = append(expired, claim)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return expired, nil
}

// GCTerminalClaims prunes terminal (released/expired/preempted) claims whose
// updated_at is older than olderThan, returning how many rows were deleted and
// the sum of their last-recorded amount_bytes (informational). Active claims
// (reserved/granted/degraded) are NEVER pruned — only history is collected. GC
// is always safe regardless of enforce mode (it never frees live capacity, it
// only trims dead rows).
func (s *SQLiteStore) GCTerminalClaims(ctx context.Context, olderThan time.Time) (GCResult, error) {
	cutoff := formatTime(olderThan.UTC())
	var res GCResult
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		row := tx.QueryRowContext(ctx, `
SELECT COUNT(*), COALESCE(SUM(amount_bytes), 0)
FROM capacity_claims
WHERE status NOT IN (?, ?, ?) AND updated_at < ?`,
			StatusReserved, StatusGranted, StatusDegraded, cutoff)
		if scanErr := row.Scan(&res.Count, &res.Bytes); scanErr != nil {
			return fmt.Errorf("count terminal capacity claims: %w", scanErr)
		}
		if res.Count == 0 {
			return nil
		}
		if _, execErr := tx.ExecContext(ctx, `
DELETE FROM capacity_claims
WHERE status NOT IN (?, ?, ?) AND updated_at < ?`,
			StatusReserved, StatusGranted, StatusDegraded, cutoff); execErr != nil {
			return fmt.Errorf("prune terminal capacity claims: %w", execErr)
		}
		return nil
	})
	if err != nil {
		return GCResult{}, err
	}
	return res, nil
}

// RecordObserved persists a usage-sampling result on an active claim (§Phase 2):
// the latest observed bytes, the decaying peak, and the sample time. It is pure
// telemetry — it does NOT bump the generation (so it never invalidates a pending
// activity report or arbitration decision) and only writes active claims (a
// terminal claim is never resurrected). Observed usage NEVER feeds Decide
// (contract C1).
func (s *SQLiteStore) RecordObserved(ctx context.Context, claimID string, observed, peak int64, at time.Time, absence *AbsenceProof) (CapacityClaim, error) {
	if strings.TrimSpace(claimID) == "" {
		return CapacityClaim{}, fmt.Errorf("%w: claim_id is required", ErrInvalidClaim)
	}
	var out CapacityClaim
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		if observed == 0 {
			if err := acquireClaimWriteAuthority(ctx, tx); err != nil {
				return err
			}
			current, err := getClaimTx(ctx, tx, claimID)
			if err != nil {
				return err
			}
			if absence == nil || !absence.validFor(current, s.now()) {
				return nil
			}
		}
		result, execErr := tx.ExecContext(ctx, `
UPDATE capacity_claims
SET observed_bytes = ?, observed_peak_bytes = ?, observed_at = ?
WHERE claim_id = ? AND status IN (?, ?, ?)`,
			observed, peak, formatTime(at.UTC()), claimID, StatusReserved, StatusGranted, StatusDegraded)
		if execErr != nil {
			return fmt.Errorf("record observed usage: %w", execErr)
		}
		affected, raErr := result.RowsAffected()
		if raErr != nil {
			return fmt.Errorf("inspect observed-usage update: %w", raErr)
		}
		if affected == 0 {
			// Claim terminal or absent — telemetry on a dead row is a silent no-op.
			return nil
		}
		out, execErr = getClaimTx(ctx, tx, claimID)
		return execErr
	})
	if err != nil {
		return CapacityClaim{}, err
	}
	return out, nil
}

func (s *SQLiteStore) RecordFootprint(ctx context.Context, observation FootprintObservation) (Footprint, error) {
	if strings.TrimSpace(observation.Resource) == "" || strings.TrimSpace(observation.Rung) == "" {
		return Footprint{}, fmt.Errorf("%w: footprint resource and rung are required", ErrInvalidClaim)
	}
	if observation.Bytes < 0 {
		return Footprint{}, fmt.Errorf("%w: footprint bytes must be non-negative", ErrInvalidClaim)
	}
	if observation.Source != FootprintSourceManifestDefault && observation.Source != FootprintSourceMeasured {
		return Footprint{}, fmt.Errorf("%w: footprint source %q is invalid", ErrInvalidClaim, observation.Source)
	}
	at := observation.ObservedAt.UTC()
	if at.IsZero() {
		at = s.now()
	}
	var out Footprint
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
INSERT INTO capacity_footprints(resource, rung, tunables_key, gpu_index, peak_bytes, samples, first_seen_at, last_seen_at, source)
VALUES (?, ?, ?, ?, ?, CASE WHEN ? = 'measured' THEN 1 ELSE 0 END, ?, ?, ?)
ON CONFLICT(resource, rung, tunables_key, gpu_index) DO UPDATE SET
  peak_bytes = MAX(capacity_footprints.peak_bytes, excluded.peak_bytes),
  samples = capacity_footprints.samples + CASE WHEN excluded.source = 'measured' THEN 1 ELSE 0 END,
  last_seen_at = excluded.last_seen_at,
  source = CASE WHEN excluded.peak_bytes > capacity_footprints.peak_bytes THEN excluded.source
    WHEN excluded.peak_bytes = capacity_footprints.peak_bytes AND excluded.source = 'measured' THEN 'measured'
    ELSE capacity_footprints.source END`,
			observation.Resource, observation.Rung, observation.TunablesKey, observation.GPUIndex,
			observation.Bytes, observation.Source, formatTime(at), formatTime(at), observation.Source)
		if err != nil {
			return fmt.Errorf("record capacity footprint: %w", err)
		}
		return scanFootprint(tx.QueryRowContext(ctx, `
SELECT resource, rung, tunables_key, gpu_index, peak_bytes, samples, first_seen_at, last_seen_at, source
FROM capacity_footprints WHERE resource = ? AND rung = ? AND tunables_key = ? AND gpu_index = ?`,
			observation.Resource, observation.Rung, observation.TunablesKey, observation.GPUIndex), &out)
	})
	return out, err
}

func (s *SQLiteStore) ListFootprints(ctx context.Context, filter FootprintFilter) ([]Footprint, error) {
	query := `SELECT resource, rung, tunables_key, gpu_index, peak_bytes, samples, first_seen_at, last_seen_at, source FROM capacity_footprints WHERE 1=1`
	var args []any
	if filter.Resource != "" {
		query += ` AND resource = ?`
		args = append(args, filter.Resource)
	}
	if filter.Rung != "" {
		query += ` AND rung = ?`
		args = append(args, filter.Rung)
	}
	if filter.TunablesKey != nil {
		query += ` AND tunables_key = ?`
		args = append(args, *filter.TunablesKey)
	}
	if filter.GPUIndex != nil {
		query += ` AND gpu_index = ?`
		args = append(args, *filter.GPUIndex)
	}
	query += ` ORDER BY resource, peak_bytes DESC, rung, tunables_key, gpu_index`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list capacity footprints: %w", err)
	}
	defer rows.Close()
	var out []Footprint
	for rows.Next() {
		var footprint Footprint
		if err := scanFootprint(rows, &footprint); err != nil {
			return nil, err
		}
		out = append(out, footprint)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) ResetFootprints(ctx context.Context, resource string) (int64, error) {
	query := `DELETE FROM capacity_footprints`
	var args []any
	if strings.TrimSpace(resource) != "" {
		query += ` WHERE resource = ?`
		args = append(args, strings.TrimSpace(resource))
	}
	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("reset capacity footprints: %w", err)
	}
	return result.RowsAffected()
}

func scanFootprint(row rowScanner, out *Footprint) error {
	var firstSeen, lastSeen string
	if err := row.Scan(&out.Resource, &out.Rung, &out.TunablesKey, &out.GPUIndex, &out.PeakBytes, &out.Samples, &firstSeen, &lastSeen, &out.Source); err != nil {
		return err
	}
	var err error
	if out.FirstSeenAt, err = parseRequiredTime(firstSeen); err != nil {
		return err
	}
	out.LastSeenAt, err = parseRequiredTime(lastSeen)
	return err
}

// GetClaim returns a single claim by ID.
func (s *SQLiteStore) GetClaim(ctx context.Context, claimID string) (CapacityClaim, error) {
	var out CapacityClaim
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		claim, getErr := getClaimTx(ctx, tx, claimID)
		if getErr != nil {
			return getErr
		}
		out = claim
		return nil
	})
	if err != nil {
		return CapacityClaim{}, err
	}
	return out, nil
}

// ListClaims returns claims matching the filter, newest first.
func (s *SQLiteStore) ListClaims(ctx context.Context, filter ClaimFilter) ([]CapacityClaim, error) {
	query := claimSelectSQL
	var clauses []string
	var args []any
	if filter.OwnerKind != "" {
		clauses = append(clauses, "owner_kind = ?")
		args = append(args, filter.OwnerKind)
	}
	if filter.OwnerID != "" {
		clauses = append(clauses, "owner_id = ?")
		args = append(args, filter.OwnerID)
	}
	if filter.OwnerIDPrefix != "" {
		clauses = append(clauses, "owner_id LIKE ?")
		args = append(args, filter.OwnerIDPrefix+"%")
	}
	if filter.ResourceKind != "" {
		clauses = append(clauses, "resource_kind = ?")
		args = append(args, filter.ResourceKind)
	}
	if filter.GPUIndex != nil {
		clauses = append(clauses, "gpu_index = ?")
		args = append(args, *filter.GPUIndex)
	}
	if len(filter.Statuses) > 0 {
		clauses = append(clauses, "status IN ("+placeholders(len(filter.Statuses))+")")
		for _, st := range filter.Statuses {
			args = append(args, st)
		}
	}
	if len(clauses) > 0 {
		query += "\nWHERE " + strings.Join(clauses, " AND ")
	}
	query += "\nORDER BY created_at DESC, claim_id ASC"

	var out []CapacityClaim
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		rows, queryErr := tx.QueryContext(ctx, query, args...)
		if queryErr != nil {
			return fmt.Errorf("list capacity claims: %w", queryErr)
		}
		var scanErr error
		out, scanErr = scanClaims(rows)
		if closeErr := rows.Close(); closeErr != nil && scanErr == nil {
			scanErr = closeErr
		}
		return scanErr
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetPolicy reads the stored policy, falling back to DefaultPolicy for any key
// not yet set.
func (s *SQLiteStore) GetPolicy(ctx context.Context) (Policy, error) {
	policy := DefaultPolicy()
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		rows, queryErr := tx.QueryContext(ctx, `SELECT key, value FROM capacity_policy`)
		if queryErr != nil {
			return fmt.Errorf("read capacity policy: %w", queryErr)
		}
		defer rows.Close()
		for rows.Next() {
			var key, value string
			if scanErr := rows.Scan(&key, &value); scanErr != nil {
				return scanErr
			}
			next, applyErr := policy.withKey(key, value)
			if applyErr != nil {
				// A stored value that no longer validates is skipped rather than
				// poisoning the whole read; the default for that key stands.
				continue
			}
			policy = next
		}
		return rows.Err()
	})
	if err != nil {
		return Policy{}, err
	}
	return policy, nil
}

// SetPolicyKey validates and persists a single policy key, returning the
// resulting effective policy.
func (s *SQLiteStore) SetPolicyKey(ctx context.Context, key, value string) (Policy, error) {
	current, err := s.GetPolicy(ctx)
	if err != nil {
		return Policy{}, err
	}
	next, err := current.withKey(key, value)
	if err != nil {
		return Policy{}, err
	}
	// Re-derive the canonical stored value from the validated policy so a value
	// like "60s" round-trips consistently.
	stored, err := next.Get(key)
	if err != nil {
		return Policy{}, err
	}
	now := s.now()
	err = s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		_, execErr := tx.ExecContext(ctx, `
INSERT INTO capacity_policy (key, value, updated_at) VALUES (?, ?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
			key, stored, formatTime(now))
		if execErr != nil {
			return fmt.Errorf("set capacity policy %s: %w", key, execErr)
		}
		return nil
	})
	if err != nil {
		return Policy{}, err
	}
	return next, nil
}

// sweepCursorKey is the reserved capacity_policy row holding the last
// resident-claim sweep time (§8.6). It is NOT a tunable lever: withKey rejects
// it as unknown, so GetPolicy skips it and `policy get` never lists it. Storing
// it in the existing key/value table avoids a schema migration while letting the
// debounce survive across the short-lived CLI/maintenance processes that drive
// the sweep.
const sweepCursorKey = "__sweep_last_at"

// LastSweepAt returns the last recorded resident-claim sweep time. ok is false
// (with a zero time, nil error) when no sweep has been recorded yet.
func (s *SQLiteStore) LastSweepAt(ctx context.Context) (time.Time, bool, error) {
	var raw string
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, `SELECT value FROM capacity_policy WHERE key = ?`, sweepCursorKey).Scan(&raw)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, err
	}
	t, perr := parseRequiredTime(raw)
	if perr != nil || t.IsZero() {
		return time.Time{}, false, nil
	}
	return t.UTC(), true, nil
}

// RecordSweepAt stamps the time of the most recent sweep so the opportunistic
// callers can debounce to policy.SweepInterval.
func (s *SQLiteStore) RecordSweepAt(ctx context.Context, at time.Time) error {
	return s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		_, execErr := tx.ExecContext(ctx, `
INSERT INTO capacity_policy (key, value, updated_at) VALUES (?, ?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
			sweepCursorKey, formatTime(at.UTC()), formatTime(s.now()))
		if execErr != nil {
			return fmt.Errorf("record capacity sweep cursor: %w", execErr)
		}
		return nil
	})
}

// --- transaction + scan helpers (mirror scenarioruntime) ---

// acquireClaimWriteAuthority prevents database waits from making a claim's
// mutation-time clock stale. BeginTx is deferred and does not suffice: UPDATE
// may still wait behind another writer. This no-row write changes no claim or
// trigger, and existing retries reacquire authority before checking fresh time.
// Only the three time-sensitive claim mutations use this owner-local boundary.
func acquireClaimWriteAuthority(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, "UPDATE capacity_claims SET claim_id=claim_id WHERE 0"); err != nil {
		return fmt.Errorf("acquire capacity claim write authority: %w", err)
	}
	return nil
}

func (s *SQLiteStore) withTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin capacity ledger transaction: %w", err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit capacity ledger transaction: %w", err)
	}
	return nil
}

func (s *SQLiteStore) withRetryableTx(ctx context.Context, fn func(*sql.Tx) error) error {
	var err error
	delay := capacityTxRetryBase
	for attempt := 1; attempt <= capacityTxRetryAttempts; attempt++ {
		err = s.withTx(ctx, fn)
		if err == nil || !isSQLiteLockContention(err) || attempt == capacityTxRetryAttempts {
			return err
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		delay *= 2
		if delay > capacityTxRetryMax {
			delay = capacityTxRetryMax
		}
	}
	return err
}

func isSQLiteLockContention(err error) bool {
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) {
		switch sqliteErr.Code() & 0xff {
		case sqlite3.SQLITE_BUSY, sqlite3.SQLITE_LOCKED:
			return true
		}
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") || strings.Contains(msg, "table in the database is locked")
}

// finishMutation interprets a single-row UPDATE result: 0 rows affected means
// either a stale generation or an absent claim; otherwise it re-reads the row.
func finishMutation(ctx context.Context, tx *sql.Tx, result sql.Result, claimID string, out *CapacityClaim) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect capacity claim mutation: %w", err)
	}
	if affected == 0 {
		if _, getErr := getClaimTx(ctx, tx, claimID); getErr != nil {
			return getErr
		}
		return ErrStaleGeneration
	}
	updated, err := getClaimTx(ctx, tx, claimID)
	if err != nil {
		return err
	}
	*out = updated
	return nil
}

const claimSelectSQL = `
SELECT claim_id, owner_kind, owner_id, instance_id, resource_kind, gpu_index,
  amount_bytes, preferred_bytes, floor_bytes, priority, protected, yield_when_idle, status,
  activity_state, generation, created_at, updated_at, last_heartbeat_at,
  heartbeat_deadline_at, last_active_at, degrade_profile,
  observed_bytes, observed_peak_bytes, observed_at, idle_unload_ttl_seconds, idle_grace_seconds
FROM capacity_claims`

func getClaimTx(ctx context.Context, tx *sql.Tx, claimID string) (CapacityClaim, error) {
	row := tx.QueryRowContext(ctx, claimSelectSQL+`
WHERE claim_id = ?`, claimID)
	claim, err := scanClaimRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return CapacityClaim{}, ErrNotFound
	}
	return claim, err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanClaimRow(row rowScanner) (CapacityClaim, error) {
	var (
		c            CapacityClaim
		gpuIndex     sql.NullInt64
		protected    int64
		yieldOnIdle  int64
		lastHB       sql.NullString
		hbDeadline   sql.NullString
		lastActive   sql.NullString
		profile      string
		created      string
		updated      string
		observedAt   sql.NullString
		idleUnloadTS int64
		idleGraceS   int64
	)
	if err := row.Scan(
		&c.ClaimID, &c.OwnerKind, &c.OwnerID, &c.InstanceID, &c.ResourceKind, &gpuIndex,
		&c.AmountBytes, &c.PreferredBytes, &c.FloorBytes, &c.Priority, &protected, &yieldOnIdle, &c.Status,
		&c.ActivityState, &c.Generation, &created, &updated, &lastHB,
		&hbDeadline, &lastActive, &profile,
		&c.ObservedBytes, &c.ObservedPeakBytes, &observedAt, &idleUnloadTS, &idleGraceS,
	); err != nil {
		return CapacityClaim{}, err
	}
	c.IdleUnloadTTL = time.Duration(idleUnloadTS) * time.Second
	c.IdleGrace = time.Duration(idleGraceS) * time.Second
	if gpuIndex.Valid {
		idx := int(gpuIndex.Int64)
		c.GPUIndex = &idx
	}
	c.Protected = protected != 0
	c.YieldWhenIdle = yieldOnIdle != 0
	var err error
	if c.CreatedAt, err = parseRequiredTime(created); err != nil {
		return CapacityClaim{}, err
	}
	if c.UpdatedAt, err = parseRequiredTime(updated); err != nil {
		return CapacityClaim{}, err
	}
	if c.LastHeartbeatAt, err = parseOptionalTime(lastHB); err != nil {
		return CapacityClaim{}, err
	}
	if c.HeartbeatDeadlineAt, err = parseOptionalTime(hbDeadline); err != nil {
		return CapacityClaim{}, err
	}
	if c.LastActiveAt, err = parseOptionalTime(lastActive); err != nil {
		return CapacityClaim{}, err
	}
	if c.ObservedAt, err = parseOptionalTime(observedAt); err != nil {
		return CapacityClaim{}, err
	}
	if c.DegradeProfile, err = unmarshalProfile(profile); err != nil {
		return CapacityClaim{}, err
	}
	return c, nil
}

func scanClaims(rows *sql.Rows) ([]CapacityClaim, error) {
	var out []CapacityClaim
	for rows.Next() {
		claim, err := scanClaimRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, claim)
	}
	return out, rows.Err()
}

func marshalProfile(p *DegradeProfile) (string, error) {
	if p == nil {
		return "", nil
	}
	b, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("marshal degrade profile: %w", err)
	}
	return string(b), nil
}

func unmarshalProfile(s string) (*DegradeProfile, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	var p DegradeProfile
	if err := json.Unmarshal([]byte(s), &p); err != nil {
		return nil, fmt.Errorf("unmarshal degrade profile: %w", err)
	}
	return &p, nil
}

func buildDSN(path string) string {
	return "file:" + url.PathEscape(path) + "?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)"
}

func buildReadOnlyDSN(path string) string {
	if _, err := os.Stat(path + "-wal"); err != nil {
		return "file:" + url.PathEscape(path) + "?mode=ro&immutable=1&_pragma=foreign_keys(ON)&_pragma=query_only(ON)&_pragma=busy_timeout(10000)&_pragma=temp_store(MEMORY)"
	}
	return "file:" + url.PathEscape(path) + "?mode=ro&nolock=1&_pragma=foreign_keys(ON)&_pragma=query_only(ON)&_pragma=busy_timeout(10000)&_pragma=temp_store(MEMORY)"
}

func newID(prefix string) string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(b[:])
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", n), ",")
}

func optionalIntValue(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

var (
	formatTime         = storagetime.FormatUTC
	formatOptionalTime = storagetime.FormatOptionalUTC
)

func parseOptionalTime(v sql.NullString) (*time.Time, error) {
	if !v.Valid || strings.TrimSpace(v.String) == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339Nano, v.String)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func parseRequiredTime(v string) (time.Time, error) {
	if strings.TrimSpace(v) == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339Nano, v)
}
