package runtime

import (
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreq"
)

const (
	stripeAPTKeyringPath = "/usr/share/keyrings/stripe.gpg"
	stripeAPTSourcePath  = "/etc/apt/sources.list.d/stripe.list"
	stripeAPTSourceLine  = "deb [signed-by=/usr/share/keyrings/stripe.gpg] https://packages.stripe.dev/stripe-cli-debian-local stable main"
)

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

	var script string
	switch {
	case host.OS == "linux" && (host.PackageManager == "apt" || host.PackageManager == "apt-get"):
		script = strings.Join([]string{
			"set -e",
			"if ! command -v gpg >/dev/null 2>&1; then apt-get update -qq && apt-get install -y gpg; fi",
			"mkdir -p /usr/share/keyrings /etc/apt/sources.list.d",
			fmt.Sprintf("curl -fsSL https://packages.stripe.dev/api/security/keypair/stripe-cli-gpg/public | gpg --dearmor -o %s", stripeAPTKeyringPath),
			fmt.Sprintf("printf '%%s\\n' %q > %s", stripeAPTSourceLine, stripeAPTSourcePath),
			"apt-get update -qq",
			"apt-get install -y stripe",
		}, "\n")
	case host.OS == "darwin" && host.PackageManager == "brew":
		script = "set -e\nbrew install stripe/stripe-cli/stripe"
	default:
		status.SupportClass = SupportUnsupported
		status.ExecutionState = ExecutionUnsupported
		status.Notes = append(status.Notes, "automatic Stripe CLI install unavailable on this host")
		return status, nil
	}

	if opts.DryRun {
		status.ExecutionState = ExecutionWouldInstall
		status.Notes = append(status.Notes, "dry-run: would configure the Stripe CLI installation flow")
		status.Notes = append(status.Notes, firstLine(script))
		return status, nil
	}

	if err := runShellScript(script, opts.SudoMode, opts); err != nil {
		status.ExecutionState = ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
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
