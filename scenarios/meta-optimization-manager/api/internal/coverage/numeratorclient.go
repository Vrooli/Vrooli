package coverage

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/spacedoc"

	bindingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings"
	bindingsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings/bindings_v1connect"
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

// numeratorDeadline bounds a single owner's live numerator read. The previous
// substrate spawned the owner's CLI with a 30s timeout, so one slow/hung owner
// (e.g. test-genie health aggregating its fleet ledger) stalled the whole
// scoreboard for ~30s. Reads are now a single typed Connect-RPC call resolved
// through api-core/discovery; they should return sub-second. The short deadline
// turns a slow/unreachable owner into a fast, honest per-projection UNAVAILABLE
// instead of a board-wide hang.
const numeratorDeadline = 3 * time.Second

// answerEvalFreshnessWindow is deliberately explicit and shared by every
// Answer join. A run older than this is historical evidence, not current
// readiness evidence. Keep this separate from the Search Hub index-age signal:
// the index can be fresh while the quality corpus has not been exercised.
const answerEvalFreshnessWindow = 30 * 24 * time.Hour

// answerEvalMinimumPassRate is the minimum graded pass fraction that can
// substantiate a current Answer cell. A recent run with zero indexed documents
// or zero graded cases is evidence of a broken measurement path, not freshness.
const answerEvalMinimumPassRate = 0.8

// scenarioResolver resolves an owner scenario slug to its API base URL. It seams
// over discovery.Resolver so tests can point reads at an httptest server (or
// simulate a not-running owner) without shelling out to the vrooli CLI.
type scenarioResolver interface {
	ResolveScenarioURLDefault(ctx context.Context, scenarioSlug string) (string, error)
}

// apiNumeratorJoiner is the production NumeratorJoiner: it resolves each owner's
// API base URL, calls the owner's typed Connect-RPC numerator read under a short
// deadline, and maps the typed response onto the projection's per-cell statuses.
// Any resolve/RPC/deadline failure yields a graceful Available=false JoinResult
// with an honest reason — never a fabricated number, never a board-wide hang.
type apiNumeratorJoiner struct {
	resolver scenarioResolver
	http     connect.HTTPClient
	deadline time.Duration
}

var _ NumeratorJoiner = (*apiNumeratorJoiner)(nil)

// NewNumeratorJoiner returns the production NumeratorJoiner wired to the CLI-
// backed discovery resolver and an HTTP client bounded by the read deadline.
func NewNumeratorJoiner() NumeratorJoiner {
	return newAPINumeratorJoiner(
		discovery.NewResolver(discovery.ResolverConfig{}),
		&http.Client{Timeout: numeratorDeadline},
		numeratorDeadline,
	)
}

// newAPINumeratorJoiner builds an apiNumeratorJoiner over an injected resolver,
// HTTP client, and deadline (tests).
func newAPINumeratorJoiner(r scenarioResolver, h connect.HTTPClient, deadline time.Duration) *apiNumeratorJoiner {
	if deadline <= 0 {
		deadline = numeratorDeadline
	}
	return &apiNumeratorJoiner{resolver: r, http: h, deadline: deadline}
}

