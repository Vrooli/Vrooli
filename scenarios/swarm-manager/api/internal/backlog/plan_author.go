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

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/planclient"

	"github.com/gorilla/mux"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

type planAuthorPending struct {
	ExecutionID      string `json:"execution_id"`
	DefinitionDigest string `json:"definition_digest"`
	EntityVersion    string `json:"entity_version"`
}

type planAuthorResult struct {
	Outcome       string `json:"outcome"`
	CandidatePlan string `json:"candidatePlan,omitempty"`
	Reason        string `json:"reason,omitempty"`
}

func planAuthorPendingPath(itemDir, executionID string) string {
	return filepath.Join(itemDir, "workshop", "plan-author-pending-"+executionID+".json")
}

func planAuthorProvenancePath(itemDir, executionID string) string {
	return filepath.Join(itemDir, "plan-author-provenance-"+executionID+".json")
}

// startPlanAuthorWorkflow snapshots the item and its answered workshop history;
// no prompt is assembled in Swarm and the workflow receives no write authority.
func (h *Handler) startPlanAuthorWorkflow(ctx context.Context, item BacklogItem) (agentmanager.WorkflowStart, error) {
	if h.planAuthorWorkflow == nil {
		return agentmanager.WorkflowStart{}, agentmanager.ErrNotAvailable
	}
	workflow, err := h.resolveWorkflow("plan.author")
	if err != nil {
		return agentmanager.WorkflowStart{}, err
	}
	itemDir := h.store.ItemDir(item.Kind, item.Name)
	version := immutableBacklogSnapshotVersion(item)
	// StructPB accepts JSON-shaped values only. Round-trip the domain structs so
	// the typed immutable snapshot stays bounded and transport-neutral.
	snapshotRaw, err := json.Marshal(map[string]any{"item": item})
	if err != nil {
		return agentmanager.WorkflowStart{}, err
	}
	var snapshot any
	if err := json.Unmarshal(snapshotRaw, &snapshot); err != nil {
		return agentmanager.WorkflowStart{}, err
	}
	input, err := structpb.NewValue(map[string]any{"entity": map[string]any{"kind": string(item.Kind), "name": item.Name, "version": version}, "snapshot": snapshot})
	if err != nil {
		return agentmanager.WorkflowStart{}, fmt.Errorf("plan author input: %w", err)
	}
	started, err := h.planAuthorWorkflow.StartWorkflow(ctx, agentmanager.Invocation{Owner: workflow.Owner, WorkflowKey: workflow.Key, Input: input, IdempotencyKey: "plan-author/" + string(item.Kind) + "/" + item.Name + "/" + strings.TrimPrefix(version, "sha256:"), FirstRunNodeID: "author"})
	if err != nil {
		return agentmanager.WorkflowStart{}, err
	}
	if err := os.MkdirAll(filepath.Join(itemDir, "workshop"), 0o750); err != nil {
		return agentmanager.WorkflowStart{}, err
	}
	pending := planAuthorPending{ExecutionID: started.ExecutionID, DefinitionDigest: started.DefinitionDigest, EntityVersion: version}
	if err := writeJSONRedacted(planAuthorPendingPath(itemDir, started.ExecutionID), pending); err != nil {
		return agentmanager.WorkflowStart{}, err
	}
	return started, nil
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
	result, err := h.applyPlanAuthor(r.Context(), kind, name, executionID)
	if err != nil {
		apierr.MapError(w, "[backlog] plan-author", apierr.Conflict("%s", err))
		return
	}
	_ = httputil.JSON(w, result)
}

func (h *Handler) applyPlanAuthor(ctx context.Context, kind BacklogKind, name, executionID string) (map[string]any, error) {
	if h.planAuthorWorkflow == nil || h.planClient == nil {
		return nil, errors.New("plan author is not configured")
	}
	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		return nil, err
	}
	itemDir := h.store.ItemDir(kind, name)
	provenancePath := planAuthorProvenancePath(itemDir, executionID)
	if data, readErr := os.ReadFile(provenancePath); readErr == nil {
		var existing map[string]any
		if json.Unmarshal(data, &existing) == nil {
			existing["idempotent"] = true
			return existing, nil
		}
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return nil, readErr
	}
	pendingData, err := os.ReadFile(planAuthorPendingPath(itemDir, executionID))
	if err != nil {
		return nil, fmt.Errorf("plan author workflow is not pending for this item")
	}
	var pending planAuthorPending
	if err := json.Unmarshal(pendingData, &pending); err != nil || pending.ExecutionID != executionID {
		return nil, errors.New("invalid plan author correlation")
	}
	completion, err := h.planAuthorWorkflow.CollectWorkflow(ctx, executionID)
	if err != nil {
		return nil, err
	}
	if completion.ExecutionID != executionID || completion.DefinitionDigest != pending.DefinitionDigest || completion.Status != domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED {
		return nil, errors.New("plan author workflow terminal result is not applicable")
	}
	if !planAuthorInputMatches(completion.Input, kind, name, pending.EntityVersion) {
		return nil, errors.New("plan author workflow snapshot does not match item")
	}
	if immutableBacklogSnapshotVersion(item) != pending.EntityVersion {
		return nil, errors.New("backlog item changed after plan author start")
	}
	if item.PlanRef != nil {
		return nil, errors.New("backlog item acquired a plan_ref after plan author start")
	}
	result, err := decodePlanAuthorResult(completion.Output)
	if err != nil {
		return nil, err
	}
	response := map[string]any{"execution_id": executionID, "outcome": result.Outcome, "reason": result.Reason, "applied_at": time.Now().UTC().Format(time.RFC3339Nano)}
	if result.Outcome == "ready" {
		plan, importErr := h.planClient.ImportPlan(ctx, planclient.ImportPlanInput{Markdown: result.CandidatePlan, SourcePath: "swarm-manager:author/" + executionID})
		if importErr != nil || plan == nil || strings.TrimSpace(plan.GetId()) == "" {
			return nil, fmt.Errorf("import candidate plan: %w", importErr)
		}
		rendered, renderErr := h.planClient.RenderMarkdown(ctx, plan.GetId(), true)
		if renderErr != nil {
			return nil, renderErr
		}
		if rendered.QualityStatus != "pass" {
			return nil, fmt.Errorf("canonical authored plan quality is %q", rendered.QualityStatus)
		}
		item.PlanRef = &PlanRef{Provider: PlanRefProviderPlanManager, PlanID: plan.GetId(), Slug: plan.GetSlug(), Role: PlanRefRoleExecutionSpec}
		item.PlanAcceptance = nil
		item.Updated = time.Now().UTC().Format(time.RFC3339)
		if err := h.store.SaveItem(item); err != nil {
			return nil, err
		}
		response["plan_id"], response["plan_slug"] = plan.GetId(), plan.GetSlug()
	}
	if err := writeJSONRedacted(provenancePath, response); err != nil {
		return nil, err
	}
	_ = os.Remove(planAuthorPendingPath(itemDir, executionID))
	return response, nil
}

func planAuthorInputMatches(input *structpb.Value, kind BacklogKind, name, version string) bool {
	if input == nil {
		return false
	}
	payload, ok := input.AsInterface().(map[string]any)
	if !ok {
		return false
	}
	entity, ok := payload["entity"].(map[string]any)
	if !ok {
		return false
	}
	return entity["kind"] == string(kind) && entity["name"] == name && entity["version"] == version
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
