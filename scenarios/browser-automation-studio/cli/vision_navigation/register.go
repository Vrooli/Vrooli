// Package vision_navigation wires the BAS `vision-navigation` CLI group
// to the proto VisionNavigationService via the generic protodispatch
// dispatcher. VisionNavigationService shares a proto file with AIService.
//
// Part of Phase 10 of the BAS proto+Connect migration
// (plans:bas-migration-to-proto-connect-rpc): the prior hand-coded
// `ai navigators`/`ai navigate` UX has been collapsed onto
// `vision-navigation list-navigators`/`start`/etc.
package vision_navigation

import (
	"fmt"

	"browser-automation-studio/cli/internal/protodispatch"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/ai"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this register package owns.
const GroupName = "vision-navigation"

// Register builds the vision-navigation subcommand group by binding every
// manifest command to a generic protodispatch handler resolved against the
// generated VisionNavigationService descriptor.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	svc := aiv1.File_browser_automation_studio_v1_ai_ai_proto.Services().ByName("VisionNavigationService")
	if svc == nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: VisionNavigationService descriptor not found", GroupName)
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
