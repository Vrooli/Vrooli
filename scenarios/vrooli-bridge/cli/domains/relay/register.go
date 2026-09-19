// Package relay exposes the owner-facing signed channel relay as a typed CLI
// command. The project CLI uses this surface for addressed scenario status;
// operators can also invoke it directly for bounded read-classified calls.
package relay

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "relay"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"RelayService.Call": h.call,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("relay: load from manifest: %w", err)
	}
	return group, nil
}
