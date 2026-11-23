package schedule

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	// Importing modernc.org/sqlite registers the pure-Go SQLite driver.
	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
)

// SchemaVersion is stamped into the scheduler database via PRAGMA user_version.
// The schema is declarative: a version bump needs a one-shot migration script,
// not an in-code ladder. Unknown or newer versions are rejected loudly so a
// downgrade cannot silently corrupt a queue.
const SchemaVersion = 1

const (
	scheduleTxRetryAttempts = 5
	// timeLayout is fixed-width UTC. Fixed width matters: these strings are
	// compared and ordered by SQLite, and RFC3339Nano drops trailing zeroes,
	// which would sort "12:00:00Z" after "12:00:00.5Z".
	timeLayout = "2006-01-02T15:04:05.000000000Z"
)

var (
	scheduleTxRetryBase = 5 * time.Millisecond
	scheduleTxRetryMax  = 250 * time.Millisecond
)

// Sentinel errors. Callers distinguish an idempotency collision from a real
// failure without parsing strings.
var (
	// ErrJobExists is returned when a job ID is already stored.
	ErrJobExists = errors.New("schedule: job already exists")
	// ErrJobNotFound is returned when a job ID is unknown.
	ErrJobNotFound = errors.New("schedule: job not found")
	// ErrNotLeasable is returned when a job is not in a waiting state, is not
	// yet due, or its deadline has passed.
	ErrNotLeasable = errors.New("schedule: job is not leasable")
	// ErrLeaseLost is returned when a lease expired or was taken by another
	// owner before the mutation landed.
	ErrLeaseLost = errors.New("schedule: lease is no longer held")
)

// Config configures the scheduler store.
type Config struct {
	// Path is the SQLite file. It must be a distinct database from the capacity
	// broker's claim ledger.
	Path string
	// HomeDir may be supplied instead of Path; the ledger then lives at
	// <runtime-home>/state/capacity-schedule.db.
	HomeDir string
	// Clock is a test seam. Nil uses time.Now.
	Clock func() time.Time
}

// Store is the durable scheduler state: jobs, execution runs, and model dwell.
// It is not the capacity ledger; claims still live in the broker's database.
type Store struct {
	db     *sql.DB
	dbPath string
	clock  func() time.Time
}

// DefaultPath resolves the scheduler database beside the capacity ledger under
// the runtime-home state directory.
func DefaultPath(homeDir string) (string, error) {
	if strings.TrimSpace(homeDir) == "" {
		dir, err := config.HomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		homeDir = dir
	}
	stateDir, err := repocontract.RuntimeHomeEntryPath(homeDir, repocontract.HomeKeyState)
	if err != nil {
		return "", fmt.Errorf("resolve runtime-home state dir: %w", err)
	}
	return filepath.Join(stateDir, "capacity-schedule.db"), nil
}

// ResolveConfiguredPath validates isolation before resolving a default path.
// Under `go test` an implicit path would open the live scheduler database, so it
// is refused exactly like the capacity ledger refuses its live path.
func ResolveConfiguredPath(cfg Config) (string, error) {
	if testing.Testing() && strings.TrimSpace(cfg.Path) == "" && strings.TrimSpace(cfg.HomeDir) == "" {
		return "", errors.New("schedule store: tests must pass an explicit Config.Path (or HomeDir); refusing to open the live scheduler database")
	}
	if strings.TrimSpace(cfg.Path) == "" {
		return DefaultPath(cfg.HomeDir)
	}
	return cfg.Path, nil
}

