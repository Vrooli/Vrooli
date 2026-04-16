package version

import (
	"github.com/vrooli/cli-core/cliutil"
)

// Manifest loads runtime metadata from the installed sibling manifest first and
// falls back to an explicit development/source override when configured.
type Manifest struct {
	InstalledPath string
	SourcePath    string
}

// Load reads the installed sibling manifest first. A source manifest is only
// consulted when an explicit development/source override has been configured.
func (m Manifest) Load() ([]byte, error) {
	return cliutil.LoadRuntimeManifest(m.InstalledPath, m.SourcePath)
}
