package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type RunScopeRepository interface {
	CreateRunScope(context.Context, RunScope) error
	GetRunScope(context.Context, string) (RunScope, error)
	CloseRunScope(context.Context, string, int64, time.Time) (RunScope, []RunScopeChild, error)
	ListRunScopeChildren(context.Context, string) ([]RunScopeChild, error)
	FinalizeRunScope(context.Context, string, RunScopeState, int64, time.Time) (RunScope, error)
}

type RunScopeService interface {
	Create(context.Context, RunScope) error
	Get(context.Context, string) (RunScope, error)
	Close(context.Context, string, int64) (RunScope, []RunScopeChild, error)
	Children(context.Context, string) ([]RunScopeChild, error)
	Finalize(context.Context, string, RunScopeState, int64) (RunScope, error)
}

type runScopeService struct {
	repo RunScopeRepository
	now  func() time.Time
}

func NewRunScopeService(repo RunScopeRepository, now func() time.Time) RunScopeService {
	if now == nil {
		now = time.Now
	}
	return &runScopeService{repo: repo, now: now}
}

func (s *runScopeService) Create(ctx context.Context, scope RunScope) error {
	scope.ID = strings.TrimSpace(scope.ID)
	scope.OwnerKind = strings.TrimSpace(scope.OwnerKind)
	if scope.ID == "" || len(scope.ID) > 128 || scope.OwnerKind == "" || len(scope.OwnerKind) > 80 || scope.MaxChildren < 1 || scope.MaxProviderCalls < 0 || scope.MaxProviderCalls > 64 || len(scope.Policy) == 0 || len(scope.Policy) > 64<<10 || !json.Valid(scope.Policy) {
		return fmt.Errorf("%w: invalid durable run scope", ErrInvalid)
	}
	scope.State, scope.Version, scope.CreatedAt = RunScopeActive, 1, s.now().UTC()
	scope.UpdatedAt = scope.CreatedAt
	return s.repo.CreateRunScope(ctx, scope)
}

func (s *runScopeService) Get(ctx context.Context, id string) (RunScope, error) {
	return s.repo.GetRunScope(ctx, strings.TrimSpace(id))
}

func (s *runScopeService) Close(ctx context.Context, id string, expected int64) (RunScope, []RunScopeChild, error) {
	return s.repo.CloseRunScope(ctx, strings.TrimSpace(id), expected, s.now().UTC())
}

func (s *runScopeService) Children(ctx context.Context, id string) ([]RunScopeChild, error) {
	return s.repo.ListRunScopeChildren(ctx, strings.TrimSpace(id))
}

func (s *runScopeService) Finalize(ctx context.Context, id string, state RunScopeState, expected int64) (RunScope, error) {
	if state != RunScopeCancelled && state != RunScopeCompleted {
		return RunScope{}, ErrInvalid
	}
	return s.repo.FinalizeRunScope(ctx, strings.TrimSpace(id), state, expected, s.now().UTC())
}

var _ RunScopeRepository = (*sqliteRepository)(nil)

func (r *sqliteRepository) CreateRunScope(ctx context.Context, s RunScope) error {
	policy := string(s.Policy)
	// Use the same serialized connection boundary as job admission.
	return r.withImmediate(ctx, func(c *sql.Conn) error {
		_, e := c.ExecContext(ctx, `INSERT INTO render_job_scopes(id,owner_kind,state,policy_json,max_children,max_provider_calls,children_admitted,provider_calls_reserved,version,created_at,updated_at) VALUES(?,?,?,?,?,?,0,0,?,?,?)`, s.ID, s.OwnerKind, s.State, policy, s.MaxChildren, s.MaxProviderCalls, s.Version, s.CreatedAt.UnixNano(), s.UpdatedAt.UnixNano())
		return e
	})
}

func (r *sqliteRepository) GetRunScope(ctx context.Context, id string) (RunScope, error) {
	var s RunScope
	var state string
	var policy string
	var created, updated int64
	err := r.db.QueryRowContext(ctx, `SELECT id,owner_kind,state,policy_json,max_children,max_provider_calls,children_admitted,provider_calls_reserved,version,created_at,updated_at FROM render_job_scopes WHERE id=?`, id).Scan(&s.ID, &s.OwnerKind, &state, &policy, &s.MaxChildren, &s.MaxProviderCalls, &s.ChildrenAdmitted, &s.ProviderCallsReserved, &s.Version, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return RunScope{}, ErrNotFound
	}
	if err != nil {
		return RunScope{}, err
	}
	s.State, s.Policy = RunScopeState(state), json.RawMessage(policy)
	s.CreatedAt, s.UpdatedAt = time.Unix(0, created).UTC(), time.Unix(0, updated).UTC()
	return s, nil
}

