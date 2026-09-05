package metrics

import (
	"context"
	"fmt"
	"time"

	"search-hub/internal/routing"

	"github.com/vrooli/api-core/schedule"
)

type sqliteDemotionStore struct {
	db    SQLExecutor
	clock schedule.Clock
}

func NewSQLiteDemotionStore(db SQLExecutor, clk schedule.Clock) routing.DemotionStore {
	return &sqliteDemotionStore{db: db, clock: clk}
}

func (s *sqliteDemotionStore) Load(ctx context.Context) ([]routing.ProviderDemotionState, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT provider_id, routed, hits, empty_streak, demoted, probation, decay_deadline, trigger FROM provider_demotion_state ORDER BY provider_id`)
	if err != nil {
		return nil, fmt.Errorf("load provider demotion state: %w", err)
	}
	defer rows.Close()
	var out []routing.ProviderDemotionState
	for rows.Next() {
		var state routing.ProviderDemotionState
		var demoted, probation int
		var deadline string
		if err := rows.Scan(&state.ProviderID, &state.Routed, &state.Hits, &state.EmptyStreak, &demoted, &probation, &deadline, &state.Trigger); err != nil {
			return nil, fmt.Errorf("scan provider demotion state: %w", err)
		}
		state.Demoted, state.Probation = demoted != 0, probation != 0
		if deadline != "" {
			state.DecayDeadline, _ = time.Parse(time.RFC3339Nano, deadline)
		}
		out = append(out, state)
	}
	return out, rows.Err()
}

func (s *sqliteDemotionStore) Save(ctx context.Context, state routing.ProviderDemotionState) error {
	deadline := ""
	if !state.DecayDeadline.IsZero() {
		deadline = state.DecayDeadline.UTC().Format(time.RFC3339Nano)
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO provider_demotion_state (provider_id, routed, hits, empty_streak, demoted, probation, decay_deadline, trigger, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(provider_id) DO UPDATE SET routed=excluded.routed, hits=excluded.hits, empty_streak=excluded.empty_streak, demoted=excluded.demoted, probation=excluded.probation, decay_deadline=excluded.decay_deadline, trigger=excluded.trigger, updated_at=excluded.updated_at`,
		state.ProviderID, state.Routed, state.Hits, state.EmptyStreak, boolToInt(state.Demoted), boolToInt(state.Probation), deadline, state.Trigger, s.clock.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (s *sqliteDemotionStore) StuckProviderCount(ctx context.Context, at time.Time) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM provider_demotion_state WHERE demoted = 1 AND probation = 1 AND decay_deadline <> '' AND decay_deadline <= ?`, at.UTC().Format(time.RFC3339Nano)).Scan(&count)
	return count, err
}
