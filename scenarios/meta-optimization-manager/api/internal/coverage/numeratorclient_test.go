package coverage

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/spacedoc"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/graph"
	graphconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/graph/graph_v1connect"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	evalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval/eval_v1connect"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry/registry_v1connect"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing/routing_v1connect"
	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	runsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
)

// ---------------------------------------------------------------------------
// Pure per-cell join rules (transport-independent)
// ---------------------------------------------------------------------------

func TestRecomputeAnswer(t *testing.T) {
	out, evidence := recomputeAnswer(answerDef().Cells, []answerProviderEvidence{
		{ProviderID: "ui-health.surfaces", Active: true, Reachable: true, FreshEval: true, ReachabilityEvidence: "reachable", EvalEvidence: "run-1"},
		{ProviderID: "code-facts", Active: true, Reachable: true, FreshEval: true, ReachabilityEvidence: "reachable", EvalEvidence: "run-2"},
	})
	if out["1"] != spacedoc.StatusNow {
		t.Errorf("cell1 (live provider) = %v, want now", out["1"])
	}
	if st, ok := out["2"]; ok && st == spacedoc.StatusNow {
		t.Errorf("cell2 (provider not live) must not be promoted to now: %v", st)
	}
	if len(evidence["1"]) != 3 {
		t.Fatalf("answer evidence = %+v, want all three signals", evidence["1"])
	}
}

func TestRecomputeAnswerDriftDowngrade(t *testing.T) {
	// Authored-NOW cell 1, but its provider is not reachable -> honest downgrade.
	out, _ := recomputeAnswer(answerDef().Cells, []answerProviderEvidence{{
		ProviderID: "ui-health.surfaces", Active: true, Reachable: false, FreshEval: true,
		ReachabilityEvidence: "ui-health.surfaces reachability=\"unreachable\"", EvalEvidence: "run-1",
	}})
	if out["1"] != spacedoc.StatusInReach {
		t.Errorf("authored-NOW with unreachable provider should downgrade to in_reach, got %v", out["1"])
	}
}

func TestRecomputeAnswerThreeSignalTable(t *testing.T) {
	cells := []spacedoc.Cell{{ID: "now", Owner: "provider.one", Status: spacedoc.StatusMissing}, {ID: "stale", Owner: "provider.two", Status: spacedoc.StatusNow}, {ID: "down", Owner: "provider.three", Status: spacedoc.StatusNow}, {ID: "gap", Owner: "provider.four", Status: spacedoc.StatusMissing}, {ID: "authored", Owner: "missing.provider", Status: spacedoc.StatusInReach}, {ID: "authored-now", Owner: "missing.provider-now", Status: spacedoc.StatusNow}}
	providers := []answerProviderEvidence{
		{ProviderID: "provider.one", Active: true, Reachable: true, FreshEval: true, ReachabilityEvidence: "reachable", EvalEvidence: "fresh run"},
		{ProviderID: "provider.two", Active: true, Reachable: true, FreshEval: false, ReachabilityEvidence: "reachable", EvalEvidence: "no non-degraded eval run in the last 30d"},
		{ProviderID: "provider.three", Active: true, Reachable: false, FreshEval: true, ReachabilityEvidence: "unreachable", EvalEvidence: "fresh run"},
	}
	out, evidence := recomputeAnswer(cells, providers)
	if out["now"] != spacedoc.StatusNow {
		t.Errorf("all signals should produce NOW, got %v", out["now"])
	}
	if out["stale"] != spacedoc.StatusInReach {
		t.Errorf("stale eval should produce IN_REACH, got %v", out["stale"])
	}
	if out["down"] != spacedoc.StatusInReach {
		t.Errorf("unreachable provider should produce IN_REACH, got %v", out["down"])
	}
	if _, ok := out["gap"]; ok {
		t.Error("capability gap must not promote a cell")
	}
	if _, ok := out["authored"]; ok {
		t.Error("unresolved provider must preserve authored status")
	}
	if out["authored-now"] != spacedoc.StatusInReach {
		t.Errorf("unresolved authored NOW must not remain NOW, got %v", out["authored-now"])
	}
	for _, id := range []string{"now", "stale", "down", "gap", "authored", "authored-now"} {
		if len(evidence[id]) != 3 {
			t.Errorf("cell %s evidence = %+v, want all three signals", id, evidence[id])
		}
	}
}

