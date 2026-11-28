package search

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

type inferenceGatewayFunc func(context.Context, *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error)

func (f inferenceGatewayFunc) Run(ctx context.Context, request *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
	return f(ctx, request)
}

func TestGatewayAnswererUsesLocalOnlyAndSchemasCitationsToEvidence(t *testing.T) {
	evidence := []Evidence{{ID: "asset:button", Kind: "asset", Title: "Button", Body: "Primary action control."}}
	answerer := gatewayAnswerer{resolve: func(context.Context) (inferenceGateway, error) {
		return inferenceGatewayFunc(func(_ context.Context, request *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
			require.Equal(t, "extract.structured", request.Msg.GetRole())
			require.Equal(t, sharedv1.Profile_PROFILE_LOCAL_ONLY, request.Msg.GetProfile())
			require.Equal(t, sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL, request.Msg.GetPrivacyClass())
			require.Equal(t, "react_component_library.indexed_workbench_search", request.Msg.GetPrivacyProvenance())
			require.Contains(t, request.Msg.GetInstruction(), "What is this button?")
			require.Contains(t, request.Msg.GetInstruction(), "Always cite at least one supplied result_id")
			require.Contains(t, request.Msg.GetInstruction(), "A citation is a pointer, not proof")
			require.Contains(t, request.Msg.GetInstruction(), "Write the answer for a person in ordinary, concise language")
			require.Contains(t, request.Msg.GetInstruction(), "Never put a raw source ID")
			require.Contains(t, request.Msg.GetInstruction(), "not ‘workspace:ws_curated_marlow’")
			require.Contains(t, request.Msg.GetInstruction(), "Do not infer typical features, accessibility, responsiveness")
			require.Contains(t, request.Msg.GetInstruction(), "say it is insufficient")
			require.Contains(t, request.Msg.GetInstruction(), "Copy citation IDs exactly from this allowed list")
			require.Contains(t, request.Msg.GetInstruction(), "allowed list and use no others: asset:button")
			require.Equal(t, int32(256), request.Msg.GetMaxOutputTokens())
			require.Contains(t, request.Msg.GetInstruction(), "Distinguish an imported HTML source region")
			require.Contains(t, request.Msg.GetInstruction(), "prefer semantically matching RCL assets")
			require.Contains(t, request.Msg.GetSchemaJson(), "asset:button")
			require.NotContains(t, request.Msg.GetSchemaJson(), "not-supplied")
			var got []Evidence
			require.NoError(t, json.Unmarshal([]byte(request.Msg.GetSource()), &got))
			require.Equal(t, evidence, got)
			return connect.NewResponse(&inferencev1.RunResponse{ValueJson: `{"answer":"A primary action control.","citations":[{"result_id":"asset:button"}]}`, Validated: true}), nil
		}), nil
	}}

	answer, err := answerer.Answer(context.Background(), "What is this button?", evidence)
	require.NoError(t, err)
	require.Equal(t, "A primary action control.", answer.Text)
	require.Equal(t, []string{"asset:button"}, answer.Citations)
}

func TestGatewayAnswererDeduplicatesCitationIDsWithoutChangingOrder(t *testing.T) {
	answerer := gatewayAnswerer{resolve: func(context.Context) (inferenceGateway, error) {
		return inferenceGatewayFunc(func(context.Context, *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
			return connect.NewResponse(&inferencev1.RunResponse{
				ValueJson: `{"answer":"Supported by both results.","citations":[{"result_id":"asset:first"},{"result_id":"asset:second"},{"result_id":"asset:first"}]}`,
				Validated: true,
			}), nil
		}), nil
	}}

	answer, err := answerer.Answer(context.Background(), "question", []Evidence{{ID: "asset:first"}, {ID: "asset:second"}})
	require.NoError(t, err)
	require.Equal(t, []string{"asset:first", "asset:second"}, answer.Citations)
}

func TestGatewayAnswererPreservesUnknownCitationForDomainValidation(t *testing.T) {
	answerer := gatewayAnswerer{resolve: func(context.Context) (inferenceGateway, error) {
		return inferenceGatewayFunc(func(context.Context, *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
			return connect.NewResponse(&inferencev1.RunResponse{
				ValueJson: `{"answer":"Unsupported citation.","citations":[{"result_id":"invented:result"}]}`,
				Validated: true,
			}), nil
		}), nil
	}}

	answer, err := answerer.Answer(context.Background(), "question", []Evidence{{ID: "asset:first"}})
	require.NoError(t, err)
	require.Equal(t, []string{"invented:result"}, answer.Citations)
}

func TestAskSchemaBoundsCitationIDsAndEvidence(t *testing.T) {
	schema, err := askSchema([]string{"known:id"})
	require.NoError(t, err)
	require.True(t, strings.Contains(schema, "known:id"))
	require.True(t, strings.Contains(schema, "additionalProperties"))
	require.False(t, strings.Contains(schema, "invented:id"))
	var shape map[string]any
	require.NoError(t, json.Unmarshal([]byte(schema), &shape))
	citations := shape["properties"].(map[string]any)["citations"].(map[string]any)
	require.Equal(t, float64(1), citations["minItems"], "answers must cite at least one supplied result")
	require.Equal(t, float64(maxAskEvidence), citations["maxItems"])
	require.NotContains(t, citations, "uniqueItems", "AI Gateway rejects this schema keyword; uniqueness is validated after decoding")
	require.Equal(t, "é", boundedEvidence("éx", 2))
}

func TestGatewayAnswererClassifiesProviderFailures(t *testing.T) {
	want := errors.New("gateway offline")
	answerer := gatewayAnswerer{resolve: func(context.Context) (inferenceGateway, error) {
		return nil, want
	}}
	_, err := answerer.Answer(context.Background(), "question", []Evidence{{ID: "asset:button"}})
	require.ErrorIs(t, err, ErrAskProviderUnavailable)
	require.ErrorIs(t, err, want)
	require.ErrorContains(t, err, "resolve AI Gateway")

	answerer = gatewayAnswerer{resolve: func(context.Context) (inferenceGateway, error) {
		return inferenceGatewayFunc(func(context.Context, *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
			return nil, fmt.Errorf("transport failed")
		}), nil
	}}
	_, err = answerer.Answer(context.Background(), "question", []Evidence{{ID: "asset:button"}})
	require.ErrorIs(t, err, ErrAskProviderUnavailable)
	require.ErrorContains(t, err, "call AI Gateway")
}
