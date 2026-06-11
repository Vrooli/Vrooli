package eval_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"

	"search-hub/internal/eval"
)

func TestValidatorClassifiesReviewedPositiveLabels(t *testing.T) {
	client := fakeClient{byQuery: map[string][]*routingv1.SearchHit{
		"live":       {hit("want-live", 0.9)},
		"hard":       {hit("x1", 0.9), hit("x2", 0.8), hit("want-hard", 0.7)},
		"stale":      {hit("x", 0.9)},
		"want-stale": {hit("other", 0.9)},
		"candidate":  {hit("want-candidate", 0.9)},
	}}
	suite := suiteWith(
		&evalv1.EvalCase{CaseId: "live", Query: "live", ExpectIds: []string{"want-live"}},
		&evalv1.EvalCase{CaseId: "hard", Query: "hard", ExpectIds: []string{"want-hard"}, ExpectWithinTopK: 2},
		&evalv1.EvalCase{CaseId: "stale", Query: "stale", ExpectIds: []string{"want-stale"}},
		&evalv1.EvalCase{CaseId: "candidate", Query: "candidate", Status: "candidate", ExpectIds: []string{"want-candidate"}},
		&evalv1.EvalCase{CaseId: "junk", Query: "junk", ExpectNoStrongHit: true, ExpectMaxScore: 0.2},
	)
	resp, err := eval.NewValidator(fakeResolver{desc: validationDescriptor()}, client).ValidateCorpus(context.Background(), suite, 5)
	require.NoError(t, err)

	got := map[string]evalv1.ReferentialOutcome{}
	for _, row := range resp.GetCases() {
		got[row.GetCaseId()] = row.GetReferential()
	}
	require.Equal(t, evalv1.ReferentialOutcome_REFERENTIAL_OUTCOME_LIVE, got["live"])
	require.Equal(t, evalv1.ReferentialOutcome_REFERENTIAL_OUTCOME_HARD, got["hard"])
	require.Equal(t, evalv1.ReferentialOutcome_REFERENTIAL_OUTCOME_STALE, got["stale"])
	require.NotContains(t, got, "candidate")
	require.NotContains(t, got, "junk")
	require.EqualValues(t, 3, resp.GetRollup().GetPositives())
	require.EqualValues(t, 1, resp.GetRollup().GetCandidate())
	require.EqualValues(t, 1, resp.GetRollup().GetLive())
	require.EqualValues(t, 1, resp.GetRollup().GetHard())
	require.EqualValues(t, 1, resp.GetRollup().GetStale())
}

func TestValidatorConfirmProbeCanRecoverHardLabel(t *testing.T) {
	client := fakeClient{byQuery: map[string][]*routingv1.SearchHit{
		"case query": {hit("x", 0.9)},
		"want":       {hit("x", 0.9), hit("want", 0.8)},
	}}
	suite := suiteWith(&evalv1.EvalCase{CaseId: "c", Query: "case query", ExpectIds: []string{"want"}})
	resp, err := eval.NewValidator(fakeResolver{desc: validationDescriptor()}, client).ValidateCorpus(context.Background(), suite, 5)
	require.NoError(t, err)
	require.Len(t, resp.GetCases(), 1)
	require.Equal(t, evalv1.ReferentialOutcome_REFERENTIAL_OUTCOME_LIVE, resp.GetCases()[0].GetReferential())
	require.Equal(t, []string{"case query", "want"}, resp.GetCases()[0].GetProbedQueries())
}

func TestValidatorProbeErrorIsInconclusive(t *testing.T) {
	client := fakeClient{errQuery: map[string]error{"boom": errors.New("provider timeout")}}
	suite := suiteWith(&evalv1.EvalCase{CaseId: "c", Query: "boom", ExpectIds: []string{"want"}})
	resp, err := eval.NewValidator(fakeResolver{desc: validationDescriptor()}, client).ValidateCorpus(context.Background(), suite, 5)
	require.NoError(t, err)
	require.Equal(t, evalv1.ReferentialOutcome_REFERENTIAL_OUTCOME_INCONCLUSIVE, resp.GetCases()[0].GetReferential())
	require.EqualValues(t, 1, resp.GetRollup().GetInconclusive())
}

func validationDescriptor() *registryv1.ProviderDescriptor {
	return &registryv1.ProviderDescriptor{ProviderId: "p"}
}
