// Package handlers provides HTTP handlers for the agent-manager API.
//
// This package is the thin PRESENTATION layer. Handlers are responsible for:
// - HTTP request parsing and validation
// - Response formatting (JSON)
// - Error translation to HTTP status codes
// - Request ID tracking for observability
// - Authentication/authorization checks (when implemented)
//
// Handlers do NOT contain business logic - they delegate to the orchestration layer.
//
// ERROR HANDLING DESIGN:
// All errors are converted to domain.ErrorResponse for consistent API responses.
// Each response includes:
// - code: Machine-readable error identifier
// - message: Technical description
// - userMessage: Human-friendly explanation
// - recovery: Recommended action (retry, fix_input, wait, escalate)
// - retryable: Whether automatic retry may succeed
// - details: Structured context for debugging
// - requestId: Correlation ID for log aggregation
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/permissionpolicy"
	"agent-manager/internal/protoconv"
	"agent-manager/internal/rolepolicy"
	"agent-manager/internal/runreport"
	"agent-manager/internal/storage"

	agentconfig "agent-manager/internal/config"

	"buf.build/go/protovalidate"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/eventbus"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	commonpb "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// Handler provides HTTP handlers for all API endpoints.
type Handler struct {
	svc                   orchestration.HandlerServices
	profiles              orchestration.ProfileService
	hub                   *WebSocketHub
	validator             protovalidate.Validator
	storage               storage.Service
	rolePolicy            *rolepolicy.State
	permissionPolicyState *permissionpolicy.State
	permissionPolicy      *permissionpolicy.Service
	receipts              eventbus.Client
	receiptAvailability   ReceiptAvailabilityReader
}

// ReceiptAvailabilityReader explains an otherwise empty receipt ledger.
// It is deliberately a bounded availability seam: receipt payload disclosure
// remains owned by the observed-receipts endpoint.
type ReceiptAvailabilityReader func(context.Context) runreport.Availability

// HandlerOption configures the Handler.
type HandlerOption func(*Handler)

// WithStorage sets the file storage service for attachment uploads.
func WithStorage(s storage.Service) HandlerOption {
	return func(h *Handler) {
		h.storage = s
	}
}

// WithRolePolicyState installs the sole catalog activation owner used by the
// read-only operator surface and controlled validate/reload commands.
func WithRolePolicyState(state *rolepolicy.State) HandlerOption {
	return func(h *Handler) {
		h.rolePolicy = state
	}
}

// WithPermissionPolicy installs the desired-permission state and service.
// The service owns resource projection and audit evidence; handlers only
// translate the generated API contract.
func WithPermissionPolicy(state *permissionpolicy.State, service *permissionpolicy.Service) HandlerOption {
	return func(h *Handler) {
		h.permissionPolicyState = state
		h.permissionPolicy = service
	}
}

// WithObservedReceipts installs the optional Vrooli Events read client. It is
// deliberately a read-only projection: workflow output and state transitions
// never depend on receipt availability.
func WithObservedReceipts(client eventbus.Client) HandlerOption {
	return func(h *Handler) { h.receipts = client }
}

// WithReceiptAvailabilityReader installs the shared runtime-state reader used
// when Vrooli Events returns no correlated receipts.
func WithReceiptAvailabilityReader(reader ReceiptAvailabilityReader) HandlerOption {
	return func(h *Handler) { h.receiptAvailability = reader }
}

// New creates a new Handler with the given orchestration service.
func New(svc orchestration.HandlerServices, opts ...HandlerOption) *Handler {
	validator, err := protovalidate.New()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize protovalidate: %v", err))
	}
	h := &Handler{svc: svc, profiles: svc.ProfileService, validator: validator}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// SetWebSocketHub sets the WebSocket hub for event broadcasting.
func (h *Handler) SetWebSocketHub(hub *WebSocketHub) {
	h.hub = hub
}

// GetWebSocketHub returns the WebSocket hub for external event broadcasting.
func (h *Handler) GetWebSocketHub() *WebSocketHub {
	return h.hub
}

