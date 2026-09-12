package measures

import (
	"context"
	"testing"
	"time"
)

// fakeMatcher returns a fixed declaration for any question (the engine's match
// step is a seam; matching quality is exercised by the index in Phase 4).
type fakeMatcher struct {
	decl  MeasureDeclaration
	score float64
	empty bool
}

func (m fakeMatcher) Match(_ context.Context, _ string, _ int) ([]Match, error) {
	if m.empty {
		return nil, nil
	}
	return []Match{{Decl: m.decl, Score: m.score}}, nil
}

// fakeExecutor records the params it was called with and returns a fixed result.
type fakeExecutor struct {
	called bool
	params map[string]string
	result MeasureResult
}

func (x *fakeExecutor) Execute(_ context.Context, _ MeasureDeclaration, params map[string]string) (MeasureResult, error) {
	x.called = true
	x.params = params
	return x.result, nil
}

// backlogCompleted is the canonical read-only measure used across tests: a
// window (time_window, default this_week) + an optional initiative filter.
func backlogCompleted(effect Effect) MeasureDeclaration {
	return MeasureDeclaration{
		Name:     "backlog.completed",
		Scenario: "swarm-manager",
		Domain:   "backlog",
		Intent:   "How many backlog items were completed in a time window.",
		Questions: []string{
			"how many backlog items did we complete this week",
			"backlog items closed last month",
		},
		Params: map[string]Param{
			"window":     {Name: "window", Type: ParamTypeTimeWindow, Default: "this_week"},
			"initiative": {Name: "initiative", Type: ParamTypeEnum, ValuesSource: "initiative_names", Required: false},
		},
		Result: Result{
			Kind: ResultScalar, ValueField: "count", Unit: "items",
			SummaryTemplate: "{count} backlog items completed ({window})",
		},
		Effect:      effect,
		RunEligible: true,
	}
}

func anchorClock() func() time.Time {
	t := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
	return func() time.Time { return t }
}

func TestEngine_Answer_ReadOnly_Executes(t *testing.T) {
	exec := &fakeExecutor{result: MeasureResult{
		Value:      "42",
		Provenance: Provenance{ExecutedQuery: "SELECT count(*) ... [2026-06-08]"},
	}}
	eng := NewEngine(
		fakeMatcher{decl: backlogCompleted(EffectRead), score: 0.91},
		WithEngineClock(anchorClock()),
		WithExecutor(exec),
	)

	hit, err := eng.Answer(context.Background(), "how many backlog items did we complete this week")
	if err != nil {
		t.Fatalf("Answer: %v", err)
	}
	if hit == nil {
		t.Fatal("expected a hit, got nil")
	}
	if !exec.called {
		t.Fatal("executor was not called for a clean read-only measure")
	}
	if hit.Answer != "42 backlog items completed (this_week)" {
		t.Fatalf("answer = %q", hit.Answer)
	}
	if hit.ExecutedQuery == "" {
		t.Fatal("executed_query not set on an executed measure")
	}
	if got := hit.Params["window"]; got != "this_week" {
		t.Fatalf("window param = %q, want this_week", got)
	}
	if len(hit.Needs) != 0 {
		t.Fatalf("unexpected needs: %v", hit.Needs)
	}
	if hit.Score != 0.91 {
		t.Fatalf("score = %v, want 0.91", hit.Score)
	}
	if hit.Scenario != "swarm-manager" {
		t.Fatalf("scenario = %q", hit.Scenario)
	}
}

func TestEngine_Answer_DefaultWindow_StillExecutes(t *testing.T) {
	// The question names no window; the manifest default (this_week, conf 0.8)
	// is exactly at θ, so a read-only measure still auto-executes.
	exec := &fakeExecutor{result: MeasureResult{Value: "7"}}
	eng := NewEngine(
		fakeMatcher{decl: backlogCompleted(EffectRead), score: 0.8},
		WithEngineClock(anchorClock()),
		WithExecutor(exec),
	)
	hit, err := eng.Answer(context.Background(), "how many backlog items have we completed")
	if err != nil {
		t.Fatalf("Answer: %v", err)
	}
	if !exec.called {
		t.Fatal("expected execution at default-window confidence (0.8 == θ)")
	}
	if hit.Params["window"] != "this_week" {
		t.Fatalf("default window not applied: %v", hit.Params)
	}
}

func TestEngine_Answer_MissingRequiredParam_AbstainsToNeeds(t *testing.T) {
	decl := backlogCompleted(EffectRead)
	// Make initiative REQUIRED with no default and no resolvable value (Noop
	// extractor abstains) → it must land in needs[], and the measure must NOT
	// execute.
	p := decl.Params["initiative"]
	p.Required = true
	decl.Params["initiative"] = p

	exec := &fakeExecutor{result: MeasureResult{Value: "42"}}
	eng := NewEngine(
		fakeMatcher{decl: decl, score: 0.9},
		WithEngineClock(anchorClock()),
		WithValues(staticValues{"initiative_names": {"desktop-release", "mobile"}}),
		WithExecutor(exec),
	)
	hit, err := eng.Answer(context.Background(), "how many backlog items did we complete this week")
	if err != nil {
		t.Fatalf("Answer: %v", err)
	}
	if exec.called {
		t.Fatal("executor must NOT be called when a required param is unresolved")
	}
	if hit.Answer != "" {
		t.Fatalf("answer must be empty on abstention, got %q", hit.Answer)
	}
	if len(hit.Needs) != 1 || hit.Needs[0] != "initiative" {
		t.Fatalf("needs = %v, want [initiative]", hit.Needs)
	}
}

