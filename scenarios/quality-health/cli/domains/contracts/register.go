package contracts

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "contracts"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"AuditService.ListContracts": h.list,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("contracts: load from manifest: %w", err)
	}
	return group, nil
}
