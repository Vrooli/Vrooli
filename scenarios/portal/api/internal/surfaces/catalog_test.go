package surfaces

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/targetmodel"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/shared"
	targetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/targets"
	targetsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/targets/targets_v1connect"
)

type providerFunc func(context.Context) (ProviderResult, error)

func (f providerFunc) List(ctx context.Context) (ProviderResult, error) { return f(ctx) }
func offered(owner, id string, now time.Time) targetmodel.SurfaceDescriptor {
	return targetmodel.SurfaceDescriptor{
		Ref:  targetmodel.SurfaceRef{Target: targetmodel.TargetRef{OwnerScenario: "vrooli-bridge", ResourceID: id, HostNodeID: id}, OwnerScenario: owner, SurfaceID: id},
		Kind: targetmodel.SurfaceTerminal, DisplayLabel: "Office PC", ProtocolVersions: []string{"vrooli.terminal.v1"},
		Capabilities: []targetmodel.CapabilityFact{{Capability: "terminal.session", State: targetmodel.CapabilityReady, EvidenceID: "probe", ObservedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Minute)}},
	}
}
func TestCatalogPartialInventoryAndAmbiguousNames(t *testing.T) { // [REQ:PORTAL-EVERYWHERE-CAT-02] [REQ:PORTAL-EVERYWHERE-CAT-03]
	now := time.Now()
	sources := []Source{
		{Owner: "web-console", Provider: providerFunc(func(context.Context) (ProviderResult, error) {
			return ProviderResult{Surfaces: []targetmodel.SurfaceDescriptor{offered("web-console", "node-1", now), offered("web-console", "node-2", now)}}, nil
		})},
		{Owner: "device-control", Provider: providerFunc(func(context.Context) (ProviderResult, error) {
			return ProviderResult{}, errors.New("https://private:credential@example.invalid")
		})},
	}
	catalog, err := NewCatalog(sources, time.Second, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	snapshot := catalog.List(context.Background())
	if len(snapshot.Surfaces) != 2 || snapshot.Sources[1].State != "unavailable" {
		t.Fatalf("healthy inventory lost: %+v", snapshot)
	}
	matches, err := snapshot.Resolve("Office PC", nil)
	if err != nil || len(matches) != 2 {
		t.Fatalf("ambiguous selection guessed: %+v %v", matches, err)
	}
	exact := matches[1].Ref
	matches, err = snapshot.Resolve("", &exact)
	if err != nil || len(matches) != 1 || matches[0].Ref.Target.ResourceID != "node-2" {
		t.Fatal("exact selection lost identity")
	}
	for _, source := range snapshot.Sources {
		if strings.Contains(source.ReasonCode, "credential") {
			t.Fatal("provider error leaked")
		}
	}
}
func TestCatalogRejectsForgedOwnerAndExpiresReadiness(t *testing.T) { // [REQ:PORTAL-EVERYWHERE-CAT-05] [REQ:PORTAL-EVERYWHERE-CAT-07]
	now := time.Now()
	stale := offered("web-console", "node-1", now)
	stale.Capabilities[0].ExpiresAt = now
	catalog, _ := NewCatalog([]Source{
		{Owner: "web-console", Provider: providerFunc(func(context.Context) (ProviderResult, error) {
			return ProviderResult{Surfaces: []targetmodel.SurfaceDescriptor{stale}}, nil
		})},
		{Owner: "device-control", Provider: providerFunc(func(context.Context) (ProviderResult, error) {
			return ProviderResult{Surfaces: []targetmodel.SurfaceDescriptor{offered("web-console", "node-2", now)}}, nil
		})},
	}, time.Second, func() time.Time { return now })
	snapshot := catalog.List(context.Background())
	if len(snapshot.Surfaces) != 1 || snapshot.Sources[1].State != "invalid" {
		t.Fatalf("forged owner accepted: %+v", snapshot)
	}
	if snapshot.Surfaces[0].Capabilities[0].State != targetmodel.CapabilityUnknown {
		t.Fatal("expired capability stayed ready")
	}
	if stale.Capabilities[0].State != targetmodel.CapabilityReady {
		t.Fatal("provider cache mutated")
	}
}
func TestCatalogBoundsUncooperativeProviders(t *testing.T) { // [REQ:PORTAL-EVERYWHERE-OPT-08] OPT-08
	release := make(chan struct{})
	defer close(release)
	catalog, _ := NewCatalog([]Source{{Owner: "web-console", Provider: providerFunc(func(context.Context) (ProviderResult, error) { <-release; return ProviderResult{}, nil })}}, time.Millisecond, time.Now)
	snapshot := catalog.List(context.Background())
	if snapshot.Sources[0].ReasonCode != "source_deadline" {
		t.Fatalf("deadline not explicit: %+v", snapshot)
	}
	// A timed-out worker retains its slot until it actually returns. Further
	// callers cannot create more than the catalog's fixed worker capacity.
	for i := 0; i < 15; i++ {
		catalog.List(context.Background())
	}
	if got := catalog.List(context.Background()).Sources[0].ReasonCode; got != "source_busy" {
		t.Fatalf("worker bound not enforced: %s", got)
	}
}

type terminalServer struct {
	targetsconnect.UnimplementedTargetCatalogServiceHandler
}

func (terminalServer) List(context.Context, *connect.Request[targetsv1.ListRequest]) (*connect.Response[targetsv1.ListResponse], error) {
	return connect.NewResponse(&targetsv1.ListResponse{State: targetsv1.CatalogState_CATALOG_STATE_REGISTRY_ERROR, Targets: []*sharedv1.Target{{Id: "local", Kind: "local", Label: "This machine", Dispatchable: true}, {Id: "remote", Kind: "bridge-node", NodeId: "node-2", Label: "Office PC", Dispatchable: true}}}), nil
}
func TestWebConsoleAdapterUsesTypedOwnerAndDoesNotInventLocalIdentity(t *testing.T) { // [REQ:PORTAL-EVERYWHERE-CAT-01] [REQ:PORTAL-EVERYWHERE-CAT-06]
	_, handler := targetsconnect.NewTargetCatalogServiceHandler(terminalServer{})
	server := httptest.NewServer(handler)
	defer server.Close()
	provider := WebConsoleProvider{ResolveURL: func(_ context.Context, owner string) (string, error) {
		if owner != "web-console" {
			t.Fatal("wrong discovery owner")
		}
		return server.URL, nil
	}, Client: server.Client(), Now: time.Now}
	result, err := provider.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !result.Partial || len(result.Surfaces) != 2 {
		t.Fatalf("partial terminal catalog lost: %+v", result)
	}
	if result.Surfaces[0].DisplayLabel == "This machine" {
		t.Fatal("API host mislabeled as companion host")
	}
	for _, surface := range result.Surfaces {
		if surface.Kind != targetmodel.SurfaceTerminal || surface.Capabilities[0].State != targetmodel.CapabilityUnknown {
			t.Fatal("terminal inventory invented desktop readiness or grant")
		}
	}
	if result.Surfaces[1].Ref.Target.ResourceID != "node-2" {
		t.Fatal("remote node identity lost")
	}
}
