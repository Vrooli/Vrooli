package permissions

import "github.com/vrooli/agentharness"

// State is the provider-neutral permissions sidecar shared by all coding
// agent adapters. The alias preserves the native adapter API for callers.
type State = agentharness.PermissionState

const StateSchemaVersion = agentharness.PermissionStateSchemaVersion

func (a *Adapter) StatePath() string {
	return agentharness.PermissionStateSettingsPath(a.SettingsPath)
}

func (a *Adapter) LoadState() (*State, error) {
	return agentharness.LoadPermissionState(a.StatePath())
}

func (a *Adapter) WriteState(p Policy, writtenByVersion string) error {
	return agentharness.WritePermissionState(a.StatePath(), agentharness.PermissionPolicy{
		BashDeny: p.BashDeny, BashAsk: p.BashAsk, BashAllow: p.BashAllow, SettingsPath: a.SettingsPath,
	}, Fingerprint(p), writtenByVersion, "")
}
