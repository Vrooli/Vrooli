package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type workflowExecutionRepository struct {
	db  *DB
	log *logrus.Logger
}

var _ repository.WorkflowExecutionRepository = (*workflowExecutionRepository)(nil)

type workflowExecutionRow struct {
	ID                 string         `db:"id"`
	Owner              string         `db:"owner"`
	WorkflowKey        string         `db:"workflow_key"`
	DefinitionDigest   string         `db:"definition_digest"`
	Status             string         `db:"status"`
	CurrentNodeID      string         `db:"current_node_id"`
	InputJSON          string         `db:"input_json"`
	OutputJSON         sql.NullString `db:"output_json"`
	TerminalReasonJSON sql.NullString `db:"terminal_reason_json"`
	BudgetUsageJSON    string         `db:"budget_usage_json"`
	EdgeTraversalsJSON string         `db:"edge_traversals_json"`
	Version            int64          `db:"version"`
	IdempotencyKey     string         `db:"idempotency_key"`
	ParentExecutionID  sql.NullString `db:"parent_execution_id"`
	ParentAttemptID    sql.NullString `db:"parent_attempt_id"`
	Depth              int            `db:"depth"`
	CreatedAt          SQLiteTime     `db:"created_at"`
	UpdatedAt          SQLiteTime     `db:"updated_at"`
	EndedAt            sql.NullString `db:"ended_at"`
}

func (r *workflowExecutionRepository) Create(ctx context.Context, e *domain.WorkflowExecution, initial *domain.WorkflowJournalEntry) error {
	if e == nil || initial == nil {
		return errors.New("execution and initial journal entry are required")
	}
	return r.db.WithTransaction(func(tx *sqlx.Tx) error {
		if err := insertWorkflowExecution(ctx, tx, e); err != nil {
			return err
		}
		return insertJournal(ctx, tx, initial)
	})
}

func insertWorkflowExecution(ctx context.Context, tx *sqlx.Tx, e *domain.WorkflowExecution) error {
	budget, _ := json.Marshal(e.BudgetUsage)
	edges, _ := json.Marshal(e.EdgeTraversals)
	terminal, _ := json.Marshal(e.TerminalReason)
	var output any
	if len(e.Output) != 0 {
		output = string(e.Output)
	}
	var terminalValue any
	if e.TerminalReason != nil {
		terminalValue = string(terminal)
	}
	var parent any
	if e.ParentExecutionID != nil {
		parent = e.ParentExecutionID.String()
	}
	var ended any
	if e.EndedAt != nil {
		ended = SQLiteTime(*e.EndedAt)
	}
	var parentAttempt any
	if e.ParentAttemptID != nil {
		parentAttempt = e.ParentAttemptID.String()
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO workflow_executions (id,owner,workflow_key,definition_digest,status,current_node_id,input_json,output_json,terminal_reason_json,budget_usage_json,edge_traversals_json,version,idempotency_key,parent_execution_id,parent_attempt_id,depth,created_at,updated_at,ended_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, e.ID, e.Owner, e.WorkflowKey, e.DefinitionDigest, e.Status, e.CurrentNodeID, string(e.Input), output, terminalValue, string(budget), string(edges), e.Version, e.IdempotencyKey, parent, parentAttempt, e.Depth, SQLiteTime(e.CreatedAt), SQLiteTime(e.UpdatedAt), ended)
	return err
}

func (r *workflowExecutionRepository) Get(ctx context.Context, id uuid.UUID) (*domain.WorkflowExecution, error) {
	return r.get(ctx, `SELECT id,owner,workflow_key,definition_digest,status,current_node_id,input_json,output_json,terminal_reason_json,budget_usage_json,edge_traversals_json,version,idempotency_key,parent_execution_id,parent_attempt_id,depth,created_at,updated_at,ended_at FROM workflow_executions WHERE id=?`, id)
}

func (r *workflowExecutionRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.WorkflowExecution, error) {
	return r.get(ctx, `SELECT id,owner,workflow_key,definition_digest,status,current_node_id,input_json,output_json,terminal_reason_json,budget_usage_json,edge_traversals_json,version,idempotency_key,parent_execution_id,parent_attempt_id,depth,created_at,updated_at,ended_at FROM workflow_executions WHERE idempotency_key=?`, key)
}

func (r *workflowExecutionRepository) List(ctx context.Context, filter repository.WorkflowExecutionListFilter) ([]*domain.WorkflowExecution, error) {
	query := `SELECT id,owner,workflow_key,definition_digest,status,current_node_id,input_json,output_json,terminal_reason_json,budget_usage_json,edge_traversals_json,version,idempotency_key,parent_execution_id,parent_attempt_id,depth,created_at,updated_at,ended_at FROM workflow_executions`
	clauses := make([]string, 0, 3)
	args := make([]any, 0, 5)
	if filter.Owner != "" {
		clauses = append(clauses, "owner=?")
		args = append(args, filter.Owner)
	}
	if filter.WorkflowKey != "" {
		clauses = append(clauses, "workflow_key=?")
		args = append(args, filter.WorkflowKey)
	}
	if filter.Status != "" {
		clauses = append(clauses, "status=?")
		args = append(args, filter.Status)
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, max(filter.Offset, 0))
	var rows []workflowExecutionRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}
	items := make([]*domain.WorkflowExecution, 0, len(rows))
	for i := range rows {
		execution, err := workflowExecutionFromRow(rows[i])
		if err != nil {
			return nil, err
		}
		items = append(items, execution)
	}
	return items, nil
}

