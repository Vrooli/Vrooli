package discovery

import (
	"encoding/json"
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
	if overrideRoot := cliutil.ResolveSourceRoot(runtime.SourceRoot, "VROOLI_CLI_SOURCE_ROOT", "RESOURCE_ROOT"); overrideRoot != "" {
		runtime.SourceRoot = overrideRoot
		runtime.SourceManifestPath = filepath.Join(overrideRoot, "resource.json")
	}
	if metadataRoot := sourceRootFromBuildMetadata(runtime.ExecutablePath); metadataRoot != "" && !manifestExists(runtime.SourceManifestPath) {
		runtime.SourceRoot = metadataRoot
		runtime.SourceManifestPath = filepath.Join(metadataRoot, "resource.json")
	}
	return runtime
}

func manifestExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

type buildMetadata struct {
	ModulePath string `json:"module_path"`
}

// sourceRootFromBuildMetadata recovers the checked-out resource root for an
// installed CLI. The control plane writes module_path beside every installed
// resource command, so development artifacts can be resolved without
// requiring callers to inject a shell-specific RESOURCE_ROOT.
func sourceRootFromBuildMetadata(executablePath string) string {
	if strings.TrimSpace(executablePath) == "" {
		return ""
	}
	data, err := os.ReadFile(executablePath + ".build.meta")
	if err != nil {
		return ""
	}
	var metadata buildMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return ""
	}
	modulePath := strings.TrimSpace(metadata.ModulePath)
	if modulePath == "" {
		return ""
	}
	root, err := filepath.Abs(filepath.Dir(modulePath))
	if err != nil {
		return ""
	}
	return root
}
