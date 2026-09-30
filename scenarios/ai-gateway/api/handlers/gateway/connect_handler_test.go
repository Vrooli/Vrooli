package gateway_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	gatewayv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/gateway"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	handler "ai-gateway/handlers/gateway"
)

func TestValidateGatewayRequestRejectsProviderSpecificFields(t *testing.T) { // [REQ:AIGW-CONTRACT-TEXT]
	h := handler.NewConnectHandler(handler.Deps{})
	resp, err := h.ValidateGatewayRequest(context.Background(), connect.NewRequest(&gatewayv1.ValidateGatewayRequestRequest{
		Request: &sharedv1.GatewayRequest{
			Kind:         sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION,
			Role:         "chat.default",
			Profile:      sharedv1.Profile_PROFILE_LOCAL_FIRST,
			PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL,
			Metadata: map[string]string{
				"provider":   "openrouter",
				"model_slug": "provider/model",
			},
		},
	}))
	require.NoError(t, err)
	require.False(t, resp.Msg.GetValid())
	require.Len(t, resp.Msg.GetIssues(), 2)
	require.NotEmpty(t, resp.Msg.GetAcceptedProfiles())
}

func TestValidateGatewayRequestAcceptsProviderNeutralShape(t *testing.T) { // [REQ:AIGW-CONTRACT-TEXT]
	h := handler.NewConnectHandler(handler.Deps{})
	resp, err := h.ValidateGatewayRequest(context.Background(), connect.NewRequest(&gatewayv1.ValidateGatewayRequestRequest{
		Request: &sharedv1.GatewayRequest{
			Kind:         sharedv1.RequestKind_REQUEST_KIND_TEXT_EMBEDDING,
			Role:         "embedding.default",
			Profile:      sharedv1.Profile_PROFILE_LOCAL_ONLY,
			PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
		},
	}))
	require.NoError(t, err)
	require.True(t, resp.Msg.GetValid())
	require.Empty(t, resp.Msg.GetIssues())
}
