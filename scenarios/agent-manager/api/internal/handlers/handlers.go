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
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"
	"agent-manager/internal/modelregistry"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/protoconv"

	"buf.build/go/protovalidate"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	commonpb "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// Handler provides HTTP handlers for all API endpoints.
type Handler struct {
	svc       orchestration.Service
	hub       *WebSocketHub
	validator protovalidate.Validator
}

// New creates a new Handler with the given orchestration service.
func New(svc orchestration.Service) *Handler {
	validator, err := protovalidate.New()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize protovalidate: %v", err))
	}
	return &Handler{svc: svc, validator: validator}
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
	r.HandleFunc("/api/v1/profiles", h.ListProfiles).Methods("GET")
	r.HandleFunc("/api/v1/profiles/{id}", h.GetProfile).Methods("GET")
	r.HandleFunc("/api/v1/profiles/{id}", h.UpdateProfile).Methods("PUT")
	r.HandleFunc("/api/v1/profiles/{id}", h.DeleteProfile).Methods("DELETE")

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
	r.HandleFunc("/api/v1/runs", h.ListRuns).Methods("GET")
	r.HandleFunc("/api/v1/runs/stop-all", h.StopAllRuns).Methods("POST") // Must be before /{id}
	r.HandleFunc("/api/v1/runs/tag/{tag}", h.GetRunByTag).Methods("GET")
	r.HandleFunc("/api/v1/runs/tag/{tag}/stop", h.StopRunByTag).Methods("POST")
	r.HandleFunc("/api/v1/runs/{id}", h.GetRun).Methods("GET")
	r.HandleFunc("/api/v1/runs/{id}", h.DeleteRun).Methods("DELETE")
	r.HandleFunc("/api/v1/runs/{id}/stop", h.StopRun).Methods("POST")
	r.HandleFunc("/api/v1/runs/{id}/continue", h.ContinueRun).Methods("POST")
	r.HandleFunc("/api/v1/runs/{id}/events", h.GetRunEvents).Methods("GET")
	r.HandleFunc("/api/v1/runs/{id}/diff", h.GetRunDiff).Methods("GET")
	r.HandleFunc("/api/v1/runs/{id}/approve", h.ApproveRun).Methods("POST")
	r.HandleFunc("/api/v1/runs/{id}/reject", h.RejectRun).Methods("POST")

	// Status endpoints
	r.HandleFunc("/api/v1/runners", h.GetRunnerStatus).Methods("GET")
	r.HandleFunc("/api/v1/runners/{runner_type}/probe", h.ProbeRunner).Methods("POST")
	r.HandleFunc("/api/v1/runner-models", h.GetRunnerModels).Methods("GET")
	r.HandleFunc("/api/v1/runner-models", h.UpdateRunnerModels).Methods("PUT")

	// Maintenance endpoints
	r.HandleFunc("/api/v1/maintenance/purge", h.PurgeData).Methods("POST")

	// Investigation Settings endpoints
	r.HandleFunc("/api/v1/investigation-settings", h.GetInvestigationSettings).Methods("GET")
	r.HandleFunc("/api/v1/investigation-settings", h.UpdateInvestigationSettings).Methods("PUT")
	r.HandleFunc("/api/v1/investigation-settings/reset", h.ResetInvestigationSettings).Methods("POST")
	r.HandleFunc("/api/v1/runs/detect-scenarios", h.DetectScenariosForRuns).Methods("POST")
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

// writeProtoJSON writes a proto message as JSON using protojson.
// This ensures consistent snake_case field names per the proto schema.
func writeProtoJSON(w http.ResponseWriter, status int, msg proto.Message) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	data, err := protoconv.MarshalJSON(msg)
	if err != nil {
		// Fallback to empty object on marshal error
		_, _ = w.Write([]byte("{}"))
		return
	}
	_, _ = w.Write(data)
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
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