// NewStore opens (and lazily creates) the scheduler database.
func NewStore(ctx context.Context, cfg Config) (*Store, error) {
	path, err := ResolveConfiguredPath(cfg)
	if err != nil {
		return nil, err
	}
	if path != ":memory:" {
		absolute, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("resolve scheduler database path: %w", err)
		}
		path = absolute
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, fmt.Errorf("prepare scheduler database directory: %w", err)
		}
	}
	db, err := sql.Open("sqlite", buildDSN(path))
	if err != nil {
		return nil, fmt.Errorf("open scheduler database: %w", err)
	}
	db.SetMaxOpenConns(1)
	clock := cfg.Clock
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	store := &Store{db: db, dbPath: path, clock: clock}
	if err := store.ensureSchema(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

// Close releases the database handle.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Path returns the database path this store opened.
func (s *Store) Path() string {
	if s == nil {
		return ""
	}
	return s.dbPath
}

func (s *Store) now() time.Time {
	if s == nil || s.clock == nil {
		return time.Now().UTC()
	}
	return s.clock().UTC()
}

const schemaSQL = `
CREATE TABLE IF NOT EXISTS schedule_jobs (
  job_id            TEXT PRIMARY KEY,
  kind              TEXT NOT NULL,
  owner             TEXT NOT NULL,
  priority          TEXT NOT NULL,
  resource_class    TEXT NOT NULL,
  model_affinity    TEXT NOT NULL,
  est_seconds       REAL NOT NULL,
  est_ram_bytes     INTEGER NOT NULL,
  est_vram_bytes    INTEGER NOT NULL,
  est_items         INTEGER NOT NULL,
  est_tokens        INTEGER NOT NULL,
  value             REAL NOT NULL,
  not_before_at     TEXT NOT NULL,
  deadline_at       TEXT,
  batch_key         TEXT NOT NULL,
  max_batch         INTEGER NOT NULL,
  max_attempts      INTEGER NOT NULL,
  payload_ref       TEXT NOT NULL,
  checkpoint        TEXT NOT NULL,
  state             TEXT NOT NULL,
  attempts          INTEGER NOT NULL,
  deferrals         INTEGER NOT NULL,
  last_reason       TEXT NOT NULL,
  lease_owner       TEXT NOT NULL,
  lease_expires_at  TEXT,
  created_at        TEXT NOT NULL,
  updated_at        TEXT NOT NULL,
  started_at        TEXT,
  finished_at       TEXT
);
CREATE INDEX IF NOT EXISTS idx_schedule_jobs_ready
  ON schedule_jobs (state, not_before_at);
CREATE INDEX IF NOT EXISTS idx_schedule_jobs_kind ON schedule_jobs (kind);
CREATE INDEX IF NOT EXISTS idx_schedule_jobs_owner ON schedule_jobs (owner);
CREATE INDEX IF NOT EXISTS idx_schedule_jobs_model ON schedule_jobs (model_affinity);

CREATE TABLE IF NOT EXISTS schedule_runs (
  run_id             TEXT PRIMARY KEY,
  job_id             TEXT NOT NULL,
  kind               TEXT NOT NULL,
  owner              TEXT NOT NULL,
  priority           TEXT NOT NULL,
  model_affinity     TEXT NOT NULL,
  started_at         TEXT NOT NULL,
  finished_at        TEXT,
  estimated_seconds  REAL NOT NULL,
  actual_seconds     REAL NOT NULL,
  items              INTEGER NOT NULL,
  tokens             INTEGER NOT NULL,
  model_load         INTEGER NOT NULL,
  outcome            TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_schedule_runs_window ON schedule_runs (started_at);
CREATE INDEX IF NOT EXISTS idx_schedule_runs_job ON schedule_runs (job_id, outcome);

CREATE TABLE IF NOT EXISTS schedule_models (
  model            TEXT PRIMARY KEY,
  first_loaded_at  TEXT NOT NULL,
  last_used_at     TEXT NOT NULL
);
`

func (s *Store) ensureSchema(ctx context.Context) error {
	return s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		var version int
		if err := tx.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
			return fmt.Errorf("read scheduler schema version: %w", err)
		}
		switch {
		case version == SchemaVersion:
			return nil
		case version == 0:
			if _, err := tx.ExecContext(ctx, schemaSQL); err != nil {
				return fmt.Errorf("create scheduler schema: %w", err)
			}
			if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", SchemaVersion)); err != nil {
				return fmt.Errorf("stamp scheduler schema version: %w", err)
			}
			return nil
		default:
			return fmt.Errorf("scheduler database schema version %d is unsupported (want %d); refusing to guess a migration", version, SchemaVersion)
		}
	})
}

// Submit stores a new job. A duplicate ID returns ErrJobExists so a producer can
// treat retries as idempotent.
func (s *Store) Submit(ctx context.Context, job Job) (Job, error) {
	jobs, err := s.SubmitBatch(ctx, []Job{job})
	if err != nil {
		return Job{}, err
	}
	return jobs[0], nil
}

