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
	"swarm-manager/internal/transitionrunner"

	"github.com/gorilla/mux"
)

// Handler exposes HTTP endpoints for goal operations.
type Handler struct {
	service          *Service
	transitionRunner *transitionrunner.Runner
	proposalRecorder WorkflowProposalRecorder
	// goalProposalOps is the proposal vocabulary rendered into goal workflow
	// prompts. It is injected (rather than read from the proposals package,
	// which depends on this one) so the ops an agent is told about stay the
	// ops the server accepts, with no second list to drift.
	goalProposalOps []string
}

// SetGoalProposalOps declares the proposal vocabulary shown to goal workflows.
func (h *Handler) SetGoalProposalOps(ops []string) {
	h.goalProposalOps = append([]string(nil), ops...)
}

func (h *Handler) supportedOps() []any {
	out := make([]any, 0, len(h.goalProposalOps))
	for _, op := range h.goalProposalOps {
		out = append(out, op)
	}
	return out
}

type WorkflowProposalRecorder interface {
	RecordGoalWorkflowProposals(context.Context, GoalWorkflowProposal) (GoalWorkflowProposalReceipt, error)
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
	// Precedes the {name} routes so "workflow-pending" is not read as a goal name.
	r.HandleFunc("/api/v1/goals/workflow-pending", h.ListPendingWorkflowRuns).Methods("GET")
	r.HandleFunc("/api/v1/goals/{name}/targets", h.AddTargets).Methods("POST")
	r.HandleFunc("/api/v1/goals/{name}/targets", h.RemoveTargets).Methods("DELETE")
	r.HandleFunc("/api/v1/goals/{name}/archive-item", h.Archive).Methods("PATCH")
	r.HandleFunc("/api/v1/goals/{name}/close-out", h.CloseOut).Methods("POST")
	// Deprecated transition aliases: use TransitionService.StartTransition/ApplyTransition.
	r.HandleFunc("/api/v1/goals/{name}/plan-run", h.StartPlan).Methods("POST")
	r.HandleFunc("/api/v1/goals/{name}/discover-run", h.StartDiscover).Methods("POST")
	r.HandleFunc("/api/v1/goals/{name}/milestones", h.CreateMilestone).Methods("POST")
	r.HandleFunc("/api/v1/goals/{name}/milestones/{milestone}", h.UpdateMilestone).Methods("PUT")
	r.HandleFunc("/api/v1/goals/{name}/milestones/{milestone}", h.ArchiveMilestone).Methods("DELETE")
	r.HandleFunc("/api/v1/goals/{name}/milestones/{milestone}/items", h.AssignMilestoneItems).Methods("POST")
	r.HandleFunc("/api/v1/goals/{name}/milestones/{milestone}/items", h.UnassignMilestoneItems).Methods("DELETE")
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
	if h.transitionRunner == nil {
		apierr.MapError(w, "[goals] workflow", apierr.Unavailable("transition runner is not configured"))
		return
	}
	goal, err := h.service.Get(nameVar(r))
	if err != nil {
		mapServiceError(w, "[goals] workflow", err)
		return
	}
	subjectRef := goal.Goal.Name
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
		subjectRef = goal.Goal.Name + "/" + name
	}
	activity := &transitionrunner.Activity{OwnerType: "scenario", OwnerKind: "goal", OwnerName: goal.Goal.Name, Purpose: "process"}
	if transition == "milestone.review" {
		activity = &transitionrunner.Activity{OwnerType: "milestone", OwnerKind: "goal", OwnerName: subjectRef, Purpose: "milestone_review"}
	}
	start, err := h.transitionRunner.StartWith(r.Context(), transition, subjectRef, transitionrunner.PreparedInput{FirstRunNodeID: node, Activity: activity})
	if err != nil {
		apierr.MapError(w, "[goals] workflow", apierr.BadGateway("start workflow: %s", err))
		return
	}
	writeJSON(w, "[goals] workflow", map[string]any{"execution_id": start.ExecutionID, "definition_digest": start.DefinitionDigest})
}

