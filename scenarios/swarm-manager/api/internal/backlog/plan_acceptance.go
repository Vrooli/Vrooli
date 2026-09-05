package backlog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/identity"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

type acceptPlanRequest struct {
	Actor           string `json:"actor"`
	PlanContentHash string `json:"plan_content_hash,omitempty"`
}

// AcceptPlan records the caller's acceptance of the live Plan Manager frontier.
// An omitted actor is derived from verified request provenance; a caller may
// pin plan_content_hash to reject a stale UI action.
func (h *Handler) AcceptPlan(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "accept-plan")
	if !ok {
		return
	}
	var request acceptPlanRequest
	if err := httputil.DecodeJSONStrict(r, &request); err != nil {
		apierr.MapError(w, "[backlog] accept-plan", apierr.BadRequest("invalid request body"))
		return
	}
	if strings.TrimSpace(request.Actor) == "" {
		request.Actor = identity.FromContext(r.Context()).FormatStartedBy()
	}
	if h.planClient == nil {
		apierr.MapError(w, "[backlog] accept-plan", apierr.Internal("plan-manager client is not configured"))
		return
	}
	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apierr.MapError(w, "[backlog] accept-plan", apierr.NotFound("backlog item not found"))
		} else {
			apierr.MapError(w, "[backlog] accept-plan", apierr.Internal("failed to load backlog item"))
		}
		return
	}
	if err := validatePlanRef(item.PlanRef, PlanRefRoleExecutionSpec); err != nil {
		apierr.MapError(w, "[backlog] accept-plan", apierr.Conflict("a canonical execution plan is required: %s", err))
		return
	}
	planID := firstNonBlank(item.PlanRef.PlanID, item.PlanRef.Slug)
	plan, err := h.planClient.GetPlan(r.Context(), planID)
	if err != nil {
		apierr.MapError(w, "[backlog] accept-plan", apierr.Conflict("resolve canonical plan: %s", err))
		return
	}
	contentHash := strings.TrimSpace(plan.GetContentHash())
	// Plan Manager's DRAFT status means no phase has started; it is not an
	// authoring/finalization signal. Rejecting it here makes first execution
	// circular because only a started execution moves the computed status to
	// ACTIVE. Render quality and the pinned content hash are the acceptance
	// authorities; only an archived plan is categorically non-executable.
	if plan.GetStatus() == sharedv1.PlanStatus_PLAN_STATUS_ARCHIVED {
		apierr.MapError(w, "[backlog] accept-plan", apierr.Conflict("canonical plan is not executable in status %q", plan.GetStatus().String()))
		return
	}
	if contentHash == "" {
		apierr.MapError(w, "[backlog] accept-plan", apierr.Conflict("canonical plan has no content hash"))
		return
	}
	if requested := strings.TrimSpace(request.PlanContentHash); requested != "" && requested != contentHash {
		apierr.MapError(w, "[backlog] accept-plan", apierr.Conflict("canonical plan changed; refresh before accepting"))
		return
	}
	rendered, err := h.planClient.RenderMarkdown(r.Context(), planID, true)
	if err != nil {
		apierr.MapError(w, "[backlog] accept-plan", apierr.Conflict("validate canonical plan: %s", err))
		return
	}
	if strings.TrimSpace(rendered.QualityStatus) != "pass" {
		apierr.MapError(w, "[backlog] accept-plan", apierr.Conflict("canonical plan is not valid: quality status is %q", rendered.QualityStatus))
		return
	}
	item.Updated = time.Now().UTC().Format(time.RFC3339Nano)
	item.PlanAcceptance = &PlanAcceptance{
		Actor:           strings.TrimSpace(request.Actor),
		AcceptedAt:      item.Updated,
		PlanContentHash: contentHash,
		SubjectVersion:  PlanAcceptanceSubjectVersion(item),
	}
	if err := h.store.SaveItem(item); err != nil {
		apierr.MapError(w, "[backlog] accept-plan", apierr.Internal("failed to save plan acceptance"))
		return
	}
	_ = httputil.JSON(w, map[string]any{"plan_acceptance": item.PlanAcceptance})
}

// UnacceptPlan clears an explicit acceptance. It is forbidden while execution
// is queued or running: the operator must cancel the execution first so the
// readiness contract cannot change beneath live work.
func (h *Handler) UnacceptPlan(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "unaccept-plan")
	if !ok {
		return
	}
	if h.executionActivity != nil && h.executionActivity.HasActiveForBacklog(r.Context(), string(kind), name) {
		apierr.MapError(w, "[backlog] unaccept-plan", apierr.Conflict("cannot un-accept while an execution is queued or running; cancel it first"))
		return
	}
	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apierr.MapError(w, "[backlog] unaccept-plan", apierr.NotFound("backlog item not found"))
		} else {
			apierr.MapError(w, "[backlog] unaccept-plan", apierr.Internal("failed to load backlog item"))
		}
		return
	}
	item.PlanAcceptance = nil
	item.Updated = time.Now().UTC().Format(time.RFC3339Nano)
	if err := h.store.SaveItem(item); err != nil {
		apierr.MapError(w, "[backlog] unaccept-plan", apierr.Internal("failed to clear plan acceptance"))
		return
	}
	_ = httputil.JSON(w, map[string]any{"plan_acceptance": nil})
}

// PlanAcceptanceSubjectVersion hashes the plan-relevant work contract while
// deliberately excluding lifecycle timestamps/status. Queueing must not make
// an already accepted plan stale, but changing scope or the plan reference
// must.
func PlanAcceptanceSubjectVersion(item BacklogItem) string {
	payload := struct {
		Kind            BacklogKind `json:"kind"`
		Name            string      `json:"name"`
		Title           string      `json:"title"`
		Description     string      `json:"description"`
		AcceptanceAllow []string    `json:"acceptance_allow,omitempty"`
		AcceptanceDeny  []string    `json:"acceptance_deny,omitempty"`
		Creates         []string    `json:"creates,omitempty"`
		PlanRef         *PlanRef    `json:"plan_ref,omitempty"`
	}{
		Kind: item.Kind, Name: item.Name, Title: item.Title, Description: item.Description,
		AcceptanceAllow: item.AcceptanceAllow, AcceptanceDeny: item.AcceptanceDeny,
		Creates: item.Creates, PlanRef: item.PlanRef,
	}
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// PlanAcceptanceMatches reports whether the record authorizes this exact
// canonical plan frontier and unchanged work contract.
func PlanAcceptanceMatches(item BacklogItem, planContentHash string) bool {
	acceptance := item.PlanAcceptance
	return acceptance != nil &&
		strings.TrimSpace(acceptance.Actor) != "" &&
		strings.TrimSpace(acceptance.PlanContentHash) != "" &&
		strings.TrimSpace(acceptance.PlanContentHash) == strings.TrimSpace(planContentHash) &&
		strings.TrimSpace(acceptance.SubjectVersion) == PlanAcceptanceSubjectVersion(item)
}
