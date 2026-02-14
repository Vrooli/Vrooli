package execution

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
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

// RegisterRoutes registers execution routes.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/execution", h.List).Methods("GET")
	r.HandleFunc("/api/v1/execution", h.Create).Methods("POST")
	r.HandleFunc("/api/v1/execution/policy", h.GetPolicy).Methods("GET")
	r.HandleFunc("/api/v1/execution/policy", h.UpdatePolicy).Methods("PUT")
	r.HandleFunc("/api/v1/execution/{execution_id}", h.Get).Methods("GET")
	r.HandleFunc("/api/v1/execution/{execution_id}/start", h.Start).Methods("POST")
	r.HandleFunc("/api/v1/execution/{execution_id}/cancel", h.Cancel).Methods("POST")
	r.HandleFunc("/api/v1/execution/{execution_id}/retry", h.Retry).Methods("POST")
}

// StartScheduler launches a polling worker for scheduled starts.
func (h *Handler) StartScheduler(stop <-chan struct{}) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			_ = h.service.ProcessScheduledStarts(context.Background())
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
		httputil.InternalError(w, "[execution] list", "failed to list executions")
		return
	}
	protoItems := make([]*domainpb.ExecutionRecord, len(items))
	for i, item := range items {
		protoItems[i] = recordToProto(item)
	}
	resp := &apipb.ListExecutionResponse{Items: protoItems}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[execution] list", "failed to encode response")
	}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	executionID := strings.TrimSpace(mux.Vars(r)["execution_id"])
	if executionID == "" {
		httputil.BadRequest(w, "[execution] get", "execution_id is required")
		return
	}
	record, err := h.service.Get(r.Context(), executionID)
	if err != nil {
		if errors.Is(err, errNotFound) {
			httputil.NotFound(w, "[execution] get", "execution not found")
			return
		}
		httputil.InternalError(w, "[execution] get", "failed to fetch execution")
		return
	}
	if err := httputil.ProtoJSON(w, executionResponse(record)); err != nil {
		httputil.InternalError(w, "[execution] get", "failed to encode response")
	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var pbReq apipb.CreateExecutionRequest
	if err := httputil.DecodeProtoJSON(r, &pbReq); err != nil {
		httputil.BadRequest(w, "[execution] create", "invalid request body")
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
		BacklogKind:  pbReq.BacklogKind,
		BacklogName:  pbReq.BacklogName,
		Mode:         mode,
		DelaySeconds: pbReq.GetDelaySeconds(),
		StartedBy:    pbReq.GetStartedBy(),
		Operation:    pbReq.GetOperation(),
	}
	record, err := h.service.QueueBacklog(r.Context(), req)
	if err != nil {
		if strings.Contains(err.Error(), "cannot be queued") || strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "mode must") {
			httputil.BadRequest(w, "[execution] create", err.Error())
			return
		}
		if strings.Contains(err.Error(), "not available") {
			httputil.ServiceUnavailable(w, "[execution] create", "agent-manager is not available")
			return
		}
		if errors.Is(err, errNotFound) {
			httputil.NotFound(w, "[execution] create", "backlog item not found")
			return
		}
		httputil.InternalError(w, "[execution] create", "failed to create execution")
		return
	}
	if err := httputil.ProtoJSONWithStatus(w, http.StatusAccepted, executionResponse(record)); err != nil {
		httputil.InternalError(w, "[execution] create", "failed to encode response")
	}
}

func (h *Handler) Start(w http.ResponseWriter, r *http.Request) {
	executionID := strings.TrimSpace(mux.Vars(r)["execution_id"])
	if executionID == "" {
		httputil.BadRequest(w, "[execution] start", "execution_id is required")
		return
	}
	record, err := h.service.Start(r.Context(), executionID)
	if err != nil {
		h.mapMutationError(w, "[execution] start", err)
		return
	}
	if err := httputil.ProtoJSON(w, executionResponse(record)); err != nil {
		httputil.InternalError(w, "[execution] start", "failed to encode response")
	}
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	executionID := strings.TrimSpace(mux.Vars(r)["execution_id"])
	if executionID == "" {
		httputil.BadRequest(w, "[execution] cancel", "execution_id is required")
		return
	}
	record, err := h.service.Cancel(r.Context(), executionID)
	if err != nil {
		h.mapMutationError(w, "[execution] cancel", err)
		return
	}
	if err := httputil.ProtoJSON(w, executionResponse(record)); err != nil {
		httputil.InternalError(w, "[execution] cancel", "failed to encode response")
	}
}

func (h *Handler) Retry(w http.ResponseWriter, r *http.Request) {
	executionID := strings.TrimSpace(mux.Vars(r)["execution_id"])
	if executionID == "" {
		httputil.BadRequest(w, "[execution] retry", "execution_id is required")
		return
	}
	record, err := h.service.Retry(r.Context(), executionID)
	if err != nil {
		h.mapMutationError(w, "[execution] retry", err)
		return
	}
	if err := httputil.ProtoJSON(w, executionResponse(record)); err != nil {
		httputil.InternalError(w, "[execution] retry", "failed to encode response")
	}
}

func (h *Handler) mapMutationError(w http.ResponseWriter, prefix string, err error) {
	switch {
	case errors.Is(err, errNotFound):
		httputil.NotFound(w, prefix, "execution not found")
	case strings.Contains(err.Error(), "required"),
		strings.Contains(err.Error(), "cannot"),
		strings.Contains(err.Error(), "only"):
		httputil.BadRequest(w, prefix, err.Error())
	case strings.Contains(err.Error(), "not available"):
		httputil.ServiceUnavailable(w, prefix, "agent-manager is not available")
	default:
		httputil.InternalError(w, prefix, "execution operation failed")
	}
}

func (h *Handler) GetPolicy(w http.ResponseWriter, r *http.Request) {
	policy, err := h.service.Policy(r.Context())
	if err != nil {
		httputil.InternalError(w, "[execution] policy get", "failed to load execution policy")
		return
	}
	resp := &apipb.ExecutionPolicyResponse{Policy: policyToProto(policy)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[execution] policy get", "failed to encode response")
	}
}

func (h *Handler) UpdatePolicy(w http.ResponseWriter, r *http.Request) {
	var pbPolicy domainpb.ExecutionPolicy
	if err := httputil.DecodeProtoJSON(r, &pbPolicy); err != nil {
		httputil.BadRequest(w, "[execution] policy update", "invalid request body")
		return
	}
	if !httputil.ValidateProtoRequest(w, "[execution] policy update", "default_mode must be manual, scheduled, or yolo", &pbPolicy) {
		return
	}
	req := Policy{
		DefaultMode:         Mode(pbPolicy.DefaultMode),
		DefaultDelaySeconds: pbPolicy.DefaultDelaySeconds,
	}
	policy, err := h.service.UpdatePolicy(r.Context(), req)
	if err != nil {
		httputil.InternalError(w, "[execution] policy update", "failed to persist execution policy")
		return
	}
	resp := &apipb.ExecutionPolicyResponse{Policy: policyToProto(policy)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[execution] policy update", "failed to encode response")
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
	if r.ScheduledAt != "" {
		pb.ScheduledAt = &r.ScheduledAt
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
	return pb
}

func policyToProto(p Policy) *domainpb.ExecutionPolicy {
	return &domainpb.ExecutionPolicy{
		DefaultMode:         string(p.DefaultMode),
		DefaultDelaySeconds: p.DefaultDelaySeconds,
	}
}
