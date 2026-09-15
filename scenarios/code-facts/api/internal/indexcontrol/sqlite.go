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
	if job.ID == "" || job.Generation == "" || job.Kind == "" || job.State == "" {
		return fmt.Errorf("index job requires id, generation, kind, and state")
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
 id,generation_id,kind,state,cursor,processed,total,created_at_unix,updated_at_unix,error,cancellation_requested
) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, job.ID, job.Generation, job.Kind, job.State, job.Cursor, job.Progress, job.Total,
		created.Unix(), updated.Unix(), job.Error, boolInt(job.CancellationRequested))
	if err != nil {
		return fmt.Errorf("create index job: %w", err)
	}
	return nil
}

func (store *SQLiteJobStore) Update(ctx context.Context, job Job) error {
	if err := store.ready(); err != nil {
		return err
	}
	result, err := store.db.ExecContext(ctx, `UPDATE code_facts_index_jobs SET
 state=?,cursor=?,processed=?,total=?,updated_at_unix=?,error=?,cancellation_requested=? WHERE id=?`,
		job.State, job.Cursor, job.Progress, job.Total, job.UpdatedAt.UTC().Unix(), job.Error, boolInt(job.CancellationRequested), job.ID)
	if err != nil {
		return fmt.Errorf("update index job: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return ErrJobNotFound
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
	result, err := store.db.ExecContext(ctx, `UPDATE code_facts_index_jobs SET state='cancellation_requested',cancellation_requested=1,updated_at_unix=? WHERE id=? AND state IN ('queued','running','interrupted')`, now.UTC().Unix(), id)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return ErrJobNotFound
	}
	return nil
}

func (store *SQLiteJobStore) RecoverInterrupted(ctx context.Context, now time.Time) ([]Job, error) {
	if err := store.ready(); err != nil {
		return nil, err
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE code_facts_index_jobs SET state='interrupted',updated_at_unix=?,error='process restarted before completion' WHERE state='running'`, now.UTC().Unix()); err != nil {
		return nil, err
	}
	return store.ListActive(ctx)
}

func (store *SQLiteJobStore) ready() error {
	if store == nil || store.db == nil {
		return fmt.Errorf("index job store requires database")
	}
	return nil
}

const jobSelect = `SELECT id,kind,state,generation_id,processed,total,error,cursor,cancellation_requested,created_at_unix,updated_at_unix FROM code_facts_index_jobs`

type rowScanner interface{ Scan(...any) error }

func scanJob(row rowScanner) (Job, error) {
	var job Job
	var cancelled int
	var created, updated int64
	err := row.Scan(&job.ID, &job.Kind, &job.State, &job.Generation, &job.Progress, &job.Total, &job.Error, &job.Cursor, &cancelled, &created, &updated)
	if err != nil {
		return Job{}, err
	}
	job.CancellationRequested = cancelled == 1
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

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
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
