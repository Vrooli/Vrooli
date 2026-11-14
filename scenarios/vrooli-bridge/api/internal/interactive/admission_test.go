package interactive

import (
	"fmt"
	"strings"
	"testing"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	interactivev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/interactive"
	"google.golang.org/protobuf/proto"
)

func TestAdmissionSeparatesNodeTrustFromDesktopSemantics(t *testing.T) {
	a := NewAdmission()
	a.SetNode("node-1", true)
	req := &interactivev1.OpenChannelRequest{NodeId: "node-1", SessionId: "s1", LeaseId: "l1", LeaseEpoch: 1, Protocol: interactivev1.ChannelProtocol_CHANNEL_PROTOCOL_WEBRTC_VP8, Role: interactivev1.ChannelRole_CHANNEL_ROLE_CONTROLLER, Surface: &commonv1.SurfaceRef{SurfaceId: "desktop", Target: &commonv1.TargetRef{ResourceId: "node-1"}}}
	opened, err := a.Open("operator-1", req)
	if err != nil {
		t.Fatal(err)
	}
	if opened.GetGrant().GetPolicyRevision() != "operator-1" {
		t.Fatal("grant did not carry bounded actor policy metadata")
	}
	if _, err := a.Signal(&interactivev1.SignalRequest{Grant: opened.GetGrant(), Kind: interactivev1.SignalKind_SIGNAL_KIND_OFFER, RequestId: "request-1", Generation: "generation-1", Payload: []byte("offer")}); err != nil {
		t.Fatal(err)
	}
	if a.RevokeNode("node-1") != 1 {
		t.Fatal("expected node revoke to remove channel")
	}
	if _, err := a.Signal(&interactivev1.SignalRequest{Grant: opened.GetGrant(), Kind: interactivev1.SignalKind_SIGNAL_KIND_ICE, RequestId: "late-ice", Generation: "generation-1", Payload: []byte("late")}); err != ErrStale {
		t.Fatalf("late signal error = %v", err)
	}
}

func TestAdmissionRejectsControllerConflictAndOversizedSignal(t *testing.T) {
	a := NewAdmission()
	a.SetNode("node-1", true)
	base := &interactivev1.OpenChannelRequest{NodeId: "node-1", SessionId: "s1", LeaseId: "l1", LeaseEpoch: 1, Protocol: interactivev1.ChannelProtocol_CHANNEL_PROTOCOL_WEBRTC_VP8, Role: interactivev1.ChannelRole_CHANNEL_ROLE_CONTROLLER, Surface: &commonv1.SurfaceRef{SurfaceId: "desktop", Target: &commonv1.TargetRef{ResourceId: "node-1"}}}
	first, err := a.Open("operator-1", base)
	if err != nil {
		t.Fatal(err)
	}
	base.SessionId = "s2"
	base.LeaseId = "l2"
	if _, err = a.Open("operator-2", base); err != ErrViewerWrite {
		t.Fatalf("controller conflict error = %v", err)
	}
	base.Takeover = true
	second, err := a.Open("operator-2", base)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.Signal(&interactivev1.SignalRequest{Grant: second.GetGrant(), Kind: interactivev1.SignalKind_SIGNAL_KIND_OFFER, RequestId: "oversized", Generation: "generation-1", Payload: []byte(strings.Repeat("x", MaxSignalBytes+1))}); err != ErrSignalLimit {
		t.Fatalf("oversized signal error = %v", err)
	}
	if a.QueueDepth(first.GetGrant().GetChannelId()) != 0 {
		t.Fatal("unexpected queue depth for old controller")
	}
}

func TestAdmissionScopesControllerExclusivityToTarget(t *testing.T) {
	a := NewAdmission()
	a.SetNode("node-1", true)
	firstRequest := &interactivev1.OpenChannelRequest{
		NodeId: "node-1", SessionId: "s1", LeaseId: "l1", LeaseEpoch: 1,
		Protocol: interactivev1.ChannelProtocol_CHANNEL_PROTOCOL_WEBRTC_VP8,
		Role:     interactivev1.ChannelRole_CHANNEL_ROLE_CONTROLLER,
		Surface:  &commonv1.SurfaceRef{SurfaceId: "desktop", Target: &commonv1.TargetRef{ResourceId: "desktop-1"}},
	}
	if _, err := a.Open("operator-1", firstRequest); err != nil {
		t.Fatal(err)
	}
	secondRequest := proto.Clone(firstRequest).(*interactivev1.OpenChannelRequest)
	secondRequest.SessionId = "s2"
	secondRequest.LeaseId = "l2"
	secondRequest.Surface.Target.ResourceId = "desktop-2"
	if _, err := a.Open("operator-2", secondRequest); err != nil {
		t.Fatalf("independent target was rejected: %v", err)
	}
}

