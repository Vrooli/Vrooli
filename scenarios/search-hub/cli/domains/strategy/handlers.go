package strategy

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	evalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval/eval_v1connect"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing/routing_v1connect"
)

const benchmarkTimeout = 45 * time.Minute

// The API's embedded strategy document names this row "lexical-cross-encoder".
// The CLI remains a separate Go module, so it deliberately treats the name as
// the public benchmark contract rather than importing API internals.
const activeStrategyName = "lexical-cross-encoder"

type handlers struct {
	core *cliapp.ScenarioApp
}

func (h *handlers) routingClient(timeout time.Duration) routingconnect.RoutingServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(h.core, timeout)
	return routingconnect.NewRoutingServiceClient(httpClient, baseURL)
}

func (h *handlers) listCall(_ cliapp.OperationContext) (*routingv1.StatusResponse, error) {
	response, err := h.routingClient(30*time.Second).Status(context.Background(), connect.NewRequest(&routingv1.StatusRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list retrieval strategies", err, nil)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("server returned no routing status")
	}
	return response.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, status *routingv1.StatusResponse) cliapp.ListReport {
	results := make([]string, 0, len(status.GetStrategies()))
	for _, strategy := range status.GetStrategies() {
		marker := ""
		if strategy.GetName() == status.GetActiveStrategy() {
			marker = " [active]"
		}
		results = append(results, strategy.GetName()+marker+": "+strategy.GetDescription())
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d registered retrieval strategies; active=%s.", len(results), status.GetActiveStrategy())}, ResultsHeading: "Strategies", Results: results, RetrievalHints: []string{"`search-hub strategy show <name>` — inspect the ordered stage ladder", "`search-hub strategy compare --apply=false` — run persisted guarded comparison"}}
}

func (h *handlers) showCall(ctx cliapp.OperationContext) (*routingv1.StatusResponse, error) {
	status, err := h.listCall(ctx)
	if err != nil {
		return nil, err
	}
	want := strings.TrimSpace(ctx.Positional("strategy_name"))
	for _, strategy := range status.GetStrategies() {
		if strategy.GetName() == want {
			return &routingv1.StatusResponse{ActiveStrategy: status.GetActiveStrategy(), Strategies: []*routingv1.RetrievalStrategyInfo{strategy}}, nil
		}
	}
	return nil, fmt.Errorf("retrieval strategy %q is not registered", want)
}

func (h *handlers) showReport(_ cliapp.OperationContext, status *routingv1.StatusResponse) cliapp.ListReport {
	strategy := status.GetStrategies()[0]
	results := make([]string, 0, len(strategy.GetStages()))
	for index, stage := range strategy.GetStages() {
		results = append(results, fmt.Sprintf("%d. %s params=%s", index+1, stage.GetKind(), stage.GetParamsJson()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%s%s: %s", strategy.GetName(), activeSuffix(strategy.GetName(), status.GetActiveStrategy()), strategy.GetDescription())}, ResultsHeading: "Stages", Results: results}
}

func activeSuffix(name, active string) string {
	if name == active {
		return " [active]"
	}
	return ""
}

func (h *handlers) compareCall(ctx cliapp.OperationContext) (*evalv1.CompareStrategiesResponse, error) {
	status, err := h.listCall(ctx)
	if err != nil {
		return nil, err
	}
	names := splitNames(ctx.Flag("strategies"))
	if len(names) == 0 {
		for _, strategy := range status.GetStrategies() {
			names = append(names, strategy.GetName())
		}
	}
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(h.core, benchmarkTimeout)
	client := evalconnect.NewEvalServiceClient(httpClient, baseURL)
	response, err := client.CompareStrategies(context.Background(), connect.NewRequest(&evalv1.CompareStrategiesRequest{SuiteId: "router.routing", StrategyNames: names, Apply: ctx.BoolFlag("apply"), Limit: 10}))
	if err != nil {
		return nil, cliapp.WrapAPIError("compare retrieval strategies", err, nil)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("server returned no strategy comparison")
	}
	return response.Msg, nil
}

func (h *handlers) compareReport(_ cliapp.OperationContext, result *evalv1.CompareStrategiesResponse) cliapp.MutationReport {
	changes := make([]string, 0, len(result.GetArms())+1)
	for _, arm := range result.GetArms() {
		changes = append(changes, fmt.Sprintf("%s run=%s all=%d precision=%.4f routable=%d precision=%.4f top1=%d top3=%d top6=%d heldout=%t significant=%t accepted=%t rejection=%s", arm.GetStrategyName(), arm.GetRunId(), arm.GetAllDenominator(), arm.GetAllRoutingPrecision(), arm.GetRoutableDenominator(), arm.GetRoutableRoutingPrecision(), arm.GetTop1(), arm.GetTop3(), arm.GetTop6(), arm.GetHeldoutHolds(), arm.GetSignificant(), arm.GetAccepted(), emptyDash(arm.GetRejectionReason())))
	}
	if result.GetWritebackReason() != "" {
		changes = append(changes, "write-back: "+result.GetWritebackReason())
	}
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Compared %d strategy arms against %s; active=%s.", len(result.GetArms()), result.GetSuiteId(), result.GetActiveStrategy())}, Changes: changes, NextCommand: []string{"`search-hub evals runs router.routing --tag strategy-compare:... --json` — inspect persisted arm evidence"}}
}

func splitNames(raw string) []string {
	seen := map[string]struct{}{}
	var names []string
	for _, name := range strings.Split(raw, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	return names
}

func emptyDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

type benchmarkResult struct {
	Strategy         string  `json:"strategy"`
	ActiveStrategy   string  `json:"active_strategy"`
	SuiteID          string  `json:"suite_id"`
	FixtureCases     int     `json:"fixture_cases"`
	Denominator      int     `json:"denominator"`
	Attempted        int     `json:"attempted"`
	Routed           int     `json:"routed"`
	RoutingPrecision float64 `json:"routing_precision"`
	Top1             float64 `json:"top1"`
	Top3             float64 `json:"top3"`
	Top6             float64 `json:"top6"`
	LatencyP50Ms     int64   `json:"latency_p50_ms"`
	LatencyP95Ms     int64   `json:"latency_p95_ms"`
	Unavailable      int     `json:"unavailable"`
	Note             string  `json:"note,omitempty"`
}

type benchmarkCaseMeasurement struct {
	latencyMs int64
	available bool
	routed    bool
	rank      int
}

func newHandlers(core *cliapp.ScenarioApp) *handlers { return &handlers{core: core} }

// benchmarkCall is the reproducible command-level harness for the router
// fixture. It deliberately issues the same public Query RPC an operator uses,
// so a benchmark cannot accidentally measure a private classifier seam.
func (h *handlers) benchmarkCall(ctx cliapp.OperationContext) (benchmarkResult, error) {
	strategyName := strings.TrimSpace(ctx.Flag("strategy"))
	if strategyName == "" {
		strategyName = activeStrategyName
	}
	if strategyName != activeStrategyName {
		return benchmarkResult{}, fmt.Errorf("strategy %q is defined but not active; use strategy compare for registered arms", strategyName)
	}

	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(h.core, benchmarkTimeout)
	evalClient := evalconnect.NewEvalServiceClient(httpClient, baseURL)
	routingClient := routingconnect.NewRoutingServiceClient(httpClient, baseURL)
	response, err := evalClient.GetSuite(context.Background(), connect.NewRequest(&evalv1.GetSuiteRequest{SuiteId: "router.routing"}))
	if err != nil {
		return benchmarkResult{}, cliapp.WrapAPIError("load router.routing benchmark fixture", err, nil)
	}
	if response == nil || response.Msg == nil || response.Msg.GetSuite() == nil {
		return benchmarkResult{}, fmt.Errorf("server returned no router.routing suite")
	}
	suite := response.Msg.GetSuite()
	result := benchmarkResult{Strategy: strategyName, ActiveStrategy: activeStrategyName, SuiteID: suite.GetSuiteId()}
	eligible := make([]*evalv1.EvalCase, 0, len(suite.GetCases()))
	for _, testCase := range suite.GetCases() {
		if testCase == nil || testCase.GetStatus() == "candidate" || testCase.GetExpectedProviderId() == "" || len(testCase.GetExpectIds()) == 0 {
			continue
		}
		eligible = append(eligible, testCase)
	}
	measurements := make([]benchmarkCaseMeasurement, len(eligible))
	jobs := make(chan int)
	var wg sync.WaitGroup
	// Router fan-out concurrency is eight, but the benchmark itself must not
	// create eight simultaneous model generations. Keeping two public queries
	// in flight preserves throughput while respecting local model residency and
	// makes the command reproducible on smaller desktop GPUs.
	workerCount := 2
	if len(eligible) < workerCount {
		workerCount = len(eligible)
	}
	for worker := 0; worker < workerCount; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				testCase := eligible[index]
				started := time.Now()
				queryResponse, queryErr := routingClient.Query(context.Background(), connect.NewRequest(&routingv1.QueryRequest{
					Query: testCase.GetQuery(), Limit: 10,
				}))
				measurement := benchmarkCaseMeasurement{latencyMs: time.Since(started).Milliseconds()}
				if queryErr == nil && queryResponse != nil && queryResponse.Msg != nil && !queryResponse.Msg.GetDegraded() {
					measurement.available = true
					measurement.routed = contains(queryResponse.Msg.GetCorporaSearched(), testCase.GetExpectedProviderId())
					measurement.rank = expectedRank(testCase.GetExpectIds(), rankedHits(queryResponse.Msg))
				}
				measurements[index] = measurement
			}
		}()
	}
	for index := range eligible {
		jobs <- index
	}
	close(jobs)
	wg.Wait()
	latencies := make([]int64, 0, len(measurements))
	var top1, top3, top6 int
	for _, measurement := range measurements {
		latencies = append(latencies, measurement.latencyMs)
		if !measurement.available {
			result.Unavailable++
			continue
		}
		result.Attempted++
		if measurement.routed {
			result.Routed++
		}
		if measurement.rank > 0 && measurement.rank <= 1 {
			top1++
		}
		if measurement.rank > 0 && measurement.rank <= 3 {
			top3++
		}
		if measurement.rank > 0 && measurement.rank <= 6 {
			top6++
		}
	}
	result.FixtureCases = len(eligible)
	result.Denominator = result.Attempted
	if result.Attempted > 0 {
		result.RoutingPrecision = float64(result.Routed) / float64(result.Attempted)
		result.Top1 = float64(top1) / float64(result.Attempted)
		result.Top3 = float64(top3) / float64(result.Attempted)
		result.Top6 = float64(top6) / float64(result.Attempted)
	}
	result.LatencyP50Ms = percentile(latencies, 50)
	result.LatencyP95Ms = percentile(latencies, 95)
	result.Note = "fixture_cases is the reviewed positive corpus size. denominator is the successful, non-degraded subset used for every quality rate; unavailable cases are reported separately. top1/top3/top6 are expected-id retrieval recall, while routing_precision measures expected-provider inclusion."
	return result, nil
}

func (h *handlers) benchmarkReport(_ cliapp.OperationContext, result benchmarkResult) cliapp.MutationReport {
	return cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Benchmarked %s against %s: denominator %d, routing precision %.4f, top-1 %.4f, top-3 %.4f, top-6 %.4f.", result.Strategy, result.SuiteID, result.Attempted, result.RoutingPrecision, result.Top1, result.Top3, result.Top6)},
		Changes: []string{
			fmt.Sprintf("Latency p50=%dms p95=%dms; routed=%d; unavailable=%d.", result.LatencyP50Ms, result.LatencyP95Ms, result.Routed, result.Unavailable),
			result.Note,
		},
		NextCommand: []string{"`search-hub evals runs router.routing --json` — inspect persisted federated runs", "`search-hub strategy benchmark --strategy lexical-cross-encoder --json` — reproduce the active measurement"},
	}
}

func rankedHits(response *routingv1.QueryResponse) []*routingv1.SearchHit {
	if response == nil {
		return nil
	}
	if len(response.GetRanked()) > 0 {
		return response.GetRanked()
	}
	var hits []*routingv1.SearchHit
	for _, group := range response.GetGroups() {
		if group != nil {
			hits = append(hits, group.GetHits()...)
		}
	}
	return hits
}

func expectedRank(expected []string, hits []*routingv1.SearchHit) int {
	wanted := make(map[string]struct{}, len(expected))
	for _, id := range expected {
		wanted[id] = struct{}{}
	}
	for index, hit := range hits {
		if hit != nil {
			if _, ok := wanted[hit.GetId()]; ok {
				return index + 1
			}
		}
	}
	return 0
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func percentile(values []int64, percentile int) int64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]int64(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	index := (percentile*len(sorted) + 99) / 100
	if index < 1 {
		index = 1
	}
	if index > len(sorted) {
		index = len(sorted)
	}
	return sorted[index-1]
}