func TestRecomputeValidate(t *testing.T) {
	index := map[string]validateProviderStatus{
		"green-health":   {},
		"red-health":     {failing: true},
		"autofix-health": {autofixPending: true},
	}
	cells := []spacedoc.Cell{
		{ID: "V1", Owner: "`green-health`", Status: spacedoc.StatusInReach},
		{ID: "V2", Owner: "`red-health`", Status: spacedoc.StatusNow},
		{ID: "V3", Owner: "`autofix-health`", Status: spacedoc.StatusNow},
		{ID: "V4", Owner: "`unknown-health`", Status: spacedoc.StatusMissing},
	}
	out := recomputeValidate(cells, index)
	if out["V1"] != spacedoc.StatusNow {
		t.Errorf("green phase = %v, want now", out["V1"])
	}
	if out["V2"] != spacedoc.StatusNow {
		t.Errorf("failing phase = %v, want now (condition is separate)", out["V2"])
	}
	if out["V3"] != spacedoc.StatusNow {
		t.Errorf("autofix-pending phase = %v, want now (condition is separate)", out["V3"])
	}
	_, conditions := recomputeValidateWithConditions(cells, index)
	if conditions["V2"] != ConditionDegraded || conditions["V1"] != ConditionOK {
		t.Fatalf("conditions = %v, want V2 degraded and V1 ok", conditions)
	}
	if _, ok := out["V4"]; ok {
		t.Errorf("unknown phase should keep authored status, got overlay %v", out["V4"])
	}
}

func TestRecomputeValidateOnlySustainedDegradationDowngradesCoverage(t *testing.T) {
	cells := []spacedoc.Cell{{ID: "V1", Owner: "`red-health`", Status: spacedoc.StatusNow}}
	index := map[string]validateProviderStatus{"red-health": {failing: true, condition: ConditionDegraded, sustained: true}}
	statuses, conditions := recomputeValidateWithConditions(cells, index)
	if statuses["V1"] != spacedoc.StatusInReach || conditions["V1"] != ConditionDegraded {
		t.Fatalf("statuses=%v conditions=%v, want sustained downgrade with degraded condition", statuses, conditions)
	}
}

func TestRecomputeGuide(t *testing.T) {
	scores := map[string]float64{
		"explore":                      0.9,
		"polish":                       0.4,
		"architecture-scope":           0.8,
		"screaming-architecture-audit": 0.7,
	}
	cells := []spacedoc.Cell{
		{ID: "G1", Owner: "`explore` + the Answer projection", Status: spacedoc.StatusInReach},
		{ID: "G2", Owner: "`screaming-architecture-audit`, `architecture-scope`", Status: spacedoc.StatusInReach},
		{ID: "G3", Owner: "`polish`", Status: spacedoc.StatusNow},
		{ID: "G4", Owner: "`explore`, `missing-skill`", Status: spacedoc.StatusNow},
		{ID: "G5", Owner: "`missing-skill`", Status: spacedoc.StatusMissing},
	}
	out := recomputeGuide(cells, scores)
	if out["G1"] != spacedoc.StatusNow {
		t.Errorf("single healthy skill = %v, want now", out["G1"])
	}
	if out["G2"] != spacedoc.StatusNow {
		t.Errorf("multi-skill healthy row = %v, want now", out["G2"])
	}
	if out["G3"] != spacedoc.StatusInReach {
		t.Errorf("unhealthy skill = %v, want in_reach", out["G3"])
	}
	if out["G4"] != spacedoc.StatusInReach {
		t.Errorf("partially resolved row = %v, want in_reach", out["G4"])
	}
	if _, ok := out["G5"]; ok {
		t.Errorf("unresolved row should keep authored status, got overlay %v", out["G5"])
	}
}

// ---------------------------------------------------------------------------
// Typed proto message -> normalized index mapping
// ---------------------------------------------------------------------------

func TestProvidersToLive(t *testing.T) {
	resp := &registryv1.ListProvidersResponse{Providers: []*registryv1.ProviderDescriptor{
		{ProviderId: "ui-health.surfaces", Type: "structured", ProviderGroup: "ui-health"},
		{ProviderId: "code-facts"},
	}}
	live := providersToLive(resp)
	for _, want := range []string{"ui-health.surfaces", "ui-health", "code-facts"} {
		if !live[want] {
			t.Errorf("expected live key %q in %v", want, live)
		}
	}
	if live["structured"] {
		t.Error("descriptor type must not be treated as a live provider")
	}
	if providersToLive(nil) == nil {
		t.Error("nil response should yield an empty (non-nil) set")
	}
}

