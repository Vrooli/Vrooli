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
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/protoconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"google.golang.org/protobuf/proto"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	commonpb "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// Handler provides HTTP handlers for all API endpoints.
type Handler struct {
	svc orchestration.Service
	hub *WebSocketHub
}

// New creates a new Handler with the given orchestration service.
func New(svc orchestration.Service) *Handler {
	return &Handler{svc: svc}
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

	// Health endpoints
	r.HandleFunc("/health", h.Health).Methods("GET")
	r.HandleFunc("/api/v1/health", h.Health).Methods("GET")

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
	r.HandleFunc("/api/v1/runs", h.ListRuns).Methods("GET")
	r.HandleFunc("/api/v1/runs/stop-all", h.StopAllRuns).Methods("POST") // Must be before /{id}
	r.HandleFunc("/api/v1/runs/tag/{tag}", h.GetRunByTag).Methods("GET")
	r.HandleFunc("/api/v1/runs/tag/{tag}/stop", h.StopRunByTag).Methods("POST")
	r.HandleFunc("/api/v1/runs/{id}", h.GetRun).Methods("GET")
	r.HandleFunc("/api/v1/runs/{id}/stop", h.StopRun).Methods("POST")
	r.HandleFunc("/api/v1/runs/{id}/events", h.GetRunEvents).Methods("GET")
	r.HandleFunc("/api/v1/runs/{id}/diff", h.GetRunDiff).Methods("GET")
	r.HandleFunc("/api/v1/runs/{id}/approve", h.ApproveRun).Methods("POST")
	r.HandleFunc("/api/v1/runs/{id}/reject", h.RejectRun).Methods("POST")

	// Status endpoints
	r.HandleFunc("/api/v1/runners", h.GetRunnerStatus).Methods("GET")
	r.HandleFunc("/api/v1/runners/{type}/probe", h.ProbeRunner).Methods("POST")
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

// writeJSON writes a JSON response with the given status code.
// For non-proto types (legacy compatibility).
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

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

func parseRunnerType(raw string) (domain.RunnerType, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", false
	}
	if strings.HasPrefix(value, "RUNNER_TYPE_") {
		value = strings.ToLower(strings.TrimPrefix(value, "RUNNER_TYPE_"))
		value = strings.ReplaceAll(value, "_", "-")
	}
	runnerType := domain.RunnerType(value)
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
	if strings.HasPrefix(value, "TASK_STATUS_") {
		value = strings.ToLower(strings.TrimPrefix(value, "TASK_STATUS_"))
	}
	status := domain.TaskStatus(value)
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
	if strings.HasPrefix(value, "RUN_STATUS_") {
		value = strings.ToLower(strings.TrimPrefix(value, "RUN_STATUS_"))
	}
	status := domain.RunStatus(value)
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

func parseEventTypes(values []string) []domain.RunEventType {
	var types []domain.RunEventType
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
			}
		}
	}
	return types
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

	writeJSON(w, status, errResp)
}

// writeSimpleError creates a simple validation error for request parsing issues.
func writeSimpleError(w http.ResponseWriter, r *http.Request, field, message string) {
	err := domain.NewValidationError(field, message)
	writeError(w, r, err)
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
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for profile ID")
		return
	}

	profile, err := h.svc.GetProfile(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.GetProfileResponse{
		Profile: protoconv.AgentProfileToProto(profile),
	})
}

