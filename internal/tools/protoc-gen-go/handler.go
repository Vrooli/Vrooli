// Package protocgengo implements the host-tool handler for protoc-gen-go,
// the Go protobuf code generator used by packages/proto codegen.
//
// Install path: `go install google.golang.org/protobuf/cmd/protoc-gen-go@<pinned>`
// where the pinned version comes from the manifest. The binary lands in
// the user's $GOBIN (or $HOME/go/bin if unset). Vrooli's PATH conventions
// require ~/.local/bin to be on PATH; the handler symlinks the installed
// binary there so the plugin is discoverable by buf without further env
// tweaks.
//
// # Sudo-context handling
//
// When the vrooli process is running as root (typically `sudo vrooli
// setup`), naively spawning `go install` inherits HOME=/root and writes
// the binary to /root/go/bin instead of the operator's home — and the
// resulting symlink ends up root-owned, which the operator's normal-user
// processes cannot maintain. To avoid this, all per-user operations
// (go install, mkdir, ln -sfn) run through hostreqkit.RunAsInvokingUser,
// which drops privileges back to $SUDO_USER when we're root and runs
// natively otherwise. Path resolution uses InvokingUserHomeDir so HOME
// resolves to the operator's home regardless of sudo's env scrubbing.
package protocgengo

import (
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	goModule       = "google.golang.org/protobuf/cmd/protoc-gen-go"
	pluginBinName  = "protoc-gen-go"
	defaultVersion = "v1.36.11"
)

func NewHandler(manifest hostreqkit.ToolManifest) hostreqkit.Handler {
	installer := newInstaller(manifest)
	return hostreqkit.InstallerHandler{Manifest: manifest, KindValue: hostreqspec.KindTool, InspectFunc: installer.Inspect, ApplyFunc: installer.Apply}
}

func newInstaller(manifest hostreqkit.ToolManifest) hostreqkit.GoInstallInstaller {
	return hostreqkit.GoInstallInstaller{
		Manifest: manifest, ModulePath: goModule, Version: hostreqkit.VersionOrDefault(manifest.Version, defaultVersion),
		BinaryName: pluginBinName, Kind: hostreqkit.InstallKindGo,
		RecordArtifacts: cliinstall.RecordGoToolInstallArtifacts,
	}
}
