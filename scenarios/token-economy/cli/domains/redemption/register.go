package redemption

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
	_ "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access"
)

const GroupName = "redemption"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	minter, err := cliapp.ProtoPrimitiveBindings(core, "vrooli.token_economy.v1.access.MinterService", cliapp.ProtoBindingOptions{}, map[string]bool{"ListPendingRedemptions": true})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("build redemption approval bindings: %w", err)
	}
	holder, err := cliapp.ProtoPrimitiveBindings(core, "vrooli.token_economy.v1.access.HolderService", cliapp.ProtoBindingOptions{}, nil)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("build holder redemption bindings: %w", err)
	}
	selected := map[string]cliapp.PrimitiveHandler{
		"HolderService.RequestRedemption":      holder["HolderService.RequestRedemption"],
		"MinterService.ListPendingRedemptions": minter["MinterService.ListPendingRedemptions"],
		"MinterService.ApproveRedemption":      minter["MinterService.ApproveRedemption"],
		"MinterService.DenyRedemption":         minter["MinterService.DenyRedemption"],
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, selected)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("load redemption manifest group: %w", err)
	}
	return group, nil
}
