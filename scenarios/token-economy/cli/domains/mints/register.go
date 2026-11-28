package mints

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
	// Register the authenticated MinterService descriptor used by the generic proto dispatcher.
	_ "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access"
)

const GroupName = "mints"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	bindings, err := cliapp.ProtoPrimitiveBindings(
		core,
		"vrooli.token_economy.v1.access.MinterService",
		cliapp.ProtoBindingOptions{},
		map[string]bool{"GetTokenType": true, "ListTokenTypes": true},
	)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("build mints bindings: %w", err)
	}
	selected := make(map[string]cliapp.PrimitiveHandler, 5)
	for _, method := range []string{"CreateTokenType", "GetTokenType", "ListTokenTypes", "RetireTokenType", "MintSupply"} {
		selected["MinterService."+method] = bindings["MinterService."+method]
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, selected)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("load mints manifest group: %w", err)
	}
	return group, nil
}