func TestAnswerEvidenceKeepsCapabilityGapRegisteredButNotActive(t *testing.T) {
	resp := &registryv1.ListProvidersResponse{Providers: []*registryv1.ProviderDescriptor{
		{ProviderId: "code-reference.code", State: registryv1.ProviderState_PROVIDER_STATE_CAPABILITY_GAP},
	}}
	evidence := answerEvidence(resp, nil, nil)
	if len(evidence) != 1 {
		t.Fatalf("evidence = %+v, want one registered provider", evidence)
	}
	if evidence[0].ProviderID != "code-reference.code" {
		t.Fatalf("provider id = %q", evidence[0].ProviderID)
	}
	if evidence[0].Active {
		t.Fatal("capability-gap provider must remain non-active evidence")
	}
}

func TestProviderReachableUsesTypedBoolAndReason(t *testing.T) {
	cases := []struct {
		name   string
		health *routingv1.ProviderHealth
		want   bool
	}{
		{name: "endpoint resolved", health: &routingv1.ProviderHealth{Reachable: true, Reachability: "endpoint resolved"}, want: true},
		{name: "explicit reachable", health: &routingv1.ProviderHealth{Reachable: true, Reachability: "reachable"}, want: true},
		{name: "probe due but reachable", health: &routingv1.ProviderHealth{Reachable: true, Degraded: true, Reachability: "circuit_cooldown_elapsed_probe_due"}, want: true},
		{name: "unreachable reason", health: &routingv1.ProviderHealth{Reachable: true, Reachability: "provider unreachable"}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := providerReachable(tc.health); got != tc.want {
				t.Errorf("providerReachable(%+v) = %v, want %v", tc.health, got, tc.want)
			}
		})
	}
}

func TestMatchesLiveDoesNotPromoteSiblingLeaves(t *testing.T) {
	live := map[string]bool{
		"architecture-cartographer.domain-map": true,
		"architecture-cartographer.zones":      true,
	}
	cases := []struct {
		name  string
		token string
		want  bool
	}{
		{name: "exact leaf", token: "architecture-cartographer.domain-map", want: true},
		{name: "sibling leaf", token: "architecture-cartographer.coupling", want: false},
		{name: "scenario head is not a leaf", token: "architecture-cartographer", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := matchesLive(tc.token, live); got != tc.want {
				t.Errorf("matchesLive(%q) = %v, want %v", tc.token, got, tc.want)
			}
		})
	}
}

func TestSelfHealthToValidateIndex(t *testing.T) {
	h := &runsv1.SelfHealth{
		Catalog: &runsv1.CatalogSummary{Phases: []*runsv1.CatalogPhase{
			{Provider: "green-health"}, {Provider: "red-health"}, {Provider: "autofix-health"},
		}},
		Ledger: &runsv1.ReliabilityLedger{Phases: []*runsv1.PhaseReliability{
			{Provider: "green-health", FailureRate: 0},
			{Provider: "red-health", FailureRate: 0.25},
			{Provider: "autofix-health", FailureRate: 0},
		}},
		Conformance: []*runsv1.ProviderConformance{
			{Provider: "autofix-health", Autofix: &runsv1.AutofixCoverage{Pending: 2}},
		},
	}
	idx := selfHealthToValidateIndex(h)
	if st, ok := idx["green-health"]; !ok || st.failing || st.autofixPending {
		t.Errorf("green-health = %+v (ok=%v), want healthy", st, ok)
	}
	if !idx["red-health"].failing {
		t.Error("red-health should be failing")
	}
	if !idx["autofix-health"].autofixPending {
		t.Error("autofix-health should have pending autofix")
	}
	if selfHealthToValidateIndex(nil) == nil {
		t.Error("nil self-health should yield an empty (non-nil) index")
	}
}

func TestScoresToGuideIndex(t *testing.T) {
	scores := []*graphv1.HealthScore{
		{NodeId: "Explore", Score: 0.9},
		{NodeId: "polish", Score: 0.4},
	}
	idx := scoresToGuideIndex(scores)
	if idx["explore"] != 0.9 { // lower-cased
		t.Errorf("explore = %v, want 0.9", idx["explore"])
	}
	if idx["polish"] != 0.4 {
		t.Errorf("polish = %v", idx["polish"])
	}
}

