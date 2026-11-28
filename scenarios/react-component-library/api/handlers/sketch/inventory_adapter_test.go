package sketch

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	provenancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/contracts/provenance"
	inventoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/inventory"
	"react-component-library/internal/reconcile"
)

type inventoryRPCFunc func(context.Context, *connect.Request[inventoryv1.ScanScenarioRequest]) (*connect.Response[inventoryv1.ScanScenarioResponse], error)

func (f inventoryRPCFunc) ScanScenario(ctx context.Context, req *connect.Request[inventoryv1.ScanScenarioRequest]) (*connect.Response[inventoryv1.ScanScenarioResponse], error) {
	return f(ctx, req)
}

func TestInventoryScannerAdapterMapsProvenanceAndSurfaceNames(t *testing.T) {
	ctx := context.Background()
	var gotScenario string
	adapter := InventoryScannerAdapter{Scanner: inventoryRPCFunc(func(_ context.Context, req *connect.Request[inventoryv1.ScanScenarioRequest]) (*connect.Response[inventoryv1.ScanScenarioResponse], error) {
		gotScenario = req.Msg.GetScenario()
		return connect.NewResponse(&inventoryv1.ScanScenarioResponse{
			Provenance: []*provenancev1.ComponentProvenance{
				{FilePath: "src/Custom.tsx", ComponentName: "Custom", Provenance: provenancev1.Provenance_PROVENANCE_CUSTOM, Library: "", LibraryVersion: ""},
				{FilePath: "src/Original.tsx", ComponentName: "Original", Provenance: provenancev1.Provenance_PROVENANCE_ADOPTED_UNMODIFIED, Library: "kit", LibraryVersion: "1.0"},
				{FilePath: "src/Changed.tsx", ComponentName: "Changed", Provenance: provenancev1.Provenance_PROVENANCE_ADOPTED_MODIFIED, Library: "kit", LibraryVersion: "2.0"},
				{FilePath: "src/Unknown.tsx", ComponentName: "Unknown", Provenance: provenancev1.Provenance_PROVENANCE_UNSPECIFIED},
			},
			Surfaces: []*inventoryv1.SurfaceRecord{{FilePath: "src/Original.tsx", DisplayName: "Original surface"}, {FilePath: "src/Unmatched.tsx", DisplayName: "ignored"}},
		}), nil
	})}

	files, err := adapter.ScanScenario(ctx, "demo")
	if err != nil {
		t.Fatal(err)
	}
	if gotScenario != "demo" {
		t.Fatalf("scanner scenario = %q, want demo", gotScenario)
	}
	want := []reconcile.ObservedFile{
		{Path: "src/Custom.tsx", ComponentName: "Custom", Provenance: reconcile.ProvenanceCustom},
		{Path: "src/Original.tsx", ComponentName: "Original", Library: "kit", Version: "1.0", DisplayName: "Original surface", Provenance: reconcile.ProvenanceAdoptedUnmodified},
		{Path: "src/Changed.tsx", ComponentName: "Changed", Library: "kit", Version: "2.0", Provenance: reconcile.ProvenanceAdoptedModified},
		{Path: "src/Unknown.tsx", ComponentName: "Unknown", Provenance: reconcile.ProvenanceUnknown},
	}
	if len(files) != len(want) {
		t.Fatalf("got %d files, want %d: %#v", len(files), len(want), files)
	}
	for i := range want {
		if files[i] != want[i] {
			t.Errorf("file %d = %#v, want %#v", i, files[i], want[i])
		}
	}
}

func TestInventoryScannerAdapterPropagatesRPCError(t *testing.T) {
	wantErr := errors.New("inventory unavailable")
	adapter := InventoryScannerAdapter{Scanner: inventoryRPCFunc(func(context.Context, *connect.Request[inventoryv1.ScanScenarioRequest]) (*connect.Response[inventoryv1.ScanScenarioResponse], error) {
		return nil, wantErr
	})}
	files, err := adapter.ScanScenario(context.Background(), "demo")
	if !errors.Is(err, wantErr) || files != nil {
		t.Fatalf("ScanScenario() = (%#v, %v), want (nil, original error)", files, err)
	}
}

func TestInventoryScannerAdapterForWorkspaceUsesWorkspaceRoot(t *testing.T) {
	var roots []string
	factory := func(root string) inventoryRPC {
		roots = append(roots, root)
		return inventoryRPCFunc(func(context.Context, *connect.Request[inventoryv1.ScanScenarioRequest]) (*connect.Response[inventoryv1.ScanScenarioResponse], error) {
			return connect.NewResponse(&inventoryv1.ScanScenarioResponse{}), nil
		})
	}
	adapter := InventoryScannerAdapter{Scanner: factory("primary"), forRoot: factory}
	workspace := adapter.ForWorkspace("workspace")
	if workspace == nil {
		t.Fatal("ForWorkspace returned nil with a configured factory")
	}
	if _, err := workspace.ScanScenario(context.Background(), "demo"); err != nil {
		t.Fatal(err)
	}
	if len(roots) != 2 || roots[0] != "primary" || roots[1] != "workspace" {
		t.Fatalf("factory roots = %#v, want primary then workspace", roots)
	}
	if (InventoryScannerAdapter{}).ForWorkspace("workspace") != nil {
		t.Fatal("ForWorkspace without a factory should be unavailable")
	}
}
