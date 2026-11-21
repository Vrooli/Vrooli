package gateway

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "gateway"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"GatewayService.ValidateGatewayRequest": h.validate,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("gateway: load from manifest: %w", err)
	}
	return group, nil
}
