// Package env derives the resource's runtime storage paths. Per
// docs/resources/storage.md, kopia never uses repo-local data/ roots: all
// runtime state resolves to host-scoped, storage-class paths
// (<class-root>/vrooli/resources/kopia/...) with XDG defaults. Operators may
// override the roots via KOPIA_*_DIR env vars.
package env

import (
	"os"
	"path/filepath"
	"strings"
)

const resourceSlug = "kopia"

// Runtime holds the resolved storage-class roots and the concrete paths the
// resource writes to (the repo registry, per-repo kopia config files, caches).
type Runtime struct {
	ConfigRoot   string // config class: kopia config files, operator config
	StateRoot    string // state class: registry, locks, transient control state
	CacheRoot    string // cache class: kopia content caches (rebuildable)
	LogsRoot     string // logs class
	ReposDir     string // <config>/repos: one subdir per repository
	RegistryFile string // <state>/registry.json: name -> repo metadata
}

// Load resolves the runtime from env overrides and XDG defaults.
func Load() Runtime {
	configRoot := classRoot("KOPIA_CONFIG_DIR", "XDG_CONFIG_HOME", filepath.Join(".config"))
	stateRoot := classRoot("KOPIA_STATE_DIR", "XDG_STATE_HOME", filepath.Join(".local", "state"))
	cacheRoot := classRoot("KOPIA_CACHE_DIR", "XDG_CACHE_HOME", filepath.Join(".cache"))
	logsRoot := classRoot("KOPIA_LOGS_DIR", "XDG_STATE_HOME", filepath.Join(".local", "state"))
	if logsRoot == stateRoot {
		logsRoot = filepath.Join(logsRoot, "logs")
	}

	return Runtime{
		ConfigRoot:   configRoot,
		StateRoot:    stateRoot,
		CacheRoot:    cacheRoot,
		LogsRoot:     logsRoot,
		ReposDir:     filepath.Join(configRoot, "repos"),
		RegistryFile: filepath.Join(stateRoot, "registry.json"),
	}
}

// RepoConfigFile returns the kopia config-file path for a named repository.
func (r Runtime) RepoConfigFile(repo string) string {
	return filepath.Join(r.ReposDir, repo, "repository.config")
}

// RepoCacheDir returns the kopia cache directory for a named repository.
func (r Runtime) RepoCacheDir(repo string) string {
	return filepath.Join(r.CacheRoot, "repos", repo)
}

// EnsureDirectories creates the storage-class roots the resource writes to.
func (r Runtime) EnsureDirectories() error {
	for _, path := range []string{r.ConfigRoot, r.StateRoot, r.CacheRoot, r.LogsRoot, r.ReposDir} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			return err
		}
	}
	return nil
}

// classRoot resolves a storage-class root: an explicit KOPIA_*_DIR override
// wins; otherwise an XDG base joined with vrooli/resources/kopia.
func classRoot(overrideEnv, xdgEnv, homeRel string) string {
	if v := strings.TrimSpace(os.Getenv(overrideEnv)); v != "" {
		return v
	}
	base := strings.TrimSpace(os.Getenv(xdgEnv))
	if base == "" {
		base = filepath.Join(userHomeDir(), homeRel)
	}
	return filepath.Join(base, "vrooli", "resources", resourceSlug)
}

func userHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "."
	}
	return home
}
