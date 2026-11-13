package eval

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"connectrpc.com/connect"
	aisearch "github.com/vrooli/ai-go/search"

	internaleval "search-hub/internal/eval"
	internalregistry "search-hub/internal/registry"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

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
	exclusions := make(map[string]string)
	if h.deps.Routability != nil {
		status, statusErr := h.deps.Routability.Status(ctx)
		if statusErr != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.New("internal error"))
		}
		for _, provider := range status.GetProviders() {
			if provider == nil || provider.GetProviderId() == "" {
				continue
			}
			if provider.GetAutomaticEligible() {
				routable[provider.GetProviderId()] = struct{}{}
				continue
			}
			reason := strings.TrimSpace(provider.GetAutomaticExclusionReason())
			if reason == "" {
				reason = "automatic eligibility policy withheld the provider"
			}
			exclusions[provider.GetProviderId()] = reason
		}
	} else {
		activeProviders, listErr := h.deps.Registry.List(ctx, internalregistry.ListFilter{State: int32(registryv1.ProviderState_PROVIDER_STATE_ACTIVE)})
		if listErr != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.New("internal error"))
		}
		for _, provider := range activeProviders {
			if provider != nil && provider.GetEndpoint() != nil && provider.GetLifecycle() == registryv1.Lifecycle_LIFECYCLE_PRODUCTION {
				routable[provider.GetProviderId()] = struct{}{}
			} else if provider != nil && provider.GetProviderId() != "" {
				exclusions[provider.GetProviderId()] = "provider is not production-eligible"
			}
		}
	}

	runs := make(map[string]*evalv1.EvalRun, len(names))
	for _, name := range names {
		tag := "strategy-compare:" + name
		var run *evalv1.EvalRun
		var runErr error
		if snapshotRunner, ok := strategyRunner.(interface {
			RunWithStrategyAndExclusions(context.Context, *evalv1.EvalSuite, string, int32, string, map[string]string) (*evalv1.EvalRun, error)
		}); ok {
			run, runErr = snapshotRunner.RunWithStrategyAndExclusions(ctx, suite, tag, req.Msg.GetLimit(), name, exclusions)
		} else {
			run, runErr = strategyRunner.RunWithStrategy(ctx, suite, tag, req.Msg.GetLimit(), name)
		}
		if runErr != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("strategy %q: %w", name, runErr))
		}
		if runErr = h.deps.Store.AppendRun(ctx, run); runErr != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.New("internal error"))
		}
		runs[name] = run
	}

	result := &evalv1.CompareStrategiesResponse{SuiteId: suite.GetSuiteId(), ActiveStrategy: active}
	tuningScore := make(map[string]float64, len(names))
	for _, name := range names {
		arm := scoreStrategyArm(suite, runs[name], routable)
		result.Arms = append(result.Arms, arm)
	}
	baseline := runs[active]
	baselineRecall := routingRecallByCase(suite, baseline)
	tuning, heldout := strategySplit(suite)
	for _, arm := range result.Arms {
		candidate := runs[arm.GetStrategyName()]
		if arm.GetStrategyName() == active {
			arm.Accepted = true
			arm.HeldoutHolds = true
			arm.SignificanceReason = "incumbent"
			continue
		}
		if reason := incompleteStrategyRun(suite, baseline); reason != "" {
			arm.HeldoutReason = "incumbent comparison incomplete: " + reason
			arm.RejectionReason = arm.HeldoutReason
			continue
		}
		if reason := incompleteStrategyRun(suite, candidate); reason != "" {
			arm.HeldoutReason = "candidate comparison incomplete: " + reason
			arm.RejectionReason = arm.HeldoutReason
			continue
		}
		if identityErr := internaleval.ValidateComparableRuns(baseline.GetComparisonIdentity(), candidate.GetComparisonIdentity()); identityErr != nil {
			arm.HeldoutReason = "unbound or mismatched immutable comparison identity: " + identityErr.Error()
			arm.RejectionReason = arm.HeldoutReason
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
		tuningScore[arm.GetStrategyName()] = meanRecallForCaseIDs(candidateRecall, tuning)
		mean, low := strategySignificance(candidateRecall, baselineRecall, tuning)
		arm.Significant = low > 0
		arm.SignificanceReason = fmt.Sprintf("paired routing delta mean=%.4f, 95%% CI lower=%.4f", mean, low)
		arm.HeldoutHolds, arm.HeldoutReason = heldoutRoutingHolds(candidateRecall, baselineRecall, heldout)
		arm.Accepted = arm.Significant && arm.HeldoutHolds
		if !arm.Accepted {
			arm.RejectionReason = joinReasons(arm.SignificanceReason, arm.HeldoutReason, !arm.Significant, !arm.HeldoutHolds)
		}
	}

	if req.Msg.GetApply() {
		winner := bestAccepted(result.Arms, active, tuningScore)
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
		if !isStrategyPositiveCase(c) {
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
		cr := resultByCase(run, c.GetCaseId())
		if cr != nil && cr.GetProviderRouted() {
			arm.AllRouted++
			if isRoutable {
				arm.RoutableRouted++
			}
		}
		if cr != nil && strategyGradeable(cr.GetOutcome()) && cr.GetExpectedRank() == 1 {
			arm.Top1++
		}
		if cr != nil && strategyGradeable(cr.GetOutcome()) && cr.GetExpectedRank() > 0 && cr.GetExpectedRank() <= 3 {
			arm.Top3++
		}
		if cr != nil && strategyGradeable(cr.GetOutcome()) && cr.GetExpectedRank() > 0 && cr.GetExpectedRank() <= 6 {
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
	case "met", "below_expectation", "above_expectation", "unexpected_hit", "answered_by_sibling", "misrouted", "no_result", "thin_margin":
		return true
	default:
		return false
	}
}

func routingRecallByCase(suite *evalv1.EvalSuite, run *evalv1.EvalRun) map[string]float64 {
	result := make(map[string]float64)
	for _, c := range suite.GetCases() {
		if !isStrategyPositiveCase(c) {
			continue
		}
		cr := resultByCase(run, c.GetCaseId())
		if cr != nil && strategyGradeable(cr.GetOutcome()) && cr.GetProviderRouted() {
			result[c.GetCaseId()] = 1
		} else {
			result[c.GetCaseId()] = 0
		}
	}
	return result
}

func isStrategyPositiveCase(c *evalv1.EvalCase) bool {
	if c == nil || c.GetStatus() == "candidate" || c.GetExpectedProviderId() == "" || len(c.GetExpectIds()) == 0 {
		return false
	}
	return !hasStrategyTag(c.GetTags(), aisearch.TagGenerated) || hasStrategyTag(c.GetTags(), aisearch.TagHeldout)
}

func strategySplit(suite *evalv1.EvalSuite) (tuning, heldout []string) {
	ids := make([]string, 0, len(suite.GetCases()))
	generated := make(map[string]bool)
	heldoutCases := make(map[string]bool)
	for _, c := range suite.GetCases() {
		if isStrategyPositiveCase(c) {
			ids = append(ids, c.GetCaseId())
			if hasStrategyTag(c.GetTags(), "generated") {
				generated[c.GetCaseId()] = true
			}
			if hasStrategyTag(c.GetTags(), aisearch.TagHeldout) {
				heldoutCases[c.GetCaseId()] = true
			}
		}
	}
	tuning, heldout, _ = aisearch.FixedEvaluationSplit(ids, heldoutCases, generated)
	return tuning, heldout
}

func hasStrategyTag(tags []string, want string) bool {
	for _, tag := range tags {
		if tag == want {
			return true
		}
	}
	return false
}

func recallForCaseIDs(recall map[string]float64, ids []string) map[string]float64 {
	selected := make(map[string]float64, len(ids))
	for _, id := range ids {
		if value, ok := recall[id]; ok {
			selected[id] = value
		}
	}
	return selected
}

func strategySignificance(candidate, baseline map[string]float64, tuning []string) (mean, low float64) {
	return pairedRoutingCI(recallForCaseIDs(candidate, tuning), recallForCaseIDs(baseline, tuning))
}

func meanRecallForCaseIDs(recall map[string]float64, ids []string) float64 {
	if len(ids) == 0 {
		return 0
	}
	selected := recallForCaseIDs(recall, ids)
	if len(selected) != len(ids) {
		return 0
	}
	var total float64
	for _, value := range selected {
		total += value
	}
	return total / float64(len(selected))
}

func pairedRoutingCI(candidate, baseline map[string]float64) (mean, low float64) {
	if len(candidate) < 3 || len(candidate) != len(baseline) {
		return 0, 0
	}
	ids := make([]string, 0, len(baseline))
	for id := range baseline {
		if _, ok := candidate[id]; !ok {
			return 0, 0
		}
		ids = append(ids, id)
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
			return false, "held-out denominator differs between arms"
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

func incompleteStrategyRun(suite *evalv1.EvalSuite, run *evalv1.EvalRun) string {
	if run == nil {
		return "run is missing"
	}
	if run.GetDegraded() {
		return "run is degraded: " + run.GetDegradedReason()
	}
	want := strategyPositiveCaseIDs(suite)
	seen := make(map[string]bool, len(want))
	for _, result := range run.GetResults() {
		if result == nil || !containsString(want, result.GetCaseId()) {
			continue
		}
		if seen[result.GetCaseId()] {
			return "duplicate result for case " + result.GetCaseId()
		}
		seen[result.GetCaseId()] = true
		if !strategyGradeable(result.GetOutcome()) {
			return fmt.Sprintf("case %s has non-gradeable outcome %q", result.GetCaseId(), result.GetOutcome())
		}
	}
	for _, id := range want {
		if !seen[id] {
			return "missing fixed-denominator case " + id
		}
	}
	if !equalCaseIDs(want, run.GetComparisonIdentity().GetDenominatorCaseIds()) {
		return "run identity does not bind the fixed positive case IDs"
	}
	return ""
}

func strategyPositiveCaseIDs(suite *evalv1.EvalSuite) []string {
	ids := make([]string, 0, len(suite.GetCases()))
	for _, c := range suite.GetCases() {
		if isStrategyPositiveCase(c) {
			ids = append(ids, c.GetCaseId())
		}
	}
	sort.Strings(ids)
	return ids
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func equalCaseIDs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func bestAccepted(arms []*evalv1.StrategyComparisonArm, active string, tuningScore map[string]float64) *evalv1.StrategyComparisonArm {
	var best *evalv1.StrategyComparisonArm
	for _, arm := range arms {
		if !arm.GetAccepted() || arm.GetStrategyName() == active {
			continue
		}
		if best == nil || tuningScore[arm.GetStrategyName()] > tuningScore[best.GetStrategyName()] ||
			(tuningScore[arm.GetStrategyName()] == tuningScore[best.GetStrategyName()] && arm.GetStrategyName() < best.GetStrategyName()) {
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
