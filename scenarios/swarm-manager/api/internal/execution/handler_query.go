package execution

import (
	"context"
	"net/http"
	"strings"
	"time"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"

	"github.com/gorilla/mux"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
)

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
	items, err := h.service.ListSnapshot(r.Context(), filters)
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

// GetProgress proxies the active workflow trace on demand. It does not persist
// or infer live state from the local execution record.
func (h *Handler) GetProgress(w http.ResponseWriter, r *http.Request) {
	executionID := strings.TrimSpace(mux.Vars(r)["execution_id"])
	if executionID == "" {
		apierr.MapError(w, "[execution] progress", apierr.BadRequest("execution_id is required"))
		return
	}
	progress, err := h.service.WorkflowProgress(r.Context(), executionID)
	if err != nil {
		apierr.MapError(w, "[execution] progress", err)
		return
	}
	if err := httputil.JSON(w, progress); err != nil {
		apierr.MapError(w, "[execution] progress", apierr.Internal("encode workflow progress"))
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
