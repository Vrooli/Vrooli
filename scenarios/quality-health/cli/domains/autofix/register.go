package autofix

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "fix-config"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"AuditService.PreviewFixConfig": h.preview,
		"AuditService.ApplyFixConfig":   h.apply,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("fix-config: load from manifest: %w", err)
	}
	return group, nil
}