// Join resolves the projection's owner, reads its live numerator over a typed
// client under the per-owner deadline, and returns the joined per-cell statuses.
func (j *apiNumeratorJoiner) Join(ctx context.Context, p Projection, cells []spacedoc.Cell) JoinResult {
	owner := OwnerFor(p)
	if owner == "" {
		return JoinResult{Available: false, Reason: "unknown coverage projection: " + string(p)}
	}

	ctx, cancel := context.WithTimeout(ctx, j.deadline)
	defer cancel()

	base, err := j.resolver.ResolveScenarioURLDefault(ctx, owner)
	if err != nil {
		return JoinResult{Available: false, Reason: resolveReason(owner, err)}
	}

	switch p {
	case ProjectionAnswer:
		return j.joinAnswer(ctx, base, cells)

	case ProjectionValidate:
		client := runsconnect.NewRunsServiceClient(j.http, base)
		resp, err := client.GetSelfHealth(ctx, connect.NewRequest(&runsv1.GetSelfHealthRequest{SkipConformance: true}))
		if err != nil {
			return JoinResult{Available: false, Reason: rpcReason(owner, "GetSelfHealth", err)}
		}
		statuses, conditions := recomputeValidateWithConditions(cells, selfHealthToValidateIndex(resp.Msg.GetSelfHealth()))
		return JoinResult{Available: true, Statuses: statuses, Conditions: conditions}

	case ProjectionGuide:
		client := graphconnect.NewGraphServiceClient(j.http, base)
		resp, err := client.GetHealthScores(ctx, connect.NewRequest(&graphv1.GetHealthScoresRequest{}))
		if err != nil {
			return JoinResult{Available: false, Reason: rpcReason(owner, "GetHealthScores", err)}
		}
		return JoinResult{Available: true, Statuses: recomputeGuide(cells, scoresToGuideIndex(resp.Msg.GetScores()))}

	case ProjectionAct:
		client := bindingsconnect.NewBindingRegistryServiceClient(j.http, base)
		request := &bindingsv1.ResolveActCellsRequest{Cells: make([]*bindingsv1.ActCell, 0, len(cells))}
		for _, cell := range cells {
			// The denominator owns the taxonomy, while the registry owns the
			// mechanical namespace. Owner text is the stable bridge between the
			// two: it names the scenario(s) expected to serve the operation class.
			request.Cells = append(request.Cells, &bindingsv1.ActCell{
				Id: cell.ID, Operations: []string{cell.Owner}, AuthoredStatus: string(cell.Status),
			})
		}
		resp, err := client.ResolveActCells(ctx, connect.NewRequest(request))
		if err != nil {
			return JoinResult{Available: false, Reason: rpcReason(owner, "ResolveActCells", err)}
		}
		return JoinResult{Available: true, Statuses: recomputeAct(cells, resp.Msg.GetCells()), DenominatorConfidence: actConfidence(resp.Msg.GetDenominatorConfidence())}

	default:
		return JoinResult{Available: false, Reason: "unknown coverage projection: " + string(p)}
	}
}

func (j *apiNumeratorJoiner) joinAnswer(ctx context.Context, base string, cells []spacedoc.Cell) JoinResult {
	registryClient := registryconnect.NewRegistryServiceClient(j.http, base)
	routingClient := routingconnect.NewRoutingServiceClient(j.http, base)
	evalClient := evalconnect.NewEvalServiceClient(j.http, base)

	// These reads are independent. Keep the caller's single short deadline so a
	// slow owner degrades the whole Answer projection honestly instead of
	// allowing one signal to outlive the others.
	type result struct {
		providers *registryv1.ListProvidersResponse
		status    *routingv1.StatusResponse
		suites    []*evalv1.EvalSuite
		err       error
		method    string
	}
	results := make(chan result, 3)
	var wg sync.WaitGroup
	call := func(method string, fn func() (any, error)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			value, err := fn()
			item := result{err: err, method: method}
			switch v := value.(type) {
			case *registryv1.ListProvidersResponse:
				item.providers = v
			case *routingv1.StatusResponse:
				item.status = v
			case []*evalv1.EvalSuite:
				item.suites = v
			}
			results <- item
		}()
	}
	call("ListProviders", func() (any, error) {
		// Fetch the complete registry for the authoring join. Capability-gap
		// descriptors are registered owners and must therefore resolve for the
		// base-doc integrity check, but they remain non-active evidence and can
		// never promote a cell to NOW. Filtering ACTIVE here conflates those two
		// questions and misreports every intentional gap stub as an unknown owner.
		resp, err := registryClient.ListProviders(ctx, connect.NewRequest(&registryv1.ListProvidersRequest{}))
		if err != nil {
			return nil, err
		}
		return resp.Msg, nil
	})
	call("Status", func() (any, error) {
		resp, err := routingClient.Status(ctx, connect.NewRequest(&routingv1.StatusRequest{}))
		if err != nil {
			return nil, err
		}
		return resp.Msg, nil
	})
	call("ListSuites", func() (any, error) {
		resp, err := evalClient.ListSuites(ctx, connect.NewRequest(&evalv1.ListSuitesRequest{}))
		if err != nil {
			return nil, err
		}
		return resp.Msg.GetSuites(), nil
	})
	wg.Wait()
	close(results)

	var providers *registryv1.ListProvidersResponse
	var status *routingv1.StatusResponse
	var suites []*evalv1.EvalSuite
	for item := range results {
		if item.err != nil {
			return JoinResult{Available: false, Reason: rpcReason(ownerForAnswer(), item.method, item.err)}
		}
		switch item.method {
		case "ListProviders":
			providers = item.providers
		case "Status":
			status = item.status
		case "ListSuites":
			suites = item.suites
		}
	}

	fresh, err := j.freshEvalByProvider(ctx, evalClient, suites)
	if err != nil {
		return JoinResult{Available: false, Reason: rpcReason(ownerForAnswer(), "ListRuns", err)}
	}
	providerEvidence := answerEvidence(providers, status, fresh)
	statuses, evidence := recomputeAnswer(cells, providerEvidence)
	conditions := make(map[string]ConditionVerdict)
	for _, cell := range cells {
		for _, token := range providerTokens(cell.Owner) {
			for _, provider := range providerEvidence {
				if matchesProviderToken(token, provider.ProviderID) && provider.Condition != "" {
					conditions[cell.ID] = provider.Condition
					break
				}
			}
			if _, ok := conditions[cell.ID]; ok {
				break
			}
		}
	}
	known := make(map[string]bool, len(providers.GetProviders()))
	for _, provider := range providers.GetProviders() {
		if provider == nil {
			continue
		}
		known[strings.ToLower(strings.TrimSpace(provider.GetProviderId()))] = true
	}
	resolved := make(map[string]bool, len(cells))
	for _, cell := range cells {
		resolved[cell.ID] = true
		for _, token := range providerTokens(cell.Owner) {
			if !known[token] {
				resolved[cell.ID] = false
				break
			}
		}
	}
	return JoinResult{Available: true, Statuses: statuses, Evidence: evidence, Conditions: conditions, OwnerResolved: resolved}
}

