package discovery

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// Runtime captures how the sqlite resource discovers its manifest and source tree at runtime.
type Runtime struct {
	ExecutablePath     string
	InstalledManifest  string
	SourceManifestPath string
	SourceRoot         string
}

// ResolveSourceRoot returns the explicit source root when set, otherwise falls back to shared CLI source-root discovery.
func (r Runtime) ResolveSourceRoot() string {
	return cliutil.ResolveSourceRoot(r.SourceRoot, "SQLITE_CLI_SOURCE_ROOT", "VROOLI_CLI_SOURCE_ROOT")
}

// ModuleRoot returns the sqlite CLI module root when source checkout access is explicitly configured.
func (r Runtime) ModuleRoot() string {
	if root := r.ResolveSourceRoot(); root != "" {
		return filepath.Join(root, "cli")
	}
	return ""
}

// DiscoverRuntime derives the installed and development manifest lookup paths for the sqlite CLI.
func DiscoverRuntime(sourceRoot string) Runtime {
	executablePath, _ := os.Executable()
	runtime := Runtime{
		ExecutablePath: executablePath,
		SourceRoot:     strings.TrimSpace(sourceRoot),
	}
	if runtime.ExecutablePath != "" {
		runtime.InstalledManifest = cliutil.InstalledManifestPath(runtime.ExecutablePath)
	}
	if overrideRoot := cliutil.ResolveSourceRoot("", "SQLITE_CLI_SOURCE_ROOT", "VROOLI_CLI_SOURCE_ROOT"); overrideRoot != "" {
		runtime.SourceManifestPath = filepath.Join(overrideRoot, "resource.json")
	}
	return runtime
}
