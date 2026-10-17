package focus

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type RepositoryOptions struct{ IDGenerator func() string }

type sqliteRepository struct {
	db          SQLExecutor
	clock       schedule.Clock
	idGenerator func() string
}

func NewSQLiteRepository(db SQLExecutor, clock schedule.Clock) Repository {
	return NewSQLiteRepositoryWithOptions(db, clock, RepositoryOptions{})
}

func NewSQLiteRepositoryWithOptions(db SQLExecutor, clock schedule.Clock, options RepositoryOptions) Repository {
	if options.IDGenerator == nil {
		options.IDGenerator = uuid.NewString
	}
	return &sqliteRepository{db: db, clock: clock, idGenerator: options.IDGenerator}
}

func unix(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

func (r *sqliteRepository) Create(ctx context.Context, session Session) (Session, error) {
	if session.ID == "" {
		session.ID = r.idGenerator()
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO focus_sessions (id,work_item_id,title,mode,state,started_at,active_started_at,ended_at,active_seconds,wall_seconds,revision) VALUES (?,?,?,?,?,?,?,?,?,?,?)`, session.ID, session.WorkItemID, session.Title, session.Mode, session.State, unix(session.StartedAt), unixPtr(session.ActiveStartedAt), unix(session.EndedAt), session.ActiveSeconds, session.WallSeconds, session.Revision)
	if err != nil {
		return Session{}, fmt.Errorf("insert focus session %q: %w", session.ID, err)
	}
	return session, nil
}

func unixPtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Unix()
}

func (r *sqliteRepository) Current(ctx context.Context) (Session, bool, error) {
	return r.scanOne(r.db.QueryRowContext(ctx, `SELECT id,work_item_id,title,mode,state,started_at,active_started_at,ended_at,active_seconds,wall_seconds,revision FROM focus_sessions WHERE state IN ('running','paused') ORDER BY started_at DESC LIMIT 1`))
}

func (r *sqliteRepository) Get(ctx context.Context, id string) (Session, error) {
	s, ok, err := r.scanOne(r.db.QueryRowContext(ctx, `SELECT id,work_item_id,title,mode,state,started_at,active_started_at,ended_at,active_seconds,wall_seconds,revision FROM focus_sessions WHERE id = ?`, id))
	if err != nil {
		return Session{}, err
	}
	if !ok {
		return Session{}, ErrSessionNotFound{id}
	}
	return s, nil
}

func (r *sqliteRepository) Transition(ctx context.Context, session Session, expectedRevision int64) (Session, error) {
	ended := unix(session.EndedAt)
	result, err := r.db.ExecContext(ctx, `UPDATE focus_sessions SET state=?,active_started_at=?,ended_at=?,active_seconds=?,wall_seconds=?,revision=revision+1 WHERE id=? AND revision=?`, session.State, unixPtr(session.ActiveStartedAt), ended, session.ActiveSeconds, session.WallSeconds, session.ID, expectedRevision)
	if err != nil {
		if stringsContainsConstraint(err.Error()) {
			return Session{}, ErrSessionConflict{"another current session already exists"}
		}
		return Session{}, fmt.Errorf("transition focus session %q: %w", session.ID, err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return Session{}, err
	}
	if count != 1 {
		return Session{}, ErrRevisionConflict{session.ID}
	}
	session.Revision = expectedRevision + 1
	return session, nil
}

func (r *sqliteRepository) SaveNote(ctx context.Context, note SessionNote) (SessionNote, error) {
	_, err := r.db.ExecContext(ctx, `INSERT INTO focus_session_notes(session_id,local_date,note,updated_at) VALUES (?,?,?,?) ON CONFLICT(session_id) DO UPDATE SET local_date=excluded.local_date,note=excluded.note,updated_at=excluded.updated_at`, note.SessionID, note.LocalDate, note.Note, unix(note.UpdatedAt))
	if err != nil {
		return SessionNote{}, fmt.Errorf("save focus session note: %w", err)
	}
	return note, nil
}

func (r *sqliteRepository) ListNotes(ctx context.Context, localDate string) ([]SessionNote, error) {
	query := `SELECT session_id,local_date,note,updated_at FROM focus_session_notes`
	args := []any{}
	if localDate != "" {
		query += ` WHERE local_date=?`
		args = append(args, localDate)
	}
	query += ` ORDER BY updated_at DESC,session_id`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list focus session notes: %w", err)
	}
	defer rows.Close()
	out := []SessionNote{}
	for rows.Next() {
		var note SessionNote
		var updated int64
		if err := rows.Scan(&note.SessionID, &note.LocalDate, &note.Note, &updated); err != nil {
			return nil, err
		}
		note.UpdatedAt = time.Unix(updated, 0).UTC()
		out = append(out, note)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) RecordPauseEvent(ctx context.Context, event PauseEvent) (PauseEvent, error) {
	if event.ID == "" {
		event.ID = r.idGenerator()
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO focus_pause_events(id,session_id,local_date,reason,recorded_at) VALUES (?,?,?,?,?)`, event.ID, event.SessionID, event.LocalDate, event.Reason, unix(event.RecordedAt))
	if err != nil {
		return PauseEvent{}, fmt.Errorf("record focus pause event: %w", err)
	}
	return event, nil
}

func (r *sqliteRepository) ListPauseEvents(ctx context.Context, localDate string) ([]PauseEvent, error) {
	query := `SELECT id,session_id,local_date,reason,recorded_at FROM focus_pause_events`
	args := []any{}
	if localDate != "" {
		query += ` WHERE local_date=?`
		args = append(args, localDate)
	}
	query += ` ORDER BY recorded_at DESC,id`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list focus pause events: %w", err)
	}
	defer rows.Close()
	out := []PauseEvent{}
	for rows.Next() {
		var event PauseEvent
		var recorded int64
		if err := rows.Scan(&event.ID, &event.SessionID, &event.LocalDate, &event.Reason, &recorded); err != nil {
			return nil, err
		}
		event.RecordedAt = time.Unix(recorded, 0).UTC()
		out = append(out, event)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) CreateActual(ctx context.Context, actual Actual) (Actual, error) {
	if actual.ID == "" {
		actual.ID = r.idGenerator()
	}
	if actual.AllocationID == "" {
		if err := r.db.QueryRowContext(ctx, `SELECT COALESCE((SELECT id FROM calendar_allocations WHERE work_item_id=? AND local_date=? AND state='accepted' ORDER BY start_minutes LIMIT 1),'')`, actual.WorkItemID, actual.LocalDate).Scan(&actual.AllocationID); err != nil {
			return Actual{}, fmt.Errorf("resolve plan allocation: %w", err)
		}
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO manual_actuals (id,work_item_id,title,local_date,reported_minutes,certainty,note,allocation_id,created_at,revision) VALUES (?,?,?,?,?,?,?,COALESCE(NULLIF(?,''),(SELECT id FROM calendar_allocations WHERE work_item_id=? AND local_date=? AND state='accepted' ORDER BY start_minutes LIMIT 1),''),?,?)`, actual.ID, actual.WorkItemID, actual.Title, actual.LocalDate, actual.ReportedMinutes, actual.Certainty, actual.Note, actual.AllocationID, actual.WorkItemID, actual.LocalDate, unix(actual.CreatedAt), actual.Revision)
	if err != nil {
		return Actual{}, fmt.Errorf("insert manual actual %q: %w", actual.ID, err)
	}
	return actual, nil
}

func (r *sqliteRepository) ListActuals(ctx context.Context, localDate string) ([]Actual, error) {
	query := `SELECT id,work_item_id,title,local_date,reported_minutes,certainty,note,allocation_id,created_at,revision FROM manual_actuals`
	args := []any{}
	if localDate != "" {
		query += ` WHERE local_date = ?`
		args = append(args, localDate)
	}
	query += ` ORDER BY local_date DESC, created_at DESC, id DESC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list manual actuals: %w", err)
	}
	defer rows.Close()
	var actuals []Actual
	for rows.Next() {
		actual, err := scanActual(rows)
		if err != nil {
			return nil, err
		}
		actuals = append(actuals, actual)
	}
	return actuals, rows.Err()
}

func (r *sqliteRepository) ListCorrections(ctx context.Context, actualID, localDate string, limit int) ([]Correction, error) {
	query := `SELECT c.id,c.actual_id,c.previous_minutes,c.new_minutes,c.previous_certainty,c.new_certainty,c.reason,c.created_at FROM actual_corrections c JOIN manual_actuals a ON a.id=c.actual_id`
	args := []any{}
	conditions := []string{}
	if actualID != "" {
		conditions = append(conditions, "c.actual_id = ?")
		args = append(args, actualID)
	}
	if localDate != "" {
		conditions = append(conditions, "a.local_date = ?")
		args = append(args, localDate)
	}
	if len(conditions) > 0 {
		query += " WHERE " + stringsJoin(conditions, " AND ")
	}
	query += ` ORDER BY c.created_at DESC,c.id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list actual corrections: %w", err)
	}
	defer rows.Close()
	corrections := []Correction{}
	for rows.Next() {
		var correction Correction
		var created int64
		if err := rows.Scan(&correction.ID, &correction.ActualID, &correction.PreviousMinutes, &correction.NewMinutes, &correction.PreviousCertainty, &correction.NewCertainty, &correction.Reason, &created); err != nil {
			return nil, err
		}
		correction.CreatedAt = time.Unix(created, 0).UTC()
		corrections = append(corrections, correction)
	}
	return corrections, rows.Err()
}

func (r *sqliteRepository) CorrectActual(ctx context.Context, update Actual, expectedRevision int64) (Actual, error) {
	current, err := r.getActual(ctx, update.ID)
	if err != nil {
		return Actual{}, err
	}
	if current.Revision != expectedRevision {
		return Actual{}, ErrRevisionConflict{update.ID}
	}
	result, err := r.db.ExecContext(ctx, `UPDATE manual_actuals SET reported_minutes=?,certainty=?,note=?,revision=revision+1 WHERE id=? AND revision=?`, update.ReportedMinutes, update.Certainty, update.Note, update.ID, expectedRevision)
	if err != nil {
		return Actual{}, fmt.Errorf("correct manual actual %q: %w", update.ID, err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return Actual{}, err
	}
	if count != 1 {
		return Actual{}, ErrRevisionConflict{update.ID}
	}
	if _, err := r.db.ExecContext(ctx, `INSERT INTO actual_corrections (id,actual_id,previous_minutes,new_minutes,previous_certainty,new_certainty,reason,created_at) VALUES (?,?,?,?,?,?,?,?)`, r.idGenerator(), update.ID, current.ReportedMinutes, update.ReportedMinutes, current.Certainty, update.Certainty, update.Note, unix(r.clock.Now().UTC())); err != nil {
		return Actual{}, fmt.Errorf("record actual correction %q: %w", update.ID, err)
	}
	current.ReportedMinutes = update.ReportedMinutes
	current.Certainty = update.Certainty
	current.Note = update.Note
	current.Revision = expectedRevision + 1
	return current, nil
}

func (r *sqliteRepository) getActual(ctx context.Context, id string) (Actual, error) {
	actual, err := scanActual(r.db.QueryRowContext(ctx, `SELECT id,work_item_id,title,local_date,reported_minutes,certainty,note,allocation_id,created_at,revision FROM manual_actuals WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Actual{}, ErrActualNotFound{id}
	}
	return actual, err
}

