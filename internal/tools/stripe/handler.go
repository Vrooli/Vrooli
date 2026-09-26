package stripe

import (
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	aptKeyringPath = "/usr/share/keyrings/stripe.gpg"
	aptSourcePath  = "/etc/apt/sources.list.d/stripe.list"
	aptSourceLine  = "deb [signed-by=/usr/share/keyrings/stripe.gpg] https://packages.stripe.dev/stripe-cli-debian-local stable main"
	aptKeyURL      = "https://packages.stripe.dev/api/security/keypair/stripe-cli-gpg/public"
)

var KeyDownloadFn func() ([]byte, error)

type handler struct {
	manifest hostreqkit.ToolManifest
}

func NewHandler(manifest hostreqkit.ToolManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}

func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindTool }

func (h handler) installer() hostreqkit.AptRepoInstaller {
	return hostreqkit.AptRepoInstaller{
		Manifest: h.manifest, AptPackage: "stripe", KeyringPath: aptKeyringPath,
		SourcePath: aptSourcePath, KeyURL: aptKeyURL,
		SourceLine:     func(hostreqkit.Host) string { return aptSourceLine },
		DownloadKey:    KeyDownloadFn,
		AptDryRunNotes: []string{"dry-run: would configure the Stripe CLI installation flow", "apt-get update -qq"},
		BrewDryRunNote: "dry-run: would configure the Stripe CLI installation flow\nbrew install", WingetDryRunNote: "dry-run: winget install",
		UnsupportedApplyNote: "automatic Stripe CLI install unavailable on this host",
	}
}

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := h.installer().Inspect(host, requirement)
	if status.SupportClass == hostreqkit.SupportUnsupported && status.ExecutionState == hostreqkit.ExecutionUnsupported {
		status.Notes = append(status.Notes, "automatic Stripe CLI install is implemented for apt-based Linux, Homebrew-managed macOS, and winget-managed Windows hosts")
	}
	return status
}

func (h handler) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	result, err := h.installer().ApplyWithNotes(host, status, opts)
	if result.Installed {
		result.Notes = append(result.Notes, "use `stripe login` to authenticate before forwarding webhooks")
	}
	return result, err
}
