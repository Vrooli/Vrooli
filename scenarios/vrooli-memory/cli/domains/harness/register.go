package harness

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "harness"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	g, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"HarnessService.RunImport":       cliapp.ProtoMutation(h.importCall, h.importReport),
		"HarnessService.GetImportStatus": cliapp.ProtoList(h.statusCall, h.statusReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("harness: load manifest: %w", err)
	}
	return g, nil
}
