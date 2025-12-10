package agents

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	pq "github.com/lib/pq"
)

// PostgresAgentRepository persists agents to PostgreSQL.
type PostgresAgentRepository struct {
	db *sql.DB
}

// NewPostgresAgentRepository creates a new PostgreSQL-backed agent repository.
func NewPostgresAgentRepository(db *sql.DB) *PostgresAgentRepository {
	return &PostgresAgentRepository{db: db}
}

// Create inserts a new agent record.
func (r *PostgresAgentRepository) Create(ctx context.Context, input CreateAgentInput) (*SpawnedAgent, error) {
	const q = `
INSERT INTO spawned_agents (
	id, idempotency_key, scenario, scope, phases, model, status, prompt_hash, prompt_index, prompt_text
) VALUES (
	$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING started_at, created_at, updated_at
`
	agent := &SpawnedAgent{
		ID:             input.ID,
		IdempotencyKey: input.IdempotencyKey,
		Scenario:       input.Scenario,
		Scope:          input.Scope,
		Phases:         input.Phases,
		Model:          input.Model,
		Status:         AgentStatusPending,
		PromptHash:     input.PromptHash,
		PromptIndex:    input.PromptIndex,
		PromptText:     input.PromptText,
	}

	if agent.Scope == nil {
		agent.Scope = []string{}
	}
	if agent.Phases == nil {
		agent.Phases = []string{}
	}

	// Use NULL for empty idempotency key
	var idempKey interface{}
	if agent.IdempotencyKey != "" {
		idempKey = agent.IdempotencyKey
	}

	err := r.db.QueryRowContext(
		ctx,
		q,
		agent.ID,
		idempKey,
		agent.Scenario,
		pq.Array(agent.Scope),
		pq.Array(agent.Phases),
		agent.Model,
		string(agent.Status),
		agent.PromptHash,
		agent.PromptIndex,
		agent.PromptText,
	).Scan(&agent.StartedAt, &agent.CreatedAt, &agent.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
	}
	return agent, nil
}

// GetByIdempotencyKey retrieves an agent by its idempotency key.
func (r *PostgresAgentRepository) GetByIdempotencyKey(ctx context.Context, key string) (*SpawnedAgent, error) {
	if key == "" {
		return nil, nil
	}
	const q = `
SELECT
	id, idempotency_key, session_id, scenario, scope, phases, model, status,
	prompt_hash, prompt_index, prompt_text, output, error,
	started_at, completed_at, created_at, updated_at
FROM spawned_agents
WHERE idempotency_key = $1
`
	row := r.db.QueryRowContext(ctx, q, key)
	agent, err := scanAgentWithIdempKey(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get agent by idempotency key %s: %w", key, err)
	}
	return agent, nil
}

// Get retrieves an agent by ID.
func (r *PostgresAgentRepository) Get(ctx context.Context, id string) (*SpawnedAgent, error) {
	const q = `
SELECT
	id, session_id, scenario, scope, phases, model, status,
	prompt_hash, prompt_index, prompt_text, output, error,
	started_at, completed_at, created_at, updated_at, pid, hostname
FROM spawned_agents
WHERE id = $1
`
	row := r.db.QueryRowContext(ctx, q, id)
	agent, err := scanAgent(row, true)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get agent %s: %w", id, err)
	}
	return agent, nil
}

// Update modifies an existing agent.
func (r *PostgresAgentRepository) Update(ctx context.Context, id string, input UpdateAgentInput) error {
	var setClauses []string
	var args []interface{}
	argNum := 1

	if input.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argNum))
		args = append(args, string(*input.Status))
		argNum++

		// Set completed_at for terminal statuses
		if input.Status.IsTerminal() {
			setClauses = append(setClauses, fmt.Sprintf("completed_at = $%d", argNum))
			args = append(args, time.Now())
			argNum++
			// Clear PID when agent terminates
			setClauses = append(setClauses, fmt.Sprintf("pid = $%d", argNum))
			args = append(args, nil)
			argNum++
		}
	}
	if input.SessionID != nil {
		setClauses = append(setClauses, fmt.Sprintf("session_id = $%d", argNum))
		args = append(args, *input.SessionID)
		argNum++
	}
	if input.Output != nil {
		setClauses = append(setClauses, fmt.Sprintf("output = $%d", argNum))
		args = append(args, *input.Output)
		argNum++
	}
	if input.Error != nil {
		setClauses = append(setClauses, fmt.Sprintf("error = $%d", argNum))
		args = append(args, *input.Error)
		argNum++
	}
	if input.PID != nil {
		setClauses = append(setClauses, fmt.Sprintf("pid = $%d", argNum))
		args = append(args, *input.PID)
		argNum++
	}
	if input.Hostname != nil {
		setClauses = append(setClauses, fmt.Sprintf("hostname = $%d", argNum))
		args = append(args, *input.Hostname)
		argNum++
	}

	if len(setClauses) == 0 {
		return nil // Nothing to update
	}

	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argNum))
	args = append(args, time.Now())
	argNum++

	args = append(args, id)
	q := fmt.Sprintf("UPDATE spawned_agents SET %s WHERE id = $%d", strings.Join(setClauses, ", "), argNum)

	res, err := r.db.ExecContext(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("update agent %s: %w", id, err)
	}
	affected, err := res.RowsAffected()
	if err == nil && affected == 0 {
		return sql.ErrNoRows
	}
	return err
}

