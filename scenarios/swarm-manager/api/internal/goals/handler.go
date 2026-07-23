package goals

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/transitions"

	"github.com/gorilla/mux"
	"google.golang.org/protobuf/types/known/structpb"
)

// Handler exposes HTTP endpoints for goal operations.
type Handler struct {
	service          *Service
	workflow         WorkflowInvoker
	registry         transitions.Registry
	proposalRecorder WorkflowProposalRecorder
}

type (
	WorkflowInvocation struct {
		Owner, WorkflowKey, IdempotencyKey, FirstRunNodeID                                           string
		Input                                                                                        *structpb.Value
		ActivityOwnerType, ActivityOwnerKind, ActivityOwnerName, ActivityOwnerTitle, ActivityPurpose string
	}
	WorkflowStart      struct{ ExecutionID, RunID, DefinitionDigest string }
	WorkflowCompletion struct {
		ExecutionID, DefinitionDigest string
		Succeeded                     bool
		Input, Output                 *structpb.Value
	}
	WorkflowInvoker interface {
		StartWorkflow(context.Context, WorkflowInvocation) (WorkflowStart, error)
		CollectWorkflow(context.Context, string) (WorkflowCompletion, error)
	}
	WorkflowProposalRecorder interface {
		RecordGoalWorkflowProposals(context.Context, GoalWorkflowProposal) (GoalWorkflowProposalReceipt, error)
	}
)

func (h *Handler) SetWorkflow(invoker WorkflowInvoker, registry transitions.Registry) {
	h.workflow, h.registry = invoker, registry
}

func (h *Handler) SetWorkflowProposalRecorder(recorder WorkflowProposalRecorder) {
	h.proposalRecorder = recorder
}

// NewHandler creates a goals Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers goal API routes. Target routes precede the {name}
// catch-all so gorilla/mux matches them first.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/goals", h.List).Methods("GET")
	r.HandleFunc("/api/v1/goals", h.Create).Methods("POST")
	r.HandleFunc("/api/v1/goals/{name}/targets", h.AddTargets).Methods("POST")
	r.HandleFunc("/api/v1/goals/{name}/targets", h.RemoveTargets).Methods("DELETE")
	r.HandleFunc("/api/v1/goals/{name}/archive-item", h.Archive).Methods("PATCH")
	r.HandleFunc("/api/v1/goals/{name}/plan-run", h.StartPlan).Methods("POST")
	r.HandleFunc("/api/v1/goals/{name}/discover-run", h.StartDiscover).Methods("POST")
	r.HandleFunc("/api/v1/goals/{name}/milestones/{milestone}/review-run", h.StartMilestoneReview).Methods("POST")
	r.HandleFunc("/api/v1/goals/{name}/workflow-runs/{execution_id}/apply", h.ApplyWorkflow).Methods("POST")
	r.HandleFunc("/api/v1/goals/{name}/files", h.ListFiles).Methods("GET")
	r.HandleFunc("/api/v1/goals/{name}/files", h.UploadFile).Methods("POST")
	r.HandleFunc("/api/v1/goals/{name}/files", h.OperateFile).Methods("PATCH")
	r.HandleFunc("/api/v1/goals/{name}/files/{filepath:.*}", h.GetFileContent).Methods("GET")
	r.HandleFunc("/api/v1/goals/{name}", h.Get).Methods("GET")
	r.HandleFunc("/api/v1/goals/{name}", h.Update).Methods("PUT")
	r.HandleFunc("/api/v1/goals/{name}", h.Delete).Methods("DELETE")
}

func (h *Handler) StartPlan(w http.ResponseWriter, r *http.Request) {
	h.startWorkflow(w, r, "goal.plan", "plan")
}

func (h *Handler) StartDiscover(w http.ResponseWriter, r *http.Request) {
	h.startWorkflow(w, r, "goal.discover", "discover")
}

func (h *Handler) StartMilestoneReview(w http.ResponseWriter, r *http.Request) {
	h.startWorkflow(w, r, "milestone.review", "review")
}

func (h *Handler) startWorkflow(w http.ResponseWriter, r *http.Request, transition, node string) {
	if h.workflow == nil {
		apierr.MapError(w, "[goals] workflow", apierr.Unavailable("goal workflow service is unavailable"))
		return
	}
	locator, err := h.registry.ResolveWorkflow(transition)
	if err != nil {
		apierr.MapError(w, "[goals] workflow", apierr.Conflict("%s", err))
		return
	}
	goal, err := h.service.Get(nameVar(r))
	if err != nil {
		mapServiceError(w, "[goals] workflow", err)
		return
	}
	entity := map[string]any{"kind": "goal", "name": goal.Goal.Name, "version": goal.Goal.Updated}
	if transition == "milestone.review" {
		name := strings.TrimSpace(mux.Vars(r)["milestone"])
		found := false
		for _, milestone := range goal.Goal.Milestones {
			if milestone.Name == name {
				found = true
				break
			}
		}
		if !found {
			apierr.MapError(w, "[goals] workflow", apierr.NotFound("milestone not found"))
			return
		}
		entity = map[string]any{"kind": "milestone", "name": name, "goalName": goal.Goal.Name, "version": goal.Goal.Updated}
	}
	snapshot, err := workflowSnapshot(goal)
	if err != nil {
		apierr.MapError(w, "[goals] workflow", apierr.Internal("encode goal snapshot"))
		return
	}
	input, err := structpb.NewValue(map[string]any{"entity": entity, "snapshot": snapshot})
	if err != nil {
		apierr.MapError(w, "[goals] workflow", apierr.BadRequest("build workflow input"))
		return
	}
	activityType, activityKind, activityName, activityPurpose := "scenario", "goal", goal.Goal.Name, "process"
	if milestone := strings.TrimSpace(mux.Vars(r)["milestone"]); milestone != "" {
		activityType, activityKind, activityName, activityPurpose = "milestone", "goal", goal.Goal.Name+"/"+milestone, "milestone_review"
	}
	start, err := h.workflow.StartWorkflow(r.Context(), WorkflowInvocation{
		Owner: locator.Owner, WorkflowKey: locator.Key, Input: input,
		IdempotencyKey: transition + "/" + goal.Goal.Name + "/" + goal.Goal.Updated, FirstRunNodeID: node,
		ActivityOwnerType: activityType, ActivityOwnerKind: activityKind, ActivityOwnerName: activityName, ActivityPurpose: activityPurpose,
	})
	if err != nil {
		apierr.MapError(w, "[goals] workflow", apierr.BadGateway("start workflow: %s", err))
		return
	}
	if err := h.writeWorkflowPending(goal.Goal.Name, workflowPending{ExecutionID: start.ExecutionID, DefinitionDigest: start.DefinitionDigest, Transition: transition, GoalVersion: goal.Goal.Updated, Milestone: strings.TrimSpace(mux.Vars(r)["milestone"])}); err != nil {
		apierr.MapError(w, "[goals] workflow", apierr.Internal("persist workflow correlation"))
		return
	}
	writeJSON(w, "[goals] workflow", map[string]any{"execution_id": start.ExecutionID, "run_id": start.RunID, "definition_digest": start.DefinitionDigest})
}