// SubmitBatch stores several jobs atomically. Either every job is accepted or
// none is, so a producer cannot half-persist a unit of work.
func (s *Store) SubmitBatch(ctx context.Context, jobs []Job) ([]Job, error) {
	if len(jobs) == 0 {
		return nil, nil
	}
	now := s.now()
	prepared := make([]Job, len(jobs))
	for i, job := range jobs {
		normalized, err := job.normalize(now)
		if err != nil {
			return nil, err
		}
		prepared[i] = normalized
	}
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		for _, job := range prepared {
			if _, err := tx.ExecContext(ctx, `INSERT INTO schedule_jobs (
				job_id, kind, owner, priority, resource_class, model_affinity,
				est_seconds, est_ram_bytes, est_vram_bytes, est_items, est_tokens,
				value, not_before_at, deadline_at, batch_key, max_batch, max_attempts,
				payload_ref, checkpoint, state, attempts, deferrals, last_reason,
				lease_owner, lease_expires_at, created_at, updated_at, started_at, finished_at
			) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
				job.ID, job.Kind, job.Owner, string(job.Priority), string(job.ResourceClass), job.ModelAffinity,
				job.Estimate.Seconds, job.Estimate.RAMBytes, job.Estimate.VRAMBytes, job.Estimate.Items, job.Estimate.Tokens,
				job.Value, formatTime(job.NotBefore), nullableTime(job.Deadline), job.BatchKey, job.MaxBatch, job.MaxAttempts,
				job.PayloadRef, job.Checkpoint, string(job.State), job.Attempts, job.Deferrals, job.LastReason,
				"", nil, formatTime(job.CreatedAt), formatTime(job.UpdatedAt), nil, nil,
			); err != nil {
				if isUniqueViolation(err) {
					return fmt.Errorf("%w: %s", ErrJobExists, job.ID)
				}
				return fmt.Errorf("insert job %s: %w", job.ID, err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return prepared, nil
}

// Ready returns waiting jobs whose NotBefore has arrived, oldest first.
func (s *Store) Ready(ctx context.Context, limit int) ([]Job, error) {
	if limit <= 0 {
		limit = 500
	}
	now := s.now()
	rows, err := s.db.QueryContext(ctx, jobSelectSQL+`
WHERE state IN (?, ?) AND not_before_at <= ?
ORDER BY not_before_at ASC, created_at ASC
LIMIT ?`, string(StateQueued), string(StateDeferred), formatTime(now), limit)
	if err != nil {
		return nil, fmt.Errorf("query ready jobs: %w", err)
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

// List returns stored jobs matching the filter, oldest due first.
type ListFilter struct {
	States []State
	Kinds  []string
	Owners []string
	Models []string
	Limit  int
}

// List returns stored jobs matching the filter.
func (s *Store) List(ctx context.Context, filter ListFilter) ([]Job, error) {
	query := jobSelectSQL + " WHERE 1=1"
	var args []any
	if len(filter.States) > 0 {
		query += " AND state IN (" + placeholders(len(filter.States)) + ")"
		for _, state := range filter.States {
			args = append(args, string(state))
		}
	}
	if len(filter.Kinds) > 0 {
		query += " AND kind IN (" + placeholders(len(filter.Kinds)) + ")"
		for _, kind := range filter.Kinds {
			args = append(args, kind)
		}
	}
	if len(filter.Owners) > 0 {
		query += " AND owner IN (" + placeholders(len(filter.Owners)) + ")"
		for _, owner := range filter.Owners {
			args = append(args, owner)
		}
	}
	if len(filter.Models) > 0 {
		query += " AND model_affinity IN (" + placeholders(len(filter.Models)) + ")"
		for _, model := range filter.Models {
			args = append(args, model)
		}
	}
	query += " ORDER BY not_before_at ASC, created_at ASC"
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
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

// Get returns one job.
func (s *Store) Get(ctx context.Context, jobID string) (Job, error) {
	row := s.db.QueryRowContext(ctx, jobSelectSQL+" WHERE job_id = ?", jobID)
	job, err := scanJob(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Job{}, fmt.Errorf("%w: %s", ErrJobNotFound, jobID)
	}
	return job, err
}

// LeaseOptions describes one batch lease.
type LeaseOptions struct {
	// JobIDs are leased together. The operation is atomic: if any job is no
	// longer leasable, none is leased.
	JobIDs []string
	// Owner identifies this runner instance on the lease.
	Owner string
	// TTL bounds the lease without renewal.
	TTL time.Duration
	// ModelDwell is the window in which a model counts as already loaded.
	ModelDwell time.Duration
}

// LeasedJob is a job this process now owns plus its run identity.
type LeasedJob struct {
	Job       Job
	RunID     string
	ModelLoad bool
	ExpiresAt time.Time
}

// LeaseClaim is the fencing token returned with a running job. Attempts are
// monotonic per job, so the owner/attempt pair cannot authorize a later retry.
type LeaseClaim struct {
	JobID   string
	Owner   string
	Attempt int
}

const activeLeaseWhere = "job_id = ? AND state = ? AND lease_owner = ? AND attempts = ? AND lease_expires_at > ?"

func activeLeaseArgs(claim LeaseClaim, now time.Time) []any {
	return []any{claim.JobID, string(StateRunning), claim.Owner, claim.Attempt, formatTime(now)}
}

// LeaseClaim returns the identity that may update this job's current attempt.
func (j Job) LeaseClaim() LeaseClaim {
	return LeaseClaim{JobID: j.ID, Owner: j.LeaseOwner, Attempt: j.Attempts}
}

func ownsActiveLease(job Job, claim LeaseClaim, now time.Time) bool {
	return claim.JobID != "" && claim.Owner != "" && claim.Attempt > 0 &&
		job.ID == claim.JobID && job.State == StateRunning && job.LeaseOwner == claim.Owner &&
		job.Attempts == claim.Attempt && job.LeaseExpiresAt != nil && job.LeaseExpiresAt.After(now)
}

// Lease atomically marks jobs running, records a run for each, and decides which
// of them pay for a cold model load. Lease is the only path into execution.
func (s *Store) Lease(ctx context.Context, opts LeaseOptions) ([]LeasedJob, error) {
	if len(opts.JobIDs) == 0 {
		return nil, fmt.Errorf("%w: no job ids", ErrNotLeasable)
	}
	if strings.TrimSpace(opts.Owner) == "" {
		return nil, fmt.Errorf("%w: lease owner is required", ErrNotLeasable)
	}
	ttl := opts.TTL
	if ttl <= 0 {
		ttl = DefaultLeaseTTL
	}
	dwell := opts.ModelDwell
	if dwell <= 0 {
		dwell = DefaultModelDwell
	}
	now := s.now()
	expiresAt := now.Add(ttl)
	leased := make([]LeasedJob, 0, len(opts.JobIDs))
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		leased = leased[:0]
		for _, jobID := range opts.JobIDs {
			job, err := getJobTx(ctx, tx, jobID)
			if err != nil {
				return err
			}
			if !job.State.Waiting() {
				return fmt.Errorf("%w: job %s is %s", ErrNotLeasable, jobID, job.State)
			}
			if !job.Due(now) {
				return fmt.Errorf("%w: job %s is not due until %s", ErrNotLeasable, jobID, formatTime(job.NotBefore))
			}
			if job.Overdue(now) {
				return fmt.Errorf("%w: job %s is past its deadline", ErrNotLeasable, jobID)
			}
			modelLoad, err := markModelUsedTx(ctx, tx, job.ModelAffinity, dwell, now)
			if err != nil {
				return err
			}
			runID := NewID("run")
			if _, err := tx.ExecContext(ctx, `INSERT INTO schedule_runs (
				run_id, job_id, kind, owner, priority, model_affinity, started_at,
				estimated_seconds, actual_seconds, items, tokens, model_load, outcome
			) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
				runID, job.ID, job.Kind, job.Owner, string(job.Priority), job.ModelAffinity, formatTime(now),
				job.Estimate.Seconds, 0.0, 0, 0, boolToInt(modelLoad), runOutcomeRunning,
			); err != nil {
				return fmt.Errorf("record run for job %s: %w", jobID, err)
			}
			startedAt := now
			if job.StartedAt != nil {
				startedAt = *job.StartedAt
			}
			if _, err := tx.ExecContext(ctx, `UPDATE schedule_jobs SET
				state = ?, attempts = attempts + 1, lease_owner = ?, lease_expires_at = ?,
				started_at = ?, updated_at = ?
				WHERE job_id = ?`,
				string(StateRunning), opts.Owner, formatTime(expiresAt), formatTime(startedAt), formatTime(now), jobID,
			); err != nil {
				return fmt.Errorf("lease job %s: %w", jobID, err)
			}
			job, err = getJobTx(ctx, tx, jobID)
			if err != nil {
				return err
			}
			leased = append(leased, LeasedJob{Job: job, RunID: runID, ModelLoad: modelLoad, ExpiresAt: expiresAt})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return leased, nil
}

// HeartbeatLease extends a running lease. It fails with ErrLeaseLost when the
// lease expired or was taken over.
func (s *Store) HeartbeatLease(ctx context.Context, claim LeaseClaim, ttl time.Duration) (time.Time, error) {
	if claim.JobID == "" || claim.Owner == "" || claim.Attempt <= 0 {
		return time.Time{}, fmt.Errorf("%w: lease claim is required", ErrLeaseLost)
	}
	if ttl <= 0 {
		ttl = DefaultLeaseTTL
	}
	now := s.now()
	expiresAt := now.Add(ttl)
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		args := append([]any{formatTime(expiresAt), formatTime(now)}, activeLeaseArgs(claim, now)...)
		result, err := tx.ExecContext(ctx, `UPDATE schedule_jobs
			SET lease_expires_at = ?, updated_at = ?
			WHERE `+activeLeaseWhere, args...)
		if err != nil {
			return fmt.Errorf("heartbeat job %s: %w", claim.JobID, err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return fmt.Errorf("%w: job %s lease is expired or belongs to another attempt", ErrLeaseLost, claim.JobID)
		}
		return nil
	})
	if err != nil {
		return time.Time{}, err
	}
	return expiresAt, nil
}

