package catalog

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
	_ "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access"
)

const GroupName = "catalog"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	minter, err := cliapp.ProtoPrimitiveBindings(core, "vrooli.token_economy.v1.access.MinterService", cliapp.ProtoBindingOptions{}, map[string]bool{
		"GetCatalogEntry": true, "ListCatalogEntries": true,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("build catalog administration bindings: %w", err)
	}
	holder, err := cliapp.ProtoPrimitiveBindings(core, "vrooli.token_economy.v1.access.HolderService", cliapp.ProtoBindingOptions{}, map[string]bool{"BrowseCatalog": true})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("build catalog browsing bindings: %w", err)
	}
	selected := map[string]cliapp.PrimitiveHandler{
		"MinterService.CreateCatalogEntry": minter["MinterService.CreateCatalogEntry"],
		"MinterService.UpdateCatalogEntry": minter["MinterService.UpdateCatalogEntry"],
		"MinterService.GetCatalogEntry":    minter["MinterService.GetCatalogEntry"],
		"MinterService.ListCatalogEntries": minter["MinterService.ListCatalogEntries"],
		"MinterService.RetireCatalogEntry": minter["MinterService.RetireCatalogEntry"],
		"HolderService.BrowseCatalog":      holder["HolderService.BrowseCatalog"],
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, selected)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("load catalog manifest group: %w", err)
	}
	return group, nil
}
