package execution

import (
	"context"
	"fmt"
	"strings"

	"swarm-manager/internal/planclient"
)

const (
	planRefProviderPlanManager = "plan-manager"
	planRefRoleExecutionSpec   = "execution_spec"
)

type renderedPlanContent struct {
	Path            string
	Markdown        string
	ContentHash     string
	Status          string
	QualityStatus   string
	QualityFindings []string
}

func resolveRenderedPlanContent(ctx context.Context, item backlogItem, renderer planclient.MarkdownRenderer) (renderedPlanContent, error) {
	if renderer == nil {
		return renderedPlanContent{}, fmt.Errorf("plan_ref render unavailable: plan-manager client is not configured")
	}
	ref := item.PlanRef
	if ref == nil {
		return renderedPlanContent{}, fmt.Errorf("backlog item %s/%s has no plan_ref; finalize the item through plan-manager before queueing", item.Kind, item.Name)
	}
	if strings.TrimSpace(ref.Provider) != planRefProviderPlanManager {
		return renderedPlanContent{}, fmt.Errorf("backlog item %s/%s plan_ref.provider must be %q", item.Kind, item.Name, planRefProviderPlanManager)
	}
	if strings.TrimSpace(ref.Role) != planRefRoleExecutionSpec {
		return renderedPlanContent{}, fmt.Errorf("backlog item %s/%s plan_ref.role must be %q", item.Kind, item.Name, planRefRoleExecutionSpec)
	}
	planID := firstNonEmpty(ref.PlanID, ref.Slug)
	if planID == "" {
		return renderedPlanContent{}, fmt.Errorf("backlog item %s/%s plan_ref requires plan_id or slug", item.Kind, item.Name)
	}

	result, err := renderer.RenderMarkdown(ctx, planID, true)
	if err != nil {
		return renderedPlanContent{}, fmt.Errorf("render linked plan for backlog item %s/%s: %w", item.Kind, item.Name, err)
	}
	if strings.TrimSpace(result.Markdown) == "" {
		return renderedPlanContent{}, fmt.Errorf("render linked plan for backlog item %s/%s: plan-manager returned empty markdown", item.Kind, item.Name)
	}
	pathID := firstNonEmpty(ref.Slug, ref.PlanID, planID)
	return renderedPlanContent{
		Path:            "plan-manager:" + pathID,
		Markdown:        result.Markdown,
		ContentHash:     result.Plan.GetContentHash(),
		Status:          result.Plan.GetStatus().String(),
		QualityStatus:   result.QualityStatus,
		QualityFindings: append([]string(nil), result.QualityFindings...),
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