type actualScanner interface{ Scan(...any) error }

func scanActual(scanner actualScanner) (Actual, error) {
	var actual Actual
	var created int64
	if err := scanner.Scan(&actual.ID, &actual.WorkItemID, &actual.Title, &actual.LocalDate, &actual.ReportedMinutes, &actual.Certainty, &actual.Note, &actual.AllocationID, &created, &actual.Revision); err != nil {
		return Actual{}, err
	}
	actual.CreatedAt = time.Unix(created, 0).UTC()
	return actual, nil
}

func (r *sqliteRepository) scanOne(row *sql.Row) (Session, bool, error) {
	var s Session
	var started, activeStarted, ended int64
	var activeStartedNull sql.NullInt64
	err := row.Scan(&s.ID, &s.WorkItemID, &s.Title, &s.Mode, &s.State, &started, &activeStartedNull, &ended, &s.ActiveSeconds, &s.WallSeconds, &s.Revision)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, false, nil
	}
	if err != nil {
		return Session{}, false, err
	}
	s.StartedAt = time.Unix(started, 0).UTC()
	if activeStartedNull.Valid {
		activeStarted = activeStartedNull.Int64
		t := time.Unix(activeStarted, 0).UTC()
		s.ActiveStartedAt = &t
	}
	if ended != 0 {
		s.EndedAt = time.Unix(ended, 0).UTC()
	}
	return s, true, nil
}

func stringsContainsConstraint(message string) bool {
	return len(message) > 0 && (contains(message, "UNIQUE constraint") || contains(message, "constraint failed"))
}

func contains(value, fragment string) bool {
	for i := 0; i+len(fragment) <= len(value); i++ {
		if value[i:i+len(fragment)] == fragment {
			return true
		}
	}
	return false
}

func stringsJoin(values []string, separator string) string {
	result := ""
	for i, value := range values {
		if i > 0 {
			result += separator
		}
		result += value
	}
	return result
}
