package pairing

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/scopecatalog"
	"vrooli-bridge/internal/session"
)

func TestEveryEnforcedTransportCapabilityIsPresetReachable(t *testing.T) {
	root, err := repositoryRoot()
	require.NoError(t, err)
	catalog, err := scopecatalog.Build(root)
	require.NoError(t, err)
	presets := PermissionPresets(catalog)
	for _, enforced := range []string{session.TransportScope} {
		reachable := false
		for _, preset := range presets {
			for _, scope := range preset.Scopes {
				if scope == enforced {
					reachable = true
				}
			}
		}
		require.Truef(t, reachable, "enforced transport capability %q is unreachable from every permission preset", enforced)
	}
}

func repositoryRoot() (string, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for directory := filepath.Clean(workingDirectory); ; directory = filepath.Dir(directory) {
		if isDirectory(filepath.Join(directory, "packages")) && isDirectory(filepath.Join(directory, "scenarios")) {
			return directory, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", os.ErrNotExist
		}
	}
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