func parseQueryInt64(r *http.Request, keys ...string) (int64, bool) {
	raw := queryFirst(r, keys...)
	if raw == "" {
		return 0, false
	}
	value, err := strconv.ParseInt(raw, 10, 64)
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
	normalized := strings.ToUpper(value)
	if strings.HasPrefix(normalized, "RUNNER_TYPE_") {
		normalized = strings.TrimPrefix(normalized, "RUNNER_TYPE_")
	}
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
	normalized := strings.ToUpper(value)
	if strings.HasPrefix(normalized, "TASK_STATUS_") {
		normalized = strings.TrimPrefix(normalized, "TASK_STATUS_")
	}
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
	normalized := strings.ToUpper(value)
	if strings.HasPrefix(normalized, "RUN_STATUS_") {
		normalized = strings.TrimPrefix(normalized, "RUN_STATUS_")
	}
	status := domain.RunStatus(strings.ToLower(normalized))
	switch status {
	case domain.RunStatusPending,
		domain.RunStatusStarting,
		domain.RunStatusRunning,
		domain.RunStatusNeedsReview,
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
		"active_runs":  {Kind: &commonpb.JsonValue_IntValue{IntValue: int64(status.ActiveRuns)}},
		"queued_tasks": {Kind: &commonpb.JsonValue_IntValue{IntValue: int64(status.QueuedTasks)}},
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
// PROFILE HANDLERS
// =============================================================================

// CreateProfile creates a new agent profile.
func (h *Handler) CreateProfile(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var req apipb.CreateProfileRequest
	if err := protoconv.UnmarshalJSON(body, &req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}
	if !h.validateProto(w, r, &req) {
		return
	}
	if req.Profile == nil {
		writeSimpleError(w, r, "profile", "profile is required")
		return
	}

	profile := protoconv.AgentProfileFromProto(req.Profile)

	// Validate before sending to service
	if err := profile.Validate(); err != nil {
		writeError(w, r, err)
		return
	}

	result, err := h.svc.CreateProfile(r.Context(), profile)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusCreated, &apipb.CreateProfileResponse{
		Profile: protoconv.AgentProfileToProto(result),
	})
}

// EnsureProfile resolves a profile by key, creating it with defaults if needed.
func (h *Handler) EnsureProfile(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var req apipb.EnsureProfileRequest
	if err := protoconv.UnmarshalJSON(body, &req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}
	if !h.validateProto(w, r, &req) {
		return
	}

	result, err := h.svc.EnsureProfile(r.Context(), orchestration.EnsureProfileRequest{
		ProfileKey:     req.ProfileKey,
		Defaults:       protoconv.AgentProfileFromProto(req.Defaults),
		UpdateExisting: req.UpdateExisting,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.EnsureProfileResponse{
		Profile: protoconv.AgentProfileToProto(result.Profile),
		Created: result.Created,
		Updated: result.Updated,
	})
}

// GetProfile retrieves a profile by ID.
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	req := apipb.GetProfileRequest{ProfileId: idStr}
	if !h.validateProto(w, r, &req) {
		return
	}
	id, err := uuid.Parse(req.ProfileId)
	if err != nil {
		writeSimpleError(w, r, "profile_id", "invalid UUID format for profile ID")
		return
	}

	profile, err := h.svc.GetProfile(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	availableModels, modelPresets := h.resolveProfileModels(r.Context(), profile)
	writeProtoJSON(w, http.StatusOK, &apipb.GetProfileResponse{
		Profile:         protoconv.AgentProfileToProto(profile),
		AvailableModels: availableModels,
		ModelPresets:    modelPresets,
	})
}

// ListProfiles returns all agent profiles.
func (h *Handler) ListProfiles(w http.ResponseWriter, r *http.Request) {
	limit, limitProvided, err := parseQueryIntStrict(r, "limit")
	if err != nil {
		writeSimpleError(w, r, "limit", "must be a number")
		return
	}
	offset, offsetProvided, err := parseQueryIntStrict(r, "offset")
	if err != nil {
		writeSimpleError(w, r, "offset", "must be a number")
		return
	}
	runnerTypeRaw := queryFirst(r, "runner_type", "runnerType")
	var runnerTypeFilter *domain.RunnerType
	var runnerTypeProto *domainpb.RunnerType
	if runnerTypeRaw != "" {
		if parsed, ok := parseRunnerType(runnerTypeRaw); ok {
			runnerTypeFilter = &parsed
			converted := protoconv.RunnerTypeToProto(parsed)
			runnerTypeProto = &converted
		} else {
			writeSimpleError(w, r, "runner_type", "invalid runner type")
			return
		}
	}

	req := apipb.ListProfilesRequest{}
	if limitProvided {
		value := int32(limit)
		req.Limit = &value
	}
	if offsetProvided {
		value := int32(offset)
		req.Offset = &value
	}
	if runnerTypeProto != nil {
		req.RunnerType = runnerTypeProto
	}
	if !h.validateProto(w, r, &req) {
		return
	}

	opts := orchestration.ListOptions{}
	if req.Limit != nil {
		opts.Limit = int(req.GetLimit())
	}
	if req.Offset != nil {
		opts.Offset = int(req.GetOffset())
	}
	profiles, err := h.svc.ListProfiles(r.Context(), orchestration.ListOptions{
		Limit:  opts.Limit,
		Offset: opts.Offset,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	if runnerTypeFilter != nil {
		filtered := make([]*domain.AgentProfile, 0, len(profiles))
		for _, profile := range profiles {
			if profile.RunnerType == *runnerTypeFilter {
				filtered = append(filtered, profile)
			}
		}
		profiles = filtered
	}

	writeProtoJSON(w, http.StatusOK, &apipb.ListProfilesResponse{
		Profiles: protoconv.AgentProfilesToProto(profiles),
		Total:    int32(len(profiles)),
	})
}

// UpdateProfile updates an existing profile.
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for profile ID")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var req apipb.UpdateProfileRequest
	if err := protoconv.UnmarshalJSON(body, &req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}
	if !h.validateProto(w, r, &req) {
		return
	}
	if req.Profile == nil {
		writeSimpleError(w, r, "profile", "profile is required")
		return
	}
	if req.ProfileId != "" {
		if req.ProfileId != id.String() {
			writeSimpleError(w, r, "profile_id", "profile_id does not match URL")
			return
		}
	}

	profile := protoconv.AgentProfileFromProto(req.Profile)
	profile.ID = id

	// Validate before sending to service
	if err := profile.Validate(); err != nil {
		writeError(w, r, err)
		return
	}

	result, err := h.svc.UpdateProfile(r.Context(), profile)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.UpdateProfileResponse{
		Profile: protoconv.AgentProfileToProto(result),
	})
}

// DeleteProfile removes a profile.
func (h *Handler) DeleteProfile(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	req := apipb.DeleteProfileRequest{ProfileId: idStr}
	if !h.validateProto(w, r, &req) {
		return
	}
	id, err := uuid.Parse(req.ProfileId)
	if err != nil {
		writeSimpleError(w, r, "profile_id", "invalid UUID format for profile ID")
		return
	}

	if err := h.svc.DeleteProfile(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.DeleteProfileResponse{Success: true})
}

// =============================================================================
// TASK HANDLERS
// =============================================================================

// CreateTask creates a new task.
func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var req apipb.CreateTaskRequest
	if err := protoconv.UnmarshalJSON(body, &req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}
	if !h.validateProto(w, r, &req) {
		return
	}
	if req.Task == nil {
		writeSimpleError(w, r, "task", "task is required")
		return
	}

	task := protoconv.TaskFromProto(req.Task)

	// Validate before sending to service
	if err := task.Validate(); err != nil {
		writeError(w, r, err)
		return
	}

	result, err := h.svc.CreateTask(r.Context(), task)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusCreated, &apipb.CreateTaskResponse{
		Task: protoconv.TaskToProto(result),
	})
}

