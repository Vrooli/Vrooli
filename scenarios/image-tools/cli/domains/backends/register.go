// Package backends owns the CLI's backend diagnostics surface.
package backends

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "backends"

// Register builds the backends subcommand group from the embedded manifest.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ModelsService.DoctorBackends": h.doctor,
		"ModelsService.EnsureBackend":  h.ensure,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("backends: load from manifest: %w", err)
	}
	return group, nil
}
