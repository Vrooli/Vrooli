// Package permissions manages the OpenCode agent's permission config at
// ~/.config/opencode/opencode.json.
//
// Scope is narrow on purpose: the adapter owns only the
// `permission.bash` map (string→"allow"|"ask"|"deny") and only the
// entries it previously wrote. Every other top-level key, every other
// `permission.*` key (e.g. `edit`), and every `permission.bash` entry
// not previously tagged Vrooli-managed round-trips untouched.
//
// CRITICAL — opencode.json carries ONLY opencode-schema-valid keys.
// opencode rejects any unknown top-level key ("Configuration is invalid …
// Unrecognized key"), so the list of which bash patterns this adapter
// manages is NOT stored inline. It lives in the sidecar state file
// (.vrooli-permissions-state.json, backed by agentharness), which is the source of
// truth for "which entries are managed"; hand-written entries are detected
// by their absence from it. A retired pre-1.0 build wrote an inline
// `x-vrooli-managed-permissions` key — that is the bug this design fixes;
// it is now migrated into the sidecar on read and stripped on write.
//
// OpenCode honours the config directly per upstream docs (last-match-wins),
// and the adapter also projects a Vrooli plugin using the installed
// tool.execute.before seam. The plugin's live firing still requires a canary;
// configuration presence alone is not reported as verified enforcement. The adapter
// writes alphabetically-sorted keys so output is deterministic; under
// alphabetical order `*` (0x2A) sorts before letters, so specific
// patterns end up later and win the last-match-wins evaluation. Users
// needing different ordering can hand-edit; drift-check will surface
// it.
package permissions

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	opencodeversion "github.com/vrooli/vrooli/packages/opencode-version"
	"strings"

	"github.com/vrooli/vrooli/internal/cliout"
)

// legacyManagedKey is the retired pre-1.0 inline top-level key that once
// recorded the managed bash patterns directly in opencode.json. opencode
// rejects it as an unrecognized key, so the adapter no longer writes it: on
// read it is migrated into the sidecar state file, and on write it is
// stripped from opencode.json.
const legacyManagedKey = "x-vrooli-managed-permissions"

// Policy is the canonical in-memory shape of the bash-pattern subset of
// the OpenCode permission file the adapter manages.
type Policy struct {
	BashDeny  []string
	BashAsk   []string
	BashAllow []string
}

// Adapter binds the on-disk opencode.json path.
type Adapter struct {
	// SettingsPath is the absolute path of opencode.json (typically
	// ~/.config/opencode/opencode.json).
	SettingsPath string
	// PluginPath is the OpenCode 1 plugin projection path. An empty value keeps
	// the adapter focused on settings only, which is useful for isolated tests.
	PluginPath string
	// PluginPathV2 is the OpenCode 2 plugin projection path. Empty derives a
	// sibling `-v2` filename from PluginPath. Keeping the files separate lets a
	// rollback switch majors without leaving the wrong module format behind.
	PluginPathV2 string
	// ProjectionPath is an additional config file that must receive the same
	// permission object for the installed major. OpenCode 2 reads only the
	// explicit OPENCODE_CONFIG_DIR when it is set, while the managed permission
	// map lives in the XDG config; projection keeps native rules effective.
	ProjectionPath string
	// Major pins the installed OpenCode major for tests. Zero detects it from
	// the `opencode` binary once and caches the result on this adapter.
	Major int
	major int
}

// DefaultAdapter returns an Adapter rooted at $HOME/.config/opencode.
func DefaultAdapter() (*Adapter, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve $HOME: %w", err)
	}
	settingsPath := filepath.Join(home, ".config", "opencode", "opencode.json")
	pluginDir := filepath.Join(home, ".config", "opencode", "plugins")
	v2PluginDir := pluginDir
	projectionPath := ""
	// OpenCode 2 discovers plugins and reads configuration only from the
	// explicit config root when OPENCODE_CONFIG_DIR is set; V1 also discovers
	// the XDG roots. Keep the V1 projection where V1 finds it, and project the
	// V2 module and native permissions into the root V2 actually reads.
	if dir := strings.TrimSpace(os.Getenv("OPENCODE_CONFIG_DIR")); dir != "" {
		v2PluginDir = filepath.Join(dir, "plugins")
		if candidate := filepath.Join(dir, "opencode.json"); filepath.Clean(candidate) != filepath.Clean(settingsPath) {
			projectionPath = candidate
		}
	}
	return &Adapter{
		SettingsPath:   settingsPath,
		PluginPath:     filepath.Join(pluginDir, "vrooli-policy.js"),
		PluginPathV2:   filepath.Join(v2PluginDir, "vrooli-policy-v2.js"),
		ProjectionPath: projectionPath,
	}, nil
}