// List returns agents matching the given options.
func (r *PostgresAgentRepository) List(ctx context.Context, opts AgentListOptions) ([]*SpawnedAgent, error) {
	var whereClauses []string
	var args []interface{}
	argNum := 1

	if opts.Scenario != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("scenario = $%d", argNum))
		args = append(args, opts.Scenario)
		argNum++
	}

	if opts.ActiveOnly {
		whereClauses = append(whereClauses, fmt.Sprintf("status IN ($%d, $%d)", argNum, argNum+1))
		args = append(args, string(AgentStatusPending), string(AgentStatusRunning))
		argNum += 2
	} else if len(opts.StatusIn) > 0 {
		placeholders := make([]string, len(opts.StatusIn))
		for i, s := range opts.StatusIn {
			placeholders[i] = fmt.Sprintf("$%d", argNum)
			args = append(args, string(s))
			argNum++
		}
		whereClauses = append(whereClauses, fmt.Sprintf("status IN (%s)", strings.Join(placeholders, ", ")))
	}

	if opts.OlderThan != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("completed_at < $%d", argNum))
		args = append(args, *opts.OlderThan)
		argNum++
	}

	selectCols := `id, session_id, scenario, scope, phases, model, status,
		prompt_hash, prompt_index, output, error,
		started_at, completed_at, created_at, updated_at, pid, hostname`
	if opts.IncludeText {
		selectCols = `id, session_id, scenario, scope, phases, model, status,
			prompt_hash, prompt_index, prompt_text, output, error,
			started_at, completed_at, created_at, updated_at, pid, hostname`
	}

	q := fmt.Sprintf("SELECT %s FROM spawned_agents", selectCols)
	if len(whereClauses) > 0 {
		q += " WHERE " + strings.Join(whereClauses, " AND ")
	}
	q += " ORDER BY started_at DESC"

	if opts.Limit > 0 {
		q += fmt.Sprintf(" LIMIT $%d", argNum)
		args = append(args, opts.Limit)
	}

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()

	var agents []*SpawnedAgent
	for rows.Next() {
		agent, err := scanAgent(rows, opts.IncludeText)
		if err != nil {
			return nil, fmt.Errorf("scan agent: %w", err)
		}
		agents = append(agents, agent)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate agents: %w", err)
	}
	return agents, nil
}

// Delete removes an agent and its scope locks (cascades via FK).
func (r *PostgresAgentRepository) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM spawned_agents WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("delete agent %s: %w", id, err)
	}
	return nil
}

// DeleteOlderThan removes completed agents older than the given time.
func (r *PostgresAgentRepository) DeleteOlderThan(ctx context.Context, completedBefore time.Time) (int64, error) {
	const q = `
DELETE FROM spawned_agents
WHERE completed_at IS NOT NULL AND completed_at < $1
`
	res, err := r.db.ExecContext(ctx, q, completedBefore)
	if err != nil {
		return 0, fmt.Errorf("delete old agents: %w", err)
	}
	return res.RowsAffected()
}

