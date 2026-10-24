package measures

import (
	"context"
	"testing"
)

// fakeExtractor is a programmable ParamExtractor for tests: it returns a canned
// value for a named param, simulating the Phase 3 constrained-LLM extractor.
type fakeExtractor struct {
	hits map[string]ExtractResult
}

func (f fakeExtractor) Extract(_ context.Context, _ string, p Param, _ []string) (ExtractResult, error) {
	if r, ok := f.hits[p.Name]; ok {
		return r, nil
	}
	return ExtractResult{}, nil
}

// staticValues is a ValuesProvider returning fixed dynamic-enum values.
type staticValues map[string][]string

func (s staticValues) Values(_ context.Context, src string) ([]string, error) {
	return s[src], nil
}

// declFixture builds a declaration with one time_window (default), one required
// dynamic enum, and one required bare param.
func declFixture() MeasureDeclaration {
	return MeasureDeclaration{
		Name:      "backlog.completed",
		Domain:    "backlog",
		Intent:    "How many backlog items completed in a window.",
		Questions: []string{"how many backlog items did we complete this week"},
		Params: map[string]Param{
			"window":     {Name: "window", Type: ParamTypeTimeWindow, Default: "this_week"},
			"initiative": {Name: "initiative", Type: ParamTypeEnum, ValuesSource: "initiative_names", Required: true},
			"label":      {Name: "label", Type: "string", Required: true},
		},
		Result:      Result{Kind: ResultScalar, ValueField: "count", Unit: "items"},
		Effect:      EffectRead,
		RunEligible: true,
	}
}

func TestResolveParamsDeterministicWindowAndNeeds(t *testing.T) {
	decl := declFixture()
	// NoopExtractor (default) → enum + bare cannot resolve, both go to needs.
	res, err := ResolveParams(context.Background(), "how many backlog items did we complete this week", decl, ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Params["window"] != "this_week" {
		t.Errorf("window = %q, want this_week (deterministic)", res.Params["window"])
	}
	if res.Confidence != ConfidenceDeterministic {
		t.Errorf("confidence = %v, want %v", res.Confidence, ConfidenceDeterministic)
	}
	if res.Complete() {
		t.Error("expected incomplete (required initiative+label unresolved)")
	}
	// Needs is sorted and contains exactly the two required-missing params.
	wantNeeds := []string{"initiative", "label"}
	if len(res.Needs) != 2 || res.Needs[0] != wantNeeds[0] || res.Needs[1] != wantNeeds[1] {
		t.Errorf("needs = %v, want %v", res.Needs, wantNeeds)
	}
}

func TestResolveParamsDefaultFallback(t *testing.T) {
	decl := declFixture()
	// No time phrase → window falls back to its default at reduced confidence.
	// Provide extractor + values so initiative/label resolve and don't dominate.
	opts := ResolveOptions{
		Extractor: fakeExtractor{hits: map[string]ExtractResult{
			"initiative": {Value: "desktop-release", Found: true, Confidence: 0.9},
			"label":      {Value: "urgent", Found: true, Confidence: 0.95},
		}},
		Values: staticValues{"initiative_names": {"desktop-release", "mobile"}},
	}
	res, err := ResolveParams(context.Background(), "how many backlog items total", decl, opts)
	if err != nil {
		t.Fatal(err)
	}
	if res.Params["window"] != "this_week" {
		t.Errorf("window = %q, want default this_week", res.Params["window"])
	}
	if !res.Complete() {
		t.Errorf("expected complete, got needs %v", res.Needs)
	}
	// Min confidence is the default fallback (0.8), below the extractor hits.
	if res.Confidence != ConfidenceDefault {
		t.Errorf("confidence = %v, want %v (default fallback dominates)", res.Confidence, ConfidenceDefault)
	}
}

func TestResolveParamsEnumConstraintEnforced(t *testing.T) {
	decl := declFixture()
	// Extractor returns a value NOT in the dynamic enum → rejected → needs.
	opts := ResolveOptions{
		Extractor: fakeExtractor{hits: map[string]ExtractResult{
			"initiative": {Value: "nonexistent", Found: true, Confidence: 0.99},
			"label":      {Value: "x", Found: true, Confidence: 0.99},
		}},
		Values: staticValues{"initiative_names": {"desktop-release"}},
	}
	res, err := ResolveParams(context.Background(), "items this week", decl, opts)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res.Params["initiative"]; ok {
		t.Error("out-of-enum value should be rejected, not set")
	}
	if !contains(res.Needs, "initiative") {
		t.Errorf("initiative should be in needs, got %v", res.Needs)
	}
}

func TestResolveParamsDynamicEnumNoProviderAbstains(t *testing.T) {
	decl := declFixture()
	// Values provider absent → dynamic enum has no value space → abstain (needs),
	// never guess.
	opts := ResolveOptions{Extractor: fakeExtractor{hits: map[string]ExtractResult{
		"initiative": {Value: "anything", Found: true, Confidence: 1.0},
		"label":      {Value: "x", Found: true, Confidence: 1.0},
	}}}
	res, err := ResolveParams(context.Background(), "items this week", decl, opts)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(res.Needs, "initiative") {
		t.Errorf("dynamic enum with no provider must abstain; needs = %v", res.Needs)
	}
}

func TestResolveParamsOptionalOmitted(t *testing.T) {
	decl := MeasureDeclaration{
		Name: "x.y", Domain: "x", Questions: []string{"q"},
		Params: map[string]Param{
			"opt": {Name: "opt", Type: "string", Required: false},
		},
		Result: Result{Kind: ResultScalar, ValueField: "v"}, Effect: EffectRead,
	}
	res, err := ResolveParams(context.Background(), "no params here", decl, ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Needs) != 0 {
		t.Errorf("optional unresolved param must not appear in needs; got %v", res.Needs)
	}
	if _, ok := res.Params["opt"]; ok {
		t.Error("optional unresolved param must be omitted")
	}
	if res.Confidence != ConfidenceDeterministic {
		t.Errorf("confidence with nothing resolved = %v, want %v", res.Confidence, ConfidenceDeterministic)
	}
}