func TestAdmissionReturnsRevokedGrantForNodeCleanup(t *testing.T) {
	a := NewAdmission()
	a.SetNode("node-1", true)
	opened, err := a.Open("operator-1", &interactivev1.OpenChannelRequest{NodeId: "node-1", SessionId: "session-1", LeaseId: "lease-1", LeaseEpoch: 3, Protocol: interactivev1.ChannelProtocol_CHANNEL_PROTOCOL_WEBRTC_VP8, Role: interactivev1.ChannelRole_CHANNEL_ROLE_CONTROLLER, Surface: &commonv1.SurfaceRef{SurfaceId: "desktop", Target: &commonv1.TargetRef{ResourceId: "node-1"}}})
	if err != nil {
		t.Fatal(err)
	}
	grants := a.RevokeNodeGrants("node-1")
	if len(grants) != 1 || grants[0].GetChannelId() != opened.GetGrant().GetChannelId() || grants[0].GetLeaseEpoch() != 3 {
		t.Fatalf("revoked grants = %+v", grants)
	}
	if _, err := a.Signal(&interactivev1.SignalRequest{Grant: opened.GetGrant(), Kind: interactivev1.SignalKind_SIGNAL_KIND_OFFER, RequestId: "late-offer", Generation: "generation-1", Payload: []byte("late")}); err != ErrStale {
		t.Fatalf("late signal error = %v", err)
	}
}

