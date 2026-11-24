package permissions

import (
	"github.com/vrooli/agentharness"
	"github.com/vrooli/vrooli/resources/opencode/cli/internal/config"
)

type State = agentharness.PermissionState

const StateSchemaVersion = agentharness.PermissionStateSchemaVersion

func (a *Adapter) StatePath() string {
	settingsPath := a.SettingsPath
	if canonical, err := config.CanonicalConfigPath(settingsPath); err == nil {
		settingsPath = canonical
	}
	return agentharness.PermissionStateSettingsPath(settingsPath)
}

func (a *Adapter) LoadState() (*State, error) {
	return agentharness.LoadPermissionState(a.StatePath())
}

func (a *Adapter) WriteState(p Policy, writtenByVersion string) error {
	return agentharness.WritePermissionState(a.StatePath(), agentharness.PermissionPolicy{
		BashDeny: p.BashDeny, BashAsk: p.BashAsk, BashAllow: p.BashAllow, SettingsPath: a.SettingsPath,
	}, Fingerprint(p), writtenByVersion, "")
}
