package classification

import (
	"context"
	"fmt"
	"time"

	"github.com/vrooli/api-core/database"
)

type SQLiteRepository struct{ db *database.RoutedDB }

func NewSQLiteRepository(db *database.RoutedDB) *SQLiteRepository { return &SQLiteRepository{db: db} }

const observations = `
p.position,e.id,e.scope,e.body,e.kind,COALESCE(c.evidence,''),COALESCE(c.routing_hint,''),COALESCE(e.import_key,'')
FROM journal_entry_positions p
JOIN entries e ON e.id=p.entry_id
LEFT JOIN journal_observation_context c ON c.entry_id=e.id
`

func (r *SQLiteRepository) prepare(ctx context.Context, def definition, now time.Time, limit int) (work, error) {
	var pending work
	var cutoff int64
	// Fix the historical boundary at the first scheduling of this definition.
	// A separate cursor gives newer appends capacity during the backfill.
	if _, err := r.db.ExecContext(ctx, `
INSERT INTO classification_progress(category_id,definition_version,definition_spec,history_through,live_position)
SELECT ?,?,?,max_rowid,max_rowid FROM journal_high_water_mark WHERE id=1
ON CONFLICT(category_id,definition_version) DO NOTHING
`, def.category.ID, def.version, def.spec); err != nil {
		return pending, fmt.Errorf("initialize classification progress: %w", err)
	}
	if err := r.db.QueryRowContext(ctx, `
SELECT history_position,history_through,live_position FROM classification_progress
WHERE category_id=? AND definition_version=?
`, def.category.ID, def.version).Scan(&pending.historyStart, &cutoff, &pending.liveStart); err != nil {
		return pending, fmt.Errorf("read classification progress: %w", err)
	}

	retries, err := r.readObservations(ctx, `SELECT q.attempts,`+observations+`
JOIN classification_retries q ON q.position=p.position
WHERE q.category_id=? AND q.definition_version=? AND q.next_attempt<=?
ORDER BY q.next_attempt,q.position LIMIT ?
`, def.category.ID, def.version, now.Unix(), limit/4)
	if err != nil {
		return pending, fmt.Errorf("read due classification retries: %w", err)
	}
	liveLimit := limit / 2
	if pending.historyStart >= cutoff {
		liveLimit = limit - len(retries)
	}
	live, err := r.readObservations(ctx, `SELECT 0,`+observations+`
WHERE p.position>? ORDER BY p.position LIMIT ?
`, pending.liveStart, liveLimit)
	if err != nil {
		return pending, fmt.Errorf("read journal appends for classification: %w", err)
	}
	var history []observation
	if pending.historyStart < cutoff {
		history, err = r.readObservations(ctx, `SELECT 0,`+observations+`
WHERE p.position>? AND p.position<=? ORDER BY p.position LIMIT ?
`, pending.historyStart, cutoff, limit-len(live)-len(retries))
		if err != nil {
			return pending, fmt.Errorf("read journal history for classification: %w", err)
		}
	}
	pending.liveEnd = lastPosition(live, pending.liveStart)
	pending.historyEnd = lastPosition(history, pending.historyStart)
	pending.items = append(append(retries, live...), history...)
	return pending, nil
}

func lastPosition(items []observation, fallback int64) int64 {
	if len(items) == 0 {
		return fallback
	}
	return items[len(items)-1].position
}

