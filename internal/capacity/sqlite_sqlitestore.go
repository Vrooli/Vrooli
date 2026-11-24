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
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/sqliteutil"
	"github.com/vrooli/vrooli/internal/storagetime"
)

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
	if out.FirstSeenAt, err = sqliteutil.ParseRequiredTime(firstSeen); err != nil {
		return err
	}
	out.LastSeenAt, err = sqliteutil.ParseRequiredTime(lastSeen)
	return err
}

// GetClaim returns a single claim by ID.
func (s *SQLiteStore) GetClaim(ctx context.Context, claimID string) (CapacityClaim, error) {
	return getClaimRow(ctx, s.db, claimID)
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
		clauses = append(clauses, "status IN ("+sqliteutil.Placeholders(len(filter.Statuses))+")")
		for _, st := range filter.Statuses {
			args = append(args, st)
		}
	}
	if len(clauses) > 0 {
		query += "\nWHERE " + strings.Join(clauses, " AND ")
	}
	query += "\nORDER BY created_at DESC, claim_id ASC"

	var out []CapacityClaim
	// A read runs outside the connection's immediate (write) transaction lock.
	// _txlock=immediate is correct for mutations, but forcing every pure read
	// through it made list/get contend with heartbeat writers and surface as
	// SQLITE_BUSY on a busy host.
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list capacity claims: %w", err)
	}
	out, err = scanClaims(rows)
	if closeErr := rows.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetPolicy reads the stored policy, falling back to DefaultPolicy for any key
// not yet set.
func (s *SQLiteStore) GetPolicy(ctx context.Context) (Policy, error) {
	policy := DefaultPolicy()
	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM capacity_policy`)
	if err != nil {
		return Policy{}, fmt.Errorf("read capacity policy: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var key, value string
		if scanErr := rows.Scan(&key, &value); scanErr != nil {
			return Policy{}, scanErr
		}
		next, applyErr := policy.withKey(key, value)
		if applyErr != nil {
			// A stored value that no longer validates is skipped rather than
			// poisoning the whole read; the default for that key stands.
			continue
		}
		policy = next
	}
	if err := rows.Err(); err != nil {
		return Policy{}, err
	}
	return policy, nil
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
	err := s.db.QueryRowContext(ctx, `SELECT value FROM capacity_policy WHERE key = ?`, sweepCursorKey).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, err
	}
	t, perr := sqliteutil.ParseRequiredTime(raw)
	if perr != nil || t.IsZero() {
		return time.Time{}, false, nil
	}
	return t.UTC(), true, nil
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
		if err == nil || !sqliteutil.IsLockContention(err) || attempt == capacityTxRetryAttempts {
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

// staleGenerationError is ErrStaleGeneration naming the claim's current
// generation and status, so the caller can retry once with the right value
// (or without the optional guard) instead of guessing.
func staleGenerationError(current CapacityClaim) error {
	return fmt.Errorf("%w: claim %s is %s at generation %d; re-read it with `vrooli capacity list --json` or omit --generation", ErrStaleGeneration, current.ClaimID, current.Status, current.Generation)
}

// finishMutation interprets a single-row UPDATE result: 0 rows affected means
// either a stale generation or an absent claim; otherwise it re-reads the row.
func finishMutation(ctx context.Context, tx *sql.Tx, result sql.Result, claimID string, out *CapacityClaim) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect capacity claim mutation: %w", err)
	}
	if affected == 0 {
		current, getErr := getClaimTx(ctx, tx, claimID)
		if getErr != nil {
			return getErr
		}
		return staleGenerationError(current)
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
	return getClaimRow(ctx, tx, claimID)
}

// queryRower is the read surface shared by a pool handle and a transaction, so
// the same scan serves a direct read and a write transaction's re-read.
type queryRower interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func getClaimRow(ctx context.Context, q queryRower, claimID string) (CapacityClaim, error) {
	row := q.QueryRowContext(ctx, claimSelectSQL+`
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
	if c.CreatedAt, err = sqliteutil.ParseRequiredTime(created); err != nil {
		return CapacityClaim{}, err
	}
	if c.UpdatedAt, err = sqliteutil.ParseRequiredTime(updated); err != nil {
		return CapacityClaim{}, err
	}
	if c.LastHeartbeatAt, err = sqliteutil.ParseOptionalTime(lastHB); err != nil {
		return CapacityClaim{}, err
	}
	if c.HeartbeatDeadlineAt, err = sqliteutil.ParseOptionalTime(hbDeadline); err != nil {
		return CapacityClaim{}, err
	}
	if c.LastActiveAt, err = sqliteutil.ParseOptionalTime(lastActive); err != nil {
		return CapacityClaim{}, err
	}
	if c.ObservedAt, err = sqliteutil.ParseOptionalTime(observedAt); err != nil {
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
	// Capacity writers read before they update claims. In WAL mode, a deferred
	// transaction can lose its snapshot upgrade with SQLITE_BUSY_SNAPSHOT, which
	// bypasses busy_timeout. Reserve the writer at BEGIN so competing processes
	// queue under the existing timeout instead.
	return "file:" + url.PathEscape(path) + "?_txlock=immediate&_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)"
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

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func formatTime(t time.Time) string       { return storagetime.FormatUTC(t) }
func formatOptionalTime(t *time.Time) any { return storagetime.FormatOptionalUTC(t) }