// TestCapturedOwnerFixturesStillMap guards the typed mapping against owner wire
// drift, using the captured real-data fixtures. The test-genie fixture is the
// Connect GetSelfHealth response shape; the prompt-manager fixture predates the
// Connect contract (the legacy bare `[]HealthScore` REST array) and is wrapped
// into the GetHealthScores response shape for the mapping.
func TestCapturedOwnerFixturesStillMap(t *testing.T) {
	var selfResp runsv1.GetSelfHealthResponse
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(readCoverageTestdata(t, "test_genie_health.json"), &selfResp); err != nil {
		t.Fatalf("decode test-genie fixture: %v", err)
	}
	validateIndex := selfHealthToValidateIndex(selfResp.GetSelfHealth())
	if _, ok := validateIndex["structure-health"]; !ok {
		t.Fatal("captured test-genie health fixture no longer exposes structure-health")
	}
	if !validateIndex["storage-manager"].autofixPending {
		t.Fatal("captured test-genie health fixture should expose storage-manager pending autofix work")
	}

	wrapped := append(append([]byte(`{"scores":`), readCoverageTestdata(t, "pm_graph_health.json")...), '}')
	var graphResp graphv1.GetHealthScoresResponse
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(wrapped, &graphResp); err != nil {
		t.Fatalf("decode prompt-manager fixture: %v", err)
	}
	guideScores := scoresToGuideIndex(graphResp.GetScores())
	for _, skill := range []string{"explore", "idea-workshop", "performance", "polish"} {
		if _, ok := guideScores[skill]; !ok {
			t.Fatalf("captured prompt-manager graph health fixture no longer exposes %q", skill)
		}
	}
}

// ---------------------------------------------------------------------------
// End-to-end typed Join (discovery-resolved base URL -> Connect client -> map)
// ---------------------------------------------------------------------------

// fakeResolver returns a fixed base URL (or error) for any owner, standing in
// for discovery.Resolver so reads target an httptest server (or a simulated
// not-running owner) without shelling out to the vrooli CLI.
type fakeResolver struct {
	url string
	err error
}

func (f fakeResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return f.url, f.err
}

type fakeRegistry struct {
	registryconnect.UnimplementedRegistryServiceHandler
	resp *registryv1.ListProvidersResponse
	err  error
}

func (f fakeRegistry) ListProviders(context.Context, *connect.Request[registryv1.ListProvidersRequest]) (*connect.Response[registryv1.ListProvidersResponse], error) {
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(f.resp), nil
}

type fakeRuns struct {
	runsconnect.UnimplementedRunsServiceHandler
	resp  *runsv1.GetSelfHealthResponse
	sleep time.Duration
}

func (f fakeRuns) GetSelfHealth(ctx context.Context, _ *connect.Request[runsv1.GetSelfHealthRequest]) (*connect.Response[runsv1.GetSelfHealthResponse], error) {
	if f.sleep > 0 {
		select {
		case <-time.After(f.sleep):
		case <-ctx.Done():
			return nil, connect.NewError(connect.CodeDeadlineExceeded, ctx.Err())
		}
	}
	return connect.NewResponse(f.resp), nil
}

type fakeGraph struct {
	graphconnect.UnimplementedGraphServiceHandler
	resp *graphv1.GetHealthScoresResponse
}

type fakeRouting struct {
	routingconnect.UnimplementedRoutingServiceHandler
	resp *routingv1.StatusResponse
	err  error
}

func (f fakeRouting) Status(context.Context, *connect.Request[routingv1.StatusRequest]) (*connect.Response[routingv1.StatusResponse], error) {
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(f.resp), nil
}

type fakeEval struct {
	evalconnect.UnimplementedEvalServiceHandler
	suites []*evalv1.EvalSuite
	runs   map[string][]*evalv1.EvalRun
	delay  map[string]time.Duration
	err    error
}

func (f fakeEval) ListSuites(context.Context, *connect.Request[evalv1.ListSuitesRequest]) (*connect.Response[evalv1.ListSuitesResponse], error) {
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(&evalv1.ListSuitesResponse{Suites: f.suites}), nil
}

func (f fakeEval) ListRuns(ctx context.Context, req *connect.Request[evalv1.ListRunsRequest]) (*connect.Response[evalv1.ListRunsResponse], error) {
	if f.err != nil {
		return nil, f.err
	}
	if delay := f.delay[req.Msg.GetSuiteId()]; delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, connect.NewError(connect.CodeDeadlineExceeded, ctx.Err())
		}
	}
	return connect.NewResponse(&evalv1.ListRunsResponse{Runs: f.runs[req.Msg.GetSuiteId()]}), nil
}

