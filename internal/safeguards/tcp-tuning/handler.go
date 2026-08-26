package tcptuning

import (
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const configPath = "/etc/sysctl.d/99-vrooli-tcp.conf"

type handler struct {
	manifest hostreqkit.SafeguardManifest
}

func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}

func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }

func (h handler) applier() hostreqkit.SysctlApplier {
	return hostreqkit.SysctlApplier{
		ConfigPath: configPath,
		Parameters: []hostreqkit.SysctlParameter{
			{Name: "net.ipv4.tcp_ecn", Value: 0, ReadFailure: -1},
			{Name: "net.ipv4.tcp_mtu_probing", Value: 1, ReadFailure: -1},
			{Name: "net.ipv4.tcp_base_mss", Value: 1024, ReadFailure: -1},
		},
		UnsupportedNote:   "TCP tuning is only supported on Linux",
		NotApplicableNote: "host does not support sysctl",
		ManualNote:        "manual safeguard action required by manifest declaration",
		PendingNote:       "TCP parameters need adjustment",
		AppliedNote:       "all TCP parameters at desired values",
		DryRunNote:        "dry-run: would configure TCP parameters in " + configPath,
	}
}

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	return h.applier().Inspect(host, requirement)
}

func (h handler) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	result, err := h.applier().Apply(status, opts)
	if err != nil {
		result.ExecutionState = hostreqkit.ExecutionFailed
		result.Notes = append(result.Notes, err.Error())
	}
	if result.ExecutionState == hostreqkit.ExecutionApplied {
		result.Notes = append(result.Notes, "TCP parameters configured and applied")
	}
	return result, nil
}

// buildConfigContent remains a package-local compatibility seam for the
// safeguard's focused tests; the implementation is owned by SysctlApplier.
func buildConfigContent() string { return (&handler{}).applier().ConfigContent() }