func ownerForAnswer() string { return "search-hub" }

type evalFreshness struct {
	Fresh     bool
	Available bool
	Evidence  string
}

type evalSuiteFreshness struct {
	ProviderID string
	SuiteID    string
	Freshness  evalFreshness
	Err        error
}

func (j *apiNumeratorJoiner) freshEvalByProvider(ctx context.Context, client evalconnect.EvalServiceClient, suites []*evalv1.EvalSuite) (map[string]evalFreshness, error) {
	results := make(chan evalSuiteFreshness, len(suites))
	var wg sync.WaitGroup
	for _, suite := range suites {
		if suite == nil || strings.TrimSpace(suite.GetProviderId()) == "" {
			continue
		}
		suite := suite
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := client.ListRuns(ctx, connect.NewRequest(&evalv1.ListRunsRequest{SuiteId: suite.GetSuiteId(), Limit: 20}))
			if err != nil {
				results <- evalSuiteFreshness{ProviderID: suite.GetProviderId(), SuiteID: suite.GetSuiteId(), Err: err}
				return
			}
			current := evalFreshness{Available: true, Evidence: "no graded eval run in the last 30d (graded_cases=0)"}
			var unavailableReason string
			for _, run := range resp.Msg.GetRuns() {
				if run == nil {
					continue
				}
				at, parseErr := time.Parse(time.RFC3339Nano, run.GetCreatedAt())
				if parseErr != nil {
					continue
				}
				age := time.Since(at)
				if age >= 0 && age <= answerEvalFreshnessWindow {
					graded := gradedCases(run)
					if graded == 0 {
						if reason := strings.TrimSpace(run.GetUnavailableReason()); reason != "" {
							unavailableReason = reason
						} else if len(run.GetUnavailableCases()) > 0 {
							unavailableReason = run.GetUnavailableCases()[0].GetReason()
						} else if run.GetDegraded() {
							unavailableReason = run.GetDegradedReason()
						}
						continue
					}
					indexed := run.GetConfig().GetIndexedCount()
					passRate := 0.0
					if graded > 0 {
						passRate = float64(runMetCases(run)) / float64(graded)
					}
					current.Evidence = fmt.Sprintf("suite %s: latest graded eval run %s pass_rate=%.2f graded_cases=%d indexed_count=%d", suite.GetSuiteId(), run.GetRunId(), passRate, graded, indexed)
					if graded > 0 && passRate >= answerEvalMinimumPassRate {
						current.Fresh = true
					}
					if indexed == 0 {
						current.Evidence += " (instrumentation note: indexed_count not reported)"
					}
					break
				}
			}
			if !current.Fresh && unavailableReason != "" {
				current.Available = false
				current.Evidence = fmt.Sprintf("UNAVAILABLE: suite %s has no graded eval run in the last 30d; reason: %s", suite.GetSuiteId(), unavailableReason)
			}
			results <- evalSuiteFreshness{ProviderID: suite.GetProviderId(), SuiteID: suite.GetSuiteId(), Freshness: current}
		}()
	}
	wg.Wait()
	close(results)

	byProvider := make(map[string][]evalSuiteFreshness, len(suites))
	for result := range results {
		if result.Err != nil {
			return nil, result.Err
		}
		byProvider[result.ProviderID] = append(byProvider[result.ProviderID], result)
	}

	out := make(map[string]evalFreshness, len(byProvider))
	for providerID, providerResults := range byProvider {
		sort.Slice(providerResults, func(i, k int) bool {
			return providerResults[i].SuiteID < providerResults[k].SuiteID
		})
		aggregate := evalFreshness{Fresh: true, Available: true}
		for _, result := range providerResults {
			if !result.Freshness.Available {
				aggregate = result.Freshness
				break
			}
			if !result.Freshness.Fresh {
				// Worst-suite-wins: the first failing suite in stable suite
				// order decides both the verdict and the evidence.
				aggregate = result.Freshness
				break
			}
			// If every suite is fresh, the final suite is the deterministic
			// deciding evidence for the all-suites fold.
			aggregate.Evidence = result.Freshness.Evidence
		}
		out[providerID] = aggregate
	}
	return out, nil
}

