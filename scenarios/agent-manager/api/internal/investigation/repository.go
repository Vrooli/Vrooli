package investigation

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/sqlcompat"

	"github.com/google/uuid"
)

type Lifecycle struct {
	ID              string     `json:"investigationId"`
	Request         Request    `json:"request"`
	RequestDigest   string     `json:"requestDigest"`
	OperationStatus string     `json:"operationStatus"`
	Result          *Result    `json:"result,omitempty"`
	SourceCutJSON   string     `json:"-"`
	WorkflowRef     string     `json:"workflowRef,omitempty"`
	CancelRequested bool       `json:"cancelRequested"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	CompletedAt     *time.Time `json:"completedAt,omitempty"`
	CancelledAt     *time.Time `json:"cancelledAt,omitempty"`
}

type Repository interface {
	Reserve(context.Context, Request) (*Lifecycle, bool, error)
	Get(context.Context, string) (*Lifecycle, error)
	List(context.Context, string, int) ([]*Lifecycle, error)
	Wait(context.Context, string, time.Duration) (*Lifecycle, bool, error)
	UpdateStatus(context.Context, string, string, string) (*Lifecycle, error)
	Complete(context.Context, string, Result) (*Lifecycle, error)
	Cancel(context.Context, string) (*Lifecycle, error)
}

var (
	ErrNotFound          = errors.New("investigation not found")
	ErrInvalidTransition = errors.New("invalid investigation lifecycle transition")
)

type SQLiteRepository struct {
	db  sqlcompat.DB
	now func() time.Time
}

func NewSQLiteRepository(db sqlcompat.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db, now: time.Now}
}

func (r *SQLiteRepository) Reserve(ctx context.Context, request Request) (*Lifecycle, bool, error) {
	digest, err := request.CanonicalDigest()
	if err != nil {
		return nil, false, err
	}
	authority := strings.TrimSpace(request.CallerAuthority)
	if authority == "" {
		// Requests authored before the optional caller-authority field still
		// receive same-owner/same-key idempotency semantics.
		authority = strings.TrimSpace(request.Subject.Owner)
	}
	if existing, getErr := r.getByKey(ctx, authority, strings.TrimSpace(request.RequestKey)); getErr == nil {
		if existing.RequestDigest != digest {
			return nil, false, fmt.Errorf("%w: callerAuthority=%q requestKey=%q", ErrRequestKeyConflict, authority, request.RequestKey)
		}
		return existing, true, nil
	} else if !errors.Is(getErr, ErrNotFound) {
		return nil, false, getErr
	}

	now := r.now().UTC()
	id := uuid.NewString()
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, false, fmt.Errorf("marshal investigation request: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO investigations
		(investigation_id,caller_authority,request_key,request_digest,request_json,operation_status,created_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?)`, id, authority, strings.TrimSpace(request.RequestKey), digest, string(requestJSON), OperationQueued, formatTime(now), formatTime(now))
	if err != nil {
		// Another caller may have won the same-key race. Re-read the durable
		// admission before exposing a transient uniqueness error.
		if existing, getErr := r.getByKey(ctx, authority, strings.TrimSpace(request.RequestKey)); getErr == nil {
			if existing.RequestDigest != digest {
				return nil, false, fmt.Errorf("%w: callerAuthority=%q requestKey=%q", ErrRequestKeyConflict, authority, request.RequestKey)
			}
			return existing, true, nil
		}
		return nil, false, fmt.Errorf("reserve investigation: %w", err)
	}
	return &Lifecycle{ID: id, Request: request, RequestDigest: digest, OperationStatus: OperationQueued, CreatedAt: now, UpdatedAt: now}, false, nil
}

func (r *SQLiteRepository) Get(ctx context.Context, id string) (*Lifecycle, error) {
	return r.scan(ctx, r.db.QueryRowContext(ctx, investigationSelect+` WHERE investigation_id=?`, strings.TrimSpace(id)))
}

