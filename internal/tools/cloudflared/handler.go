package cloudflared

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	aptKeyringPath = "/usr/share/keyrings/cloudflare-main.gpg"
	aptSourcePath  = "/etc/apt/sources.list.d/cloudflared.list"
	aptSourceLine  = "deb [signed-by=/usr/share/keyrings/cloudflare-main.gpg] https://pkg.cloudflare.com/cloudflared any main"
	aptKeyURL      = "https://pkg.cloudflare.com/cloudflare-main.gpg"
)

var KeyDownloadFn = downloadKey

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
	case host.OS == "linux" && (host.PackageManager == "apt" || host.PackageManager == "apt-get"):
		status.PackageName = "cloudflared"
		status.InstallSupported = true
	case host.OS == "darwin" && host.PackageManager == "brew":
		status.PackageName = h.manifest.Packages["brew"]
		status.InstallSupported = true
	case host.OS == "windows" && host.PackageManager == "winget":
		status.PackageName = h.manifest.Packages["winget"]
		status.InstallSupported = true
	default:
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "automatic cloudflared install is implemented for apt-based Linux, Homebrew-managed macOS, and winget-managed Windows hosts")
	}

	if status.Installed {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Version = hostreqkit.ReadVersion(status.Command, h.manifest.VersionArgs)
		return status
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
		status.Notes = append(status.Notes, "manual install required by manifest declaration")
		return status, nil
	case hostreqkit.SupportUnsupported:
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "automatic install unavailable on this host")
		return status, nil
	case hostreqkit.SupportNotApplicable:
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "requirement is not applicable on this host")
		return status, nil
	}

	switch {
	case host.OS == "linux" && (host.PackageManager == "apt" || host.PackageManager == "apt-get"):
		if opts.DryRun {
			status.ExecutionState = hostreqkit.ExecutionWouldInstall
			status.Notes = append(status.Notes, "dry-run: would configure the Cloudflare apt repository and install cloudflared")
			return status, nil
		}
		if err := ensureLinuxInstall(host, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, err.Error())
			return status, nil
		}
	case host.OS == "darwin" && host.PackageManager == "brew":
		brewPkg := h.manifest.Packages["brew"]
		if opts.DryRun {
			status.ExecutionState = hostreqkit.ExecutionWouldInstall
			status.Notes = append(status.Notes, "dry-run: brew install "+brewPkg)
			return status, nil
		}
		if err := hostreqkit.RunInstallCommand("brew", []string{"install", brewPkg}, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, err.Error())
			return status, nil
		}
	case host.OS == "windows" && host.PackageManager == "winget":
		wingetPkg := h.manifest.Packages["winget"]
		if opts.DryRun {
			status.ExecutionState = hostreqkit.ExecutionWouldInstall
			status.Notes = append(status.Notes, "dry-run: winget install "+wingetPkg)
			return status, nil
		}
		if err := hostreqkit.RunInstallCommand("winget", []string{"install", "--id", wingetPkg, "-e", "--accept-source-agreements", "--accept-package-agreements"}, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, err.Error())
			return status, nil
		}
	default:
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "automatic cloudflared install unavailable on this host")
		return status, nil
	}

	status.Command, status.Installed = hostreqkit.ResolveCommand(h.manifest.Commands)
	if status.Installed {
		status.ExecutionState = hostreqkit.ExecutionInstalled
		status.Version = hostreqkit.ReadVersion(status.Command, h.manifest.VersionArgs)
		return status, nil
	}
	status.ExecutionState = hostreqkit.ExecutionFailed
	status.Notes = append(status.Notes, "install commands completed but cloudflared is still not available on PATH")
	return status, nil
}

func ensureLinuxInstall(host hostreqkit.Host, opts hostreqkit.EnsureOptions) error {
	keyData, err := KeyDownloadFn()
	if err != nil {
		return err
	}

	// Cloudflare provides the key already in binary GPG format, so no dearmor needed.
	keyFile, err := os.CreateTemp("", "vrooli-cloudflare-key-*.gpg")
	if err != nil {
		return fmt.Errorf("create Cloudflare key temp file: %w", err)
	}
	keyTempPath := keyFile.Name()
	if _, err := keyFile.Write(keyData); err != nil {
		keyFile.Close()
		return fmt.Errorf("write Cloudflare key temp file: %w", err)
	}
	if err := keyFile.Close(); err != nil {
		return fmt.Errorf("close Cloudflare key temp file: %w", err)
	}
	defer os.Remove(keyTempPath)

	sourceFile, err := os.CreateTemp("", "vrooli-cloudflare-source-*.list")
	if err != nil {
		return fmt.Errorf("create Cloudflare source temp file: %w", err)
	}
	sourceTempPath := sourceFile.Name()
	if _, err := sourceFile.WriteString(aptSourceLine + "\n"); err != nil {
		sourceFile.Close()
		return fmt.Errorf("write Cloudflare source temp file: %w", err)
	}
	if err := sourceFile.Close(); err != nil {
		return fmt.Errorf("close Cloudflare source temp file: %w", err)
	}
	defer os.Remove(sourceTempPath)

	for _, dir := range []string{filepath.Dir(aptKeyringPath), filepath.Dir(aptSourcePath)} {
		if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "mkdir", []string{"-p", dir}, opts); err != nil {
			return err
		}
	}
	if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "install", []string{"-m", "0644", keyTempPath, aptKeyringPath}, opts); err != nil {
		return err
	}
	if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "install", []string{"-m", "0644", sourceTempPath, aptSourcePath}, opts); err != nil {
		return err
	}
	if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "apt-get", []string{"update", "-qq"}, opts); err != nil {
		return err
	}
	command, args, err := hostreqkit.InstallCommand(host, "cloudflared", opts.SudoMode)
	if err != nil {
		return err
	}
	return hostreqkit.RunInstallCommand(command, args, opts)
}

func downloadKey() ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	response, err := client.Get(aptKeyURL)
	if err != nil {
		return nil, fmt.Errorf("download Cloudflare signing key: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("download Cloudflare signing key: unexpected HTTP %d", response.StatusCode)
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read Cloudflare signing key: %w", err)
	}
	return data, nil
}
