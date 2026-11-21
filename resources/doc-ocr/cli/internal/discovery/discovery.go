package discovery

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// Runtime is the default home for runtime and source-root discovery for
// native-cli resources.
type Runtime struct {
	ExecutablePath     string
	InstalledManifest  string
	SourceManifestPath string
	SourceRoot         string
}

// DiscoverRuntime derives the installed and development manifest lookup paths
// for the resource CLI.
func DiscoverRuntime(sourceRoot string) Runtime {
	executablePath, _ := os.Executable()
	runtime := Runtime{
		ExecutablePath: executablePath,
		SourceRoot:     strings.TrimSpace(sourceRoot),
	}
	if runtime.ExecutablePath != "" {
		runtime.InstalledManifest = cliutil.InstalledManifestPath(runtime.ExecutablePath)
	}
	if overrideRoot := cliutil.ResolveSourceRoot(runtime.SourceRoot, "VROOLI_CLI_SOURCE_ROOT"); overrideRoot != "" {
		runtime.SourceManifestPath = filepath.Join(overrideRoot, "resource.json")
	}
	return runtime
}
