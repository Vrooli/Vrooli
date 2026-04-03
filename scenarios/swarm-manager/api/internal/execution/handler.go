package execution

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
)

// DOC: docs/concepts/ARCHITECTURE.md#api-boundaries
// DOC: docs/reference/operational-targets.md
// DOC: docs/internal/TEMPORAL-FLOWS.md

// Handler exposes execution-control endpoints.
type Handler struct {
	service *Service
}

// NewHandler creates a handler with filesystem-backed storage.
func NewHandler(cfg ServiceConfig) *Handler {
	return &Handler{service: NewService(cfg)}
}

// NewHandlerFromService creates a handler from an existing Service.
func NewHandlerFromService(svc *Service) *Handler {
	return &Handler{service: svc}
}

// RegisterRoutes registers execution routes.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/execution", h.List).Methods("GET")
	r.HandleFunc("/api/v1/execution", h.Create).Methods("POST")
	r.HandleFunc("/api/v1/execution/{execution_id}", h.Get).Methods("GET")
	r.HandleFunc("/api/v1/execution/{execution_id}/prompt-trace", h.GetPromptTrace).Methods("GET")
	r.HandleFunc("/api/v1/execution/{execution_id}/start", h.Start).Methods("POST")
	r.HandleFunc("/api/v1/execution/{execution_id}/cancel", h.Cancel).Methods("POST")
	r.HandleFunc("/api/v1/execution/{execution_id}/retry", h.Retry).Methods("POST")
	r.HandleFunc("/api/v1/execution/{execution_id}/follow-up", h.FollowUp).Methods("POST")
	r.HandleFunc("/api/v1/execution/{execution_id}/trigger-review", h.TriggerReview).Methods("POST")
	r.HandleFunc("/api/v1/execution/circuit-breaker/reset", h.ResetCircuitBreaker).Methods("POST")
	r.HandleFunc("/api/v1/gct/status", h.GCTStatus).Methods("GET")
}

// StartBackgroundWorker launches the background worker for active execution
// progression.
func (h *Handler) StartBackgroundWorker(stop <-chan struct{}) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			_ = h.service.ProcessActiveExecutions(context.Background())
		}
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filters := ListFilters{
		Status:      strings.TrimSpace(r.URL.Query().Get("status")),
		Mode:        strings.TrimSpace(r.URL.Query().Get("mode")),
		BacklogKind: strings.TrimSpace(r.URL.Query().Get("backlog_kind")),
		BacklogName: strings.TrimSpace(r.URL.Query().Get("backlog_name")),
		StartedBy:   strings.TrimSpace(r.URL.Query().Get("started_by")),
		CreatedFrom: strings.TrimSpace(r.URL.Query().Get("created_from")),
		CreatedTo:   strings.TrimSpace(r.URL.Query().Get("created_to")),
	}
	items, err := h.service.List(r.Context(), filters)
	if err != nil {
		apierr.MapError(w, "[execution] list", err)
		return
	}
	protoItems := make([]*domainpb.ExecutionRecord, len(items))
	for i, item := range items {
		protoItems[i] = recordToProto(item)
	}
	resp := &apipb.ListExecutionResponse{Items: protoItems}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[execution] list", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	executionID := strings.TrimSpace(mux.Vars(r)["execution_id"])
	if executionID == "" {
		apierr.MapError(w, "[execution] get", apierr.BadRequest("execution_id is required"))
		return
	}
	record, err := h.service.Get(r.Context(), executionID)
	if err != nil {
		apierr.MapError(w, "[execution] get", err)
		return
	}
	if err := httputil.ProtoJSON(w, executionResponse(record)); err != nil {
		apierr.MapError(w, "[execution] get", apierr.Internal("failed to encode response"))
	}
	h.service.RecordView(executionID)
}