func gradedCases(run *evalv1.EvalRun) int32 {
	if run == nil {
		return 0
	}
	if value := run.GetAggregate().GetGradedCases(); value > 0 {
		return value
	}
	var count int32
	for _, result := range run.GetResults() {
		switch result.GetOutcome() {
		case "met", "below_expectation", "above_expectation", "unexpected_hit", "answered_by_sibling", "misrouted", "thin_margin":
			count++
		}
	}
	return count
}

func runMetCases(run *evalv1.EvalRun) int32 {
	if run == nil {
		return 0
	}
	if run.GetAggregate().GetGradedCases() > 0 {
		return run.GetAggregate().GetMet()
	}
	var count int32
	for _, result := range run.GetResults() {
		if result.GetOutcome() == "met" {
			count++
		}
	}
	return count
}

func answerEvidence(resp *registryv1.ListProvidersResponse, status *routingv1.StatusResponse, fresh map[string]evalFreshness) []answerProviderEvidence {
	reachable := make(map[string]*routingv1.ProviderHealth)
	if status != nil {
		for _, health := range status.GetProviders() {
			if health != nil {
				reachable[strings.ToLower(health.GetProviderId())] = health
			}
		}
	}
	out := make([]answerProviderEvidence, 0)
	if resp == nil {
		return out
	}
	for _, provider := range resp.GetProviders() {
		if provider == nil {
			continue
		}
		id := strings.ToLower(strings.TrimSpace(provider.GetProviderId()))
		health, ok := reachable[id]
		reachEvidence := "provider health not reported"
		reach := false
		if ok {
			reach = providerReachable(health)
			reachEvidence = fmt.Sprintf("%s reachability=%q reachable=%t degraded=%t", id, health.GetReachability(), health.GetReachable(), health.GetDegraded())
		}
		condition := ConditionOK
		if !ok || !health.GetReachable() || health.GetDegraded() {
			condition = ConditionDegraded
		}
		eval := fresh[id]
		if eval.Evidence == "" {
			eval.Evidence = "no graded eval run in the last 30d (graded_cases=0)"
		}
		active := provider.GetState() != registryv1.ProviderState_PROVIDER_STATE_CAPABILITY_GAP
		out = append(out, answerProviderEvidence{ProviderID: id, Active: active, Reachable: reach, FreshEval: eval.Fresh, EvalAvailable: eval.Available, Condition: condition, ReachabilityEvidence: reachEvidence, EvalEvidence: eval.Evidence})
	}
	return out
}

func providerReachable(health *routingv1.ProviderHealth) bool {
	if !health.GetReachable() {
		return false
	}
	// The status contract deliberately carries a human-readable reason, not a
	// closed enum. `endpoint resolved` is the current healthy value; accept
	// other positive prose while rejecting explicit failure/not-reported text.
	reason := strings.ToLower(strings.TrimSpace(health.GetReachability()))
	for _, negative := range []string{"unreachable", "failed", "not_reported", "unavailable", "no http endpoint"} {
		if strings.Contains(reason, negative) {
			return false
		}
	}
	return true
}

func actConfidence(value string) spacedoc.DenominatorConfidence {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "authoritative":
		return spacedoc.ConfidenceAuthoritative
	case "partial", "measured":
		return spacedoc.ConfidencePartial
	default:
		return spacedoc.ConfidenceSketch
	}
}

