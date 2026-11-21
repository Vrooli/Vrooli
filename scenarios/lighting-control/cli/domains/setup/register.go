package setup

import (
	"github.com/vrooli/cli-core/cliapp"
	setupconnect "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/setup/setupconnect"
)

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	return cliapp.LoadProtoPrimitiveGroupFromManifest(core, setupconnect.SetupServiceName, manifest, "setup", cliapp.ProtoBindingOptions{}, map[string]bool{"GetSetupStatus": true, "GetSetupDevices": true})
}