// RegisterRoutes registers all API routes on the given router.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	// Apply request ID middleware to all routes
	r.Use(requestIDMiddleware)

	// NOTE: Health endpoints (/health and /api/v1/health) are now registered in main.go
	// using api-core/health for standardized response format.

	// WebSocket endpoint (registered separately, no middleware needed)
	if h.hub != nil {
		r.HandleFunc("/api/v1/ws", h.HandleWebSocket(h.hub))
	}

	// Profile endpoints
	r.HandleFunc("/api/v1/profiles", h.CreateProfile).Methods("POST")
	r.HandleFunc("/api/v1/profiles/ensure", h.EnsureProfile).Methods("POST")
	r.HandleFunc("/api/v1/profiles/reconcile-scenario", h.ReconcileScenarioProfiles).Methods("POST")

	// Unified scenario-owned declaration reconcile (profiles + workflows).
	r.HandleFunc("/api/v1/declarations/reconcile-scenario", h.ReconcileScenarioDeclarations).Methods("POST")
	r.HandleFunc("/api/v1/declarations/plan", h.PlanScenarioDeclarations).Methods("POST")
	r.HandleFunc("/api/v1/profiles", h.ListProfiles).Methods("GET")
	r.HandleFunc("/api/v1/profiles/{id}", h.GetProfile).Methods("GET")
	r.HandleFunc("/api/v1/profiles/{id}", h.UpdateProfile).Methods("PUT")
	r.HandleFunc("/api/v1/profiles/{id}", h.DeleteProfile).Methods("DELETE")

	// Scenario-owned immutable workflow catalog.
	r.HandleFunc("/api/v1/workflows/validate", h.ValidateWorkflow).Methods("POST")
	r.HandleFunc("/api/v1/workflows/plan", h.PlanScenarioWorkflows).Methods("POST")
	r.HandleFunc("/api/v1/workflows/reconcile-scenario", h.ReconcileScenarioWorkflows).Methods("POST")
	r.HandleFunc("/api/v1/workflows/reload", h.ReconcileScenarioWorkflows).Methods("POST")
	r.HandleFunc("/api/v1/workflows", h.ListWorkflowRevisions).Methods("GET")
	r.HandleFunc("/api/v1/workflows/revision", h.GetWorkflowRevision).Methods("GET")
	r.HandleFunc("/api/v1/workflows/explain", h.GetWorkflowRevision).Methods("GET")
	r.HandleFunc("/api/v1/workflows/simulate", h.SimulateWorkflow).Methods("POST")
	r.HandleFunc("/api/v1/workflow-executions", h.StartWorkflowExecution).Methods("POST")
	r.HandleFunc("/api/v1/workflow-executions", h.ListWorkflowExecutions).Methods("GET")
	r.HandleFunc("/api/v1/workflow-executions/{id}", h.GetWorkflowExecution).Methods("GET")
	r.HandleFunc("/api/v1/workflow-executions/{id}/result", h.GetWorkflowExecutionResult).Methods("GET")
	r.HandleFunc("/api/v1/workflow-executions/{id}/advance", h.AdvanceWorkflowExecution).Methods("POST")
	r.HandleFunc("/api/v1/workflow-executions/{id}/wait", h.WaitWorkflowExecution).Methods("POST")
	r.HandleFunc("/api/v1/workflow-executions/{id}/trace", h.GetWorkflowExecutionTrace).Methods("GET")
	r.HandleFunc("/api/v1/workflow-executions/{id}/runs", h.ListWorkflowExecutionRuns).Methods("GET")
	r.HandleFunc("/api/v1/workflow-executions/{id}/signals", h.SignalWorkflowExecution).Methods("POST")
	r.HandleFunc("/api/v1/workflow-executions/{id}/cancel", h.CancelWorkflowExecution).Methods("POST")
	r.HandleFunc("/api/v1/workflow-executions/{id}/retry", h.RetryWorkflowExecution).Methods("POST")
	r.HandleFunc("/api/v1/workflow-executions/{id}/resume", h.ResumeWorkflowExecution).Methods("POST")

	// Task endpoints
	r.HandleFunc("/api/v1/tasks", h.CreateTask).Methods("POST")
	r.HandleFunc("/api/v1/tasks", h.ListTasks).Methods("GET")
	r.HandleFunc("/api/v1/tasks/{id}", h.GetTask).Methods("GET")
	r.HandleFunc("/api/v1/tasks/{id}", h.UpdateTask).Methods("PUT")
	r.HandleFunc("/api/v1/tasks/{id}", h.DeleteTask).Methods("DELETE")
	r.HandleFunc("/api/v1/tasks/{id}/cancel", h.CancelTask).Methods("POST")

	// Run endpoints
	r.HandleFunc("/api/v1/runs", h.CreateRun).Methods("POST")
	r.HandleFunc("/api/v1/runs/investigate", h.CreateInvestigationRun).Methods("POST")
	r.HandleFunc("/api/v1/runs/investigation-apply", h.CreateInvestigationApplyRun).Methods("POST")
	r.HandleFunc("/api/v1/runs/resume-from-failed", h.ResumeFromFailedRun).Methods("POST")
	r.HandleFunc("/api/v1/runs", h.ListRuns).Methods("GET")
	r.HandleFunc("/api/v1/runs/stop-all", h.StopAllRuns).Methods("POST")    // Must be before /{id}
	r.HandleFunc("/api/v1/runs/quiesce", h.QuiesceScenario).Methods("POST") // Must be before /{id}
	r.HandleFunc("/api/v1/runs/tag/{tag}", h.GetRunByTag).Methods("GET")
	r.HandleFunc("/api/v1/runs/tag/{tag}/stop", h.StopRunByTag).Methods("POST")
	r.HandleFunc("/api/v1/runs/cohort-report", h.GetCohortReport).Methods("GET") // Must be before /{id}
	r.HandleFunc("/api/v1/runs/goal-cohort", h.GetGoalCohort).Methods("GET")
	r.HandleFunc("/api/v1/runs/invocation-facts/replay", h.ReplayInvocationCorpus).Methods("POST")
	r.HandleFunc("/api/v1/runs/invocation-facts/aggregate", h.AggregateInvocationFacts).Methods("GET")
	r.HandleFunc("/api/v1/runs/invocation-facts/cohort", h.SelectInvocationCohort).Methods("GET")
	r.HandleFunc("/api/v1/runs/cohorts", h.DefineInvocationCohort).Methods("POST")
	r.HandleFunc("/api/v1/runs/cohorts", h.ListInvocationCohorts).Methods("GET")
	r.HandleFunc("/api/v1/runs/cohorts/{name}", h.ShowInvocationCohort).Methods("GET")
	r.HandleFunc("/api/v1/runs/cohorts/{name}", h.DeleteInvocationCohort).Methods("DELETE")
	r.HandleFunc("/api/v1/runs/episode-cohort", h.EpisodeCohort).Methods("GET")
	r.HandleFunc("/api/v1/runs/episode-cohort/compare", h.CompareEpisodeCohorts).Methods("POST")
	r.HandleFunc("/api/v1/runs/episode-trend", h.EpisodeTrend).Methods("GET")
	r.HandleFunc("/api/v1/runs/episodes/publish-recurring", h.PublishRecurringFriction).Methods("POST")
	r.HandleFunc("/api/v1/runs/import-transcript", h.ImportTranscriptHTTP).Methods("POST")
	r.HandleFunc("/api/v1/runs/import-session-corpus", h.ImportSessionCorpus).Methods("POST")
	r.HandleFunc("/api/v1/import/sources", h.ListImportSources).Methods("GET")
	r.HandleFunc("/api/v1/import/sessions", h.ListRunnerSessions).Methods("GET")
	r.HandleFunc("/api/v1/import/sessions", h.ImportRunnerSession).Methods("POST")
	r.HandleFunc("/api/v1/runs/invocation-facts/metrics", h.InvocationMetrics).Methods("GET")
	r.HandleFunc("/api/v1/runs/{id}", h.GetRun).Methods("GET")
	r.HandleFunc("/api/v1/runs/{id}/report", h.GetRunReport).Methods("GET")
	r.HandleFunc("/api/v1/runs/{id}/invocation-facts", h.GetInvocationFacts).Methods("GET")
	r.HandleFunc("/api/v1/runs/{id}/episodes", h.GetEpisodesHTTP).Methods("GET")
	r.HandleFunc("/api/v1/runs/{id}/messages-friction", h.GetMessageFriction).Methods("GET")
	r.HandleFunc("/api/v1/runs/{id}/ledger", h.GetLedger).Methods("GET")
	r.HandleFunc("/api/v1/runs/{id}/invocation-facts/replay", h.ReplayInvocationFacts).Methods("POST")
	r.HandleFunc("/api/v1/runs/{id}/invocation-facts/refresh", h.RefreshInvocationFacts).Methods("POST")
	r.HandleFunc("/api/v1/findings", h.ListFindings).Methods("GET")
	r.HandleFunc("/api/v1/runs/{id}/observed-receipts", h.GetObservedReceipts).Methods("GET")
	r.HandleFunc("/api/v1/runs/{id}/audit-transcript", h.GetAuditTranscript).Methods("GET")
	r.HandleFunc("/api/v1/runs/{id}", h.DeleteRun).Methods("DELETE")
	r.HandleFunc("/api/v1/runs/{id}/stop", h.StopRun).Methods("POST")
	r.HandleFunc("/api/v1/runs/{id}/recover", h.RecoverRun).Methods("POST")
	r.HandleFunc("/api/v1/runs/{id}/continue", h.ContinueRun).Methods("POST")
	r.HandleFunc("/api/v1/runs/{id}/park", h.ParkRun).Methods("POST")
	r.HandleFunc("/api/v1/runs/{id}/await-result", h.GetAwaitResult).Methods("GET")
	r.HandleFunc("/api/v1/runs/{id}/wake", h.WakeRun).Methods("POST")
	r.HandleFunc("/api/v1/runs/{id}/messages/{event_id}/delete", h.DeleteRunMessage).Methods("POST")
	r.HandleFunc("/api/v1/runs/{id}/events", h.GetRunEvents).Methods("GET")
	r.HandleFunc("/api/v1/runs/{id}/diff", h.GetRunDiff).Methods("GET")
	r.HandleFunc("/api/v1/runs/{id}/approve", h.ApproveRun).Methods("POST")
	r.HandleFunc("/api/v1/runs/{id}/reject", h.RejectRun).Methods("POST")
	r.HandleFunc("/api/v1/runs/{id}/partial-approve", h.PartialApproveRun).Methods("POST")
	r.HandleFunc("/api/v1/runs/{id}/sandbox-sync", h.SyncRunFromSandbox).Methods("POST")

	// Status endpoints
	r.HandleFunc("/api/v1/runners", h.GetRunnerStatus).Methods("GET")
	r.HandleFunc("/api/v1/runners/{runner_type}/probe", h.ProbeRunner).Methods("POST")
	r.HandleFunc("/api/v1/role-policy/status", h.GetRolePolicyStatus).Methods("GET")
	r.HandleFunc("/api/v1/role-policy/catalog", h.GetRolePolicyCatalog).Methods("GET")
	r.HandleFunc("/api/v1/role-policy/validate", h.ValidateRolePolicyCatalog).Methods("POST")
	r.HandleFunc("/api/v1/role-policy/reload", h.ReloadRolePolicyCatalog).Methods("POST")
	r.HandleFunc("/api/v1/role-policy/explain", h.ExplainRolePolicy).Methods("POST")
	r.HandleFunc("/api/v1/permission-policy/status", h.GetPermissionPolicyStatus).Methods("GET")
	r.HandleFunc("/api/v1/permission-policy/catalog", h.GetPermissionPolicyCatalog).Methods("GET")
	r.HandleFunc("/api/v1/permission-policy/validate", h.ValidatePermissionPolicyCatalog).Methods("POST")
	r.HandleFunc("/api/v1/permission-policy/reload", h.ReloadPermissionPolicyCatalog).Methods("POST")
	r.HandleFunc("/api/v1/permission-policy/plan", h.PlanPermissionPolicy).Methods("POST")
	r.HandleFunc("/api/v1/permission-policy/reconcile", h.ReconcilePermissionPolicy).Methods("POST")
	r.HandleFunc("/api/v1/permission-policy/doctor", h.DoctorPermissionPolicy).Methods("POST")

	// Path validation (proxied to workspace-sandbox)
	r.HandleFunc("/api/v1/validate-path", h.ValidatePath).Methods("GET")

	// Maintenance endpoints
	r.HandleFunc("/api/v1/maintenance/purge", h.PurgeData).Methods("POST")

	// Investigation Settings endpoints
	r.HandleFunc("/api/v1/investigation-settings", h.GetInvestigationSettings).Methods("GET")
	r.HandleFunc("/api/v1/investigation-settings", h.UpdateInvestigationSettings).Methods("PUT")
	r.HandleFunc("/api/v1/investigation-settings/reset", h.ResetInvestigationSettings).Methods("POST")

	// Orchestration Settings endpoints
	r.HandleFunc("/api/v1/orchestration-settings", h.GetOrchestrationSettings).Methods("GET")
	r.HandleFunc("/api/v1/orchestration-settings", h.UpdateOrchestrationSettings).Methods("PUT")
	r.HandleFunc("/api/v1/orchestration-settings/reset", h.ResetOrchestrationSettings).Methods("POST")

	// Attachment endpoints
	if h.storage != nil {
		r.HandleFunc("/api/v1/attachments/upload", h.UploadAttachment).Methods("POST")
		r.HandleFunc("/api/v1/uploads/{path:.*}", h.ServeUpload).Methods("GET")
	}

	// Identity verification endpoint
	r.HandleFunc("/api/v1/identity/verify", h.VerifyIdentityToken).Methods("POST")
}