// ApplyWorkflow validates and projects a terminal workflow result into the
// session-backed operator proposal store. It never mutates the goal graph.
func (h *Handler) ApplyWorkflow(w http.ResponseWriter, r *http.Request) {
	result, err := h.applyWorkflow(r.Context(), nameVar(r), strings.TrimSpace(mux.Vars(r)["execution_id"]))
	if err != nil {
		apierr.MapError(w, "[goals] apply workflow", apierr.Conflict("%s", err))
		return
	}
	writeJSON(w, "[goals] apply workflow", result)
}

func workflowSnapshot(goal *GoalWithScope) (map[string]any, error) {
	raw, err := json.Marshal(goal)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	err = json.Unmarshal(raw, &out)
	return out, err
}

func (h *Handler) List(w http.ResponseWriter, _ *http.Request) {
	items, err := h.service.List()
	if err != nil {
		apierr.MapError(w, "[goals] list", apierr.Internal("failed to list goals"))
		return
	}
	writeJSON(w, "[goals] list", map[string]any{"items": items})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.MapError(w, "[goals] create", apierr.BadRequest("invalid request body"))
		return
	}
	result, err := h.service.Create(req)
	if err != nil {
		mapServiceError(w, "[goals] create", err)
		return
	}
	if err := httputil.JSONWithStatus(w, http.StatusCreated, result); err != nil {
		apierr.MapError(w, "[goals] create", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.Get(nameVar(r))
	if err != nil {
		mapServiceError(w, "[goals] get", err)
		return
	}
	writeJSON(w, "[goals] get", result)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.MapError(w, "[goals] update", apierr.BadRequest("invalid request body"))
		return
	}
	result, err := h.service.Update(nameVar(r), req)
	if err != nil {
		mapServiceError(w, "[goals] update", err)
		return
	}
	writeJSON(w, "[goals] update", result)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(nameVar(r)); err != nil {
		mapServiceError(w, "[goals] delete", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	g, err := h.service.Archive(nameVar(r))
	if err != nil {
		mapServiceError(w, "[goals] archive", err)
		return
	}
	writeJSON(w, "[goals] archive", g)
}

type targetsRequest struct {
	Targets []string `json:"targets"`
}

func (h *Handler) AddTargets(w http.ResponseWriter, r *http.Request) {
	var req targetsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.MapError(w, "[goals] add-targets", apierr.BadRequest("invalid request body"))
		return
	}
	result, err := h.service.AddTargets(nameVar(r), req.Targets)
	if err != nil {
		mapServiceError(w, "[goals] add-targets", err)
		return
	}
	writeJSON(w, "[goals] add-targets", result)
}

func (h *Handler) RemoveTargets(w http.ResponseWriter, r *http.Request) {
	var req targetsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.MapError(w, "[goals] remove-targets", apierr.BadRequest("invalid request body"))
		return
	}
	result, err := h.service.RemoveTargets(nameVar(r), req.Targets)
	if err != nil {
		mapServiceError(w, "[goals] remove-targets", err)
		return
	}
	writeJSON(w, "[goals] remove-targets", result)
}

func nameVar(r *http.Request) string {
	return strings.TrimSpace(mux.Vars(r)["name"])
}

func writeJSON(w http.ResponseWriter, ctx string, payload any) {
	if err := httputil.JSON(w, payload); err != nil {
		apierr.MapError(w, ctx, apierr.Internal("failed to encode response"))
	}
}

// mapServiceError maps a service error to the right HTTP status: validation
// errors become 400, not-found becomes 404, everything else 500.
func mapServiceError(w http.ResponseWriter, ctx string, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		apierr.MapError(w, ctx, apierr.BadRequest("%s", strings.TrimPrefix(err.Error(), "goal validation error: ")))
	case strings.Contains(err.Error(), "not found"):
		apierr.MapError(w, ctx, apierr.NotFound("%s", err.Error()))
	default:
		slog.Error("goals handler error", "ctx", ctx, "err", err)
		apierr.MapError(w, ctx, apierr.Internal("goal operation failed"))
	}
}
