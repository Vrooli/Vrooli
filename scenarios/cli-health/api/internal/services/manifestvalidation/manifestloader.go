package manifestvalidation

import (
	"context"
	"fmt"
	"os"

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
func (l *FilesystemManifestLoader) Load(_ context.Context, scenario string) ([]byte, string, error) {
	path, err := repocontract.ScenarioCLIManifestPath(l.RepoRoot, scenario)
	if err != nil {
		return nil, "", fmt.Errorf("resolve manifest path: %w", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, path, err
	}
	return raw, path, nil
}