// parsedDoc holds the full top-level structure so Save can rebuild it
// preserving unknown keys and unmanaged bash entries.
type parsedDoc struct {
	TopLevel      map[string]json.RawMessage
	BashMap       map[string]string
	OtherPermKeys map[string]json.RawMessage
	// LegacyManaged holds the value of the retired inline managed-list key
	// when an old opencode.json still carries it, so Load/Save can migrate
	// it into the sidecar. The key itself is stripped from TopLevel during
	// parsing so it never round-trips back into the (schema-validated) file.
	LegacyManaged []string
}

func parseDoc(data []byte) (*parsedDoc, error) {
	doc := &parsedDoc{
		TopLevel:      map[string]json.RawMessage{},
		BashMap:       map[string]string{},
		OtherPermKeys: map[string]json.RawMessage{},
	}
	if err := json.Unmarshal(data, &doc.TopLevel); err != nil {
		return nil, fmt.Errorf("parse opencode.json: %w", err)
	}
	if raw, ok := doc.TopLevel["permission"]; ok {
		var permAll map[string]json.RawMessage
		if err := json.Unmarshal(raw, &permAll); err != nil {
			return nil, fmt.Errorf("parse permission: %w", err)
		}
		for k, v := range permAll {
			if k == "bash" {
				// Bash may be either a default-all string ("ask") or a
				// map. Only the map form has manageable entries.
				var asMap map[string]string
				if err := json.Unmarshal(v, &asMap); err == nil {
					doc.BashMap = asMap
					continue
				}
				// Default-all form: round-trip under OtherPermKeys so
				// it survives Save.
				doc.OtherPermKeys["bash"] = v
				continue
			}
			doc.OtherPermKeys[k] = v
		}
	}
	// Migrate + strip the retired inline managed-list key. Capturing its
	// value lets managedSet fall back to it once (until the sidecar is
	// written); deleting it from TopLevel guarantees Save never re-emits the
	// schema-invalid key.
	if raw, ok := doc.TopLevel[legacyManagedKey]; ok {
		var lst []string
		if err := json.Unmarshal(raw, &lst); err == nil {
			doc.LegacyManaged = lst
		}
		delete(doc.TopLevel, legacyManagedKey)
	}
	return doc, nil
}

// managedSet returns the set of bash patterns this adapter owns. The sidecar
// state file is the source of truth; the retired inline key is consulted only
// as a one-time migration fallback for configs written before the sidecar
// carried the managed list.
func (a *Adapter) managedSet(doc *parsedDoc) (map[string]struct{}, error) {
	st, err := a.LoadState()
	if err != nil {
		return nil, err
	}
	var managed []string
	switch {
	case st != nil && len(st.ManagedBash) > 0:
		managed = st.ManagedBash
	case doc != nil:
		managed = doc.LegacyManaged
	}
	out := make(map[string]struct{}, len(managed))
	for _, k := range managed {
		out[k] = struct{}{}
	}
	return out, nil
}

// Load reads and parses the opencode.json file. A missing file resolves
// to an empty Policy so callers can use Save to create a new file.
func (a *Adapter) Load() (Policy, error) {
	data, err := os.ReadFile(a.SettingsPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Policy{}, nil
		}
		return Policy{}, fmt.Errorf("read %s: %w", a.SettingsPath, err)
	}
	if len(data) == 0 {
		return Policy{}, nil
	}
	doc, err := parseDoc(data)
	if err != nil {
		return Policy{}, err
	}
	managed, err := a.managedSet(doc)
	if err != nil {
		return Policy{}, err
	}
	p := Policy{}
	for pat, action := range doc.BashMap {
		// Only surface entries we previously wrote — hand-written
		// entries stay in the file but aren't reported as policy.
		if _, ok := managed[pat]; !ok {
			continue
		}
		switch action {
		case "deny":
			p.BashDeny = append(p.BashDeny, pat)
		case "ask":
			p.BashAsk = append(p.BashAsk, pat)
		case "allow":
			p.BashAllow = append(p.BashAllow, pat)
		}
	}
	sort.Strings(p.BashDeny)
	sort.Strings(p.BashAsk)
	sort.Strings(p.BashAllow)
	return p, nil
}

