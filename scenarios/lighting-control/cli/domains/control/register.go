package control

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
	controlconnect "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control/controlconnect"
)

const GroupName = "control"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	group, err := cliapp.LoadProtoPrimitiveGroupFromManifest(core, controlconnect.ControlServiceName, manifest, GroupName, cliapp.ProtoBindingOptions{}, map[string]bool{
		"GetBackendStatus": true, "ListControllers": true, "GetDesiredState": true,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("control: load from manifest: %w", err)
	}
	return group, nil
}
