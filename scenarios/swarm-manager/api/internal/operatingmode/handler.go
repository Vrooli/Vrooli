package operatingmode

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/initiativelock"

	"github.com/gorilla/mux"
)

type Handler struct {
	service *Service
}

type startPhaseBody struct {
	Phase       string `json:"phase,omitempty"`
	Note        string `json:"note,omitempty"`
	Override    bool   `json:"override,omitempty"`
	RequestedBy string `json:"requested_by,omitempty"`
}

type switchModeBody struct {
	Mode                       string `json:"mode"`
	CancelActiveItemExecutions bool   `json:"cancel_active_item_executions,omitempty"`
	RequestedBy                string `json:"requested_by,omitempty"`
}

type completeItemsBody struct {
	Mode        string   `json:"mode,omitempty"`
	RunID       string   `json:"run_id"`
	ItemRefs    []string `json:"item_refs,omitempty"`
	RequestedBy string   `json:"requested_by,omitempty"`
}

type applyBacklogSyncBody struct {
	Mode                string   `json:"mode,omitempty"`
	RunID               string   `json:"run_id"`
	AcceptedMutationIDs []string `json:"accepted_mutation_ids,omitempty"`
	RequestedBy         string   `json:"requested_by,omitempty"`
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/operating-modes", h.Catalog).Methods("GET")
	r.HandleFunc("/api/v1/operating-modes/{mode}", h.GetMode).Methods("GET")
	r.HandleFunc("/api/v1/operating-modes/{mode}", h.UpdateMode).Methods("PATCH")
	r.HandleFunc("/api/v1/operating-modes/{mode}/simulate", h.SimulateMode).Methods("POST")
	r.HandleFunc("/api/v1/initiatives/{name}/operating-mode/workspace", h.Workspace).Methods("GET")
	r.HandleFunc("/api/v1/initiatives/{name}/operating-mode/switch", h.SwitchMode).Methods("POST")
	r.HandleFunc("/api/v1/initiatives/{name}/operating-mode/phases/{phase}/start", h.StartPhase).Methods("POST")
	r.HandleFunc("/api/v1/initiatives/{name}/operating-mode/rounds/{round:[0-9]+}/refresh", h.RefreshRound).Methods("POST")
	r.HandleFunc("/api/v1/initiatives/{name}/operating-mode/rounds/{round:[0-9]+}/cancel", h.CancelRound).Methods("POST")
	r.HandleFunc("/api/v1/initiatives/{name}/operating-mode/rounds/{round:[0-9]+}/complete-items", h.CompleteItems).Methods("POST")
	r.HandleFunc("/api/v1/initiatives/{name}/operating-mode/rounds/{round:[0-9]+}/apply-backlog-sync", h.ApplyBacklogSync).Methods("POST")
}

// POST /api/v1/operating-modes/{mode}/simulate — deterministically walks a
// phase-based operating mode against an ephemeral mock initiative. The service
// returns the same structured-result projection live rounds use, without
// acquiring locks, spawning agents, or writing artifacts.
func (h *Handler) SimulateMode(w http.ResponseWriter, r *http.Request) {
	rawMode := strings.TrimSpace(mux.Vars(r)["mode"])
	if rawMode == "" {
		apierr.MapError(w, "[operating-mode] simulate", apierr.BadRequest("mode is required"))
		return
	}
	if !ValidateMode(rawMode) {
		apierr.MapError(w, "[operating-mode] simulate", apierr.NotFound("unknown operating mode %q", rawMode))
		return
	}
	result, err := h.service.SimulateMode(r.Context(), NormalizeMode(rawMode))
	if err != nil {
		mapOperatingModeError(w, "[operating-mode] simulate", err)
		return
	}
	writeOperatingModeJSON(w, "[operating-mode] simulate", result)
}

// writeOperatingModeJSON encodes payload as the success response, mapping an
// encode failure to a consistent Internal error under the given log label.
func writeOperatingModeJSON(w http.ResponseWriter, label string, payload any) {
	if err := httputil.JSON(w, payload); err != nil {
		apierr.MapError(w, label, apierr.Internal("failed to encode response"))
	}
}