// ListProfiles returns all agent profiles.
func (h *Handler) ListProfiles(w http.ResponseWriter, r *http.Request) {
	limit, _ := parseQueryInt(r, "limit")
	offset, _ := parseQueryInt(r, "offset")
	runnerTypeRaw := queryFirst(r, "runner_type", "runnerType")
	var runnerTypeFilter *domain.RunnerType
	if runnerTypeRaw != "" {
		if parsed, ok := parseRunnerType(runnerTypeRaw); ok {
			runnerTypeFilter = &parsed
		} else {
			writeSimpleError(w, r, "runner_type", "invalid runner type")
			return
		}
	}

	profiles, err := h.svc.ListProfiles(r.Context(), orchestration.ListOptions{
		Limit:  limit,
		Offset: offset,
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
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for profile ID")
		return
	}

	if err := h.svc.DeleteProfile(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for task ID")
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
	limit, _ := parseQueryInt(r, "limit")
	offset, _ := parseQueryInt(r, "offset")
	statusRaw := queryFirst(r, "status")
	scopePrefix := queryFirst(r, "scope_prefix", "scopePrefix")

	var statusFilter *domain.TaskStatus
	if statusRaw != "" {
		if parsed, ok := parseTaskStatus(statusRaw); ok {
			statusFilter = &parsed
		} else {
			writeSimpleError(w, r, "status", "invalid task status")
			return
		}
	}

	tasks, err := h.svc.ListTasks(r.Context(), orchestration.ListOptions{
		Limit:  limit,
		Offset: offset,
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
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for task ID")
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
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for task ID")
		return
	}

	if err := h.svc.DeleteTask(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
	if protoReq.ProfileRef != nil {
		req.ProfileRef = &orchestration.ProfileRef{
			ProfileKey: protoReq.ProfileRef.ProfileKey,
			Defaults:   protoconv.AgentProfileFromProto(protoReq.ProfileRef.Defaults),
		}
	}
	if protoReq.InlineConfig != nil {
		inline := protoReq.InlineConfig
		if inline.RunnerType != domainpb.RunnerType_RUNNER_TYPE_UNSPECIFIED {
			runner := protoconv.RunnerTypeFromProto(inline.RunnerType)
			req.RunnerType = &runner
		}
		if inline.Model != "" {
			model := inline.Model
			req.Model = &model
		}
		if inline.MaxTurns != 0 {
			maxTurns := int(inline.MaxTurns)
			req.MaxTurns = &maxTurns
		}
		if inline.Timeout != nil {
			timeout := inline.Timeout.AsDuration()
			req.Timeout = &timeout
		}
		if len(inline.AllowedTools) > 0 {
			req.AllowedTools = inline.AllowedTools
		}
		if len(inline.DeniedTools) > 0 {
			req.DeniedTools = inline.DeniedTools
		}
		skipPermissionPrompt := inline.SkipPermissionPrompt
		req.SkipPermissionPrompt = &skipPermissionPrompt
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

// GetRun retrieves a run by ID.
func (h *Handler) GetRun(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for run ID")
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
	opts := orchestration.RunListOptions{}

	// Parse status filter
	if statusStr := queryFirst(r, "status"); statusStr != "" {
		if status, ok := parseRunStatus(statusStr); ok {
			opts.Status = &status
		} else {
			writeSimpleError(w, r, "status", "invalid run status")
			return
		}
	}

	// Parse task ID filter
	if taskIDStr := queryFirst(r, "task_id", "taskId"); taskIDStr != "" {
		if taskID, err := uuid.Parse(taskIDStr); err == nil {
			opts.TaskID = &taskID
		}
	}

	// Parse profile ID filter
	if profileIDStr := queryFirst(r, "agent_profile_id", "profileId", "agentProfileId"); profileIDStr != "" {
		if profileID, err := uuid.Parse(profileIDStr); err == nil {
			opts.AgentProfileID = &profileID
		}
	}

	// Parse tag prefix filter
	if tagPrefix := queryFirst(r, "tag_prefix", "tagPrefix"); tagPrefix != "" {
		opts.TagPrefix = tagPrefix
	}

	if limit, ok := parseQueryInt(r, "limit"); ok {
		opts.Limit = limit
	}
	if offset, ok := parseQueryInt(r, "offset"); ok {
		opts.Offset = offset
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

// StopRun stops a running run.
func (h *Handler) StopRun(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for run ID")
		return
	}

	if err := h.svc.StopRun(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}

	writeProtoJSON(w, http.StatusOK, &apipb.StopRunResponse{Status: "stopped"})
}

// GetRunByTag retrieves a run by its custom tag.
func (h *Handler) GetRunByTag(w http.ResponseWriter, r *http.Request) {
	tag := mux.Vars(r)["tag"]
	if tag == "" {
		writeSimpleError(w, r, "tag", "tag is required")
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
	if tag == "" {
		writeSimpleError(w, r, "tag", "tag is required")
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

	opts := event.GetOptions{}
	if after, ok := parseQueryInt64(r, "after_sequence", "afterSequence"); ok {
		opts.AfterSequence = after
	}
	if limit, ok := parseQueryInt(r, "limit"); ok {
		opts.Limit = limit
	}
	eventTypesRaw := r.URL.Query()["event_types"]
	if len(eventTypesRaw) == 0 {
		eventTypesRaw = r.URL.Query()["eventTypes"]
	}
	opts.EventTypes = parseEventTypes(eventTypesRaw)

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
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for run ID")
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

	result, err := h.svc.ApproveRun(r.Context(), orchestration.ApproveRequest{
		RunID:     id,
		Actor:     req.Actor,
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

	if err := h.svc.RejectRun(r.Context(), id, req.Actor, req.Reason); err != nil {
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
	runnerType := mux.Vars(r)["type"]
	if runnerType == "" {
		writeSimpleError(w, r, "type", "runner type is required")
		return
	}

	result, err := h.svc.ProbeRunner(r.Context(), domain.RunnerType(runnerType))
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
