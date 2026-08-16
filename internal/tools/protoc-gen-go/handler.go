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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	goModule       = "google.golang.org/protobuf/cmd/protoc-gen-go"
	pluginBinName  = "protoc-gen-go"
	defaultVersion = "v1.36.11"
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
	// ResolveCommandForInvokingUser also probes the operator's
	// ~/.local/bin and ~/go/bin — sudo'd processes inherit root's PATH
	// which excludes those, so a plain LookPath would false-negative
	// even when the binary is right there in the operator's home.
	status.Command, status.Installed = hostreqkit.ResolveCommandForInvokingUser(h.manifest.Commands)
	status.SupportClass = hostreqkit.SupportSupported
	status.Notes = append(status.Notes, h.manifest.InstallHint)
	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status
	}

	// Install is supported anywhere Go itself is supported. We don't carve
	// per-OS branches because `go install` is itself cross-platform.
	if !hostreqkit.CommandAvailable("go") {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "go toolchain not on PATH; install Go first (handled by the `go` host tool)")
		return status
	}
	status.InstallSupported = true
	status.PackageName = goModule + "@" + h.versionRef()

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

	pkgRef := goModule + "@" + h.versionRef()
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldInstall
		status.Notes = append(status.Notes, "dry-run: go install "+pkgRef)
		return status, nil
	}

	// Run as the invoking user — go install writes to $GOBIN / $HOME/go/bin,
	// which must be the operator's home, not /root, when invoked under sudo.
	if err := hostreqkit.RunAsInvokingUser("go", []string{"install", pkgRef}, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}

	if err := ensureSymlinkOnPath(opts); err != nil {
		// Symlink failure is not fatal — go install put the binary in $GOBIN/
		// or $HOME/go/bin, which the user may already have on PATH. Surface
		// the issue as a note so operators can debug PATH if `which protoc-gen-go`
		// fails after install.
		status.Notes = append(status.Notes, "post-install symlink: "+err.Error())
	}
	if err := recordGoToolInstall(); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "record install inventory: "+err.Error())
		return status, nil
	}

	status.Command, status.Installed = hostreqkit.ResolveCommandForInvokingUser(h.manifest.Commands)
	if !status.Installed {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, fmt.Sprintf("go install %s succeeded but %s is not on PATH; ensure $GOBIN (or $HOME/go/bin) is on PATH or rerun setup", pkgRef, pluginBinName))
		return status, nil
	}
	status.ExecutionState = hostreqkit.ExecutionInstalled
	status.Version = hostreqkit.ReadVersion(status.Command, h.manifest.VersionArgs)
	return status, nil
}

func recordGoToolInstall() error {
	source, err := goInstallTarget()
	if err != nil {
		return err
	}
	home, err := hostreqkit.InvokingUserHomeDir()
	if err != nil {
		return err
	}
	link := filepath.Join(home, ".local", "bin", pluginBinName)
	return cliinstall.RecordToolArtifacts(home,
		cliinstall.InstallEntry{Scope: cliinstall.ScopeRuntime, Kind: cliinstall.EntryBinary, Path: source, Prefix: filepath.Dir(source)},
		cliinstall.InstallEntry{Scope: cliinstall.ScopeRuntime, Kind: cliinstall.EntryFile, Path: link, Prefix: home},
	)
}

func (h handler) versionRef() string {
	if v := strings.TrimSpace(h.manifest.Version); v != "" {
		return v
	}
	return defaultVersion
}

// ensureSymlinkOnPath links $GOBIN-or-default/<bin> into ~/.local/bin/<bin>
// so the plugin is discoverable by buf via PATH. mkdir and ln run via
// RunAsInvokingUser so the resulting directory and symlink are owned by
// the operator (not root) when invoked under sudo. Idempotent: `ln -sfn`
// atomically replaces an existing link or non-directory file.
func ensureSymlinkOnPath(opts hostreqkit.EnsureOptions) error {
	source, err := goInstallTarget()
	if err != nil {
		return err
	}
	if _, statErr := os.Stat(source); statErr != nil {
		return fmt.Errorf("post-install: %s not found at %s: %w", pluginBinName, source, statErr)
	}
	home, err := hostreqkit.InvokingUserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home dir: %w", err)
	}
	linkDir := filepath.Join(home, ".local", "bin")
	link := filepath.Join(linkDir, pluginBinName)

	if mkErr := hostreqkit.RunAsInvokingUser("mkdir", []string{"-p", linkDir}, opts); mkErr != nil {
		return fmt.Errorf("ensure %s: %w", linkDir, mkErr)
	}
	// `ln -sfn`: -s symbolic, -f force replace, -n don't follow existing
	// symlink-to-dir (so the link is created at the target path itself,
	// not nested inside a previously-existing directory entry).
	if symErr := hostreqkit.RunAsInvokingUser("ln", []string{"-sfn", source, link}, opts); symErr != nil {
		return fmt.Errorf("create symlink %s -> %s: %w", link, source, symErr)
	}
	return nil
}

// goInstallTarget returns the path where `go install <pkg>` will deposit
// the binary. Resolution order matches Go's: $GOBIN, then $GOPATH/bin,
// then $HOME/go/bin. When running as root via sudo, $GOBIN/$GOPATH from
// the current (root) process are typically empty, so we fall through to
// the home-dir path — which uses InvokingUserHomeDir so it resolves to
// the operator's home, not /root.
func goInstallTarget() (string, error) {
	if gobin := strings.TrimSpace(os.Getenv("GOBIN")); gobin != "" {
		return filepath.Join(gobin, pluginBinName), nil
	}
	if gopath := strings.TrimSpace(os.Getenv("GOPATH")); gopath != "" {
		return filepath.Join(gopath, "bin", pluginBinName), nil
	}
	home, err := hostreqkit.InvokingUserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "go", "bin", pluginBinName), nil
}
