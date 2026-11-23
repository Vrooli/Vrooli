package routing

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	mediaflow "ai-gateway/internal/routing/flow"
	"ai-gateway/internal/routing/flow/generated"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var (
	ErrMediaIdempotencyConflict       = errors.New("media idempotency key is bound to a different request")
	ErrMediaOutcomeUnknown            = errors.New("media provider outcome is unknown and cannot be retried")
	ErrMediaRetainedSubmissionInvalid = errors.New("retained media submission cannot be decoded")
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
	ActualCostUSD *float64
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
}

func NewMediaService(db *sql.DB, executor MediaExecutor) *MediaService {
	return &MediaService{db: db, executor: executor, clock: time.Now}
}

func (s *MediaService) Submit(ctx context.Context, req *routingv1.SubmitMediaRequest) (*routingv1.MediaExecution, error) {
	// Retain and dispatch the same owned request, independent of later caller
	// changes. Only the existing key whitespace normalization is permitted.
	if req != nil {
		req = proto.Clone(req).(*routingv1.SubmitMediaRequest)
		req.IdempotencyKey = strings.TrimSpace(req.GetIdempotencyKey())
	}
	if err := s.validateSubmissionStore(req); err != nil {
		return nil, err
	}
	key := strings.TrimSpace(req.GetIdempotencyKey())
	if key != "" {
		if existing, found, err := s.existingMediaSubmission(ctx, key, req); found || err != nil {
			return existing, err
		}
	}
	if err := validateMediaSubmission(req); err != nil {
		return nil, err
	}
	return s.persistAndDispatchMedia(ctx, req, key)
}

func (s *MediaService) validateSubmissionStore(req *routingv1.SubmitMediaRequest) error {
	if s == nil || s.db == nil {
		if err := validateMediaSubmission(req); err != nil {
			return err
		}
		return errors.New("media execution store is not configured")
	}
	return nil
}

func (s *MediaService) existingMediaSubmission(ctx context.Context, key string, req *routingv1.SubmitMediaRequest) (*routingv1.MediaExecution, bool, error) {
	existing, payload, err := s.getByIdempotency(ctx, key)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	result, err := matchMediaSubmission(existing, payload, req)
	return result, true, err
}

