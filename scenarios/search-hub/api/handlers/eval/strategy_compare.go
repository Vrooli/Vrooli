package eval

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"sort"
	"strings"

	"connectrpc.com/connect"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	internalregistry "search-hub/internal/registry"
)

const strategyHeldoutFraction = 0.30

// CompareStrategies runs every requested strategy through the existing
// federated evaluator, persists every run before scoring it, and applies the
// same conservative paired-bootstrap + held-out gates used by parameter
// sweeps. It intentionally does not mutate the embedded strategy catalog: an
// accepted proposal is reported as write-back eligible, while an unqualified
// proposal is explicitly refused and its evidence remains inspectable.
func (h *connectHandler) CompareStrategies(ctx context.Context, req *connect.Request[evalv1.CompareStrategiesRequest]) (*connect.Response[evalv1.CompareStrategiesResponse], error) {
	if h.deps.Federated == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("federated strategy comparison is not configured"))
	}
	suite, err := h.getSuite(ctx, req.Msg.GetSuiteId())
	if err != nil {
		return nil, h.logged("eval.CompareStrategies.getSuite", req.Msg.GetSuiteId(), err)
	}
	if suite.GetSuiteId() != "router.routing" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("strategy comparison requires the router.routing suite"))
	}
	strategyRunner, ok := h.deps.Federated.(StrategyRunner)
	if !ok {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("federated strategy evaluation is not configured"))
	}
	names := uniqueStrategyNames(req.Msg.GetStrategyNames())
	if len(names) < 3 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("at least three registered strategies are required for comparison"))
	}
	active := strings.TrimSpace(h.deps.ActiveStrategy)
	if active == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("active strategy is not configured"))
	}
	if !containsName(names, active) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("comparison must include active strategy %q", active))
	}
	routable := make(map[string]struct{})
	if h.deps.Routability != nil {
		status, statusErr := h.deps.Routability.Status(ctx)
		if statusErr != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.New("internal error"))
		}
		for _, provider := range status.GetProviders() {
			if provider != nil && provider.GetAutomaticEligible() {
				routable[provider.GetProviderId()] = struct{}{}
			}
		}
	} else {
		activeProviders, listErr := h.deps.Registry.List(ctx, internalregistry.ListFilter{State: int32(registryv1.ProviderState_PROVIDER_STATE_ACTIVE)})
		if listErr != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.New("internal error"))
		}
		for _, provider := range activeProviders {
			if provider != nil && provider.GetEndpoint() != nil && provider.GetLifecycle() == registryv1.Lifecycle_LIFECYCLE_PRODUCTION {
				routable[provider.GetProviderId()] = struct{}{}
			}
		}
	}

	runs := make(map[string]*evalv1.EvalRun, len(names))
	for _, name := range names {
		tag := "strategy-compare:" + name
		run, runErr := strategyRunner.RunWithStrategy(ctx, suite, tag, req.Msg.GetLimit(), name)
		if runErr != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("strategy %q: %w", name, runErr))
		}
		if runErr = h.deps.Store.AppendRun(ctx, run); runErr != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.New("internal error"))
		}
		runs[name] = run
	}

	result := &evalv1.CompareStrategiesResponse{SuiteId: suite.GetSuiteId(), ActiveStrategy: active}
	for _, name := range names {
		arm := scoreStrategyArm(suite, runs[name], routable)
		result.Arms = append(result.Arms, arm)
	}
	baseline := runs[active]
	baselineRecall := routingRecallByCase(suite, baseline)
	_, heldout := strategySplit(suite)
	for _, arm := range result.Arms {
		candidate := runs[arm.GetStrategyName()]
		if arm.GetStrategyName() == active {
			arm.Accepted = true
			arm.HeldoutHolds = true
			arm.SignificanceReason = "incumbent"
			continue
		}
		candidateRecall := routingRecallByCase(suite, candidate)
		if len(candidateRecall) == 0 {
			arm.HeldoutReason = "no gradeable evidence retained for this strategy arm"
			arm.RejectionReason = strings.TrimSpace(candidate.GetDegradedReason())
			if arm.RejectionReason == "" {
				arm.RejectionReason = arm.HeldoutReason
			}
			continue
		}
		mean, low := pairedRoutingCI(candidateRecall, baselineRecall)
		arm.Significant = low > 0
		arm.SignificanceReason = fmt.Sprintf("paired routing delta mean=%.4f, 95%% CI lower=%.4f", mean, low)
		arm.HeldoutHolds, arm.HeldoutReason = heldoutRoutingHolds(candidateRecall, baselineRecall, heldout)
		arm.Accepted = arm.Significant && arm.HeldoutHolds
		if !arm.Accepted {
			arm.RejectionReason = joinReasons(arm.SignificanceReason, arm.HeldoutReason, !arm.Significant, !arm.HeldoutHolds)
		}
	}

	if req.Msg.GetApply() {
		winner := bestAccepted(result.Arms, active)
		if winner == nil {
			result.WritebackReason = "write-back refused: no strategy cleared both paired significance and held-out validation"
		} else {
			result.WritebackReason = fmt.Sprintf("write-back eligible for %q; active strategy data is embedded and requires an explicit reviewed data-file change", winner.GetStrategyName())
		}
	}
	return connect.NewResponse(result), nil
}