// AcquireLocks creates scope locks for an agent.
func (r *PostgresAgentRepository) AcquireLocks(ctx context.Context, agentID, scenario string, paths []string, expiresAt time.Time) error {
	if len(paths) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	const q = `
INSERT INTO agent_scope_locks (agent_id, scenario, path, expires_at)
VALUES ($1, $2, $3, $4)
`
	for _, path := range paths {
		if _, err := tx.ExecContext(ctx, q, agentID, scenario, path, expiresAt); err != nil {
			return fmt.Errorf("acquire lock for path %s: %w", path, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit locks: %w", err)
	}
	return nil
}

// ReleaseLocks removes all locks for an agent.
func (r *PostgresAgentRepository) ReleaseLocks(ctx context.Context, agentID string) error {
	const q = `DELETE FROM agent_scope_locks WHERE agent_id = $1`
	_, err := r.db.ExecContext(ctx, q, agentID)
	if err != nil {
		return fmt.Errorf("release locks for agent %s: %w", agentID, err)
	}
	return nil
}

// RenewLocks extends the expiration of locks for an agent.
func (r *PostgresAgentRepository) RenewLocks(ctx context.Context, agentID string, newExpiry time.Time) error {
	const q = `
UPDATE agent_scope_locks
SET expires_at = $1, renewed_at = NOW()
WHERE agent_id = $2
`
	_, err := r.db.ExecContext(ctx, q, newExpiry, agentID)
	if err != nil {
		return fmt.Errorf("renew locks for agent %s: %w", agentID, err)
	}
	return nil
}

// GetActiveLocks returns all non-expired locks.
func (r *PostgresAgentRepository) GetActiveLocks(ctx context.Context) ([]*AgentScopeLock, error) {
	const q = `
SELECT id, agent_id, scenario, path, acquired_at, expires_at, renewed_at
FROM agent_scope_locks
WHERE expires_at > NOW()
ORDER BY acquired_at DESC
`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("get active locks: %w", err)
	}
	defer rows.Close()

	var locks []*AgentScopeLock
	for rows.Next() {
		lock := &AgentScopeLock{}
		if err := rows.Scan(
			&lock.ID, &lock.AgentID, &lock.Scenario, &lock.Path,
			&lock.AcquiredAt, &lock.ExpiresAt, &lock.RenewedAt,
		); err != nil {
			return nil, fmt.Errorf("scan lock: %w", err)
		}
		locks = append(locks, lock)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate locks: %w", err)
	}
	return locks, nil
}

// GetLocksForScenario returns non-expired locks for a specific scenario.
func (r *PostgresAgentRepository) GetLocksForScenario(ctx context.Context, scenario string) ([]*AgentScopeLock, error) {
	const q = `
SELECT id, agent_id, scenario, path, acquired_at, expires_at, renewed_at
FROM agent_scope_locks
WHERE scenario = $1 AND expires_at > NOW()
ORDER BY acquired_at DESC
`
	rows, err := r.db.QueryContext(ctx, q, scenario)
	if err != nil {
		return nil, fmt.Errorf("get locks for scenario %s: %w", scenario, err)
	}
	defer rows.Close()

	var locks []*AgentScopeLock
	for rows.Next() {
		lock := &AgentScopeLock{}
		if err := rows.Scan(
			&lock.ID, &lock.AgentID, &lock.Scenario, &lock.Path,
			&lock.AcquiredAt, &lock.ExpiresAt, &lock.RenewedAt,
		); err != nil {
			return nil, fmt.Errorf("scan lock: %w", err)
		}
		locks = append(locks, lock)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate locks: %w", err)
	}
	return locks, nil
}

// CheckConflicts returns lock details for paths that overlap with the given scope.
func (r *PostgresAgentRepository) CheckConflicts(ctx context.Context, scenario string, paths []string) ([]ConflictDetail, error) {
	if len(paths) == 0 {
		return nil, nil
	}

	// Get all active locks for this scenario
	locks, err := r.GetLocksForScenario(ctx, scenario)
	if err != nil {
		return nil, err
	}

	if len(locks) == 0 {
		return nil, nil
	}

	// Get agent info for conflict details
	agentMap := make(map[string]*SpawnedAgent)
	for _, lock := range locks {
		if _, ok := agentMap[lock.AgentID]; !ok {
			agent, err := r.Get(ctx, lock.AgentID)
			if err != nil {
				return nil, fmt.Errorf("get agent %s: %w", lock.AgentID, err)
			}
			if agent != nil {
				agentMap[lock.AgentID] = agent
			}
		}
	}

	// Check for overlapping paths
	var conflicts []ConflictDetail
	for _, requestedPath := range paths {
		for _, lock := range locks {
			if pathsOverlap(requestedPath, lock.Path) {
				agent := agentMap[lock.AgentID]
				var startedAt time.Time
				if agent != nil {
					startedAt = agent.StartedAt
				}
				conflicts = append(conflicts, ConflictDetail{
					Path: requestedPath,
					LockedBy: ScopeLockInfo{
						Path:      lock.Path,
						AgentID:   lock.AgentID,
						Scenario:  lock.Scenario,
						StartedAt: startedAt,
						ExpiresAt: lock.ExpiresAt,
					},
				})
			}
		}
	}

	return conflicts, nil
}

// pathsOverlap checks if two paths overlap (one is a prefix of the other or they're equal).
func pathsOverlap(a, b string) bool {
	a = strings.TrimSuffix(a, "/")
	b = strings.TrimSuffix(b, "/")

	if a == b {
		return true
	}
	if strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/") {
		return true
	}
	return false
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanAgent(scanner rowScanner, includeText bool) (*SpawnedAgent, error) {
	agent := &SpawnedAgent{}
	var sessionID, output, errStr, promptText, hostname sql.NullString
	var completedAt sql.NullTime
	var scope, phases pq.StringArray
	var status string
	var pid sql.NullInt32

	var err error
	if includeText {
		err = scanner.Scan(
			&agent.ID, &sessionID, &agent.Scenario, &scope, &phases, &agent.Model, &status,
			&agent.PromptHash, &agent.PromptIndex, &promptText, &output, &errStr,
			&agent.StartedAt, &completedAt, &agent.CreatedAt, &agent.UpdatedAt,
			&pid, &hostname,
		)
	} else {
		err = scanner.Scan(
			&agent.ID, &sessionID, &agent.Scenario, &scope, &phases, &agent.Model, &status,
			&agent.PromptHash, &agent.PromptIndex, &output, &errStr,
			&agent.StartedAt, &completedAt, &agent.CreatedAt, &agent.UpdatedAt,
			&pid, &hostname,
		)
	}
	if err != nil {
		return nil, err
	}

	agent.Status = AgentStatus(status)
	agent.Scope = append([]string(nil), scope...)
	agent.Phases = append([]string(nil), phases...)

	if sessionID.Valid {
		agent.SessionID = sessionID.String
	}
	if output.Valid {
		agent.Output = output.String
	}
	if errStr.Valid {
		agent.Error = errStr.String
	}
	if promptText.Valid {
		agent.PromptText = promptText.String
	}
	if completedAt.Valid {
		agent.CompletedAt = &completedAt.Time
	}
	if pid.Valid {
		agent.PID = int(pid.Int32)
	}
	if hostname.Valid {
		agent.Hostname = hostname.String
	}

	return agent, nil
}

// scanAgentWithIdempKey scans an agent row that includes the idempotency_key column.
func scanAgentWithIdempKey(scanner rowScanner) (*SpawnedAgent, error) {
	agent := &SpawnedAgent{}
	var idempKey, sessionID, output, errStr, promptText sql.NullString
	var completedAt sql.NullTime
	var scope, phases pq.StringArray
	var status string

	err := scanner.Scan(
		&agent.ID, &idempKey, &sessionID, &agent.Scenario, &scope, &phases, &agent.Model, &status,
		&agent.PromptHash, &agent.PromptIndex, &promptText, &output, &errStr,
		&agent.StartedAt, &completedAt, &agent.CreatedAt, &agent.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	agent.Status = AgentStatus(status)
	agent.Scope = append([]string(nil), scope...)
	agent.Phases = append([]string(nil), phases...)

	if idempKey.Valid {
		agent.IdempotencyKey = idempKey.String
	}
	if sessionID.Valid {
		agent.SessionID = sessionID.String
	}
	if output.Valid {
		agent.Output = output.String
	}
	if errStr.Valid {
		agent.Error = errStr.String
	}
	if promptText.Valid {
		agent.PromptText = promptText.String
	}
	if completedAt.Valid {
		agent.CompletedAt = &completedAt.Time
	}

	return agent, nil
}

// AcquireSpawnIntent atomically acquires a spawn intent lock.
// Returns (intent, isNew, error). If isNew is false, the intent already existed.
func (r *PostgresAgentRepository) AcquireSpawnIntent(ctx context.Context, key, scenario string, scope []string, ttl time.Duration) (*SpawnIntent, bool, error) {
	if scope == nil {
		scope = []string{}
	}

	// Try to insert; on conflict, return the existing row
	const q = `
INSERT INTO spawn_intents (key, scenario, scope, status, expires_at)
VALUES ($1, $2, $3, 'pending', $4)
ON CONFLICT (key) DO UPDATE SET key = spawn_intents.key
RETURNING key, scenario, scope, agent_id, status, result_json, created_at, expires_at,
          (xmax = 0) AS is_new
`
	expiresAt := time.Now().Add(ttl)

	intent := &SpawnIntent{}
	var agentID, resultJSON sql.NullString
	var scopeArr pq.StringArray
	var isNew bool

	err := r.db.QueryRowContext(ctx, q, key, scenario, pq.Array(scope), expiresAt).Scan(
		&intent.Key, &intent.Scenario, &scopeArr, &agentID, &intent.Status, &resultJSON,
		&intent.CreatedAt, &intent.ExpiresAt, &isNew,
	)
	if err != nil {
		return nil, false, fmt.Errorf("acquire spawn intent: %w", err)
	}

	intent.Scope = append([]string(nil), scopeArr...)
	if agentID.Valid {
		intent.AgentID = agentID.String
	}
	if resultJSON.Valid {
		intent.ResultJSON = resultJSON.String
	}

	return intent, isNew, nil
}

// UpdateSpawnIntent updates an existing spawn intent with result info.
func (r *PostgresAgentRepository) UpdateSpawnIntent(ctx context.Context, key string, agentID, status, resultJSON string) error {
	const q = `
UPDATE spawn_intents
SET agent_id = $2, status = $3, result_json = $4
WHERE key = $1
`
	var agentIDVal, resultJSONVal interface{}
	if agentID != "" {
		agentIDVal = agentID
	}
	if resultJSON != "" {
		resultJSONVal = resultJSON
	}

	_, err := r.db.ExecContext(ctx, q, key, agentIDVal, status, resultJSONVal)
	if err != nil {
		return fmt.Errorf("update spawn intent %s: %w", key, err)
	}
	return nil
}

// GetSpawnIntent retrieves a spawn intent by key.
func (r *PostgresAgentRepository) GetSpawnIntent(ctx context.Context, key string) (*SpawnIntent, error) {
	const q = `
SELECT key, scenario, scope, agent_id, status, result_json, created_at, expires_at
FROM spawn_intents
WHERE key = $1
`
	intent := &SpawnIntent{}
	var agentID, resultJSON sql.NullString
	var scopeArr pq.StringArray

	err := r.db.QueryRowContext(ctx, q, key).Scan(
		&intent.Key, &intent.Scenario, &scopeArr, &agentID, &intent.Status, &resultJSON,
		&intent.CreatedAt, &intent.ExpiresAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get spawn intent %s: %w", key, err)
	}

	intent.Scope = append([]string(nil), scopeArr...)
	if agentID.Valid {
		intent.AgentID = agentID.String
	}
	if resultJSON.Valid {
		intent.ResultJSON = resultJSON.String
	}

	return intent, nil
}

// CleanupSpawnIntents removes expired spawn intents.
func (r *PostgresAgentRepository) CleanupSpawnIntents(ctx context.Context) (int64, error) {
	const q = `DELETE FROM spawn_intents WHERE expires_at < NOW()`
	res, err := r.db.ExecContext(ctx, q)
	if err != nil {
		return 0, fmt.Errorf("cleanup spawn intents: %w", err)
	}
	return res.RowsAffected()
}

// RecordFileOperation logs a file operation by an agent for auditing.
func (r *PostgresAgentRepository) RecordFileOperation(ctx context.Context, input FileOperationInput) error {
	const q = `
INSERT INTO agent_file_operations (
	agent_id, scenario, operation, file_path, content_hash, content_before, content_after
) VALUES (
	$1, $2, $3, $4, $5, $6, $7
)
`
	var contentHash, contentBefore, contentAfter interface{}
	if input.ContentHash != "" {
		contentHash = input.ContentHash
	}
	if input.ContentBefore != "" {
		contentBefore = input.ContentBefore
	}
	if input.ContentAfter != "" {
		contentAfter = input.ContentAfter
	}

	_, err := r.db.ExecContext(ctx, q,
		input.AgentID,
		input.Scenario,
		input.Operation,
		input.FilePath,
		contentHash,
		contentBefore,
		contentAfter,
	)
	if err != nil {
		return fmt.Errorf("record file operation: %w", err)
	}
	return nil
}

// GetFileOperations returns file operations for an agent.
func (r *PostgresAgentRepository) GetFileOperations(ctx context.Context, agentID string) ([]*FileOperation, error) {
	const q = `
SELECT id, agent_id, scenario, operation, file_path, content_hash, content_before, content_after, recorded_at
FROM agent_file_operations
WHERE agent_id = $1
ORDER BY recorded_at DESC
`
	rows, err := r.db.QueryContext(ctx, q, agentID)
	if err != nil {
		return nil, fmt.Errorf("get file operations for agent %s: %w", agentID, err)
	}
	defer rows.Close()

	var ops []*FileOperation
	for rows.Next() {
		op := &FileOperation{}
		var contentHash, contentBefore, contentAfter sql.NullString

		if err := rows.Scan(
			&op.ID, &op.AgentID, &op.Scenario, &op.Operation, &op.FilePath,
			&contentHash, &contentBefore, &contentAfter, &op.RecordedAt,
		); err != nil {
			return nil, fmt.Errorf("scan file operation: %w", err)
		}

		if contentHash.Valid {
			op.ContentHash = contentHash.String
		}
		if contentBefore.Valid {
			op.ContentBefore = contentBefore.String
		}
		if contentAfter.Valid {
			op.ContentAfter = contentAfter.String
		}

		ops = append(ops, op)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate file operations: %w", err)
	}

	return ops, nil
}

// GetFileOperationsForScenario returns file operations for a scenario.
func (r *PostgresAgentRepository) GetFileOperationsForScenario(ctx context.Context, scenario string, limit int) ([]*FileOperation, error) {
	q := `
SELECT id, agent_id, scenario, operation, file_path, content_hash, content_before, content_after, recorded_at
FROM agent_file_operations
WHERE scenario = $1
ORDER BY recorded_at DESC
`
	args := []interface{}{scenario}
	if limit > 0 {
		q += " LIMIT $2"
		args = append(args, limit)
	}

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("get file operations for scenario %s: %w", scenario, err)
	}
	defer rows.Close()

	var ops []*FileOperation
	for rows.Next() {
		op := &FileOperation{}
		var contentHash, contentBefore, contentAfter sql.NullString

		if err := rows.Scan(
			&op.ID, &op.AgentID, &op.Scenario, &op.Operation, &op.FilePath,
			&contentHash, &contentBefore, &contentAfter, &op.RecordedAt,
		); err != nil {
			return nil, fmt.Errorf("scan file operation: %w", err)
		}

		if contentHash.Valid {
			op.ContentHash = contentHash.String
		}
		if contentBefore.Valid {
			op.ContentBefore = contentBefore.String
		}
		if contentAfter.Valid {
			op.ContentAfter = contentAfter.String
		}

		ops = append(ops, op)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate file operations for scenario: %w", err)
	}

	return ops, nil
}

// CreateSpawnSession creates a new server-side spawn session.
func (r *PostgresAgentRepository) CreateSpawnSession(ctx context.Context, input CreateSpawnSessionInput) (*SpawnSession, error) {
	if input.Scope == nil {
		input.Scope = []string{}
	}
	if input.Phases == nil {
		input.Phases = []string{}
	}
	if input.AgentIDs == nil {
		input.AgentIDs = []string{}
	}
	if input.TTL <= 0 {
		input.TTL = 30 * time.Minute // Default 30 minute TTL
	}

	expiresAt := time.Now().Add(input.TTL)

	const q = `
INSERT INTO spawn_sessions (user_identifier, scenario, scope, phases, agent_ids, status, expires_at)
VALUES ($1, $2, $3, $4, $5, 'active', $6)
RETURNING id, created_at, last_activity_at
`
	session := &SpawnSession{
		UserIdentifier: input.UserIdentifier,
		Scenario:       input.Scenario,
		Scope:          input.Scope,
		Phases:         input.Phases,
		AgentIDs:       input.AgentIDs,
		Status:         "active",
		ExpiresAt:      expiresAt,
	}

	err := r.db.QueryRowContext(ctx, q,
		session.UserIdentifier,
		session.Scenario,
		pq.Array(session.Scope),
		pq.Array(session.Phases),
		pq.Array(session.AgentIDs),
		session.ExpiresAt,
	).Scan(&session.ID, &session.CreatedAt, &session.LastActivityAt)

	if err != nil {
		return nil, fmt.Errorf("create spawn session: %w", err)
	}
	return session, nil
}

// GetActiveSpawnSessions returns active spawn sessions for a user/scenario.
func (r *PostgresAgentRepository) GetActiveSpawnSessions(ctx context.Context, userIdentifier, scenario string) ([]*SpawnSession, error) {
	q := `
SELECT id, user_identifier, scenario, scope, phases, agent_ids, status, created_at, expires_at, last_activity_at
FROM spawn_sessions
WHERE user_identifier = $1 AND status = 'active' AND expires_at > NOW()
`
	args := []interface{}{userIdentifier}
	if scenario != "" {
		q += " AND scenario = $2"
		args = append(args, scenario)
	}
	q += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("get active spawn sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*SpawnSession
	for rows.Next() {
		session := &SpawnSession{}
		var scope, phases, agentIDs pq.StringArray

		if err := rows.Scan(
			&session.ID, &session.UserIdentifier, &session.Scenario,
			&scope, &phases, &agentIDs,
			&session.Status, &session.CreatedAt, &session.ExpiresAt, &session.LastActivityAt,
		); err != nil {
			return nil, fmt.Errorf("scan spawn session: %w", err)
		}

		session.Scope = append([]string(nil), scope...)
		session.Phases = append([]string(nil), phases...)
		session.AgentIDs = append([]string(nil), agentIDs...)
		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate spawn sessions: %w", err)
	}
	return sessions, nil
}

// CheckSpawnSessionConflicts checks for conflicting active spawn sessions.
func (r *PostgresAgentRepository) CheckSpawnSessionConflicts(ctx context.Context, userIdentifier, scenario string, scope []string) ([]SpawnSessionConflict, error) {
	sessions, err := r.GetActiveSpawnSessions(ctx, userIdentifier, scenario)
	if err != nil {
		return nil, err
	}

	var conflicts []SpawnSessionConflict
	for _, session := range sessions {
		// Check if scopes overlap
		if scopesOverlap(session.Scope, scope) {
			conflicts = append(conflicts, SpawnSessionConflict{
				SessionID: session.ID,
				Scenario:  session.Scenario,
				Scope:     session.Scope,
				AgentIDs:  session.AgentIDs,
				CreatedAt: session.CreatedAt,
				ExpiresAt: session.ExpiresAt,
			})
		}
	}
	return conflicts, nil
}

// scopesOverlap checks if two scope arrays have any overlapping paths.
func scopesOverlap(a, b []string) bool {
	// Empty scope means "entire scenario" - conflicts with everything
	if len(a) == 0 || len(b) == 0 {
		return true
	}

	for _, pathA := range a {
		for _, pathB := range b {
			if pathsOverlap(pathA, pathB) {
				return true
			}
		}
	}
	return false
}

// UpdateSpawnSessionStatus updates the status of a spawn session.
func (r *PostgresAgentRepository) UpdateSpawnSessionStatus(ctx context.Context, sessionID int, status string) error {
	const q = `
UPDATE spawn_sessions
SET status = $2, last_activity_at = NOW()
WHERE id = $1
`
	_, err := r.db.ExecContext(ctx, q, sessionID, status)
	if err != nil {
		return fmt.Errorf("update spawn session %d: %w", sessionID, err)
	}
	return nil
}

// ClearSpawnSessions marks all active spawn sessions for a user as cleared.
func (r *PostgresAgentRepository) ClearSpawnSessions(ctx context.Context, userIdentifier string) (int64, error) {
	const q = `
UPDATE spawn_sessions
SET status = 'cleared', last_activity_at = NOW()
WHERE user_identifier = $1 AND status = 'active'
`
	res, err := r.db.ExecContext(ctx, q, userIdentifier)
	if err != nil {
		return 0, fmt.Errorf("clear spawn sessions: %w", err)
	}
	return res.RowsAffected()
}

// CleanupExpiredSpawnSessions removes expired spawn sessions.
func (r *PostgresAgentRepository) CleanupExpiredSpawnSessions(ctx context.Context) (int64, error) {
	const q = `DELETE FROM spawn_sessions WHERE expires_at < NOW()`
	res, err := r.db.ExecContext(ctx, q)
	if err != nil {
		return 0, fmt.Errorf("cleanup expired spawn sessions: %w", err)
	}
	return res.RowsAffected()
}
