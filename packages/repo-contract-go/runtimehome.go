package repocontract

import (
	"path/filepath"
	"sort"
	"strings"
)

// Runtime-home entry keys. These are the stable logical identifiers for the
// well-known entries under the operator runtime home. Consumers reference these
// constants instead of bare path literals so the contract remains the single
// source of truth for the on-disk names (the contract maps key -> path).
const (
	HomeKeyPlans      = "plans"
	HomeKeyState      = "state"
	HomeKeyConfig     = "config"
	HomeKeyData       = "data"
	HomeKeyRuntimeDB  = "runtime_db"
	HomeKeySecrets    = "secrets"
	HomeKeySecretsEnc = "secrets_enc"
	HomeKeyBin        = "bin"
	HomeKeyCache      = "cache"
	HomeKeyLogs       = "logs"
	HomeKeyMetrics    = "metrics"
	HomeKeyProcesses  = "processes"
	HomeKeyBuild      = "build"
	HomeKeyTestRuns   = "test_runs"

	// Scoped (parameterized) runtime-home path keys.
	ScopedScenarioSecrets = "scenario_secrets"
	ScopedProjectState    = "project_state"
)

// RuntimeHomeEntryPath resolves the absolute path of a well-known runtime-home
// entry by loading the canonical contract from env/CWD. No fallback.
func RuntimeHomeEntryPath(home, key string) (string, error) {
	contract, err := loadRuntimeHomeContract()
	if err != nil {
		return "", err
	}
	entry, err := contract.RuntimeHomeEntry(home, key)
	if err != nil {
		return "", err
	}
	return entry.AbsPath, nil
}

// RuntimeHomeScopedPath expands a parameterized runtime-home template by loading
// the canonical contract from env/CWD. No fallback.
func RuntimeHomeScopedPath(home, key string, params map[string]string) (string, error) {
	contract, err := loadRuntimeHomeContract()
	if err != nil {
		return "", err
	}
	return contract.ScopedRuntimePath(home, key, params)
}

// RuntimeHomeSpec is the structural authority for the operator runtime home
// ($HOME/.vrooli): its directory name and the well-known entries inside it.
// It carries STRUCTURE only — resolution of the OS home (sudo-awareness) lives
// in the consuming layer (internal/config), never here.
type RuntimeHomeSpec struct {
	DirName      string                          `json:"dir_name"`
	EnvOverrides []string                        `json:"env_overrides"`
	Entries      map[string]RuntimeHomeEntrySpec `json:"entries"`
	Scoped       map[string]string               `json:"scoped"`
}

// RuntimeHomeEntrySpec is a single well-known entry directly under the runtime
// home, as authored in the contract.
type RuntimeHomeEntrySpec struct {
	Path        string `json:"path"`
	Kind        string `json:"kind"`
	Regenerable bool   `json:"regenerable"`
	Format      string `json:"format,omitempty"`
	Sensitive   bool   `json:"sensitive,omitempty"`
}

// HomeEntry is a runtime-home entry resolved against a concrete OS home dir.
type HomeEntry struct {
	Key         string // stable logical name (contract key)
	AbsPath     string // absolute path under the resolved home
	RelPath     string // home-relative path (slash form, as authored)
	Kind        string // "dir" | "file"
	Regenerable bool
	Format      string
	Sensitive   bool
}

// RuntimeHomeDirName returns the contract-defined runtime-home directory name
// (e.g. ".vrooli"). This is the single structural authority for that name.
func (c *Contract) RuntimeHomeDirName() string {
	return c.doc.RuntimeHome.DirName
}

// RuntimeHomeEnvOverrides returns the env vars (if any) the contract permits to
// override the resolved home root. The canonical contract ships none.
func (c *Contract) RuntimeHomeEnvOverrides() []string {
	return slicesClone(c.doc.RuntimeHome.EnvOverrides)
}

// RuntimeHome returns the absolute runtime-home root for the given OS home dir.
// The caller is responsible for resolving `home` (sudo-aware where relevant).
func (c *Contract) RuntimeHome(home string) (string, error) {
	root, err := joinHomeRoot(home, c.doc.RuntimeHome.DirName)
	if err != nil {
		return "", err
	}
	return root, nil
}

// RuntimeHomeEntry resolves a single well-known entry by its contract key.
func (c *Contract) RuntimeHomeEntry(home, key string) (HomeEntry, error) {
	root, err := c.RuntimeHome(home)
	if err != nil {
		return HomeEntry{}, err
	}
	spec, ok := c.doc.RuntimeHome.Entries[strings.TrimSpace(key)]
	if !ok {
		return HomeEntry{}, &Error{Kind: ErrNotFound, Message: "runtime-home entry not found", Details: key}
	}
	return resolveHomeEntry(root, strings.TrimSpace(key), spec), nil
}

// RuntimeHomeEntries resolves every well-known entry, sorted by key for
// deterministic output.
func (c *Contract) RuntimeHomeEntries(home string) ([]HomeEntry, error) {
	root, err := c.RuntimeHome(home)
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(c.doc.RuntimeHome.Entries))
	for key := range c.doc.RuntimeHome.Entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]HomeEntry, 0, len(keys))
	for _, key := range keys {
		out = append(out, resolveHomeEntry(root, key, c.doc.RuntimeHome.Entries[key]))
	}
	return out, nil
}

// ScopedRuntimePath expands a parameterized home-relative template (e.g.
// "scenarios/{scenario}/secrets.json") against the resolved home. Placeholder
// values are validated as identifiers (no path traversal, no separators).
func (c *Contract) ScopedRuntimePath(home, key string, params map[string]string) (string, error) {
	root, err := c.RuntimeHome(home)
	if err != nil {
		return "", err
	}
	tmpl, ok := c.doc.RuntimeHome.Scoped[strings.TrimSpace(key)]
	if !ok {
		return "", &Error{Kind: ErrNotFound, Message: "scoped runtime path not found", Details: key}
	}
	expanded, err := expandScopedTemplate(tmpl, params)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, filepath.FromSlash(expanded)), nil
}

func resolveHomeEntry(root, key string, spec RuntimeHomeEntrySpec) HomeEntry {
	return HomeEntry{
		Key:         key,
		AbsPath:     filepath.Join(root, filepath.FromSlash(spec.Path)),
		RelPath:     spec.Path,
		Kind:        spec.Kind,
		Regenerable: spec.Regenerable,
		Format:      spec.Format,
		Sensitive:   spec.Sensitive,
	}
}

func joinHomeRoot(home, dirName string) (string, error) {
	home = filepath.Clean(strings.TrimSpace(home))
	if home == "" || home == "." {
		return "", &Error{Kind: ErrInvalidInput, Message: "user home dir is required"}
	}
	if strings.TrimSpace(dirName) == "" {
		return "", &Error{Kind: ErrInvalidContract, Message: "runtime_home.dir_name is empty"}
	}
	return filepath.Join(home, dirName), nil
}

func expandScopedTemplate(tmpl string, params map[string]string) (string, error) {
	out := tmpl
	for name, value := range params {
		cleaned, err := cleanIdentifier(value)
		if err != nil {
			return "", err
		}
		out = strings.ReplaceAll(out, "{"+name+"}", cleaned)
	}
	if strings.ContainsAny(out, "{}") {
		return "", &Error{Kind: ErrInvalidInput, Message: "scoped template has unresolved placeholders", Details: out}
	}
	return out, nil
}