func uniqueStrategyNames(names []string) []string {
	seen := make(map[string]struct{}, len(names))
	result := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func containsName(names []string, want string) bool {
	for _, name := range names {
		if name == want {
			return true
		}
	}
	return false
}

func scoreStrategyArm(suite *evalv1.EvalSuite, run *evalv1.EvalRun, routable map[string]struct{}) *evalv1.StrategyComparisonArm {
	arm := &evalv1.StrategyComparisonArm{StrategyName: strings.TrimPrefix(run.GetTag(), "strategy-compare:"), RunId: run.GetRunId(), Aggregate: run.GetAggregate()}
	for _, c := range suite.GetCases() {
		if c == nil || c.GetStatus() == "candidate" || c.GetExpectedProviderId() == "" || len(c.GetExpectIds()) == 0 {
			continue
		}
		cr := resultByCase(run, c.GetCaseId())
		if cr == nil || !strategyGradeable(cr.GetOutcome()) {
			continue
		}
		arm.AllDenominator++
		isRoutable := false
		if _, ok := routable[c.GetExpectedProviderId()]; ok {
			isRoutable = true
		}
		if isRoutable {
			arm.RoutableDenominator++
		}
		if cr.GetProviderRouted() {
			arm.AllRouted++
			if isRoutable {
				arm.RoutableRouted++
			}
		}
		if cr.GetExpectedRank() == 1 {
			arm.Top1++
		}
		if cr.GetExpectedRank() > 0 && cr.GetExpectedRank() <= 3 {
			arm.Top3++
		}
		if cr.GetExpectedRank() > 0 && cr.GetExpectedRank() <= 6 {
			arm.Top6++
		}
	}
	if arm.AllDenominator > 0 {
		arm.AllRoutingPrecision = float64(arm.AllRouted) / float64(arm.AllDenominator)
	}
	if arm.RoutableDenominator > 0 {
		arm.RoutableRoutingPrecision = float64(arm.RoutableRouted) / float64(arm.RoutableDenominator)
	}
	return arm
}

func resultByCase(run *evalv1.EvalRun, id string) *evalv1.CaseResult {
	for _, result := range run.GetResults() {
		if result.GetCaseId() == id {
			return result
		}
	}
	return nil
}

func strategyGradeable(outcome string) bool {
	switch outcome {
	case "met", "below_expectation", "above_expectation", "unexpected_hit", "answered_by_sibling", "misrouted", "thin_margin":
		return true
	default:
		return false
	}
}

func routingRecallByCase(suite *evalv1.EvalSuite, run *evalv1.EvalRun) map[string]float64 {
	result := make(map[string]float64)
	for _, c := range suite.GetCases() {
		if c == nil || c.GetStatus() == "candidate" || c.GetExpectedProviderId() == "" || len(c.GetExpectIds()) == 0 {
			continue
		}
		cr := resultByCase(run, c.GetCaseId())
		if cr != nil && strategyGradeable(cr.GetOutcome()) {
			if cr.GetProviderRouted() {
				result[c.GetCaseId()] = 1
			} else {
				result[c.GetCaseId()] = 0
			}
		}
	}
	return result
}

func strategySplit(suite *evalv1.EvalSuite) (tuning, heldout []string) {
	ids := make([]string, 0, len(suite.GetCases()))
	for _, c := range suite.GetCases() {
		if c != nil && c.GetStatus() != "candidate" && c.GetExpectedProviderId() != "" && len(c.GetExpectIds()) > 0 {
			ids = append(ids, c.GetCaseId())
		}
	}
	sort.Slice(ids, func(i, j int) bool { return strategyHash(ids[i]) < strategyHash(ids[j]) })
	k := int(math.Ceil(strategyHeldoutFraction * float64(len(ids))))
	if k < 3 {
		k = 3
	}
	if k > len(ids) {
		k = len(ids)
	}
	return ids[k:], ids[:k]
}

func strategyHash(id string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(id))
	return h.Sum64()
}

