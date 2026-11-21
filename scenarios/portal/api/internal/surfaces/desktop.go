package surfaces

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/targetmodel"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop/desktopv1connect"
)

// DesktopProvider uses an explicitly configured destination-owner Unix socket.
// The socket stays server-side; neither a browser argument nor API-host identity
// chooses the desktop. Describe grants no observation or control session.
type DesktopProvider struct{ Socket string }

func (p DesktopProvider) List(ctx context.Context) (ProviderResult, error) {
	if !filepath.IsAbs(p.Socket) {
		return ProviderResult{}, fmt.Errorf("desktop owner socket is not configured")
	}
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", p.Socket)
	}}
	defer transport.CloseIdleConnections()
	client := desktopv1connect.NewDesktopOwnerServiceClient(&http.Client{Transport: transport, Timeout: 2 * time.Second}, "http://desktop-owner", connect.WithReadMaxBytes(64*1024))
	response, err := client.Describe(ctx, connect.NewRequest(&desktopv1.OwnerDescribeRequest{}))
	if err != nil {
		return ProviderResult{}, err
	}
	descriptor, err := targetmodel.SurfaceDescriptorFromProto(response.Msg.Surface)
	if err != nil || descriptor.Ref.OwnerScenario != "device-control" || descriptor.Kind != targetmodel.SurfaceDesktop || descriptor.Ref.Target.OwnerScenario != "vrooli-bridge" || descriptor.Ref.Target.HostNodeID == "" || descriptor.Ref.Target.ResourceID != descriptor.Ref.Target.HostNodeID || descriptor.DesktopSessionID == "" {
		return ProviderResult{}, fmt.Errorf("invalid desktop owner descriptor")
	}
	return ProviderResult{Surfaces: []targetmodel.SurfaceDescriptor{descriptor}}, nil
}