// Checkpoint persists resumable progress while a job is waiting. A running
// executor must use CheckpointLeased so its write is fenced to its attempt.
func (s *Store) Checkpoint(ctx context.Context, jobID, checkpoint string) error {
	now := s.now()
	return s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE schedule_jobs SET checkpoint = ?, updated_at = ?
			WHERE job_id = ? AND state IN (?, ?)`,
			checkpoint, formatTime(now), jobID, string(StateQueued), string(StateDeferred))
		if err != nil {
			return fmt.Errorf("checkpoint waiting job %s: %w", jobID, err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			current, err := getJobTx(ctx, tx, jobID)
			if err != nil {
				return err
			}
			return fmt.Errorf("%w: job %s is %s; running checkpoints require a lease claim", ErrNotLeasable, jobID, current.State)
		}
		return nil
	})
}

// CheckpointLeased persists progress from the active attempt. An expired or
// stale worker cannot overwrite a checkpoint from its successor.
func (s *Store) CheckpointLeased(ctx context.Context, claim LeaseClaim, checkpoint string) error {
	if claim.JobID == "" || claim.Owner == "" || claim.Attempt <= 0 {
		return fmt.Errorf("%w: lease claim is required", ErrLeaseLost)
	}
	now := s.now()
	return s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		args := append([]any{checkpoint, formatTime(now)}, activeLeaseArgs(claim, now)...)
		result, err := tx.ExecContext(ctx, `UPDATE schedule_jobs SET checkpoint = ?, updated_at = ?
			WHERE `+activeLeaseWhere, args...)
		if err != nil {
			return fmt.Errorf("checkpoint job %s: %w", claim.JobID, err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			current, err := getJobTx(ctx, tx, claim.JobID)
			if err != nil {
				return err
			}
			return fmt.Errorf("%w: job %s lease is expired or belongs to another attempt (state %s)", ErrLeaseLost, claim.JobID, current.State)
		}
		return nil
	})
}

// Completion records the terminal outcome of a leased job.
type Completion struct {
	JobID string
	// Lease must match the running attempt, including its expiry, before it can
	// close that attempt's run.
	Lease         LeaseClaim
	Outcome       State // completed or failed
	ActualSeconds float64
	Items         int
	Tokens        int
	Checkpoint    string
	Reason        string
}

// Complete closes a leased job and its run, recording actual cost.
func (s *Store) Complete(ctx context.Context, completion Completion) (Job, error) {
	if completion.Outcome != StateCompleted && completion.Outcome != StateFailed {
		return Job{}, fmt.Errorf("%w: completion outcome must be completed or failed", ErrInvalidJob)
	}
	if completion.ActualSeconds < 0 || completion.Items < 0 || completion.Tokens < 0 {
		return Job{}, fmt.Errorf("%w: completion costs cannot be negative", ErrInvalidJob)
	}
	now := s.now()
	var job Job
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		current, err := getJobTx(ctx, tx, completion.JobID)
		if err != nil {
			return err
		}
		if completion.Lease.JobID != completion.JobID || !ownsActiveLease(current, completion.Lease, now) {
			return fmt.Errorf("%w: job %s lease is expired or belongs to another attempt", ErrLeaseLost, completion.JobID)
		}
		actual := completion.ActualSeconds
		if actual <= 0 {
			actual = current.Estimate.Seconds
		}
		if err := finishOpenRunTx(ctx, tx, current, runOutcome(completion.Outcome), actual, completion.Items, completion.Tokens, now); err != nil {
			return err
		}
		checkpoint := completion.Checkpoint
		if checkpoint == "" {
			checkpoint = current.Checkpoint
		}
		args := append([]any{string(completion.Outcome), checkpoint, completion.Reason, formatTime(now), formatTime(now)}, activeLeaseArgs(completion.Lease, now)...)
		result, err := tx.ExecContext(ctx, `UPDATE schedule_jobs SET
			state = ?, checkpoint = ?, last_reason = ?, finished_at = ?, updated_at = ?,
			lease_owner = '', lease_expires_at = NULL
			WHERE `+activeLeaseWhere, args...)
		if err != nil {
			return fmt.Errorf("complete job %s: %w", completion.JobID, err)
		}
		if affected, err := result.RowsAffected(); err != nil {
			return err
		} else if affected == 0 {
			return fmt.Errorf("%w: job %s lease is expired or belongs to another attempt", ErrLeaseLost, completion.JobID)
		}
		if current.ModelAffinity != "" {
			if err := touchModelTx(ctx, tx, current.ModelAffinity, now); err != nil {
				return err
			}
		}
		job, err = getJobTx(ctx, tx, completion.JobID)
		return err
	})
	if err != nil {
		return Job{}, err
	}
	return job, nil
}

// DeferOptions reschedules a job after a denial or a recoverable failure.
type DeferOptions struct {
	JobID string
	// Lease is required when the job is running; queued/deferred admission
	// deferrals are not owned by an executor attempt.
	Lease     LeaseClaim
	Reason    string
	NotBefore time.Time // zero means "as soon as the backoff allows"
	// Backoff computes NotBefore from the job's deferral count when NotBefore
	// is zero. Nil uses DefaultDeferBase/DefaultDeferCap.
	Backoff func(deferrals int) time.Duration
	// Failed marks the run outcome as failed instead of deferred. The job still
	// returns to the queue; use Completion{Outcome: StateFailed} for terminal
	// failure.
	Failed bool
	// ActualSeconds, Items and Tokens record what a failed attempt consumed.
	// Zero actual seconds falls back to the elapsed lease time.
	ActualSeconds float64
	Items         int
	Tokens        int
}

// Defer returns a job to the waiting set with a reason and a next-due time. A
// running job's open run is closed as deferred, so its elapsed time still counts
// against the seconds budget.
func (s *Store) Defer(ctx context.Context, opts DeferOptions) (Job, error) {
	now := s.now()
	var job Job
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		current, err := getJobTx(ctx, tx, opts.JobID)
		if err != nil {
			return err
		}
		if current.State.Terminal() {
			return fmt.Errorf("%w: job %s is %s", ErrNotLeasable, opts.JobID, current.State)
		}
		if current.State == StateRunning && (opts.Lease.JobID != opts.JobID || !ownsActiveLease(current, opts.Lease, now)) {
			return fmt.Errorf("%w: job %s lease is expired or belongs to another attempt", ErrLeaseLost, opts.JobID)
		}
		notBefore := opts.NotBefore
		if notBefore.IsZero() {
			deferrals := current.Deferrals + 1
			delay := DefaultDeferBase
			if opts.Backoff != nil {
				delay = opts.Backoff(deferrals)
			}
			if delay < 0 {
				delay = 0
			}
			notBefore = now.Add(delay)
		}
		if current.State == StateRunning {
			actual := opts.ActualSeconds
			if actual <= 0 {
				actual = elapsedSeconds(current, now)
			}
			outcome := runOutcomeDeferred
			if opts.Failed {
				outcome = runOutcomeFailed
			}
			if err := finishOpenRunTx(ctx, tx, current, outcome, actual, opts.Items, opts.Tokens, now); err != nil {
				return err
			}
		}
		result, err := tx.ExecContext(ctx, `UPDATE schedule_jobs SET
			state = ?, not_before_at = ?, deferrals = deferrals + 1, last_reason = ?,
			updated_at = ?, lease_owner = '', lease_expires_at = NULL
			WHERE job_id = ? AND state = ?`,
			string(StateDeferred), formatTime(notBefore), opts.Reason, formatTime(now), opts.JobID, string(current.State),
		)
		if err != nil {
			return fmt.Errorf("defer job %s: %w", opts.JobID, err)
		}
		if affected, err := result.RowsAffected(); err != nil {
			return err
		} else if affected == 0 {
			return fmt.Errorf("%w: job %s changed during deferral", ErrLeaseLost, opts.JobID)
		}
		job, err = getJobTx(ctx, tx, opts.JobID)
		return err
	})
	if err != nil {
		return Job{}, err
	}
	return job, nil
}

// Cancel withdraws a job. Canceling a running job closes its open run and
// charges the elapsed time.
func (s *Store) Cancel(ctx context.Context, jobID, reason string) (Job, error) {
	now := s.now()
	var job Job
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		current, err := getJobTx(ctx, tx, jobID)
		if err != nil {
			return err
		}
		if current.State.Terminal() {
			return fmt.Errorf("%w: job %s is already %s", ErrNotLeasable, jobID, current.State)
		}
		if current.State == StateRunning {
			if err := finishOpenRunTx(ctx, tx, current, runOutcomeCanceled, elapsedSeconds(current, now), current.Estimate.Items, current.Estimate.Tokens, now); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE schedule_jobs SET
			state = ?, last_reason = ?, finished_at = ?, updated_at = ?,
			lease_owner = '', lease_expires_at = NULL
			WHERE job_id = ?`,
			string(StateCanceled), reason, formatTime(now), formatTime(now), jobID,
		); err != nil {
			return fmt.Errorf("cancel job %s: %w", jobID, err)
		}
		job, err = getJobTx(ctx, tx, jobID)
		return err
	})
	if err != nil {
		return Job{}, err
	}
	return job, nil
}

