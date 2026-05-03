// Package protocgenes implements the host-tool handler for
// @bufbuild/protoc-gen-es, the TypeScript/JavaScript protobuf code generator
// used by packages/proto codegen.
//
// Install path: `npm install --prefix <cache>/node @bufbuild/protoc-gen-es@<pinned>`
// where <cache> is `${XDG_CACHE_HOME:-$HOME/.cache}/vrooli/protoc-plugins`.
// The handler then symlinks the npm-installed shim into ~/.local/bin/ so
// buf finds the plugin on PATH.
package protocgenes

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	npmPackage     = "@bufbuild/protoc-gen-es"
	pluginBinName  = "protoc-gen-es"
	defaultVersion = "2.12.0"
)

// UserHomeDirFn is overridable for tests.
var UserHomeDirFn = os.UserHomeDir

type handler struct {
	manifest hostreqkit.ToolManifest
}

func NewHandler(manifest hostreqkit.ToolManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}

func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindTool }

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	status.Command, status.Installed = hostreqkit.ResolveCommand(h.manifest.Commands)
	status.SupportClass = hostreqkit.SupportSupported
	status.Notes = append(status.Notes, h.manifest.InstallHint)
	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status
	}
	if !hostreqkit.CommandAvailable("npm") {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "npm not on PATH; install Node.js first (handled by the `node` host tool)")
		return status
	}
	status.InstallSupported = true
	status.PackageName = npmPackage + "@" + h.versionRef()
	if status.Installed {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Version = hostreqkit.ReadVersion(status.Command, h.manifest.VersionArgs)
	}
	return status
}

func (h handler) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	if status.Installed {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		return status, nil
	}
	switch status.SupportClass {
	case hostreqkit.SupportManualOnly:
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status, nil
	case hostreqkit.SupportUnsupported:
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		return status, nil
	case hostreqkit.SupportNotApplicable:
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		return status, nil
	}

	pkgRef := npmPackage + "@" + h.versionRef()
	prefix, err := pluginCacheDir()
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldInstall
		status.Notes = append(status.Notes, fmt.Sprintf("dry-run: npm install --prefix %s %s", prefix, pkgRef))
		return status, nil
	}

	if err := os.MkdirAll(prefix, 0o755); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, fmt.Sprintf("create %s: %v", prefix, err))
		return status, nil
	}
	args := []string{"install", "--prefix", prefix, "--no-audit", "--no-fund", "--silent", pkgRef}
	if err := hostreqkit.RunInstallCommand("npm", args, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}
	if err := ensureSymlinkOnPath(prefix); err != nil {
		status.Notes = append(status.Notes, "post-install symlink: "+err.Error())
	}

	status.Command, status.Installed = hostreqkit.ResolveCommand(h.manifest.Commands)
	if !status.Installed {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, fmt.Sprintf("npm install %s succeeded but %s is not on PATH; ensure ~/.local/bin is on PATH", pkgRef, pluginBinName))
		return status, nil
	}
	status.ExecutionState = hostreqkit.ExecutionInstalled
	status.Version = hostreqkit.ReadVersion(status.Command, h.manifest.VersionArgs)
	return status, nil
}

func (h handler) versionRef() string {
	if v := strings.TrimSpace(h.manifest.Version); v != "" {
		return v
	}
	return defaultVersion
}

// pluginCacheDir resolves the per-user cache directory the protoc plugins
// are installed into. Honors $XDG_CACHE_HOME, falling back to ~/.cache.
func pluginCacheDir() (string, error) {
	if cache := strings.TrimSpace(os.Getenv("XDG_CACHE_HOME")); cache != "" {
		return filepath.Join(cache, "vrooli", "protoc-plugins", "node"), nil
	}
	home, err := UserHomeDirFn()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".cache", "vrooli", "protoc-plugins", "node"), nil
}

func ensureSymlinkOnPath(prefix string) error {
	source := filepath.Join(prefix, "node_modules", ".bin", pluginBinName)
	if _, err := os.Stat(source); err != nil {
		return fmt.Errorf("post-install: %s missing at %s: %w", pluginBinName, source, err)
	}
	home, err := UserHomeDirFn()
	if err != nil {
		return fmt.Errorf("resolve home dir: %w", err)
	}
	linkDir := filepath.Join(home, ".local", "bin")
	if mkErr := os.MkdirAll(linkDir, 0o755); mkErr != nil {
		return fmt.Errorf("ensure %s: %w", linkDir, mkErr)
	}
	link := filepath.Join(linkDir, pluginBinName)
	existing, readErr := os.Readlink(link)
	if readErr == nil && existing == source {
		return nil
	}
	if _, statErr := os.Lstat(link); statErr == nil {
		if rmErr := os.Remove(link); rmErr != nil {
			return fmt.Errorf("replace stale symlink %s: %w", link, rmErr)
		}
	}
	if symErr := os.Symlink(source, link); symErr != nil {
		return fmt.Errorf("create symlink %s -> %s: %w", link, source, symErr)
	}
	return nil
}
