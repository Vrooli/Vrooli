package kernelconfig

import (
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const configPath = "/etc/sysctl.d/99-vrooli.conf"

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
			{Name: "fs.inotify.max_user_watches", Value: 1048576, Minimum: true, ReadFailure: 0},
			{Name: "fs.inotify.max_user_instances", Value: 2048, Minimum: true, ReadFailure: 0},
		},
		UnsupportedNote:   "kernel parameter management is only supported on Linux",
		NotApplicableNote: "host does not support sysctl",
		ManualNote:        "manual safeguard action required by manifest declaration",
		PendingNote:       "kernel parameters below minimum values",
		AppliedNote:       "all kernel parameters meet minimum values",
		DryRunNote:        "dry-run: would configure kernel parameters in " + configPath,
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
		result.Notes = append(result.Notes, "kernel parameters configured and applied")
	}
	return result, nil
}

// buildConfigContent remains a package-local compatibility seam for the
// safeguard's focused tests; the implementation is owned by SysctlApplier.
func buildConfigContent() string { return (&handler{}).applier().ConfigContent() }
