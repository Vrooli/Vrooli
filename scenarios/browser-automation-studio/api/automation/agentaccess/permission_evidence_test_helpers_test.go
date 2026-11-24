package agentaccess

import (
	"time"

	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
)

func permissionEvidence(status, source string, observedAt time.Time) sessionpermission.ExtensionPermissionEvidence {
	scope := sessionpermission.ExtensionPermissionScopeUnavailable
	if status == sessionpermission.ExtensionPermissionEvidenceEffective {
		scope = sessionpermission.ExtensionPermissionScopeGrantSetOnly
	}
	return sessionpermission.ExtensionPermissionEvidence{
		Status: status, Source: source, Scope: scope,
		PerTabSiteAccess: sessionpermission.ExtensionPerTabSiteAccessNotMeasured,
		ObservedAt:       observedAt,
	}
}
