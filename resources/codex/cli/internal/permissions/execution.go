package permissions

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"

	"github.com/vrooli/agentharness"
)

const ExecutionProfile = "vrooli"

func nativeProfile(e *agentharness.ExecutionPermissions, active bool) map[string]any {
	parent := ":read-only"
	if e.Filesystem.Workspace == "write" {
		parent = ":workspace"
	}
	profile := map[string]any{"extends": parent, "network": map[string]any{"enabled": active && *e.Network.Enabled}}
	if len(e.Network.Domains) > 0 {
		profile["network"].(map[string]any)["domains"] = e.Network.Domains
	}
	if len(e.Filesystem.WritableRoots) > 0 {
		files := map[string]any{}
		for _, root := range e.Filesystem.WritableRoots {
			files[root] = map[string]any{".": "write", ".git": "read", ".codex": "read", ".agents": "read"}
		}
		profile["filesystem"] = files
	}
	return profile
}

func legacyConflict(doc map[string]any) bool {
	for key, value := range doc {
		if key == "sandbox_mode" || key == "sandbox_workspace_write" {
			return true
		}
		if nested, ok := value.(map[string]any); ok && legacyConflict(nested) {
			return true
		}
	}
	return false
}

func projectExecution(doc map[string]any, e *agentharness.ExecutionPermissions, active bool) error {
	for _, key := range []string{vrooliSectionKey, "permissions", "features"} {
		if value, exists := doc[key]; exists {
			if _, ok := value.(map[string]any); !ok {
				return fmt.Errorf("native %s must be a table; refusing to replace an incompatible value", key)
			}
		}
	}

	if active && legacyConflict(doc) {
		return errors.New("legacy sandbox settings conflict with permission profiles; migrate explicitly before activation")
	}
	section, _ := doc[vrooliSectionKey].(map[string]any)
	if section == nil {
		section = map[string]any{}
		doc[vrooliSectionKey] = section
	}
	profiles, _ := doc["permissions"].(map[string]any)
	if profiles == nil {
		profiles = map[string]any{}
		doc["permissions"] = profiles
	}
	if _, exists := profiles[ExecutionProfile]; exists && section["execution"] == nil {
		return errors.New("permissions.vrooli already exists without Vrooli ownership; choose an explicit migration")
	}
	profiles[ExecutionProfile] = nativeProfile(e, active)
	section["execution"] = e
	section["execution_active"] = active
	if active {
		doc["default_permissions"] = ExecutionProfile
		doc["approval_policy"] = e.Approval.Policy
		doc["approvals_reviewer"] = e.Approval.Reviewer
		features, _ := doc["features"].(map[string]any)
		if features == nil {
			features = map[string]any{}
			doc["features"] = features
		}
		// Keep the proxy active even for network-off profiles: enabling the profile
		// later must never turn an allowlist into unrestricted direct access.
		features["network_proxy"] = true
	}
	return nil
}

func executionFingerprint(doc map[string]any, active bool) string {
	profiles, _ := doc["permissions"].(map[string]any)
	selected := map[string]any{"profile": profiles[ExecutionProfile]}
	if active {
		features, _ := doc["features"].(map[string]any)
		selected["default_permissions"] = doc["default_permissions"]
		selected["approval_policy"] = doc["approval_policy"]
		selected["approvals_reviewer"] = doc["approvals_reviewer"]
		selected["network_proxy"] = features["network_proxy"]
		selected["legacy_conflict"] = legacyConflict(doc)
	}
	return jsonFingerprint(selected)
}

func DesiredExecutionFingerprint(e *agentharness.ExecutionPermissions, active bool) string {
	if e == nil {
		return ""
	}
	doc := map[string]any{}
	_ = projectExecution(doc, e, active)
	return executionFingerprint(doc, active)
}

