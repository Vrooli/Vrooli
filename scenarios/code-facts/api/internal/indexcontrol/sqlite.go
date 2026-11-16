package indexcontrol

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrJobNotFound = errors.New("index job not found")

type SQLiteJobStore struct{ db *sql.DB }

func NewSQLiteJobStore(db *sql.DB) *SQLiteJobStore { return &SQLiteJobStore{db: db} }

func (store *SQLiteJobStore) Create(ctx context.Context, job Job) error {
	if err := store.ready(); err != nil {
		return err
	}
	if job.ID == "" || job.Generation == "" || job.Kind == "" || (job.State != JobQueued && job.State != JobRunning) {
		return fmt.Errorf("index job requires id, generation, kind, and an initial queued or running state")
	}
	created := job.CreatedAt.UTC()
	if created.IsZero() {
		return fmt.Errorf("index job requires created timestamp")
	}
	updated := job.UpdatedAt.UTC()
	if updated.IsZero() {
		updated = created
	}
	_, err := store.db.ExecContext(ctx, `INSERT INTO code_facts_index_jobs(
 id,generation_id,kind,state,cursor,processed,total,created_at_unix,updated_at_unix,error
) VALUES(?,?,?,?,?,?,?,?,?,?)`, job.ID, job.Generation, job.Kind, job.State, job.Cursor, job.Progress, job.Total,
		created.Unix(), updated.Unix(), job.Error)
	if err != nil {
		return fmt.Errorf("create index job: %w", err)
	}
	return nil
}

