package backlog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/planclient"
	"swarm-manager/internal/transitionrunner"

	"github.com/gorilla/mux"
	"google.golang.org/protobuf/types/known/structpb"
)

type planAuthorResult struct {
	Outcome       string `json:"outcome"`
	CandidatePlan string `json:"candidatePlan,omitempty"`
	Reason        string `json:"reason,omitempty"`
}

// RegisterTransitionAdapter contributes plan.author's immutable snapshot and
// terminal plan-reference binding to the shared runner.
func (h *Handler) RegisterTransitionAdapter(registrar transitionrunner.Registrar) {
	registrar.RegisterInput("plan.author", h.buildPlanAuthorInput)
	registrar.RegisterInput("plan.repair", h.buildPlanRepairInput)
	registrar.RegisterApply("bind_validated_plan_ref", h.applyValidatedPlanRef)
}

// StartPlanAuthor starts the declared plan.author workflow from the current
// immutable backlog snapshot. The resulting execution is visible in Activity;
// binding a candidate plan remains a separate, explicit terminal action.
func (h *Handler) StartPlanAuthor(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "plan-author")
	if !ok {
		return
	}
	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		apierr.MapError(w, "[backlog] plan-author", apierr.NotFound("backlog item not found"))
		return
	}
	if item.PlanRef != nil {
		apierr.MapError(w, "[backlog] plan-author", apierr.Conflict("backlog item already has a plan_ref"))
		return
	}
	if h.transitionRunner == nil {
		apierr.MapError(w, "[backlog] plan-author", apierr.Unavailable("transition runner is not configured"))
		return
	}
	started, err := h.transitionRunner.Start(r.Context(), "plan.author", backlogSubjectRef(item.Kind, item.Name))
	if err != nil {
		apierr.MapError(w, "[backlog] plan-author", apierr.Conflict("start plan author: %s", err))
		return
	}
	_ = httputil.JSON(w, map[string]any{
		"execution_id":      started.ExecutionID,
		"definition_digest": started.DefinitionDigest,
	})
}

func planAuthorProvenancePath(itemDir, executionID string) string {
	return filepath.Join(itemDir, "plan-author-provenance-"+executionID+".json")
}

// buildPlanAuthorInput snapshots the item and its answered workshop history;
// no prompt is assembled in Swarm and the workflow receives no write authority.
func (h *Handler) buildPlanAuthorInput(_ context.Context, subjectRef string) (transitionrunner.Snapshot, error) {
	kind, name, err := parseBacklogSubjectRef(subjectRef)
	if err != nil {
		return transitionrunner.Snapshot{}, err
	}
	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		return transitionrunner.Snapshot{}, err
	}
	if item.PlanRef != nil {
		return transitionrunner.Snapshot{}, errors.New("backlog item already has a plan_ref")
	}
	// StructPB accepts JSON-shaped values only. Round-trip the domain structs so
	// the typed immutable snapshot stays bounded and transport-neutral.
	snapshotRaw, err := json.Marshal(map[string]any{"item": item})
	if err != nil {
		return transitionrunner.Snapshot{}, err
	}
	var snapshot any
	if err := json.Unmarshal(snapshotRaw, &snapshot); err != nil {
		return transitionrunner.Snapshot{}, err
	}
	input, err := structpb.NewValue(map[string]any{"entity": map[string]any{"kind": string(item.Kind), "name": item.Name, "version": immutableBacklogSnapshotVersion(item)}, "snapshot": snapshot})
	if err != nil {
		return transitionrunner.Snapshot{}, fmt.Errorf("plan author input: %w", err)
	}
	return transitionrunner.SnapshotFromSubject(input, item, nil)
}

// ApplyPlanAuthor is the single domain mutation boundary for plan.author.
func (h *Handler) ApplyPlanAuthor(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "plan-author-apply")
	if !ok {
		return
	}
	executionID := strings.TrimSpace(mux.Vars(r)["executionID"])
	if executionID == "" {
		apierr.MapError(w, "[backlog] plan-author", apierr.BadRequest("workflow execution id is required"))
		return
	}
	if h.transitionRunner == nil {
		apierr.MapError(w, "[backlog] plan-author", apierr.Unavailable("transition runner is not configured"))
		return
	}
	correlation, err := h.transitionRunner.Apply(r.Context(), "plan.author", executionID)
	if err != nil {
		apierr.MapError(w, "[backlog] plan-author", apierr.Conflict("%s", err))
		return
	}
	if correlation.SubjectRef != backlogSubjectRef(kind, name) {
		apierr.MapError(w, "[backlog] plan-author", apierr.Conflict("plan author workflow belongs to a different backlog item"))
		return
	}
	result, err := h.planAuthorResponse(kind, name, executionID)
	if err != nil {
		apierr.MapError(w, "[backlog] plan-author", apierr.Internal("read plan author provenance"))
		return
	}
	_ = httputil.JSON(w, result)
}

