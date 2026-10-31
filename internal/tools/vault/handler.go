package vault

import (
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	aptKeyringPath = "/usr/share/keyrings/hashicorp-archive-keyring.gpg"
	aptSourcePath  = "/etc/apt/sources.list.d/hashicorp.list"
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
		Manifest: h.manifest, AptPackage: "vault", KeyringPath: aptKeyringPath,
		SourcePath: aptSourcePath, KeyURL: "https://apt.releases.hashicorp.com/gpg",
		SourceLine:     func(hostreqkit.Host) string { return aptSourceLine(hostreqkit.Host{}) },
		DownloadKey:    KeyDownloadFn,
		AptDryRunNotes: []string{"dry-run: would configure the HashiCorp apt repository and install vault"},
		BrewDryRunNote: "dry-run: brew install", WingetDryRunNote: "dry-run: winget install",
		UnsupportedApplyNote: "automatic Vault CLI install unavailable on this host",
	}
}

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := h.installer().Inspect(host, requirement)
	if status.SupportClass == hostreqkit.SupportUnsupported && status.ExecutionState == hostreqkit.ExecutionUnsupported {
		status.Notes = append(status.Notes, "automatic Vault CLI install is implemented for apt-based Linux, Homebrew-managed macOS, and winget-managed Windows hosts")
	}
	return status
}

func (h handler) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	return h.installer().ApplyWithNotes(host, status, opts)
}

func aptSourceLine(host hostreqkit.Host) string {
	arch := "amd64"
	codename := "jammy"
	return "deb [signed-by=" + aptKeyringPath + " arch=" + arch + "] https://apt.releases.hashicorp.com " + codename + " main"
}