// GetTask retrieves a task by ID.
func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	req := apipb.GetTaskRequest{TaskId: idStr}
	if !h.validateProto(w, r, &req) {
		return
	}
	id, err := uuid.Parse(req.TaskId)
	if err != nil {
		writeSimpleError(w, r, "task_id", "invalid UUID format for task ID")
		return
	}

	task, err := h.svc.GetTask(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.GetTaskResponse{
		Task: protoconv.TaskToProto(task),
	})
}

// ListTasks returns all tasks.
func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	limit, limitProvided, err := parseQueryIntStrict(r, "limit")
	if err != nil {
		writeSimpleError(w, r, "limit", "must be a number")
		return
	}
	offset, offsetProvided, err := parseQueryIntStrict(r, "offset")
	if err != nil {
		writeSimpleError(w, r, "offset", "must be a number")
		return
	}
	statusRaw := queryFirst(r, "status")
	scopePrefix := queryFirst(r, "scope_prefix", "scopePrefix")

	var statusFilter *domain.TaskStatus
	var statusProto *domainpb.TaskStatus
	if statusRaw != "" {
		if parsed, ok := parseTaskStatus(statusRaw); ok {
			statusFilter = &parsed
			converted := protoconv.TaskStatusToProto(parsed)
			statusProto = &converted
		} else {
			writeSimpleError(w, r, "status", "invalid task status")
			return
		}
	}

	req := apipb.ListTasksRequest{}
	if statusProto != nil {
		req.Status = statusProto
	}
	if scopePrefix != "" {
		req.ScopePrefix = &scopePrefix
	}
	if limitProvided {
		value := int32(limit)
		req.Limit = &value
	}
	if offsetProvided {
		value := int32(offset)
		req.Offset = &value
	}
	if !h.validateProto(w, r, &req) {
		return
	}

	opts := orchestration.ListOptions{}
	if req.Limit != nil {
		opts.Limit = int(req.GetLimit())
	}
	if req.Offset != nil {
		opts.Offset = int(req.GetOffset())
	}
	tasks, err := h.svc.ListTasks(r.Context(), orchestration.ListOptions{
		Limit:  opts.Limit,
		Offset: opts.Offset,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	if statusFilter != nil || scopePrefix != "" {
		filtered := make([]*domain.Task, 0, len(tasks))
		for _, task := range tasks {
			if statusFilter != nil && task.Status != *statusFilter {
				continue
			}
			if scopePrefix != "" && !strings.HasPrefix(task.ScopePath, scopePrefix) {
				continue
			}
			filtered = append(filtered, task)
		}
		tasks = filtered
	}

	writeProtoJSON(w, http.StatusOK, &apipb.ListTasksResponse{
		Tasks: protoconv.TasksToProto(tasks),
		Total: int32(len(tasks)),
	})
}

// UpdateTask updates an existing task.
func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for task ID")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var req apipb.UpdateTaskRequest
	if err := protoconv.UnmarshalJSON(body, &req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}
	if !h.validateProto(w, r, &req) {
		return
	}
	if req.Task == nil {
		writeSimpleError(w, r, "task", "task is required")
		return
	}
	if req.TaskId != "" {
		if req.TaskId != id.String() {
			writeSimpleError(w, r, "task_id", "task_id does not match URL")
			return
		}
	}

	task := protoconv.TaskFromProto(req.Task)
	task.ID = id

	// Validate before sending to service
	if err := task.Validate(); err != nil {
		writeError(w, r, err)
		return
	}

	result, err := h.svc.UpdateTask(r.Context(), task)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.UpdateTaskResponse{
		Task: protoconv.TaskToProto(result),
	})
}

// CancelTask cancels a queued or running task.
func (h *Handler) CancelTask(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	req := apipb.CancelTaskRequest{TaskId: idStr}
	if !h.validateProto(w, r, &req) {
		return
	}
	id, err := uuid.Parse(req.TaskId)
	if err != nil {
		writeSimpleError(w, r, "task_id", "invalid UUID format for task ID")
		return
	}

	if err := h.svc.CancelTask(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.CancelTaskResponse{Success: true, Status: "cancelled"})
}

// DeleteTask permanently removes a cancelled task.
func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	req := apipb.DeleteTaskRequest{TaskId: idStr}
	if !h.validateProto(w, r, &req) {
		return
	}
	id, err := uuid.Parse(req.TaskId)
	if err != nil {
		writeSimpleError(w, r, "task_id", "invalid UUID format for task ID")
		return
	}

	if err := h.svc.DeleteTask(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.DeleteTaskResponse{Success: true})
}

// =============================================================================
// RUN HANDLERS
// =============================================================================

