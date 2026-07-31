package routing

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	"google.golang.org/protobuf/encoding/protojson"
)

// MediaExecutor is the narrow bridge from Gateway policy/receipts to a media
// capability. It deliberately accepts no provider URL, credentials, or model
// selector. Implementations own their bytes and return stable references.
type MediaExecutor interface {
	Execute(context.Context, *routingv1.SubmitMediaRequest) (*MediaExecutionResult, error)
	Cancel(context.Context, string) error
}

type MediaExecutionResult struct {
	RouteEvidence *routingv1.RouteEvidence
	Outputs       []*routingv1.MediaOutput
	ActualCostUSD float64
	ResolvedModel string
	Seed          string
	Warnings      []string
}

// MediaService persists a durable receipt before dispatching execution. A
// nil executor is an explicit unavailable state, never a synthetic success.
type MediaService struct {
	db       *sql.DB
	executor MediaExecutor
	clock    func() time.Time
	mu       sync.Mutex
}

func NewMediaService(db *sql.DB, executor MediaExecutor) *MediaService {
	return &MediaService{db: db, executor: executor, clock: time.Now}
}

func (s *MediaService) Submit(ctx context.Context, req *routingv1.SubmitMediaRequest) (*routingv1.MediaExecution, error) {
	if err := validateMediaSubmission(req); err != nil {
		return nil, err
	}
	if s == nil || s.db == nil {
		return nil, errors.New("media execution store is not configured")
	}
	key := strings.TrimSpace(req.GetIdempotencyKey())
	if existing, err := s.getByIdempotency(ctx, key); err == nil {
		return existing, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	payload, err := protojson.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encode media submission: %w", err)
	}
	now := s.now().Format(time.RFC3339Nano)
	exec := &routingv1.MediaExecution{
		ExecutionId:    newID("media"),
		IdempotencyKey: key,
		Status:         routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_QUEUED,
		CreatedAt:      now,
	}
	if err := s.insert(ctx, exec, string(payload)); err != nil {
		// SQLite's unique constraint is the final idempotency guard when two
		// callers race after the read above.
		if existing, lookupErr := s.getByIdempotency(ctx, key); lookupErr == nil {
			return existing, nil
		}
		return nil, err
	}

	// Dispatch only after the receipt is durable. The worker owns all later
	// transitions, so a caller can reconnect using execution_id immediately.
	go s.run(exec.GetExecutionId(), req)
	return exec, nil
}

func (s *MediaService) Get(ctx context.Context, id string) (*routingv1.MediaExecution, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("media execution store is not configured")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("execution_id is required")
	}
	return s.get(ctx, id)
}

// Wait blocks inside the Gateway until a durable media receipt reaches a
// terminal state. This is the canonical consumer path; clients never need to
// manufacture polling loops around Get.
func (s *MediaService) Wait(ctx context.Context, id string) (*routingv1.MediaExecution, error) {
	const interval = 100 * time.Millisecond
	for {
		exec, err := s.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		if terminalMediaStatus(exec.GetStatus()) {
			return exec, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}
	}
}

func (s *MediaService) Cancel(ctx context.Context, id string) (*routingv1.MediaExecution, error) {
	exec, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if terminalMediaStatus(exec.GetStatus()) {
		return exec, nil
	}
	if s.executor != nil {
		if err := s.executor.Cancel(ctx, exec.GetExecutionId()); err != nil {
			return nil, fmt.Errorf("cancel media executor: %w", err)
		}
	}
	now := s.now().Format(time.RFC3339Nano)
	if _, err := s.db.ExecContext(ctx, `UPDATE media_executions SET status = ?, completed_at = ?, error_code = '', error_message = '' WHERE execution_id = ? AND status IN (?, ?)`,
		int32(routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_CANCELLED), now, exec.GetExecutionId(),
		int32(routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_QUEUED), int32(routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_RUNNING)); err != nil {
		return nil, err
	}
	return s.Get(ctx, exec.GetExecutionId())
}

func (s *MediaService) Retry(ctx context.Context, id, key string) (*routingv1.MediaExecution, error) {
	if strings.TrimSpace(key) == "" {
		return nil, errors.New("idempotency_key is required")
	}
	exec, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if !terminalMediaStatus(exec.GetStatus()) {
		return nil, errors.New("only terminal media executions can be retried")
	}
	var payload string
	if err := s.db.QueryRowContext(ctx, `SELECT request_json FROM media_executions WHERE execution_id = ?`, exec.GetExecutionId()).Scan(&payload); err != nil {
		return nil, err
	}
	var request routingv1.SubmitMediaRequest
	if err := protojson.Unmarshal([]byte(payload), &request); err != nil {
		return nil, fmt.Errorf("decode retained media submission: %w", err)
	}
	request.IdempotencyKey = strings.TrimSpace(key)
	return s.Submit(ctx, &request)
}

