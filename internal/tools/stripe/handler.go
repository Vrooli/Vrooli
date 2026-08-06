package stripe

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
	aptKeyringPath = "/usr/share/keyrings/stripe.gpg"
	aptSourcePath  = "/etc/apt/sources.list.d/stripe.list"
	aptSourceLine  = "deb [signed-by=/usr/share/keyrings/stripe.gpg] https://packages.stripe.dev/stripe-cli-debian-local stable main"
	aptKeyURL      = "https://packages.stripe.dev/api/security/keypair/stripe-cli-gpg/public"
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
		status.PackageName = "stripe"
		status.InstallSupported = true
	case host.OS == "darwin" && host.PackageManager == "brew":
		status.PackageName = h.manifest.Packages["brew"]
		status.InstallSupported = true
	case host.OS == "windows" && host.PackageManager == "winget":
		status.PackageName = h.manifest.Packages["winget"]
		status.InstallSupported = status.PackageName != ""
	default:
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "automatic Stripe CLI install is implemented for apt-based Linux, Homebrew-managed macOS, and winget-managed Windows hosts")
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
			status.Notes = append(status.Notes, "dry-run: would configure the Stripe CLI installation flow")
			status.Notes = append(status.Notes, "apt-get update -qq")
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
			status.Notes = append(status.Notes, "dry-run: would configure the Stripe CLI installation flow")
			status.Notes = append(status.Notes, "brew install "+brewPkg)
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
		status.Notes = append(status.Notes, "automatic Stripe CLI install unavailable on this host")
		return status, nil
	}

	status.Command, status.Installed = hostreqkit.ResolveCommand(h.manifest.Commands)
	if status.Installed {
		status.ExecutionState = hostreqkit.ExecutionInstalled
		status.Version = hostreqkit.ReadVersion(status.Command, h.manifest.VersionArgs)
		status.Notes = append(status.Notes, "use `stripe login` to authenticate before forwarding webhooks")
		return status, nil
	}
	status.ExecutionState = hostreqkit.ExecutionFailed
	status.Notes = append(status.Notes, "install commands completed but the Stripe CLI is still not available on PATH")
	return status, nil
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

	keyFile, err := os.CreateTemp("", "vrooli-stripe-key-*.gpg")
	if err != nil {
		return fmt.Errorf("create Stripe key temp file: %w", err)
	}
	keyTempPath := keyFile.Name()
	if err := keyFile.Close(); err != nil {
		return fmt.Errorf("close Stripe key temp file: %w", err)
	}
	defer os.Remove(keyTempPath)

	asciiFile, err := os.CreateTemp("", "vrooli-stripe-key-*.asc")
	if err != nil {
		return fmt.Errorf("create Stripe ASCII key temp file: %w", err)
	}
	asciiTempPath := asciiFile.Name()
	if _, err := asciiFile.Write(keyData); err != nil {
		asciiFile.Close()
		return fmt.Errorf("write Stripe ASCII key temp file: %w", err)
	}
	if err := asciiFile.Close(); err != nil {
		return fmt.Errorf("close Stripe ASCII key temp file: %w", err)
	}
	defer os.Remove(asciiTempPath)

	if err := hostreqkit.RunCommandFn("gpg", []string{"--dearmor", "--yes", "--output", keyTempPath, asciiTempPath}, opts); err != nil {
		return err
	}

	sourceFile, err := os.CreateTemp("", "vrooli-stripe-source-*.list")
	if err != nil {
		return fmt.Errorf("create Stripe source temp file: %w", err)
	}
	sourceTempPath := sourceFile.Name()
	if _, err := sourceFile.WriteString(aptSourceLine + "\n"); err != nil {
		sourceFile.Close()
		return fmt.Errorf("write Stripe source temp file: %w", err)
	}
	if err := sourceFile.Close(); err != nil {
		return fmt.Errorf("close Stripe source temp file: %w", err)
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
	command, args, err := hostreqkit.InstallCommand(host, "stripe", opts.SudoMode)
	if err != nil {
		return err
	}
	return hostreqkit.RunInstallCommand(command, args, opts)
}

func downloadKey() ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	response, err := client.Get(aptKeyURL)
	if err != nil {
		return nil, fmt.Errorf("download Stripe signing key: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("download Stripe signing key: unexpected HTTP %d", response.StatusCode)
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read Stripe signing key: %w", err)
	}
	return data, nil
}