func (r *SQLiteRepository) List(ctx context.Context, status string, limit int) ([]*Lifecycle, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query := investigationSelect + ` WHERE 1=1`
	args := []any{}
	if status = strings.TrimSpace(status); status != "" {
		query += ` AND operation_status=?`
		args = append(args, status)
	}
	query += ` ORDER BY created_at, investigation_id LIMIT ?`
	args = append(args, limit)
	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list investigations: %w", err)
	}
	defer rows.Close()
	result := make([]*Lifecycle, 0, limit)
	for rows.Next() {
		item, err := r.scan(ctx, rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate investigations: %w", err)
	}
	return result, nil
}

// Wait is a bounded reattachment operation. It observes only durable
// lifecycle state and returns the current item on timeout; a detached caller
// can safely issue Get later without owning a worker or an in-memory handle.
func (r *SQLiteRepository) Wait(ctx context.Context, id string, timeout time.Duration) (*Lifecycle, bool, error) {
	if timeout <= 0 || timeout > 30*time.Minute {
		timeout = 30 * time.Second
	}
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		item, err := r.Get(ctx, id)
		if err != nil {
			return nil, false, err
		}
		if item.OperationStatus == OperationCompleted || item.OperationStatus == OperationFailed || item.OperationStatus == OperationCancelled {
			return item, true, nil
		}
		select {
		case <-ctx.Done():
			return item, false, ctx.Err()
		case <-deadline.C:
			return item, false, nil
		case <-ticker.C:
		}
	}
}

func (r *SQLiteRepository) UpdateStatus(ctx context.Context, id, status, workflowRef string) (*Lifecycle, error) {
	item, err := r.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if !validOperationStatus(status) || !allowedTransition(item.OperationStatus, status) {
		return nil, fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, item.OperationStatus, status)
	}
	now := r.now().UTC()
	_, err = r.db.ExecContext(ctx, `UPDATE investigations SET operation_status=?,workflow_ref=CASE WHEN ?<>'' THEN ? ELSE workflow_ref END,updated_at=? WHERE investigation_id=?`, status, strings.TrimSpace(workflowRef), strings.TrimSpace(workflowRef), formatTime(now), id)
	if err != nil {
		return nil, fmt.Errorf("update investigation status: %w", err)
	}
	return r.Get(ctx, id)
}

func (r *SQLiteRepository) Complete(ctx context.Context, id string, result Result) (*Lifecycle, error) {
	item, err := r.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if item.OperationStatus == OperationCompleted {
		result.InvestigationID = item.ID
		if item.Result != nil && sameResult(*item.Result, result) {
			return item, nil
		}
		return nil, fmt.Errorf("%w: completed investigation already has a different result", ErrInvalidTransition)
	}
	if item.OperationStatus == OperationCancelled || item.OperationStatus == OperationFailed || item.CancelRequested {
		return nil, fmt.Errorf("%w: cancelled investigation cannot complete", ErrInvalidTransition)
	}
	if result.OperationStatus != OperationCompleted {
		return nil, fmt.Errorf("%w: completion result must have operationStatus=%q", ErrInvalidTransition, OperationCompleted)
	}
	// The repository owns the durable investigation identity. Callers submit
	// the result body, but cannot manufacture a second result identity at the
	// completion boundary.
	result.InvestigationID = item.ID
	if err := result.Validate(item.Request); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal investigation result: %w", err)
	}
	now := r.now().UTC()
	update, err := r.db.ExecContext(ctx, `UPDATE investigations SET operation_status=?,result_json=?,source_cut_json=?,updated_at=?,completed_at=? WHERE investigation_id=? AND operation_status NOT IN (?,?) AND cancel_requested=0`, OperationCompleted, string(raw), marshalCut(result.EvidenceCut), formatTime(now), formatTime(now), id, OperationCompleted, OperationFailed)
	if err != nil {
		return nil, fmt.Errorf("complete investigation: %w", err)
	}
	if affected, affectedErr := update.RowsAffected(); affectedErr == nil && affected == 0 {
		current, getErr := r.Get(ctx, id)
		if getErr == nil && current.OperationStatus == OperationCompleted && current.Result != nil && sameResult(*current.Result, result) {
			return current, nil
		}
		return nil, fmt.Errorf("%w: investigation changed before completion", ErrInvalidTransition)
	}
	return r.Get(ctx, id)
}

