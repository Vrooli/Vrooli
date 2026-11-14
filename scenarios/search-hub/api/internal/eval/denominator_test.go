package eval_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/scheduletest"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	"search-hub/internal/eval"
)

type denominatorReader struct {
	snapshot *evalv1.FixedDenominatorSnapshot
}

func (r denominatorReader) GetFixedDenominator(context.Context, string) (*evalv1.FixedDenominatorSnapshot, error) {
	return r.snapshot, nil
}

func TestFixedDenominatorSnapshotIsSealedAndIndependent(t *testing.T) {
	suite := suiteWith(&evalv1.EvalCase{CaseId: "a", Query: "query a", ExpectIds: []string{"doc-a"}})
	snapshot, err := eval.NewFixedDenominatorSnapshot(suite, "reviewer-1", "https://evidence.example/review/1", time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Equal(t, []string{"a"}, snapshot.GetCaseIds())
	_, err = eval.ValidateFixedDenominatorSnapshot(snapshot)
	require.NoError(t, err)
	snapshot.Cases[0].ExpectIds[0] = "doc-b"
	_, err = eval.ValidateFixedDenominatorSnapshot(snapshot)
	require.ErrorContains(t, err, "digest")
	_, err = eval.NewFixedDenominatorSnapshot(suite, "p", "https://evidence.example/review/1", time.Now())
	require.ErrorContains(t, err, "independent")
}

func TestRunnerUsesPersistedFixedDenominatorAndCountsOmittedTargetAsMiss(t *testing.T) {
	suite := suiteWith(&evalv1.EvalCase{CaseId: "frozen", Query: "frozen query", ExpectIds: []string{"missing-document"}})
	snapshot, err := eval.NewFixedDenominatorSnapshot(suite, "reviewer-1", "https://evidence.example/review/2", time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	client := fakeClient{byQuery: map[string][]*routingv1.SearchHit{"frozen query": {hit("other-document", 0.8)}}}
	runner := eval.NewRunner(fakeResolver{desc: &registryv1.ProviderDescriptor{ProviderId: "p"}}, client, scheduletest.New(time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)), func() string { return "fixed-run" }, eval.IdentityEvidenceReaders{Denominators: denominatorReader{snapshot: snapshot}})
	run, err := runner.Run(context.Background(), suite, "fixed", 0)
	require.NoError(t, err)
	require.Len(t, run.GetResults(), 1)
	require.Equal(t, "below_expectation", run.GetResults()[0].GetOutcome())
	require.Equal(t, "fixed-denominator expected document missing", run.GetResults()[0].GetOutcomeReason())
	require.EqualValues(t, 1, run.GetAggregate().GetRetrievalDenominator())
	require.Zero(t, run.GetAggregate().GetRetrievalHits())
	require.Equal(t, 0.0, run.GetAggregate().GetFixedDenominatorRecall())
	require.Equal(t, snapshot.GetDenominatorSha256(), run.GetComparisonIdentity().GetDenominatorSha256())
	require.Equal(t, []string{"frozen"}, run.GetComparisonIdentity().GetDenominatorCaseIds())
}

func TestFederatedRunnerRestoresOmittedFrozenCaseAndScoresMissingAsMiss(t *testing.T) {
	original := federatedSuite(&evalv1.EvalCase{CaseId: "frozen", Query: "frozen query", ExpectIds: []string{"missing-document"}})
	snapshot, err := eval.NewFixedDenominatorSnapshot(original, "reviewer-1", "https://evidence.example/review/3", time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	calls := 0
	query := fakeFederatedQuery{calls: &calls, responses: map[string]*routingv1.QueryResponse{
		"frozen query": {CorporaSearched: []string{"owner.leaf"}, Ranked: []*routingv1.SearchHit{federatedHit("owner.leaf", "other-document", 0.9)}},
	}}
	clk := scheduletest.New(time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC))
	runner := eval.NewFederatedRunnerWithRoutability(fakeResolver{desc: &registryv1.ProviderDescriptor{ProviderId: "p"}}, query, clk, func() string { return "fixed-federated" }, nil, eval.IdentityEvidenceReaders{Denominators: denominatorReader{snapshot: snapshot}})
	// Current suite drops the reviewed case. The frozen query must still be
	// exercised, and its absent expected document must count as a miss.
	run, err := runner.Run(context.Background(), federatedSuite(), "fixed", 10)
	require.NoError(t, err)
	require.Equal(t, 1, calls)
	require.Len(t, run.GetResults(), 1)
	require.Equal(t, "frozen", run.GetResults()[0].GetCaseId())
	require.Equal(t, "below_expectation", run.GetResults()[0].GetOutcome())
	require.EqualValues(t, 1, run.GetAggregate().GetRetrievalDenominator())
	require.Zero(t, run.GetAggregate().GetRetrievalHits())
	require.Equal(t, 0.0, run.GetAggregate().GetFixedDenominatorRecall())
	require.Equal(t, snapshot.GetDenominatorSha256(), run.GetComparisonIdentity().GetDenominatorSha256())
}
