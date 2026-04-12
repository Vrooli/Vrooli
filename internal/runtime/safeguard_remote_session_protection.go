package runtime

import "github.com/vrooli/vrooli/internal/hostreq"

type remoteSessionProtectionHandler struct{}

func newRemoteSessionProtectionSafeguard() handler {
	return remoteSessionProtectionHandler{}
}

func (remoteSessionProtectionHandler) Name() string       { return "remote_session_protection" }
func (remoteSessionProtectionHandler) Kind() hostreq.Kind { return hostreq.KindSafeguard }

func (remoteSessionProtectionHandler) Inspect(host Host, requirement hostreq.ResolvedRequirement) ItemStatus {
	status := baseStatus(requirement)
	status.SupportClass = SupportSupported
	if requirement.Manual {
		status.SupportClass = SupportManualOnly
		return status
	}
	if host.OS != "linux" {
		status.SupportClass = SupportUnsupported
		status.Notes = append(status.Notes, "remote session protection is only supported on Linux hosts")
		return status
	}
	if !host.SupportsSysctl && !host.SupportsSystemd {
		status.SupportClass = SupportNotApplicable
		status.Notes = append(status.Notes, "host does not expose sysctl or systemd hooks needed for safeguard application")
		return status
	}
	status.Notes = append(status.Notes, "native safeguard plumbing is active; concrete host mutations remain intentionally deferred")
	return status
}

func (remoteSessionProtectionHandler) Apply(_ Host, status ItemStatus, opts EnsureOptions) (ItemStatus, error) {
	switch status.SupportClass {
	case SupportUnsupported, SupportNotApplicable:
		return status, nil
	case SupportManualOnly:
		status.Notes = append(status.Notes, "manual safeguard action required by manifest declaration")
		return status, nil
	}
	if opts.DryRun {
		status.Notes = append(status.Notes, "dry-run: would apply remote session protection")
		return status, nil
	}
	status.Applied = true
	status.Notes = append(status.Notes, "remote session protection marked applied by native safeguard stub")
	return status, nil
}
