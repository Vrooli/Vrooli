package sketch

import (
	"context"
	"log"
	"path/filepath"
	inventory "react-component-library/handlers/inventory"
	"react-component-library/internal/uimanifest"

	"connectrpc.com/connect"
	"react-component-library/internal/reconcile"

	inventoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/inventory"
)

type inventoryRPC interface {
	ScanScenario(context.Context, *connect.Request[inventoryv1.ScanScenarioRequest]) (*connect.Response[inventoryv1.ScanScenarioResponse], error)
}

type InventoryScannerAdapter struct {
	Scanner inventoryRPC
	forRoot func(string) inventoryRPC
}

func NewInventoryScannerAdapter(repoRoot string, logger *log.Logger, adoptions inventory.AdoptionsReader) InventoryScannerAdapter {
	factory := func(root string) inventoryRPC {
		return inventory.NewConnectHandler(inventory.Deps{Logger: logger, Adoptions: adoptions, ManifestLoad: uimanifest.NewFSLoader(root), ScenariosRoot: filepath.Join(root, "scenarios")})
	}
	return InventoryScannerAdapter{Scanner: factory(repoRoot), forRoot: factory}
}
func (a InventoryScannerAdapter) ForWorkspace(root string) reconcile.Scanner {
	if a.forRoot == nil {
		return nil
	}
	return InventoryScannerAdapter{Scanner: a.forRoot(root), forRoot: a.forRoot}
}

func (a InventoryScannerAdapter) ScanScenario(ctx context.Context, scenario string) ([]reconcile.ObservedFile, error) {
	response, err := a.Scanner.ScanScenario(ctx, connect.NewRequest(&inventoryv1.ScanScenarioRequest{Scenario: scenario}))
	if err != nil {
		return nil, err
	}
	byPath := map[string]*inventoryv1.SurfaceRecord{}
	for _, surface := range response.Msg.GetSurfaces() {
		byPath[surface.GetFilePath()] = surface
	}
	out := make([]reconcile.ObservedFile, 0, len(response.Msg.GetProvenance()))
	for _, provenance := range response.Msg.GetProvenance() {
		surface := byPath[provenance.GetFilePath()]
		observed := reconcile.ObservedFile{Path: provenance.GetFilePath(), ComponentName: provenance.GetComponentName(), Library: provenance.GetLibrary(), Version: provenance.GetLibraryVersion()}
		if surface != nil {
			observed.DisplayName = surface.GetDisplayName()
		}
		switch provenance.GetProvenance().String() {
		case "PROVENANCE_CUSTOM":
			observed.Provenance = reconcile.ProvenanceCustom
		case "PROVENANCE_ADOPTED_UNMODIFIED":
			observed.Provenance = reconcile.ProvenanceAdoptedUnmodified
		case "PROVENANCE_ADOPTED_MODIFIED":
			observed.Provenance = reconcile.ProvenanceAdoptedModified
		default:
			observed.Provenance = reconcile.ProvenanceUnknown
		}
		out = append(out, observed)
	}
	return out, nil
}