func TestEngine_Answer_WriteEffect_NeverAutoExecutes(t *testing.T) {
	for _, eff := range []Effect{EffectWrite, EffectDestructive} {
		exec := &fakeExecutor{result: MeasureResult{Value: "42"}}
		eng := NewEngine(
			fakeMatcher{decl: backlogCompleted(eff), score: 0.99},
			WithEngineClock(anchorClock()),
			WithExecutor(exec),
		)
		// Params fully resolvable + high confidence — only the effect blocks it.
		hit, err := eng.Answer(context.Background(), "how many backlog items did we complete this week")
		if err != nil {
			t.Fatalf("[%s] Answer: %v", eff, err)
		}
		if exec.called {
			t.Fatalf("[%s] a %s measure must NEVER auto-execute", eff, eff)
		}
		if hit.Answer != "" {
			t.Fatalf("[%s] answer must be empty, got %q", eff, hit.Answer)
		}
		if len(hit.Needs) != 0 {
			t.Fatalf("[%s] params were resolvable; needs should be empty, got %v", eff, hit.Needs)
		}
		if hit.GateReason == "" {
			t.Fatalf("[%s] expected a confirmation reason", eff)
		}
	}
}

func TestEngine_Answer_LowConfidence_Confirms(t *testing.T) {
	decl := backlogCompleted(EffectRead)
	// initiative is named in the question and extracted at low confidence (0.6),
	// dragging the min confidence below θ → confirm, not execute.
	exec := &fakeExecutor{result: MeasureResult{Value: "42"}}
	eng := NewEngine(
		fakeMatcher{decl: decl, score: 0.9},
		WithEngineClock(anchorClock()),
		WithValues(staticValues{"initiative_names": {"desktop-release"}}),
		WithExtractor(constExtractor{value: "desktop-release", found: true, confidence: 0.6}),
		WithExecutor(exec),
	)
	hit, err := eng.Answer(context.Background(), "completed work in the desktop-release initiative this week")
	if err != nil {
		t.Fatalf("Answer: %v", err)
	}
	if exec.called {
		t.Fatal("must not execute below θ")
	}
	if hit.Answer != "" {
		t.Fatalf("answer must be empty below θ, got %q", hit.Answer)
	}
	if hit.Confidence != 0.6 {
		t.Fatalf("confidence = %v, want 0.6 (min over params)", hit.Confidence)
	}
}

func TestEngine_Answer_NoMatch_ReturnsNil(t *testing.T) {
	eng := NewEngine(fakeMatcher{empty: true})
	hit, err := eng.Answer(context.Background(), "anything")
	if err != nil {
		t.Fatalf("Answer: %v", err)
	}
	if hit != nil {
		t.Fatalf("expected nil hit on no match, got %+v", hit)
	}
}

func TestEngine_Answer_NoExecutor_ResolvesButDoesNotExecute(t *testing.T) {
	// Resolve-first: with no executor wired, a clean read-only measure resolves
	// params + clears the gate but returns unexecuted (no answer).
	eng := NewEngine(
		fakeMatcher{decl: backlogCompleted(EffectRead), score: 0.9},
		WithEngineClock(anchorClock()),
	)
	hit, err := eng.Answer(context.Background(), "how many backlog items did we complete this week")
	if err != nil {
		t.Fatalf("Answer: %v", err)
	}
	if hit.Answer != "" {
		t.Fatalf("no executor → no answer, got %q", hit.Answer)
	}
	if hit.Params["window"] != "this_week" {
		t.Fatalf("params should still be resolved: %v", hit.Params)
	}
}

func TestRenderSummary(t *testing.T) {
	decl := backlogCompleted(EffectRead)
	got := RenderSummary(decl, MeasureResult{Value: "5"}, map[string]string{"window": "last_month"})
	if got != "5 backlog items completed (last_month)" {
		t.Fatalf("template render = %q", got)
	}

	// No template → "<value> <unit>".
	decl.Result.SummaryTemplate = ""
	got = RenderSummary(decl, MeasureResult{Value: "9"}, nil)
	if got != "9 items" {
		t.Fatalf("fallback render = %q", got)
	}

	// Table result with no scalar → row count.
	decl.Result = Result{Kind: ResultTable, ValueField: "rows"}
	got = RenderSummary(decl, MeasureResult{Fields: []map[string]string{{"a": "1"}, {"a": "2"}}}, nil)
	if got != "2 rows" {
		t.Fatalf("table render = %q", got)
	}
}

// constExtractor returns a fixed extraction outcome (a deterministic stand-in
// for the LLM extractor).
type constExtractor struct {
	value      string
	found      bool
	confidence float64
}

func (c constExtractor) Extract(_ context.Context, _ string, _ Param, _ []string) (ExtractResult, error) {
	return ExtractResult{Value: c.value, Found: c.found, Confidence: c.confidence}, nil
}