func recomputeAct(cells []spacedoc.Cell, verdicts []*bindingsv1.ActCellVerdict) map[string]spacedoc.CellStatus {
	out := make(map[string]spacedoc.CellStatus, len(verdicts))
	for _, verdict := range verdicts {
		if verdict == nil {
			continue
		}
		switch verdict.GetVerdict() {
		case bindingsv1.ActVerdict_ACT_VERDICT_NOW:
			out[verdict.GetId()] = spacedoc.StatusNow
		case bindingsv1.ActVerdict_ACT_VERDICT_IN_REACH:
			out[verdict.GetId()] = spacedoc.StatusInReach
		case bindingsv1.ActVerdict_ACT_VERDICT_AUTHORED:
			for _, cell := range cells {
				if cell.ID == verdict.GetId() {
					out[cell.ID] = cell.Status
					break
				}
			}
		}
	}
	return out
}

// resolveReason formats an honest reason for a failed owner URL resolution,
// distinguishing the not-running case from other discovery failures.
func resolveReason(owner string, err error) string {
	if discovery.IsScenarioNotRunning(err) {
		return owner + " not running"
	}
	return owner + " registry unreachable: " + err.Error()
}

// rpcReason formats an honest reason for a failed owner RPC. A deadline trip
// reads as a timeout; any other failure carries the underlying error.
func rpcReason(owner, method string, err error) string {
	if isDeadline(err) {
		return fmt.Sprintf("%s %s timed out", owner, method)
	}
	return fmt.Sprintf("%s %s failed: %v", owner, method, err)
}

func isDeadline(err error) bool {
	if connect.CodeOf(err) == connect.CodeDeadlineExceeded {
		return true
	}
	return strings.Contains(err.Error(), context.DeadlineExceeded.Error())
}

// providersToLive distills an ACTIVE search-hub ListProviders response into
// the provider key set the Answer join matches against. Only provider identity
// and grouping are valid denominator owners; the descriptor type is a routing
// classification and must never satisfy an Answer cell by itself.
func providersToLive(resp *registryv1.ListProvidersResponse) map[string]bool {
	out := map[string]bool{}
	if resp == nil {
		return out
	}
	add := func(s string) {
		if s = strings.ToLower(strings.TrimSpace(s)); s != "" {
			out[s] = true
		}
	}
	for _, p := range resp.GetProviders() {
		add(p.GetProviderId())
		add(p.GetProviderGroup())
	}
	return out
}

// selfHealthToValidateIndex distills a test-genie SelfHealth message into the
// per-provider Validate signal: catalog phases seed the provider set; a ledger
// phase with a positive failure rate marks the provider failing; a conformance
// entry with pending autofix work marks it pending. Mirrors the shape the
// previous JSON-walking joiner derived.
func selfHealthToValidateIndex(h *runsv1.SelfHealth) map[string]validateProviderStatus {
	out := map[string]validateProviderStatus{}
	if h == nil {
		return out
	}
	if cat := h.GetCatalog(); cat != nil {
		for _, ph := range cat.GetPhases() {
			if prov := strings.ToLower(ph.GetProvider()); prov != "" {
				if _, ok := out[prov]; !ok {
					out[prov] = validateProviderStatus{}
				}
			}
		}
	}
	if led := h.GetLedger(); led != nil {
		for _, ph := range led.GetPhases() {
			prov := strings.ToLower(ph.GetProvider())
			if prov == "" {
				continue
			}
			st := out[prov]
			if ph.GetFailureRate() > 0 {
				st.failing = true
				st.condition = ConditionDegraded
				if since, err := time.Parse(time.RFC3339Nano, ph.GetFailureStreakSince()); err == nil {
					st.sustained = time.Since(since) >= sustainedDegradationWindow
				}
			} else if st.condition == "" {
				st.condition = ConditionOK
			}
			out[prov] = st
		}
	}
	for provider, status := range out {
		if status.condition == "" {
			status.condition = ConditionUninstrumented
			out[provider] = status
		}
	}
	for _, cf := range h.GetConformance() {
		prov := strings.ToLower(cf.GetProvider())
		if prov == "" {
			continue
		}
		st := out[prov]
		if af := cf.GetAutofix(); af != nil && af.GetPending() > 0 {
			st.autofixPending = true
			st.condition = ConditionDegraded
		}
		out[prov] = st
	}
	return out
}

// scoresToGuideIndex distills prompt-manager graph health scores into the
// node-id → score index the Guide join resolves skills against.
func scoresToGuideIndex(scores []*graphv1.HealthScore) map[string]float64 {
	out := make(map[string]float64, len(scores))
	for _, s := range scores {
		if id := strings.ToLower(s.GetNodeId()); id != "" {
			out[id] = s.GetScore()
		}
	}
	return out
}