// GET /api/v1/operating-modes — list all registered modes with overlay
// merged in and current usage counts. Backs the sidebar Operating Modes tab.
func (h *Handler) Catalog(w http.ResponseWriter, _ *http.Request) {
	catalog, err := h.service.Catalog()
	if err != nil {
		mapOperatingModeError(w, "[operating-mode] catalog", err)
		return
	}
	writeOperatingModeJSON(w, "[operating-mode] catalog", catalog)
}

// GET /api/v1/operating-modes/{mode} — single mode detail including the
// list of initiatives currently bound to it. Backs the operating-mode
// details page.
func (h *Handler) GetMode(w http.ResponseWriter, r *http.Request) {
	rawMode := strings.TrimSpace(mux.Vars(r)["mode"])
	if rawMode == "" {
		apierr.MapError(w, "[operating-mode] get mode", apierr.BadRequest("mode is required"))
		return
	}
	if !ValidateMode(rawMode) {
		apierr.MapError(w, "[operating-mode] get mode", apierr.NotFound("unknown operating mode %q", rawMode))
		return
	}
	detail, err := h.service.GetMode(NormalizeMode(rawMode))
	if err != nil {
		mapOperatingModeError(w, "[operating-mode] get mode", err)
		return
	}
	writeOperatingModeJSON(w, "[operating-mode] get mode", detail)
}

type updateModeBody struct {
	Label       *string `json:"label,omitempty"`
	Description *string `json:"description,omitempty"`
}

// PATCH /api/v1/operating-modes/{mode} — persist user-editable overlay
// fields (label, description). Pointer semantics: nil leaves field unchanged,
// "" rejected for label, present-but-empty description allowed (clears the
// description override).
func (h *Handler) UpdateMode(w http.ResponseWriter, r *http.Request) {
	rawMode := strings.TrimSpace(mux.Vars(r)["mode"])
	if rawMode == "" {
		apierr.MapError(w, "[operating-mode] update mode", apierr.BadRequest("mode is required"))
		return
	}
	if !ValidateMode(rawMode) {
		apierr.MapError(w, "[operating-mode] update mode", apierr.NotFound("unknown operating mode %q", rawMode))
		return
	}
	var body updateModeBody
	if err := httputil.DecodeJSONStrict(r, &body); err != nil {
		apierr.MapError(w, "[operating-mode] update mode", apierr.BadRequest("invalid request body"))
		return
	}
	override := Override(body)
	if !override.HasChanges() {
		apierr.MapError(w, "[operating-mode] update mode", apierr.BadRequest("at least one of label or description is required"))
		return
	}
	detail, err := h.service.UpdateMode(NormalizeMode(rawMode), override)
	if err != nil {
		if strings.Contains(err.Error(), "blank") {
			apierr.MapError(w, "[operating-mode] update mode", apierr.BadRequest("%s", err.Error()))
			return
		}
		mapOperatingModeError(w, "[operating-mode] update mode", err)
		return
	}
	writeOperatingModeJSON(w, "[operating-mode] update mode", detail)
}

func (h *Handler) Workspace(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(mux.Vars(r)["name"])
	if name == "" {
		apierr.MapError(w, "[operating-mode] workspace", apierr.BadRequest("initiative name is required"))
		return
	}
	workspace, err := h.service.Workspace(r.Context(), name)
	if err != nil {
		mapOperatingModeError(w, "[operating-mode] workspace", err)
		return
	}
	writeOperatingModeJSON(w, "[operating-mode] workspace", workspace)
}

func (h *Handler) SwitchMode(w http.ResponseWriter, r *http.Request) {
	var body switchModeBody
	if err := httputil.DecodeJSONStrict(r, &body); err != nil {
		apierr.MapError(w, "[operating-mode] switch mode", apierr.BadRequest("invalid request body"))
		return
	}
	name := strings.TrimSpace(mux.Vars(r)["name"])
	if name == "" || strings.TrimSpace(body.Mode) == "" {
		apierr.MapError(w, "[operating-mode] switch mode", apierr.BadRequest("initiative name and mode are required"))
		return
	}
	result, err := h.service.SwitchMode(r.Context(), SwitchModeRequest{
		InitiativeName:             name,
		Mode:                       body.Mode,
		CancelActiveItemExecutions: body.CancelActiveItemExecutions,
		RequestedBy:                body.RequestedBy,
	})
	if err != nil {
		mapOperatingModeError(w, "[operating-mode] switch mode", err)
		return
	}
	writeOperatingModeJSON(w, "[operating-mode] switch mode", result)
}

