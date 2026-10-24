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

type countingMarkdownRenderer struct {
	calls  int
	result planclient.RenderMarkdownResult
	err    error
}

func (c *countingMarkdownRenderer) RenderMarkdown(context.Context, string, bool) (planclient.RenderMarkdownResult, error) {
	c.calls++
	if c.err != nil {
		return planclient.RenderMarkdownResult{}, c.err
	}
	return c.result, nil
}

// Rendering is a remote call. Within one request a plan is rendered once no
// matter how many items link it, and each item still gets its own plan_ref
// derived path.
func TestPlanRenderMemoRendersEachPlanOncePerRequest(t *testing.T) {
	renderer := &countingMarkdownRenderer{result: planclient.RenderMarkdownResult{Markdown: "# Plan", QualityStatus: "pass"}}
	ctx := WithPlanRenderMemo(context.Background())

	shared := []backlogItem{
		{Kind: "execute", Name: "first", PlanRef: &planRef{Provider: "plan-manager", PlanID: "plan-1", Slug: "first-slug", Role: "execution_spec"}},
		{Kind: "execute", Name: "second", PlanRef: &planRef{Provider: "plan-manager", PlanID: "plan-1", Role: "execution_spec"}},
		{Kind: "execute", Name: "third", PlanRef: &planRef{Provider: "plan-manager", PlanID: "plan-2", Role: "execution_spec"}},
	}
	paths := make([]string, 0, len(shared))
	for _, item := range shared {
		content, err := resolveRenderedPlanContent(ctx, item, renderer)
		if err != nil {
			t.Fatalf("resolveRenderedPlanContent(%s): %v", item.Name, err)
		}
		paths = append(paths, content.Path)
	}

	if renderer.calls != 2 {
		t.Fatalf("render calls = %d for 3 items across 2 distinct plans; want 2", renderer.calls)
	}
	want := []string{"plan-manager:first-slug", "plan-manager:plan-1", "plan-manager:plan-2"}
	for i, path := range paths {
		if path != want[i] {
			t.Fatalf("path[%d] = %q, want %q; the memo must not share a per-item path", i, path, want[i])
		}
	}
}

// Without a request scope each resolution renders independently, so an
// isolated caller never reads another request's plan state.
func TestPlanRenderWithoutMemoRendersEveryTime(t *testing.T) {
	renderer := &countingMarkdownRenderer{result: planclient.RenderMarkdownResult{Markdown: "# Plan", QualityStatus: "pass"}}
	item := backlogItem{Kind: "execute", Name: "solo", PlanRef: &planRef{Provider: "plan-manager", PlanID: "plan-1", Role: "execution_spec"}}
	for range 2 {
		if _, err := resolveRenderedPlanContent(context.Background(), item, renderer); err != nil {
			t.Fatal(err)
		}
	}
	if renderer.calls != 2 {
		t.Fatalf("render calls = %d without a memo; want 2", renderer.calls)
	}
}

// A failing renderer is retried once per request, not once per item.
func TestPlanRenderMemoDoesNotRetryFailuresPerItem(t *testing.T) {
	renderer := &countingMarkdownRenderer{err: errors.New("plan-manager unavailable")}
	ctx := WithPlanRenderMemo(context.Background())
	for _, name := range []string{"first", "second"} {
		item := backlogItem{Kind: "execute", Name: name, PlanRef: &planRef{Provider: "plan-manager", PlanID: "plan-1", Role: "execution_spec"}}
		_, err := resolveRenderedPlanContent(ctx, item, renderer)
		if err == nil {
			t.Fatalf("resolveRenderedPlanContent(%s) succeeded against a failing renderer", name)
		}
		if !strings.Contains(err.Error(), "execute/"+name) {
			t.Fatalf("error %q must name the item it was resolved for", err)
		}
	}
	if renderer.calls != 1 {
		t.Fatalf("render calls = %d for a failing plan across 2 items; want 1", renderer.calls)
	}
}