func (s *MediaService) persistAndDispatchMedia(ctx context.Context, req *routingv1.SubmitMediaRequest, key string) (*routingv1.MediaExecution, error) {
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
		if existing, found, lookupErr := s.existingMediaSubmission(ctx, key, req); found {
			return existing, lookupErr
		}
		return nil, err
	}

	// Dispatch only after the receipt is durable. The worker owns all later
	// transitions, so a caller can reconnect using execution_id immediately.
	go s.run(context.WithoutCancel(ctx), exec.GetExecutionId(), req)
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
	next, err := mediaflow.TransitionReceipt(exec.GetStatus(), exec.GetErrorCode(), generated.MediaCancel)
	if err != nil {
		return nil, err
	}
	if s.executor != nil {
		if err := s.executor.Cancel(ctx, exec.GetExecutionId()); err != nil {
			return nil, fmt.Errorf("cancel media executor: %w", err)
		}
	}
	now := s.now().Format(time.RFC3339Nano)
	if _, err := s.db.ExecContext(ctx, `UPDATE media_executions SET status = ?, completed_at = ?, error_code = '', error_message = '' WHERE execution_id = ? AND status IN (?, ?)`,
		int32(next.Status), now, exec.GetExecutionId(),
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
	if exec.GetErrorCode() == mediaflow.OutcomeUnknownCode {
		return nil, ErrMediaOutcomeUnknown
	}
	if _, err := mediaflow.TransitionReceipt(exec.GetStatus(), exec.GetErrorCode(), generated.MediaRetry); err != nil {
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

func (s *MediaService) run(ctx context.Context, id string, req *routingv1.SubmitMediaRequest) {
	ctx = context.WithValue(ctx, mediaExecutionIDKey{}, id)
	if !s.claimMediaDispatch(ctx, id) {
		return
	}
	if s.executor == nil {
		s.fail(ctx, id, "executor_unavailable", "no media executor is configured")
		return
	}
	result, err := s.executor.Execute(ctx, req)
	if err != nil {
		s.failWithResult(ctx, id, result, "executor_failed", err.Error())
		if result != nil && result.RouteEvidence != nil {
			s.recordRoute(ctx, result.RouteEvidence)
		} else {
			s.recordRoute(ctx, failedMediaRoute(req, s.now().Format(time.RFC3339Nano)))
		}
		return
	}
	if result == nil {
		s.fail(ctx, id, "executor_protocol", "media executor returned no result")
		return
	}
	if code, message := validateMediaExecutionResult(result); code != "" {
		s.failWithResult(ctx, id, result, code, message)
		return
	}
	s.persistMediaExecutionResult(ctx, id, result)
}

func (s *MediaService) claimMediaDispatch(ctx context.Context, id string) bool {
	started := s.now().Format(time.RFC3339Nano)
	next, err := mediaflow.TransitionReceipt(routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_QUEUED, "", generated.MediaDispatch)
	if err != nil {
		return false
	}
	res, err := s.db.ExecContext(ctx, `UPDATE media_executions SET status = ?, started_at = ? WHERE execution_id = ? AND status = ?`,
		int32(next.Status), started, id,
		int32(routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_QUEUED))
	if err != nil {
		return false
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return false // cancellation won the race.
	}
	return true
}

func validateMediaExecutionResult(result *MediaExecutionResult) (string, string) {
	if result.RouteEvidence == nil {
		return "executor_protocol", "media executor returned no route evidence"
	}
	if !validMediaCost(result.ActualCostUSD) {
		return "executor_protocol", "media executor returned an invalid actual cost"
	}
	if err := validateMediaOutputs(result.Outputs); err != nil {
		return "executor_protocol", err.Error()
	}
	return "", ""
}

func (s *MediaService) persistMediaExecutionResult(ctx context.Context, id string, result *MediaExecutionResult) {
	outputs, err := protojson.Marshal(&routingv1.MediaExecution{Outputs: result.Outputs})
	if err != nil {
		s.failWithResult(ctx, id, result, "receipt_encode_failed", err.Error())
		return
	}
	completed := s.now().Format(time.RFC3339Nano)
	next, err := mediaflow.TransitionReceipt(routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_RUNNING, "", generated.MediaSucceed)
	if err != nil {
		return
	}
	cost, known := mediaCostValue(result.ActualCostUSD)
	_, _ = s.db.ExecContext(ctx, `UPDATE media_executions SET status = ?, completed_at = ?, route_evidence_json = ?, outputs_json = ?, actual_cost_usd = ?, actual_cost_known = ?, resolved_model = ?, seed = ?, warnings_json = ? WHERE execution_id = ? AND status = ?`,
		int32(next.Status), completed,
		marshalRouteEvidence(result.RouteEvidence), string(outputs), cost, known, result.ResolvedModel, result.Seed, marshalStrings(result.Warnings), id,
		int32(routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_RUNNING))
	s.recordRoute(ctx, result.RouteEvidence)
}

// recordRoute adds a media execution to the route log every inference path
// writes, so image generation is counted with its caller like text routes.
// The media receipt stays the execution's own record; a failed log write is
// reported but does not change the receipt.
func (s *MediaService) recordRoute(ctx context.Context, ev *routingv1.RouteEvidence) {
	if ev == nil {
		return
	}
	if err := NewSQLRepository(s.db).Create(ctx, ev); err != nil {
		log.Printf("media route event %s: %v", ev.GetEventId(), err)
	}
}

// failedMediaRoute is the route event for a media execution that failed
// before its executor reported evidence of its own.
func failedMediaRoute(req *routingv1.SubmitMediaRequest, now string) *routingv1.RouteEvidence {
	gateway := req.GetRequest()
	return &routingv1.RouteEvidence{
		EventId: newID("media-route"), RequestId: firstNonEmpty(gateway.GetRequestId(), newID("req")),
		Scenario: strings.TrimSpace(gateway.GetScenario()), Operation: strings.TrimSpace(gateway.GetOperation()),
		Role: strings.TrimSpace(gateway.GetRole()), TaskRole: strings.TrimSpace(firstNonEmpty(gateway.GetTaskRole(), gateway.GetRole())),
		Profile: gateway.GetProfile(), PrivacyClass: gateway.GetPrivacyClass(), PrivacyProvenance: strings.TrimSpace(gateway.GetPrivacyProvenance()),
		Status: "failed", FailureReasons: []string{failureReasonExecute}, CreatedAt: now,
	}
}

func (s *MediaService) fail(ctx context.Context, id, code, message string) {
	_ = s.failFrom(ctx, id, routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_RUNNING, code, message)
}

func (s *MediaService) failWithResult(ctx context.Context, id string, result *MediaExecutionResult, code, message string) {
	from := routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_RUNNING
	next, err := mediaflow.TransitionReceipt(from, "", generated.MediaFail)
	if err != nil {
		return
	}
	cost, known := 0.0, 0
	if result != nil && validMediaCost(result.ActualCostUSD) && result.ActualCostUSD != nil {
		cost, known = *result.ActualCostUSD, 1
	}
	_, _ = s.db.ExecContext(ctx, `UPDATE media_executions SET status = ?, completed_at = ?, error_code = ?, error_message = ?, actual_cost_usd = ?, actual_cost_known = ? WHERE execution_id = ? AND status = ?`,
		int32(next.Status), s.now().Format(time.RFC3339Nano), code, message, cost, known, id, int32(from))
}

func (s *MediaService) failFrom(ctx context.Context, id string, from routingv1.MediaExecutionStatus, code, message string) error {
	next, err := mediaflow.TransitionReceipt(from, "", generated.MediaFail)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `UPDATE media_executions SET status = ?, completed_at = ?, error_code = ?, error_message = ? WHERE execution_id = ? AND status = ?`, int32(next.Status), s.now().Format(time.RFC3339Nano), code, message, id, int32(from))
	return err
}

// Recover is called once at startup before routes are mounted. RUNNING work
// may already have completed at its provider; quarantine it before reading any
// retained payload. Only work that never crossed the dispatch CAS can resume.
func (s *MediaService) Recover(ctx context.Context) error {
	if s == nil || s.db == nil {
		return nil
	}
	if err := s.quarantineInterruptedMedia(ctx); err != nil {
		return err
	}
	pending, err := s.queuedMediaReceipts(ctx)
	if err != nil {
		return err
	}
	return s.resumeQueuedMedia(ctx, pending)
}

func (s *MediaService) quarantineInterruptedMedia(ctx context.Context) error {
	unknown, err := mediaflow.TransitionReceipt(routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_RUNNING, "", generated.MediaRestart)
	if err != nil {
		return err
	}
	if _, err = s.db.ExecContext(ctx, `UPDATE media_executions SET status = ?, completed_at = ?, error_code = ?, error_message = ? WHERE status = ?`, int32(unknown.Status), s.now().Format(time.RFC3339Nano), unknown.ErrorCode, "provider outcome unknown; not replayed; cost unknown", int32(routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_RUNNING)); err != nil {
		return fmt.Errorf("quarantine interrupted media executions: %w", err)
	}
	return nil
}

type pendingMediaReceipt struct{ id, payload string }

func (s *MediaService) queuedMediaReceipts(ctx context.Context) ([]pendingMediaReceipt, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT execution_id, request_json FROM media_executions WHERE status = ?`, int32(routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_QUEUED))
	if err != nil {
		return nil, fmt.Errorf("read queued media executions: %w", err)
	}
	defer rows.Close()
	pending := make([]pendingMediaReceipt, 0)
	for rows.Next() {
		var id, payload string
		if err := rows.Scan(&id, &payload); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("read queued media execution: %w", err)
		}
		pending = append(pending, pendingMediaReceipt{id: id, payload: payload})
	}
	if err := errors.Join(rows.Err(), rows.Close()); err != nil {
		return nil, fmt.Errorf("read queued media executions: %w", err)
	}
	return pending, nil
}

func (s *MediaService) resumeQueuedMedia(ctx context.Context, pending []pendingMediaReceipt) error {
	var recoveryErrors []error
	for _, receipt := range pending {
		id, payload := receipt.id, receipt.payload
		var req routingv1.SubmitMediaRequest
		if err := protojson.Unmarshal([]byte(payload), &req); err != nil {
			recoveryErrors = append(recoveryErrors, s.failFrom(ctx, id, routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_QUEUED, "receipt_decode_failed", "stored media submission could not be decoded"))
			continue
		}
		if err := validateMediaSubmission(&req); err != nil {
			recoveryErrors = append(recoveryErrors, s.failFrom(ctx, id, routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_QUEUED, "receipt_invalid", "stored media submission is invalid"))
			continue
		}
		go s.run(context.WithoutCancel(ctx), id, &req)
	}
	return errors.Join(recoveryErrors...)
}

func (s *MediaService) insert(ctx context.Context, exec *routingv1.MediaExecution, requestJSON string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO media_executions (execution_id, idempotency_key, status, created_at, request_json) VALUES (?, ?, ?, ?, ?)`, exec.GetExecutionId(), exec.GetIdempotencyKey(), int32(exec.GetStatus()), exec.GetCreatedAt(), requestJSON)
	return err
}

