package gatewayreq

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	gatewaypb "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	documentpb "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/shared"
)

func TestConfidentialRequestFailsClosedForRemoteProfile(t *testing.T) {
	_, err := New("document-manager").For(context.Background(), DocumentClass{
		PrivacyClass: documentpb.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
	}, Options{
		Profile: gatewaypb.Profile_PROFILE_REMOTE_ONLY,
		Kind:    gatewaypb.RequestKind_REQUEST_KIND_TEXT_GENERATION,
		Role:    "summarize.default",
		Timeout: time.Second,
	})
	require.Error(t, err, "[REQ:DOC-P0-026] confidential documents must fail closed")
}

func TestBuilderEmitsPolicyCheckedRequest(t *testing.T) {
	request, err := New("document-manager").For(context.Background(), DocumentClass{
		PrivacyClass: documentpb.PrivacyClass_PRIVACY_CLASS_INTERNAL,
	}, Options{
		Profile:   gatewaypb.Profile_PROFILE_LOCAL_FIRST,
		Kind:      gatewaypb.RequestKind_REQUEST_KIND_TEXT_EMBEDDING,
		Role:      "embedding.default",
		Operation: "near-duplicate",
		Timeout:   250 * time.Millisecond,
		RequestID: "req-1",
		Metadata:  map[string]string{"document": "sha256-test"},
	})
	require.NoError(t, err)
	require.Equal(t, gatewaypb.Profile_PROFILE_LOCAL_FIRST, request.GetProfile())
	require.Equal(t, gatewaypb.PrivacyClass_PRIVACY_CLASS_INTERNAL, request.GetPrivacyClass())
	require.Equal(t, int32(250), request.GetTimeoutMs())
	require.Equal(t, "document-manager", request.GetScenario())
}