func (r *workflowExecutionRepository) get(ctx context.Context, q string, args ...any) (*domain.WorkflowExecution, error) {
	var row workflowExecutionRow
	if err := r.db.GetContext(ctx, &row, q, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return workflowExecutionFromRow(row)
}

func workflowExecutionFromRow(row workflowExecutionRow) (*domain.WorkflowExecution, error) {
	id, err := uuid.Parse(row.ID)
	if err != nil {
		return nil, err
	}
	e := &domain.WorkflowExecution{ID: id, Owner: row.Owner, WorkflowKey: row.WorkflowKey, DefinitionDigest: row.DefinitionDigest, Status: domain.WorkflowExecutionStatus(row.Status), CurrentNodeID: row.CurrentNodeID, Input: json.RawMessage(row.InputJSON), Version: row.Version, IdempotencyKey: row.IdempotencyKey, Depth: row.Depth, CreatedAt: row.CreatedAt.Time(), UpdatedAt: row.UpdatedAt.Time()}
	if row.OutputJSON.Valid {
		e.Output = json.RawMessage(row.OutputJSON.String)
	}
	if row.TerminalReasonJSON.Valid {
		var v domain.WorkflowTerminalReason
		if err := json.Unmarshal([]byte(row.TerminalReasonJSON.String), &v); err != nil {
			return nil, err
		}
		e.TerminalReason = &v
	}
	if err := json.Unmarshal([]byte(row.BudgetUsageJSON), &e.BudgetUsage); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(row.EdgeTraversalsJSON), &e.EdgeTraversals); err != nil {
		return nil, err
	}
	if row.ParentExecutionID.Valid {
		v, err := uuid.Parse(row.ParentExecutionID.String)
		if err != nil {
			return nil, err
		}
		e.ParentExecutionID = &v
	}
	if row.ParentAttemptID.Valid {
		v, err := uuid.Parse(row.ParentAttemptID.String)
		if err != nil {
			return nil, err
		}
		e.ParentAttemptID = &v
	}
	if row.EndedAt.Valid {
		var t SQLiteTime
		if err := t.Scan(row.EndedAt.String); err != nil {
			return nil, err
		}
		v := t.Time()
		e.EndedAt = &v
	}
	return e, nil
}

func (r *workflowExecutionRepository) Commit(ctx context.Context, c repository.WorkflowCommit) (bool, error) {
	if c.Execution == nil || c.Execution.Version != c.ExpectedVersion+1 {
		return false, fmt.Errorf("execution version must advance by one")
	}
	committed := false
	err := r.db.WithTransaction(func(tx *sqlx.Tx) error {
		e := c.Execution
		budget, _ := json.Marshal(e.BudgetUsage)
		edges, _ := json.Marshal(e.EdgeTraversals)
		terminal, _ := json.Marshal(e.TerminalReason)
		var output any
		if len(e.Output) != 0 {
			output = string(e.Output)
		}
		var terminalValue any
		if e.TerminalReason != nil {
			terminalValue = string(terminal)
		}
		var ended any
		if e.EndedAt != nil {
			ended = SQLiteTime(*e.EndedAt)
		}
		res, err := tx.ExecContext(ctx, `UPDATE workflow_executions SET status=?,current_node_id=?,output_json=?,terminal_reason_json=?,budget_usage_json=?,edge_traversals_json=?,version=?,updated_at=?,ended_at=? WHERE id=? AND version=?`, e.Status, e.CurrentNodeID, output, terminalValue, string(budget), string(edges), e.Version, SQLiteTime(e.UpdatedAt), ended, e.ID, c.ExpectedVersion)
		if err != nil {
			return err
		}
		count, _ := res.RowsAffected()
		if count == 0 {
			return nil
		}
		if c.Attempt != nil {
			if err := upsertAttempt(ctx, tx, c.Attempt); err != nil {
				return err
			}
		}
		for _, attempt := range c.Attempts {
			if err := upsertAttempt(ctx, tx, attempt); err != nil {
				return err
			}
		}
		for _, entry := range c.Journal {
			if err := insertJournal(ctx, tx, entry); err != nil {
				return err
			}
		}
		committed = true
		return nil
	})
	return committed, err
}

