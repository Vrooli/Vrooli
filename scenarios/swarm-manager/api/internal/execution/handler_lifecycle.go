package execution

import (
	"encoding/json"
	"net/http"
	"strings"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"

	"github.com/gorilla/mux"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

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
