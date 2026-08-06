package vault

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
	aptKeyringPath = "/usr/share/keyrings/hashicorp-archive-keyring.gpg"
	aptSourcePath  = "/etc/apt/sources.list.d/hashicorp.list"
	aptKeyURL      = "https://apt.releases.hashicorp.com/gpg"
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
		status.PackageName = "vault"
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
		status.Notes = append(status.Notes, "automatic Vault CLI install is implemented for apt-based Linux, Homebrew-managed macOS, and winget-managed Windows hosts")
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
			status.Notes = append(status.Notes, "dry-run: would configure the HashiCorp apt repository and install vault")
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
		command, args, err := hostreqkit.InstallCommand(host, brewPkg, opts.SudoMode)
		if err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, err.Error())
			return status, nil
		}
		if err := hostreqkit.RunInstallCommand(command, args, opts); err != nil {
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
		command, args, err := hostreqkit.InstallCommand(host, wingetPkg, opts.SudoMode)
		if err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, err.Error())
			return status, nil
		}
		if err := hostreqkit.RunInstallCommand(command, args, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, err.Error())
			return status, nil
		}
	default:
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "automatic Vault CLI install unavailable on this host")
		return status, nil
	}

	status.Command, status.Installed = hostreqkit.ResolveCommand(h.manifest.Commands)
	if status.Installed {
		status.ExecutionState = hostreqkit.ExecutionInstalled
		status.Version = hostreqkit.ReadVersion(status.Command, h.manifest.VersionArgs)
		return status, nil
	}
	status.ExecutionState = hostreqkit.ExecutionFailed
	status.Notes = append(status.Notes, "install commands completed but vault is still not available on PATH")
	return status, nil
}

func aptSourceLine(host hostreqkit.Host) string {
	arch := "amd64"
	codename := "jammy"
	return fmt.Sprintf("deb [signed-by=%s arch=%s] https://apt.releases.hashicorp.com %s main", aptKeyringPath, arch, codename)
}

func ensureLinuxInstall(host hostreqkit.Host, opts hostreqkit.EnsureOptions) error {
	if !hostreqkit.CommandAvailable("gpg") {
		command, args, err := hostreqkit.InstallCommand(host, "gpg", opts.SudoMode)
		if err != nil {
			return err
		}
		if err := hostreqkit.RunInstallCommand(command, args, opts); err != nil {
			return err
		}
	}

	keyData, err := KeyDownloadFn()
	if err != nil {
		return err
	}

	asciiFile, err := os.CreateTemp("", "vrooli-hashicorp-key-*.asc")
	if err != nil {
		return fmt.Errorf("create HashiCorp key temp file: %w", err)
	}
	asciiTempPath := asciiFile.Name()
	if _, err := asciiFile.Write(keyData); err != nil {
		asciiFile.Close()
		return fmt.Errorf("write HashiCorp key temp file: %w", err)
	}
	if err := asciiFile.Close(); err != nil {
		return fmt.Errorf("close HashiCorp key temp file: %w", err)
	}
	defer os.Remove(asciiTempPath)

	keyFile, err := os.CreateTemp("", "vrooli-hashicorp-key-*.gpg")
	if err != nil {
		return fmt.Errorf("create HashiCorp binary key temp file: %w", err)
	}
	keyTempPath := keyFile.Name()
	if err := keyFile.Close(); err != nil {
		return fmt.Errorf("close HashiCorp binary key temp file: %w", err)
	}
	defer os.Remove(keyTempPath)

	if err := hostreqkit.RunCommandFn("gpg", []string{"--dearmor", "--yes", "--output", keyTempPath, asciiTempPath}, opts); err != nil {
		return err
	}

	sourceLine := aptSourceLine(host)
	sourceFile, err := os.CreateTemp("", "vrooli-hashicorp-source-*.list")
	if err != nil {
		return fmt.Errorf("create HashiCorp source temp file: %w", err)
	}
	sourceTempPath := sourceFile.Name()
	if _, err := sourceFile.WriteString(sourceLine + "\n"); err != nil {
		sourceFile.Close()
		return fmt.Errorf("write HashiCorp source temp file: %w", err)
	}
	if err := sourceFile.Close(); err != nil {
		return fmt.Errorf("close HashiCorp source temp file: %w", err)
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
	command, args, err := hostreqkit.InstallCommand(host, "vault", opts.SudoMode)
	if err != nil {
		return err
	}
	return hostreqkit.RunInstallCommand(command, args, opts)
}

func downloadKey() ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	response, err := client.Get(aptKeyURL)
	if err != nil {
		return nil, fmt.Errorf("download HashiCorp signing key: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("download HashiCorp signing key: unexpected HTTP %d", response.StatusCode)
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read HashiCorp signing key: %w", err)
	}
	return data, nil
}