// =============================================================================
// MIDDLEWARE
// =============================================================================

// requestIDMiddleware ensures each request has a unique ID for tracing.
func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r)
	})
}

// =============================================================================
// RESPONSE HELPERS
// =============================================================================

// newCreateRunResponse builds the canonical CreateRunResponse with the
// dispatcher snapshot embedded. Centralised so all four run-creation
// handlers (CreateRun, CreateInvestigationRun, ResumeFromFailedRun,
// CreateInvestigationApplyRun) report queue depth identically.
func (h *Handler) newCreateRunResponse(run *domain.Run) *apipb.CreateRunResponse {
	stats := h.svc.SpawnStats()
	return &apipb.CreateRunResponse{
		Run:           protoconv.RunToProto(run),
		QueueDepth:    int32(stats.QueueDepth),
		ActiveCount:   int32(stats.ActiveCount),
		StartingCount: int32(stats.StartingCount),
	}
}

// writeProtoJSON writes a proto message as JSON using protojson.
// This ensures consistent snake_case field names per the proto schema.
func writeProtoJSON(w http.ResponseWriter, status int, msg proto.Message) {
	data, err := protoconv.MarshalJSON(msg)
	if err != nil {
		// A serialization failure is an internal error, never an implicit 200.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("{}"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func (h *Handler) validateProto(w http.ResponseWriter, r *http.Request, msg proto.Message) bool {
	if h.validator == nil {
		return true
	}
	if err := h.validator.Validate(msg); err != nil {
		writeError(w, r, protovalidateToDomainError(err))
		return false
	}
	return true
}

func protovalidateToDomainError(err error) error {
	var valErr *protovalidate.ValidationError
	if errors.As(err, &valErr) {
		if len(valErr.Violations) > 0 {
			violation := valErr.Violations[0]
			field := protovalidate.FieldPathString(violation.Proto.GetField())
			if field == "" {
				field = "body"
			}
			message := violation.Proto.GetMessage()
			if message == "" {
				message = "validation failed"
			}
			return domain.NewValidationError(field, message)
		}
	}
	return domain.NewValidationError("body", "validation failed")
}

// queryFirst returns the first non-empty query value for any of the keys.
func queryFirst(r *http.Request, keys ...string) string {
	query := r.URL.Query()
	for _, key := range keys {
		if value := strings.TrimSpace(query.Get(key)); value != "" {
			return value
		}
	}
	return ""
}

func parseQueryInt(r *http.Request, keys ...string) (int, bool) {
	raw := queryFirst(r, keys...)
	if raw == "" {
		return 0, false
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return value, true
}

func parseQueryIntStrict(r *http.Request, keys ...string) (int, bool, error) {
	raw := queryFirst(r, keys...)
	if raw == "" {
		return 0, false, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, true, err
	}
	return value, true, nil
}

func parseQueryInt64Strict(r *http.Request, keys ...string) (int64, bool, error) {
	raw := queryFirst(r, keys...)
	if raw == "" {
		return 0, false, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, true, err
	}
	return value, true, nil
}

func parseRunnerType(raw string) (domain.RunnerType, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", false
	}
	if numeric, err := strconv.Atoi(value); err == nil {
		parsed := protoconv.RunnerTypeFromProto(domainpb.RunnerType(numeric))
		return parsed, parsed.IsValid()
	}
	normalized := strings.TrimPrefix(strings.ToUpper(value), "RUNNER_TYPE_")
	normalized = strings.ToLower(normalized)
	if strings.Contains(normalized, "_") {
		normalized = strings.ReplaceAll(normalized, "_", "-")
	}
	runnerType := domain.RunnerType(normalized)
	if !runnerType.IsValid() {
		return "", false
	}
	return runnerType, true
}

func parseTaskStatus(raw string) (domain.TaskStatus, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", false
	}
	if numeric, err := strconv.Atoi(value); err == nil {
		parsed := protoconv.TaskStatusFromProto(domainpb.TaskStatus(numeric))
		return parsed, parsed != ""
	}
	normalized := strings.TrimPrefix(strings.ToUpper(value), "TASK_STATUS_")
	status := domain.TaskStatus(strings.ToLower(normalized))
	switch status {
	case domain.TaskStatusQueued,
		domain.TaskStatusRunning,
		domain.TaskStatusNeedsReview,
		domain.TaskStatusApproved,
		domain.TaskStatusRejected,
		domain.TaskStatusFailed,
		domain.TaskStatusCancelled:
		return status, true
	default:
		return "", false
	}
}