func jsonFingerprint(value any) string {
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (a *Adapter) hookSnapshot() (agentharness.PermissionFileSnapshot, error) {
	if a.HookPath == "" {
		return agentharness.PermissionFileSnapshot{}, nil
	}
	return agentharness.ReadPermissionFile(a.HookPath)
}

// hookFingerprint includes only our registration, preserving independent hooks.
func (a *Adapter) hookFingerprint(data []byte) (string, error) {
	if a.HookPath == "" {
		return "", nil
	}
	var doc map[string]any
	if len(data) > 0 {
		if err := json.Unmarshal(data, &doc); err != nil {
			return "", fmt.Errorf("parse hooks: %w", err)
		}
	}
	if len(data) > 0 && doc == nil {
		return "", errors.New("hooks must be a JSON object")
	}
	hooks, _ := doc["hooks"].(map[string]any)
	groups, _ := hooks["PreToolUse"].([]any)
	found := []any{}
	for _, raw := range groups {
		group, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		entries, _ := group["hooks"].([]any)
		for _, rawEntry := range entries {
			entry, ok := rawEntry.(map[string]any)
			if !ok {
				continue
			}
			id, _ := entry["_id"].(string)
			if id == "vrooli-policy-runner" {
				found = append(found, map[string]any{"matcher": group["matcher"], "hook": entry})
			}
		}
	}
	if len(found) == 0 {
		return "", nil
	}
	return jsonFingerprint(found), nil
}

func (a *Adapter) DesiredHookFingerprint(p Policy) string {
	if a.HookPath == "" {
		return ""
	}
	found := []any{}
	if hook := a.RenderHook(p); hook != nil {
		group := hook["hooks"].(map[string]any)["PreToolUse"].([]any)[0].(map[string]any)
		entry := group["hooks"].([]any)[0].(map[string]any)
		entry["_id"] = "vrooli-policy-runner"
		entry["managedBy"] = "vrooli"
		found = append(found, map[string]any{"matcher": group["matcher"], "hook": entry})
	}
	if len(found) == 0 {
		return ""
	}
	return jsonFingerprint(found)
}

// CheckExecutionProjection detects collisions and activation conflicts without
// modifying either native file.
func (a *Adapter) CheckExecutionProjection(p Policy) error {
	if p.Execution == nil {
		return nil
	}
	if a.Scope == ScopeAdmin {
		return errors.New("execution permissions currently support user scope only; admin requirements need a separately qualified projector")
	}
	snapshot, err := agentharness.ReadPermissionFile(a.SettingsPath)
	if err != nil {
		return err
	}
	doc := map[string]any{}
	if len(snapshot.Data) > 0 {
		if err = unmarshalConfig(snapshot.Data, &doc); err != nil {
			return err
		}
	}
	// Staging must not silently deactivate a previously active profile.
	section, _ := doc[vrooliSectionKey].(map[string]any)
	wasActive, _ := section["execution_active"].(bool)
	if wasActive && !p.ExecutionActive {
		return errors.New("profile was activated previously; use --activate to reconcile active settings")
	}
	return projectExecution(doc, p.Execution, p.ExecutionActive)
}

func retainBackup(snapshot agentharness.PermissionFileSnapshot) (string, error) {
	if !snapshot.Exists {
		return "", nil
	}
	file, err := agentharness.CreatePermissionTemp(filepath.Dir(snapshot.Path), ".vrooli-permissions-backup-")
	if err != nil {
		return "", err
	}
	path := file.Name()

	if _, err = file.Write(snapshot.Data); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return path, nil
}
func unmarshalConfig(data []byte, doc *map[string]any) error { return toml.Unmarshal(data, doc) }

// Reconcile keeps configuration publication, native readback and state publication
// under the same resource lock. Hook writes use the broker's own document lock.
func (a *Adapter) Reconcile(p Policy, version string, stateDrift bool) error {
	return agentharness.NewHookBroker().WithLock(a.SettingsPath, func() error {
		live, err := a.Load()
		if err != nil {
			return err
		}
		if p.SnapshotDigest != live.SnapshotDigest {
			return errors.New("native files changed since preview; plan again")
		}
		state, err := agentharness.ReadPermissionFile(a.StatePath())
		if err != nil {
			return err
		}
		if p.StateSnapshotDigest != agentharness.PermissionFilesDigest(state) {
			return errors.New("state file changed since preview; plan again")
		}
		changed := Fingerprint(live) != Fingerprint(p)
		if changed {
			if err = a.saveLocked(p); err != nil {
				return err
			}
		}
		live, err = a.Load()
		if err != nil {
			return fmt.Errorf("native settings readback failed: %w; backup: %s", err, a.LastBackup)
		}
		if Fingerprint(live) != Fingerprint(p) {
			return fmt.Errorf("native readback differs from desired projection; inspect and re-plan; backup: %s", a.LastBackup)
		}
		if changed || stateDrift {
			if err = a.writeStateLocked(p, version); err != nil {
				return fmt.Errorf("native settings applied but state publication failed: %w; backup: %s; inspect and re-plan", err, a.LastBackup)
			}
		}
		return nil
	})
}