func upsertAttempt(ctx context.Context, tx *sqlx.Tx, a *domain.WorkflowNodeAttempt) error {
	var run, source, child, completed any
	if a.RunID != nil {
		run = a.RunID.String()
	}
	if a.SourceAttemptID != nil {
		source = a.SourceAttemptID.String()
	}
	if a.ChildExecutionID != nil {
		child = a.ChildExecutionID.String()
	}
	if a.CompletedAt != nil {
		completed = SQLiteTime(*a.CompletedAt)
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO workflow_node_attempts (id,execution_id,node_id,ordinal,strategy,status,idempotency_key,input_snapshot_json,prompt_snapshot,run_id,conversation_id,source_attempt_id,child_execution_id,error_code,version,created_at,updated_at,completed_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET status=excluded.status,run_id=excluded.run_id,conversation_id=excluded.conversation_id,source_attempt_id=excluded.source_attempt_id,child_execution_id=excluded.child_execution_id,error_code=excluded.error_code,version=excluded.version,updated_at=excluded.updated_at,completed_at=excluded.completed_at`, a.ID, a.ExecutionID, a.NodeID, a.Ordinal, a.Strategy, a.Status, a.IdempotencyKey, string(a.InputSnapshot), a.PromptSnapshot, run, a.ConversationID, source, child, a.ErrorCode, a.Version, SQLiteTime(a.CreatedAt), SQLiteTime(a.UpdatedAt), completed)
	return err
}

func insertJournal(ctx context.Context, tx *sqlx.Tx, e *domain.WorkflowJournalEntry) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO workflow_journal (id,execution_id,sequence,kind,node_id,attempt_id,payload_json,created_at) VALUES (?,?,?,?,?,?,?,?)`, e.ID, e.ExecutionID, e.Sequence, e.Kind, e.NodeID, e.AttemptID, string(e.Payload), SQLiteTime(e.CreatedAt))
	return err
}

func (r *workflowExecutionRepository) GetAttemptByIdempotencyKey(ctx context.Context, key string) (*domain.WorkflowNodeAttempt, error) {
	var rows []workflowAttemptRow
	if err := r.db.SelectContext(ctx, &rows, `SELECT id,execution_id,node_id,ordinal,strategy,status,idempotency_key,input_snapshot_json,prompt_snapshot,run_id,conversation_id,source_attempt_id,child_execution_id,error_code,version,created_at,updated_at,completed_at FROM workflow_node_attempts WHERE idempotency_key=?`, key); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0].domain()
}

type workflowAttemptRow struct {
	ID                string         `db:"id"`
	ExecutionID       string         `db:"execution_id"`
	NodeID            string         `db:"node_id"`
	Ordinal           int            `db:"ordinal"`
	Strategy          string         `db:"strategy"`
	Status            string         `db:"status"`
	IdempotencyKey    string         `db:"idempotency_key"`
	InputSnapshotJSON string         `db:"input_snapshot_json"`
	PromptSnapshot    string         `db:"prompt_snapshot"`
	RunID             sql.NullString `db:"run_id"`
	ConversationID    string         `db:"conversation_id"`
	SourceAttemptID   sql.NullString `db:"source_attempt_id"`
	ChildExecutionID  sql.NullString `db:"child_execution_id"`
	ErrorCode         string         `db:"error_code"`
	Version           int64          `db:"version"`
	CreatedAt         SQLiteTime     `db:"created_at"`
	UpdatedAt         SQLiteTime     `db:"updated_at"`
	CompletedAt       sql.NullString `db:"completed_at"`
}

