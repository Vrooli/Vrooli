package backlog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/planclient"
	"swarm-manager/internal/planrepair"
	"swarm-manager/internal/transitionrunner"

	"github.com/gorilla/mux"
	"google.golang.org/protobuf/types/known/structpb"
)

type planRepairRequest struct {
	MaxRepairAttempts int `json:"maxRepairAttempts"`
}

func (h *Handler) StartPlanRepair(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "plan-repair")
	if !ok {
		return
	}
	if h.transitionRunner == nil || h.planClient == nil {
		apierr.MapError(w, "[backlog] plan-repair", apierr.Internal("plan repair is not configured"))
		return
	}
	var request planRepairRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil && err.Error() != "EOF" {
		apierr.MapError(w, "[backlog] plan-repair", apierr.BadRequest("invalid request body"))
		return
	}
	if request.MaxRepairAttempts == 0 {
		request.MaxRepairAttempts = 1
	}
	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		apierr.MapError(w, "[backlog] plan-repair", apierr.NotFound("backlog item not found"))
		return
	}
	if item.PlanRef == nil {
		apierr.MapError(w, "[backlog] plan-repair", apierr.Conflict("backlog item has no plan_ref"))
		return
	}
	rendered, err := h.planClient.RenderMarkdown(r.Context(), item.PlanRef.PlanID, true)
	if err != nil {
		apierr.MapError(w, "[backlog] plan-repair", apierr.Conflict("cannot render canonical plan: %s", err))
		return
	}
	if len(rendered.QualityFindings) == 0 {
		apierr.MapError(w, "[backlog] plan-repair", apierr.Conflict("plan has no validation findings to repair"))
		return
	}
	findings := make([]any, len(rendered.QualityFindings))
	for i := range rendered.QualityFindings {
		findings[i] = map[string]any{"message": rendered.QualityFindings[i]}
	}
	frontier := repairFrontier(item.PlanRef.PlanID, rendered.Markdown)
	input, err := h.planRepairInput(item, rendered.Plan.GetContentHash(), rendered.Markdown, frontier, findings, time.Now().UTC().Format(time.RFC3339Nano), request.MaxRepairAttempts)
	if err != nil {
		apierr.MapError(w, "[backlog] plan-repair", apierr.Conflict("prepare plan repair: %s", err))
		return
	}
	started, err := h.transitionRunner.StartPrepared(r.Context(), "plan.repair", backlogSubjectRef(kind, name), input)
	if err != nil {
		apierr.MapError(w, "[backlog] plan-repair", apierr.Conflict("start plan repair: %s", err))
		return
	}
	_ = httputil.JSON(w, map[string]any{"execution_id": started.ExecutionID, "definition_digest": started.DefinitionDigest, "entity_version": started.EntityVersion})
}

func (h *Handler) ApplyPlanRepair(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "plan-repair-apply")
	if !ok {
		return
	}
	if h.transitionRunner == nil || h.planClient == nil {
		apierr.MapError(w, "[backlog] plan-repair", apierr.Internal("plan repair is not configured"))
		return
	}
	executionID := strings.TrimSpace(mux.Vars(r)["repairID"])
	correlation, err := h.transitionRunner.Apply(r.Context(), "plan.repair", executionID)
	if err != nil {
		apierr.MapError(w, "[backlog] plan-repair", apierr.Conflict("apply plan repair: %s", err))
		return
	}
	if correlation.SubjectRef != backlogSubjectRef(kind, name) {
		apierr.MapError(w, "[backlog] plan-repair", apierr.Conflict("repair record belongs to a different backlog item"))
		return
	}
	response, err := h.planRepairResponse(kind, name, executionID)
	if err != nil {
		apierr.MapError(w, "[backlog] plan-repair", apierr.Internal("read plan repair provenance"))
		return
	}
	_ = httputil.JSON(w, response)
}

func (h *Handler) buildPlanRepairInput(ctx context.Context, subjectRef string) (transitionrunner.Snapshot, error) {
	kind, name, err := parseBacklogSubjectRef(subjectRef)
	if err != nil {
		return transitionrunner.Snapshot{}, err
	}
	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		return transitionrunner.Snapshot{}, err
	}
	if item.PlanRef == nil {
		return transitionrunner.Snapshot{}, fmt.Errorf("backlog item has no plan_ref")
	}
	if h.planClient == nil {
		return transitionrunner.Snapshot{}, fmt.Errorf("plan manager client is not configured")
	}
	rendered, err := h.planClient.RenderMarkdown(ctx, item.PlanRef.PlanID, true)
	if err != nil {
		return transitionrunner.Snapshot{}, err
	}
	if len(rendered.QualityFindings) == 0 {
		return transitionrunner.Snapshot{}, fmt.Errorf("plan has no validation findings to repair")
	}
	findings := make([]any, len(rendered.QualityFindings))
	for i := range rendered.QualityFindings {
		findings[i] = map[string]any{"message": rendered.QualityFindings[i]}
	}
	prepared, err := h.planRepairInput(item, rendered.Plan.GetContentHash(), rendered.Markdown, repairFrontier(item.PlanRef.PlanID, rendered.Markdown), findings, time.Now().UTC().Format(time.RFC3339Nano), 1)
	if err != nil {
		return transitionrunner.Snapshot{}, err
	}
	return transitionrunner.SnapshotFromSubject(prepared.Input, item, map[string]any{"plan": item.PlanRef.PlanID, "frontier": prepared.FrontierDigest})
}

