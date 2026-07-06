package routing_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	handler "ai-gateway/handlers/routing"
)

func TestPreviewRouteUsesGatewayValidationBeforeRouting(t *testing.T) { // [REQ:AIGW-ROUTE-PREVIEW]
	h := handler.NewConnectHandler(handler.Deps{})
	resp, err := h.PreviewRoute(context.Background(), connect.NewRequest(&routingv1.PreviewRouteRequest{
		Request: &sharedv1.GatewayRequest{
			Kind:         sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION,
			Role:         "chat.default",
			Profile:      sharedv1.Profile_PROFILE_REMOTE_ONLY,
			PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_SECRET,
		},
	}))
	require.NoError(t, err)
	require.False(t, resp.Msg.GetValid())
	require.Len(t, resp.Msg.GetIssues(), 1)
	require.Equal(t, "privacy_profile_conflict", resp.Msg.GetIssues()[0].GetCode())
	require.Empty(t, resp.Msg.GetCandidates(), "Phase 2 preview must not synthesize provider candidates")
}
