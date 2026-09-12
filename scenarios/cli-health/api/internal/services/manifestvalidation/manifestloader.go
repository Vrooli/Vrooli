package manifestvalidation

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	repocontract "github.com/vrooli/repo-contract-go"
)

// FilesystemManifestLoader reads a scenario's cli/manifest.json from disk,
// resolving the canonical path through the repo contract so it tracks any
// future relocation of well-known paths.
type FilesystemManifestLoader struct {
	RepoRoot string
}

// NewFilesystemManifestLoader returns a loader rooted at the given repo dir.
func NewFilesystemManifestLoader(repoRoot string) *FilesystemManifestLoader {
	return &FilesystemManifestLoader{RepoRoot: repoRoot}
}

// Load returns (raw, absolute path, err). If the file doesn't exist Load
// returns os.ErrNotExist so the service can emit a manifest_missing warning.
//
// When ctx carries an explicit scenario root (WithScenarioPath) — used when the
// caller validates a scenario outside the repo scenarios/ tree, such as deep
// template validation's temp-generated scenario — the manifest is read from that
// directory's cli/manifest.json. Otherwise it resolves through the repo contract
// keyed by scenario name.
func (l *FilesystemManifestLoader) Load(ctx context.Context, scenario string) ([]byte, string, error) {
	var path string
	if root := scenarioPathFrom(ctx); root != "" {
		path = filepath.Join(root, "cli", "manifest.json")
	} else if isProjectTarget(scenario) {
		path = filepath.Join(l.RepoRoot, "cli", "manifest.json")
	} else {
		p, err := repocontract.ScenarioCLIManifestPath(l.RepoRoot, scenario)
		if err != nil {
			return nil, "", fmt.Errorf("resolve manifest path: %w", err)
		}
		path = p
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, path, err
	}
	return raw, path, nil
}
