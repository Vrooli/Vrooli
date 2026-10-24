package surfaces

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/targetmodel"
	targetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/targets"
	targetsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/targets/targets_v1connect"
)

// WebConsoleProvider consumes the existing terminal owner's safe catalog. The
// address comes from server-side discovery, never a catalog request argument.
// Dispatchability is not a desktop capability or a terminal session grant.
type WebConsoleProvider struct {
	ResolveURL func(context.Context, string) (string, error)
	Client     connect.HTTPClient
	Now        func() time.Time
}

func (p WebConsoleProvider) List(ctx context.Context) (ProviderResult, error) {
	base, err := p.ResolveURL(ctx, "web-console")
	if err != nil {
		return ProviderResult{}, err
	}
	client := targetsconnect.NewTargetCatalogServiceClient(p.Client, base, connect.WithReadMaxBytes(2<<20))
	response, err := client.List(ctx, connect.NewRequest(&targetsv1.ListRequest{}))
	if err != nil {
		return ProviderResult{}, err
	}
	if len(response.Msg.Targets) > 256 {
		return ProviderResult{}, fmt.Errorf("terminal catalog exceeds bound")
	}
	result := ProviderResult{Surfaces: []targetmodel.SurfaceDescriptor{}, Partial: response.Msg.State != targetsv1.CatalogState_CATALOG_STATE_READY}
	now := p.Now()
	for _, target := range response.Msg.Targets {
		if target == nil {
			return ProviderResult{}, fmt.Errorf("nil terminal target")
		}
		ref := targetmodel.TargetRef{OwnerScenario: "web-console", ResourceID: target.Id}
		label := target.Label
		if target.Kind == "local" || target.Id == "local" {
			label = "Web Console host"
		}
		if target.NodeId != "" {
			ref = targetmodel.TargetRef{OwnerScenario: "vrooli-bridge", ResourceID: target.NodeId, HostNodeID: target.NodeId}
		}
		result.Surfaces = append(result.Surfaces, targetmodel.SurfaceDescriptor{
			Ref: targetmodel.SurfaceRef{Target: ref, OwnerScenario: "web-console", SurfaceID: target.Id}, Kind: targetmodel.SurfaceTerminal, DisplayLabel: label,
			ProtocolVersions: []string{"vrooli.terminal.v1"}, Capabilities: []targetmodel.CapabilityFact{{Capability: "terminal.session", State: targetmodel.CapabilityUnknown, ReasonCode: "owner_admission_required", ObservedAt: now, ExpiresAt: now.Add(30 * time.Second)}},
		})
	}
	return result, nil
}

func DefaultCatalog(now func() time.Time) *Catalog {
	catalog, err := NewCatalog([]Source{
		{Owner: "web-console", Provider: WebConsoleProvider{ResolveURL: discovery.ResolveScenarioURLDefault, Client: &http.Client{Timeout: 2 * time.Second}, Now: now}},
		{Owner: "device-control", Provider: DesktopProvider{Socket: os.Getenv("PORTAL_DESKTOP_OWNER_SOCKET")}},
		{Owner: "vrooli-bridge", Provider: BridgeAttachedProvider{ResolveURL: discovery.ResolveScenarioURLDefault, Client: &http.Client{Timeout: 2 * time.Second}, Now: now}},
		{Owner: "compute-manager", Provider: ComputeProvider{ResolveURL: discovery.ResolveScenarioURLDefault, Client: &http.Client{Timeout: 2 * time.Second}, Now: now}},
	}, 3*time.Second, now)
	if err != nil {
		panic(err)
	}
	return catalog
}