// CreateRun creates a new run.
func (h *Handler) CreateRun(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var protoReq apipb.CreateRunRequest
	if err := protoconv.UnmarshalJSON(body, &protoReq); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}
	if !h.validateProto(w, r, &protoReq) {
		return
	}
	if protoReq.TaskId == "" {
		writeSimpleError(w, r, "task_id", "task_id is required")
		return
	}

	taskID, err := uuid.Parse(protoReq.TaskId)
	if err != nil {
		writeSimpleError(w, r, "task_id", "invalid UUID format for task ID")
		return
	}

	req := orchestration.CreateRunRequest{
		TaskID: taskID,
		Force:  protoReq.Force,
	}
	if protoReq.AgentProfileId != nil {
		agentProfileID, err := uuid.Parse(protoReq.GetAgentProfileId())
		if err != nil {
			writeSimpleError(w, r, "agent_profile_id", "invalid UUID format for agent profile ID")
			return
		}
		req.AgentProfileID = &agentProfileID
	}
	if protoReq.Tag != nil {
		req.Tag = protoReq.GetTag()
	}
	if protoReq.RunMode != nil {
		mode := protoconv.RunModeFromProto(*protoReq.RunMode)
		req.RunMode = &mode
	}
	if protoReq.IdempotencyKey != nil {
		req.IdempotencyKey = protoReq.GetIdempotencyKey()
	}
	if protoReq.Prompt != nil {
		req.Prompt = protoReq.GetPrompt()
	}
	if protoReq.ExistingSandboxId != nil {
		existingSandboxID, err := uuid.Parse(protoReq.GetExistingSandboxId())
		if err != nil {
			writeSimpleError(w, r, "existing_sandbox_id", "invalid UUID format for existing sandbox ID")
			return
		}
		req.ExistingSandboxID = &existingSandboxID
	}
	if protoReq.ProfileRef != nil {
		req.ProfileRef = &orchestration.ProfileRef{
			ProfileKey: protoReq.ProfileRef.ProfileKey,
			Defaults:   protoconv.AgentProfileFromProto(protoReq.ProfileRef.Defaults),
		}
	}
	if protoReq.InlineConfig != nil {
		inline := protoReq.InlineConfig
		if inline.RunnerType != nil && *inline.RunnerType != domainpb.RunnerType_RUNNER_TYPE_UNSPECIFIED {
			runner := protoconv.RunnerTypeFromProto(*inline.RunnerType)
			req.RunnerType = &runner
		}
		if inline.Model != nil {
			model := inline.GetModel()
			req.Model = &model
		}
		if inline.ModelPreset != nil {
			preset := protoconv.ModelPresetFromProto(*inline.ModelPreset)
			req.ModelPreset = &preset
		}
		if inline.MaxTurns != nil {
			maxTurns := int(inline.GetMaxTurns())
			req.MaxTurns = &maxTurns
		}
		if inline.Timeout != nil {
			timeout := inline.Timeout.AsDuration()
			req.Timeout = &timeout
		}
		if len(inline.FallbackRunnerTypes) > 0 || inline.ClearFallbackRunnerTypes {
			fallback := make([]domain.RunnerType, 0, len(inline.FallbackRunnerTypes))
			for _, rt := range inline.FallbackRunnerTypes {
				if rt == domainpb.RunnerType_RUNNER_TYPE_UNSPECIFIED {
					continue
				}
				fallback = append(fallback, protoconv.RunnerTypeFromProto(rt))
			}
			req.FallbackRunnerTypes = fallback
		}
		if len(inline.AllowedTools) > 0 || inline.ClearAllowedTools {
			req.AllowedTools = inline.AllowedTools
		}
		if len(inline.DeniedTools) > 0 || inline.ClearDeniedTools {
			req.DeniedTools = inline.DeniedTools
		}
		if inline.SkipPermissionPrompt != nil {
			skipPermissionPrompt := inline.GetSkipPermissionPrompt()
			req.SkipPermissionPrompt = &skipPermissionPrompt
		}
		if inline.RequiresSandbox != nil {
			requiresSandbox := inline.GetRequiresSandbox()
			req.RequiresSandbox = &requiresSandbox
		}
		if inline.RequiresApproval != nil {
			requiresApproval := inline.GetRequiresApproval()
			req.RequiresApproval = &requiresApproval
		}
		if inline.SandboxConfig != nil {
			req.SandboxConfig = protoconv.SandboxConfigFromProto(inline.SandboxConfig)
		}
		if len(inline.AllowedPaths) > 0 || inline.ClearAllowedPaths {
			req.AllowedPaths = inline.AllowedPaths
		}
		if len(inline.DeniedPaths) > 0 || inline.ClearDeniedPaths {
			req.DeniedPaths = inline.DeniedPaths
		}
	}

	run, err := h.svc.CreateRun(r.Context(), req)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusCreated, &apipb.CreateRunResponse{
		Run: protoconv.RunToProto(run),
	})
}

// CreateInvestigationRun creates a new investigation run for specified run IDs.
func (h *Handler) CreateInvestigationRun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RunIDs        []string `json:"runIds"`
		CustomContext string   `json:"customContext,omitempty"`
		Depth         string   `json:"depth,omitempty"` // quick, standard, or deep
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON")
		return
	}

	runIDs := make([]uuid.UUID, 0, len(req.RunIDs))
	for _, idStr := range req.RunIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			writeSimpleError(w, r, "runIds", "invalid UUID format: "+idStr)
			return
		}
		runIDs = append(runIDs, id)
	}

	// Validate depth if provided
	depth := domain.InvestigationDepth(req.Depth)
	if !depth.IsValid() {
		writeSimpleError(w, r, "depth", "must be 'quick', 'standard', or 'deep'")
		return
	}

	run, err := h.svc.CreateInvestigationRun(r.Context(), orchestration.CreateInvestigationRequest{
		RunIDs:        runIDs,
		CustomContext: req.CustomContext,
		Depth:         depth,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusCreated, &apipb.CreateRunResponse{
		Run: protoconv.RunToProto(run),
	})
}