func (s *MediaService) run(id string, req *routingv1.SubmitMediaRequest) {
	ctx := context.Background()
	started := s.now().Format(time.RFC3339Nano)
	res, err := s.db.ExecContext(ctx, `UPDATE media_executions SET status = ?, started_at = ? WHERE execution_id = ? AND status = ?`,
		int32(routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_RUNNING), started, id,
		int32(routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_QUEUED))
	if err != nil {
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return // cancellation won the race.
	}
	if s.executor == nil {
		s.fail(id, "executor_unavailable", "no media executor is configured")
		return
	}
	result, err := s.executor.Execute(ctx, req)
	if err != nil {
		s.fail(id, "executor_failed", err.Error())
		return
	}
	if result == nil {
		s.fail(id, "executor_protocol", "media executor returned no result")
		return
	}
	if result.RouteEvidence == nil {
		s.fail(id, "executor_protocol", "media executor returned no route evidence")
		return
	}
	if err := validateMediaOutputs(result.Outputs); err != nil {
		s.fail(id, "executor_protocol", err.Error())
		return
	}
	outputs, err := protojson.Marshal(&routingv1.MediaExecution{Outputs: result.Outputs})
	if err != nil {
		s.fail(id, "receipt_encode_failed", err.Error())
		return
	}
	completed := s.now().Format(time.RFC3339Nano)
	_, _ = s.db.ExecContext(ctx, `UPDATE media_executions SET status = ?, completed_at = ?, route_evidence_json = ?, outputs_json = ?, actual_cost_usd = ?, resolved_model = ?, seed = ?, warnings_json = ? WHERE execution_id = ? AND status = ?`,
		int32(routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_SUCCEEDED), completed,
		marshalRouteEvidence(result.RouteEvidence), string(outputs), result.ActualCostUSD, result.ResolvedModel, result.Seed, marshalStrings(result.Warnings), id,
		int32(routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_RUNNING))
}

func (s *MediaService) fail(id, code, message string) {
	_, _ = s.db.ExecContext(context.Background(), `UPDATE media_executions SET status = ?, completed_at = ?, error_code = ?, error_message = ? WHERE execution_id = ? AND status = ?`,
		int32(routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_FAILED), s.now().Format(time.RFC3339Nano), code, message, id,
		int32(routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_RUNNING))
}

// recoverPending re-dispatches receipts that were durable but had not reached a
// terminal state when the process stopped. A previously RUNNING receipt is
// returned to QUEUED first; the regular CAS in run preserves cancellation
// races. If no executor is configured, run records an honest terminal failure.
func (s *MediaService) Recover(ctx context.Context) {
	if s == nil || s.db == nil {
		return
	}
	rows, err := s.db.QueryContext(ctx, `SELECT execution_id, request_json FROM media_executions WHERE status IN (?, ?)`,
		int32(routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_QUEUED),
		int32(routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_RUNNING))
	if err != nil {
		return
	}
	type pendingReceipt struct{ id, payload string }
	pending := make([]pendingReceipt, 0)
	for rows.Next() {
		var id, payload string
		if err := rows.Scan(&id, &payload); err != nil {
			continue
		}
		pending = append(pending, pendingReceipt{id: id, payload: payload})
	}
	if err := rows.Close(); err != nil {
		return
	}
	for _, receipt := range pending {
		id, payload := receipt.id, receipt.payload
		var req routingv1.SubmitMediaRequest
		if err := protojson.Unmarshal([]byte(payload), &req); err != nil {
			s.fail(id, "receipt_decode_failed", "stored media submission could not be decoded")
			continue
		}
		_, err = s.db.ExecContext(ctx, `UPDATE media_executions SET status = ?, started_at = '' WHERE execution_id = ? AND status = ?`,
			int32(routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_QUEUED), id,
			int32(routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_RUNNING))
		if err != nil {
			continue
		}
		go s.run(id, &req)
	}
}