func (f fakeGraph) GetHealthScores(context.Context, *connect.Request[graphv1.GetHealthScoresRequest]) (*connect.Response[graphv1.GetHealthScoresResponse], error) {
	return connect.NewResponse(f.resp), nil
}

func newJoinerFor(url string, deadline time.Duration) *apiNumeratorJoiner {
	return newAPINumeratorJoiner(fakeResolver{url: url}, http.DefaultClient, deadline)
}

func TestJoinAnswerEndToEnd(t *testing.T) {
	mux := http.NewServeMux()
	path, h := registryconnect.NewRegistryServiceHandler(fakeRegistry{resp: &registryv1.ListProvidersResponse{
		Providers: []*registryv1.ProviderDescriptor{{ProviderId: "ui-health.surfaces"}},
	}})
	mux.Handle(path, h)
	path, h = routingconnect.NewRoutingServiceHandler(fakeRouting{resp: &routingv1.StatusResponse{Providers: []*routingv1.ProviderHealth{{ProviderId: "ui-health.surfaces", Reachable: true, Reachability: "reachable"}}}})
	mux.Handle(path, h)
	created := time.Now().UTC().Format(time.RFC3339Nano)
	path, h = evalconnect.NewEvalServiceHandler(fakeEval{
		suites: []*evalv1.EvalSuite{{SuiteId: "ui-suite", ProviderId: "ui-health.surfaces"}},
		runs: map[string][]*evalv1.EvalRun{"ui-suite": {{
			RunId: "run-1", CreatedAt: created, Tier: "provider_direct",
			// IndexedCount is intentionally omitted: it is optional telemetry,
			// not a quality gate for a graded evaluation.
			Config:    &evalv1.ConfigSnapshot{},
			Aggregate: &evalv1.EvalAggregate{Cases: 1, Met: 1, GradedCases: 1},
		}}},
	})
	mux.Handle(path, h)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	res := newJoinerFor(srv.URL, numeratorDeadline).Join(context.Background(), ProjectionAnswer, answerDef().Cells)
	if !res.Available {
		t.Fatalf("expected available: %s", res.Reason)
	}
	if res.Statuses["1"] != spacedoc.StatusNow {
		t.Errorf("cell1 (live provider) = %v, want now", res.Statuses["1"])
	}
	if !strings.Contains(fmt.Sprint(res.Evidence), "indexed_count not reported") {
		t.Errorf("evidence = %v, want instrumentation note for missing indexed_count", res.Evidence)
	}
}

func TestJoinAnswerEvalFreshDoesNotHoldForUngradedEmptyRun(t *testing.T) {
	mux := http.NewServeMux()
	path, h := registryconnect.NewRegistryServiceHandler(fakeRegistry{resp: &registryv1.ListProvidersResponse{
		Providers: []*registryv1.ProviderDescriptor{{ProviderId: "ui-health.surfaces"}},
	}})
	mux.Handle(path, h)
	path, h = routingconnect.NewRoutingServiceHandler(fakeRouting{resp: &routingv1.StatusResponse{Providers: []*routingv1.ProviderHealth{{ProviderId: "ui-health.surfaces", Reachable: true, Reachability: "reachable"}}}})
	mux.Handle(path, h)
	path, h = evalconnect.NewEvalServiceHandler(fakeEval{
		suites: []*evalv1.EvalSuite{{SuiteId: "ui-suite", ProviderId: "ui-health.surfaces"}},
		runs: map[string][]*evalv1.EvalRun{"ui-suite": {{
			RunId: "empty-run", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
			Config: &evalv1.ConfigSnapshot{}, Aggregate: &evalv1.EvalAggregate{Cases: 1, Met: 0, GradedCases: 0},
		}}},
	})
	mux.Handle(path, h)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	res := newJoinerFor(srv.URL, numeratorDeadline).Join(context.Background(), ProjectionAnswer, answerDef().Cells)
	if !res.Available {
		t.Fatalf("expected available: %s", res.Reason)
	}
	if res.Statuses["1"] == spacedoc.StatusNow {
		t.Fatalf("cell1 promoted by ungraded empty eval: statuses=%v evidence=%v", res.Statuses, res.Evidence)
	}
	if !strings.Contains(fmt.Sprint(res.Evidence), "graded_cases=0") {
		t.Fatalf("evidence = %v, want graded_cases=0", res.Evidence)
	}
}

