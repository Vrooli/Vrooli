package runtime

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/vrooli/vrooli/internal/hostreq"
)

const (
	stripeAPTKeyringPath = "/usr/share/keyrings/stripe.gpg"
	stripeAPTSourcePath  = "/etc/apt/sources.list.d/stripe.list"
	stripeAPTSourceLine  = "deb [signed-by=/usr/share/keyrings/stripe.gpg] https://packages.stripe.dev/stripe-cli-debian-local stable main"
	stripeAPTKeyURL      = "https://packages.stripe.dev/api/security/keypair/stripe-cli-gpg/public"
)

var stripeKeyDownloadFn = downloadStripeKey

type stripeToolHandler struct{}

func newStripeTool() handler {
	return stripeToolHandler{}
}

func (stripeToolHandler) Name() string       { return "stripe" }
func (stripeToolHandler) Kind() hostreq.Kind { return hostreq.KindTool }

func (stripeToolHandler) Inspect(host Host, requirement hostreq.ResolvedRequirement) ItemStatus {
	status := baseStatus(requirement)
	status.Command, status.Installed = resolveCommand([]string{"stripe"})
	status.SupportClass = SupportSupported
	status.Notes = append(status.Notes, "Install the Stripe CLI for local webhook forwarding and payment-workflow validation")
	if requirement.Manual {
		status.SupportClass = SupportManualOnly
		status.ExecutionState = ExecutionManualActionRequired
		return status
	}

	switch {
	case host.OS == "linux" && (host.PackageManager == "apt" || host.PackageManager == "apt-get"):
		status.PackageName = "stripe"
		status.InstallSupported = true
	case host.OS == "darwin" && host.PackageManager == "brew":
		status.PackageName = "stripe/stripe-cli/stripe"
		status.InstallSupported = true
	default:
		status.SupportClass = SupportUnsupported
		status.ExecutionState = ExecutionUnsupported
		status.Notes = append(status.Notes, "automatic Stripe CLI install is implemented for apt-based Linux hosts and Homebrew-managed macOS hosts")
	}

	if status.Installed {
		status.ExecutionState = ExecutionAlreadyPresent
		status.Version = readVersion(status.Command, []string{"version"})
		return status
	}
	return status
}

func (stripeToolHandler) Apply(host Host, status ItemStatus, opts EnsureOptions) (ItemStatus, error) {
	if status.Installed {
		status.ExecutionState = ExecutionAlreadyPresent
		return status, nil
	}

	switch status.SupportClass {
	case SupportManualOnly:
		status.ExecutionState = ExecutionManualActionRequired
		status.Notes = append(status.Notes, "manual install required by manifest declaration")
		return status, nil
	case SupportUnsupported:
		status.ExecutionState = ExecutionUnsupported
		status.Notes = append(status.Notes, "automatic install unavailable on this host")
		return status, nil
	case SupportNotApplicable:
		status.ExecutionState = ExecutionNotApplicable
		status.Notes = append(status.Notes, "requirement is not applicable on this host")
		return status, nil
	}

	switch {
	case host.OS == "linux" && (host.PackageManager == "apt" || host.PackageManager == "apt-get"):
		if opts.DryRun {
			status.ExecutionState = ExecutionWouldInstall
			status.Notes = append(status.Notes, "dry-run: would configure the Stripe CLI installation flow")
			status.Notes = append(status.Notes, "apt-get update -qq")
			return status, nil
		}
		if err := ensureStripeLinuxInstall(host, opts); err != nil {
			status.ExecutionState = ExecutionFailed
			status.Notes = append(status.Notes, err.Error())
			return status, nil
		}
	case host.OS == "darwin" && host.PackageManager == "brew":
		if opts.DryRun {
			status.ExecutionState = ExecutionWouldInstall
			status.Notes = append(status.Notes, "dry-run: would configure the Stripe CLI installation flow")
			status.Notes = append(status.Notes, "brew install stripe/stripe-cli/stripe")
			return status, nil
		}
		if err := runInstallCommand("brew", []string{"install", "stripe/stripe-cli/stripe"}, opts); err != nil {
			status.ExecutionState = ExecutionFailed
			status.Notes = append(status.Notes, err.Error())
			return status, nil
		}
	default:
		status.SupportClass = SupportUnsupported
		status.ExecutionState = ExecutionUnsupported
		status.Notes = append(status.Notes, "automatic Stripe CLI install unavailable on this host")
		return status, nil
	}

	status.Command, status.Installed = resolveCommand([]string{"stripe"})
	if status.Installed {
		status.ExecutionState = ExecutionInstalled
		status.Version = readVersion(status.Command, []string{"version"})
		status.Notes = append(status.Notes, "use `stripe login` to authenticate before forwarding webhooks")
		return status, nil
	}
	status.ExecutionState = ExecutionFailed
	status.Notes = append(status.Notes, "install commands completed but the Stripe CLI is still not available on PATH")
	return status, nil
}

func ensureStripeLinuxInstall(host Host, opts EnsureOptions) error {
	if !commandAvailable("gpg") {
		command, args, err := installCommand(host, "gpg", opts.SudoMode)
		if err != nil {
			return err
		}
		if err := runInstallCommand(command, args, opts); err != nil {
			return err
		}
	}

	keyData, err := stripeKeyDownloadFn()
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

	if err := runCommandFn("gpg", []string{"--dearmor", "--yes", "--output", keyTempPath, asciiTempPath}, opts); err != nil {
		return err
	}

	sourceFile, err := os.CreateTemp("", "vrooli-stripe-source-*.list")
	if err != nil {
		return fmt.Errorf("create Stripe source temp file: %w", err)
	}
	sourceTempPath := sourceFile.Name()
	if _, err := sourceFile.WriteString(stripeAPTSourceLine + "\n"); err != nil {
		sourceFile.Close()
		return fmt.Errorf("write Stripe source temp file: %w", err)
	}
	if err := sourceFile.Close(); err != nil {
		return fmt.Errorf("close Stripe source temp file: %w", err)
	}
	defer os.Remove(sourceTempPath)

	for _, dir := range []string{filepath.Dir(stripeAPTKeyringPath), filepath.Dir(stripeAPTSourcePath)} {
		if err := runPrivilegedCommand(opts.SudoMode, "mkdir", []string{"-p", dir}, opts); err != nil {
			return err
		}
	}
	if err := runPrivilegedCommand(opts.SudoMode, "install", []string{"-m", "0644", keyTempPath, stripeAPTKeyringPath}, opts); err != nil {
		return err
	}
	if err := runPrivilegedCommand(opts.SudoMode, "install", []string{"-m", "0644", sourceTempPath, stripeAPTSourcePath}, opts); err != nil {
		return err
	}
	if err := runPrivilegedCommand(opts.SudoMode, "apt-get", []string{"update", "-qq"}, opts); err != nil {
		return err
	}
	command, args, err := installCommand(host, "stripe", opts.SudoMode)
	if err != nil {
		return err
	}
	return runInstallCommand(command, args, opts)
}

func downloadStripeKey() ([]byte, error) {
	response, err := http.Get(stripeAPTKeyURL)
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
