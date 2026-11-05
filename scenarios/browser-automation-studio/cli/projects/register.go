// Package projects wires the BAS `projects` CLI group to the proto
// ProjectsService via the generic protodispatch dispatcher. The manifest
// (cli/manifest.json) is the single source of truth for command shape.
//
// Part of Phase 10 of the BAS proto+Connect migration
// (plans:bas-migration-to-proto-connect-rpc).
package projects

import (
	"fmt"


	pbsvc "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/projects"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this register package owns.
const GroupName = "projects"

// Register builds the projects subcommand group by binding every manifest
// command to a generic protodispatch handler resolved against the
// generated ProjectsService descriptor.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	svc := pbsvc.File_browser_automation_studio_v1_projects_project_proto.Services().ByName("ProjectsService")
	if svc == nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: ProjectsService descriptor not found", GroupName)
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
