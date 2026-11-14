package backlog

import (
	"errors"
	"net/http"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
)

type RenderLinkedPlanResponse struct {
	Path            string   `json:"path"`
	Markdown        string   `json:"markdown"`
	QualityStatus   string   `json:"quality_status,omitempty"`
	QualityFindings []string `json:"quality_findings,omitempty"`
	PlanRef         *PlanRef `json:"plan_ref,omitempty"`
}

// RenderLinkedPlan returns the canonical plan-manager rendered projection for a
// backlog item.
func (h *Handler) RenderLinkedPlan(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "plan-render")
	if !ok {
		return
	}

	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apierr.MapError(w, "[backlog] plan-render", apierr.NotFound("backlog item not found"))
			return
		}
		apierr.MapError(w, "[backlog] plan-render", apierr.Internal("failed to load backlog item"))
		return
	}

	ref := normalizePlanRef(item.PlanRef)
	if ref == nil {
		apierr.MapError(w, "[backlog] plan-render", apierr.NotFound("plan_ref not found").WithCode("plan_ref_not_found"))
		return
	}
	if err := validatePlanRef(ref, PlanRefRoleExecutionSpec); err != nil {
		apierr.MapError(w, "[backlog] plan-render", apierr.BadRequest("%s", err.Error()))
		return
	}
	if h.planClient == nil {
		apierr.MapError(w, "[backlog] plan-render", apierr.Internal("plan-manager client is not configured"))
		return
	}
	rendered, err := h.planClient.RenderMarkdown(r.Context(), ref.PlanID, true)
	if err != nil {
		apierr.MapError(w, "[backlog] plan-render", apierr.Internal("failed to render linked plan: %s", err.Error()))
		return
	}
	if strings.TrimSpace(rendered.Markdown) == "" {
		apierr.MapError(w, "[backlog] plan-render", apierr.Internal("plan-manager returned empty markdown"))
		return
	}
	_ = httputil.JSON(w, RenderLinkedPlanResponse{
		Path:            "plan-manager:" + firstNonBlank(ref.Slug, ref.PlanID),
		Markdown:        rendered.Markdown,
		QualityStatus:   rendered.QualityStatus,
		QualityFindings: append([]string(nil), rendered.QualityFindings...),
		PlanRef:         ref,
	})
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
