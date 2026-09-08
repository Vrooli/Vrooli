package surfaces

import (
	"context"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/targetmodel"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop/desktopv1connect"
)

type desktopCatalogFixture struct {
	desktopv1connect.UnimplementedDesktopOwnerServiceHandler
	descriptor targetmodel.SurfaceDescriptor
}

func (f desktopCatalogFixture) Describe(context.Context, *connect.Request[desktopv1.OwnerDescribeRequest]) (*connect.Response[desktopv1.OwnerDescribeResponse], error) {
	wire, err := f.descriptor.Proto()
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&desktopv1.OwnerDescribeResponse{Surface: wire}), nil
}
func TestDesktopProviderPreservesExactHostAndRejectsWrongOwner(t *testing.T) {
	for _, owner := range []string{"device-control", "web-console"} {
		t.Run(owner, func(t *testing.T) {
			now := time.Now()
			descriptor := targetmodel.SurfaceDescriptor{Ref: targetmodel.SurfaceRef{Target: targetmodel.TargetRef{OwnerScenario: "vrooli-bridge", ResourceID: "companion-host", HostNodeID: "companion-host"}, OwnerScenario: owner, SurfaceID: "desktop"}, Kind: targetmodel.SurfaceDesktop, DisplayLabel: "Desktop session 2", DesktopSessionID: "2", ProtocolVersions: []string{"vrooli.desktop.owner.v1"}, Capabilities: []targetmodel.CapabilityFact{{Capability: "desktop.observe", State: targetmodel.CapabilityUnknown, ReasonCode: "owner_admission_required", ObservedAt: now, ExpiresAt: now.Add(time.Second)}}}
			socket := filepath.Join(t.TempDir(), "owner.sock")
			listener, err := net.Listen("unix", socket)
			require.NoError(t, err)
			_, handler := desktopv1connect.NewDesktopOwnerServiceHandler(desktopCatalogFixture{descriptor: descriptor})
			server := &http.Server{Handler: handler}
			done := make(chan error, 1)
			go func() { done <- server.Serve(listener) }()
			t.Cleanup(func() { server.Close(); <-done })
			result, err := (DesktopProvider{Socket: socket}).List(context.Background())
			if owner != "device-control" {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, result.Surfaces, 1)
			require.Equal(t, descriptor.Ref, result.Surfaces[0].Ref)
			require.Equal(t, targetmodel.CapabilityUnknown, result.Surfaces[0].Capabilities[0].State)
		})
	}
	_, err := (DesktopProvider{}).List(context.Background())
	require.Error(t, err)
}
