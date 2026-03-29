package agentactivity

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
	"swarm-manager/internal/httputil"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/agent-activities", h.List).Methods("GET")
	r.HandleFunc("/api/v1/agent-activities/{activity_id}", h.Get).Methods("GET")
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filters := ListFilters{
		OwnerType:   strings.TrimSpace(r.URL.Query().Get("owner_type")),
		OwnerKind:   strings.TrimSpace(r.URL.Query().Get("owner_kind")),
		OwnerName:   strings.TrimSpace(r.URL.Query().Get("owner_name")),
		ExecutionID: strings.TrimSpace(r.URL.Query().Get("execution_id")),
		Purpose:     strings.TrimSpace(r.URL.Query().Get("purpose")),
		Status:      strings.TrimSpace(r.URL.Query().Get("status")),
		RunID:       strings.TrimSpace(r.URL.Query().Get("run_id")),
	}
	if activeRaw := strings.TrimSpace(r.URL.Query().Get("active")); activeRaw != "" {
		active, err := strconv.ParseBool(activeRaw)
		if err != nil {
			httputil.BadRequest(w, "[agent-activity] list", "active must be true or false")
			return
		}
		filters.ActiveOnly = active
	}

	items, err := h.service.List(r.Context(), filters)
	if err != nil {
		httputil.InternalError(w, "[agent-activity] list", "failed to list agent activities")
		return
	}

	resp := &apipb.ListAgentActivitiesResponse{Items: make([]*domainpb.AgentActivity, 0, len(items))}
	for _, item := range items {
		resp.Items = append(resp.Items, recordToProto(item))
	}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[agent-activity] list", "failed to encode response")
	}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	activityID := strings.TrimSpace(mux.Vars(r)["activity_id"])
	if activityID == "" {
		httputil.BadRequest(w, "[agent-activity] get", "activity_id is required")
		return
	}
	item, err := h.service.Get(r.Context(), activityID)
	if err != nil {
		if errors.Is(err, errNotFound) {
			httputil.NotFound(w, "[agent-activity] get", "agent activity not found")
			return
		}
		httputil.InternalError(w, "[agent-activity] get", "failed to fetch agent activity")
		return
	}
	if err := httputil.ProtoJSON(w, &apipb.AgentActivityResponse{Activity: recordToProto(item)}); err != nil {
		httputil.InternalError(w, "[agent-activity] get", "failed to encode response")
	}
}
