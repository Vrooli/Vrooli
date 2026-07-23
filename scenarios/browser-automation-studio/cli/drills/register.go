package drills

import (
	"fmt"

	"browser-automation-studio/cli/internal/protodispatch"
	"github.com/vrooli/cli-core/cliapp"
	drillsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/drills"
)

const GroupName = "drills"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	svc := drillsv1.File_browser_automation_studio_v1_drills_drills_proto.Services().ByName("FailureDrillService")
	if svc == nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: FailureDrillService descriptor not found", GroupName)
	}
	bindings, err := protodispatch.Bindings(core, svc.FullName())
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: %w", GroupName, err)
	}
	return cliapp.LoadFromManifest(manifest, GroupName, bindings)
}
