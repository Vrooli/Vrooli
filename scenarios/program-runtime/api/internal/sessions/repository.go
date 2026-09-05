package sessions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

// SQLExecutor is the narrow database seam used by the sessions repository.
// Both *sql.DB and the routed test/production handles can satisfy it.
type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Repository interface {
	Create(context.Context, *Session) error
	Get(context.Context, string) (*Session, error)
	List(context.Context) ([]*Session, error)
	Delete(context.Context, string, string, time.Time) (*Session, error)
	Grant(context.Context, string, []string, time.Time) (*Session, error)
	HasGrant(context.Context, string, string) (bool, error)
	Touch(context.Context, string, time.Time) error
	SetMemoryBytes(context.Context, string, int64) error
	RecordInferenceUsage(context.Context, string, int64, int64) error
	RecordDelegationUsage(context.Context, string, int64, bool, string) error
	SaveDelegation(context.Context, *Delegation) error
	GetDelegation(context.Context, string, string) (*Delegation, error)
	ListDelegations(context.Context) ([]*Delegation, error)
	CountDelegations(context.Context) (int, error)
	RecordExecutionUsage(context.Context, string, time.Duration, time.Duration, time.Time) error
	Reclaim(context.Context, string, string, time.Time) error
}

type Delegation struct {
	SessionID   string
	ExecutionID string
	Owner       string
	WorkflowKey string
	CreatedAt   time.Time
	LastStatus  string
}

var (
	ErrDelegationNotFound = errors.New("delegation not found")
	ErrDelegationNotOwned = errors.New("delegation does not belong to session")
)

type sqliteRepository struct{ db SQLExecutor }

func NewRepository(db SQLExecutor) Repository { return &sqliteRepository{db: db} }

func (r *sqliteRepository) Create(ctx context.Context, s *Session) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO sessions
	 (id, name, state, created_at, last_activity_at, sandbox_workspace, memory_bytes, reclaimed_reason, inference_cost_micros, inference_tokens, delegation_cost_micros, inference_ceiling_micros, delegation_ceiling_micros, delegation_spend_measured, delegation_spend_note, wall_budget_millis, wall_consumed_millis, cpu_budget_millis, cpu_consumed_millis)
	 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, s.ID, s.Name, s.State, formatTime(s.CreatedAt), formatTime(s.LastActivityAt), s.SandboxWorkspace, s.MemoryBytes, s.ReclaimedReason, s.InferenceCostMicros, s.InferenceTokens, s.DelegationCostMicros, s.InferenceCeilingMicros, s.DelegationCeilingMicros, boolInt(s.DelegationSpendMeasured), s.DelegationSpendNote, durationMillis(s.WallBudget), durationMillis(s.WallConsumed), durationMillis(s.CPUBudget), durationMillis(s.CPUConsumed))
	if err != nil {
		return fmt.Errorf("create session %q: %w", s.ID, err)
	}
	for grant := range s.Grants {
		if _, err := r.db.ExecContext(ctx, `INSERT INTO session_grants (session_id, grant_name) VALUES (?, ?)`, s.ID, grant); err != nil {
			return fmt.Errorf("create session grant %q: %w", grant, err)
		}
	}
	return nil
}

