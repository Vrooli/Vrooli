package permissions

import (
	"fmt"
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
	return agentharness.NewHookBroker().WithLock(a.SettingsPath, func() error { return a.writeStateLocked(p, writtenByVersion) })
}

func (a *Adapter) writeStateLocked(p Policy, writtenByVersion string) error {
	live, err := a.Load()
	if err != nil {
		return err
	}
	expected := p
	expected.NativeFingerprint = DesiredExecutionFingerprint(live.Execution, live.ExecutionActive)
	expected.HookFingerprint = a.DesiredHookFingerprint(p)
	if Fingerprint(live) != Fingerprint(expected) {
		return fmt.Errorf("native files differ from desired intent; state publication refused")
	}

	return agentharness.WritePermissionState(a.StatePath(), agentharness.PermissionPolicy{
		BashDeny: p.BashDeny, BashAsk: p.BashAsk, BashAllow: p.BashAllow, SettingsPath: a.SettingsPath,
	}, Fingerprint(live), writtenByVersion, string(a.Scope))
}
