package execution

import (
	"context"
	"errors"
	"strings"
	"testing"

	"swarm-manager/internal/planclient"
)

type fakeMarkdownRenderer struct {
	gotID      string
	gotCompact bool
	result     planclient.RenderMarkdownResult
	err        error
}

func (f *fakeMarkdownRenderer) RenderMarkdown(_ context.Context, id string, compact bool) (planclient.RenderMarkdownResult, error) {
	f.gotID = id
	f.gotCompact = compact
	if f.err != nil {
		return planclient.RenderMarkdownResult{}, f.err
	}
	return f.result, nil
}

func TestResolveRenderedPlanContent_UsesPlanRefAndCompactRender(t *testing.T) {
	renderer := &fakeMarkdownRenderer{result: planclient.RenderMarkdownResult{
		Markdown:        "# Compact plan\nDo it.",
		QualityStatus:   "clean",
		QualityFindings: []string{"ok"},
	}}
	item := backlogItem{
		Kind: "execute",
		Name: "do-it",
		PlanRef: &planRef{
			Provider: "plan-manager",
			PlanID:   "plan-123",
			Slug:     "do-it-plan",
			Role:     "execution_spec",
		},
	}

	content, err := resolveRenderedPlanContent(context.Background(), item, renderer)
	if err != nil {
		t.Fatalf("resolveRenderedPlanContent: %v", err)
	}
	if renderer.gotID != "plan-123" || !renderer.gotCompact {
		t.Fatalf("RenderMarkdown called with id=%q compact=%v", renderer.gotID, renderer.gotCompact)
	}
	if content.Path != "plan-manager:do-it-plan" {
		t.Fatalf("path = %q", content.Path)
	}
	if content.Markdown != "# Compact plan\nDo it." {
		t.Fatalf("markdown = %q", content.Markdown)
	}
	if content.QualityStatus != "clean" || len(content.QualityFindings) != 1 {
		t.Fatalf("quality not carried: %+v", content)
	}
}

func TestResolveRenderedPlanContent_MissingPlanRefBlocksClearly(t *testing.T) {
	_, err := resolveRenderedPlanContent(context.Background(), backlogItem{Kind: "execute", Name: "missing"}, &fakeMarkdownRenderer{})
	if err == nil || !strings.Contains(err.Error(), "has no plan_ref") {
		t.Fatalf("expected missing plan_ref error, got %v", err)
	}
}

func TestResolveRenderedPlanContent_RenderErrorIsTypedToItem(t *testing.T) {
	renderer := &fakeMarkdownRenderer{err: errors.New("connect unavailable")}
	item := backlogItem{
		Kind: "execute",
		Name: "broken",
		PlanRef: &planRef{
			Provider: "plan-manager",
			Slug:     "broken-plan",
			Role:     "execution_spec",
		},
	}

	_, err := resolveRenderedPlanContent(context.Background(), item, renderer)
	if err == nil || !strings.Contains(err.Error(), "render linked plan for backlog item execute/broken") {
		t.Fatalf("expected item-scoped render error, got %v", err)
	}
	if renderer.gotID != "broken-plan" || !renderer.gotCompact {
		t.Fatalf("RenderMarkdown called with id=%q compact=%v", renderer.gotID, renderer.gotCompact)
	}
}
