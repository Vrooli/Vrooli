package earning

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
	_ "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/earning"
)

const GroupName = "earning"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	bindings, err := cliapp.ProtoPrimitiveBindings(core, "vrooli.token_economy.v1.earning.EarningService", cliapp.ProtoBindingOptions{}, map[string]bool{"ListEarnings": true})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("build earning bindings: %w", err)
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("load earning manifest group: %w", err)
	}
	return group, nil
}
