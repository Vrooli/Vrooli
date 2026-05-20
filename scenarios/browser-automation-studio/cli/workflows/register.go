// Package workflows wires the BAS `workflows` CLI group to the proto
// WorkflowsService via the generic protodispatch dispatcher.
//
// The manifest (cli/manifest.json) is the single source of truth for
// command shape. WorkflowsService lives in the
// browser_automation_studio.v1 namespace alongside ExecutionsService;
// both share the same proto file (api/service.proto).
//
// Part of Phase 10 of the BAS proto+Connect migration
// (plans:bas-migration-to-proto-connect-rpc): the prior hand-coded
// REST-driven workflow handlers (execute.go, list.go, etc.) targeted
// chi routes that have been deleted from the API; this register
// supersedes them.
//
// step_parser.go and workflow_builder.go remain in this package as pure
// helpers used by the deprecated `execute.go` path. They are kept while
// any embedder still imports them; their unit tests still pass.
package workflows

import (
	"browser-automation-studio/cli/internal/protodispatch"
	"fmt"

	apiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this register package owns.
const GroupName = "workflows"

// Register builds the workflows subcommand group from the embedded
// manifest, binding every command to a generic protodispatch handler
// resolved against the generated WorkflowsService descriptor.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	svc := apiv1.File_browser_automation_studio_v1_api_service_proto.Services().ByName("WorkflowsService")
	if svc == nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: WorkflowsService descriptor not found", GroupName)
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
