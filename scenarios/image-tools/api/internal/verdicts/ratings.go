package verdicts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// SQLExecutor is the narrow DB surface the stores need. Both *sql.DB (tests) and
// *database.RoutedDB (production) satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Rating is the operator's judgement of one job output.
type Rating struct {
	JobID     string
	Score     int
	Note      string
	RatedBy   string
	RatedAt   time.Time
	ModelID   string
	Operation string
}

// MaxNoteLength bounds a rating note; it is a remark, not a report.
const MaxNoteLength = 1000

// Rating errors callers translate to a status.
var (
	ErrJobUnknown     = errors.New("job has no recorded result")
	ErrJobNotRateable = errors.New("only a succeeded job has an output to rate")
	ErrBadScore       = errors.New("score must be a whole number from 1 to 5")
	ErrNoteTooLong    = fmt.Errorf("note is longer than %d characters", MaxNoteLength)
	ErrNotRated       = errors.New("job has no rating")
)

// Ratings persists operator ratings.
type Ratings struct {
	db  SQLExecutor
	now func() time.Time
}

// NewRatings constructs the rating store over db.
func NewRatings(db SQLExecutor) *Ratings { return &Ratings{db: db, now: time.Now} }

// Rate records (or replaces) the rating for a job. The job must have a durable
// trace in the succeeded state; its model and operation are copied from it. A
// job that ran no model (a deterministic operation) can still be rated: the
// rating is kept, and simply feeds no model verdict.
func (r *Ratings) Rate(ctx context.Context, jobID string, score int, note, ratedBy string) (Rating, error) {
	jobID = strings.TrimSpace(jobID)
	if score < 1 || score > 5 {
		return Rating{}, ErrBadScore
	}
	note = strings.TrimSpace(note)
	if len([]rune(note)) > MaxNoteLength {
		return Rating{}, ErrNoteTooLong
	}
	var state, model, op string
	err := r.db.QueryRowContext(ctx, `SELECT state, model_id, operation FROM job_trace WHERE job_id = ?`, jobID).Scan(&state, &model, &op)
	if errors.Is(err, sql.ErrNoRows) {
		return Rating{}, ErrJobUnknown
	}
	if err != nil {
		return Rating{}, fmt.Errorf("verdicts: read job trace: %w", err)
	}
	if state != "succeeded" {
		return Rating{}, ErrJobNotRateable
	}
	rating := Rating{
		JobID: jobID, Score: score, Note: note, RatedBy: strings.TrimSpace(ratedBy),
		RatedAt: r.now().UTC(), ModelID: model, Operation: op,
	}
	if _, err := r.db.ExecContext(ctx, `
INSERT INTO job_rating (job_id, score, note, rated_by, rated_at, model_id, operation) VALUES (?,?,?,?,?,?,?)
ON CONFLICT(job_id) DO UPDATE SET score=excluded.score, note=excluded.note, rated_by=excluded.rated_by,
  rated_at=excluded.rated_at, model_id=excluded.model_id, operation=excluded.operation`,
		rating.JobID, rating.Score, rating.Note, rating.RatedBy, rating.RatedAt.Format(time.RFC3339), rating.ModelID, rating.Operation); err != nil {
		return Rating{}, fmt.Errorf("verdicts: save rating: %w", err)
	}
	return rating, nil
}

// Clear removes a job's rating.
func (r *Ratings) Clear(ctx context.Context, jobID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM job_rating WHERE job_id = ?`, strings.TrimSpace(jobID))
	if err != nil {
		return fmt.Errorf("verdicts: clear rating: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotRated
	}
	return nil
}

const ratingColumns = `job_id, score, note, rated_by, rated_at, model_id, operation`

func scanRating(scan func(...any) error) (Rating, error) {
	var out Rating
	var at string
	if err := scan(&out.JobID, &out.Score, &out.Note, &out.RatedBy, &at, &out.ModelID, &out.Operation); err != nil {
		return Rating{}, err
	}
	out.RatedAt, _ = time.Parse(time.RFC3339, at)
	return out, nil
}

// Get returns one job's rating.
func (r *Ratings) Get(ctx context.Context, jobID string) (Rating, bool, error) {
	out, err := scanRating(r.db.QueryRowContext(ctx, `SELECT `+ratingColumns+` FROM job_rating WHERE job_id = ?`, jobID).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return Rating{}, false, nil
	}
	return out, err == nil, err
}

// ForJobs returns the ratings of the given jobs keyed by job id (one query per
// call, however many ids: the job list rides on it).
func (r *Ratings) ForJobs(ctx context.Context, jobIDs []string) (map[string]Rating, error) {
	out := make(map[string]Rating, len(jobIDs))
	const batch = 400
	for start := 0; start < len(jobIDs); start += batch {
		end := min(start+batch, len(jobIDs))
		args := make([]any, 0, end-start)
		marks := make([]string, 0, end-start)
		for _, id := range jobIDs[start:end] {
			args = append(args, id)
			marks = append(marks, "?")
		}
		rows, err := r.db.QueryContext(ctx, `SELECT `+ratingColumns+` FROM job_rating WHERE job_id IN (`+strings.Join(marks, ",")+`)`, args...)
		if err != nil {
			return nil, fmt.Errorf("verdicts: read ratings: %w", err)
		}
		for rows.Next() {
			rating, err := scanRating(rows.Scan)
			if err != nil {
				rows.Close()
				return nil, fmt.Errorf("verdicts: scan rating: %w", err)
			}
			out[rating.JobID] = rating
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}
	return out, nil
}

// List returns ratings newest first, optionally narrowed to a model and/or
// operation. limit <= 0 means 50.
func (r *Ratings) List(ctx context.Context, modelID, operation string, limit int) ([]Rating, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `SELECT ` + ratingColumns + ` FROM job_rating`
	var where []string
	var args []any
	if modelID != "" {
		where = append(where, "model_id = ?")
		args = append(args, modelID)
	}
	if operation != "" {
		where = append(where, "operation = ?")
		args = append(args, operation)
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += ` ORDER BY rated_at DESC, job_id LIMIT ?`
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("verdicts: list ratings: %w", err)
	}
	defer rows.Close()
	var out []Rating
	for rows.Next() {
		rating, err := scanRating(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("verdicts: scan rating: %w", err)
		}
		out = append(out, rating)
	}
	return out, rows.Err()
}
