// Package replay_config wires the BAS `replay-config` CLI group to the proto
// ReplayConfigService via the generic protodispatch dispatcher. The manifest
// (cli/manifest.json) is the single source of truth for command shape.
//
// Part of Phase 10 of the BAS proto+Connect migration
// (plans:bas-migration-to-proto-connect-rpc).
package replay_config

import (
	"fmt"


	pbsvc "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/replay_config"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this register package owns.
const GroupName = "replay-config"

// Register builds the replay-config subcommand group by binding every manifest
// command to a generic protodispatch handler resolved against the
// generated ReplayConfigService descriptor.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	svc := pbsvc.File_browser_automation_studio_v1_replay_config_replay_config_proto.Services().ByName("ReplayConfigService")
	if svc == nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: ReplayConfigService descriptor not found", GroupName)
	}
	bindings, err := cliapp.ProtoBindings(core, svc.FullName(), cliapp.ProtoBindingOptions{})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: %w", GroupName, err)
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: load from manifest: %w", GroupName, err)
	}
	return group, nil
}
