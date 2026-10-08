// Package scenariospec resolves the on-disk facts data-backup-manager needs to
// derive a scenario's backup targets: which storage engines the scenario
// declares (read from its .vrooli/service.json) and whether its durable data
// directory exists on disk.
//
// It is deliberately read-only and best-effort: an unreadable or malformed
// manifest yields zero derivable facts rather than an error, so one bad scenario
// never breaks the safety substrate. The only hard error is an unresolvable
// repository root (the whole derivation is impossible without it).
//
// The repo-root and home resolvers are package-level seams so tests can point at
// temp trees; production wires the repo-contract authority and the OS home.
package scenariospec

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

// storageAppID is the api-core storage convention: per-scenario class roots live
// under "<class-root>/<app>/<scenario>", and every Vrooli scenario uses the
// default app id "vrooli" (packages/api-core/storage/resolver.go defaultAppID).
// data-backup-manager does not import api-core/storage, so the convention is
// mirrored here with this comment as the cross-reference.
const storageAppID = "vrooli"

// Facts are the derivable storage facts for one scenario.
type Facts struct {
	// UsesPostgres is true when the scenario's service.json declares an enabled
	// Postgres resource dependency — the signal that its lifecycle Postgres
	// database (vrooli_<scenario>) is a backup target.
	UsesPostgres bool
	// DataDir is the conventional absolute durable-data directory for the
	// scenario (~/.vrooli/data/vrooli/<scenario>). Empty when the home/data root
	// cannot be resolved.
	DataDir string
	// DataDirPresent is true when DataDir exists on disk and is a non-empty
	// directory — the self-validating signal that there is filesystem state worth
	// backing up (and that the conventional layout holds for this scenario).
	DataDirPresent bool
}

// Inspector resolves Facts for a scenario.
type Inspector struct {
	// repoRoot resolves the repository root (where scenarios/<s>/.vrooli lives).
	repoRoot func() (string, error)
	// homeDir resolves the operator home (where ~/.vrooli/data lives).
	homeDir func() (string, error)
}

// NewInspector returns an Inspector wired to the production repo-contract repo
// root resolver and the OS user home (data-backup-manager runs as the operator,
// consistent with the discovery domain's resolvers).
func NewInspector() *Inspector {
	return &Inspector{
		repoRoot: repocontract.FindRepoRootFromEnvOrCWD,
		homeDir:  os.UserHomeDir,
	}
}

// Inspect reads the scenario's service.json and on-disk data layout. A blank
// scenario or an unresolvable repository root is an error; everything else is
// best-effort (a missing/malformed manifest simply yields UsesPostgres=false).
func (i *Inspector) Inspect(_ context.Context, scenario string) (Facts, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return Facts{}, errors.New("scenario is required")
	}

	root, err := i.repoRoot()
	if err != nil || strings.TrimSpace(root) == "" {
		return Facts{}, errors.New("cannot resolve repository root to read the scenario manifest")
	}

	servicePath, err := repocontract.ScenarioServiceManifestPath(root, scenario)
	if err != nil {
		return Facts{}, err
	}
	facts := Facts{
		UsesPostgres: declaresEnabledPostgres(servicePath),
	}

	if dataDir, ok := i.scenarioDataDir(scenario); ok {
		facts.DataDir = dataDir
		facts.DataDirPresent = isNonEmptyDir(dataDir)
	}
	return facts, nil
}

// scenarioDataDir resolves the conventional durable-data directory for a
// scenario. Returns ok=false when the home/data root cannot be resolved (the
// filesystem target is then simply not derivable, not an error).
func (i *Inspector) scenarioDataDir(scenario string) (string, bool) {
	home, err := i.homeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "", false
	}
	dataRoot, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyData)
	if err != nil || strings.TrimSpace(dataRoot) == "" {
		return "", false
	}
	return filepath.Join(dataRoot, storageAppID, scenario), true
}

// declaresEnabledPostgres reports whether the service.json at path declares an
// enabled resource dependency of type postgres. Unreadable/malformed manifests
// report false (best-effort — never an error).
func declaresEnabledPostgres(manifestPath string) bool {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return false
	}
	var doc struct {
		Dependencies struct {
			Resources map[string]struct {
				Enabled bool   `json:"enabled"`
				Type    string `json:"type"`
			} `json:"resources"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return false
	}
	for name, res := range doc.Dependencies.Resources {
		// Match on the declared type, falling back to the resource key so a
		// scenario that names the dependency "postgres" without a type field is
		// still detected (the type field is conventional but not guaranteed).
		kind := strings.ToLower(strings.TrimSpace(res.Type))
		if kind == "" {
			kind = strings.ToLower(strings.TrimSpace(name))
		}
		if res.Enabled && kind == "postgres" {
			return true
		}
	}
	return false
}

// isNonEmptyDir reports whether path is an existing directory containing at
// least one entry. A non-existent path, a non-directory, or an unreadable
// directory all report false — the filesystem target is registered only when
// there is genuinely durable data to back up.
func isNonEmptyDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	return len(entries) > 0
}
