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

func NewHandler(manifest hostreqkit.ToolManifest) hostreqkit.Handler {
	installer := newInstaller(manifest)
	return hostreqkit.InstallerHandler{Manifest: manifest, KindValue: hostreqspec.KindTool, InspectFunc: installer.Inspect, ApplyFunc: installer.Apply}
}

func newInstaller(manifest hostreqkit.ToolManifest) hostreqkit.GoInstallInstaller {
	return hostreqkit.GoInstallInstaller{
		Manifest: manifest, ModulePath: npmPackage, Version: versionRef(manifest),
		BinaryName: toolBinName, Kind: hostreqkit.InstallKindNPM,
		CacheDir: toolCacheDir, RecordArtifacts: cliinstall.RecordNPMToolInstallArtifacts,
	}
}

func versionRef(manifest hostreqkit.ToolManifest) string {
	if v := strings.TrimSpace(manifest.Version); v != "" {
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