func TestFreshEvalByProviderDoesNotMixSuiteVerdictAndEvidence(t *testing.T) {
	mux := http.NewServeMux()
	path, handler := evalconnect.NewEvalServiceHandler(fakeEval{
		suites: []*evalv1.EvalSuite{
			{SuiteId: "failing-suite", ProviderId: "provider.one"},
			{SuiteId: "passing-suite", ProviderId: "provider.one"},
		},
		delay: map[string]time.Duration{
			// Force the failing suite to complete first. The old merge kept
			// that verdict and then replaced its evidence with the later
			// passing suite, reproducing the production symptom.
			"passing-suite": 20 * time.Millisecond,
		},
		runs: map[string][]*evalv1.EvalRun{
			"failing-suite": {{
				RunId: "failing-run", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
				Config:    &evalv1.ConfigSnapshot{IndexedCount: 1},
				Aggregate: &evalv1.EvalAggregate{Cases: 1, Met: 0, GradedCases: 1},
			}},
			"passing-suite": {{
				RunId: "passing-run", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
				Config:    &evalv1.ConfigSnapshot{IndexedCount: 1},
				Aggregate: &evalv1.EvalAggregate{Cases: 1, Met: 1, GradedCases: 1},
			}},
		},
	})
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := evalconnect.NewEvalServiceClient(srv.Client(), srv.URL)
	got, err := newJoinerFor(srv.URL, time.Second).freshEvalByProvider(context.Background(), client, []*evalv1.EvalSuite{
		{SuiteId: "failing-suite", ProviderId: "provider.one"},
		{SuiteId: "passing-suite", ProviderId: "provider.one"},
	})
	if err != nil {
		t.Fatalf("freshEvalByProvider() error = %v", err)
	}
	result := got["provider.one"]
	if result.Fresh {
		t.Fatal("provider with a failing suite must not be fresh")
	}
	if !strings.Contains(result.Evidence, "failing-run") {
		t.Fatalf("evidence = %q, want the suite that decided the failing verdict", result.Evidence)
	}
	if strings.Contains(result.Evidence, "passing-run") {
		t.Fatalf("evidence = %q, must not describe a different suite than the verdict", result.Evidence)
	}
}

func TestFreshEvalByProviderWorstSuiteWinsAcrossCompletionOrders(t *testing.T) {
	orders := []struct {
		name  string
		delay map[string]time.Duration
	}{
		{name: "failing completes first", delay: map[string]time.Duration{"passing-suite": 20 * time.Millisecond}},
		{name: "passing completes first", delay: map[string]time.Duration{"failing-suite": 20 * time.Millisecond}},
	}
	for _, order := range orders {
		t.Run(order.name, func(t *testing.T) {
			mux := http.NewServeMux()
			path, handler := evalconnect.NewEvalServiceHandler(fakeEval{
				delay: order.delay,
				runs: map[string][]*evalv1.EvalRun{
					"failing-suite": {{RunId: "failing-run", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), Config: &evalv1.ConfigSnapshot{IndexedCount: 1}, Aggregate: &evalv1.EvalAggregate{Cases: 1, Met: 0, GradedCases: 1}}},
					"passing-suite": {{RunId: "passing-run", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), Config: &evalv1.ConfigSnapshot{IndexedCount: 1}, Aggregate: &evalv1.EvalAggregate{Cases: 1, Met: 1, GradedCases: 1}}},
				},
			})
			mux.Handle(path, handler)
			srv := httptest.NewServer(mux)
			defer srv.Close()

			client := evalconnect.NewEvalServiceClient(srv.Client(), srv.URL)
			got, err := newJoinerFor(srv.URL, time.Second).freshEvalByProvider(context.Background(), client, []*evalv1.EvalSuite{
				{SuiteId: "failing-suite", ProviderId: "provider.one"},
				{SuiteId: "passing-suite", ProviderId: "provider.one"},
			})
			if err != nil {
				t.Fatalf("freshEvalByProvider() error = %v", err)
			}
			result := got["provider.one"]
			if result.Fresh || !strings.Contains(result.Evidence, "failing-run") {
				t.Fatalf("result = %+v, want failing verdict and matching evidence", result)
			}
		})
	}
}

