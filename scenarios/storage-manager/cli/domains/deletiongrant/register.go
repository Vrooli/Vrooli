package deletiongrant

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "deletion-grant"

// Register exposes the operator deletion-grant request: the route an agent
// takes when the agent policy withholds a deletion, instead of handing the
// deletion back to the operator.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"request": cliapp.Action(h.requestCall, requestReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: load from manifest: %w", GroupName, err)
	}
	return group, nil
}
