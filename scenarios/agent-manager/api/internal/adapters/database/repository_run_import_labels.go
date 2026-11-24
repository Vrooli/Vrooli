package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var (
	_ repository.RunLabelRegenerationStore = (*runRepository)(nil)
	_ repository.RunImportPolicyScopeStore = (*runRepository)(nil)
)

func labelSourceValues(sources []domain.RunLabelSource) []string {
	values := make([]string, 0, len(sources))
	for _, source := range sources {
		if value := strings.TrimSpace(string(source)); value != "" {
			values = append(values, value)
		}
	}
	return values
}

// LabelRegenerationTargets lists imported runs whose label source is one of
// sources and that generator has not settled, newest first. The total counts
// every such run, so a bounded call can report what remains.
func (r *runRepository) LabelRegenerationTargets(ctx context.Context, generator string, sources []domain.RunLabelSource, limit int) ([]uuid.UUID, int, error) {
	values := labelSourceValues(sources)
	if strings.TrimSpace(generator) == "" || len(values) == 0 {
		return nil, 0, fmt.Errorf("label regeneration needs a generator revision and at least one label source")
	}
	const where = `FROM runs r
		WHERE r.execution_mode = ? AND r.label_source IN (?)
		  AND NOT EXISTS (SELECT 1 FROM run_label_generations g WHERE g.run_id = r.id AND g.generator = ?)`
	countQuery, countArgs, err := sqlx.In(`SELECT COUNT(*) `+where, string(domain.ExecutionModeImported), values, generator)
	if err != nil {
		return nil, 0, wrapDBError("count_label_regeneration_targets", "Run", "", err)
	}
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, countArgs...); err != nil {
		return nil, 0, wrapDBError("count_label_regeneration_targets", "Run", "", err)
	}
	if limit <= 0 || total == 0 {
		return nil, total, nil
	}
	listQuery, listArgs, err := sqlx.In(`SELECT r.id `+where+` ORDER BY r.created_at DESC, r.id DESC LIMIT ?`, string(domain.ExecutionModeImported), values, generator, limit)
	if err != nil {
		return nil, 0, wrapDBError("list_label_regeneration_targets", "Run", "", err)
	}
	var ids []uuid.UUID
	if err := r.db.SelectContext(ctx, &ids, listQuery, listArgs...); err != nil {
		return nil, 0, wrapDBError("list_label_regeneration_targets", "Run", "", err)
	}
	return ids, total, nil
}

// SettleRunLabel records a generator's outcome for one run and applies its
// label in the same transaction. The label update is conditional on the run's
// current label source, so it can only replace labels the caller declared
// replaceable.
func (r *runRepository) SettleRunLabel(ctx context.Context, id uuid.UUID, generation repository.RunLabelGeneration, label string, source domain.RunLabelSource, replaceable []domain.RunLabelSource) (bool, error) {
	generator, outcome := strings.TrimSpace(generation.Generator), strings.TrimSpace(generation.Outcome)
	if generator == "" || outcome == "" {
		return false, fmt.Errorf("run label generation needs a generator revision and outcome")
	}
	label = strings.TrimSpace(label)
	values := labelSourceValues(replaceable)
	if label != "" && (source == "" || len(values) == 0) {
		return false, fmt.Errorf("a settled run label needs its source and the sources it may replace")
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return false, wrapDBError("settle_label", "Run", id.String(), err)
	}
	defer tx.Rollback()
	now := SQLiteTime(time.Now())
	changed := false
	if label != "" {
		query, args, err := sqlx.In(`UPDATE runs SET label = ?, label_source = ?, updated_at = ? WHERE id = ? AND label_source IN (?)`, label, string(source), now, id, values)
		if err != nil {
			return false, wrapDBError("settle_label", "Run", id.String(), err)
		}
		result, err := tx.ExecContext(ctx, query, args...)
		if err != nil {
			return false, wrapDBError("settle_label", "Run", id.String(), err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return false, wrapDBError("settle_label", "Run", id.String(), err)
		}
		changed = affected == 1
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO run_label_generations (run_id, generator, outcome, recorded_at) VALUES (?, ?, ?, ?)
		ON CONFLICT(run_id) DO UPDATE SET generator = excluded.generator, outcome = excluded.outcome, recorded_at = excluded.recorded_at`,
		id, generator, outcome, now); err != nil {
		return false, wrapDBError("record_label_generation", "Run", id.String(), err)
	}
	if err := tx.Commit(); err != nil {
		return false, wrapDBError("settle_label", "Run", id.String(), err)
	}
	return changed, nil
}

// SetRunImportPolicyScope records the source scope a caller declared for an
// imported run. The latest declaration wins.
func (r *runRepository) SetRunImportPolicyScope(ctx context.Context, id uuid.UUID, sourceHarness string) error {
	sourceHarness = strings.TrimSpace(sourceHarness)
	if sourceHarness == "" {
		return fmt.Errorf("import policy scope is required")
	}
	if _, err := r.db.ExecContext(ctx, `INSERT INTO run_import_policy_scopes (run_id, source_harness, recorded_at) VALUES (?, ?, ?)
		ON CONFLICT(run_id) DO UPDATE SET source_harness = excluded.source_harness, recorded_at = excluded.recorded_at`,
		id, sourceHarness, SQLiteTime(time.Now())); err != nil {
		return wrapDBError("set_import_policy_scope", "Run", id.String(), err)
	}
	return nil
}

// RunImportPolicyScope returns the declared source scope of a run, or "".
func (r *runRepository) RunImportPolicyScope(ctx context.Context, id uuid.UUID) (string, error) {
	var scopes []string
	if err := r.db.SelectContext(ctx, &scopes, `SELECT source_harness FROM run_import_policy_scopes WHERE run_id = ?`, id); err != nil {
		return "", wrapDBError("get_import_policy_scope", "Run", id.String(), err)
	}
	if len(scopes) == 0 {
		return "", nil
	}
	return scopes[0], nil
}
