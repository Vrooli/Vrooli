package consumerdeclarations

import (
	"fmt"
	"browser-automation-studio/cli/internal/protodispatch"
	consumerdeclarationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/consumer_declarations"
	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "consumer-declarations"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	svc := consumerdeclarationsv1.File_browser_automation_studio_v1_consumer_declarations_consumer_declarations_proto.Services().ByName("ConsumerDeclarationsService")
	if svc == nil { return cliapp.SubcommandGroup{}, fmt.Errorf("%s: ConsumerDeclarationsService descriptor not found", GroupName) }
	bindings, err := protodispatch.Bindings(core, svc.FullName()); if err != nil { return cliapp.SubcommandGroup{}, fmt.Errorf("%s: %w", GroupName, err) }
	return cliapp.LoadFromManifest(manifest, GroupName, bindings)
}
