package backlog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/planrepair"
	"swarm-manager/internal/workshop"

	"github.com/gorilla/mux"
)

type planRepairRequest struct {
	MaxRepairAttempts int `json:"maxRepairAttempts"`
}

func (h *Handler) StartPlanRepair(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "plan-repair")
	if !ok {
		return
	}
	if h.planRepair == nil || h.planClient == nil {
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
	_, rounds, err := workshop.LoadLatestRound(h.store.ItemDir(kind, name))
	if err != nil {
		apierr.MapError(w, "[backlog] plan-repair", apierr.Internal("load workshop frontier failed"))
		return
	}
	findings := make([]any, len(rendered.QualityFindings))
	for i := range rendered.QualityFindings {
		findings[i] = map[string]any{"message": rendered.QualityFindings[i]}
	}
	frontier := repairFrontier(item.PlanRef.PlanID, rendered.Markdown)
	record, err := h.planRepair.Start(r.Context(), planrepair.StartRequest{EntityKind: string(kind), EntityName: name, EntityVersion: workshopSnapshotVersion(item, rounds), PlanReference: item.PlanRef.PlanID, PlanContent: rendered.Markdown, FrontierDigest: frontier, ValidationFindings: findings, CheckedAt: time.Now().UTC().Format(time.RFC3339Nano), MaxRepairAttempts: request.MaxRepairAttempts})
	if err != nil {
		apierr.MapError(w, "[backlog] plan-repair", apierr.Conflict("start plan repair: %s", err))
		return
	}
	_ = httputil.JSON(w, record)
}

func (h *Handler) ApplyPlanRepair(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "plan-repair-apply")
	if !ok {
		return
	}
	if h.planRepair == nil || h.planClient == nil {
		apierr.MapError(w, "[backlog] plan-repair", apierr.Internal("plan repair is not configured"))
		return
	}
	record, result, err := h.planRepair.Collect(r.Context(), mux.Vars(r)["repairID"])
	if err != nil {
		apierr.MapError(w, "[backlog] plan-repair", apierr.Conflict("collect plan repair: %s", err))
		return
	}
	if record.EntityKind != string(kind) || record.EntityName != name {
		apierr.MapError(w, "[backlog] plan-repair", apierr.Conflict("repair record belongs to a different backlog item"))
		return
	}
	if result.Outcome != "ready" {
		_ = httputil.JSON(w, result)
		return
	}
	plan, err := planrepair.Canonicalize(r.Context(), h.planClient, record, result)
	if err != nil {
		apierr.MapError(w, "[backlog] plan-repair", apierr.Conflict("canonicalize repaired plan: %s", err))
		return
	}
	completed, err := h.planRepair.CompleteApply(r.Context(), h, record, plan)
	if err != nil {
		apierr.MapError(w, "[backlog] plan-repair", apierr.Conflict("apply repaired plan: %s", err))
		return
	}
	_ = httputil.JSON(w, completed)
}

func repairFrontier(planID, markdown string) string {
	sum := sha256.Sum256([]byte(planID + "\x00" + markdown))
	return "sha256:" + hex.EncodeToString(sum[:])
}
