package pstoreobservability

import (
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

// Windows release builds replace the Linux-only pstore collector with this
// handler. It preserves the manifest/registry contract while honestly reporting
// that the safeguard cannot be applied on this platform.
type unsupportedHandler struct {
	manifest hostreqkit.SafeguardManifest
}

func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return unsupportedHandler{manifest: manifest}
}

func (h unsupportedHandler) Name() string           { return h.manifest.Name }
func (h unsupportedHandler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }

func (h unsupportedHandler) Inspect(_ hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	status.SupportClass = hostreqkit.SupportUnsupported
	status.ExecutionState = hostreqkit.ExecutionUnsupported
	status.Notes = append(status.Notes, "pstore observability export is Linux-only")
	return status
}

func (h unsupportedHandler) Apply(_ hostreqkit.Host, status hostreqkit.ItemStatus, _ hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	status.SupportClass = hostreqkit.SupportUnsupported
	status.ExecutionState = hostreqkit.ExecutionUnsupported
	return status, nil
}
