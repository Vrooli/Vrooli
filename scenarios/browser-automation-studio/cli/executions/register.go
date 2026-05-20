// Package executions wires the BAS `executions` CLI group to the proto
// ExecutionsService via the generic protodispatch dispatcher.
//
// ExecutionsService lives in the browser_automation_studio.v1 namespace
// alongside WorkflowsService (api/service.proto). Multipart export and
// HAR/video streaming endpoints remain on REST per
// docs/internal/REST_EXCEPTIONS.md; the JSON metadata methods migrate to
// Connect via this register.
//
// Part of Phase 10 of the BAS proto+Connect migration
// (plans:bas-migration-to-proto-connect-rpc). The prior hand-coded
// REST-driven execution handlers (list.go, watch.go, render.go,
// render_video.go, etc.) have been deleted; the chi routes they targeted
// were removed in earlier phases.
package executions

import (
	"browser-automation-studio/cli/internal/protodispatch"
	"fmt"

	apiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this register package owns.
const GroupName = "executions"

// Register builds the executions subcommand group by binding every
// manifest command to a generic protodispatch handler resolved against
// the generated ExecutionsService descriptor.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	svc := apiv1.File_browser_automation_studio_v1_api_service_proto.Services().ByName("ExecutionsService")
	if svc == nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: ExecutionsService descriptor not found", GroupName)
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
