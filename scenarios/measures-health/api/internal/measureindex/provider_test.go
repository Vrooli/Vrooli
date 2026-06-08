package measureindex

import (
	"context"
	"testing"
	"time"

	measures "github.com/vrooli/measures-go"
)

// fakeExecutor records whether Execute was called and returns a canned result —
// the seam that proves the gate authorizes (or withholds) execution.
type fakeExecutor struct {
	called bool
	result measures.MeasureResult
}

func (f *fakeExecutor) Execute(_ context.Context, _ measures.MeasureDeclaration, _ map[string]string) (measures.MeasureResult, error) {
	f.called = true
	return f.result, nil
}

func fixedNow() time.Time { return time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC) }

func readMeasure() measures.MeasureDeclaration {
	d := backlogCompleted()
	d.RunEligible = true
	d.Params = map[string]measures.Param{
		"window": {Name: "window", Type: measures.ParamTypeTimeWindow, Default: "this_week"},
	}
	d.Result = measures.Result{Kind: measures.ResultScalar, ValueField: "count", Unit: "items", SummaryTemplate: "{count} backlog items completed ({window})"}
	return d
}

func writeMeasure() measures.MeasureDeclaration {
	return measures.MeasureDeclaration{
		Name:        "backlog.archive",
		Scenario:    "swarm-manager",
		Domain:      "backlog",
		Intent:      "Archive backlog items older than a window.",
		Questions:   []string{"archive backlog items older than last month", "archive stale backlog"},
		Effect:      measures.EffectWrite,
		RunEligible: true,
		Params: map[string]measures.Param{
			"window": {Name: "window", Type: measures.ParamTypeTimeWindow, Default: "last_month"},
		},
		Result: measures.Result{Kind: measures.ResultScalar, ValueField: "archived"},
	}
}

func needsMeasure() measures.MeasureDeclaration {
	return measures.MeasureDeclaration{
		Name:        "backlog.byinitiative",
		Scenario:    "swarm-manager",
		Domain:      "backlog",
		Intent:      "How many backlog items are open for a given initiative.",
		Questions:   []string{"how many open backlog items for an initiative", "open backlog by initiative"},
		Effect:      measures.EffectRead,
		RunEligible: true,
		Params: map[string]measures.Param{
			// A dynamic enum with no value source available -> unresolvable ->
			// (required, no default) -> needs[].
			"initiative": {Name: "initiative", Type: measures.ParamTypeEnum, ValuesSource: "initiative_names", Required: true},
		},
		Result: measures.Result{Kind: measures.ResultScalar, ValueField: "count"},
	}
}

func newTestProvider(t *testing.T, decls []measures.MeasureDeclaration, exec *fakeExecutor) *Provider {
	t.Helper()
	return NewProvider(decls, Config{
		Now:             fixedNow,
		Executor:        exec,
		Extractor:       measures.NoopExtractor{},
		OllamaAvailable: func(context.Context) bool { return false },
	})
}

func TestProvider_ReadMeasureAutoExecutes(t *testing.T) {
	exec := &fakeExecutor{result: measures.MeasureResult{
		Value:      "42",
		Provenance: measures.Provenance{ExecutedQuery: "SELECT count(*) FROM backlog WHERE done", ComputedAt: fixedNow()},
	}}
	p := newTestProvider(t, []measures.MeasureDeclaration{readMeasure()}, exec)

	hits, matcher, err := p.Query(context.Background(), "how many backlog items did we complete this week", 1)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if matcher != "lexical" {
		t.Fatalf("expected lexical matcher, got %q", matcher)
	}
	if len(hits) != 1 {
		t.Fatalf("expected one hit, got %d", len(hits))
	}
	h := hits[0]
	if !exec.called {
		t.Fatal("read measure at full confidence must auto-execute")
	}
	if h.Answer != "42 backlog items completed (this_week)" {
		t.Fatalf("unexpected answer: %q", h.Answer)
	}
	if h.ExecutedQuery == "" {
		t.Fatal("executed_query (provenance) must be populated on an auto-executed measure")
	}
	if h.Params["window"] != "this_week" {
		t.Fatalf("expected window=this_week, got %q", h.Params["window"])
	}
	if h.Effect != "read" {
		t.Fatalf("expected effect read, got %q", h.Effect)
	}
}

func TestProvider_WriteMeasureNeverAutoExecutes(t *testing.T) {
	exec := &fakeExecutor{result: measures.MeasureResult{Value: "should-not-happen"}}
	p := newTestProvider(t, []measures.MeasureDeclaration{writeMeasure()}, exec)

	hits, _, err := p.Query(context.Background(), "archive backlog items older than last month", 1)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected one hit, got %d", len(hits))
	}
	h := hits[0]
	if exec.called {
		t.Fatal("a write measure must NEVER auto-execute, even at full confidence")
	}
	if h.Answer != "" {
		t.Fatalf("write measure must not carry an answer, got %q", h.Answer)
	}
	if len(h.Needs) != 0 {
		t.Fatalf("write measure with a resolvable param should have no needs, got %v", h.Needs)
	}
	if h.Effect != "write" {
		t.Fatalf("expected effect write, got %q", h.Effect)
	}
	// Params resolved but execution withheld — the confirmation signal.
	if h.Params["window"] != "last_month" {
		t.Fatalf("expected resolved window=last_month, got %q", h.Params["window"])
	}
}

func TestProvider_MissingRequiredParamAbstains(t *testing.T) {
	exec := &fakeExecutor{}
	p := newTestProvider(t, []measures.MeasureDeclaration{needsMeasure()}, exec)

	hits, _, err := p.Query(context.Background(), "how many open backlog items for an initiative", 1)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected one hit, got %d", len(hits))
	}
	h := hits[0]
	if exec.called {
		t.Fatal("a measure with an unresolved required param must not execute")
	}
	if h.Answer != "" {
		t.Fatalf("expected no answer when params are missing, got %q", h.Answer)
	}
	if len(h.Needs) != 1 || h.Needs[0] != "initiative" {
		t.Fatalf("expected needs=[initiative], got %v", h.Needs)
	}
}

func TestProvider_NoMatchReturnsEmpty(t *testing.T) {
	p := newTestProvider(t, []measures.MeasureDeclaration{readMeasure()}, &fakeExecutor{})
	hits, matcher, err := p.Query(context.Background(), "qwerty zxcvb florbnax", 1)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(hits) != 0 {
		t.Fatalf("expected no hits for an unrelated question, got %d", len(hits))
	}
	if matcher != "lexical" {
		t.Fatalf("expected lexical matcher label, got %q", matcher)
	}
}

func TestProvider_EmptyIndex(t *testing.T) {
	p := NewProvider(nil, Config{OllamaAvailable: func(context.Context) bool { return false }})
	hits, matcher, err := p.Query(context.Background(), "how many backlog items this week", 1)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(hits) != 0 || matcher != "none" {
		t.Fatalf("empty index must answer honest-empty, got hits=%d matcher=%q", len(hits), matcher)
	}
	available, _, qdrant, indexed, label := p.Status(context.Background())
	if available || indexed != 0 || qdrant {
		t.Fatalf("empty index status wrong: available=%v indexed=%d qdrant=%v", available, indexed, qdrant)
	}
	if label != "lexical" {
		t.Fatalf("expected lexical matcher label, got %q", label)
	}
}
