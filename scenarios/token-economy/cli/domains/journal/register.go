package journal

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
	_ "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access"
)

const GroupName = "journal"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	bindings, err := cliapp.ProtoPrimitiveBindings(core, "vrooli.token_economy.v1.access.MinterService", cliapp.ProtoBindingOptions{}, map[string]bool{
		"ListJournalEvents": true, "ShowBalance": true, "ExportJournal": true,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("build journal bindings: %w", err)
	}
	selected := make(map[string]cliapp.PrimitiveHandler, 4)
	for _, method := range []string{"ListJournalEvents", "ShowBalance", "ExportJournal", "ReverseEvent"} {
		selected["MinterService."+method] = bindings["MinterService."+method]
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, selected)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("load journal manifest group: %w", err)
	}
	return group, nil
}
