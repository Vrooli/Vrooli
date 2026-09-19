package command

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "command"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"CommandReferenceValidationService.ValidateCommandReference": cliapp.ProtoList(h.validateCall, h.validateReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("command: load from manifest: %w", err)
	}
	return group, nil
}
