package audit

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "audit"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"AuditService.AuditQuality": h.run,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("audit: load from manifest: %w", err)
	}
	return group, nil
}