func pairedRoutingCI(candidate, baseline map[string]float64) (mean, low float64) {
	ids := make([]string, 0, len(baseline))
	for id := range baseline {
		if _, ok := candidate[id]; ok {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return 0, 0
	}
	sort.Strings(ids)
	delta := make([]float64, len(ids))
	for i, id := range ids {
		delta[i] = candidate[id] - baseline[id]
		mean += delta[i]
	}
	mean /= float64(len(delta))
	// Keep the bootstrap reproducible without using a non-cryptographic random
	// package in production code. This is a statistical sampler, not a source
	// of security-sensitive randomness.
	seed := uint64(1)
	means := make([]float64, 2000)
	for i := range means {
		for range delta {
			seed ^= seed << 13
			seed ^= seed >> 7
			seed ^= seed << 17
			means[i] += delta[seed%uint64(len(delta))]
		}
		means[i] /= float64(len(delta))
	}
	sort.Float64s(means)
	index := int(math.Ceil(0.025*float64(len(means)))) - 1
	if index < 0 {
		index = 0
	}
	return mean, means[index]
}

func heldoutRoutingHolds(candidate, baseline map[string]float64, heldout []string) (bool, string) {
	if len(heldout) < 3 {
		return false, "held-out fold too small to validate"
	}
	var candidateSum, baselineSum float64
	count := 0
	for _, id := range heldout {
		candidateValue, candidateOK := candidate[id]
		baselineValue, baselineOK := baseline[id]
		if !candidateOK || !baselineOK {
			continue
		}
		candidateSum += candidateValue
		baselineSum += baselineValue
		count++
	}
	if count < 3 {
		return false, "held-out fold has fewer than three gradeable paired cases"
	}
	if candidateSum+1e-9 < baselineSum {
		return false, "routing precision regresses on held-out fold"
	}
	return true, fmt.Sprintf("held-out paired cases=%d candidate=%.4f incumbent=%.4f", count, candidateSum/float64(count), baselineSum/float64(count))
}

func bestAccepted(arms []*evalv1.StrategyComparisonArm, active string) *evalv1.StrategyComparisonArm {
	var best *evalv1.StrategyComparisonArm
	for _, arm := range arms {
		if !arm.GetAccepted() || arm.GetStrategyName() == active {
			continue
		}
		if best == nil || arm.GetRoutableRoutingPrecision() > best.GetRoutableRoutingPrecision() {
			best = arm
		}
	}
	return best
}

func joinReasons(significance, heldout string, significanceFailed, heldoutFailed bool) string {
	parts := make([]string, 0, 2)
	if significanceFailed {
		parts = append(parts, significance)
	}
	if heldoutFailed {
		parts = append(parts, heldout)
	}
	return strings.Join(parts, "; ")
}