// ReapExpiredLeases requeues running jobs whose lease expired (a crashed or
// partitioned runner) and fails those that exhausted their attempts. The next
// tick calls it before selecting, which is what makes crash recovery automatic.
func (s *Store) ReapExpiredLeases(ctx context.Context) ([]Job, error) {
	now := s.now()
	var recovered []Job
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		recovered = recovered[:0]
		rows, err := tx.QueryContext(ctx, jobSelectSQL+`
WHERE state = ? AND lease_expires_at IS NOT NULL AND lease_expires_at <= ?
ORDER BY lease_expires_at ASC`, string(StateRunning), formatTime(now))
		if err != nil {
			return fmt.Errorf("query expired leases: %w", err)
		}
		var expired []Job
		for rows.Next() {
			job, err := scanJob(rows)
			if err != nil {
				rows.Close()
				return err
			}
			expired = append(expired, job)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		for _, job := range expired {
			if err := finishOpenRunTx(ctx, tx, job, runOutcomeExpired, elapsedSeconds(job, now), job.Estimate.Items, job.Estimate.Tokens, now); err != nil {
				return err
			}
			reason := fmt.Sprintf("lease expired after %d attempt(s); requeued", job.Attempts)
			state := StateDeferred
			finished := any(nil)
			if job.Attempts >= job.MaxAttempts {
				state = StateFailed
				reason = fmt.Sprintf("lease expired on the final attempt (%d/%d)", job.Attempts, job.MaxAttempts)
				finished = formatTime(now)
			}
			if _, err := tx.ExecContext(ctx, `UPDATE schedule_jobs SET
				state = ?, not_before_at = ?, last_reason = ?, updated_at = ?,
				finished_at = ?, lease_owner = '', lease_expires_at = NULL
				WHERE job_id = ? AND state = ? AND lease_expires_at = ?`,
				string(state), formatTime(now), reason, formatTime(now), finished,
				job.ID, string(StateRunning), formatTimePointer(job.LeaseExpiresAt),
			); err != nil {
				return fmt.Errorf("recover job %s: %w", job.ID, err)
			}
			updated, err := getJobTx(ctx, tx, job.ID)
			if err != nil {
				return err
			}
			recovered = append(recovered, updated)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return recovered, nil
}

// ExpireOverdue fails waiting jobs whose deadline passed.
func (s *Store) ExpireOverdue(ctx context.Context) (int, error) {
	now := s.now()
	var expired int64
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE schedule_jobs SET
			state = ?, last_reason = ?, finished_at = ?, updated_at = ?
			WHERE state IN (?, ?) AND deadline_at IS NOT NULL AND deadline_at <= ?`,
			string(StateFailed), "deadline exceeded", formatTime(now), formatTime(now),
			string(StateQueued), string(StateDeferred), formatTime(now))
		if err != nil {
			return fmt.Errorf("expire overdue jobs: %w", err)
		}
		expired, err = result.RowsAffected()
		return err
	})
	if err != nil {
		return 0, err
	}
	return int(expired), nil
}

// budgetSnapshot is the rolling-hour consumption used by selection and status.
type budgetSnapshot struct {
	From       time.Time
	Seconds    float64
	Jobs       int
	ModelLoads int
	ByOwner    map[string]float64
}

// Budget returns the consumption inside the rolling hour ending now.
func (s *Store) Budget(ctx context.Context) (BudgetStatus, error) {
	snapshot, err := s.budgetSnapshot(ctx, s.now())
	if err != nil {
		return BudgetStatus{}, err
	}
	return BudgetStatus{
		WindowStart: snapshot.From,
		SecondsUsed: snapshot.Seconds,
		JobsUsed:    snapshot.Jobs,
		ModelLoads:  snapshot.ModelLoads,
	}, nil
}

func (s *Store) budgetSnapshot(ctx context.Context, at time.Time) (budgetSnapshot, error) {
	from := at.Add(-time.Hour)
	snapshot := budgetSnapshot{From: from, ByOwner: map[string]float64{}}
	rows, err := s.db.QueryContext(ctx, `SELECT owner,
		COALESCE(SUM(CASE WHEN outcome = ? OR actual_seconds <= 0 THEN estimated_seconds ELSE actual_seconds END), 0),
		COUNT(*),
		COALESCE(SUM(model_load), 0)
		FROM schedule_runs WHERE started_at >= ? GROUP BY owner`,
		runOutcomeRunning, formatTime(from))
	if err != nil {
		return budgetSnapshot{}, fmt.Errorf("read scheduler budget: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var owner string
		var seconds float64
		var jobs, loads int
		if err := rows.Scan(&owner, &seconds, &jobs, &loads); err != nil {
			return budgetSnapshot{}, err
		}
		snapshot.Seconds += seconds
		snapshot.Jobs += jobs
		snapshot.ModelLoads += loads
		snapshot.ByOwner[owner] += seconds
	}
	return snapshot, rows.Err()
}

func getJobTx(ctx context.Context, tx *sql.Tx, jobID string) (Job, error) {
	row := tx.QueryRowContext(ctx, jobSelectSQL+" WHERE job_id = ?", jobID)
	job, err := scanJob(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Job{}, fmt.Errorf("%w: %s", ErrJobNotFound, jobID)
	}
	return job, err
}

const jobSelectSQL = `SELECT
  job_id, kind, owner, priority, resource_class, model_affinity,
  est_seconds, est_ram_bytes, est_vram_bytes, est_items, est_tokens,
  value, not_before_at, deadline_at, batch_key, max_batch, max_attempts,
  payload_ref, checkpoint, state, attempts, deferrals, last_reason,
  lease_owner, lease_expires_at, created_at, updated_at, started_at, finished_at
FROM schedule_jobs`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanJob(row rowScanner) (Job, error) {
	var (
		job                                               Job
		priority, resourceClass, state                    string
		notBefore, createdAt, updatedAt                   string
		deadlineAt, leaseExpiresAt, startedAt, finishedAt sql.NullString
	)
	err := row.Scan(
		&job.ID, &job.Kind, &job.Owner, &priority, &resourceClass, &job.ModelAffinity,
		&job.Estimate.Seconds, &job.Estimate.RAMBytes, &job.Estimate.VRAMBytes, &job.Estimate.Items, &job.Estimate.Tokens,
		&job.Value, &notBefore, &deadlineAt, &job.BatchKey, &job.MaxBatch, &job.MaxAttempts,
		&job.PayloadRef, &job.Checkpoint, &state, &job.Attempts, &job.Deferrals, &job.LastReason,
		&job.LeaseOwner, &leaseExpiresAt, &createdAt, &updatedAt, &startedAt, &finishedAt,
	)
	if err != nil {
		return Job{}, err
	}
	job.Priority = Priority(priority)
	job.ResourceClass = ResourceClass(resourceClass)
	job.State = State(state)
	if job.NotBefore, err = parseTime(notBefore); err != nil {
		return Job{}, err
	}
	if job.CreatedAt, err = parseTime(createdAt); err != nil {
		return Job{}, err
	}
	if job.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return Job{}, err
	}
	if job.Deadline, err = scanOptionalTime(deadlineAt); err != nil {
		return Job{}, err
	}
	if job.LeaseExpiresAt, err = scanOptionalTime(leaseExpiresAt); err != nil {
		return Job{}, err
	}
	if job.StartedAt, err = scanOptionalTime(startedAt); err != nil {
		return Job{}, err
	}
	if job.FinishedAt, err = scanOptionalTime(finishedAt); err != nil {
		return Job{}, err
	}
	return job, nil
}

// finishOpenRunTx closes the newest open run for job. A missing open run is not
// an error: it means the job never reached execution (or was already closed).
const (
	runOutcomeRunning   = "running"
	runOutcomeCompleted = "completed"
	runOutcomeFailed    = "failed"
	runOutcomeCanceled  = "canceled"
	runOutcomeDeferred  = "deferred"
	runOutcomeExpired   = "expired"
)

func runOutcome(state State) string {
	if state == StateFailed {
		return runOutcomeFailed
	}
	return runOutcomeCompleted
}

// finishOpenRunTx closes the newest open run for job. A missing open run is not
// an error: it means the job never reached execution (or was already closed).
func finishOpenRunTx(ctx context.Context, tx *sql.Tx, job Job, outcome string, actualSeconds float64, items, tokens int, now time.Time) error {
	var runID string
	err := tx.QueryRowContext(ctx, `SELECT run_id FROM schedule_runs
		WHERE job_id = ? AND outcome = ? ORDER BY started_at DESC LIMIT 1`,
		job.ID, runOutcomeRunning).Scan(&runID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read open run for job %s: %w", job.ID, err)
	}
	if actualSeconds < 0 {
		actualSeconds = 0
	}
	if _, err := tx.ExecContext(ctx, `UPDATE schedule_runs SET
		outcome = ?, finished_at = ?, actual_seconds = ?, items = ?, tokens = ?
		WHERE run_id = ?`,
		outcome, formatTime(now), actualSeconds, items, tokens, runID); err != nil {
		return fmt.Errorf("close run for job %s: %w", job.ID, err)
	}
	return nil
}

func markModelUsedTx(ctx context.Context, tx *sql.Tx, model string, dwell time.Duration, now time.Time) (bool, error) {
	if strings.TrimSpace(model) == "" {
		return false, nil
	}
	var lastUsed string
	err := tx.QueryRowContext(ctx, `SELECT last_used_at FROM schedule_models WHERE model = ?`, model).Scan(&lastUsed)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if _, err := tx.ExecContext(ctx, `INSERT INTO schedule_models (model, first_loaded_at, last_used_at) VALUES (?,?,?)`,
			model, formatTime(now), formatTime(now)); err != nil {
			return false, fmt.Errorf("record first model load %s: %w", model, err)
		}
		return true, nil
	case err != nil:
		return false, fmt.Errorf("read model dwell for %s: %w", model, err)
	}
	last, err := parseTime(lastUsed)
	if err != nil {
		return false, err
	}
	hot := dwell > 0 && now.Sub(last) < dwell
	if err := touchModelTx(ctx, tx, model, now); err != nil {
		return false, err
	}
	return !hot, nil
}

func touchModelTx(ctx context.Context, tx *sql.Tx, model string, now time.Time) error {
	if strings.TrimSpace(model) == "" {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schedule_models (model, first_loaded_at, last_used_at)
		VALUES (?,?,?)
		ON CONFLICT(model) DO UPDATE SET last_used_at = excluded.last_used_at`,
		model, formatTime(now), formatTime(now)); err != nil {
		return fmt.Errorf("touch model %s: %w", model, err)
	}
	return nil
}

// HotModels returns the models still inside their dwell window at now.
func (s *Store) HotModels(ctx context.Context, dwell time.Duration) (map[string]time.Time, error) {
	if dwell <= 0 {
		dwell = DefaultModelDwell
	}
	now := s.now()
	cutoff := now.Add(-dwell)
	rows, err := s.db.QueryContext(ctx, `SELECT model, last_used_at FROM schedule_models WHERE last_used_at > ?`, formatTime(cutoff))
	if err != nil {
		return nil, fmt.Errorf("read hot models: %w", err)
	}
	defer rows.Close()
	hot := map[string]time.Time{}
	for rows.Next() {
		var model, lastUsed string
		if err := rows.Scan(&model, &lastUsed); err != nil {
			return nil, err
		}
		at, err := parseTime(lastUsed)
		if err != nil {
			return nil, err
		}
		hot[model] = at
	}
	return hot, rows.Err()
}

func elapsedSeconds(job Job, now time.Time) float64 {
	if job.StartedAt == nil {
		return 0
	}
	elapsed := now.Sub(*job.StartedAt).Seconds()
	if elapsed < 0 {
		return 0
	}
	return elapsed
}

func formatTime(t time.Time) string { return t.UTC().Format(timeLayout) }

func formatTimePointer(t *time.Time) any {
	if t == nil {
		return nil
	}
	return formatTime(*t)
}

func parseTime(raw string) (time.Time, error) {
	parsed, err := time.Parse(timeLayout, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse scheduler timestamp %q: %w", raw, err)
	}
	return parsed.UTC(), nil
}

func nullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return formatTime(*t)
}

