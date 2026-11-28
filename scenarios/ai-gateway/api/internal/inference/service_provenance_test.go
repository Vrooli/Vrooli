package inference

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
)

func TestTypedRunRequiresValidPrivacyProvenanceBeforeProviderWork(t *testing.T) {
	service := NewService(UnavailableRepository{Reason: "must not be reached"})
	base := ProviderRequest{Source: "source", SchemaJSON: `{"type":"string"}`, Role: "classify.fast"}

	for _, tc := range []struct {
		name       string
		provenance string
	}{
		{name: "missing"},
		{name: "invalid", provenance: "user supplied private content"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := base
			request.PrivacyProvenance = tc.provenance
			response := service.Run(context.Background(), request)
			require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, response.GetError().GetCode())
			require.Equal(t, "privacy_provenance", response.GetError().GetConstruct())
			require.Zero(t, response.GetProviderCallsMade())
		})
	}

	base.PrivacyProvenance = "tests.explicit_internal_input"
	response := service.Run(context.Background(), base)
	require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNAVAILABLE, response.GetError().GetCode())
}

func TestTypedBatchesRequirePrivacyProvenance(t *testing.T) {
	service := NewService(UnavailableRepository{Reason: "must not be reached"})
	run := service.RunBatch(context.Background(), []ProviderRequest{{
		Source: "source", SchemaJSON: `{"type":"string"}`, Role: "classify.fast",
	}})
	require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, run.GetResults()[0].GetError().GetCode())
	require.Equal(t, "privacy_provenance", run.GetResults()[0].GetError().GetConstruct())

	decision := service.Decide(context.Background(), DecisionRequest{Role: "decide.test", Questions: []*inferencev1.DecisionQuestion{{
		Id: "choice", Instructions: "Choose", Type: inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_CHOICE,
		Options: []*inferencev1.DecisionOption{{Label: "a"}, {Label: "b"}},
	}}})
	require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, decision.GetError().GetCode())
	require.Equal(t, "privacy_provenance", decision.GetError().GetConstruct())

	batch := service.DecideBatch(context.Background(), DecisionRequest{Role: "decide.test"}, []DecisionRequest{{StateJSON: `{}`}})
	require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, batch.GetResults()[0].GetError().GetCode())
	require.Equal(t, "privacy_provenance", batch.GetResults()[0].GetError().GetConstruct())
}
