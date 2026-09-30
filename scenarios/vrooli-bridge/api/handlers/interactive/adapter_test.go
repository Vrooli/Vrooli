package interactive

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	interactivev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/interactive"
	"vrooli-bridge/internal/auth"
	internalinteractive "vrooli-bridge/internal/interactive"
)

func TestHandlerRequiresOwnerAndTrustedNode(t *testing.T) {
	h := NewConnectHandler(Deps{Admission: internalinteractive.NewAdmission(), AuthorizeNode: func(context.Context, string) bool { return true }})
	req := connect.NewRequest(&interactivev1.OpenChannelRequest{NodeId: "n1", SessionId: "s", LeaseId: "l", LeaseEpoch: 1, Protocol: interactivev1.ChannelProtocol_CHANNEL_PROTOCOL_WEBRTC_VP8, Role: interactivev1.ChannelRole_CHANNEL_ROLE_CONTROLLER, Surface: &commonv1.SurfaceRef{SurfaceId: "desktop", Target: &commonv1.TargetRef{ResourceId: "n1"}}})
	if _, err := h.OpenChannel(context.Background(), req); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("unauthenticated code=%v err=%v", connect.CodeOf(err), err)
	}
	ctx := auth.WithIdentity(context.Background(), auth.Identity{OwnerID: "owner"})
	if _, err := h.OpenChannel(ctx, req); err != nil {
		t.Fatal(err)
	}
}
