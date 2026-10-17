// Package protocgenes implements the host-tool handler for
// @bufbuild/protoc-gen-es, the TypeScript/JavaScript protobuf code generator
// used by packages/proto codegen.
//
// Install path: `npm install --prefix <cache>/node @bufbuild/protoc-gen-es@<pinned>`
// where <cache> is `${XDG_CACHE_HOME:-$HOME/.cache}/vrooli/protoc-plugins`.
// The handler then symlinks the npm-installed shim into ~/.local/bin/ so
// buf finds the plugin on PATH.
//
// # Sudo-context handling
//
// When the vrooli process is running as root (typically `sudo vrooli
// setup`), naively spawning `npm install` inherits HOME=/root, so the
// install lands in /root/.cache instead of the operator's cache and the
// post-install symlink is created root-owned. To avoid this, all
// per-user operations (npm install, mkdir, ln -sfn) run through
// hostreqkit.RunAsInvokingUser, which drops privileges back to
// $SUDO_USER when we're root and runs natively otherwise. Cache and link
// paths use InvokingUserHomeDir so HOME resolves to the operator's home
// regardless of sudo's env scrubbing.
package protocgenes

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
	npmPackage     = "@bufbuild/protoc-gen-es"
	pluginBinName  = "protoc-gen-es"
	defaultVersion = "2.12.0"
)

func NewHandler(manifest hostreqkit.ToolManifest) hostreqkit.Handler {
	installer := newInstaller(manifest)
	return hostreqkit.InstallerHandler{Manifest: manifest, KindValue: hostreqspec.KindTool, InspectFunc: installer.Inspect, ApplyFunc: installer.Apply}
}

func newInstaller(manifest hostreqkit.ToolManifest) hostreqkit.GoInstallInstaller {
	return hostreqkit.GoInstallInstaller{
		Manifest: manifest, ModulePath: npmPackage, Version: versionRef(manifest),
		BinaryName: pluginBinName, Kind: hostreqkit.InstallKindNPM,
		CacheDir: pluginCacheDir, RecordArtifacts: cliinstall.RecordNPMToolInstallArtifacts,
	}
}

func versionRef(manifest hostreqkit.ToolManifest) string {
	if v := strings.TrimSpace(manifest.Version); v != "" {
		return v
	}
	return defaultVersion
}

// pluginCacheDir resolves the per-user cache directory the protoc plugins
// are installed into. Honors $XDG_CACHE_HOME, falling back to
// InvokingUserHomeDir/.cache (correct under sudo where $HOME=/root).
func pluginCacheDir() (string, error) {
	if cache := strings.TrimSpace(os.Getenv("XDG_CACHE_HOME")); cache != "" {
		return filepath.Join(cache, "vrooli", "protoc-plugins", "node"), nil
	}
	home, err := hostreqkit.InvokingUserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".cache", "vrooli", "protoc-plugins", "node"), nil
}

// ensureSymlinkOnPath links <prefix>/node_modules/.bin/<plugin> into
// ~/.local/bin/<plugin>. mkdir and ln run via RunAsInvokingUser so the
// resulting directory and symlink are owned by the operator (not root)
// when invoked under sudo.