func (r *SQLiteRepository) readObservations(ctx context.Context, query string, args ...any) ([]observation, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []observation
	for rows.Next() {
		var item observation
		if err := rows.Scan(&item.attempts, &item.position, &item.id, &item.scope, &item.body, &item.kind, &item.evidence, &item.hint, &item.importKey); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *SQLiteRepository) save(ctx context.Context, def definition, pending work, answers []decision, now time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin classification outcome: %w", err)
	}
	defer tx.Rollback()
	category, version := def.category.ID, def.version
	// Reject work from another overlapping pass before it can overwrite newer
	// checkpoints. No transaction or database lock spans model inference.
	res, err := tx.ExecContext(ctx, `
UPDATE classification_progress SET history_position=?,live_position=?
WHERE category_id=? AND definition_version=? AND history_position=? AND live_position=?
`, pending.historyEnd, pending.liveEnd, category, version, pending.historyStart, pending.liveStart)
	if err != nil {
		return fmt.Errorf("advance classification progress: %w", err)
	}
	updated, err := res.RowsAffected()
	if err != nil || updated == 0 {
		return err
	}
	for index, item := range pending.items {
		if item.attempts > 0 {
			res, err := tx.ExecContext(ctx, `
DELETE FROM classification_retries WHERE category_id=? AND definition_version=? AND position=? AND attempts=?
`, category, version, item.position, item.attempts)
			if err != nil {
				return fmt.Errorf("claim classification retry outcome: %w", err)
			}
			deleted, err := res.RowsAffected()
			if err != nil {
				return err
			}
			if deleted == 0 {
				continue
			}
		}
		answer := answers[index]
		if answer.failure != "" {
			delay := min(time.Hour, time.Minute<<min(item.attempts, 6))
			if _, err := tx.ExecContext(ctx, `
INSERT INTO classification_retries(category_id,definition_version,position,attempts,next_attempt,reason)
VALUES(?,?,?,?,?,?)
`, category, version, item.position, item.attempts+1, now.Add(delay).Unix(), answer.failure); err != nil {
				return fmt.Errorf("defer classification: %w", err)
			}
			continue
		}
		if !answer.match {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO classification_matches(category_id,definition_version,scope,entry_id,decision_method,provider,model)
VALUES(?,?,?,?,?,?,?)
ON CONFLICT(category_id,definition_version,scope,entry_id) DO NOTHING
`, category, version, item.scope, item.id, answer.method, answer.provider, answer.model); err != nil {
			return fmt.Errorf("store classification match: %w", err)
		}
	}
	return tx.Commit()
}

func (r *SQLiteRepository) matchingEntries(ctx context.Context, scope string, def definition) ([]string, error) {
	query := `SELECT entry_id FROM classification_matches WHERE category_id=? AND definition_version=? AND scope=?`
	args := []any{def.category.ID, def.version, scope}
	for _, rule := range def.category.Sources {
		// A typed source is already classified by its owner. A scoped key
		// prefix range can serve that decision while model backfill catches up.
		prefix := "request:" + rule.RequestKeyPrefix
		upper := prefix[:len(prefix)-1] + "0" // Valid source prefixes end in '/'.
		query += `
UNION SELECT id FROM entries WHERE scope=? AND kind=? AND import_key>=? AND import_key<?`
		args = append(args, scope, rule.Kind, prefix, upper)
	}
	rows, err := r.db.Reader().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("read classification matches: %w", err)
	}
	defer rows.Close()
	var entries []string
	for rows.Next() {
		var entry string
		if err := rows.Scan(&entry); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (r *SQLiteRepository) coverage(ctx context.Context, scope string, def definition) (bool, error) {
	var pending bool
	err := r.db.Reader().QueryRowContext(ctx, `
SELECT CASE
  WHEN cp.category_id IS NULL THEN EXISTS (SELECT 1 FROM entries WHERE scope=?3)
  ELSE (
    ((cp.history_position<cp.history_through OR cp.live_position<wm.max_rowid) AND EXISTS (
      SELECT 1 FROM entries e JOIN journal_entry_positions p ON p.entry_id=e.id
      WHERE e.scope=?3 AND (p.position>cp.live_position OR
            (p.position>cp.history_position AND p.position<=cp.history_through))
    )) OR EXISTS (
      SELECT 1 FROM classification_retries q
      JOIN journal_entry_positions p ON p.position=q.position JOIN entries e ON e.id=p.entry_id
      WHERE q.category_id=?1 AND q.definition_version=?2 AND e.scope=?3
    )
  )
END
FROM journal_high_water_mark wm
LEFT JOIN classification_progress cp ON cp.category_id=?1 AND cp.definition_version=?2
WHERE wm.id=1
`, def.category.ID, def.version, scope).Scan(&pending)
	if err != nil {
		return false, fmt.Errorf("read classification coverage: %w", err)
	}
	return pending, nil
}

func (r *SQLiteRepository) notices(ctx context.Context, scope string, def definition, after int64, limit int) ([]Notice, error) {
	// Category recall uses every match. Deliver deliberate observations and
	// explicitly routed evidence to notice consumers. Existing work records,
	// imports, and generated accounts should not open intake on their own.
	rows, err := r.db.Reader().QueryContext(ctx, `
SELECT m.sequence,m.entry_id FROM classification_matches m
JOIN entries e ON e.id=m.entry_id
LEFT JOIN journal_observation_context c ON c.entry_id=e.id
WHERE m.scope=? AND m.category_id=? AND m.definition_version=? AND m.sequence>?
AND (e.kind='observation' OR COALESCE(TRIM(c.routing_hint),'') NOT IN ('','defect-account'))
AND NOT (e.kind='observation' AND COALESCE(c.routing_hint,'')='defect-account'
         AND COALESCE(e.import_key,'') GLOB 'request:remember/scenario-qa/defect-*/*/0')
ORDER BY m.sequence LIMIT ?
`, scope, def.category.ID, def.version, after, limit)
	if err != nil {
		return nil, fmt.Errorf("read classification notices: %w", err)
	}
	defer rows.Close()
	var notices []Notice
	for rows.Next() {
		notice := Notice{CategoryID: def.category.ID, Scope: scope}
		if err := rows.Scan(&notice.Sequence, &notice.ObservationID); err != nil {
			return nil, err
		}
		notices = append(notices, notice)
	}
	return notices, rows.Err()
}
