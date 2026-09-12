package grants

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
	_ "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access"
)

const GroupName = "grants"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	bindings, err := cliapp.ProtoPrimitiveBindings(core, "vrooli.token_economy.v1.access.MinterService", cliapp.ProtoBindingOptions{}, map[string]bool{
		"GetGrant": true, "ListGrants": true,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("build grants bindings: %w", err)
	}
	selected := make(map[string]cliapp.PrimitiveHandler, 4)
	for _, method := range []string{"CreateGrant", "GetGrant", "ListGrants", "RevokeGrant"} {
		selected["MinterService."+method] = bindings["MinterService."+method]
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, selected)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("load grants manifest group: %w", err)
	}
	return group, nil
}
