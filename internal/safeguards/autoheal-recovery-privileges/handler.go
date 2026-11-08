// Package autohealrecoveryprivileges verifies the recovery capability installed
// by project setup. It no longer maintains a second runtime sudoers policy.
package autohealrecoveryprivileges

import (
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/privilegebroker"
)

var inspectBroker = privilegebroker.Inspect

type handler struct{ manifest hostreqkit.SafeguardManifest }

func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}
func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }
func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	if host.OS != string(hostreqspec.PlatformLinux) {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "Linux broker service recovery does not apply to this platform")
		return status
	}
	broker := inspectBroker()
	for _, action := range broker.Capabilities {
		if broker.Available && action == privilegebroker.ActionServiceRecover {
			status.Applied = true
			status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
			status.Notes = append(status.Notes, "service recovery is provided by the setup-managed privilege broker; runtime sudoers grants are unused")
			return status
		}
	}
	status.ExecutionState = hostreqkit.ExecutionManualActionRequired
	status.Notes = append(status.Notes, "run `vrooli setup broker --sudo-mode=ask` to install the host.service.recover capability")
	return status
}
func (h handler) Apply(_ hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	if status.Applied || status.SupportClass == hostreqkit.SupportNotApplicable {
		return status, nil
	}
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, "project setup installs the root-owned broker before applying safeguards")
		return status, nil
	}
	status.ExecutionState = hostreqkit.ExecutionManualActionRequired
	status.Notes = append(status.Notes, "broker installation is owned exclusively by `vrooli setup broker --sudo-mode=ask`; no sudoers file was written")
	return status, nil
}
