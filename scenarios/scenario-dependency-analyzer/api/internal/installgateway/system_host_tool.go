package installgateway

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	hostapp "github.com/vrooli/vrooli/internal/app/host"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/tools"
)

// System packages retain the platform's package-manager policy. The gateway
// approves a declared tool identity, never arbitrary packages, paths or argv.
func resolveSystemHostTool(root, name, pkg, version string) (Resolution, error) {
	if err := validateScenarioName(name); err != nil {
		return Resolution{}, err
	}
	if name != pkg || version != "" {
		return Resolution{}, fmt.Errorf("system host-tool requires the declared tool name and no portable version pin; the host repository selects its version")
	}
	path := filepath.Join(root, "internal", "tools", name, "tool.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return Resolution{}, err
	}
	embedded, err := tools.Manifests.ReadFile(name + "/tool.json")
	if err != nil || !bytes.Equal(raw, embedded) {
		return Resolution{}, fmt.Errorf("host-tool manifest is not current in the control-plane owner; rebuild Scenario Dependency Analyzer")
	}
	var manifest hostreqkit.ToolManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return Resolution{}, err
	}
	if manifest.Name != name || manifest.Manual || manifest.Handler != "" || manifest.SourceType() != "package" || (manifest.DefaultPackage == "" && len(manifest.Packages) == 0) {
		return Resolution{}, fmt.Errorf("tool %q does not declare a system-package acquisition route", name)
	}
	return Resolution{RepositoryRoot: root, SurfaceRoot: filepath.Dir(path), ManifestPath: path, PackageName: name, PackageManager: "native-host-tool", HostManifestSHA256: fmt.Sprintf("%x", sha256.Sum256(raw)), Argv: []string{"vrooli", "host", "install", name, "--json"}, Profile: InstallProfile{Governance: "scenario-dependency-analyzer", LifecycleMode: "system-host-tool"}}, nil
}

func provisionSystemHostTool(ctx context.Context, r Resolution) (string, error) {
	want, err := resolveSystemHostTool(r.RepositoryRoot, r.PackageName, r.PackageName, "")
	if err != nil {
		return "", err
	}
	if !reflect.DeepEqual(r, want) {
		return "", fmt.Errorf("system host-tool resolution changed before installation")
	}
	status, err := (hostapp.Service{}).Install(ctx, r.PackageName, hostapp.InstallOptions{})
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(hostapp.HostInstallStatusResponse(status))
	if err != nil {
		return "", err
	}
	if !hostapp.HostInstallOK(status) {
		return string(raw), fmt.Errorf("host owner did not install %s: %s", r.PackageName, status.ExecutionState)
	}
	return string(raw), nil
}
