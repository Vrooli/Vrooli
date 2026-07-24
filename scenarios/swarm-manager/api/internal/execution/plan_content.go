package execution

import (
	"context"
	"fmt"
	"strings"
	"sync"

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

// planRenderMemoKey scopes the request-scoped render memo carried in a
// context.
type planRenderMemoKey struct{}

// planRenderMemo holds one rendered plan per plan id for the life of a single
// request. Rendering is a remote call to plan-manager, and a list projection
// asks for the same handful of plans once per item that links them; the memo
// makes the request pay for each distinct plan exactly once.
type planRenderMemo struct {
	mu      sync.Mutex
	entries map[string]renderedPlanContent
	errs    map[string]error
}

// WithPlanRenderMemo scopes a plan-render memo to one request. Callers that
// resolve many items in a single request must wrap their context with it;
// without it every resolution renders independently, which is the correct
// behavior for an isolated call.
func WithPlanRenderMemo(ctx context.Context) context.Context {
	return context.WithValue(ctx, planRenderMemoKey{}, &planRenderMemo{
		entries: map[string]renderedPlanContent{},
		errs:    map[string]error{},
	})
}

func planRenderMemoFrom(ctx context.Context) *planRenderMemo {
	memo, _ := ctx.Value(planRenderMemoKey{}).(*planRenderMemo)
	return memo
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

	rendered, err := renderPlanOnce(ctx, planID, renderer)
	if err != nil {
		return renderedPlanContent{}, fmt.Errorf("render linked plan for backlog item %s/%s: %w", item.Kind, item.Name, err)
	}
	// Path is derived per item, not memoized: two items may address the same
	// plan by different halves of their plan_ref.
	pathID := firstNonEmpty(ref.Slug, ref.PlanID, planID)
	rendered.Path = "plan-manager:" + pathID
	return rendered, nil
}

// renderPlanOnce renders a plan through the memo when the caller scoped one to
// the request, and directly otherwise. Errors are memoized too: a failing
// renderer must not be retried once per item.
func renderPlanOnce(ctx context.Context, planID string, renderer planclient.MarkdownRenderer) (renderedPlanContent, error) {
	memo := planRenderMemoFrom(ctx)
	if memo != nil {
		memo.mu.Lock()
		defer memo.mu.Unlock()
		if err, ok := memo.errs[planID]; ok {
			return renderedPlanContent{}, err
		}
		if cached, ok := memo.entries[planID]; ok {
			return cached, nil
		}
	}
	rendered, err := renderPlan(ctx, planID, renderer)
	if memo != nil {
		if err != nil {
			memo.errs[planID] = err
		} else {
			memo.entries[planID] = rendered
		}
	}
	return rendered, err
}

func renderPlan(ctx context.Context, planID string, renderer planclient.MarkdownRenderer) (renderedPlanContent, error) {
	result, err := renderer.RenderMarkdown(ctx, planID, true)
	if err != nil {
		return renderedPlanContent{}, err
	}
	if strings.TrimSpace(result.Markdown) == "" {
		return renderedPlanContent{}, fmt.Errorf("plan-manager returned empty markdown")
	}
	return renderedPlanContent{
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
