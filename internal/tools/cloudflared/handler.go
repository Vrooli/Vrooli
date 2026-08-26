package cloudflared

import (
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	aptKeyringPath = "/usr/share/keyrings/cloudflare-main.gpg"
	aptSourcePath  = "/etc/apt/sources.list.d/cloudflared.list"
	aptSourceLine  = "deb [signed-by=/usr/share/keyrings/cloudflare-main.gpg] https://pkg.cloudflare.com/cloudflared any main"
	aptKeyURL      = "https://pkg.cloudflare.com/cloudflare-main.gpg"
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
		Manifest: h.manifest, AptPackage: "cloudflared", KeyringPath: aptKeyringPath,
		SourcePath: aptSourcePath, KeyURL: aptKeyURL,
		SourceLine:  func(hostreqkit.Host) string { return aptSourceLine },
		DownloadKey: KeyDownloadFn, BinaryKey: true,
		AptDryRunNotes: []string{"dry-run: would configure the Cloudflare apt repository and install cloudflared"},
		BrewDryRunNote: "dry-run: brew install", WingetDryRunNote: "dry-run: winget install",
		UnsupportedApplyNote: "automatic cloudflared install unavailable on this host",
	}
}

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := h.installer().Inspect(host, requirement)
	if status.SupportClass == hostreqkit.SupportUnsupported && status.ExecutionState == hostreqkit.ExecutionUnsupported {
		status.Notes = append(status.Notes, "automatic cloudflared install is implemented for apt-based Linux, Homebrew-managed macOS, and winget-managed Windows hosts")
	}
	return status
}

func (h handler) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	return h.installer().ApplyWithNotes(host, status, opts)
}
