package permissions

import (
	"path/filepath"

	"github.com/vrooli/agentharness"
)

type State = agentharness.PermissionState

const StateSchemaVersion = agentharness.PermissionStateSchemaVersion

func (a *Adapter) StatePath() string {
	name := ".vrooli-permissions-state.json"
	if a.Scope == ScopeAdmin {
		name = ".vrooli-permissions-state.admin.json"
	}
	return filepath.Join(filepath.Dir(a.SettingsPath), name)
}

func (a *Adapter) LoadState() (*State, error) {
	return agentharness.LoadPermissionState(a.StatePath())
}

func (a *Adapter) WriteState(p Policy, writtenByVersion string) error {
	return agentharness.WritePermissionState(a.StatePath(), agentharness.PermissionPolicy{
		BashDeny: p.BashDeny, BashAsk: p.BashAsk, BashAllow: p.BashAllow, SettingsPath: a.SettingsPath,
	}, Fingerprint(p), writtenByVersion, string(a.Scope))
}