func TestAnswerSignalsIndividuallyUnavailable(t *testing.T) {
	cases := []struct {
		name        string
		registryErr error
		routingErr  error
		evalErr     error
		want        string
	}{
		{name: "registry", registryErr: connect.NewError(connect.CodeUnavailable, errors.New("registry down")), want: "ListProviders"},
		{name: "reachability", routingErr: connect.NewError(connect.CodeUnavailable, errors.New("routing down")), want: "Status"},
		{name: "eval", evalErr: connect.NewError(connect.CodeUnavailable, errors.New("eval down")), want: "ListSuites"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mux := http.NewServeMux()
			path, h := registryconnect.NewRegistryServiceHandler(fakeRegistry{resp: &registryv1.ListProvidersResponse{Providers: []*registryv1.ProviderDescriptor{{ProviderId: "ui-health.surfaces"}}}, err: tc.registryErr})
			mux.Handle(path, h)
			path, h = routingconnect.NewRoutingServiceHandler(fakeRouting{resp: &routingv1.StatusResponse{Providers: []*routingv1.ProviderHealth{{ProviderId: "ui-health.surfaces", Reachable: true, Reachability: "reachable"}}}, err: tc.routingErr})
			mux.Handle(path, h)
			path, h = evalconnect.NewEvalServiceHandler(fakeEval{suites: []*evalv1.EvalSuite{{SuiteId: "ui-suite", ProviderId: "ui-health.surfaces"}}, runs: map[string][]*evalv1.EvalRun{"ui-suite": {{RunId: "run-1", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}}}, err: tc.evalErr})
			mux.Handle(path, h)
			srv := httptest.NewServer(mux)
			defer srv.Close()
			res := newJoinerFor(srv.URL, time.Second).Join(context.Background(), ProjectionAnswer, answerDef().Cells)
			if res.Available || !strings.Contains(res.Reason, tc.want) {
				t.Fatalf("result = %+v, want unavailable reason containing %q", res, tc.want)
			}
		})
	}
}

func TestJoinValidateEndToEnd(t *testing.T) {
	mux := http.NewServeMux()
	path, h := runsconnect.NewRunsServiceHandler(fakeRuns{resp: &runsv1.GetSelfHealthResponse{
		SelfHealth: &runsv1.SelfHealth{
			Catalog: &runsv1.CatalogSummary{Phases: []*runsv1.CatalogPhase{{Provider: "red-health"}}},
			Ledger:  &runsv1.ReliabilityLedger{Phases: []*runsv1.PhaseReliability{{Provider: "red-health", FailureRate: 0.5}}},
		},
	}})
	mux.Handle(path, h)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cells := []spacedoc.Cell{{ID: "V1", Owner: "`red-health`", Status: spacedoc.StatusNow}}
	res := newJoinerFor(srv.URL, numeratorDeadline).Join(context.Background(), ProjectionValidate, cells)
	if !res.Available {
		t.Fatalf("expected available: %s", res.Reason)
	}
	if res.Statuses["V1"] != spacedoc.StatusNow {
		t.Errorf("failing phase = %v, want now (condition is separate)", res.Statuses["V1"])
	}
}

func TestJoinGuideEndToEnd(t *testing.T) {
	mux := http.NewServeMux()
	path, h := graphconnect.NewGraphServiceHandler(fakeGraph{resp: &graphv1.GetHealthScoresResponse{
		Scores: []*graphv1.HealthScore{{NodeId: "explore", Score: 0.9}},
	}})
	mux.Handle(path, h)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cells := []spacedoc.Cell{{ID: "G1", Owner: "`explore`", Status: spacedoc.StatusInReach}}
	res := newJoinerFor(srv.URL, numeratorDeadline).Join(context.Background(), ProjectionGuide, cells)
	if !res.Available {
		t.Fatalf("expected available: %s", res.Reason)
	}
	if res.Statuses["G1"] != spacedoc.StatusNow {
		t.Errorf("healthy skill = %v, want now", res.Statuses["G1"])
	}
}

func TestJoinOwnerNotRunning(t *testing.T) {
	j := newAPINumeratorJoiner(
		fakeResolver{err: &discovery.Error{Kind: discovery.ErrScenarioNotRunning, Scenario: "search-hub"}},
		http.DefaultClient, numeratorDeadline,
	)
	res := j.Join(context.Background(), ProjectionAnswer, answerDef().Cells)
	if res.Available {
		t.Fatal("expected unavailable when owner not running")
	}
	if res.Reason == "" {
		t.Error("expected an honest reason")
	}
}