func (h *Handler) planAuthorResponse(kind BacklogKind, name, executionID string) (map[string]any, error) {
	data, err := os.ReadFile(planAuthorProvenancePath(h.store.ItemDir(kind, name), executionID))
	if err != nil {
		return nil, err
	}
	var response map[string]any
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}
	return response, nil
}

func (h *Handler) applyValidatedPlanRef(ctx context.Context, subjectRef string, outcome transitionrunner.Outcome) error {
	if outcome.TransitionKey == "plan.repair" {
		return h.applyPlanRepairOutcome(ctx, subjectRef, outcome)
	}
	if outcome.TransitionKey != "plan.author" {
		return fmt.Errorf("bind_validated_plan_ref does not support transition %q", outcome.TransitionKey)
	}
	if h.planClient == nil {
		return errors.New("plan author is not configured")
	}
	kind, name, err := parseBacklogSubjectRef(subjectRef)
	if err != nil {
		return err
	}
	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		return err
	}
	itemDir := h.store.ItemDir(kind, name)
	provenancePath := planAuthorProvenancePath(itemDir, outcome.ExecutionID)
	if data, readErr := os.ReadFile(provenancePath); readErr == nil {
		var existing map[string]any
		if json.Unmarshal(data, &existing) == nil {
			existing["idempotent"] = true
			return nil
		}
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return readErr
	}
	if item.PlanRef != nil {
		return errors.New("backlog item acquired a plan_ref after plan author start")
	}
	result, err := decodePlanAuthorOutcome(outcome)
	if err != nil {
		return err
	}
	response := map[string]any{"execution_id": outcome.ExecutionID, "outcome": result.Outcome, "reason": result.Reason, "applied_at": time.Now().UTC().Format(time.RFC3339Nano)}
	if result.Outcome == "ready" {
		plan, importErr := h.planClient.ImportPlan(ctx, planclient.ImportPlanInput{Markdown: result.CandidatePlan, SourcePath: "swarm-manager:author/" + outcome.ExecutionID})
		if importErr != nil || plan == nil || strings.TrimSpace(plan.GetId()) == "" {
			return fmt.Errorf("import candidate plan: %w", importErr)
		}
		rendered, renderErr := h.planClient.RenderMarkdown(ctx, plan.GetId(), true)
		if renderErr != nil {
			return renderErr
		}
		if rendered.QualityStatus != "pass" {
			return fmt.Errorf("canonical authored plan quality is %q", rendered.QualityStatus)
		}
		item.PlanRef = &PlanRef{Provider: PlanRefProviderPlanManager, PlanID: plan.GetId(), Slug: plan.GetSlug(), Role: PlanRefRoleExecutionSpec}
		item.PlanAcceptance = nil
		item.Updated = time.Now().UTC().Format(time.RFC3339)
		if err := h.store.SaveItem(item); err != nil {
			return err
		}
		response["plan_id"], response["plan_slug"] = plan.GetId(), plan.GetSlug()
	}
	if err := writeJSONRedacted(provenancePath, response); err != nil {
		return err
	}
	return nil
}

func decodePlanAuthorResult(output *structpb.Value) (planAuthorResult, error) {
	if output == nil {
		return planAuthorResult{}, errors.New("plan author output is missing")
	}
	payload, ok := output.AsInterface().(map[string]any)
	if !ok {
		return planAuthorResult{}, errors.New("plan author output is invalid")
	}
	raw, ok := payload["result"]
	if !ok {
		return planAuthorResult{}, errors.New("plan author result is missing")
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return planAuthorResult{}, err
	}
	var result planAuthorResult
	if err := json.Unmarshal(data, &result); err != nil {
		return planAuthorResult{}, err
	}
	switch result.Outcome {
	case "ready":
		if strings.TrimSpace(result.CandidatePlan) == "" {
			return planAuthorResult{}, errors.New("ready plan author result is missing candidatePlan")
		}
	case "needs_attention", "abstained":
		if strings.TrimSpace(result.Reason) == "" {
			return planAuthorResult{}, errors.New("plan author terminal result is missing reason")
		}
	default:
		return planAuthorResult{}, fmt.Errorf("unsupported plan author outcome %q", result.Outcome)
	}
	return result, nil
}

func decodePlanAuthorOutcome(outcome transitionrunner.Outcome) (planAuthorResult, error) {
	data, err := json.Marshal(map[string]any{"result": json.RawMessage(outcome.Result)})
	if err != nil {
		return planAuthorResult{}, err
	}
	value := &structpb.Value{}
	if err := value.UnmarshalJSON(data); err != nil {
		return planAuthorResult{}, err
	}
	return decodePlanAuthorResult(value)
}

func backlogSubjectRef(kind BacklogKind, name string) string { return string(kind) + "/" + name }

func parseBacklogSubjectRef(subjectRef string) (BacklogKind, string, error) {
	parts := strings.SplitN(subjectRef, "/", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("invalid backlog subject reference %q", subjectRef)
	}
	kind, err := ParseBacklogKind(parts[0])
	if err != nil {
		return "", "", err
	}
	return kind, parts[1], nil
}
