package infra

import "github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"

// desiredStateAllowsRecovery is the recovery-policy seam. A declared drift or
// an unmanaged capability is evidence for the operator, not authorization for
// autoheal to alter remote access.
func (c *RDPCheck) desiredStateAllowsRecovery(lastResult *checks.Result) bool {
	if c.desiredStateProvider == nil {
		return true
	}
	if lastResult == nil {
		return false
	}
	verdict, ok := lastResult.Details["desiredVerdict"].(string)
	return !ok || verdict == RemoteDesktopVerdictMatching
}