// Save writes the policy to disk. It rebuilds `permission.bash` preserving
// hand-written entries (those not in the managed list) and replacing only
// the entries it previously owned, writes opencode.json with ONLY
// schema-valid keys (the retired managed-list key is stripped), then records
// the new managed list + fingerprint in the sidecar state file. The sidecar
// write is what makes the managed set authoritative across calls, so Save is
// self-consistent: two consecutive Saves correctly drop the first's entries.
func (a *Adapter) Save(p Policy, writtenByVersion string) error {
	if err := os.MkdirAll(filepath.Dir(a.SettingsPath), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(a.SettingsPath), err)
	}

	doc := &parsedDoc{
		TopLevel:      map[string]json.RawMessage{},
		BashMap:       map[string]string{},
		OtherPermKeys: map[string]json.RawMessage{},
	}
	if data, err := os.ReadFile(a.SettingsPath); err == nil && len(data) > 0 {
		parsed, perr := parseDoc(data)
		if perr != nil {
			return perr
		}
		doc = parsed
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("read %s: %w", a.SettingsPath, err)
	}

	// Drop any previously-managed bash entries; hand-written entries
	// (not in the managed list) stay.
	managed, err := a.managedSet(doc)
	if err != nil {
		return err
	}
	newBash := map[string]string{}
	for pat, action := range doc.BashMap {
		if _, ok := managed[pat]; ok {
			continue
		}
		newBash[pat] = action
	}

	// Add the policy's entries.
	for _, pat := range p.BashDeny {
		newBash[pat] = "deny"
	}
	for _, pat := range p.BashAsk {
		newBash[pat] = "ask"
	}
	for _, pat := range p.BashAllow {
		newBash[pat] = "allow"
	}

	// Rebuild permission.* preserving other keys.
	permAll := map[string]json.RawMessage{}
	for k, v := range doc.OtherPermKeys {
		permAll[k] = v
	}
	if len(newBash) > 0 {
		encoded, err := marshalOrderedStringMap(newBash)
		if err != nil {
			return fmt.Errorf("encode permission.bash: %w", err)
		}
		permAll["bash"] = encoded
	} else {
		delete(permAll, "bash")
	}

	if len(permAll) == 0 {
		delete(doc.TopLevel, "permission")
	} else {
		encoded, err := marshalOrderedRawMap(permAll)
		if err != nil {
			return fmt.Errorf("encode permission: %w", err)
		}
		doc.TopLevel["permission"] = encoded
	}

	out, err := marshalOrderedRawMap(doc.TopLevel)
	if err != nil {
		return fmt.Errorf("encode opencode.json: %w", err)
	}
	pretty, err := prettifyJSON(out)
	if err != nil {
		return err
	}
	tmp := a.SettingsPath + ".tmp"
	if err := os.WriteFile(tmp, pretty, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, a.SettingsPath); err != nil {
		return fmt.Errorf("rename %s -> %s: %w", tmp, a.SettingsPath, err)
	}

	// The sidecar is the source of truth for the managed list; persist it
	// (plus fingerprint/version) so the next Load/Save sees these entries as
	// managed without an inline marker in opencode.json.
	if err := a.WriteState(p, writtenByVersion); err != nil {
		return err
	}
	return a.syncPlugin(p)
}

// RenderPlugin returns the native OpenCode plugin projection for the detected
// major. The hook sends a normalized event to the local policy runner and
// refuses the tool call if the runner returns a non-zero decision. No shell
// interpolation is used.
func (a *Adapter) RenderPlugin() string {
	if a.opencodeMajor() >= 2 {
		return a.RenderV2Plugin()
	}
	return a.RenderV1Plugin()
}

