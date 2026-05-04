package protocgenconnectgo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	goModule       = "connectrpc.com/connect/cmd/protoc-gen-connect-go"
	pluginBinName  = "protoc-gen-connect-go"
	defaultVersion = "v1.19.2"
)

// UserHomeDirFn lets tests stub home-dir resolution.
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
	if err := hostreqkit.RunInstallCommand("go", []string{"install", pkgRef}, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}
	if err := ensureSymlinkOnPath(opts); err != nil {
		status.Notes = append(status.Notes, "post-install symlink: "+err.Error())
	}
	status.Command, status.Installed = hostreqkit.ResolveCommand(h.manifest.Commands)
	if !status.Installed {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, fmt.Sprintf("go install %s succeeded but %s is not on PATH; ensure $GOBIN (or $HOME/go/bin) is on PATH or rerun setup", pkgRef, pluginBinName))
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

func ensureSymlinkOnPath(opts hostreqkit.EnsureOptions) error {
	source, err := goInstallTarget()
	if err != nil {
		return err
	}
	if _, statErr := os.Stat(source); statErr != nil {
		return fmt.Errorf("post-install: %s not found at %s: %w", pluginBinName, source, statErr)
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

func goInstallTarget() (string, error) {
	if gobin := strings.TrimSpace(os.Getenv("GOBIN")); gobin != "" {
		return filepath.Join(gobin, pluginBinName), nil
	}
	if gopath := strings.TrimSpace(os.Getenv("GOPATH")); gopath != "" {
		return filepath.Join(gopath, "bin", pluginBinName), nil
	}
	home, err := UserHomeDirFn()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "go", "bin", pluginBinName), nil
}
