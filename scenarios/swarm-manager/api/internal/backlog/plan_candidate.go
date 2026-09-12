package backlog

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/planclient"

	"github.com/gorilla/mux"
)

// ApplyPlanCandidate is Swarm's authorization and domain-provenance boundary
// for a Plan Manager candidate. Plan Manager enforces mechanical candidate
// guards; this handler verifies item ownership and clears stale acceptance
// after a successful canonical update.
func (h *Handler) ApplyPlanCandidate(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "plan-candidate-apply")
	if !ok {
		return
	}
	candidateID := strings.TrimSpace(mux.Vars(r)["candidateID"])
	if candidateID == "" {
		apierr.MapError(w, "[backlog] plan-candidate-apply", apierr.BadRequest("candidate id is required"))
		return
	}
	var request struct {
		ExpectedBaseContentHash  string `json:"expected_base_content_hash"`
		AcknowledgeQualityImpact bool   `json:"acknowledge_quality_impact"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil && err.Error() != "EOF" {
		apierr.MapError(w, "[backlog] plan-candidate-apply", apierr.BadRequest("invalid request body"))
		return
	}
	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		apierr.MapError(w, "[backlog] plan-candidate-apply", apierr.NotFound("backlog item not found"))
		return
	}
	if item.PlanRef == nil || strings.TrimSpace(item.PlanRef.PlanID) == "" {
		apierr.MapError(w, "[backlog] plan-candidate-apply", apierr.Conflict("backlog item has no canonical plan"))
		return
	}
	candidateClient, ok := h.planClient.(planclient.CandidateClient)
	if !ok {
		apierr.MapError(w, "[backlog] plan-candidate-apply", apierr.Unavailable("plan manager candidate revisions are unavailable"))
		return
	}
	plan, err := h.planClient.GetPlan(r.Context(), item.PlanRef.PlanID)
	if err != nil || plan == nil || strings.TrimSpace(plan.GetContentHash()) == "" {
		apierr.MapError(w, "[backlog] plan-candidate-apply", apierr.Conflict("load canonical plan frontier: %v", err))
		return
	}
	if request.ExpectedBaseContentHash == "" {
		request.ExpectedBaseContentHash = plan.GetContentHash()
	}
	if request.ExpectedBaseContentHash != plan.GetContentHash() {
		apierr.MapError(w, "[backlog] plan-candidate-apply", apierr.Conflict("candidate base hash does not match the item's current canonical plan"))
		return
	}
	if !request.AcknowledgeQualityImpact {
		apierr.MapError(w, "[backlog] plan-candidate-apply", apierr.BadRequest("acknowledge_quality_impact is required"))
		return
	}
	result, err := candidateClient.ApplyCandidateRevision(r.Context(), candidateID, request.ExpectedBaseContentHash, true)
	if err != nil {
		apierr.MapError(w, "[backlog] plan-candidate-apply", apierr.Conflict("apply candidate: %v", err))
		return
	}
	if result.GetPlan() == nil || result.GetPlan().GetId() != item.PlanRef.PlanID {
		apierr.MapError(w, "[backlog] plan-candidate-apply", apierr.Conflict("candidate does not apply to this backlog item's canonical plan"))
		return
	}
	item.PlanAcceptance = nil
	item.Updated = time.Now().UTC().Format(time.RFC3339)
	if err := h.store.SaveItem(item); err != nil {
		apierr.MapError(w, "[backlog] plan-candidate-apply", apierr.Internal("clear plan acceptance after candidate apply: %v", err))
		return
	}
	_ = httputil.JSON(w, map[string]any{"candidate": result.GetCandidate(), "plan": result.GetPlan(), "preview": result.GetPreview(), "acceptance_cleared": true})
}
