package repocontract

import (
	"path/filepath"
	"strings"
)

// SourceRootPointerPath returns the operator runtime-home file that records
// the source checkout associated with an installed Vrooli control plane. The
// path is OS-native even though the runtime-home default is conventionally
// written as ~/.vrooli in documentation.
func SourceRootPointerPath(home string) string {
	home = strings.TrimSpace(home)
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".vrooli", "source-root")
}
