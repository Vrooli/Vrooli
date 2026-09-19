package explain

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "explain"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"AuditService.ExplainFinding": h.explain,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("explain: load from manifest: %w", err)
	}
	return group, nil
}