func sameResult(left, right Result) bool {
	leftRaw, leftErr := json.Marshal(left)
	rightRaw, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftRaw, rightRaw)
}

func (r *SQLiteRepository) Cancel(ctx context.Context, id string) (*Lifecycle, error) {
	item, err := r.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if item.OperationStatus == OperationCompleted || item.OperationStatus == OperationFailed || item.OperationStatus == OperationCancelled {
		return item, nil
	}
	now := r.now().UTC()
	_, err = r.db.ExecContext(ctx, `UPDATE investigations SET operation_status=?,cancel_requested=1,updated_at=?,cancelled_at=? WHERE investigation_id=? AND operation_status NOT IN (?,?,?)`, OperationCancelled, formatTime(now), formatTime(now), id, OperationCompleted, OperationFailed, OperationCancelled)
	if err != nil {
		return nil, fmt.Errorf("cancel investigation: %w", err)
	}
	return r.Get(ctx, id)
}

const investigationSelect = `SELECT investigation_id,caller_authority,request_digest,request_json,operation_status,result_json,source_cut_json,workflow_ref,cancel_requested,created_at,updated_at,completed_at,cancelled_at FROM investigations`

type scanner interface{ Scan(...any) error }

func (r *SQLiteRepository) scan(_ context.Context, row scanner) (*Lifecycle, error) {
	var item Lifecycle
	var authority, requestJSON, resultJSON, sourceCutJSON string
	var completedAt, cancelledAt sql.NullString
	var createdAt, updatedAt string
	var cancelRequested int
	if err := row.Scan(&item.ID, &authority, &item.RequestDigest, &requestJSON, &item.OperationStatus, &resultJSON, &sourceCutJSON, &item.WorkflowRef, &cancelRequested, &createdAt, &updatedAt, &completedAt, &cancelledAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan investigation: %w", err)
	}
	item.SourceCutJSON = sourceCutJSON
	item.CreatedAt = parseTime(createdAt)
	item.UpdatedAt = parseTime(updatedAt)
	if err := json.Unmarshal([]byte(requestJSON), &item.Request); err != nil {
		return nil, fmt.Errorf("decode stored investigation request: %w", err)
	}
	item.CancelRequested = cancelRequested != 0
	if resultJSON != "" {
		item.Result = &Result{}
		if err := json.Unmarshal([]byte(resultJSON), item.Result); err != nil {
			return nil, fmt.Errorf("decode stored investigation result: %w", err)
		}
	}
	item.CompletedAt = parseOptionalTime(completedAt.String)
	item.CancelledAt = parseOptionalTime(cancelledAt.String)
	_ = authority // retained in request provenance only; the key is server-owned.
	return &item, nil
}

func (r *SQLiteRepository) getByKey(ctx context.Context, authority, key string) (*Lifecycle, error) {
	return r.scan(ctx, r.db.QueryRowContext(ctx, investigationSelect+` WHERE caller_authority=? AND request_key=?`, authority, key))
}

func allowedTransition(from, to string) bool {
	if from == to {
		return true
	}
	switch from {
	case OperationQueued:
		return to == OperationCollecting || to == OperationCancelled || to == OperationFailed
	case OperationCollecting:
		return to == OperationDiagnosing || to == OperationCancelled || to == OperationFailed
	case OperationDiagnosing:
		return to == OperationCompleted || to == OperationCancelled || to == OperationFailed
	default:
		return false
	}
}

func marshalCut(cut EvidenceCut) string { raw, _ := json.Marshal(cut); return string(raw) }
func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }
func parseTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}
func parseOptionalTime(value string) *time.Time {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil
	}
	return &parsed
}
