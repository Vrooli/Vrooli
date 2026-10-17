package surfaces

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/targetmodel"
	attachedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/attached_devices"
	attachedconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/attached_devices/attached_devices_v1connect"
)

type attachedProviderServer struct {
	attachedconnect.UnimplementedAttachedDeviceServiceHandler
	authorization string
}

func (s attachedProviderServer) ListAttachedDevices(_ context.Context, req *connect.Request[attachedv1.ListAttachedDevicesRequest]) (*connect.Response[attachedv1.ListAttachedDevicesResponse], error) {
	if got := req.Header().Get("Authorization"); got != s.authorization {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	return connect.NewResponse(&attachedv1.ListAttachedDevicesResponse{Devices: []*attachedv1.AttachedDevice{
		{Id: "phone-1", Name: "Field phone", HostNodeId: "node-remote", Kind: "android", Transport: "adb", Transports: []string{"adb", "remote-control"}, Serial: "serial-1", TrustState: "trusted", Reachability: "reachable"},
	}}), nil
}

func TestBridgeAttachedProviderJoinsTransportsAndPreservesTopology(t *testing.T) { // [REQ:PORTAL-EVERYWHERE-CAT-04] [REQ:PORTAL-EVERYWHERE-CAT-08]
	path, handler := attachedconnect.NewAttachedDeviceServiceHandler(attachedProviderServer{authorization: "Bearer accepted"})
	server := httptest.NewServer(handler)
	defer server.Close()
	_ = path
	now := time.Date(2026, 9, 6, 5, 0, 0, 0, time.UTC)
	provider := BridgeAttachedProvider{ResolveURL: func(context.Context, string) (string, error) { return server.URL, nil }, Client: server.Client(), Now: func() time.Time { return now }}
	result, err := provider.List(WithBearerToken(context.Background(), "accepted"))
	require.NoError(t, err)
	require.Len(t, result.Surfaces, 1)
	surface := result.Surfaces[0]
	require.Equal(t, "phone-1", surface.Ref.Target.ResourceID)
	require.Equal(t, "node-remote", surface.Ref.Target.HostNodeID)
	require.Equal(t, "vrooli-bridge", surface.Ref.Target.OwnerScenario)
	require.Equal(t, "vrooli-bridge", surface.Ref.OwnerScenario)
	require.Contains(t, surface.ProtocolVersions, "vrooli.device.transport.adb.v1")
	require.Contains(t, surface.ProtocolVersions, "vrooli.device.transport.remote-control.v1")
	require.ElementsMatch(t, []string{"device.transport.adb", "device.transport.remote-control", "device.trust"}, capabilityNames(surface))
}

func capabilityNames(surface targetmodel.SurfaceDescriptor) []string {
	names := make([]string, 0, len(surface.Capabilities))
	for _, fact := range surface.Capabilities {
		names = append(names, fact.Capability)
	}
	return names
}