// ListPendingWorkflowRuns reports terminal goal workflow results that have not
// been applied. Without it a stalled apply hop is invisible: results accumulate
// on disk and every surface still reads as if the workflow never ran.
func (h *Handler) ListPendingWorkflowRuns(w http.ResponseWriter, _ *http.Request) {
	pending, err := h.ListPendingWorkflows()
	if err != nil {
		apierr.MapError(w, "[goals] workflow-pending", apierr.Internal("failed to list pending goal workflows"))
		return
	}
	writeJSON(w, "[goals] workflow-pending", map[string]any{"pending": pending})
}

// ApplyWorkflow validates and projects a terminal workflow result into the
// session-backed operator proposal store. It never mutates the goal graph.
func (h *Handler) ApplyWorkflow(w http.ResponseWriter, r *http.Request) {
	goalName, executionID := nameVar(r), strings.TrimSpace(mux.Vars(r)["execution_id"])
	applied, err := h.ApplyWorkflowResult(r.Context(), goalName, executionID)
	if err != nil {
		apierr.MapError(w, "[goals] apply workflow", apierr.Conflict("%s", err))
		return
	}
	writeJSON(w, "[goals] apply workflow", applied)
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

// CloseOut is the operator-only endpoint for asserting the delivered goal
// outcome. Service validation rejects incomplete or unverified milestones.
func (h *Handler) CloseOut(w http.ResponseWriter, r *http.Request) {
	goal, err := h.service.CloseOut(nameVar(r))
	if err != nil {
		mapServiceError(w, "[goals] close-out", err)
		return
	}
	writeJSON(w, "[goals] close-out", goal)
}

// CloseOutGoal is the domain mutation for goal.close_out.
func (h *Handler) CloseOutGoal(name string) (*Goal, error) { return h.service.CloseOut(name) }

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

func (h *Handler) CreateMilestone(w http.ResponseWriter, r *http.Request) {
	var milestone Milestone
	if err := json.NewDecoder(r.Body).Decode(&milestone); err != nil {
		apierr.MapError(w, "[goals] create-milestone", apierr.BadRequest("invalid request body"))
		return
	}
	result, err := h.service.CreateMilestone(nameVar(r), milestone)
	if err != nil {
		mapServiceError(w, "[goals] create-milestone", err)
		return
	}
	writeJSON(w, "[goals] create-milestone", result)
}

func (h *Handler) UpdateMilestone(w http.ResponseWriter, r *http.Request) {
	var milestone Milestone
	if err := json.NewDecoder(r.Body).Decode(&milestone); err != nil {
		apierr.MapError(w, "[goals] update-milestone", apierr.BadRequest("invalid request body"))
		return
	}
	milestone.Name = strings.TrimSpace(mux.Vars(r)["milestone"])
	result, err := h.service.UpdateMilestone(nameVar(r), milestone)
	if err != nil {
		mapServiceError(w, "[goals] update-milestone", err)
		return
	}
	writeJSON(w, "[goals] update-milestone", result)
}

func (h *Handler) ArchiveMilestone(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.ArchiveMilestone(nameVar(r), strings.TrimSpace(mux.Vars(r)["milestone"]))
	if err != nil {
		mapServiceError(w, "[goals] archive-milestone", err)
		return
	}
	writeJSON(w, "[goals] archive-milestone", result)
}

func (h *Handler) AssignMilestoneItems(w http.ResponseWriter, r *http.Request) {
	h.updateMilestoneItems(w, r, true)
}

func (h *Handler) UnassignMilestoneItems(w http.ResponseWriter, r *http.Request) {
	h.updateMilestoneItems(w, r, false)
}

func (h *Handler) updateMilestoneItems(w http.ResponseWriter, r *http.Request, assign bool) {
	var req targetsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.MapError(w, "[goals] milestone-items", apierr.BadRequest("invalid request body"))
		return
	}
	milestone := strings.TrimSpace(mux.Vars(r)["milestone"])
	var result *GoalWithScope
	var err error
	if assign {
		result, err = h.service.AssignMilestoneItems(nameVar(r), milestone, req.Targets)
	} else {
		result, err = h.service.UnassignMilestoneItems(nameVar(r), milestone, req.Targets)
	}
	if err != nil {
		mapServiceError(w, "[goals] milestone-items", err)
		return
	}
	writeJSON(w, "[goals] milestone-items", result)
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