// RenderV1Plugin is the OpenCode 1 `tool.execute.before` projection. The hook
// signature is (input, output); tool arguments live on output.args, not
// input.args. A previous revision read the wrong argument, which silently sent
// an empty command to the policy runner.
func (a *Adapter) RenderV1Plugin() string {
	runner := a.policyRunner()
	return fmt.Sprintf(`// Managed by Vrooli. Do not edit; reconcile permissions to regenerate.
export const VrooliPolicy = async () => ({
  "tool.execute.before": async (input, output) => {
    if (!input || input.tool !== "bash") return;
    const args = output && output.args;
    const list = Array.isArray(args)
      ? args.map(String)
      : args && typeof args === "object" && typeof args.command === "string"
        ? [args.command]
        : args == null ? [] : [String(args)];
    const event = {
      contract_version: "agent-policy/v1",
      runner: "opencode",
      tool: "bash",
      arguments: list,
      working_directory: process.cwd(),
      occurred_at: new Date().toISOString()
    };
    if (input.sessionID) event.context = { session_id: input.sessionID };
    // Bun.spawnSync feeds a TypedArray given as stdin to the child and then
    // closes it; there is no Node-style "input" option. With stdin: "pipe" and
    // no writer the runner read an empty body, exited 1 with "hook input is
    // empty", and every bash call was refused.
    const result = Bun.spawnSync([%q, "hook", "--runner", "opencode"], {
      stdin: new TextEncoder().encode(JSON.stringify(event)),
      stdout: "pipe",
      stderr: "pipe"
    });
    if (result.exitCode !== 0) {
      throw new Error("Vrooli policy runner denied or could not evaluate this tool call");
    }
  }
});
`, runner)
}

// RenderV2Plugin is the OpenCode 2 plugin projection. The V2 permission hook is
// the only seam that can return all three effects, so ask and deny are not
// collapsed into a thrown error.
func (a *Adapter) RenderV2Plugin() string {
	runner := a.policyRunner()
	return fmt.Sprintf(`// Managed by Vrooli. Do not edit; reconcile permissions to regenerate.
import { Plugin } from "@opencode/plugin"

const runner = %q

export default Plugin.define({
  id: "vrooli.policy",
  async setup(ctx) {
    await ctx.permission.hook("evaluate", async (event) => {
      if (!event) return;
      const action = typeof event.action === "string" && event.action ? event.action : "shell";
      const resources = Array.isArray(event.resources) ? event.resources.map(String) : [];
      const payload = {
        contract_version: "agent-policy/v1",
        runner: "opencode",
        tool: action === "shell" ? "bash" : action,
        arguments: resources,
        working_directory: ctx && ctx.location && ctx.location.directory ? ctx.location.directory : process.cwd(),
        occurred_at: new Date().toISOString()
      };
      if (event.sessionID) payload.context = { session_id: event.sessionID };
      const result = Bun.spawnSync([runner, "hook", "--runner", "opencode"], {
        stdin: new TextEncoder().encode(JSON.stringify(payload)),
        stdout: "pipe",
        stderr: "pipe"
      });
      if (result.exitCode === 10) {
        event.effect = "ask";
      } else if (result.exitCode === 20) {
        event.effect = "deny";
        event.message = "Vrooli policy denied this action";
      } else if (result.exitCode !== 0) {
        event.effect = "deny";
        event.message = "Vrooli policy runner could not evaluate this action";
      }
    });
  }
});
`, runner)
}

func (a *Adapter) policyRunner() string {
	runner := os.Getenv("VROOLI_AGENT_POLICY_RUNNER")
	if strings.TrimSpace(runner) == "" {
		runner = "vrooli-policy-runner"
	}
	return runner
}

func (a *Adapter) opencodeMajor() int {
	if a.Major > 0 {
		return a.Major
	}
	if a.major > 0 {
		return a.major
	}
	a.major = 1
	path, err := exec.LookPath("opencode")
	if err != nil {
		return a.major
	}
	out, err := exec.Command(path, "--version").Output()
	if err != nil {
		return a.major
	}
	if major, ok := opencodeversion.ParseMajor(string(out)); ok {
		a.major = major
	}
	return a.major
}

func (a *Adapter) pluginPaths() (string, string) {
	v1 := strings.TrimSpace(a.PluginPath)
	v2 := strings.TrimSpace(a.PluginPathV2)
	if v2 == "" && v1 != "" {
		ext := filepath.Ext(v1)
		v2 = strings.TrimSuffix(v1, ext) + "-v2" + ext
	}
	return v1, v2
}

// SyncPlugin reconciles the native plugin projection without rewriting the
// permission map. ensure calls it even when the map is already clean, so a
// major upgrade still replaces the previous major's plugin module.
func (a *Adapter) SyncPlugin(p Policy) error { return a.syncPlugin(p) }

func (a *Adapter) syncPlugin(p Policy) error {
	if err := a.syncPluginFile(p); err != nil {
		return err
	}
	return a.projectPermissions()
}