func (h *Handler) planRepairInput(item BacklogItem, baseHash, content, frontier string, findings []any, checkedAt string, maxAttempts int) (transitionrunner.PreparedInput, error) {
	version := immutableBacklogSnapshotVersion(item)
	input, err := structpb.NewValue(map[string]any{
		"entity":      map[string]any{"kind": string(item.Kind), "name": item.Name, "version": version},
		"plan":        map[string]any{"reference": item.PlanRef.PlanID, "content": content, "frontierDigest": frontier},
		"validation":  map[string]any{"findings": findings, "checkedAt": checkedAt},
		"constraints": map[string]any{"maxRepairAttempts": maxAttempts},
	})
	if err != nil {
		return transitionrunner.PreparedInput{}, err
	}
	return transitionrunner.PreparedInput{Input: input, EntityVersion: version, FrontierDigest: frontier}, nil
}

func planRepairProvenancePath(itemDir, executionID string) string {
	return filepath.Join(itemDir, "plan-repair-provenance-"+executionID+".json")
}

func (h *Handler) applyPlanRepairOutcome(ctx context.Context, subjectRef string, outcome transitionrunner.Outcome) error {
	if h.planClient == nil {
		return fmt.Errorf("plan repair is not configured")
	}
	kind, name, err := parseBacklogSubjectRef(subjectRef)
	if err != nil {
		return err
	}
	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		return err
	}
	path := planRepairProvenancePath(h.store.ItemDir(kind, name), outcome.ExecutionID)
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	result, err := decodePlanRepairOutcome(outcome)
	if err != nil {
		return err
	}
	response := map[string]any{"execution_id": outcome.ExecutionID, "outcome": result.Outcome, "reason": result.Reason, "applied_at": time.Now().UTC().Format(time.RFC3339Nano)}
	if result.Outcome == "ready" {
		if item.PlanRef == nil {
			return fmt.Errorf("backlog item lost plan_ref after repair start")
		}
		rendered, err := h.planClient.RenderMarkdown(ctx, item.PlanRef.PlanID, true)
		if err != nil {
			return err
		}
		candidateClient, ok := h.planClient.(planclient.CandidateClient)
		if !ok {
			return fmt.Errorf("plan manager candidate revisions are unavailable")
		}
		preview, err := planrepair.Canonicalize(ctx, candidateClient, item.PlanRef.PlanID, rendered.Plan.GetContentHash(), outcome.ExecutionID, result)
		if err != nil {
			return err
		}
		response["candidate_preview"] = preview
	}
	return writeJSONRedacted(path, response)
}

func decodePlanRepairOutcome(outcome transitionrunner.Outcome) (planrepair.TerminalResult, error) {
	var result planrepair.TerminalResult
	if err := json.Unmarshal(outcome.Result, &result); err != nil {
		return result, err
	}
	switch result.Outcome {
	case "ready":
		if len(result.CandidatePlan) == 0 {
			return result, fmt.Errorf("ready repair result is missing candidatePlan")
		}
	case "needs_attention", "abstained", "failed", "budget_exhausted":
		if strings.TrimSpace(result.Reason) == "" {
			return result, fmt.Errorf("%s repair result is missing reason", result.Outcome)
		}
	default:
		return result, fmt.Errorf("unsupported plan repair outcome %q", result.Outcome)
	}
	return result, nil
}

func (h *Handler) planRepairResponse(kind BacklogKind, name, executionID string) (map[string]any, error) {
	data, err := os.ReadFile(planRepairProvenancePath(h.store.ItemDir(kind, name), executionID))
	if err != nil {
		return nil, err
	}
	var response map[string]any
	return response, json.Unmarshal(data, &response)
}

func repairFrontier(planID, markdown string) string {
	sum := sha256.Sum256([]byte(planID + "\x00" + markdown))
	return "sha256:" + hex.EncodeToString(sum[:])
}
