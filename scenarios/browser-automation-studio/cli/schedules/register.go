// Package schedules wires the BAS `schedules` CLI group to the proto
// SchedulesService via the generic protodispatch dispatcher. The manifest
// (cli/manifest.json) is the single source of truth for command shape.
//
// Part of Phase 10 of the BAS proto+Connect migration
// (plans:bas-migration-to-proto-connect-rpc).
package schedules

import (
	"fmt"


	pbsvc "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/schedules"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this register package owns.
const GroupName = "schedules"

// Register builds the schedules subcommand group by binding every manifest
// command to a generic protodispatch handler resolved against the
// generated SchedulesService descriptor.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	svc := pbsvc.File_browser_automation_studio_v1_schedules_schedules_proto.Services().ByName("SchedulesService")
	if svc == nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: SchedulesService descriptor not found", GroupName)
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
