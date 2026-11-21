package usage

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"

	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
)

const annotationsFixture = `{
  "window":{"since":"2026-11-12T00:00:00Z","until":"2026-11-19T00:00:00Z"},
  "pack":{"name":"default","version":"2026-11-18.1"},
  "eligible_runs":818,"observed_runs":557,"coverage":0.681,
  "weighting":{"mode":"census+sample","sample_rate":0.05},
  "labels":[
    {"id":"agent.spinning","group":"agent","state":"proven",
     "gate":{"state":"proven","precision":0.82,"recall":0.74,"gold":31,"min_gold":25},
     "observed":142,"eligible":557,"rate":0.174,"baseline":0.054,"ratio":3.22,
     "search_facet":"label:agent.spinning","min_probability":0.6,
     "friction":{"enabled":true,"min_probability":0.6,"cost_unit":"runs","importance":"high"}},
    {"id":"operator.valence","group":"operator","state":"unproven",
     "gate":{"state":"insufficient_gold","precision":0.0,"recall":0.0,"gold":2,"min_gold":40},
     "observed":31,"eligible":557,
     "search_facet":"label:operator.valence","min_probability":0.6,
     "friction":{"enabled":true}}
  ]}`

func runAnnotations(t *testing.T, base string, args ...string) (string, error) {
	t.Helper()
	cmd := annotationsCommand(func(*cliapp.ScenarioApp) (paceHTTPDoer, string) {
		return http.DefaultClient, base
	})
	var stdout, stderr bytes.Buffer
	ctx, err := cliapp.NewTestRunContextFromArgs(cmd.Args, args, nil, &stdout, &stderr)
	if err != nil {
		return "", err
	}
	if err := cmd.RunCtx(ctx); err != nil {
		return stdout.String(), err
	}
	return stdout.String(), nil
}

func TestAnnotationsCommandShowsCoverageAndNeverRatesAnUnprovenLabel(t *testing.T) {
	var method, path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, annotationsFixture)
	}))
	defer server.Close()

	out, err := runAnnotations(t, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if method != http.MethodGet || path != annotationsCoveragePath+"?window=7d" {
		t.Fatalf("request = %s %s", method, path)
	}
	for _, want := range []string{
		"Inferred annotations: pack default",
		"Coverage: 68% (557 of 818 runs observed",
		"[proven] agent.spinning",
		"rate 17.4%",
		"3.22x",
		"[insufficient_gold] operator.valence",
		"gold 2/40, needs more labelled rows",
		"annotations run <run-id>",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("human output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "rate 0.0%") {
		t.Fatalf("an unproven label was rendered as a rate:\n%s", out)
	}
}

func TestAnnotationsCommandEmptyWindowIsUnknownNotZeroFriction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"pack":{"name":"default","version":"1"},"eligible_runs":0,"observed_runs":0,"coverage":0,"labels":[]}`)
	}))
	defer server.Close()

	out, err := runAnnotations(t, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "unknown, not zero friction") {
		t.Fatalf("empty coverage was not stated honestly:\n%s", out)
	}
}

func TestInferredAnnotationLinesQuoteARateOnlyForAProvenGate(t *testing.T) {
	rate := 0.174
	section := &measurepb.UsageInferredAnnotations{
		PackName: "default", PackVersion: "1", EligibleRuns: 50, ObservedRuns: 34, Coverage: 0.68,
		Labels: []*measurepb.UsageInferredAnnotation{
			{Id: "agent.spinning", Group: "agent", State: "observed", Observed: 34, Eligible: 50, Coverage: 0.68, GateState: "proven", Gold: 40, Rate: &rate, Friction: true, TargetPrecision: 0.8, TargetRecall: 0.7},
			{Id: "operator.valence", Group: "operator", State: "observed", Observed: 34, Eligible: 50, Coverage: 0.68, GateState: "insufficient_gold", Gold: 2, Friction: true, TargetPrecision: 0.85, TargetRecall: 0.7},
			{Id: "run.type", Group: "run", State: "observed", Observed: 34, Eligible: 50, GateState: "unmeasured", Friction: false},
		},
	}
	joined := strings.Join(inferredAnnotationLines(section), "\n")
	for _, want := range []string{"coverage 68% (34 of 50 runs observed)", "[proven] agent.spinning", "17.4% of observed runs", "[insufficient_gold] operator.valence", "no rate (target 0.85 / 0.70, gold 2)"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("section missing %q:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "run.type") {
		t.Fatalf("a non-friction label was rendered in the friction lens:\n%s", joined)
	}
}

func TestInferredAnnotationLinesReadAnEmptyWindowAsUnknown(t *testing.T) {
	lines := inferredAnnotationLines(&measurepb.UsageInferredAnnotations{Note: "no annotations recorded in this window"})
	joined := strings.Join(lines, "\n")
	for _, want := range []string{"coverage 0% (0 of 0 runs observed)", "no annotations recorded in this window", "unknown, not zero friction"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("empty section missing %q:\n%s", want, joined)
		}
	}
}