func parseRunStatus(raw string) (domain.RunStatus, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", false
	}
	if numeric, err := strconv.Atoi(value); err == nil {
		parsed := protoconv.RunStatusFromProto(domainpb.RunStatus(numeric))
		return parsed, parsed != ""
	}
	normalized := strings.TrimPrefix(strings.ToUpper(value), "RUN_STATUS_")
	status := domain.RunStatus(strings.ToLower(normalized))
	switch status {
	case domain.RunStatusPending,
		domain.RunStatusStarting,
		domain.RunStatusRunning,
		domain.RunStatusNeedsReview,
		domain.RunStatusParked,
		domain.RunStatusComplete,
		domain.RunStatusFailed,
		domain.RunStatusCancelled:
		return status, true
	default:
		return "", false
	}
}

func parseEventTypes(values []string) ([]domain.RunEventType, []string) {
	var types []domain.RunEventType
	var invalid []string
	for _, value := range values {
		for _, raw := range strings.Split(value, ",") {
			trimmed := strings.TrimSpace(raw)
			if trimmed == "" {
				continue
			}
			if strings.HasPrefix(trimmed, "RUN_EVENT_TYPE_") {
				trimmed = strings.ToLower(strings.TrimPrefix(trimmed, "RUN_EVENT_TYPE_"))
			}
			switch domain.RunEventType(trimmed) {
			case domain.EventTypeLog,
				domain.EventTypeMessage,
				domain.EventTypeMessageDeleted,
				domain.EventTypeToolCall,
				domain.EventTypeToolResult,
				domain.EventTypeStatus,
				domain.EventTypeMetric,
				domain.EventTypeArtifact,
				domain.EventTypeError:
				types = append(types, domain.RunEventType(trimmed))
			default:
				invalid = append(invalid, trimmed)
			}
		}
	}
	return types, invalid
}