func (s *MediaService) insert(ctx context.Context, exec *routingv1.MediaExecution, requestJSON string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO media_executions (execution_id, idempotency_key, status, created_at, request_json) VALUES (?, ?, ?, ?, ?)`, exec.GetExecutionId(), exec.GetIdempotencyKey(), int32(exec.GetStatus()), exec.GetCreatedAt(), requestJSON)
	return err
}

func (s *MediaService) getByIdempotency(ctx context.Context, key string) (*routingv1.MediaExecution, error) {
	row := s.db.QueryRowContext(ctx, mediaSelect+` WHERE idempotency_key = ?`, key)
	return scanMediaExecution(row)
}

func (s *MediaService) get(ctx context.Context, id string) (*routingv1.MediaExecution, error) {
	row := s.db.QueryRowContext(ctx, mediaSelect+` WHERE execution_id = ?`, id)
	return scanMediaExecution(row)
}

func (s *MediaService) now() time.Time {
	if s.clock == nil {
		return time.Now()
	}
	return s.clock().UTC()
}

func validateMediaSubmission(req *routingv1.SubmitMediaRequest) error {
	if req == nil || req.GetRequest() == nil {
		return errors.New("request is required")
	}
	if req.GetRequest().GetKind() != sharedv1.RequestKind_REQUEST_KIND_IMAGE_GENERATION && req.GetRequest().GetKind() != sharedv1.RequestKind_REQUEST_KIND_VIDEO_GENERATION {
		return errors.New("request.kind must be image_generation or video_generation")
	}
	if strings.TrimSpace(req.GetPrompt()) == "" {
		return errors.New("prompt is required")
	}
	if req.GetOutputCount() <= 0 {
		return errors.New("output_count must be greater than zero")
	}
	if strings.TrimSpace(req.GetIdempotencyKey()) == "" {
		return errors.New("idempotency_key is required")
	}
	return nil
}

func terminalMediaStatus(status routingv1.MediaExecutionStatus) bool {
	return status == routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_SUCCEEDED || status == routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_FAILED || status == routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_CANCELLED
}

const mediaSelect = `SELECT execution_id, idempotency_key, status, created_at, started_at, completed_at, route_evidence_json, outputs_json, actual_cost_usd, resolved_model, seed, warnings_json, error_code, error_message FROM media_executions`

type mediaScanner interface{ Scan(...any) error }

func scanMediaExecution(row mediaScanner) (*routingv1.MediaExecution, error) {
	var execution routingv1.MediaExecution
	var status int32
	var evidenceJSON, outputsJSON, warningsJSON string
	if err := row.Scan(&execution.ExecutionId, &execution.IdempotencyKey, &status, &execution.CreatedAt, &execution.StartedAt, &execution.CompletedAt, &evidenceJSON, &outputsJSON, &execution.ActualCostUsd, &execution.ResolvedModel, &execution.Seed, &warningsJSON, &execution.ErrorCode, &execution.ErrorMessage); err != nil {
		return nil, err
	}
	execution.Status = routingv1.MediaExecutionStatus(status)
	if evidenceJSON != "" {
		execution.RouteEvidence = &routingv1.RouteEvidence{}
		_ = protojson.Unmarshal([]byte(evidenceJSON), execution.RouteEvidence)
	}
	if outputsJSON != "" {
		var persisted routingv1.MediaExecution
		if protojson.Unmarshal([]byte(outputsJSON), &persisted) == nil {
			execution.Outputs = persisted.GetOutputs()
		}
	}
	if warningsJSON != "" {
		var persisted routingv1.MediaExecution
		if protojson.Unmarshal([]byte(warningsJSON), &persisted) == nil {
			execution.Warnings = persisted.GetWarnings()
		}
	}
	return &execution, nil
}

func marshalRouteEvidence(evidence *routingv1.RouteEvidence) string {
	if evidence == nil {
		return ""
	}
	encoded, err := protojson.Marshal(evidence)
	if err != nil {
		return ""
	}
	return string(encoded)
}

// marshalStrings uses a tiny generated wrapper so this persistence format stays
// proto-json and remains compatible with the scenario's typed contract tools.
func marshalStrings(values []string) string {
	encoded, err := protojson.Marshal(&routingv1.MediaExecution{Warnings: values})
	if err != nil {
		return ""
	}
	return string(encoded)
}

func validateMediaOutputs(outputs []*routingv1.MediaOutput) error {
	if len(outputs) == 0 {
		return errors.New("media executor returned no outputs")
	}
	for _, output := range outputs {
		if output == nil || strings.TrimSpace(output.GetReference()) == "" || strings.TrimSpace(output.GetMediaType()) == "" {
			return errors.New("media executor returned an output without reference or media_type")
		}
	}
	return nil
}
