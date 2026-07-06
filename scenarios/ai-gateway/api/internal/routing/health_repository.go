package routing

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

// HealthRepository persists provider-health/circuit-breaker records. A nil
// repository disables breaker tracking (every route is treated as closed and
// no state is recorded), which keeps unit tests that do not exercise health
// simple and preserves the pre-breaker behavior for callers that opt out.
type HealthRepository interface {
	Get(ctx context.Context, key HealthKey) (ProviderHealth, bool, error)
	Upsert(ctx context.Context, h ProviderHealth) error
	List(ctx context.Context) ([]ProviderHealth, error)
}

// SQLHealthRepository stores provider health in the route-evidence SQLite
// database alongside route_events.
type SQLHealthRepository struct {
	db *sql.DB
}

func NewSQLHealthRepository(db *sql.DB) *SQLHealthRepository {
	return &SQLHealthRepository{db: db}
}

func (r *SQLHealthRepository) Get(ctx context.Context, key HealthKey) (ProviderHealth, bool, error) {
	if r == nil || r.db == nil {
		return ProviderHealth{}, false, fmt.Errorf("provider health repository is not configured")
	}
	key = normalizeHealthKey(key)
	row := r.db.QueryRowContext(ctx, selectHealthSQL+" WHERE provider = ? AND role = ? AND kind = ?",
		key.Provider, key.Role, int32(key.Kind))
	h, err := scanHealth(row)
	if errors.Is(err, sql.ErrNoRows) {
		return ProviderHealth{}, false, nil
	}
	if err != nil {
		return ProviderHealth{}, false, err
	}
	return h, true, nil
}

func (r *SQLHealthRepository) Upsert(ctx context.Context, h ProviderHealth) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("provider health repository is not configured")
	}
	key := normalizeHealthKey(HealthKey{Provider: h.Provider, Role: h.Role, Kind: h.Kind})
	_, err := r.db.ExecContext(ctx, `INSERT INTO provider_health (
provider, role, kind, state, consecutive_failures, last_failure_class,
last_success_at, last_failure_at, cooldown_until, opened_at, generation, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(provider, role, kind) DO UPDATE SET
state = excluded.state,
consecutive_failures = excluded.consecutive_failures,
last_failure_class = excluded.last_failure_class,
last_success_at = excluded.last_success_at,
last_failure_at = excluded.last_failure_at,
cooldown_until = excluded.cooldown_until,
opened_at = excluded.opened_at,
generation = excluded.generation,
updated_at = excluded.updated_at`,
		key.Provider,
		key.Role,
		int32(key.Kind),
		string(stateOrClosed(h.State)),
		h.ConsecutiveFailures,
		string(h.LastFailureClass),
		formatTime(h.LastSuccessAt),
		formatTime(h.LastFailureAt),
		formatTime(h.CooldownUntil),
		formatTime(h.OpenedAt),
		h.Generation,
		formatTime(h.UpdatedAt),
	)
	return err
}

func (r *SQLHealthRepository) List(ctx context.Context) ([]ProviderHealth, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("provider health repository is not configured")
	}
	rows, err := r.db.QueryContext(ctx, selectHealthSQL+" ORDER BY provider, role, kind")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProviderHealth
	for rows.Next() {
		h, err := scanHealth(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

const selectHealthSQL = `SELECT provider, role, kind, state, consecutive_failures, last_failure_class,
last_success_at, last_failure_at, cooldown_until, opened_at, generation, updated_at
FROM provider_health`

func scanHealth(s scanner) (ProviderHealth, error) {
	var h ProviderHealth
	var kind int32
	var state, failureClass string
	var lastSuccess, lastFailure, cooldown, opened, updated string
	err := s.Scan(
		&h.Provider,
		&h.Role,
		&kind,
		&state,
		&h.ConsecutiveFailures,
		&failureClass,
		&lastSuccess,
		&lastFailure,
		&cooldown,
		&opened,
		&h.Generation,
		&updated,
	)
	if err != nil {
		return ProviderHealth{}, err
	}
	h.Kind = sharedv1.RequestKind(kind)
	h.State = stateOrClosed(BreakerState(state))
	h.LastFailureClass = FailureClass(failureClass)
	h.LastSuccessAt = parseTime(lastSuccess)
	h.LastFailureAt = parseTime(lastFailure)
	h.CooldownUntil = parseTime(cooldown)
	h.OpenedAt = parseTime(opened)
	h.UpdatedAt = parseTime(updated)
	return h, nil
}

func stateOrClosed(s BreakerState) BreakerState {
	switch s {
	case BreakerOpen, BreakerHalfOpen, BreakerClosed:
		return s
	default:
		return BreakerClosed
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}
