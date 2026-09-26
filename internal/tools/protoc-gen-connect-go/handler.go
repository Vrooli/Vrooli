// Package protocgenconnectgo implements the host-tool handler for
// protoc-gen-connect-go, the Connect-RPC code generator used by
// packages/proto codegen.
//
// Install path: `go install connectrpc.com/connect/cmd/protoc-gen-connect-go@<pinned>`
// where the pinned version comes from the manifest. The binary lands in
// the user's $GOBIN (or $HOME/go/bin if unset). The handler symlinks the
// installed binary into ~/.local/bin so buf finds it on PATH.
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
package protocgenconnectgo

import (
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	goModule       = "connectrpc.com/connect/cmd/protoc-gen-connect-go"
	pluginBinName  = "protoc-gen-connect-go"
	defaultVersion = "v1.19.2"
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