func healthStatusToProto(status string) commonpb.HealthStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "healthy":
		return commonpb.HealthStatus_HEALTH_STATUS_HEALTHY
	case "degraded":
		return commonpb.HealthStatus_HEALTH_STATUS_DEGRADED
	case "unhealthy":
		return commonpb.HealthStatus_HEALTH_STATUS_UNHEALTHY
	default:
		return commonpb.HealthStatus_HEALTH_STATUS_UNSPECIFIED
	}
}

func dependencyToJsonValue(dep *orchestration.DependencyStatus) *commonpb.JsonValue {
	if dep == nil {
		return nil
	}
	status := "unhealthy"
	if dep.Connected {
		status = "healthy"
	}
	fields := map[string]*commonpb.JsonValue{
		"status": {Kind: &commonpb.JsonValue_StringValue{StringValue: status}},
	}
	if dep.LatencyMs != nil {
		fields["latency_ms"] = &commonpb.JsonValue{Kind: &commonpb.JsonValue_IntValue{IntValue: *dep.LatencyMs}}
	}
	if dep.Error != nil && *dep.Error != "" {
		fields["error"] = &commonpb.JsonValue{Kind: &commonpb.JsonValue_StringValue{StringValue: *dep.Error}}
	}
	if dep.Storage != "" {
		fields["storage"] = &commonpb.JsonValue{Kind: &commonpb.JsonValue_StringValue{StringValue: dep.Storage}}
	}
	return &commonpb.JsonValue{
		Kind: &commonpb.JsonValue_ObjectValue{
			ObjectValue: &commonpb.JsonObject{Fields: fields},
		},
	}
}

