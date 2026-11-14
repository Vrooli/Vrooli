package eval

import (
	"testing"

	"github.com/stretchr/testify/require"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

func TestStrategyComparisonRejectsMissingOrUnavailableFixedDenominatorCases(t *testing.T) {
	suite := &evalv1.EvalSuite{Cases: []*evalv1.EvalCase{
		{CaseId: "one", Query: "one", ExpectedProviderId: "p", ExpectIds: []string{"one"}},
		{CaseId: "two", Query: "two", ExpectedProviderId: "p", ExpectIds: []string{"two"}},
		{CaseId: "candidate", Query: "candidate", Status: "candidate", ExpectedProviderId: "p", ExpectIds: []string{"candidate"}},
	}}
	identity := &evalv1.EvaluationIdentity{DenominatorCaseIds: []string{"one", "two"}}
	complete := &evalv1.EvalRun{ComparisonIdentity: identity, Results: []*evalv1.CaseResult{
		{CaseId: "one", Outcome: "met"}, {CaseId: "two", Outcome: "misrouted"},
	}}
	require.Empty(t, incompleteStrategyRun(suite, complete))

	missing := &evalv1.EvalRun{ComparisonIdentity: identity, Results: []*evalv1.CaseResult{{CaseId: "one", Outcome: "met"}}}
	require.Contains(t, incompleteStrategyRun(suite, missing), "missing fixed-denominator case two")

	unavailable := &evalv1.EvalRun{ComparisonIdentity: identity, Results: []*evalv1.CaseResult{
		{CaseId: "one", Outcome: "met"}, {CaseId: "two", Outcome: "unavailable"},
	}}
	require.Contains(t, incompleteStrategyRun(suite, unavailable), "non-gradeable outcome")
}

func TestStrategyArmDenominatorDoesNotDependOnCompletedCases(t *testing.T) {
	suite := &evalv1.EvalSuite{Cases: []*evalv1.EvalCase{
		{CaseId: "one", Query: "one", ExpectedProviderId: "p", ExpectIds: []string{"one"}},
		{CaseId: "two", Query: "two", ExpectedProviderId: "q", ExpectIds: []string{"two"}},
	}}
	run := &evalv1.EvalRun{Results: []*evalv1.CaseResult{{CaseId: "one", Outcome: "met", ProviderRouted: true}}}
	arm := scoreStrategyArm(suite, run, map[string]struct{}{"p": {}})
	require.EqualValues(t, 2, arm.GetAllDenominator())
	require.EqualValues(t, 1, arm.GetAllRouted())
	require.EqualValues(t, 1, arm.GetRoutableDenominator())
	require.EqualValues(t, 1, arm.GetRoutableRouted())
}

func TestStrategySplitRequiresOwnerHeldoutTagAndExcludesGeneratedCases(t *testing.T) {
	suite := &evalv1.EvalSuite{Cases: []*evalv1.EvalCase{
		{CaseId: "tuning", ExpectedProviderId: "p", ExpectIds: []string{"one"}},
		{CaseId: "held", Tags: []string{"heldout"}, ExpectedProviderId: "p", ExpectIds: []string{"two"}},
		{CaseId: "generated", Tags: []string{"generated"}, ExpectedProviderId: "p", ExpectIds: []string{"three"}},
	}}
	tuning, heldout := strategySplit(suite)
	require.Equal(t, []string{"tuning"}, tuning)
	require.Equal(t, []string{"held"}, heldout)
	arm := scoreStrategyArm(suite, &evalv1.EvalRun{Results: []*evalv1.CaseResult{
		{CaseId: "tuning", Outcome: "met", ProviderRouted: true},
		{CaseId: "held", Outcome: "met", ProviderRouted: true},
		{CaseId: "generated", Outcome: "met", ProviderRouted: true},
	}}, map[string]struct{}{"p": {}})
	require.EqualValues(t, 2, arm.GetAllDenominator(), "unreviewed generated cases cannot improve aggregate strategy metrics")
	require.Empty(t, incompleteStrategyRun(suite, &evalv1.EvalRun{
		ComparisonIdentity: &evalv1.EvaluationIdentity{DenominatorCaseIds: []string{"held", "tuning"}},
		Results:            []*evalv1.CaseResult{{CaseId: "tuning", Outcome: "met"}, {CaseId: "held", Outcome: "below_expectation"}},
	}), "unreviewed generated cases do not enter the fixed-denominator completeness gate")
	_, untaggedHeldout := strategySplit(&evalv1.EvalSuite{Cases: []*evalv1.EvalCase{
		{CaseId: "only-untagged", ExpectedProviderId: "p", ExpectIds: []string{"four"}},
	}})
	require.Empty(t, untaggedHeldout, "hash assignment cannot establish owner-authored heldout provenance")
}

func TestStrategySignificanceUsesTuningOnlyAndExcludesGeneratedCases(t *testing.T) {
	suite := &evalv1.EvalSuite{Cases: []*evalv1.EvalCase{
		{CaseId: "t1", ExpectedProviderId: "p", ExpectIds: []string{"1"}},
		{CaseId: "t2", ExpectedProviderId: "p", ExpectIds: []string{"2"}},
		{CaseId: "t3", ExpectedProviderId: "p", ExpectIds: []string{"3"}},
		{CaseId: "h1", Tags: []string{"heldout"}, ExpectedProviderId: "p", ExpectIds: []string{"4"}},
		{CaseId: "h2", Tags: []string{"heldout"}, ExpectedProviderId: "p", ExpectIds: []string{"5"}},
		{CaseId: "h3", Tags: []string{"heldout"}, ExpectedProviderId: "p", ExpectIds: []string{"6"}},
		{CaseId: "g", Tags: []string{"generated"}, ExpectedProviderId: "p", ExpectIds: []string{"7"}},
	}}
	tuning, heldout := strategySplit(suite)
	baselineRun := &evalv1.EvalRun{Results: []*evalv1.CaseResult{
		{CaseId: "t1", Outcome: "below_expectation", ProviderRouted: false},
		{CaseId: "t2", Outcome: "below_expectation", ProviderRouted: false},
		{CaseId: "t3", Outcome: "below_expectation", ProviderRouted: false},
		{CaseId: "h1", Outcome: "below_expectation", ProviderRouted: false},
		{CaseId: "h2", Outcome: "below_expectation", ProviderRouted: false},
		{CaseId: "h3", Outcome: "below_expectation", ProviderRouted: false},
		{CaseId: "g", Outcome: "met", ProviderRouted: true},
	}}
	candidateRun := &evalv1.EvalRun{Results: []*evalv1.CaseResult{
		{CaseId: "t1", Outcome: "below_expectation", ProviderRouted: false},
		{CaseId: "t2", Outcome: "below_expectation", ProviderRouted: false},
		{CaseId: "t3", Outcome: "below_expectation", ProviderRouted: false},
		{CaseId: "h1", Outcome: "met", ProviderRouted: true},
		{CaseId: "h2", Outcome: "met", ProviderRouted: true},
		{CaseId: "h3", Outcome: "met", ProviderRouted: true},
		{CaseId: "g", Outcome: "met", ProviderRouted: true},
	}}
	baseline := routingRecallByCase(suite, baselineRun)
	candidate := routingRecallByCase(suite, candidateRun)
	mean, low := strategySignificance(candidate, baseline, tuning)
	require.Zero(t, mean, "heldout/generated-only gains must not affect the tuning significance estimate")
	require.Zero(t, low, "heldout/generated-only gains must not establish statistical significance")
	validates, _ := heldoutRoutingHolds(candidate, baseline, heldout)
	require.True(t, validates, "heldout may validate a tuning-selected arm but cannot select one")
	require.False(t, low > 0 && validates, "heldout/generated-only gains cannot satisfy both selection and validation gates")
}

func TestBestAcceptedStrategyUsesTuningScoreAfterHeldoutGate(t *testing.T) {
	arms := []*evalv1.StrategyComparisonArm{
		{StrategyName: "heldout-heavy", Accepted: true, RoutableRoutingPrecision: 1},
		{StrategyName: "tuning-winner", Accepted: true, RoutableRoutingPrecision: 0.4},
		{StrategyName: "active", Accepted: true},
	}
	winner := bestAccepted(arms, "active", map[string]float64{"heldout-heavy": 0.5, "tuning-winner": 0.8})
	require.Equal(t, "tuning-winner", winner.GetStrategyName(), "heldout summary cannot choose among accepted strategies")
}

func TestHeldoutComparisonRejectsMissingPairInsteadOfSkippingIt(t *testing.T) {
	baseline := map[string]float64{"a": 1, "b": 1, "c": 0}
	candidate := map[string]float64{"a": 1, "b": 1}
	ok, reason := heldoutRoutingHolds(candidate, baseline, []string{"a", "b", "c"})
	require.False(t, ok)
	require.Equal(t, "held-out denominator differs between arms", reason)
}