func scanOptionalTime(raw sql.NullString) (*time.Time, error) {
	if !raw.Valid || raw.String == "" {
		return nil, nil
	}
	parsed, err := parseTime(raw.String)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", n), ",")
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func buildDSN(path string) string {
	return "file:" + url.PathEscape(path) + "?_txlock=immediate&_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)&_pragma=temp_store(MEMORY)"
}

func (s *Store) withTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin scheduler transaction: %w", err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit scheduler transaction: %w", err)
	}
	return nil
}

func (s *Store) withRetryableTx(ctx context.Context, fn func(*sql.Tx) error) error {
	var err error
	delay := scheduleTxRetryBase
	for attempt := 1; attempt <= scheduleTxRetryAttempts; attempt++ {
		err = s.withTx(ctx, fn)
		if err == nil || !isSQLiteLockContention(err) || attempt == scheduleTxRetryAttempts {
			return err
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		delay *= 2
		if delay > scheduleTxRetryMax {
			delay = scheduleTxRetryMax
		}
	}
	return err
}

func isSQLiteLockContention(err error) bool {
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) {
		switch sqliteErr.Code() & 0xff {
		case sqlite3.SQLITE_BUSY, sqlite3.SQLITE_LOCKED:
			return true
		}
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") || strings.Contains(msg, "table in the database is locked")
}

func isUniqueViolation(err error) bool {
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) {
		return sqliteErr.Code()&0xff == sqlite3.SQLITE_CONSTRAINT
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint")
}
