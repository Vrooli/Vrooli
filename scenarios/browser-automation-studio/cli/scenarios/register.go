// Package scenarios wires the BAS `scenarios` CLI group to the proto
// ScenariosService via the generic protodispatch dispatcher. The manifest
// (cli/manifest.json) is the single source of truth for command shape;
// this file only declares the manifest group name and supplies the
// dispatch table.
//
// Part of Phase 10 of the BAS proto+Connect migration
// (plans:bas-migration-to-proto-connect-rpc): every manifest group that
// lacked executor wiring before Phase 10 is now reachable via a register.go
// of this shape.
package scenarios

import (
	"browser-automation-studio/cli/internal/protodispatch"
	"fmt"

	scenariosv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/scenarios"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this register package owns.
const GroupName = "scenarios"

// Register builds the scenarios subcommand group by binding every
// manifest command to a generic protodispatch handler resolved against
// the generated ScenariosService descriptor.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	svc := scenariosv1.File_browser_automation_studio_v1_scenarios_scenarios_proto.Services().ByName("ScenariosService")
	if svc == nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: ScenariosService descriptor not found", GroupName)
	}
	bindings, err := protodispatch.Bindings(core, svc.FullName())
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: %w", GroupName, err)
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: load from manifest: %w", GroupName, err)
	}
	return group, nil
}