// writeError writes a structured error response using domain.ErrorResponse.
// This provides consistent error handling across all endpoints with:
// - Machine-readable error codes
// - User-friendly messages
// - Recovery guidance
// - Request ID for log correlation
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	requestID := w.Header().Get("X-Request-ID")
	if requestID == "" {
		requestID = uuid.New().String()
	}

	// Convert to structured error response
	errResp := domain.ToErrorResponse(err, requestID)

	// Map to HTTP status code
	status := mapErrorCodeToStatus(errResp.Code)

	// Add retry hint header for retryable errors
	if errResp.Retryable {
		w.Header().Set("X-Retryable", "true")
		if errResp.Recovery == domain.RecoveryRetryBackoff {
			w.Header().Set("Retry-After", "5")
		}
	}

	writeProtoJSON(w, status, toProtoErrorResponse(errResp))
}

// writeSimpleError creates a simple validation error for request parsing issues.
func writeSimpleError(w http.ResponseWriter, r *http.Request, field, message string) {
	err := domain.NewValidationError(field, message)
	writeError(w, r, err)
}

func toProtoErrorResponse(errResp domain.ErrorResponse) *commonpb.ErrorResponse {
	details := map[string]*commonpb.JsonValue{}
	for key, value := range errResp.Details {
		if jsonValue := toJsonValue(value); jsonValue != nil {
			details[key] = jsonValue
		}
	}
	if errResp.UserMessage != "" {
		details["user_message"] = &commonpb.JsonValue{
			Kind: &commonpb.JsonValue_StringValue{StringValue: errResp.UserMessage},
		}
	}
	if errResp.Recovery != "" {
		details["recovery"] = &commonpb.JsonValue{
			Kind: &commonpb.JsonValue_StringValue{StringValue: string(errResp.Recovery)},
		}
	}
	details["retryable"] = &commonpb.JsonValue{
		Kind: &commonpb.JsonValue_BoolValue{BoolValue: errResp.Retryable},
	}
	if errResp.RequestID != "" {
		details["request_id"] = &commonpb.JsonValue{
			Kind: &commonpb.JsonValue_StringValue{StringValue: errResp.RequestID},
		}
	}

	var detailsProto *commonpb.JsonObject
	if len(details) > 0 {
		detailsProto = &commonpb.JsonObject{Fields: details}
	}

	return &commonpb.ErrorResponse{
		Code:    string(errResp.Code),
		Message: errResp.Message,
		Details: detailsProto,
	}
}

func toJsonValue(value interface{}) *commonpb.JsonValue {
	switch v := value.(type) {
	case nil:
		return &commonpb.JsonValue{
			Kind: &commonpb.JsonValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE},
		}
	case bool:
		return &commonpb.JsonValue{
			Kind: &commonpb.JsonValue_BoolValue{BoolValue: v},
		}
	case string:
		return &commonpb.JsonValue{
			Kind: &commonpb.JsonValue_StringValue{StringValue: v},
		}
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return &commonpb.JsonValue{
				Kind: &commonpb.JsonValue_IntValue{IntValue: i},
			}
		}
		if f, err := v.Float64(); err == nil {
			return &commonpb.JsonValue{
				Kind: &commonpb.JsonValue_DoubleValue{DoubleValue: f},
			}
		}
		return &commonpb.JsonValue{
			Kind: &commonpb.JsonValue_StringValue{StringValue: v.String()},
		}
	case int:
		return &commonpb.JsonValue{
			Kind: &commonpb.JsonValue_IntValue{IntValue: int64(v)},
		}
	case int8:
		return &commonpb.JsonValue{
			Kind: &commonpb.JsonValue_IntValue{IntValue: int64(v)},
		}
	case int16:
		return &commonpb.JsonValue{
			Kind: &commonpb.JsonValue_IntValue{IntValue: int64(v)},
		}
	case int32:
		return &commonpb.JsonValue{
			Kind: &commonpb.JsonValue_IntValue{IntValue: int64(v)},
		}
	case int64:
		return &commonpb.JsonValue{
			Kind: &commonpb.JsonValue_IntValue{IntValue: v},
		}
	case uint:
		return &commonpb.JsonValue{
			Kind: &commonpb.JsonValue_IntValue{IntValue: int64(v)},
		}
	case uint8:
		return &commonpb.JsonValue{
			Kind: &commonpb.JsonValue_IntValue{IntValue: int64(v)},
		}
	case uint16:
		return &commonpb.JsonValue{
			Kind: &commonpb.JsonValue_IntValue{IntValue: int64(v)},
		}
	case uint32:
		return &commonpb.JsonValue{
			Kind: &commonpb.JsonValue_IntValue{IntValue: int64(v)},
		}
	case uint64:
		return &commonpb.JsonValue{
			Kind: &commonpb.JsonValue_IntValue{IntValue: int64(v)},
		}
	case float32:
		return &commonpb.JsonValue{
			Kind: &commonpb.JsonValue_DoubleValue{DoubleValue: float64(v)},
		}
	case float64:
		return &commonpb.JsonValue{
			Kind: &commonpb.JsonValue_DoubleValue{DoubleValue: v},
		}
	case []string:
		values := make([]*commonpb.JsonValue, 0, len(v))
		for _, item := range v {
			values = append(values, toJsonValue(item))
		}
		return &commonpb.JsonValue{
			Kind: &commonpb.JsonValue_ListValue{ListValue: &commonpb.JsonList{Values: values}},
		}
	case []interface{}:
		values := make([]*commonpb.JsonValue, 0, len(v))
		for _, item := range v {
			values = append(values, toJsonValue(item))
		}
		return &commonpb.JsonValue{
			Kind: &commonpb.JsonValue_ListValue{ListValue: &commonpb.JsonList{Values: values}},
		}
	case map[string]interface{}:
		fields := map[string]*commonpb.JsonValue{}
		for key, item := range v {
			fields[key] = toJsonValue(item)
		}
		return &commonpb.JsonValue{
			Kind: &commonpb.JsonValue_ObjectValue{ObjectValue: &commonpb.JsonObject{Fields: fields}},
		}
	case map[string]string:
		fields := map[string]*commonpb.JsonValue{}
		for key, item := range v {
			fields[key] = toJsonValue(item)
		}
		return &commonpb.JsonValue{
			Kind: &commonpb.JsonValue_ObjectValue{ObjectValue: &commonpb.JsonObject{Fields: fields}},
		}
	default:
		if marshaled, err := json.Marshal(v); err == nil {
			return &commonpb.JsonValue{
				Kind: &commonpb.JsonValue_StringValue{StringValue: string(marshaled)},
			}
		}
		return &commonpb.JsonValue{
			Kind: &commonpb.JsonValue_StringValue{StringValue: fmt.Sprint(v)},
		}
	}
}

