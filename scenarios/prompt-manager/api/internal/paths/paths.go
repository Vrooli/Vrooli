// Package paths is the single seam between prompt-manager and the three
// filesystem locations it writes to.
//
// The scenario has historically conflated three classes of files under one
// store/ tree: authored configuration (skills, agents, team contracts),
// runtime execution state (heartbeats, handoffs, append-only knowledge logs,
// queues), and derived caches (indexes). After this seam is adopted only the
// authored configuration lives in the repo; the runtime and cache classes flow
// to api-core/storage so they are operator-class, backed up by
// data-backup-manager via WellKnownScanner, and never appear in git status.
//
// DOC: docs/concepts/STORAGE_CLASSES.md
//
// Resolution asymmetry between production and tests is intentional: production
// derives RepoRoot via the repo contract (env or CWD walk); tests bypass that
// entirely via RootsForTest, which roots every class under t.TempDir().
package paths

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/storage"
	repocontract "github.com/vrooli/repo-contract-go"
)

// scenarioID is the api-core/storage tenant key for prompt-manager.
const scenarioID = "prompt-manager"

// appID is the api-core/storage app key shared by every Vrooli scenario.
const appID = "vrooli"

// Roots is the constructor-injected seam every store/handler takes when it
// needs to compose a filesystem path. Threaded from main.go through
// constructors; never a package global.
type Roots struct {
	// Config is the git-tracked authored store/ tree (skills, agents,
	// teams/<team>/{team,org,roles}.json, members/<m>/{RESPONSIBILITIES,
	// HEARTBEAT}.md, schemas, templates, topics, relations, actions, config,
	// world-*.json). Writers MUST NOT emit runtime-class files under this
	// root.
	Config string

	// RuntimeData is the api-core/storage ClassData root for prompt-manager:
	// heartbeats, handoffs, append-only jsonl logs, queues, the active-runs
	// registry, experiments, and centralized .backup artifacts. Mirrored
	// against the Config tree shape so a runtime path under RuntimeData is
	// the same relative path it would have had under Config, just rooted in
	// the operator data home.
	RuntimeData string

	// RuntimeCache is the api-core/storage ClassCache root for prompt-manager:
	// derived indexes only. Contract: ok to lose; everything here is
	// reconstructable from Config + RuntimeData.
	RuntimeCache string

	// RepoRoot is the repository root resolved via the repo contract. Used
	// by main.go to derive ScenariosDir without abusing a storage path.
	RepoRoot string

	// ScenariosDir is filepath.Join(RepoRoot, "scenarios"). Used by
	// discoverScenarioNames to enumerate sibling scenarios.
	ScenariosDir string
}

// Resolve constructs production Roots: Config from configDir (already absolute
// or relative-to-cwd at the call site), RuntimeData and RuntimeCache from
// api-core/storage with ProfileAuto, and RepoRoot from the repo contract
// (env-or-CWD). configDir is required; the storage classes and repo contract
// take their inputs from process state.
func Resolve(configDir string) (Roots, error) {
	if configDir == "" {
		return Roots{}, fmt.Errorf("paths.Resolve: configDir is required")
	}
	absConfig, err := filepath.Abs(configDir)
	if err != nil {
		return Roots{}, fmt.Errorf("paths.Resolve: abs configDir: %w", err)
	}

	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   appID,
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return Roots{}, fmt.Errorf("paths.Resolve: storage resolver: %w", err)
	}
	data, err := resolver.Path(storage.Options{ScenarioID: scenarioID}, storage.ClassData, "")
	if err != nil {
		return Roots{}, fmt.Errorf("paths.Resolve: runtime data root: %w", err)
	}
	cache, err := resolver.Path(storage.Options{ScenarioID: scenarioID}, storage.ClassCache, "")
	if err != nil {
		return Roots{}, fmt.Errorf("paths.Resolve: runtime cache root: %w", err)
	}

	repoRoot, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		return Roots{}, fmt.Errorf("paths.Resolve: repo root: %w", err)
	}

	return Roots{
		Config:       absConfig,
		RuntimeData:  data,
		RuntimeCache: cache,
		RepoRoot:     repoRoot,
		ScenariosDir: filepath.Join(repoRoot, "scenarios"),
	}, nil
}

// RootsForRepoStoreTest returns Roots whose Config points at the supplied
// real repo store path (e.g. "../../store") while RuntimeData, RuntimeCache,
// RepoRoot and ScenariosDir are stubbed under t.TempDir(). Used by tests that
// must read real authored configuration but must not write runtime artifacts
// into the repo tree.
func RootsForRepoStoreTest(t *testing.T, configDir string) Roots {
	t.Helper()
	base := t.TempDir()
	return Roots{
		Config:       configDir,
		RuntimeData:  filepath.Join(base, "data"),
		RuntimeCache: filepath.Join(base, "cache"),
		RepoRoot:     filepath.Join(base, "repo"),
		ScenariosDir: filepath.Join(base, "repo", "scenarios"),
	}
}

// RootsForTest returns Roots rooted entirely under t.TempDir(). All four roots
// are distinct subdirectories and exist on entry, so tests can compose paths
// against any class without arranging fixtures. The repo contract is bypassed
// intentionally; tests that care about repo-root behavior should exercise the
// repo contract directly.
func RootsForTest(t *testing.T) Roots {
	t.Helper()
	base := t.TempDir()
	roots := Roots{
		Config:       filepath.Join(base, "config"),
		RuntimeData:  filepath.Join(base, "data"),
		RuntimeCache: filepath.Join(base, "cache"),
		RepoRoot:     filepath.Join(base, "repo"),
		ScenariosDir: filepath.Join(base, "repo", "scenarios"),
	}
	return roots
}

// BackupFor returns the centralized .backup location for a runtime-relative
// path. Every .backup writer in the scenario routes through this helper so
// backup artifacts never pollute the config tree (CD-3 in the implementation
// plan). The suffix argument is the caller's choice of disambiguator
// (timestamp, content hash, etc.) and is appended after a hyphen; empty
// suffix means a plain ".backup" sibling under the backups root.
func (r Roots) BackupFor(rel, suffix string) string {
	clean := filepath.Clean(rel)
	name := clean + ".backup"
	if suffix != "" {
		name = name + "-" + suffix
	}
	return filepath.Join(r.RuntimeData, "backups", name)
}