func TestAdmissionReturnsDeploymentRoutesWithoutPersistingThem(t *testing.T) {
	a := NewAdmission()
	a.SetRoutes(func() []*interactivev1.RouteCandidate {
		return []*interactivev1.RouteCandidate{{Kind: interactivev1.RouteKind_ROUTE_KIND_TURN, Url: "turn:relay.example", Username: "short-user", Credential: "short-secret", Priority: 2}}
	})
	a.SetNode("node-1", true)
	opened, err := a.Open("operator-1", &interactivev1.OpenChannelRequest{NodeId: "node-1", SessionId: "s1", LeaseId: "l1", LeaseEpoch: 1, Protocol: interactivev1.ChannelProtocol_CHANNEL_PROTOCOL_WEBRTC_VP8, Role: interactivev1.ChannelRole_CHANNEL_ROLE_VIEWER, Surface: &commonv1.SurfaceRef{SurfaceId: "desktop", Target: &commonv1.TargetRef{ResourceId: "node-1"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(opened.GetRoutes()) != 1 || opened.GetRoutes()[0].GetCredential() != "short-secret" {
		t.Fatalf("routes = %+v", opened.GetRoutes())
	}
	if len(opened.GetGrant().GetRoutes()) != 1 || opened.GetGrant().GetRoutes()[0].GetUrl() != "turn:relay.example" {
		t.Fatalf("grant routes = %+v", opened.GetGrant().GetRoutes())
	}
	inspected, err := a.Open("operator-2", &interactivev1.OpenChannelRequest{NodeId: "node-1", SessionId: "s2", LeaseId: "l2", LeaseEpoch: 1, Protocol: interactivev1.ChannelProtocol_CHANNEL_PROTOCOL_WEBRTC_VP8, Role: interactivev1.ChannelRole_CHANNEL_ROLE_VIEWER, Surface: &commonv1.SurfaceRef{SurfaceId: "desktop", Target: &commonv1.TargetRef{ResourceId: "node-1"}}})
	if err != nil || len(inspected.GetRoutes()) != 1 {
		t.Fatalf("second route response = %+v, err=%v", inspected, err)
	}
}

func TestAdmissionRejectsForgedGrantIdentityAndUnsupportedLane(t *testing.T) {
	a := NewAdmission()
	a.SetNode("node-1", true)
	opened, err := a.Open("operator-1", &interactivev1.OpenChannelRequest{NodeId: "node-1", SessionId: "s1", LeaseId: "l1", LeaseEpoch: 1, Protocol: interactivev1.ChannelProtocol_CHANNEL_PROTOCOL_WEBRTC_VP8, Role: interactivev1.ChannelRole_CHANNEL_ROLE_CONTROLLER, Surface: &commonv1.SurfaceRef{SurfaceId: "desktop", Target: &commonv1.TargetRef{ResourceId: "node-1"}}})
	if err != nil {
		t.Fatal(err)
	}
	forged := *opened.GetGrant()
	forged.NodeId = "node-2"
	if _, err := a.Signal(&interactivev1.SignalRequest{Grant: &forged, Kind: interactivev1.SignalKind_SIGNAL_KIND_OFFER, RequestId: "forged", Generation: "generation-1", Payload: []byte("offer")}); err != ErrStale {
		t.Fatalf("forged grant error = %v", err)
	}
	if _, err := a.Signal(&interactivev1.SignalRequest{Grant: opened.GetGrant(), Kind: interactivev1.SignalKind_SIGNAL_KIND_UNSPECIFIED, RequestId: "unsupported", Generation: "generation-1", Payload: []byte("payload")}); err != ErrProtocol {
		t.Fatalf("unsupported lane error = %v", err)
	}
}

func TestAdmissionRejectsMalformedOpenAndSignalRequests(t *testing.T) {
	a := NewAdmission()
	a.SetNode("node-1", true)
	base := &interactivev1.OpenChannelRequest{NodeId: "node-1", SessionId: "session-1", LeaseId: "lease-1", LeaseEpoch: 1, Protocol: interactivev1.ChannelProtocol_CHANNEL_PROTOCOL_WEBRTC_VP8, Role: interactivev1.ChannelRole_CHANNEL_ROLE_CONTROLLER, Surface: &commonv1.SurfaceRef{SurfaceId: "desktop", Target: &commonv1.TargetRef{ResourceId: "node-1"}}}
	if _, err := a.Open("operator", nil); err != ErrInvalid {
		t.Fatalf("nil open error=%v", err)
	}
	malformed := proto.Clone(base).(*interactivev1.OpenChannelRequest)
	malformed.LeaseId = ""
	if _, err := a.Open("operator", malformed); err != ErrInvalid {
		t.Fatalf("missing lease error=%v", err)
	}
	malformed = proto.Clone(base).(*interactivev1.OpenChannelRequest)
	malformed.Role = interactivev1.ChannelRole_CHANNEL_ROLE_UNSPECIFIED
	if _, err := a.Open("operator", malformed); err != ErrInvalid {
		t.Fatalf("unspecified role error=%v", err)
	}
	opened, err := a.Open("operator", base)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Signal(&interactivev1.SignalRequest{Grant: opened.GetGrant(), Kind: interactivev1.SignalKind_SIGNAL_KIND_OFFER, RequestId: "request-1", Generation: "generation-1"}); err != ErrInvalid {
		t.Fatalf("empty payload error=%v", err)
	}
	if _, err := a.Signal(&interactivev1.SignalRequest{Grant: opened.GetGrant(), Kind: interactivev1.SignalKind_SIGNAL_KIND_OFFER, Generation: "generation-1", Payload: []byte("offer")}); err != ErrInvalid {
		t.Fatalf("missing request id error=%v", err)
	}
}

func TestAdmissionBoundsPendingSignalsAndReleasesCapacity(t *testing.T) {
	a := NewAdmission()
	a.SetNode("node-1", true)
	opened, err := a.Open("operator-1", &interactivev1.OpenChannelRequest{NodeId: "node-1", SessionId: "s1", LeaseId: "l1", LeaseEpoch: 1, Protocol: interactivev1.ChannelProtocol_CHANNEL_PROTOCOL_WEBRTC_VP8, Role: interactivev1.ChannelRole_CHANNEL_ROLE_CONTROLLER, Surface: &commonv1.SurfaceRef{SurfaceId: "desktop", Target: &commonv1.TargetRef{ResourceId: "node-1"}}})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 64; i++ {
		if _, err := a.Signal(&interactivev1.SignalRequest{Grant: opened.GetGrant(), Kind: interactivev1.SignalKind_SIGNAL_KIND_OFFER, RequestId: fmt.Sprintf("request-%d", i), Generation: "generation-1", Payload: []byte("offer")}); err != nil {
			t.Fatalf("signal %d error = %v", i, err)
		}
	}
	if _, err := a.Signal(&interactivev1.SignalRequest{Grant: opened.GetGrant(), Kind: interactivev1.SignalKind_SIGNAL_KIND_OFFER, RequestId: "overflow", Generation: "generation-1", Payload: []byte("overflow")}); err != ErrSignalLimit {
		t.Fatalf("overflow error = %v", err)
	}
	for i := 0; i < 64; i++ {
		a.ReleaseSignal(opened.GetGrant().GetChannelId())
	}
	if a.QueueDepth(opened.GetGrant().GetChannelId()) != 0 {
		t.Fatalf("queue depth after release = %d", a.QueueDepth(opened.GetGrant().GetChannelId()))
	}
}
