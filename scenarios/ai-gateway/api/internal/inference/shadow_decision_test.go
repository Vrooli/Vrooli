package inference

import (
	"context"
	"testing"

	"ai-gateway/internal/providers"
	"github.com/stretchr/testify/require"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
)

type shadowDecisionRepository struct {
	decideCalls int
	source      string
	request     DecisionRequest
	response    *inferencev1.DecideResponse
}

func (*shadowDecisionRepository) Run(context.Context, ProviderRequest) (ProviderResult, error) {
	return ProviderResult{}, nil
}

func (r *shadowDecisionRepository) DecideBatch(ctx context.Context, items []DecisionItem, _ TypedRunner) []*inferencev1.DecideResponse {
	responses := make([]*inferencev1.DecideResponse, len(items))
	for index, item := range items {
		r.decideCalls++
		r.source = providers.Source(ctx)
		r.request = item.Request
		responses[index] = r.response
	}
	return responses
}

func TestEvaluateShadowDecisionKeepsGatewayCandidateSeparateFromTaskResult(t *testing.T) {
	baseline := []byte(`{"workflow":"existing-result"}`)
	expectedBaseline := append([]byte(nil), baseline...)
	repository := &shadowDecisionRepository{response: &inferencev1.DecideResponse{
		Answers: []*inferencev1.DecisionAnswer{{QuestionId: "next", Choice: "observe"}},
		Usage:   &inferencev1.Usage{},
	}}
	service := NewService(repository)
	ctx := providers.WithSource(context.Background(), "own-key")
	request := DecisionRequest{
		Role: "decide.fast", StateJSON: `{"selectedHistory":[{"kind":"click","target":"button#save"}],"secretRefs":["vault://refs/login"]}`,
		PrivacyProvenance: "tests.selected_human_history_fixture",
		Questions:         []*inferencev1.DecisionQuestion{{Id: "next", Type: inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_CHOICE, Options: []*inferencev1.DecisionOption{{Label: "observe"}, {Label: "click"}}}},
	}

	outcome, err := service.EvaluateShadowDecision(ctx, baseline, request)
	require.NoError(t, err)
	require.Equal(t, baseline, outcome.TaskResult, "shadow evaluation must preserve the original task result bytes")
	baseline[0] = '!'
	require.Equal(t, expectedBaseline, outcome.TaskResult, "recorded task result must not alias caller-owned bytes")
	require.Equal(t, ShadowDecisionMode, outcome.Record.Mode)
	require.Equal(t, "decide.fast", outcome.Record.CandidateRole)
	require.NotEmpty(t, outcome.Record.TaskResultSHA256)
	require.NotEmpty(t, outcome.Record.CandidateRequestSHA256)
	require.False(t, outcome.Record.TaskResultChanged)
	require.False(t, outcome.Record.BrowserActionDispatched)
	require.Equal(t, "observe", outcome.Record.CandidateResponse.GetAnswers()[0].GetChoice())
	require.Equal(t, 1, repository.decideCalls, "candidate evaluation must use the existing Gateway decision repository")
	require.Equal(t, "own-key", repository.source, "shadow evaluation must preserve the caller's selected source")
	require.Equal(t, "own-key", providers.Source(ctx), "the evaluator must not mutate the caller's funding source")
	require.Equal(t, request.Role, repository.request.Role)
}

func TestEvaluateShadowDecisionRejectsBrowserRoleAndInvalidCandidateBeforeDispatch(t *testing.T) {
	for _, test := range []struct {
		name    string
		request DecisionRequest
		wantErr error
	}{
		{name: "browser role has no verified binding", request: DecisionRequest{Role: "browser.decide", State: "candidate", PrivacyProvenance: "tests.selected_human_history_fixture"}, wantErr: ErrShadowDecisionRole},
		{name: "generic decision still validates its shape", request: DecisionRequest{Role: "decide.fast", State: "candidate", PrivacyProvenance: "tests.selected_human_history_fixture"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			baseline := []byte(`{"workflow":"unchanged"}`)
			repository := &shadowDecisionRepository{response: &inferencev1.DecideResponse{Usage: &inferencev1.Usage{}}}
			outcome, err := NewService(repository).EvaluateShadowDecision(providers.WithSource(context.Background(), "vrooli-credits"), baseline, test.request)
			if test.wantErr != nil {
				require.ErrorIs(t, err, test.wantErr)
				require.Nil(t, outcome)
				require.Zero(t, repository.decideCalls, "forbidden browser role must be denied before Gateway dispatch")
				return
			}
			require.NoError(t, err)
			require.Equal(t, baseline, outcome.TaskResult)
			require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, outcome.Record.CandidateResponse.GetError().GetCode())
			require.Zero(t, repository.decideCalls, "invalid candidate must fail validation before provider dispatch")
		})
	}
}

func TestEvaluateShadowDecisionRequiresExistingResult(t *testing.T) {
	repository := &shadowDecisionRepository{response: &inferencev1.DecideResponse{Usage: &inferencev1.Usage{}}}
	outcome, err := NewService(repository).EvaluateShadowDecision(context.Background(), nil, DecisionRequest{Role: "decide.fast"})
	require.Nil(t, outcome)
	require.ErrorContains(t, err, "existing task result")
	require.Zero(t, repository.decideCalls)
}