// CreateInvestigationApplyRun creates a new run to apply investigation recommendations.
func (h *Handler) CreateInvestigationApplyRun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InvestigationRunID string `json:"investigationRunId"`
		CustomContext      string `json:"customContext,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON")
		return
	}

	runID, err := uuid.Parse(req.InvestigationRunID)
	if err != nil {
		writeSimpleError(w, r, "investigationRunId", "invalid UUID format")
		return
	}

	run, err := h.svc.CreateInvestigationApplyRun(r.Context(), runID, req.CustomContext)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusCreated, &apipb.CreateRunResponse{
		Run: protoconv.RunToProto(run),
	})
}

// GetRun retrieves a run by ID.
func (h *Handler) GetRun(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	req := apipb.GetRunRequest{RunId: idStr}
	if !h.validateProto(w, r, &req) {
		return
	}
	id, err := uuid.Parse(req.RunId)
	if err != nil {
		writeSimpleError(w, r, "run_id", "invalid UUID format for run ID")
		return
	}

	run, err := h.svc.GetRun(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.GetRunResponse{
		Run: protoconv.RunToProto(run),
	})
}

// ListRuns returns all runs, with optional filtering.
// Query parameters:
//   - status: Filter by run status (e.g., "running", "pending", "complete")
//   - taskId: Filter by task ID
//   - profileId: Filter by agent profile ID
//   - tagPrefix: Filter by tag prefix (e.g., "ecosystem-" to get all ecosystem-manager runs)
func (h *Handler) ListRuns(w http.ResponseWriter, r *http.Request) {
	req := apipb.ListRunsRequest{}

	// Parse status filter
	if statusStr := queryFirst(r, "status"); statusStr != "" {
		if status, ok := parseRunStatus(statusStr); ok {
			converted := protoconv.RunStatusToProto(status)
			req.Status = &converted
		} else {
			writeSimpleError(w, r, "status", "invalid run status")
			return
		}
	}

	// Parse task ID filter
	if taskIDStr := queryFirst(r, "task_id", "taskId"); taskIDStr != "" {
		if _, err := uuid.Parse(taskIDStr); err == nil {
			req.TaskId = &taskIDStr
		} else {
			writeSimpleError(w, r, "task_id", "invalid UUID format for task ID")
			return
		}
	}

	// Parse profile ID filter
	if profileIDStr := queryFirst(r, "agent_profile_id", "profileId", "agentProfileId"); profileIDStr != "" {
		if _, err := uuid.Parse(profileIDStr); err == nil {
			req.AgentProfileId = &profileIDStr
		} else {
			writeSimpleError(w, r, "agent_profile_id", "invalid UUID format for agent profile ID")
			return
		}
	}

	// Parse tag prefix filter
	if tagPrefix := queryFirst(r, "tag_prefix", "tagPrefix"); tagPrefix != "" {
		req.TagPrefix = &tagPrefix
	}

	if limit, limitProvided, err := parseQueryIntStrict(r, "limit"); err != nil {
		writeSimpleError(w, r, "limit", "must be a number")
		return
	} else if limitProvided {
		value := int32(limit)
		req.Limit = &value
	}
	if offset, offsetProvided, err := parseQueryIntStrict(r, "offset"); err != nil {
		writeSimpleError(w, r, "offset", "must be a number")
		return
	} else if offsetProvided {
		value := int32(offset)
		req.Offset = &value
	}

	if !h.validateProto(w, r, &req) {
		return
	}

	opts := orchestration.RunListOptions{}
	if req.Status != nil {
		status := protoconv.RunStatusFromProto(req.GetStatus())
		if status != "" {
			opts.Status = &status
		}
	}
	if req.TaskId != nil {
		taskID, _ := uuid.Parse(req.GetTaskId())
		opts.TaskID = &taskID
	}
	if req.AgentProfileId != nil {
		profileID, _ := uuid.Parse(req.GetAgentProfileId())
		opts.AgentProfileID = &profileID
	}
	if req.TagPrefix != nil {
		opts.TagPrefix = req.GetTagPrefix()
	}
	if req.Limit != nil {
		opts.Limit = int(req.GetLimit())
	}
	if req.Offset != nil {
		opts.Offset = int(req.GetOffset())
	}

	runs, err := h.svc.ListRuns(r.Context(), opts)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.ListRunsResponse{
		Runs:  protoconv.RunsToProto(runs),
		Total: int32(len(runs)),
	})
}

// DeleteRun permanently removes a run.
func (h *Handler) DeleteRun(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	req := apipb.DeleteRunRequest{RunId: idStr}
	if !h.validateProto(w, r, &req) {
		return
	}
	id, err := uuid.Parse(req.RunId)
	if err != nil {
		writeSimpleError(w, r, "run_id", "invalid UUID format for run ID")
		return
	}

	if err := h.svc.DeleteRun(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.DeleteRunResponse{Success: true})
}

// StopRun stops a running run.
func (h *Handler) StopRun(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	req := apipb.StopRunRequest{RunId: idStr}
	if !h.validateProto(w, r, &req) {
		return
	}
	id, err := uuid.Parse(req.RunId)
	if err != nil {
		writeSimpleError(w, r, "run_id", "invalid UUID format for run ID")
		return
	}

	if err := h.svc.StopRun(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.StopRunResponse{Status: "stopped"})
}

// ContinueRun continues an existing run's conversation with a follow-up message.
// POST /api/v1/runs/{id}/continue
// Body: {"message": "Please also update the tests"}
func (h *Handler) ContinueRun(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeSimpleError(w, r, "run_id", "invalid UUID format for run ID")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var req domainpb.ContinueRunRequest
	if err := protoconv.UnmarshalJSON(body, &req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}

	// Override run_id from path
	req.RunId = idStr
	if !h.validateProto(w, r, &req) {
		return
	}

	run, err := h.svc.ContinueRun(r.Context(), orchestration.ContinueRunRequest{
		RunID:   id,
		Message: req.Message,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	resp := &domainpb.ContinueRunResponse{
		Success: true,
		Run:     protoconv.RunToProto(run),
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

// GetRunByTag retrieves a run by its custom tag.
func (h *Handler) GetRunByTag(w http.ResponseWriter, r *http.Request) {
	tag := mux.Vars(r)["tag"]
	req := apipb.GetRunByTagRequest{Tag: tag}
	if !h.validateProto(w, r, &req) {
		return
	}

	run, err := h.svc.GetRunByTag(r.Context(), tag)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.GetRunByTagResponse{
		Run: protoconv.RunToProto(run),
	})
}

// StopRunByTag stops a run identified by its custom tag.
func (h *Handler) StopRunByTag(w http.ResponseWriter, r *http.Request) {
	tag := mux.Vars(r)["tag"]
	req := apipb.StopRunByTagRequest{Tag: tag}
	if !h.validateProto(w, r, &req) {
		return
	}

	if err := h.svc.StopRunByTag(r.Context(), tag); err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.StopRunByTagResponse{Status: "stopped", Tag: tag})
}

// StopAllRuns stops all running runs, optionally filtered by tag prefix.
// POST /api/v1/runs/stop-all
// Body: {"tagPrefix": "ecosystem-", "force": true}
func (h *Handler) StopAllRuns(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var req apipb.StopAllRunsRequest
	if len(body) > 0 {
		if err := protoconv.UnmarshalJSON(body, &req); err != nil {
			writeSimpleError(w, r, "body", "invalid JSON request body")
			return
		}
		if !h.validateProto(w, r, &req) {
			return
		}
	}
	tagPrefix := ""
	if req.TagPrefix != nil {
		tagPrefix = *req.TagPrefix
	}

	result, err := h.svc.StopAllRuns(r.Context(), orchestration.StopAllOptions{
		TagPrefix: tagPrefix,
		Force:     req.Force,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.StopAllRunsResponse{
		Result: protoconv.StopAllResultToProto(&protoconv.StopAllResult{
			Stopped:   result.Stopped,
			Failed:    result.Failed,
			Skipped:   result.Skipped,
			FailedIDs: result.FailedIDs,
		}),
	})
}

// GetRunEvents returns events for a run.
func (h *Handler) GetRunEvents(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for run ID")
		return
	}

	req := apipb.GetRunEventsRequest{RunId: id.String()}
	if after, provided, err := parseQueryInt64Strict(r, "after_sequence", "afterSequence"); err != nil {
		writeSimpleError(w, r, "after_sequence", "must be a number")
		return
	} else if provided {
		value := after
		req.AfterSequence = &value
	}
	if limit, provided, err := parseQueryIntStrict(r, "limit"); err != nil {
		writeSimpleError(w, r, "limit", "must be a number")
		return
	} else if provided {
		value := int32(limit)
		req.Limit = &value
	}

	opts := event.GetOptions{}
	eventTypesRaw := r.URL.Query()["event_types"]
	if len(eventTypesRaw) == 0 {
		eventTypesRaw = r.URL.Query()["eventTypes"]
	}
	if len(eventTypesRaw) > 0 {
		types, invalid := parseEventTypes(eventTypesRaw)
		if len(invalid) > 0 {
			writeSimpleError(w, r, "event_types", "invalid event type")
			return
		}
		req.EventTypes = make([]domainpb.RunEventType, len(types))
		for i, t := range types {
			req.EventTypes[i] = protoconv.RunEventTypeToProto(t)
		}
	}
	if !h.validateProto(w, r, &req) {
		return
	}

	if req.AfterSequence != nil {
		opts.AfterSequence = req.GetAfterSequence()
	}
	if req.Limit != nil {
		opts.Limit = int(req.GetLimit())
	}
	if len(req.EventTypes) > 0 {
		opts.EventTypes = make([]domain.RunEventType, len(req.EventTypes))
		for i, t := range req.EventTypes {
			opts.EventTypes[i] = protoconv.RunEventTypeFromProto(t)
		}
	}

	events, err := h.svc.GetRunEvents(r.Context(), id, opts)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.GetRunEventsResponse{
		Events: protoconv.RunEventsToProto(events),
	})
}

// GetRunDiff returns the diff for a run.
func (h *Handler) GetRunDiff(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	req := apipb.GetRunDiffRequest{RunId: idStr}
	if !h.validateProto(w, r, &req) {
		return
	}
	id, err := uuid.Parse(req.RunId)
	if err != nil {
		writeSimpleError(w, r, "run_id", "invalid UUID format for run ID")
		return
	}

	diff, err := h.svc.GetRunDiff(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	// Convert sandbox.FileChange to protoconv.FileChange
	files := make([]protoconv.FileChange, len(diff.Files))
	for i, f := range diff.Files {
		files[i] = protoconv.FileChange{
			ID:           f.ID,
			FilePath:     f.FilePath,
			ChangeType:   string(f.ChangeType),
			FileSize:     f.FileSize,
			LinesAdded:   f.LinesAdded,
			LinesRemoved: f.LinesRemoved,
		}
	}

	writeProtoJSON(w, http.StatusOK, &apipb.GetRunDiffResponse{
		Diff: protoconv.DiffResultToProto(id, &protoconv.DiffResult{
			SandboxID:   diff.SandboxID,
			Files:       files,
			UnifiedDiff: diff.UnifiedDiff,
			Generated:   diff.Generated,
		}),
	})
}

// ApproveRun approves a run's changes.
func (h *Handler) ApproveRun(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for run ID")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var req apipb.ApproveRunRequest
	if err := protoconv.UnmarshalJSON(body, &req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}
	if !h.validateProto(w, r, &req) {
		return
	}
	if req.RunId != "" && req.RunId != id.String() {
		writeSimpleError(w, r, "run_id", "run_id does not match URL")
		return
	}

	actor := normalizeActor(req.GetActor())
	result, err := h.svc.ApproveRun(r.Context(), orchestration.ApproveRequest{
		RunID:     id,
		Actor:     actor,
		CommitMsg: req.GetCommitMsg(),
		Force:     req.Force,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.ApproveRunResponse{
		Result: protoconv.ApproveResultToProto(&protoconv.ApproveResult{
			Success:    result.Success,
			Applied:    result.Applied,
			Remaining:  result.Remaining,
			IsPartial:  result.IsPartial,
			CommitHash: result.CommitHash,
			ErrorMsg:   result.ErrorMsg,
		}),
	})
}

// RejectRun rejects a run's changes.
func (h *Handler) RejectRun(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for run ID")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var req apipb.RejectRunRequest
	if err := protoconv.UnmarshalJSON(body, &req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}
	if !h.validateProto(w, r, &req) {
		return
	}
	if req.RunId != "" && req.RunId != id.String() {
		writeSimpleError(w, r, "run_id", "run_id does not match URL")
		return
	}

	actor := normalizeActor(req.GetActor())
	reason := strings.TrimSpace(req.Reason)
	if err := h.svc.RejectRun(r.Context(), id, actor, reason); err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.RejectRunResponse{Status: "rejected"})
}

// GetRunnerStatus returns status of all runners.
func (h *Handler) GetRunnerStatus(w http.ResponseWriter, r *http.Request) {
	statuses, err := h.svc.GetRunnerStatus(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}

	// Convert orchestration.RunnerStatus to protoconv types
	protoStatuses := make([]*protoconv.OrchestratorRunnerStatus, len(statuses))
	for i, s := range statuses {
		protoStatuses[i] = &protoconv.OrchestratorRunnerStatus{
			Type:      s.Type,
			Available: s.Available,
			Message:   s.Message,
			Capabilities: protoconv.RunnerCapabilities{
				SupportsMessages:     s.Capabilities.SupportsMessages,
				SupportsToolEvents:   s.Capabilities.SupportsToolEvents,
				SupportsCostTracking: s.Capabilities.SupportsCostTracking,
				SupportsStreaming:    s.Capabilities.SupportsStreaming,
				SupportsCancellation: s.Capabilities.SupportsCancellation,
				MaxTurns:             s.Capabilities.MaxTurns,
				SupportedModels:      s.Capabilities.SupportedModels,
			},
		}
	}

	writeProtoJSON(w, http.StatusOK, &apipb.GetRunnerStatusResponse{
		Runners: protoconv.OrchestratorRunnerStatusesToProto(protoStatuses),
	})
}

// ProbeRunner sends a test request to verify a runner can respond.
func (h *Handler) ProbeRunner(w http.ResponseWriter, r *http.Request) {
	runnerType := mux.Vars(r)["runner_type"]
	if runnerType == "" {
		writeSimpleError(w, r, "runner_type", "runner type is required")
		return
	}
	parsed, ok := parseRunnerType(runnerType)
	if !ok {
		writeSimpleError(w, r, "runner_type", "invalid runner type")
		return
	}

	result, err := h.svc.ProbeRunner(r.Context(), parsed)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.ProbeRunnerResponse{
		Result: protoconv.ProbeResultToProto(&protoconv.ProbeResult{
			RunnerType: result.RunnerType,
			Success:    result.Success,
			Message:    result.Message,
			Response:   result.Response,
			DurationMs: result.DurationMs,
		}),
	})
}

// GetRunnerModels returns the model registry for all runners.
func (h *Handler) GetRunnerModels(w http.ResponseWriter, r *http.Request) {
	registry, err := h.svc.GetModelRegistry(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, registry)
}

// UpdateRunnerModels replaces the model registry with the provided payload.
func (h *Handler) UpdateRunnerModels(w http.ResponseWriter, r *http.Request) {
	var registry modelregistry.Registry
	if err := json.NewDecoder(r.Body).Decode(&registry); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}
	updated, err := h.svc.UpdateModelRegistry(r.Context(), &registry)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) resolveProfileModels(ctx context.Context, profile *domain.AgentProfile) ([]*apipb.AvailableModel, map[string]string) {
	if profile == nil {
		return nil, nil
	}

	registry, err := h.svc.GetModelRegistry(ctx)
	if err != nil || registry == nil {
		return nil, nil
	}

	statuses, _ := h.svc.GetRunnerStatus(ctx)

	runnerKey := string(profile.RunnerType)
	runnerRegistry, ok := registry.Runners[runnerKey]
	if !ok {
		return nil, nil
	}

	supported := make(map[string]struct{})
	if len(statuses) > 0 {
		for _, status := range statuses {
			if status.Type != profile.RunnerType {
				continue
			}
			for _, model := range status.Capabilities.SupportedModels {
				trimmed := strings.TrimSpace(model)
				if trimmed == "" {
					continue
				}
				supported[trimmed] = struct{}{}
			}
			break
		}
	}

	available := make([]*apipb.AvailableModel, 0, len(runnerRegistry.Models))
	for _, model := range runnerRegistry.Models {
		id := strings.TrimSpace(model.ID)
		if id == "" {
			continue
		}
		if len(supported) > 0 {
			if _, ok := supported[id]; !ok {
				continue
			}
		}

		label, provider := splitModelID(id)
		sources := []string{"registry"}
		if len(supported) > 0 {
			sources = append(sources, "capabilities")
		}
		available = append(available, &apipb.AvailableModel{
			Id:          id,
			Label:       label,
			Description: strings.TrimSpace(model.Description),
			Provider:    provider,
			Sources:     sources,
		})
	}

	sort.Slice(available, func(i, j int) bool {
		return available[i].Id < available[j].Id
	})

	var presets map[string]string
	for key, modelID := range runnerRegistry.Presets {
		modelID = strings.TrimSpace(modelID)
		if modelID == "" {
			continue
		}
		if len(supported) > 0 {
			if _, ok := supported[modelID]; !ok {
				continue
			}
		}
		if presets == nil {
			presets = make(map[string]string)
		}
		presets[key] = modelID
	}

	if len(available) == 0 {
		return nil, nil
	}

	return available, presets
}

func splitModelID(id string) (label string, provider string) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) == 2 {
		return parts[1], parts[0]
	}
	return id, ""
}

// PurgeData deletes profiles, tasks, or runs matching a regex pattern.
func (h *Handler) PurgeData(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}

	var req apipb.PurgeDataRequest
	if err := protoconv.UnmarshalJSON(body, &req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}
	if !h.validateProto(w, r, &req) {
		return
	}
	if req.Pattern == "" {
		writeSimpleError(w, r, "pattern", "pattern is required")
		return
	}
	if len(req.Targets) == 0 {
		writeSimpleError(w, r, "targets", "targets are required")
		return
	}

	targets := make([]orchestration.PurgeTarget, 0, len(req.Targets))
	for _, target := range req.Targets {
		switch target {
		case apipb.PurgeTarget_PURGE_TARGET_PROFILES:
			targets = append(targets, orchestration.PurgeTargetProfiles)
		case apipb.PurgeTarget_PURGE_TARGET_TASKS:
			targets = append(targets, orchestration.PurgeTargetTasks)
		case apipb.PurgeTarget_PURGE_TARGET_RUNS:
			targets = append(targets, orchestration.PurgeTargetRuns)
		default:
			writeSimpleError(w, r, "targets", "invalid purge target")
			return
		}
	}

	result, err := h.svc.PurgeData(r.Context(), orchestration.PurgeRequest{
		Pattern: req.Pattern,
		Targets: targets,
		DryRun:  req.DryRun,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.PurgeDataResponse{
		Matched: purgeCountsToProto(result.Matched),
		Deleted: purgeCountsToProto(result.Deleted),
		DryRun:  result.DryRun,
	})
}

func purgeCountsToProto(counts orchestration.PurgeCounts) *apipb.PurgeCounts {
	return &apipb.PurgeCounts{
		Profiles: int32(counts.Profiles),
		Tasks:    int32(counts.Tasks),
		Runs:     int32(counts.Runs),
	}
}

func normalizeActor(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "unknown"
	}
	return trimmed
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
		"promptTemplate": settings.PromptTemplate,
		"defaultDepth":   string(settings.DefaultDepth),
		"defaultContext": settings.DefaultContext,
		"updatedAt":      settings.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// UpdateInvestigationSettings updates the investigation settings.
func (h *Handler) UpdateInvestigationSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PromptTemplate string                            `json:"promptTemplate"`
		DefaultDepth   string                            `json:"defaultDepth"`
		DefaultContext *domain.InvestigationContextFlags `json:"defaultContext"`
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

	if err := h.svc.UpdateInvestigationSettings(r.Context(), current); err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"promptTemplate": current.PromptTemplate,
		"defaultDepth":   string(current.DefaultDepth),
		"defaultContext": current.DefaultContext,
		"updatedAt":      current.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
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
		"promptTemplate": settings.PromptTemplate,
		"defaultDepth":   string(settings.DefaultDepth),
		"defaultContext": settings.DefaultContext,
		"updatedAt":      settings.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// DetectScenariosForRuns detects scenarios from run data.
func (h *Handler) DetectScenariosForRuns(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RunIDs []string `json:"runIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}

	runIDs := make([]uuid.UUID, 0, len(req.RunIDs))
	for _, idStr := range req.RunIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			writeSimpleError(w, r, "runIds", "invalid UUID format: "+idStr)
			return
		}
		runIDs = append(runIDs, id)
	}

	scenarios, err := h.svc.DetectScenariosForRuns(r.Context(), runIDs)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"scenarios": scenarios,
	})
}