// parseUUID extracts and parses a UUID from the request path.
func parseUUID(r *http.Request, param string) (uuid.UUID, error) {
	return uuid.Parse(mux.Vars(r)[param])
}

// mapErrorCodeToStatus maps domain error codes to HTTP status codes.
// This centralizes the error-to-status mapping based on error semantics.
func mapErrorCodeToStatus(code domain.ErrorCode) int {
	category := code.Category()

	switch category {
	case "NOT":
		// NOT_FOUND_* errors
		return http.StatusNotFound

	case "VALIDATION":
		// VALIDATION_* errors
		return http.StatusBadRequest

	case "STATE":
		// STATE_* errors (conflict)
		return http.StatusConflict

	case "POLICY":
		// POLICY_* errors (forbidden)
		return http.StatusForbidden

	case "CAPACITY":
		// CAPACITY_* errors (temporarily unavailable)
		return http.StatusServiceUnavailable

	case "RUNNER":
		// RUNNER_* errors
		if code == domain.ErrCodeRunnerTimeout {
			return http.StatusGatewayTimeout
		}
		if code == domain.ErrCodeRunnerUnavailable {
			return http.StatusServiceUnavailable
		}
		return http.StatusBadGateway

	case "SANDBOX":
		// SANDBOX_* errors
		if strings.Contains(string(code), "CREATE") {
			return http.StatusServiceUnavailable
		}
		return http.StatusBadGateway

	case "DATABASE":
		// DATABASE_* errors
		if code == domain.ErrCodeDatabaseConnection {
			return http.StatusServiceUnavailable
		}
		return http.StatusInternalServerError

	case "CONFIG":
		// CONFIG_* errors
		return http.StatusInternalServerError

	case "INTERNAL":
		// INTERNAL_* errors
		return http.StatusInternalServerError

	default:
		return http.StatusInternalServerError
	}
}

// =============================================================================
// HEALTH HANDLERS
// =============================================================================

// Health returns system health status.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	status, err := h.svc.GetHealth(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	dependencies := map[string]*commonpb.JsonValue{}
	if status.Dependencies != nil {
		if dep := dependencyToJsonValue(status.Dependencies.Database); dep != nil {
			dependencies["database"] = dep
		}
		if dep := dependencyToJsonValue(status.Dependencies.WorkflowRuntime); dep != nil {
			dependencies["workflow_runtime"] = dep
		}
		if dep := dependencyToJsonValue(status.Dependencies.Sandbox); dep != nil {
			dependencies["sandbox"] = dep
		}
		for name, dep := range status.Dependencies.Runners {
			if depValue := dependencyToJsonValue(dep); depValue != nil {
				dependencies["runner_"+name] = depValue
			}
		}
	}

	metrics := map[string]*commonpb.JsonValue{
		"active_runs":          {Kind: &commonpb.JsonValue_IntValue{IntValue: int64(status.ActiveRuns)}},
		"queued_tasks":         {Kind: &commonpb.JsonValue_IntValue{IntValue: int64(status.QueuedTasks)}},
		"default_project_root": {Kind: &commonpb.JsonValue_StringValue{StringValue: h.svc.GetDefaultProjectRoot()}},
	}

	writeProtoJSON(w, http.StatusOK, &commonpb.HealthResponse{
		Status:       healthStatusToProto(status.Status),
		Service:      status.Service,
		Timestamp:    status.Timestamp,
		Readiness:    status.Readiness,
		Dependencies: dependencies,
		Metrics:      metrics,
	})
}

