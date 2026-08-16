// Package quint implements the host-tool handler for the Quint formal
// specification CLI used by temporal-flow model validation.
//
// Quint is distributed as the npm package @informalsystems/quint. The handler
// installs a pinned package into a Vrooli-owned per-user cache and symlinks the
// shim into ~/.local/bin so scenario/template validation can invoke `quint`
// without adding it to individual UI package manifests.
package quint

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	npmPackage     = "@informalsystems/quint"
	toolBinName    = "quint"
	defaultVersion = "0.32.0"
)

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
	status.Command, status.Installed = hostreqkit.ResolveCommandForInvokingUser(h.manifest.Commands)
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
	prefix, err := toolCacheDir()
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

	if mkErr := hostreqkit.RunAsInvokingUser("mkdir", []string{"-p", prefix}, opts); mkErr != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, fmt.Sprintf("create %s: %v", prefix, mkErr))
		return status, nil
	}
	args := []string{"install", "--prefix", prefix, "--no-audit", "--no-fund", "--silent", pkgRef}
	if err := hostreqkit.RunAsInvokingUser("npm", args, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}
	if err := ensureSymlinkOnPath(prefix, opts); err != nil {
		status.Notes = append(status.Notes, "post-install symlink: "+err.Error())
	}
	if err := recordNPMToolInstall(prefix); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "record install inventory: "+err.Error())
		return status, nil
	}

	status.Command, status.Installed = hostreqkit.ResolveCommandForInvokingUser(h.manifest.Commands)
	if !status.Installed {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, fmt.Sprintf("npm install %s succeeded but %s is not on PATH; ensure ~/.local/bin is on PATH", pkgRef, toolBinName))
		return status, nil
	}
	status.ExecutionState = hostreqkit.ExecutionInstalled
	status.Version = hostreqkit.ReadVersion(status.Command, h.manifest.VersionArgs)
	return status, nil
}

func recordNPMToolInstall(prefix string) error {
	home, err := hostreqkit.InvokingUserHomeDir()
	if err != nil {
		return err
	}
	link := filepath.Join(home, ".local", "bin", toolBinName)
	source := filepath.Join(prefix, "node_modules", ".bin", toolBinName)
	return cliinstall.RecordToolArtifacts(home,
		cliinstall.InstallEntry{Scope: cliinstall.ScopeRuntime, Kind: cliinstall.EntryDirectory, Path: prefix, Prefix: prefix},
		cliinstall.InstallEntry{Scope: cliinstall.ScopeRuntime, Kind: cliinstall.EntryFile, Path: source, Prefix: filepath.Dir(source)},
		cliinstall.InstallEntry{Scope: cliinstall.ScopeRuntime, Kind: cliinstall.EntryFile, Path: link, Prefix: home},
	)
}

func (h handler) versionRef() string {
	if v := strings.TrimSpace(h.manifest.Version); v != "" {
		return v
	}
	return defaultVersion
}

func toolCacheDir() (string, error) {
	if cache := strings.TrimSpace(os.Getenv("XDG_CACHE_HOME")); cache != "" {
		return filepath.Join(cache, "vrooli", "formal-tools", "quint", "node"), nil
	}
	home, err := hostreqkit.InvokingUserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".cache", "vrooli", "formal-tools", "quint", "node"), nil
}

func ensureSymlinkOnPath(prefix string, opts hostreqkit.EnsureOptions) error {
	source := filepath.Join(prefix, "node_modules", ".bin", toolBinName)
	if _, err := os.Stat(source); err != nil {
		return fmt.Errorf("post-install: %s missing at %s: %w", toolBinName, source, err)
	}
	home, err := hostreqkit.InvokingUserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home dir: %w", err)
	}
	linkDir := filepath.Join(home, ".local", "bin")
	link := filepath.Join(linkDir, toolBinName)
	if mkErr := hostreqkit.RunAsInvokingUser("mkdir", []string{"-p", linkDir}, opts); mkErr != nil {
		return fmt.Errorf("ensure %s: %w", linkDir, mkErr)
	}
	if symErr := hostreqkit.RunAsInvokingUser("ln", []string{"-sfn", source, link}, opts); symErr != nil {
		return fmt.Errorf("create symlink %s -> %s: %w", link, source, symErr)
	}
	return nil
}
