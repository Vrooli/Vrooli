package installgateway

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/vrooli/envkit-go"
	"github.com/vrooli/vrooli/internal/tools/hyperframes"
)

// resolveHostTool is deliberately a closed owner surface, not an arbitrary path
// or another interpretation of tools/<shared-package>.
func resolveHostTool(root, name, ecosystem, pkg, version string) (Resolution, error) {
	if name != "hyperframes" || ecosystem != "npm" || pkg != "hyperframes" || version != hyperframes.Version {
		return Resolution{}, fmt.Errorf("host-tool requires npm/hyperframes@%s and owner hyperframes", hyperframes.Version)
	}
	source := filepath.Join(root, "internal", "tools", "hyperframes")
	if err := hyperframes.ValidateSource(source); err != nil {
		return Resolution{}, err
	}
	argv := []string{"npm", "ci", "--ignore-scripts", "--no-audit", "--no-fund"}
	return Resolution{HostTool: true, RepositoryRoot: root, SurfaceRoot: source, PackageManager: "npm", ManifestPath: filepath.Join(source, "package.json"), PackageName: pkg, ExactVersion: true, Argv: argv, Profile: SafeProfileFor("npm", argv)}, nil
}

// ResolveHostToolRefresh is an explicit source-only operation. Frozen runtime
// reproduction remains closed to the currently committed version.
func ResolveHostToolRefresh(root, name, ecosystem, pkg, version string) (Resolution, error) {
	if name != "hyperframes" || ecosystem != "npm" || pkg != "hyperframes" || !exactNpmVersion.MatchString(version) {
		return Resolution{}, fmt.Errorf("host-tool source refresh requires npm/hyperframes and an exact semantic version")
	}
	source := filepath.Join(root, "internal", "tools", "hyperframes")
	if err := hyperframes.ValidateSource(source); err != nil {
		return Resolution{}, fmt.Errorf("host-tool source closure requires recovery/review before refresh: %w", err)
	}
	argv := []string{"npm", "install", "--package-lock-only", "--save-exact", "--ignore-scripts", "--no-audit", "--no-fund", "hyperframes@" + version}
	return Resolution{HostToolRefresh: true, HostToolTargetVersion: version, RepositoryRoot: root, SurfaceRoot: source, PackageManager: "npm", ManifestPath: filepath.Join(source, "package.json"), PackageName: pkg, ExactVersion: true, Argv: argv, Profile: SafeProfileFor("npm", argv)}, nil
}

func refreshHostToolSource(ctx context.Context, r Resolution) (string, error) {
	want, err := ResolveHostToolRefresh(r.RepositoryRoot, "hyperframes", "npm", "hyperframes", r.HostToolTargetVersion)
	if err != nil {
		return "", err
	}
	if r.SurfaceRoot != want.SurfaceRoot || r.Command() != want.Command() || !r.ExactVersion {
		return "", fmt.Errorf("host-tool source refresh resolution changed")
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	return hyperframes.RefreshSource(ctx, r.SurfaceRoot, r.HostToolTargetVersion, func(ctx context.Context, dir string) (string, error) { return hostToolInstall(ctx, dir, want.Argv) })
}

// hostToolInstall is an injected command seam; governance remains in the API
// caller and all package-manager execution stays inside this owner.
var hostToolInstall = func(ctx context.Context, dir string, argv []string) (string, error) {
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = dir
	cmd.Env = envkit.Toolchain(envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, nil), envkit.ToolchainOptions{})
	cmd.WaitDelay = 5 * time.Second
	var output hostToolOutput
	cmd.Stdout, cmd.Stderr = &output, &output
	err := cmd.Run()
	return output.String(), err
}

func provisionHostTool(ctx context.Context, r Resolution) (string, error) {
	// Re-resolve so a manually constructed Resolution cannot change argv or source.
	want, err := resolveHostTool(r.RepositoryRoot, "hyperframes", "npm", "hyperframes", hyperframes.Version)
	if err != nil {
		return "", err
	}
	if r.SurfaceRoot != want.SurfaceRoot || r.Command() != want.Command() {
		return "", fmt.Errorf("host tool provision resolution changed")
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	return hyperframes.Provision(ctx, r.SurfaceRoot, func(ctx context.Context, dir string) (string, error) { return hostToolInstall(ctx, dir, want.Argv) })
}