func TestJoinOwnerSlowHitsDeadline(t *testing.T) {
	mux := http.NewServeMux()
	path, h := runsconnect.NewRunsServiceHandler(fakeRuns{
		resp:  &runsv1.GetSelfHealthResponse{SelfHealth: &runsv1.SelfHealth{}},
		sleep: 3 * time.Second, // far longer than the test deadline
	})
	mux.Handle(path, h)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cells := []spacedoc.Cell{{ID: "V1", Owner: "`red-health`", Status: spacedoc.StatusNow}}
	start := time.Now()
	res := newJoinerFor(srv.URL, 150*time.Millisecond).Join(context.Background(), ProjectionValidate, cells)
	elapsed := time.Since(start)

	if res.Available {
		t.Fatal("expected unavailable on deadline")
	}
	if elapsed > time.Second {
		t.Errorf("slow owner should degrade fast (~deadline), took %s", elapsed)
	}
}

// ---------------------------------------------------------------------------
// GetStatus: concurrent reads + graceful degradation of a single owner
// ---------------------------------------------------------------------------

// sleepyJoiner sleeps a fixed delay per Join (honoring ctx cancellation) and can
// mark specific projections unavailable to simulate a timed-out owner.
type sleepyJoiner struct {
	delay       time.Duration
	unavailable map[Projection]bool
}

func (j sleepyJoiner) Join(ctx context.Context, p Projection, _ []spacedoc.Cell) JoinResult {
	select {
	case <-time.After(j.delay):
	case <-ctx.Done():
		return JoinResult{Available: false, Reason: "ctx: " + ctx.Err().Error()}
	}
	if j.unavailable[p] {
		return JoinResult{Available: false, Reason: string(p) + " simulated timeout"}
	}
	return JoinResult{Available: true}
}

// allProjectionsReader supplies a denominator for every projection that has an
// owner shipping a space doc today. ProjectionAct is deliberately absent —
// program-runtime does not exist yet — so the board must still surface an Act
// row (degraded, with an honest reason) rather than dropping it.
func allProjectionsReader() fakeReader {
	return fakeReader{defs: map[Projection]*spacedoc.SpaceDefinition{
		ProjectionAnswer:   answerDef(),
		ProjectionGuide:    guideDef(),
		ProjectionValidate: validateDefFor(map[string]spacedoc.CellStatus{"V1": spacedoc.StatusNow}),
	}}
}

func TestGetStatusRunsProjectionsConcurrently(t *testing.T) {
	const delay = 200 * time.Millisecond
	svc := NewService(Deps{
		Reader: allProjectionsReader(),
		Joiner: sleepyJoiner{delay: delay},
		Clock:  fixedClock{},
	})
	start := time.Now()
	st, err := svc.GetStatus(context.Background(), "")
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	// Every projection in the canonical model gets a board row, including ones
	// whose owner cannot be read — asserting len(AllProjections) keeps this test
	// tracking the model instead of a frozen count.
	if len(st.Projections) != len(AllProjections) {
		t.Fatalf("projections=%d, want %d", len(st.Projections), len(AllProjections))
	}
	// Answer is deliberately read twice for the uncached determinism check, so
	// the concurrent ceiling is ~2*delay. Serial would be 5*delay; allow one
	// delay of scheduling slack without masking a serial implementation.
	if elapsed > 3*delay {
		t.Errorf("projections not concurrent: %s for %dx%s reads", elapsed, len(AllProjections), delay)
	}
}

func TestGetStatusDegradesOneOwnerKeepsOthers(t *testing.T) {
	svc := NewService(Deps{
		Reader: allProjectionsReader(),
		Joiner: sleepyJoiner{unavailable: map[Projection]bool{ProjectionValidate: true}},
		Clock:  fixedClock{},
	})
	st, err := svc.GetStatus(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	byProj := map[Projection]ProjectionCoverage{}
	for _, pc := range st.Projections {
		byProj[pc.Projection] = pc
	}
	if byProj[ProjectionValidate].Available {
		t.Error("validate should be UNAVAILABLE (simulated owner timeout)")
	}
	if byProj[ProjectionValidate].UnavailableReason == "" {
		t.Error("unavailable validate should carry an honest reason")
	}
	if !byProj[ProjectionAnswer].Available || !byProj[ProjectionGuide].Available {
		t.Error("the other two projections must still be computed when one owner times out")
	}
}

func readCoverageTestdata(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