func (s *MediaService) getByIdempotency(ctx context.Context, key string) (*routingv1.MediaExecution, string, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+mediaColumns+`, request_json FROM media_executions WHERE idempotency_key = ?`, key)
	var payload string
	execution, err := scanMediaExecutionAndRequest(row, &payload)
	return execution, payload, err
}

func matchMediaSubmission(existing *routingv1.MediaExecution, payload string, req *routingv1.SubmitMediaRequest) (*routingv1.MediaExecution, error) {
	var retained routingv1.SubmitMediaRequest
	if err := protojson.Unmarshal([]byte(payload), &retained); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMediaRetainedSubmissionInvalid, err)
	}
	retained.IdempotencyKey = strings.TrimSpace(retained.GetIdempotencyKey())
	if !proto.Equal(&retained, req) {
		return nil, ErrMediaIdempotencyConflict
	}
	return existing, nil
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
	if strings.TrimSpace(req.GetOutputReference()) == "" {
		return errors.New("output_reference is required")
	}
	return nil
}

func terminalMediaStatus(status routingv1.MediaExecutionStatus) bool {
	return status == routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_SUCCEEDED || status == routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_FAILED || status == routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_CANCELLED
}

const (
	mediaColumns = `execution_id, idempotency_key, status, created_at, started_at, completed_at, route_evidence_json, outputs_json, actual_cost_usd, actual_cost_known, resolved_model, seed, warnings_json, error_code, error_message`
	mediaSelect  = `SELECT ` + mediaColumns + ` FROM media_executions`
)

