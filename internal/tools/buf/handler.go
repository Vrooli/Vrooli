// Package buf implements the host-tool handler for the Buf CLI used by
// packages/proto codegen, replacing the legacy
// scripts/migrate_candidates/tools/buf.sh shell installer.
//
// Install path: download the platform-specific GitHub release binary
// (buf-<Linux|Darwin>-<x86_64|arm64>) and install to /usr/local/bin
// when sudo is available, falling back to ~/.local/bin otherwise.
// Windows installs via the brew/winget mappings declared in tool.json.
package buf

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	defaultVersion = "1.37.0"
	binName        = "buf"
	releaseURLBase = "https://github.com/bufbuild/buf/releases/download"
)

// DownloadFn is overridable for tests.
var (
	DownloadFn    = downloadRelease
	UserHomeDirFn = os.UserHomeDir
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
	status.Command, status.Installed = hostreqkit.ResolveCommand(h.manifest.Commands)
	status.SupportClass = hostreqkit.SupportSupported
	status.Notes = append(status.Notes, h.manifest.InstallHint)
	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status
	}

	switch {
	case host.OS == "linux" || host.OS == "darwin":
		status.PackageName = h.releaseFilename(host)
		status.InstallSupported = status.PackageName != ""
		if !status.InstallSupported {
			status.SupportClass = hostreqkit.SupportUnsupported
			status.ExecutionState = hostreqkit.ExecutionUnsupported
			status.Notes = append(status.Notes, "automatic buf install unsupported on "+host.OS+"/"+runtime.GOARCH)
		}
	case host.OS == "windows" && host.PackageManager == "winget":
		status.PackageName = h.manifest.Packages["winget"]
		status.InstallSupported = status.PackageName != ""
	default:
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "automatic buf install is implemented for Linux, macOS (release tarball), and Windows (winget)")
	}

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

	switch {
	case host.OS == "linux" || host.OS == "darwin":
		filename := h.releaseFilename(host)
		if filename == "" {
			status.ExecutionState = hostreqkit.ExecutionUnsupported
			status.Notes = append(status.Notes, "no release asset for this OS/arch")
			return status, nil
		}
		url := releaseURLBase + "/v" + h.versionRef() + "/" + filename
		if opts.DryRun {
			status.ExecutionState = hostreqkit.ExecutionWouldInstall
			status.Notes = append(status.Notes, "dry-run: download "+url+" and install as "+binName)
			return status, nil
		}
		if err := h.installFromRelease(url, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, err.Error())
			return status, nil
		}
	case host.OS == "windows" && host.PackageManager == "winget":
		pkg := h.manifest.Packages["winget"]
		if opts.DryRun {
			status.ExecutionState = hostreqkit.ExecutionWouldInstall
			status.Notes = append(status.Notes, "dry-run: winget install "+pkg)
			return status, nil
		}
		if err := hostreqkit.RunInstallCommand("winget", []string{"install", "--id", pkg, "-e", "--accept-source-agreements", "--accept-package-agreements"}, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, err.Error())
			return status, nil
		}
	default:
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		return status, nil
	}

	status.Command, status.Installed = hostreqkit.ResolveCommand(h.manifest.Commands)
	if !status.Installed {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "install completed but buf is not on PATH")
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

// releaseFilename returns the GitHub release asset name for the host's
// OS/arch combination, or "" if no asset is available. Buf's release
// naming uses Linux/Darwin (capitalized) and x86_64/arm64.
func (h handler) releaseFilename(host hostreqkit.Host) string {
	platform := ""
	switch host.OS {
	case "linux":
		platform = "Linux"
	case "darwin":
		platform = "Darwin"
	default:
		return ""
	}
	arch := ""
	switch runtime.GOARCH {
	case "amd64":
		arch = "x86_64"
	case "arm64":
		arch = "arm64"
	default:
		return ""
	}
	return "buf-" + platform + "-" + arch
}

func (h handler) installFromRelease(url string, opts hostreqkit.EnsureOptions) error {
	data, err := DownloadFn(url)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}

	dest, useSudo, err := chooseInstallDir(opts)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dest, 0o755); err != nil && !useSudo {
		return fmt.Errorf("ensure %s: %w", dest, err)
	}

	tmp, err := os.CreateTemp("", "vrooli-buf-*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp: %w", err)
	}
	defer os.Remove(tmpPath)

	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return fmt.Errorf("chmod temp: %w", err)
	}

	finalPath := filepath.Join(dest, binName)
	if useSudo {
		return hostreqkit.RunPrivilegedCommand(opts.SudoMode, "install", []string{"-m", "0755", tmpPath, finalPath}, opts)
	}
	return os.Rename(tmpPath, finalPath)
}

// chooseInstallDir picks /usr/local/bin when sudo is available (matching
// the legacy bash installer), falling back to ~/.local/bin otherwise.
func chooseInstallDir(opts hostreqkit.EnsureOptions) (dir string, useSudo bool, err error) {
	mode := strings.ToLower(strings.TrimSpace(opts.SudoMode))
	if mode != "skip" && hostreqkit.SudoAvailable() {
		return "/usr/local/bin", true, nil
	}
	home, err := UserHomeDirFn()
	if err != nil {
		return "", false, fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".local", "bin"), false, nil
}

func downloadRelease(url string) ([]byte, error) {
	client := &http.Client{Timeout: 5 * time.Minute}
	response, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected HTTP %d", response.StatusCode)
	}
	return io.ReadAll(response.Body)
}