func (r workflowAttemptRow) domain() (*domain.WorkflowNodeAttempt, error) {
	id, eid := uuid.MustParse(r.ID), uuid.MustParse(r.ExecutionID)
	a := &domain.WorkflowNodeAttempt{ID: id, ExecutionID: eid, NodeID: r.NodeID, Ordinal: r.Ordinal, Strategy: domain.WorkflowAttemptStrategy(r.Strategy), Status: domain.WorkflowAttemptStatus(r.Status), IdempotencyKey: r.IdempotencyKey, InputSnapshot: json.RawMessage(r.InputSnapshotJSON), PromptSnapshot: r.PromptSnapshot, ConversationID: r.ConversationID, ErrorCode: r.ErrorCode, Version: r.Version, CreatedAt: r.CreatedAt.Time(), UpdatedAt: r.UpdatedAt.Time()}
	if r.RunID.Valid {
		v, err := uuid.Parse(r.RunID.String)
		if err != nil {
			return nil, err
		}
		a.RunID = &v
	}
	if r.SourceAttemptID.Valid {
		v, err := uuid.Parse(r.SourceAttemptID.String)
		if err != nil {
			return nil, err
		}
		a.SourceAttemptID = &v
	}
	if r.ChildExecutionID.Valid {
		v, err := uuid.Parse(r.ChildExecutionID.String)
		if err != nil {
			return nil, err
		}
		a.ChildExecutionID = &v
	}
	if r.CompletedAt.Valid {
		var t SQLiteTime
		if err := t.Scan(r.CompletedAt.String); err != nil {
			return nil, err
		}
		v := t.Time()
		a.CompletedAt = &v
	}
	return a, nil
}

func (r *workflowExecutionRepository) ListAttempts(ctx context.Context, id uuid.UUID) ([]*domain.WorkflowNodeAttempt, error) {
	var rows []workflowAttemptRow
	if err := r.db.SelectContext(ctx, &rows, `SELECT id,execution_id,node_id,ordinal,strategy,status,idempotency_key,input_snapshot_json,prompt_snapshot,run_id,conversation_id,source_attempt_id,child_execution_id,error_code,version,created_at,updated_at,completed_at FROM workflow_node_attempts WHERE execution_id=? ORDER BY created_at,id`, id); err != nil {
		return nil, err
	}
	out := make([]*domain.WorkflowNodeAttempt, 0, len(rows))
	for _, row := range rows {
		v, err := row.domain()
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

type workflowJournalRow struct {
	ID          string         `db:"id"`
	ExecutionID string         `db:"execution_id"`
	Sequence    int64          `db:"sequence"`
	Kind        string         `db:"kind"`
	NodeID      string         `db:"node_id"`
	AttemptID   sql.NullString `db:"attempt_id"`
	PayloadJSON string         `db:"payload_json"`
	CreatedAt   SQLiteTime     `db:"created_at"`
}

func (r *workflowExecutionRepository) ListJournal(ctx context.Context, id uuid.UUID, after int64, limit int) ([]*domain.WorkflowJournalEntry, error) {
	q := `SELECT id,execution_id,sequence,kind,node_id,attempt_id,payload_json,created_at FROM workflow_journal WHERE execution_id=? AND sequence>? ORDER BY sequence`
	args := []any{id, after}
	q, page := appendLimitOffset(q, limit, 0)
	args = append(args, page...)
	var rows []workflowJournalRow
	if err := r.db.SelectContext(ctx, &rows, q, args...); err != nil {
		return nil, err
	}
	out := make([]*domain.WorkflowJournalEntry, 0, len(rows))
	for _, row := range rows {
		entry := &domain.WorkflowJournalEntry{ID: uuid.MustParse(row.ID), ExecutionID: uuid.MustParse(row.ExecutionID), Sequence: row.Sequence, Kind: domain.WorkflowJournalKind(row.Kind), NodeID: row.NodeID, Payload: json.RawMessage(row.PayloadJSON), CreatedAt: row.CreatedAt.Time()}
		if row.AttemptID.Valid {
			v := uuid.MustParse(row.AttemptID.String)
			entry.AttemptID = &v
		}
		out = append(out, entry)
	}
	return out, nil
}

func (r *workflowExecutionRepository) ListRecoverable(ctx context.Context, limit int) ([]*domain.WorkflowExecution, error) {
	q := `SELECT id FROM workflow_executions WHERE status IN ('pending','running','waiting') ORDER BY updated_at`
	q, args := appendLimitOffset(q, limit, 0)
	var ids []string
	if err := r.db.SelectContext(ctx, &ids, q, args...); err != nil {
		return nil, err
	}
	out := make([]*domain.WorkflowExecution, 0, len(ids))
	for _, raw := range ids {
		e, err := r.Get(ctx, uuid.MustParse(raw))
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}