func (h *Handler) StartPhase(w http.ResponseWriter, r *http.Request) {
	var body startPhaseBody
	if err := httputil.DecodeJSONStrict(r, &body); err != nil {
		apierr.MapError(w, "[operating-mode] start phase", apierr.BadRequest("invalid request body"))
		return
	}
	name := strings.TrimSpace(mux.Vars(r)["name"])
	phase := strings.TrimSpace(mux.Vars(r)["phase"])
	if body.Phase != "" && phase == "" {
		phase = body.Phase
	}
	if name == "" || phase == "" {
		apierr.MapError(w, "[operating-mode] start phase", apierr.BadRequest("initiative name and phase are required"))
		return
	}
	round, err := h.service.StartPhase(r.Context(), StartPhaseRequest{
		InitiativeName: name,
		Phase:          phase,
		Note:           body.Note,
		Override:       body.Override,
		RequestedBy:    body.RequestedBy,
	})
	if err != nil {
		mapOperatingModeError(w, "[operating-mode] start phase", err)
		return
	}
	if err := httputil.JSONWithStatus(w, http.StatusAccepted, round); err != nil {
		apierr.MapError(w, "[operating-mode] start phase", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) RefreshRound(w http.ResponseWriter, r *http.Request) {
	name, roundNumber, ok := parseRoundRoute(w, r, "[operating-mode] refresh round")
	if !ok {
		return
	}
	mode, ok := h.resolveRoundActionMode(w, r, name, "", "[operating-mode] refresh round")
	if !ok {
		return
	}
	round, err := h.service.RefreshRound(r.Context(), name, mode, roundNumber)
	if err != nil {
		mapOperatingModeError(w, "[operating-mode] refresh round", err)
		return
	}
	writeOperatingModeJSON(w, "[operating-mode] refresh round", round)
}

func (h *Handler) CancelRound(w http.ResponseWriter, r *http.Request) {
	name, roundNumber, ok := parseRoundRoute(w, r, "[operating-mode] cancel round")
	if !ok {
		return
	}
	mode, ok := h.resolveRoundActionMode(w, r, name, "", "[operating-mode] cancel round")
	if !ok {
		return
	}
	round, err := h.service.CancelRound(r.Context(), name, mode, roundNumber)
	if err != nil {
		mapOperatingModeError(w, "[operating-mode] cancel round", err)
		return
	}
	writeOperatingModeJSON(w, "[operating-mode] cancel round", round)
}

func (h *Handler) CompleteItems(w http.ResponseWriter, r *http.Request) {
	name, roundNumber, ok := parseRoundRoute(w, r, "[operating-mode] complete items")
	if !ok {
		return
	}
	var body completeItemsBody
	if err := httputil.DecodeJSONStrict(r, &body); err != nil {
		apierr.MapError(w, "[operating-mode] complete items", apierr.BadRequest("invalid request body"))
		return
	}
	if strings.TrimSpace(body.RunID) == "" {
		apierr.MapError(w, "[operating-mode] complete items", apierr.BadRequest("run_id is required"))
		return
	}
	mode, ok := h.resolveRoundActionMode(w, r, name, body.Mode, "[operating-mode] complete items")
	if !ok {
		return
	}
	result, err := h.service.CompleteItems(r.Context(), CompleteItemsRequest{
		InitiativeName: name,
		Mode:           string(mode),
		Round:          roundNumber,
		RunID:          body.RunID,
		ItemRefs:       body.ItemRefs,
		RequestedBy:    body.RequestedBy,
	})
	if err != nil {
		mapOperatingModeError(w, "[operating-mode] complete items", err)
		return
	}
	writeOperatingModeJSON(w, "[operating-mode] complete items", result)
}

func (h *Handler) ApplyBacklogSync(w http.ResponseWriter, r *http.Request) {
	name, roundNumber, ok := parseRoundRoute(w, r, "[operating-mode] apply backlog sync")
	if !ok {
		return
	}
	var body applyBacklogSyncBody
	if err := httputil.DecodeJSONStrict(r, &body); err != nil {
		apierr.MapError(w, "[operating-mode] apply backlog sync", apierr.BadRequest("invalid request body"))
		return
	}
	if strings.TrimSpace(body.RunID) == "" {
		apierr.MapError(w, "[operating-mode] apply backlog sync", apierr.BadRequest("run_id is required"))
		return
	}
	mode, ok := h.resolveRoundActionMode(w, r, name, body.Mode, "[operating-mode] apply backlog sync")
	if !ok {
		return
	}
	result, err := h.service.ApplyBacklogSync(r.Context(), ApplyBacklogSyncRequest{
		InitiativeName:      name,
		Mode:                string(mode),
		Round:               roundNumber,
		RunID:               body.RunID,
		AcceptedMutationIDs: body.AcceptedMutationIDs,
		RequestedBy:         body.RequestedBy,
	})
	if err != nil {
		mapOperatingModeError(w, "[operating-mode] apply backlog sync", err)
		return
	}
	writeOperatingModeJSON(w, "[operating-mode] apply backlog sync", result)
}

func parseRoundRoute(w http.ResponseWriter, r *http.Request, ctx string) (string, int, bool) {
	vars := mux.Vars(r)
	name := strings.TrimSpace(vars["name"])
	if name == "" {
		apierr.MapError(w, ctx, apierr.BadRequest("initiative name is required"))
		return "", 0, false
	}
	round, err := strconv.Atoi(strings.TrimSpace(vars["round"]))
	if err != nil || round <= 0 {
		apierr.MapError(w, ctx, apierr.BadRequest("round must be a positive integer"))
		return "", 0, false
	}
	return name, round, true
}

func (h *Handler) resolveRoundActionMode(w http.ResponseWriter, r *http.Request, initiativeName, bodyMode, ctx string) (Mode, bool) {
	rawMode := strings.TrimSpace(bodyMode)
	if rawMode == "" {
		rawMode = strings.TrimSpace(r.URL.Query().Get("mode"))
	}
	mode, err := h.service.ResolveRoundActionMode(initiativeName, rawMode)
	if err != nil {
		mapOperatingModeError(w, ctx, err)
		return "", false
	}
	return mode, true
}

func mapOperatingModeError(w http.ResponseWriter, ctx string, err error) {
	var conflict *initiativelock.Conflict
	var activeConflict *ActiveItemExecutionsConflict
	var activeRoundConflict *ActiveOperatingModeRoundConflict
	switch {
	case errors.Is(err, ErrApplyModeNotImplemented):
		apierr.MapError(w, ctx, apierr.NotImplemented("%s", err.Error()).WithCode("apply_mode_not_implemented"))
	case errors.As(err, &activeConflict):
		apierr.MapError(w, ctx, apierr.Conflict("%s", err.Error()).
			WithCode("active_item_executions").
			WithDetails(activeConflict))
	case errors.As(err, &activeRoundConflict):
		apierr.MapError(w, ctx, apierr.Conflict("%s", err.Error()).
			WithCode("active_operating_mode_round").
			WithDetails(activeRoundConflict))
	case errors.As(err, &conflict):
		apierr.MapError(w, ctx, apierr.Conflict("%s", err.Error()).WithDetails(conflict.Holder))
	case errors.Is(err, ErrRoundNotFound), strings.Contains(err.Error(), "not found"):
		apierr.MapError(w, ctx, apierr.NotFound("%s", err.Error()))
	case strings.Contains(err.Error(), "requires"), strings.Contains(err.Error(), "unknown operating mode"),
		strings.Contains(err.Error(), "does not define phase"), strings.Contains(err.Error(), "item-level mode"),
		strings.Contains(err.Error(), "run_id"), strings.Contains(err.Error(), "item_refs"),
		strings.Contains(err.Error(), "must be kind/name"), strings.Contains(err.Error(), "is not a member"),
		strings.Contains(err.Error(), "does not allow"), strings.Contains(err.Error(), "no backlog_sync"),
		strings.Contains(err.Error(), "proposal"), strings.Contains(err.Error(), "mode is required"),
		strings.Contains(err.Error(), "round actions require"):
		apierr.MapError(w, ctx, apierr.BadRequest("%s", err.Error()))
	case errors.Is(err, agentmanager.ErrNotAvailable):
		apierr.MapError(w, ctx, apierr.Unavailable("agent-manager is not available"))
	default:
		apierr.MapError(w, ctx, apierr.Internal("%s", err.Error()))
	}
}