func (h *Handler) GetPromptTrace(w http.ResponseWriter, r *http.Request) {
	executionID := strings.TrimSpace(mux.Vars(r)["execution_id"])
	if executionID == "" {
		apierr.MapError(w, "[execution] prompt-trace", apierr.BadRequest("execution_id is required"))
		return
	}
	record, err := h.service.Get(r.Context(), executionID)
	if err != nil {
		apierr.MapError(w, "[execution] prompt-trace", err)
		return
	}
	if record.PromptTrace == nil {
		apierr.MapError(w, "[execution] prompt-trace", apierr.NotFound("prompt trace not found"))
		return
	}
	if err := httputil.JSON(w, map[string]any{"trace": record.PromptTrace}); err != nil {
		apierr.MapError(w, "[execution] prompt-trace", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var pbReq apipb.CreateExecutionRequest
	if err := httputil.DecodeProtoJSON(r, &pbReq); err != nil {
		apierr.MapError(w, "[execution] create", apierr.BadRequest("invalid request body"))
		return
	}
	if !httputil.ValidateProtoRequest(w, "[execution] create", "invalid execution request", &pbReq) {
		return
	}
	mode := Mode(pbReq.Mode)
	if mode == "" {
		mode = ModeYOLO
	}
	req := CreateRequest{
		BacklogKind: pbReq.BacklogKind,
		BacklogName: pbReq.BacklogName,
		Mode:        mode,
		StartedBy:   pbReq.GetStartedBy(),
		Operation:   pbReq.GetOperation(),
	}
	record, err := h.service.QueueBacklog(r.Context(), req)
	if err != nil {
		apierr.MapError(w, "[execution] create", err)
		return
	}
	if err := httputil.ProtoJSONWithStatus(w, http.StatusAccepted, executionResponse(record)); err != nil {
		apierr.MapError(w, "[execution] create", apierr.Internal("failed to encode response"))
	}
}

// ResetCircuitBreaker clears the circuit breaker for a specific item.
func (h *Handler) ResetCircuitBreaker(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Item string `json:"item"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apierr.MapError(w, "[execution] circuit-breaker-reset", apierr.BadRequest("invalid request body"))
		return
	}
	if strings.TrimSpace(body.Item) == "" {
		apierr.MapError(w, "[execution] circuit-breaker-reset", apierr.BadRequest("item is required"))
		return
	}
	if err := h.service.ResetCircuitBreaker(body.Item); err != nil {
		apierr.MapError(w, "[execution] circuit-breaker-reset", err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func (h *Handler) Start(w http.ResponseWriter, r *http.Request) {
	executionID := strings.TrimSpace(mux.Vars(r)["execution_id"])
	if executionID == "" {
		apierr.MapError(w, "[execution] start", apierr.BadRequest("execution_id is required"))
		return
	}
	record, err := h.service.Start(r.Context(), executionID)
	if err != nil {
		apierr.MapError(w, "[execution] start", err)
		return
	}
	if err := httputil.ProtoJSON(w, executionResponse(record)); err != nil {
		apierr.MapError(w, "[execution] start", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	executionID := strings.TrimSpace(mux.Vars(r)["execution_id"])
	if executionID == "" {
		apierr.MapError(w, "[execution] cancel", apierr.BadRequest("execution_id is required"))
		return
	}
	record, err := h.service.Cancel(r.Context(), executionID)
	if err != nil {
		apierr.MapError(w, "[execution] cancel", err)
		return
	}
	if err := httputil.ProtoJSON(w, executionResponse(record)); err != nil {
		apierr.MapError(w, "[execution] cancel", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) Retry(w http.ResponseWriter, r *http.Request) {
	executionID := strings.TrimSpace(mux.Vars(r)["execution_id"])
	if executionID == "" {
		apierr.MapError(w, "[execution] retry", apierr.BadRequest("execution_id is required"))
		return
	}
	record, err := h.service.Retry(r.Context(), executionID)
	if err != nil {
		apierr.MapError(w, "[execution] retry", err)
		return
	}
	if err := httputil.ProtoJSON(w, executionResponse(record)); err != nil {
		apierr.MapError(w, "[execution] retry", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) FollowUp(w http.ResponseWriter, r *http.Request) {
	executionID := strings.TrimSpace(mux.Vars(r)["execution_id"])
	if executionID == "" {
		apierr.MapError(w, "[execution] follow-up", apierr.BadRequest("execution_id is required"))
		return
	}
	var pbReq apipb.FollowUpExecutionRequest
	if err := httputil.DecodeProtoJSON(r, &pbReq); err != nil {
		apierr.MapError(w, "[execution] follow-up", apierr.BadRequest("invalid request body"))
		return
	}
	if !httputil.ValidateProtoRequest(w, "[execution] follow-up", "invalid follow-up request", &pbReq) {
		return
	}
	req := FollowUpRequest{
		ExecutionID:  executionID,
		FollowUpType: pbReq.FollowUpType,
		Context:      pbReq.GetContext(),
		RunMode:      pbReq.RunMode,
	}
	record, err := h.service.FollowUp(r.Context(), req)
	if err != nil {
		apierr.MapError(w, "[execution] follow-up", err)
		return
	}
	if err := httputil.ProtoJSONWithStatus(w, http.StatusAccepted, executionResponse(record)); err != nil {
		apierr.MapError(w, "[execution] follow-up", apierr.Internal("failed to encode response"))
	}
}

// TriggerReview manually triggers a GCT review for a terminal execution.
func (h *Handler) TriggerReview(w http.ResponseWriter, r *http.Request) {
	executionID := strings.TrimSpace(mux.Vars(r)["execution_id"])
	if executionID == "" {
		apierr.MapError(w, "[execution] trigger-review", apierr.BadRequest("execution_id is required"))
		return
	}
	record, err := h.service.TriggerReview(r.Context(), executionID)
	if err != nil {
		apierr.MapError(w, "[execution] trigger-review", err)
		return
	}
	if err := httputil.ProtoJSON(w, executionResponse(record)); err != nil {
		apierr.MapError(w, "[execution] trigger-review", apierr.Internal("failed to encode response"))
	}
}

// GCTStatus returns whether git-control-tower is reachable.
func (h *Handler) GCTStatus(w http.ResponseWriter, r *http.Request) {
	available := false
	if h.service.reviewClient != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := h.service.reviewClient.Ping(ctx); err == nil {
			available = true
		}
	}
	w.Header().Set("Content-Type", "application/json")
	if available {
		_, _ = w.Write([]byte(`{"available":true}`))
	} else {
		_, _ = w.Write([]byte(`{"available":false}`))
	}
}

func executionResponse(record Record) *apipb.ExecutionResponse {
	return &apipb.ExecutionResponse{Execution: recordToProto(record)}
}

func recordToProto(r Record) *domainpb.ExecutionRecord {
	pb := &domainpb.ExecutionRecord{
		ExecutionId: r.ExecutionID,
		BacklogKind: r.BacklogKind,
		BacklogName: r.BacklogName,
		Status:      string(r.Status),
		Mode:        string(r.Mode),
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
	if r.TaskID != "" {
		pb.TaskId = &r.TaskID
	}
	if r.RunID != "" {
		pb.RunId = &r.RunID
	}
	if r.StartedAt != "" {
		pb.StartedAt = &r.StartedAt
	}
	if r.FinishedAt != "" {
		pb.FinishedAt = &r.FinishedAt
	}
	if r.FailureReason != "" {
		pb.FailureReason = &r.FailureReason
	}
	if r.StartedBy != "" {
		pb.StartedBy = &r.StartedBy
	}
	if r.Operation != "" {
		pb.Operation = &r.Operation
	}
	if r.ArchiveContext != nil {
		ac := r.ArchiveContext
		pbAc := &domainpb.ArchiveContext{
			ScenarioName:  ac.ScenarioName,
			ScenarioPath:  ac.ScenarioPath,
			PreservePaths: ac.PreservePaths,
		}
		if ac.PresetOrCustom != "" {
			pbAc.PresetOrCustom = &ac.PresetOrCustom
		}
		if ac.PreservePreset != "" {
			pbAc.PreservePreset = &ac.PreservePreset
		}
		pb.ArchiveContext = pbAc
	}
	if r.ParentExecutionID != "" {
		pb.ParentExecutionId = &r.ParentExecutionID
	}
	pb.FixupAttempt = int32(r.FixupAttempt)
	if finalization := effectiveFinalization(r); finalization != nil {
		pb.Finalization = finalizationToProto(finalization)
	}
	return pb
}

func reviewResultToProto(rr *ReviewResult) *domainpb.ReviewResult {
	pb := &domainpb.ReviewResult{
		JobId:          rr.JobID,
		Classification: rr.Classification,
		Summary:        rr.Summary,
		ReviewedAt:     rr.ReviewedAt,
	}
	for _, dim := range rr.Dimensions {
		pbDim := &domainpb.ReviewDimension{
			Name:   dim.Name,
			Status: dim.Status,
		}
		if dim.Details != "" {
			pbDim.Details = &dim.Details
		}
		pb.Dimensions = append(pb.Dimensions, pbDim)
	}
	return pb
}

func finalizationToProto(finalization *Finalization) *domainpb.Finalization {
	if finalization == nil {
		return nil
	}
	pb := &domainpb.Finalization{
		Eligible:                finalization.Eligible,
		Status:                  string(finalization.Status),
		Phase:                   finalization.Phase,
		ScopeSource:             finalization.ScopeSource,
		AffectedScenarios:       append([]string(nil), finalization.AffectedScenarios...),
		AggregateClassification: finalization.AggregateClassification,
	}
	if finalization.SkipReason != "" {
		pb.SkipReason = &finalization.SkipReason
	}
	if finalization.StartedAt != "" {
		pb.StartedAt = &finalization.StartedAt
	}
	if finalization.CompletedAt != "" {
		pb.CompletedAt = &finalization.CompletedAt
	}
	if finalization.AggregateSummary != "" {
		pb.AggregateSummary = &finalization.AggregateSummary
	}
	for _, warning := range finalization.Warnings {
		pbWarning := &domainpb.FinalizationWarning{
			Code:      warning.Code,
			Message:   warning.Message,
			Retryable: warning.Retryable,
			CreatedAt: warning.CreatedAt,
		}
		if warning.ScenarioName != "" {
			pbWarning.ScenarioName = &warning.ScenarioName
		}
		pb.Warnings = append(pb.Warnings, pbWarning)
	}
	for _, scenario := range finalization.Scenarios {
		pbScenario := &domainpb.ScenarioFinalization{
			ScenarioName: scenario.ScenarioName,
			ChangedPaths: append([]string(nil), scenario.ChangedPaths...),
			Restart: &domainpb.RestartResult{
				Status:   string(scenario.Restart.Status),
				Attempts: int32(scenario.Restart.Attempts),
			},
			Health: &domainpb.HealthCheckResult{
				Status:      string(scenario.Health.Status),
				SchemaValid: scenario.Health.SchemaValid,
			},
			Review: &domainpb.ScenarioReview{
				Status: string(scenario.Review.Status),
			},
		}
		if scenario.Restart.LastError != "" {
			pbScenario.Restart.LastError = &scenario.Restart.LastError
		}
		if scenario.Restart.StartedAt != "" {
			pbScenario.Restart.StartedAt = &scenario.Restart.StartedAt
		}
		if scenario.Restart.FinishedAt != "" {
			pbScenario.Restart.FinishedAt = &scenario.Restart.FinishedAt
		}
		if scenario.Health.ScenarioStatus != "" {
			pbScenario.Health.ScenarioStatus = &scenario.Health.ScenarioStatus
		}
		if scenario.Health.HealthStatus != "" {
			pbScenario.Health.HealthStatus = &scenario.Health.HealthStatus
		}
		if scenario.Health.Details != "" {
			pbScenario.Health.Details = &scenario.Health.Details
		}
		if scenario.Health.CheckedAt != "" {
			pbScenario.Health.CheckedAt = &scenario.Health.CheckedAt
		}
		if scenario.Review.JobID != "" {
			pbScenario.Review.JobId = &scenario.Review.JobID
		}
		if scenario.Review.SkipReason != "" {
			pbScenario.Review.SkipReason = &scenario.Review.SkipReason
		}
		if scenario.Review.Result != nil {
			pbScenario.Review.Result = reviewResultToProto(scenario.Review.Result)
		}
		pb.Scenarios = append(pb.Scenarios, pbScenario)
	}
	return pb
}
