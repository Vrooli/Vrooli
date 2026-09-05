package version

import "github.com/vrooli/cli-core/cliutil"

// Manifest is the default home for manifest/build metadata helpers for
// native-cli resources.
type Manifest struct {
	InstalledPath string
	SourcePath    string
}

// Load reads runtime metadata from the installed sibling manifest first and
// falls back to the explicit development/source path when configured.
func (m Manifest) Load() ([]byte, error) {
	return cliutil.LoadRuntimeManifest(m.InstalledPath, m.SourcePath)
}
