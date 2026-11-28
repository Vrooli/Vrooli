package interactive

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	interactivev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/interactive"
	interactiveconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/interactive/interactive_v1connect"
	"vrooli-bridge/internal/auth"
	internalinteractive "vrooli-bridge/internal/interactive"
)

// TestSoftwareSimulationBridgeOwnerProcess is a test-only fixture endpoint for
// the cross-module MB4 integration runner. It starts the real Bridge Connect
// handler in a separate go test process; actor and node trust are explicitly
// synthetic. The parent supplies ready/shutdown paths and owns process cleanup.
func TestSoftwareSimulationBridgeOwnerProcess(t *testing.T) {
	readyFile := os.Getenv("MB4_BRIDGE_READY_FILE")
	shutdownFile := os.Getenv("MB4_BRIDGE_SHUTDOWN_FILE")
	if readyFile == "" || shutdownFile == "" {
		return
	}
	t.Log("SOFTWARE SIMULATION ONLY: Bridge owner in a separate test process; synthetic actor and trusted node")
	h := NewConnectHandler(Deps{Admission: internalinteractive.NewAdmission(), AuthorizeNode: func(_ context.Context, nodeID string) bool { return nodeID == "software-simulation-node" }})
	path, service := interactiveconnect.NewInteractiveDesktopServiceHandler(h)
	router := mux.NewRouter()
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: service})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := auth.WithIdentity(r.Context(), auth.Identity{OwnerID: "software-simulation-operator"})
		router.ServeHTTP(w, r.WithContext(ctx))
	}))
	defer server.Close()
	if err := os.WriteFile(readyFile, []byte(server.URL), 0600); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(shutdownFile); err == nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("timed out waiting for parent integration runner cleanup")
}

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

func TestSoftwareSimulationBridgeHTTPAdmissionLifecycle(t *testing.T) {
	t.Log("SOFTWARE SIMULATION ONLY: in-process Bridge Connect owner and synthetic node admission")
	trusted := false
	h := NewConnectHandler(Deps{Admission: internalinteractive.NewAdmission(), AuthorizeNode: func(_ context.Context, nodeID string) bool { return trusted && nodeID == "software-simulation-node" }})
	path, service := interactiveconnect.NewInteractiveDesktopServiceHandler(h)
	router := mux.NewRouter()
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: service})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := auth.WithIdentity(r.Context(), auth.Identity{OwnerID: "software-simulation-operator"})
		router.ServeHTTP(w, r.WithContext(ctx))
	}))
	t.Cleanup(server.Close)
	client := interactiveconnect.NewInteractiveDesktopServiceClient(server.Client(), server.URL)
	surface := &commonv1.SurfaceRef{SurfaceId: "desktop", OwnerScenario: "device-control", Target: &commonv1.TargetRef{ResourceId: "software-simulation-node", HostNodeId: "software-simulation-node"}}
	openRequest := &interactivev1.OpenChannelRequest{
		NodeId: "software-simulation-node", SessionId: "software-simulation-session", LeaseId: "software-simulation-lease", LeaseEpoch: 7,
		Protocol: interactivev1.ChannelProtocol_CHANNEL_PROTOCOL_WEBRTC_VP8,
		Role:     interactivev1.ChannelRole_CHANNEL_ROLE_CONTROLLER,
		Surface:  surface,
	}
	if _, err := client.OpenChannel(context.Background(), connect.NewRequest(openRequest)); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("untrusted-node admission code=%v err=%v", connect.CodeOf(err), err)
	}
	trusted = true
	opened, err := client.OpenChannel(context.Background(), connect.NewRequest(openRequest))
	if err != nil || opened.Msg.GetGrant().GetChannelId() == "" || opened.Msg.GetGrant().GetLeaseEpoch() != 7 {
		t.Fatalf("trusted open: response=%v err=%v", opened, err)
	}
	signal := &interactivev1.SignalRequest{Grant: opened.Msg.GetGrant(), Kind: interactivev1.SignalKind_SIGNAL_KIND_OFFER, RequestId: "simulation-offer", Generation: "simulation-generation", Payload: []byte("synthetic-offer")}
	if _, err = client.Signal(context.Background(), connect.NewRequest(signal)); err != nil {
		t.Fatalf("synthetic signal: %v", err)
	}
	closed, err := client.CloseChannel(context.Background(), connect.NewRequest(&interactivev1.CloseChannelRequest{Grant: opened.Msg.GetGrant()}))
	if err != nil || !closed.Msg.GetClosed() {
		t.Fatalf("close: response=%v err=%v", closed, err)
	}
	if _, err = client.Signal(context.Background(), connect.NewRequest(signal)); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("signal after close code=%v err=%v", connect.CodeOf(err), err)
	}
	openRequest.SessionId = "software-simulation-session-reconnected"
	openRequest.LeaseId = "software-simulation-lease-reconnected"
	openRequest.LeaseEpoch = 8
	reconnected, err := client.OpenChannel(context.Background(), connect.NewRequest(openRequest))
	if err != nil || reconnected.Msg.GetGrant().GetChannelId() == opened.Msg.GetGrant().GetChannelId() {
		t.Fatalf("fresh admission after close: response=%v err=%v", reconnected, err)
	}
	if _, err = client.CloseChannel(context.Background(), connect.NewRequest(&interactivev1.CloseChannelRequest{Grant: reconnected.Msg.GetGrant()})); err != nil {
		t.Fatalf("reconnected close: %v", err)
	}
}