// =============================================================================
// INVESTIGATION SETTINGS HANDLERS
// =============================================================================

// GetInvestigationSettings returns the current investigation settings.
func (h *Handler) GetInvestigationSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.svc.GetInvestigationSettings(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"promptTemplate":            settings.PromptTemplate,
		"applyPromptTemplate":       settings.ApplyPromptTemplate,
		"defaultDepth":              string(settings.DefaultDepth),
		"defaultContext":            settings.DefaultContext,
		"investigationTagAllowlist": settings.InvestigationTagAllowlist,
		"updatedAt":                 settings.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// UpdateInvestigationSettings updates the investigation settings.
func (h *Handler) UpdateInvestigationSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PromptTemplate            string                            `json:"promptTemplate"`
		ApplyPromptTemplate       string                            `json:"applyPromptTemplate"`
		DefaultDepth              string                            `json:"defaultDepth"`
		DefaultContext            *domain.InvestigationContextFlags `json:"defaultContext"`
		InvestigationTagAllowlist *[]domain.InvestigationTagRule    `json:"investigationTagAllowlist"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}

	// Get current settings to merge with updates
	current, err := h.svc.GetInvestigationSettings(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}

	// Apply updates
	if req.PromptTemplate != "" {
		current.PromptTemplate = req.PromptTemplate
	}
	if req.ApplyPromptTemplate != "" {
		current.ApplyPromptTemplate = req.ApplyPromptTemplate
	}
	if req.DefaultDepth != "" {
		depth := domain.InvestigationDepth(req.DefaultDepth)
		if !depth.IsValid() {
			writeSimpleError(w, r, "defaultDepth", "must be 'quick', 'standard', or 'deep'")
			return
		}
		current.DefaultDepth = depth
	}
	if req.DefaultContext != nil {
		current.DefaultContext = *req.DefaultContext
	}
	if req.InvestigationTagAllowlist != nil {
		if err := domain.ValidateInvestigationTagAllowlist(*req.InvestigationTagAllowlist); err != nil {
			writeSimpleError(w, r, "investigationTagAllowlist", err.Error())
			return
		}
		current.InvestigationTagAllowlist = *req.InvestigationTagAllowlist
	}

	if err := h.svc.UpdateInvestigationSettings(r.Context(), current); err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"promptTemplate":            current.PromptTemplate,
		"applyPromptTemplate":       current.ApplyPromptTemplate,
		"defaultDepth":              string(current.DefaultDepth),
		"defaultContext":            current.DefaultContext,
		"investigationTagAllowlist": current.InvestigationTagAllowlist,
		"updatedAt":                 current.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// ResetInvestigationSettings resets the investigation settings to defaults.
func (h *Handler) ResetInvestigationSettings(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.ResetInvestigationSettings(r.Context()); err != nil {
		writeError(w, r, err)
		return
	}

	// Return the default settings
	settings, err := h.svc.GetInvestigationSettings(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"promptTemplate":            settings.PromptTemplate,
		"applyPromptTemplate":       settings.ApplyPromptTemplate,
		"defaultDepth":              string(settings.DefaultDepth),
		"defaultContext":            settings.DefaultContext,
		"investigationTagAllowlist": settings.InvestigationTagAllowlist,
		"updatedAt":                 settings.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// =============================================================================
// ORCHESTRATION SETTINGS
// =============================================================================

// GetOrchestrationSettings returns the current orchestration settings.
func (h *Handler) GetOrchestrationSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.svc.GetOrchestrationSettings(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

// UpdateOrchestrationSettings updates the orchestration settings.
func (h *Handler) UpdateOrchestrationSettings(w http.ResponseWriter, r *http.Request) {
	var settings agentconfig.OrchestrationSettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}

	if err := h.svc.UpdateOrchestrationSettings(r.Context(), &settings); err != nil {
		writeError(w, r, err)
		return
	}

	// Return the updated settings.
	updated, err := h.svc.GetOrchestrationSettings(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// ResetOrchestrationSettings resets the orchestration settings to defaults.
func (h *Handler) ResetOrchestrationSettings(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.ResetOrchestrationSettings(r.Context()); err != nil {
		writeError(w, r, err)
		return
	}

	settings, err := h.svc.GetOrchestrationSettings(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

// =============================================================================
// PATH VALIDATION
// =============================================================================

// ValidatePath proxies path validation to workspace-sandbox.
func (h *Handler) ValidatePath(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		writeSimpleError(w, r, "path", "path query parameter is required")
		return
	}

	projectRoot := r.URL.Query().Get("projectRoot")

	result, err := h.svc.ValidatePath(r.Context(), path, projectRoot)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// VerifyIdentityToken handles POST /api/v1/identity/verify.
func (h *Handler) VerifyIdentityToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON body")
		return
	}
	if req.Token == "" {
		writeSimpleError(w, r, "token", "token is required")
		return
	}

	result, err := h.svc.VerifyIdentityToken(r.Context(), req.Token)
	if err != nil {
		writeError(w, r, err)
		return
	}

	if !result.Valid {
		writeJSON(w, http.StatusUnauthorized, result)
		return
	}

	writeJSON(w, http.StatusOK, result)
}
