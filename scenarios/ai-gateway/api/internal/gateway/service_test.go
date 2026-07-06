package gateway_test

import (
	"testing"

	"ai-gateway/internal/gateway"

	"github.com/stretchr/testify/require"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

func TestValidateAcceptsProviderNeutralRequest(t *testing.T) { // [REQ:AIGW-CONTRACT-TEXT]
	svc := gateway.New()
	issues := svc.Validate(&sharedv1.GatewayRequest{
		Kind:         sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION,
		Role:         "chat.default",
		Profile:      sharedv1.Profile_PROFILE_LOCAL_FIRST,
		PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL,
		TimeoutMs:    5000,
		Metadata: map[string]string{
			"trace_id": "req-1",
		},
	})
	require.Empty(t, issues)
}

func TestValidateRejectsProviderSpecificMetadata(t *testing.T) { // [REQ:AIGW-CONTRACT-TEXT]
	svc := gateway.New()
	issues := svc.Validate(validRequest(map[string]string{
		"provider":             "openrouter",
		"model_slug":           "provider/model",
		"embedding-dimensions": "1536",
		"trace_id":             "req-1",
	}))
	require.ElementsMatch(t, []string{
		"metadata.provider",
		"metadata.model_slug",
		"metadata.embedding-dimensions",
	}, fields(issues))
	for _, got := range issues {
		require.Equal(t, "provider_specific_field", got.GetCode())
	}
}

func TestValidateRejectsSecretRemoteProfile(t *testing.T) { // [REQ:AIGW-POLICY-CONSTRAINTS]
	svc := gateway.New()
	issues := svc.Validate(&sharedv1.GatewayRequest{
		Kind:         sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION,
		Role:         "chat.default",
		Profile:      sharedv1.Profile_PROFILE_REMOTE_ONLY,
		PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_SECRET,
	})
	require.Len(t, issues, 1)
	require.Equal(t, "profile", issues[0].GetField())
	require.Equal(t, "privacy_profile_conflict", issues[0].GetCode())
}

func TestValidateRejectsMissingRequiredFieldsAndNegativeBounds(t *testing.T) {
	svc := gateway.New()
	issues := svc.Validate(&sharedv1.GatewayRequest{
		TimeoutMs:       -1,
		MaxCostUsd:      -0.01,
		MaxOutputTokens: -1,
	})
	require.ElementsMatch(t, []string{
		"kind",
		"role",
		"profile",
		"privacy_class",
		"timeout_ms",
		"max_cost_usd",
		"max_output_tokens",
	}, fields(issues))
}

func validRequest(metadata map[string]string) *sharedv1.GatewayRequest {
	return &sharedv1.GatewayRequest{
		Kind:         sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION,
		Role:         "chat.default",
		Profile:      sharedv1.Profile_PROFILE_LOCAL_FIRST,
		PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL,
		Metadata:     metadata,
	}
}

func fields(issues []*sharedv1.ValidationIssue) []string {
	out := make([]string, 0, len(issues))
	for _, got := range issues {
		out = append(out, got.GetField())
	}
	return out
}
