package coverage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/spacedoc"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/graph"
	graphconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/graph/graph_v1connect"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry/registry_v1connect"
	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	runsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
)

// ---------------------------------------------------------------------------
// Pure per-cell join rules (transport-independent)
// ---------------------------------------------------------------------------

func TestRecomputeAnswer(t *testing.T) {
	// ui-health is live; cartographer is not.
	live := map[string]bool{"ui-health.surfaces": true, "code-facts": true}
	out := recomputeAnswer(answerDef().Cells, live)
	if out["1"] != spacedoc.StatusNow {
		t.Errorf("cell1 (live provider) = %v, want now", out["1"])
	}
	if st, ok := out["2"]; ok && st == spacedoc.StatusNow {
		t.Errorf("cell2 (provider not live) must not be promoted to now: %v", st)
	}
}

func TestRecomputeAnswerDriftDowngrade(t *testing.T) {
	// Authored-NOW cell 1, but its provider is not live -> honest downgrade.
	out := recomputeAnswer(answerDef().Cells, map[string]bool{"code-facts": true})
	if out["1"] != spacedoc.StatusInReach {
		t.Errorf("authored-NOW with dead provider should downgrade to in_reach, got %v", out["1"])
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
	if out["V2"] != spacedoc.StatusInReach {
		t.Errorf("failing phase = %v, want in_reach", out["V2"])
	}
	if out["V3"] != spacedoc.StatusInReach {
		t.Errorf("autofix-pending phase = %v, want in_reach", out["V3"])
	}
	if _, ok := out["V4"]; ok {
		t.Errorf("unknown phase should keep authored status, got overlay %v", out["V4"])
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
	for _, want := range []string{"ui-health.surfaces", "ui-health", "code-facts", "structured"} {
		if !live[want] {
			t.Errorf("expected live key %q in %v", want, live)
		}
	}
	if providersToLive(nil) == nil {
		t.Error("nil response should yield an empty (non-nil) set")
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
	if !validateIndex["storage-health"].autofixPending {
		t.Fatal("captured test-genie health fixture should expose storage-health pending autofix work")
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
}

func (f fakeRegistry) ListProviders(context.Context, *connect.Request[registryv1.ListProvidersRequest]) (*connect.Response[registryv1.ListProvidersResponse], error) {
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
	srv := httptest.NewServer(mux)
	defer srv.Close()

	res := newJoinerFor(srv.URL, numeratorDeadline).Join(context.Background(), ProjectionAnswer, answerDef().Cells)
	if !res.Available {
		t.Fatalf("expected available: %s", res.Reason)
	}
	if res.Statuses["1"] != spacedoc.StatusNow {
		t.Errorf("cell1 (live provider) = %v, want now", res.Statuses["1"])
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
	if res.Statuses["V1"] != spacedoc.StatusInReach {
		t.Errorf("failing phase = %v, want in_reach", res.Statuses["V1"])
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
	if len(st.Projections) != 3 {
		t.Fatalf("projections=%d, want 3", len(st.Projections))
	}
	// Serial would be 3*delay; concurrent is ~1*delay. Allow generous slack.
	if elapsed > 2*delay {
		t.Errorf("projections not concurrent: %s for 3x%s reads", elapsed, delay)
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