func (store *SQLiteJobStore) Update(ctx context.Context, job Job) error {
	if err := store.ready(); err != nil {
		return err
	}
	current, err := store.Get(ctx, job.ID)
	if err != nil {
		return err
	}
	if !current.State.CanTransitionTo(job.State) {
		return fmt.Errorf("invalid index job transition %q -> %q", current.State, job.State)
	}
	result, err := store.db.ExecContext(ctx, `UPDATE code_facts_index_jobs SET
 state=?,cursor=?,processed=?,total=?,updated_at_unix=?,error=? WHERE id=? AND state=?`,
		job.State, job.Cursor, job.Progress, job.Total, job.UpdatedAt.UTC().Unix(), job.Error, job.ID, current.State)
	if err != nil {
		return fmt.Errorf("update index job: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return fmt.Errorf("index job %q changed concurrently", job.ID)
	}
	return nil
}

func (store *SQLiteJobStore) Get(ctx context.Context, id string) (Job, error) {
	if err := store.ready(); err != nil {
		return Job{}, err
	}
	row := store.db.QueryRowContext(ctx, jobSelect+` WHERE id=?`, id)
	job, err := scanJob(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Job{}, ErrJobNotFound
	}
	return job, err
}

func (store *SQLiteJobStore) ListActive(ctx context.Context) ([]Job, error) {
	if err := store.ready(); err != nil {
		return nil, err
	}
	rows, err := store.db.QueryContext(ctx, jobSelect+` WHERE state IN ('queued','running','cancellation_requested','interrupted') ORDER BY created_at_unix,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var jobs []Job
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (store *SQLiteJobStore) RequestCancel(ctx context.Context, id string, now time.Time) error {
	if err := store.ready(); err != nil {
		return err
	}
	current, err := store.Get(ctx, id)
	if err != nil {
		return err
	}
	if !current.State.CanTransitionTo(JobCancellationRequested) {
		return fmt.Errorf("invalid index job transition %q -> %q", current.State, JobCancellationRequested)
	}
	result, err := store.db.ExecContext(ctx, `UPDATE code_facts_index_jobs SET state='cancellation_requested',updated_at_unix=? WHERE id=? AND state=?`, now.UTC().Unix(), id, current.State)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return fmt.Errorf("index job %q changed concurrently", id)
	}
	return nil
}

func (store *SQLiteJobStore) RecoverInterrupted(ctx context.Context, now time.Time) ([]Job, error) {
	if err := store.ready(); err != nil {
		return nil, err
	}
	rows, err := store.db.QueryContext(ctx, `SELECT id FROM code_facts_index_jobs WHERE state IN ('queued','running','cancellation_requested') ORDER BY created_at_unix,id`)
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for _, id := range ids {
		job, err := store.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		job.State, job.UpdatedAt, job.Error = JobInterrupted, now.UTC(), "process restarted before completion"
		if err := store.Update(ctx, job); err != nil {
			return nil, err
		}
	}
	return store.ListActive(ctx)
}

func (store *SQLiteJobStore) ready() error {
	if store == nil || store.db == nil {
		return fmt.Errorf("index job store requires database")
	}
	return nil
}

const jobSelect = `SELECT id,kind,state,generation_id,processed,total,error,cursor,created_at_unix,updated_at_unix FROM code_facts_index_jobs`

type rowScanner interface{ Scan(...any) error }

func scanJob(row rowScanner) (Job, error) {
	var job Job
	var created, updated int64
	err := row.Scan(&job.ID, &job.Kind, &job.State, &job.Generation, &job.Progress, &job.Total, &job.Error, &job.Cursor, &created, &updated)
	if err != nil {
		return Job{}, err
	}
	if !job.State.Valid() {
		return Job{}, fmt.Errorf("stored index job has invalid state %q", job.State)
	}
	job.CreatedAt, job.UpdatedAt = time.Unix(created, 0).UTC(), time.Unix(updated, 0).UTC()
	return job, nil
}

type SQLitePromotionStore struct{ db *sql.DB }

func NewSQLitePromotionStore(db *sql.DB) *SQLitePromotionStore { return &SQLitePromotionStore{db: db} }

func (store *SQLitePromotionStore) Prepare(ctx context.Context, id, from, to string, now time.Time) error {
	if store == nil || store.db == nil || id == "" || from == "" || to == "" || from == to {
		return fmt.Errorf("promotion requires store, id, and distinct generations")
	}
	_, err := store.db.ExecContext(ctx, `INSERT INTO code_facts_promotions(id,from_generation,to_generation,state,created_at_unix,updated_at_unix) VALUES(?,?,?,'prepared',?,?)`, id, from, to, now.UTC().Unix(), now.UTC().Unix())
	return err
}

func (store *SQLitePromotionStore) Transition(ctx context.Context, id, state, failure string, now time.Time) error {
	result, err := store.db.ExecContext(ctx, `UPDATE code_facts_promotions SET state=?,error=?,updated_at_unix=? WHERE id=?`, state, failure, now.UTC().Unix(), id)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return fmt.Errorf("promotion %q not found", id)
	}
	return nil
}

var (
	_ JobStore       = (*SQLiteJobStore)(nil)
	_ PromotionStore = (*SQLitePromotionStore)(nil)
)

type SQLiteStatusReader struct {
	db   *sql.DB
	jobs JobStore
}

func NewSQLiteStatusReader(db *sql.DB, jobs JobStore) *SQLiteStatusReader {
	return &SQLiteStatusReader{db: db, jobs: jobs}
}

func (reader *SQLiteStatusReader) Status(ctx context.Context) (Status, error) {
	if reader == nil || reader.db == nil || reader.jobs == nil {
		return Status{}, fmt.Errorf("index status reader requires database and job store")
	}
	var status Status
	var updated int64
	err := reader.db.QueryRowContext(ctx, "SELECT id,source_digest,descriptor_digest,updated_at_unix FROM code_facts_generations WHERE state='active'").Scan(&status.ActiveGeneration, &status.SourceDigest, &status.DescriptorDigest, &updated)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Status{}, err
	}
	if errors.Is(err, sql.ErrNoRows) {
		status.State = "uninitialized"
		status.Degraded = append(status.Degraded, "catalog")
		var pageCount, pageSize int64
		if err := reader.db.QueryRowContext(ctx, "PRAGMA page_count").Scan(&pageCount); err == nil {
			_ = reader.db.QueryRowContext(ctx, "PRAGMA page_size").Scan(&pageSize)
			status.StorageBytes = pageCount * pageSize
		}
		status.ActiveJobs, err = reader.jobs.ListActive(ctx)
		return status, nil
	}
	status.State = "ready"
	_ = reader.db.QueryRowContext(ctx, "SELECT id FROM code_facts_generations WHERE state='retired' ORDER BY updated_at_unix DESC LIMIT 1").Scan(&status.PreviousGeneration)
	if err := reader.db.QueryRowContext(ctx, `SELECT source_files,search_documents,semantic_cards,graph_facts FROM code_facts_generation_stats WHERE generation_id=?`, status.ActiveGeneration).Scan(
		&status.SourceFiles, &status.SearchDocuments, &status.SemanticCards, &status.GraphFacts,
	); err != nil {
		return Status{}, err
	}
	var pageCount, pageSize int64
	if err := reader.db.QueryRowContext(ctx, "PRAGMA page_count").Scan(&pageCount); err == nil {
		_ = reader.db.QueryRowContext(ctx, "PRAGMA page_size").Scan(&pageSize)
		status.StorageBytes = pageCount * pageSize
	}
	status.ActiveJobs, err = reader.jobs.ListActive(ctx)
	if err != nil {
		return Status{}, err
	}
	var outcome string
	var lastUpdated int64
	if err := reader.db.QueryRowContext(ctx, "SELECT state,updated_at_unix FROM code_facts_index_jobs ORDER BY updated_at_unix DESC,id DESC LIMIT 1").Scan(&outcome, &lastUpdated); err == nil {
		status.LastReconcileOutcome = outcome
		status.LastReconcileAt = time.Unix(lastUpdated, 0).UTC()
	}
	if len(status.ActiveJobs) > 0 {
		status.State = "updating"
	}
	return status, nil
}

var _ StatusReader = (*SQLiteStatusReader)(nil)
