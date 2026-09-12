package privacy

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "privacy"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"PrivacyService.GetRetentionSettings":    h.retention,
		"PrivacyService.UpdateRetentionSettings": h.retentionSet,
		"PrivacyService.GetVisibilitySettings":   h.visibility,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("privacy: load from manifest: %w", err)
	}
	return group, nil
}
