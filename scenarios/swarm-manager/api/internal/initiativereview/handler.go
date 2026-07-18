package initiativereview

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/review"

	"github.com/gorilla/mux"
)

// Handler mounts the initiative review HTTP endpoints onto a router. The
// handler is transport-only — all domain logic lives on Service.
type Handler struct {
	service *Service
}

// NewHandler wires a Service into an HTTP surface.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes mounts the review endpoints under the initiative namespace.
// The route shape mirrors backlog review so clients can treat them
// symmetrically.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/initiatives/{name}/review", h.listRounds).Methods("GET")
	r.HandleFunc("/api/v1/initiatives/{name}/review/trigger", h.trigger).Methods("POST")
	r.HandleFunc("/api/v1/initiatives/{name}/review/decide", h.decide).Methods("POST")
	r.HandleFunc("/api/v1/initiatives/{name}/review/decisions", h.listDecisions).Methods("GET")
	r.HandleFunc("/api/v1/initiatives/{name}/review/{round:[0-9]+}", h.getRound).Methods("GET")
	r.HandleFunc("/api/v1/initiatives/{name}/review/{round:[0-9]+}/workflow/apply", h.applyWorkflow).Methods("POST")
}

func (h *Handler) applyWorkflow(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(mux.Vars(r)["name"])
	round, err := strconv.Atoi(mux.Vars(r)["round"])
	if name == "" || err != nil || round <= 0 {
		apierr.MapError(w, "[initiative-review] workflow apply", apierr.BadRequest("initiative name and round are required"))
		return
	}
	applied, idempotent, err := h.service.ApplyWorkflowRound(r.Context(), name, round)
	if err != nil {
		apierr.MapError(w, "[initiative-review] workflow apply", apierr.Internal("apply workflow: %s", err.Error()))
		return
	}
	_ = httputil.JSONWithStatus(w, http.StatusOK, map[string]any{"round": applied, "idempotent": idempotent})
}

func (h *Handler) listRounds(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(mux.Vars(r)["name"])
	if name == "" {
		apierr.MapError(w, "[initiative-review] list", apierr.BadRequest("initiative name is required"))
		return
	}
	rounds, err := h.service.ListRounds(name)
	if err != nil {
		apierr.MapError(w, "[initiative-review] list", apierr.Internal("list rounds: %s", err.Error()))
		return
	}
	if rounds == nil {
		rounds = []review.Round{}
	}
	if err := httputil.JSONWithStatus(w, http.StatusOK, map[string]any{"rounds": rounds}); err != nil {
		apierr.MapError(w, "[initiative-review] list", apierr.Internal("encode response"))
	}
}

func (h *Handler) getRound(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := strings.TrimSpace(vars["name"])
	if name == "" {
		apierr.MapError(w, "[initiative-review] get", apierr.BadRequest("initiative name is required"))
		return
	}
	roundNum, err := strconv.Atoi(vars["round"])
	if err != nil || roundNum <= 0 {
		apierr.MapError(w, "[initiative-review] get", apierr.BadRequest("invalid round number"))
		return
	}
	round, err := h.service.GetRound(name, roundNum)
	if err != nil {
		apierr.MapError(w, "[initiative-review] get", apierr.Internal("load round: %s", err.Error()))
		return
	}
	if round == nil {
		apierr.MapError(w, "[initiative-review] get", apierr.NotFound("round %d not found", roundNum))
		return
	}
	if err := httputil.JSONWithStatus(w, http.StatusOK, round); err != nil {
		apierr.MapError(w, "[initiative-review] get", apierr.Internal("encode response"))
	}
}

func (h *Handler) trigger(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(mux.Vars(r)["name"])
	if name == "" {
		apierr.MapError(w, "[initiative-review] trigger", apierr.BadRequest("initiative name is required"))
		return
	}
	result, err := h.service.TriggerIfReady(r.Context(), name)
	if err != nil {
		apierr.MapError(w, "[initiative-review] trigger", apierr.Internal("trigger review: %s", err.Error()))
		return
	}
	status := http.StatusAccepted
	if !result.Started {
		// 200 OK + reason so the caller knows why nothing happened. 409
		// would imply the request was rejected; here it's just a no-op
		// (already in review, no items, etc.).
		status = http.StatusOK
	}
	if err := httputil.JSONWithStatus(w, status, result); err != nil {
		apierr.MapError(w, "[initiative-review] trigger", apierr.Internal("encode response"))
	}
}

func (h *Handler) decide(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(mux.Vars(r)["name"])
	if name == "" {
		apierr.MapError(w, "[initiative-review] decide", apierr.BadRequest("initiative name is required"))
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		apierr.MapError(w, "[initiative-review] decide", apierr.BadRequest("read body: %s", err.Error()))
		return
	}
	var req DecideRequest
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			apierr.MapError(w, "[initiative-review] decide", apierr.BadRequest("invalid request body: %s", err.Error()))
			return
		}
	}
	verdict, err := NormalizeVerdict(req.Verdict)
	if err != nil {
		apierr.MapError(w, "[initiative-review] decide", apierr.BadRequest("%s", err.Error()))
		return
	}
	decidedBy := strings.TrimSpace(req.DecidedBy)
	if decidedBy == "" {
		decidedBy = "user"
	}
	resp, err := h.service.Decide(r.Context(), name, verdict, req.Rationale, decidedBy)
	if err != nil {
		apierr.MapError(w, "[initiative-review] decide", apierr.BadRequest("%s", err.Error()))
		return
	}
	if err := httputil.JSONWithStatus(w, http.StatusOK, resp); err != nil {
		apierr.MapError(w, "[initiative-review] decide", apierr.Internal("encode response"))
	}
}

func (h *Handler) listDecisions(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(mux.Vars(r)["name"])
	if name == "" {
		apierr.MapError(w, "[initiative-review] decisions", apierr.BadRequest("initiative name is required"))
		return
	}
	decisions, err := h.service.ListDecisions(name)
	if err != nil {
		apierr.MapError(w, "[initiative-review] decisions", apierr.Internal("list decisions: %s", err.Error()))
		return
	}
	if decisions == nil {
		decisions = []DecisionRecord{}
	}
	if err := httputil.JSONWithStatus(w, http.StatusOK, map[string]any{"decisions": decisions}); err != nil {
		apierr.MapError(w, "[initiative-review] decisions", apierr.Internal("encode response"))
	}
}