func (r *sqliteRepository) Get(ctx context.Context, id string) (*Session, error) {
	s, err := r.scanSession(r.db.QueryRowContext(ctx, `SELECT id, name, state, created_at, last_activity_at, sandbox_workspace, memory_bytes, reclaimed_reason, inference_cost_micros, inference_tokens, delegation_cost_micros, inference_ceiling_micros, delegation_ceiling_micros, delegation_spend_measured, delegation_spend_note, wall_budget_millis, wall_consumed_millis, cpu_budget_millis, cpu_consumed_millis FROM sessions WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get session %q: %w", id, err)
	}
	if err := r.loadGrants(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

func (r *sqliteRepository) List(ctx context.Context) ([]*Session, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, state, created_at, last_activity_at, sandbox_workspace, memory_bytes, reclaimed_reason, inference_cost_micros, inference_tokens, delegation_cost_micros, inference_ceiling_micros, delegation_ceiling_micros, delegation_spend_measured, delegation_spend_note, wall_budget_millis, wall_consumed_millis, cpu_budget_millis, cpu_consumed_millis FROM sessions ORDER BY created_at, id`)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()
	var out []*Session
	for rows.Next() {
		s, err := r.scanSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sessions: %w", err)
	}
	// The production SQLite pool is intentionally single-connection. Finish
	// the parent rows cursor before loading child grants, otherwise a nested
	// QueryContext waits forever for the only connection held by rows.
	for _, s := range out {
		if err := r.loadGrants(ctx, s); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (r *sqliteRepository) Delete(ctx context.Context, id, reason string, now time.Time) (*Session, error) {
	s, err := r.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if reason == "" {
		reason = "deleted"
	}
	if err := r.Reclaim(ctx, id, reason, now); err != nil {
		return nil, err
	}
	s.State = "reclaimed"
	s.ReclaimedReason = reason
	return s, nil
}

func (r *sqliteRepository) Grant(ctx context.Context, id string, grants []string, now time.Time) (*Session, error) {
	if _, err := r.Get(ctx, id); err != nil {
		return nil, err
	}
	for _, grant := range grants {
		if grant == "" {
			continue
		}
		if _, err := r.db.ExecContext(ctx, `INSERT INTO session_grants (session_id, grant_name) VALUES (?, ?) ON CONFLICT(session_id, grant_name) DO NOTHING`, id, grant); err != nil {
			return nil, fmt.Errorf("grant session %q: %w", id, err)
		}
	}
	if err := r.Touch(ctx, id, now); err != nil {
		return nil, err
	}
	return r.Get(ctx, id)
}

func (r *sqliteRepository) HasGrant(ctx context.Context, id, grant string) (bool, error) {
	var found int
	err := r.db.QueryRowContext(ctx, `SELECT 1 FROM session_grants WHERE session_id = ? AND grant_name = ?`, id, grant).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check session grant: %w", err)
	}
	return found == 1, nil
}

func (r *sqliteRepository) Touch(ctx context.Context, id string, now time.Time) error {
	result, err := r.db.ExecContext(ctx, `UPDATE sessions SET last_activity_at = ? WHERE id = ?`, formatTime(now), id)
	if err != nil {
		return fmt.Errorf("touch session %q: %w", id, err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) SetMemoryBytes(ctx context.Context, id string, bytes int64) error {
	result, err := r.db.ExecContext(ctx, `UPDATE sessions SET memory_bytes = ? WHERE id = ?`, bytes, id)
	if err != nil {
		return fmt.Errorf("set session memory %q: %w", id, err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) RecordInferenceUsage(ctx context.Context, id string, costMicros, tokens int64) error {
	result, err := r.db.ExecContext(ctx, `UPDATE sessions SET inference_cost_micros = inference_cost_micros + ?, inference_tokens = inference_tokens + ?, last_activity_at = ? WHERE id = ? AND (inference_ceiling_micros = 0 OR inference_cost_micros < inference_ceiling_micros)`, costMicros, tokens, formatTime(time.Now().UTC()), id)
	if err != nil {
		return fmt.Errorf("record inference usage for %q: %w", id, err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		s, getErr := r.Get(ctx, id)
		if getErr != nil {
			return getErr
		}
		return &SpendExceededError{Kind: "inference", Ceiling: s.InferenceCeilingMicros, Accumulated: s.InferenceCostMicros}
	}
	return nil
}

func (r *sqliteRepository) RecordDelegationUsage(ctx context.Context, id string, costMicros int64, measured bool, note string) error {
	result, err := r.db.ExecContext(ctx, `UPDATE sessions SET delegation_cost_micros = delegation_cost_micros + ?, delegation_spend_measured = ?, delegation_spend_note = ?, last_activity_at = ? WHERE id = ? AND (? = 0 OR delegation_ceiling_micros = 0 OR delegation_cost_micros + ? <= delegation_ceiling_micros)`, costMicros, boolInt(measured), note, formatTime(time.Now().UTC()), id, boolInt(measured), costMicros)
	if err != nil {
		return fmt.Errorf("record delegation usage for %q: %w", id, err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		s, getErr := r.Get(ctx, id)
		if getErr != nil {
			return getErr
		}
		return &SpendExceededError{Kind: "delegated_run", Ceiling: s.DelegationCeilingMicros, Accumulated: s.DelegationCostMicros}
	}
	return nil
}

func (r *sqliteRepository) SaveDelegation(ctx context.Context, delegation *Delegation) error {
	if delegation == nil || delegation.SessionID == "" || delegation.ExecutionID == "" {
		return errors.New("session and execution identifiers are required")
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO session_delegations (session_id, execution_id, owner, workflow_key, created_at, last_status) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(execution_id) DO UPDATE SET last_status=excluded.last_status`, delegation.SessionID, delegation.ExecutionID, delegation.Owner, delegation.WorkflowKey, formatTime(delegation.CreatedAt), delegation.LastStatus)
	return err
}

func (r *sqliteRepository) GetDelegation(ctx context.Context, sessionID, executionID string) (*Delegation, error) {
	var d Delegation
	var created string
	err := r.db.QueryRowContext(ctx, `SELECT session_id, execution_id, owner, workflow_key, created_at, last_status FROM session_delegations WHERE execution_id = ?`, executionID).Scan(&d.SessionID, &d.ExecutionID, &d.Owner, &d.WorkflowKey, &created, &d.LastStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrDelegationNotFound
	}
	if err != nil {
		return nil, err
	}
	d.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return nil, err
	}
	if d.SessionID != sessionID {
		return nil, ErrDelegationNotOwned
	}
	return &d, nil
}

func (r *sqliteRepository) CountDelegations(ctx context.Context) (int, error) {
	var count int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM session_delegations`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count session delegations: %w", err)
	}
	return count, nil
}

func (r *sqliteRepository) ListDelegations(ctx context.Context) ([]*Delegation, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT session_id, execution_id, owner, workflow_key, created_at, last_status FROM session_delegations ORDER BY created_at, execution_id`)
	if err != nil {
		return nil, fmt.Errorf("list session delegations: %w", err)
	}
	defer rows.Close()
	var out []*Delegation
	for rows.Next() {
		var d Delegation
		var created string
		if err := rows.Scan(&d.SessionID, &d.ExecutionID, &d.Owner, &d.WorkflowKey, &created, &d.LastStatus); err != nil {
			return nil, fmt.Errorf("scan session delegation: %w", err)
		}
		parsed, err := time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return nil, fmt.Errorf("parse session delegation timestamp: %w", err)
		}
		d.CreatedAt = parsed
		out = append(out, &d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate session delegations: %w", err)
	}
	return out, nil
}

func (r *sqliteRepository) RecordExecutionUsage(ctx context.Context, id string, wall, cpu time.Duration, now time.Time) error {
	result, err := r.db.ExecContext(ctx, `UPDATE sessions SET wall_consumed_millis = wall_consumed_millis + ?, cpu_consumed_millis = cpu_consumed_millis + ?, last_activity_at = ? WHERE id = ?`, durationMillis(wall), durationMillis(cpu), formatTime(now), id)
	if err != nil {
		return fmt.Errorf("record execution usage for %q: %w", id, err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) Reclaim(ctx context.Context, id, reason string, now time.Time) error {
	if _, err := r.Get(ctx, id); err != nil {
		return err
	}
	if _, err := r.db.ExecContext(ctx, `INSERT INTO reclamation_reasons (session_id, reason, reclaimed_at) VALUES (?, ?, ?)`, id, reason, formatTime(now)); err != nil {
		return fmt.Errorf("record reclamation for %q: %w", id, err)
	}
	result, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete reclaimed session %q: %w", id, err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return ErrNotFound
	}
	return nil
}

type rowScanner interface{ Scan(...any) error }

func (r *sqliteRepository) scanSession(row rowScanner) (*Session, error) {
	var s Session
	var created, activity string
	var measured int64
	var wallBudget, wallConsumed, cpuBudget, cpuConsumed int64
	if err := row.Scan(&s.ID, &s.Name, &s.State, &created, &activity, &s.SandboxWorkspace, &s.MemoryBytes, &s.ReclaimedReason, &s.InferenceCostMicros, &s.InferenceTokens, &s.DelegationCostMicros, &s.InferenceCeilingMicros, &s.DelegationCeilingMicros, &measured, &s.DelegationSpendNote, &wallBudget, &wallConsumed, &cpuBudget, &cpuConsumed); err != nil {
		return nil, err
	}
	s.WallBudget = time.Duration(wallBudget) * time.Millisecond
	s.WallConsumed = time.Duration(wallConsumed) * time.Millisecond
	s.CPUBudget = time.Duration(cpuBudget) * time.Millisecond
	s.CPUConsumed = time.Duration(cpuConsumed) * time.Millisecond
	s.DelegationSpendMeasured = measured != 0
	var err error
	if s.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
		return nil, fmt.Errorf("parse session created_at: %w", err)
	}
	if s.LastActivityAt, err = time.Parse(time.RFC3339Nano, activity); err != nil {
		return nil, fmt.Errorf("parse session last_activity_at: %w", err)
	}
	s.Grants = make(map[string]struct{})
	return &s, nil
}

func (r *sqliteRepository) loadGrants(ctx context.Context, s *Session) error {
	rows, err := r.db.QueryContext(ctx, `SELECT grant_name FROM session_grants WHERE session_id = ? ORDER BY grant_name`, s.ID)
	if err != nil {
		return fmt.Errorf("list grants for %q: %w", s.ID, err)
	}
	defer rows.Close()
	for rows.Next() {
		var grant string
		if err := rows.Scan(&grant); err != nil {
			return fmt.Errorf("scan grant for %q: %w", s.ID, err)
		}
		s.Grants[grant] = struct{}{}
	}
	return rows.Err()
}

func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }

func durationMillis(value time.Duration) int64 { return value.Milliseconds() }

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

// memoryRepository keeps package-level tests independent from an on-disk
// database. Production always supplies a SQLite executor through Options.
type memoryRepository struct {
	mu          sync.RWMutex
	sessions    map[string]*Session
	reasons     []string
	delegations map[string]*Delegation
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{sessions: make(map[string]*Session), delegations: make(map[string]*Delegation)}
}

func (r *memoryRepository) Create(_ context.Context, s *Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[s.ID] = clone(s)
	return nil
}

func (r *memoryRepository) RecordInferenceUsage(_ context.Context, id string, costMicros, tokens int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[id]
	if !ok {
		return ErrNotFound
	}
	if s.InferenceCeilingMicros > 0 && s.InferenceCostMicros >= s.InferenceCeilingMicros {
		return &SpendExceededError{Kind: "inference", Ceiling: s.InferenceCeilingMicros, Accumulated: s.InferenceCostMicros}
	}
	s.InferenceCostMicros += costMicros
	s.InferenceTokens += tokens
	s.LastActivityAt = time.Now().UTC()
	return nil
}

func (r *memoryRepository) RecordDelegationUsage(_ context.Context, id string, costMicros int64, measured bool, note string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[id]
	if !ok {
		return ErrNotFound
	}
	if measured && s.DelegationCeilingMicros > 0 && s.DelegationCostMicros+costMicros > s.DelegationCeilingMicros {
		return &SpendExceededError{Kind: "delegated_run", Ceiling: s.DelegationCeilingMicros, Accumulated: s.DelegationCostMicros}
	}
	s.DelegationCostMicros += costMicros
	s.DelegationSpendMeasured = measured
	s.DelegationSpendNote = note
	s.LastActivityAt = time.Now().UTC()
	return nil
}

func (r *memoryRepository) SaveDelegation(_ context.Context, delegation *Delegation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.sessions[delegation.SessionID]; !ok {
		return ErrNotFound
	}
	r.delegations[delegation.ExecutionID] = cloneDelegation(delegation)
	return nil
}

func (r *memoryRepository) GetDelegation(_ context.Context, sessionID, executionID string) (*Delegation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	delegation, ok := r.delegations[executionID]
	if !ok {
		return nil, ErrDelegationNotFound
	}
	if delegation.SessionID != sessionID {
		return nil, ErrDelegationNotOwned
	}
	return cloneDelegation(delegation), nil
}

func (r *memoryRepository) CountDelegations(_ context.Context) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.delegations), nil
}

func (r *memoryRepository) ListDelegations(_ context.Context) ([]*Delegation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Delegation, 0, len(r.delegations))
	for _, delegation := range r.delegations {
		out = append(out, cloneDelegation(delegation))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ExecutionID < out[j].ExecutionID
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}

func cloneDelegation(delegation *Delegation) *Delegation {
	if delegation == nil {
		return nil
	}
	copy := *delegation
	return &copy
}

func (r *memoryRepository) RecordExecutionUsage(_ context.Context, id string, wall, cpu time.Duration, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[id]
	if !ok {
		return ErrNotFound
	}
	s.WallConsumed += wall
	s.CPUConsumed += cpu
	s.LastActivityAt = now
	return nil
}

func (r *memoryRepository) Get(_ context.Context, id string) (*Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.sessions[id]
	if !ok {
		return nil, ErrNotFound
	}
	return clone(s), nil
}

func (r *memoryRepository) List(_ context.Context) ([]*Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Session, 0, len(r.sessions))
	for _, s := range r.sessions {
		out = append(out, clone(s))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (r *memoryRepository) Delete(ctx context.Context, id, reason string, now time.Time) (*Session, error) {
	s, err := r.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if reason == "" {
		reason = "deleted"
	}
	if err := r.Reclaim(ctx, id, reason, now); err != nil {
		return nil, err
	}
	s.State = "reclaimed"
	s.ReclaimedReason = reason
	return s, nil
}

func (r *memoryRepository) Grant(ctx context.Context, id string, grants []string, now time.Time) (*Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[id]
	if !ok {
		return nil, ErrNotFound
	}
	for _, g := range grants {
		if g != "" {
			s.Grants[g] = struct{}{}
		}
	}
	s.LastActivityAt = now
	return clone(s), nil
}

func (r *memoryRepository) HasGrant(_ context.Context, id, grant string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.sessions[id]
	if !ok {
		return false, nil
	}
	_, ok = s.Grants[grant]
	return ok, nil
}

func (r *memoryRepository) Touch(_ context.Context, id string, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[id]
	if !ok {
		return ErrNotFound
	}
	s.LastActivityAt = now
	return nil
}

func (r *memoryRepository) SetMemoryBytes(_ context.Context, id string, b int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[id]
	if !ok {
		return ErrNotFound
	}
	s.MemoryBytes = b
	return nil
}

func (r *memoryRepository) Reclaim(_ context.Context, id, reason string, _ time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.sessions[id]; !ok {
		return ErrNotFound
	}
	delete(r.sessions, id)
	r.reasons = append(r.reasons, id+":"+reason)
	return nil
}