func (a *Adapter) syncPluginFile(p Policy) error {
	v1, v2 := a.pluginPaths()
	active, inactive := v1, v2
	if a.opencodeMajor() >= 2 {
		active, inactive = v2, v1
	}
	if len(p.BashDeny) == 0 {
		for _, path := range []string{v1, v2} {
			if path == "" {
				continue
			}
			if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("remove OpenCode policy plugin: %w", err)
			}
		}
		return nil
	}
	if active == "" {
		return nil
	}
	// A rolled-back major must not discover the other major's module shape.
	if inactive != "" {
		if err := os.Remove(inactive); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("remove incompatible OpenCode policy plugin: %w", err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(active), 0o755); err != nil {
		return fmt.Errorf("create OpenCode plugin directory: %w", err)
	}
	tmp := active + ".tmp"
	if err := os.WriteFile(tmp, []byte(a.RenderPlugin()), 0o644); err != nil {
		return fmt.Errorf("write OpenCode policy plugin: %w", err)
	}
	if err := os.Rename(tmp, active); err != nil {
		return fmt.Errorf("publish OpenCode policy plugin: %w", err)
	}
	return nil
}

// projectPermissions copies the permission object to the explicit V2 config
// root. The Vrooli config there already carries provider/model blocks, so only
// the permission key is replaced; every other key is preserved byte-for-byte.
func (a *Adapter) projectPermissions() error {
	if a.opencodeMajor() < 2 {
		return nil
	}
	projection := strings.TrimSpace(a.ProjectionPath)
	if projection == "" {
		return nil
	}
	source, err := os.ReadFile(a.SettingsPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read OpenCode permissions for projection: %w", err)
	}
	var sourceDoc map[string]json.RawMessage
	if err := json.Unmarshal(source, &sourceDoc); err != nil {
		return fmt.Errorf("parse OpenCode permissions for projection: %w", err)
	}
	targetDoc := map[string]json.RawMessage{}
	if target, err := os.ReadFile(projection); err == nil {
		if err := json.Unmarshal(target, &targetDoc); err != nil {
			return fmt.Errorf("parse projection config %s: %w", projection, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read projection config %s: %w", projection, err)
	}
	if permission, ok := sourceDoc["permission"]; ok {
		targetDoc["permission"] = permission
	} else {
		delete(targetDoc, "permission")
	}
	rendered, err := json.MarshalIndent(targetDoc, "", "  ")
	if err != nil {
		return fmt.Errorf("render projection config: %w", err)
	}
	rendered = append(rendered, '\n')
	if existing, err := os.ReadFile(projection); err == nil && bytes.Equal(existing, rendered) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(projection), 0o755); err != nil {
		return fmt.Errorf("create projection directory: %w", err)
	}
	tmp := projection + ".tmp"
	if err := os.WriteFile(tmp, rendered, 0o644); err != nil {
		return fmt.Errorf("write projection config: %w", err)
	}
	if err := os.Rename(tmp, projection); err != nil {
		return fmt.Errorf("publish projection config: %w", err)
	}
	return nil
}

// Fingerprint returns the sha256 hex of the canonical Policy projection.
// drift-check compares this against the state-file value.
func Fingerprint(p Policy) string {
	canon := struct {
		Allow []string `json:"allow"`
		Ask   []string `json:"ask"`
		Deny  []string `json:"deny"`
	}{
		Allow: sortedCopy(p.BashAllow),
		Ask:   sortedCopy(p.BashAsk),
		Deny:  sortedCopy(p.BashDeny),
	}
	data, _ := json.Marshal(canon)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func marshalOrderedRawMap(m map[string]json.RawMessage) (json.RawMessage, error) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var buf strings.Builder
	buf.WriteString("{")
	for i, k := range keys {
		if i > 0 {
			buf.WriteString(",")
		}
		kJSON, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		buf.Write(kJSON)
		buf.WriteString(":")
		buf.Write(m[k])
	}
	buf.WriteString("}")
	return json.RawMessage(buf.String()), nil
}

func marshalOrderedStringMap(m map[string]string) (json.RawMessage, error) {
	raw := map[string]json.RawMessage{}
	for k, v := range m {
		enc, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		raw[k] = enc
	}
	return marshalOrderedRawMap(raw)
}

func prettifyJSON(in json.RawMessage) ([]byte, error) {
	var v any
	if err := json.Unmarshal(in, &v); err != nil {
		return nil, err
	}
	out, err := cliout.MarshalIndent(v)
	if err != nil {
		return nil, err
	}
	out = append(out, '\n')
	return out, nil
}