type mediaScanner interface{ Scan(...any) error }

func scanMediaExecution(row mediaScanner) (*routingv1.MediaExecution, error) {
	return scanMediaExecutionAndRequest(row, nil)
}

func scanMediaExecutionAndRequest(row mediaScanner, payload *string) (*routingv1.MediaExecution, error) {
	var execution routingv1.MediaExecution
	var status int32
	var evidenceJSON, outputsJSON, warningsJSON string
	var actualCost float64
	var actualCostKnown int
	values := []any{&execution.ExecutionId, &execution.IdempotencyKey, &status, &execution.CreatedAt, &execution.StartedAt, &execution.CompletedAt, &evidenceJSON, &outputsJSON, &actualCost, &actualCostKnown, &execution.ResolvedModel, &execution.Seed, &warningsJSON, &execution.ErrorCode, &execution.ErrorMessage}
	if payload != nil {
		values = append(values, payload)
	}
	if err := row.Scan(values...); err != nil {
		return nil, err
	}
	execution.Status = routingv1.MediaExecutionStatus(status)
	if actualCostKnown != 0 {
		value := actualCost
		execution.ActualCostUsd = &value
	}
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

func validMediaCost(cost *float64) bool {
	return cost == nil || (!math.IsNaN(*cost) && !math.IsInf(*cost, 0) && *cost >= 0)
}

func mediaCostValue(cost *float64) (float64, int) {
	if cost == nil {
		return 0, 0
	}
	return *cost, 1
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