func (r *sqliteRepository) CloseRunScope(ctx context.Context, id string, expected int64, at time.Time) (RunScope, []RunScopeChild, error) {
	var out RunScope
	var children []RunScopeChild
	err := r.withImmediate(ctx, func(c *sql.Conn) error {
		s, e := loadRunScope(ctx, c, id)
		if e != nil {
			return e
		}
		if s.Version != expected {
			return ErrConcurrentTransition
		}
		if s.State == RunScopeActive {
			if _, e = c.ExecContext(ctx, `UPDATE render_job_scopes SET state='closing',version=version+1,updated_at=? WHERE id=? AND version=? AND state='active'`, at.UnixNano(), id, expected); e != nil {
				return e
			}
		} else if s.State != RunScopeClosing {
			return ErrAlreadyTerminal
		}
		children, e = listScopeChildren(ctx, c, id)
		if e != nil {
			return e
		}
		out, e = loadRunScope(ctx, c, id)
		return e
	})
	return out, children, err
}

func (r *sqliteRepository) ListRunScopeChildren(ctx context.Context, id string) ([]RunScopeChild, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT job_id,input_fingerprint,ordinal,provider_calls FROM render_job_scope_children WHERE scope_id=? ORDER BY ordinal`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanScopeChildren(rows)
}

func (r *sqliteRepository) FinalizeRunScope(ctx context.Context, id string, state RunScopeState, expected int64, at time.Time) (RunScope, error) {
	var out RunScope
	err := r.withImmediate(ctx, func(c *sql.Conn) error {
		s, e := loadRunScope(ctx, c, id)
		if e != nil {
			return e
		}
		if s.Version != expected {
			return ErrConcurrentTransition
		}
		if s.State != RunScopeClosing && s.State != RunScopeActive {
			return ErrAlreadyTerminal
		}
		var active int
		if e = c.QueryRowContext(ctx, `SELECT COUNT(*) FROM render_job_scope_children sc JOIN render_jobs j ON j.id=sc.job_id WHERE sc.scope_id=? AND json_extract(j.record_json,'$.state') IN ('queued','running','checkpointed','cancel_requested')`, id).Scan(&active); e != nil {
			return e
		}
		if active != 0 {
			return fmt.Errorf("%w: workflow scope still has %d active child jobs", ErrInvalidTransition, active)
		}
		if state == RunScopeCompleted && s.State != RunScopeActive {
			return ErrInvalidTransition
		}
		if _, e = c.ExecContext(ctx, `UPDATE render_job_scopes SET state=?,version=version+1,updated_at=? WHERE id=? AND version=?`, state, at.UnixNano(), id, expected); e != nil {
			return e
		}
		out, e = loadRunScope(ctx, c, id)
		return e
	})
	return out, err
}

func loadRunScope(ctx context.Context, c *sql.Conn, id string) (RunScope, error) {
	var s RunScope
	var state, policy string
	var created, updated int64
	err := c.QueryRowContext(ctx, `SELECT id,owner_kind,state,policy_json,max_children,max_provider_calls,children_admitted,provider_calls_reserved,version,created_at,updated_at FROM render_job_scopes WHERE id=?`, id).Scan(&s.ID, &s.OwnerKind, &state, &policy, &s.MaxChildren, &s.MaxProviderCalls, &s.ChildrenAdmitted, &s.ProviderCallsReserved, &s.Version, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return RunScope{}, ErrNotFound
	}
	if err != nil {
		return RunScope{}, err
	}
	s.State, s.Policy = RunScopeState(state), json.RawMessage(policy)
	s.CreatedAt, s.UpdatedAt = time.Unix(0, created).UTC(), time.Unix(0, updated).UTC()
	return s, nil
}

func listScopeChildren(ctx context.Context, c *sql.Conn, id string) ([]RunScopeChild, error) {
	rows, err := c.QueryContext(ctx, `SELECT job_id,input_fingerprint,ordinal,provider_calls FROM render_job_scope_children WHERE scope_id=? ORDER BY ordinal`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanScopeChildren(rows)
}

type scopeChildRows interface {
	Next() bool
	Scan(...any) error
	Err() error
}

func scanScopeChildren(rows scopeChildRows) ([]RunScopeChild, error) {
	children := []RunScopeChild{}
	for rows.Next() {
		var child RunScopeChild
		if err := rows.Scan(&child.JobID, &child.InputFingerprint, &child.Ordinal, &child.ProviderCalls); err != nil {
			return nil, err
		}
		children = append(children, child)
	}
	return children, rows.Err()
}
