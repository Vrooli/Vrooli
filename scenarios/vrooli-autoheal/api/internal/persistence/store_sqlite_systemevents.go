package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/systemevents"
)

func (s *Store) getTimelineEventsSQLite(ctx context.Context, limit int) ([]TimelineEvent, error) {
	query := `
		SELECT check_id, status, message, details, created_at
		FROM health_results
		ORDER BY created_at DESC
		LIMIT ?
	`
	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var events []TimelineEvent
	for rows.Next() {
		var e TimelineEvent
		var detailsJSON []byte
		var createdRaw any

		if err := rows.Scan(&e.CheckID, &e.Status, &e.Message, &detailsJSON, &createdRaw); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		ts, err := parseDBTime(createdRaw)
		if err != nil {
			return nil, fmt.Errorf("parse created_at: %w", err)
		}
		e.Timestamp = ts.UTC().Format(time.RFC3339)

		if len(detailsJSON) > 0 {
			if err := json.Unmarshal(detailsJSON, &e.Details); err != nil {
				return nil, fmt.Errorf("unmarshal timeline details failed: %w", err)
			}
		}

		events = append(events, e)
	}

	return events, rows.Err()
}

func (s *Store) upsertSystemEventsSQLite(ctx context.Context, events []systemevents.Event) (int, int, error) {
	if len(events) == 0 {
		return 0, 0, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR IGNORE INTO system_events (
			fingerprint, occurred_at, ingested_at, source, platform, category, severity, title, summary, boot_id, details_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return 0, 0, err
	}
	defer stmt.Close()

	inserted := 0
	for _, event := range events {
		details, err := json.Marshal(event.Details)
		if err != nil {
			details = []byte("{}")
		}
		result, err := stmt.ExecContext(
			ctx,
			event.Fingerprint,
			event.OccurredAt.UTC().Format(time.RFC3339Nano),
			event.IngestedAt.UTC().Format(time.RFC3339Nano),
			event.Source,
			event.Platform,
			event.Category,
			event.Severity,
			event.Title,
			event.Summary,
			event.BootID,
			details,
		)
		if err != nil {
			return 0, 0, err
		}
		rows, _ := result.RowsAffected()
		if rows > 0 {
			inserted++
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}
	return inserted, len(events) - inserted, nil
}

func (s *Store) upsertSystemEventSourceSQLite(ctx context.Context, source systemevents.SourceStatus) error {
	capabilities, err := json.Marshal(source.Capabilities)
	if err != nil {
		capabilities = []byte("{}")
	}
	if source.LastIngestedAt.IsZero() {
		source.LastIngestedAt = time.Now().UTC()
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO system_event_sources (source, platform, status, last_ingested_at, last_error, capabilities_json)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(source) DO UPDATE SET
			platform = excluded.platform,
			status = excluded.status,
			last_ingested_at = excluded.last_ingested_at,
			last_error = excluded.last_error,
			capabilities_json = excluded.capabilities_json
	`, source.Source, source.Platform, source.Status, source.LastIngestedAt.UTC().Format(time.RFC3339Nano), source.LastError, capabilities)
	return err
}

func (s *Store) getJournalCursorSQLite(ctx context.Context, sourceKey string) (systemevents.CursorState, error) {
	var state systemevents.CursorState
	var updatedRaw any
	err := s.db.QueryRowContext(ctx, `
		SELECT cursor, boot_id, updated_at FROM journal_cursors WHERE source_key = ?
	`, sourceKey).Scan(&state.Cursor, &state.BootID, &updatedRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return systemevents.CursorState{}, nil
	}
	if err != nil {
		return systemevents.CursorState{}, fmt.Errorf("get journal cursor: %w", err)
	}
	if ts, perr := parseDBTime(updatedRaw); perr == nil {
		state.UpdatedAt = ts
	}
	return state, nil
}

func (s *Store) setJournalCursorSQLite(ctx context.Context, sourceKey string, state systemevents.CursorState) error {
	updatedAt := state.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO journal_cursors (source_key, cursor, boot_id, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(source_key) DO UPDATE SET
			cursor = excluded.cursor,
			boot_id = excluded.boot_id,
			updated_at = excluded.updated_at
	`, sourceKey, state.Cursor, state.BootID, updatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("set journal cursor: %w", err)
	}
	return nil
}

func (s *Store) isBootScannedSQLite(ctx context.Context, sourceKey, bootID string) (bool, error) {
	var one int
	err := s.db.QueryRowContext(ctx, `
		SELECT 1 FROM journal_scanned_boots WHERE source_key = ? AND boot_id = ?
	`, sourceKey, bootID).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("is boot scanned: %w", err)
	}
	return true, nil
}

func (s *Store) markBootScannedSQLite(ctx context.Context, sourceKey, bootID string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO journal_scanned_boots (source_key, boot_id, scanned_at)
		VALUES (?, ?, ?)
	`, sourceKey, bootID, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("mark boot scanned: %w", err)
	}
	return nil
}

func (s *Store) listSystemEventsSQLite(ctx context.Context, filters systemevents.Filters) (*systemevents.Response, error) {
	if filters.Limit <= 0 {
		filters.Limit = 100
	}
	if filters.Limit > 500 {
		filters.Limit = 500
	}

	where := []string{}
	args := []any{}
	if filters.Since != nil {
		where = append(where, "occurred_at >= ?")
		args = append(args, filters.Since.UTC().Format(time.RFC3339Nano))
	}
	if filters.Until != nil {
		where = append(where, "occurred_at <= ?")
		args = append(args, filters.Until.UTC().Format(time.RFC3339Nano))
	}
	addIn := func(column string, values []string) {
		if len(values) == 0 {
			return
		}
		placeholders := make([]string, 0, len(values))
		for _, value := range values {
			placeholders = append(placeholders, "?")
			args = append(args, value)
		}
		where = append(where, fmt.Sprintf("%s IN (%s)", column, strings.Join(placeholders, ",")))
	}
	addIn("category", filters.Category)
	addIn("source", filters.Source)
	addIn("platform", filters.Platform)
	if len(filters.Severity) > 0 {
		values := make([]string, 0, len(filters.Severity))
		for _, severity := range filters.Severity {
			values = append(values, string(severity))
		}
		addIn("severity", values)
	}
	if strings.TrimSpace(filters.BootID) != "" {
		where = append(where, "boot_id = ?")
		args = append(args, filters.BootID)
	}

	query := `
		SELECT id, fingerprint, occurred_at, ingested_at, source, platform, category, severity, title, summary, boot_id, details_json
		FROM system_events
	`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY occurred_at DESC LIMIT ?"
	args = append(args, filters.Limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []systemevents.Event
	for rows.Next() {
		var event systemevents.Event
		var occurredRaw, ingestedRaw any
		var detailsJSON []byte
		var bootID sql.NullString
		if err := rows.Scan(&event.ID, &event.Fingerprint, &occurredRaw, &ingestedRaw, &event.Source, &event.Platform, &event.Category, &event.Severity, &event.Title, &event.Summary, &bootID, &detailsJSON); err != nil {
			return nil, err
		}
		occurred, err := parseDBTime(occurredRaw)
		if err != nil {
			return nil, err
		}
		ingested, err := parseDBTime(ingestedRaw)
		if err != nil {
			return nil, err
		}
		event.OccurredAt = occurred.UTC()
		event.IngestedAt = ingested.UTC()
		if bootID.Valid {
			event.BootID = bootID.String
		}
		if len(detailsJSON) > 0 {
			_ = json.Unmarshal(detailsJSON, &event.Details)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sources, err := s.getSystemEventSourcesSQLite(ctx)
	if err != nil {
		return nil, err
	}
	response := &systemevents.Response{
		Events:  events,
		Count:   len(events),
		Sources: sources,
		Filters: systemevents.FiltersEcho{
			Limit:     filters.Limit,
			Category:  filters.Category,
			Severity:  filters.Severity,
			Source:    filters.Source,
			Platform:  filters.Platform,
			BootID:    filters.BootID,
			Correlate: filters.Correlate,
		},
	}
	if filters.Since != nil {
		response.Filters.Since = filters.Since.UTC().Format(time.RFC3339)
	}
	if filters.Until != nil {
		response.Filters.Until = filters.Until.UTC().Format(time.RFC3339)
	}
	if filters.Correlate {
		response.Correlations = systemevents.BuildCorrelations(events)
	}
	return response, nil
}

func (s *Store) getSystemEventSourcesSQLite(ctx context.Context) ([]systemevents.SourceStatus, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT source, platform, status, last_ingested_at, last_error, capabilities_json
		FROM system_event_sources
		ORDER BY source
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sources []systemevents.SourceStatus
	for rows.Next() {
		var source systemevents.SourceStatus
		var ingestedRaw any
		var capabilitiesJSON []byte
		if err := rows.Scan(&source.Source, &source.Platform, &source.Status, &ingestedRaw, &source.LastError, &capabilitiesJSON); err != nil {
			return nil, err
		}
		if ts, err := parseDBTime(ingestedRaw); err == nil {
			source.LastIngestedAt = ts.UTC()
		}
		if len(capabilitiesJSON) > 0 {
			_ = json.Unmarshal(capabilitiesJSON, &source.Capabilities)
		}
		sources = append(sources, source)
	}
	return sources, rows.Err()
}
