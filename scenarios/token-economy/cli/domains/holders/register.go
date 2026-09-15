package holders

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
	_ "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access"
)

const GroupName = "holders"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	minter, err := cliapp.ProtoPrimitiveBindings(core, "vrooli.token_economy.v1.access.MinterService", cliapp.ProtoBindingOptions{}, map[string]bool{
		"GetHolder": true, "ListHolders": true,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("build holder administration bindings: %w", err)
	}
	holder, err := cliapp.ProtoPrimitiveBindings(core, "vrooli.token_economy.v1.access.HolderService", cliapp.ProtoBindingOptions{}, map[string]bool{"ViewEconomy": true})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("build holder view bindings: %w", err)
	}
	selected := map[string]cliapp.PrimitiveHandler{
		"MinterService.CreateHolder": minter["MinterService.CreateHolder"],
		"MinterService.GetHolder":    minter["MinterService.GetHolder"],
		"MinterService.ListHolders":  minter["MinterService.ListHolders"],
		"HolderService.ViewEconomy":  holder["HolderService.ViewEconomy"],
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, selected)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("load holders manifest group: %w", err)
	}
	return group, nil
}
