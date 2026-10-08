package calendar

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const eventColumns = `id,title,subject,notes,availability,timezone,all_day,start_date,end_date_exclusive,start_at,end_at,provider,provider_calendar_id,provider_event_id,occurrence_id,revision,created_at,updated_at`

type eventScanner interface{ Scan(...any) error }

func (r *sqliteRepository) ListEvents(ctx context.Context, start, end string) ([]Event, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+eventColumns+` FROM calendar_events WHERE range_start<=? AND range_end>? ORDER BY range_start,id`, end, start)
	if err != nil {
		return nil, fmt.Errorf("list calendar events: %w", err)
	}
	defer rows.Close()
	var result []Event
	for rows.Next() {
		e, scanErr := scanEvent(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

func (r *sqliteRepository) GetEvent(ctx context.Context, id string) (Event, error) {
	e, err := scanEvent(r.db.QueryRowContext(ctx, `SELECT `+eventColumns+` FROM calendar_events WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Event{}, ErrEventNotFound{ID: id}
	}
	if err != nil {
		return Event{}, fmt.Errorf("read calendar event: %w", err)
	}
	return e, nil
}

func (r *sqliteRepository) GetEventByIdempotencyKey(ctx context.Context, key string) (Event, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `SELECT event_id FROM calendar_event_commands WHERE idempotency_key=?`, key).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return Event{}, ErrEventNotFound{ID: "idempotency key " + key}
	}
	if err != nil {
		return Event{}, fmt.Errorf("lookup event idempotency key: %w", err)
	}
	return r.GetEvent(ctx, id)
}

func (r *sqliteRepository) CreateEvent(ctx context.Context, in CreateEventInput) (Event, error) {
	if err := validateEvent(in.Event); err != nil {
		return Event{}, err
	}
	beginner, ok := r.db.(interface {
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	})
	if !ok {
		return Event{}, fmt.Errorf("calendar storage does not support transactional event creation")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return Event{}, fmt.Errorf("begin create event: %w", err)
	}
	fail := func(cause error) (Event, error) { _ = tx.Rollback(); return Event{}, cause }
	fingerprint, err := eventFingerprint(in.Event)
	if err != nil {
		return fail(err)
	}
	var priorFingerprint, priorID string
	err = tx.QueryRowContext(ctx, `SELECT fingerprint,event_id FROM calendar_event_commands WHERE idempotency_key=?`, in.IdempotencyKey).Scan(&priorFingerprint, &priorID)
	if err == nil {
		if priorFingerprint != fingerprint {
			return fail(ErrEventIdempotencyConflict{Key: in.IdempotencyKey})
		}
		prior, loadErr := scanEvent(tx.QueryRowContext(ctx, `SELECT `+eventColumns+` FROM calendar_events WHERE id=?`, priorID))
		if loadErr != nil {
			return fail(fmt.Errorf("read idempotent calendar event: %w", loadErr))
		}
		if commitErr := tx.Commit(); commitErr != nil {
			return Event{}, fmt.Errorf("commit idempotent create event: %w", commitErr)
		}
		return prior, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fail(fmt.Errorf("read event idempotency key: %w", err))
	}
	e := in.Event
	if e.ProviderEventID != "" {
		var existingID string
		identityErr := tx.QueryRowContext(ctx, `SELECT id FROM calendar_events WHERE provider=? AND provider_calendar_id=? AND provider_event_id=? AND occurrence_id=?`, e.Provider, e.ProviderCalendarID, e.ProviderEventID, e.OccurrenceID).Scan(&existingID)
		if identityErr == nil {
			existing, readErr := scanEvent(tx.QueryRowContext(ctx, `SELECT `+eventColumns+` FROM calendar_events WHERE id=?`, existingID))
			if readErr != nil {
				return fail(fmt.Errorf("read existing provider event identity: %w", readErr))
			}
			existingFingerprint, _ := eventFingerprint(existing)
			if existingFingerprint != fingerprint {
				return fail(ErrEventIdentityConflict{Provider: e.Provider, CalendarID: e.ProviderCalendarID, EventID: e.ProviderEventID, OccurrenceID: e.OccurrenceID})
			}
			now := r.clock.Now().UTC().Format(time.RFC3339Nano)
			if _, err := tx.ExecContext(ctx, `INSERT INTO calendar_event_commands (idempotency_key,fingerprint,event_id,created_at) VALUES (?,?,?,?)`, in.IdempotencyKey, fingerprint, existingID, now); err != nil {
				return fail(fmt.Errorf("record provider identity retry: %w", err))
			}
			if err := tx.Commit(); err != nil {
				return Event{}, fmt.Errorf("commit provider identity retry: %w", err)
			}
			return existing, nil
		}
		if !errors.Is(identityErr, sql.ErrNoRows) {
			return fail(fmt.Errorf("lookup provider event identity: %w", identityErr))
		}
	}
	if e.ID == "" {
		e.ID = r.id()
	}
	now := r.clock.Now().UTC()
	e.Revision, e.CreatedAt, e.UpdatedAt = 1, now, now
	rangeStart, rangeEnd, err := eventRange(e)
	if err != nil {
		return fail(err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO calendar_events (`+eventColumns+`,range_start,range_end) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		e.ID, e.Title, e.Subject, e.Notes, e.Availability, e.Timezone, e.AllDay, e.StartDate, e.EndDateExclusive, e.StartAt, e.EndAt,
		e.Provider, e.ProviderCalendarID, e.ProviderEventID, e.OccurrenceID, e.Revision, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), rangeStart, rangeEnd); err != nil {
		return fail(fmt.Errorf("insert calendar event: %w", err))
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO calendar_event_commands (idempotency_key,fingerprint,event_id,created_at) VALUES (?,?,?,?)`, in.IdempotencyKey, fingerprint, e.ID, now.Format(time.RFC3339Nano)); err != nil {
		return fail(fmt.Errorf("record event idempotency key: %w", err))
	}
	if err := tx.Commit(); err != nil {
		return Event{}, fmt.Errorf("commit create event: %w", err)
	}
	return e, nil
}

func (r *sqliteRepository) UpdateEvent(ctx context.Context, in UpdateEventInput) (Event, error) {
	if err := validateEvent(in.Event); err != nil {
		return Event{}, err
	}
	beginner, ok := r.db.(interface {
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	})
	if !ok {
		return Event{}, fmt.Errorf("calendar storage does not support transactional event updates")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return Event{}, fmt.Errorf("begin update event: %w", err)
	}
	fail := func(cause error) (Event, error) { _ = tx.Rollback(); return Event{}, cause }
	current, err := scanEvent(tx.QueryRowContext(ctx, `SELECT `+eventColumns+` FROM calendar_events WHERE id=?`, in.ID))
	if errors.Is(err, sql.ErrNoRows) {
		return fail(ErrEventNotFound{ID: in.ID})
	}
	if err != nil {
		return fail(fmt.Errorf("read event before update: %w", err))
	}
	if current.Revision != in.ExpectedRevision {
		return fail(ErrEventRevisionConflict{Expected: in.ExpectedRevision, Current: current.Revision})
	}
	e := in.Event
	e.ID, e.Revision, e.CreatedAt = current.ID, current.Revision+1, current.CreatedAt
	// Provider identity is immutable: edits change event content, not which
	// upstream occurrence this durable local row represents.
	e.Provider, e.ProviderCalendarID, e.ProviderEventID, e.OccurrenceID = current.Provider, current.ProviderCalendarID, current.ProviderEventID, current.OccurrenceID
	e.UpdatedAt = r.clock.Now().UTC()
	rangeStart, rangeEnd, err := eventRange(e)
	if err != nil {
		return fail(err)
	}
	result, err := tx.ExecContext(ctx, `UPDATE calendar_events SET title=?,subject=?,notes=?,availability=?,timezone=?,all_day=?,start_date=?,end_date_exclusive=?,start_at=?,end_at=?,range_start=?,range_end=?,revision=?,updated_at=? WHERE id=? AND revision=?`,
		e.Title, e.Subject, e.Notes, e.Availability, e.Timezone, e.AllDay, e.StartDate, e.EndDateExclusive, e.StartAt, e.EndAt, rangeStart, rangeEnd, e.Revision, e.UpdatedAt.Format(time.RFC3339Nano), e.ID, in.ExpectedRevision)
	if err != nil {
		return fail(fmt.Errorf("update calendar event: %w", err))
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fail(fmt.Errorf("inspect calendar event update: %w", err))
	}
	if changed != 1 {
		latest, readErr := scanEvent(tx.QueryRowContext(ctx, `SELECT `+eventColumns+` FROM calendar_events WHERE id=?`, in.ID))
		if errors.Is(readErr, sql.ErrNoRows) {
			return fail(ErrEventNotFound{ID: in.ID})
		}
		if readErr != nil {
			return fail(readErr)
		}
		return fail(ErrEventRevisionConflict{Expected: in.ExpectedRevision, Current: latest.Revision})
	}
	if err := tx.Commit(); err != nil {
		return Event{}, fmt.Errorf("commit update event: %w", err)
	}
	return e, nil
}

func eventFingerprint(e Event) (string, error) {
	e.ID, e.Revision = "", 0
	e.CreatedAt, e.UpdatedAt = time.Time{}, time.Time{}
	encoded, err := json.Marshal(e)
	if err != nil {
		return "", fmt.Errorf("encode event fingerprint: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func eventRange(e Event) (string, string, error) {
	if e.AllDay {
		return e.StartDate, e.EndDateExclusive, nil
	}
	location, err := time.LoadLocation(e.Timezone)
	if err != nil {
		return "", "", fmt.Errorf("load event timezone: %w", err)
	}
	start, _ := time.Parse(time.RFC3339Nano, e.StartAt)
	end, _ := time.Parse(time.RFC3339Nano, e.EndAt)
	localStart := start.In(location).Format("2006-01-02")
	lastIncludedDay := end.Add(-time.Nanosecond).In(location).Format("2006-01-02")
	exclusiveEnd, err := time.Parse("2006-01-02", lastIncludedDay)
	if err != nil {
		return "", "", err
	}
	return localStart, exclusiveEnd.AddDate(0, 0, 1).Format("2006-01-02"), nil
}

func scanEvent(row eventScanner) (Event, error) {
	var e Event
	var allDay int
	var created, updated string
	err := row.Scan(&e.ID, &e.Title, &e.Subject, &e.Notes, &e.Availability, &e.Timezone, &allDay,
		&e.StartDate, &e.EndDateExclusive, &e.StartAt, &e.EndAt, &e.Provider, &e.ProviderCalendarID,
		&e.ProviderEventID, &e.OccurrenceID, &e.Revision, &created, &updated)
	if err != nil {
		return Event{}, err
	}
	e.AllDay = allDay != 0
	e.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return Event{}, fmt.Errorf("parse event creation time: %w", err)
	}
	e.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
	if err != nil {
		return Event{}, fmt.Errorf("parse event update time: %w", err)
	}
	return e, nil
}
