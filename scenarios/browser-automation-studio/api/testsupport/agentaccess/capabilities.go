// Package agentaccess provides fixture-only reports for cross-scenario tests.
// It is intentionally outside BAS's production dependency graph.
package agentaccess

import (
	"context"
	"time"

	basaccess "github.com/vrooli/browser-automation-studio/automation/agentaccess"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
)

type qualifiedManagedAct []sessionpermission.BackendReport

func (r qualifiedManagedAct) BackendReports(context.Context) ([]sessionpermission.BackendReport, error) {
	return r, nil
}

// QualifiedManagedAct returns a fake-owner-only qualification report so
// integration tests can exercise Broker policy without a live-browser claim.
func QualifiedManagedAct() basaccess.CapabilityReporter {
	return qualifiedManagedAct{{
		Backend: sessionpermission.BackendManaged,
		Qualified: true,
		EvidenceID: "fake-owner-integration-fixture",
		Capabilities: []sessionpermission.Capability{
			sessionpermission.CapabilityObserve,
			sessionpermission.CapabilityControl,
			sessionpermission.CapabilityHistoryRead,
		},
	}}
}

// QualifiedManagedExtensionLifecycle returns fake-owner-only capabilities for
// the S2E-to-BAS composed route test. No extension process or browser is
// started; the test injects a typed ExtensionController fake.
func QualifiedManagedExtensionLifecycle() basaccess.CapabilityReporter {
	return qualifiedManagedAct{{
		Backend:    sessionpermission.BackendManaged,
		Qualified:  true,
		EvidenceID: "fake-owner-extension-route-integration",
		Capabilities: []sessionpermission.Capability{
			sessionpermission.CapabilityObserve,
			sessionpermission.CapabilityControl,
			sessionpermission.CapabilityExtensionInstall,
			sessionpermission.CapabilityExtensionLoad,
			sessionpermission.CapabilityExtensionInventory,
			sessionpermission.CapabilityExtensionHostPermissions,
		},
	}}
}

// VerifiedExtensionInventoryEntry creates a deterministic fake-owner inventory
// row for cross-scenario lifecycle tests. It carries no browser-derived claim.
func VerifiedExtensionInventoryReceipt(binding basaccess.ExtensionSessionBinding, artifact basaccess.ExtensionArtifact, observed time.Time) basaccess.ExtensionInventoryReceipt {
	entry := sessionpermission.ExtensionInventoryEntry{
		ID: artifact.ExtensionID, Name: "fake-owner fixture", Version: artifact.Version, Enabled: true,
		Permissions: append([]string(nil), artifact.Permissions...), HostPermissions: append([]string(nil), artifact.HostPermissions...),
		DeclaredPermissions: append([]string(nil), artifact.Permissions...), DeclaredHostPermissions: append([]string(nil), artifact.HostPermissions...),
		ArtifactSHA256: artifact.Digest, ManifestSHA256: artifact.ManifestSHA256, ArtifactVerified: true,
		PermissionEvidence: sessionpermission.ExtensionPermissionEvidence{
			Status: sessionpermission.ExtensionPermissionEvidenceEffective,
			Source: sessionpermission.ExtensionPermissionSourceChromeGetAll,
			Scope: sessionpermission.ExtensionPermissionScopeGrantSetOnly,
			PerTabSiteAccess: sessionpermission.ExtensionPerTabSiteAccessNotMeasured,
			ObservedAt: observed,
		},
	}
	return basaccess.ExtensionInventoryReceipt{Binding: binding, ObservedAt: observed, Extensions: []sessionpermission.ExtensionInventoryEntry{entry}}
}

type evidenceRecorder struct{}

func (evidenceRecorder) RecordBrowserSession(context.Context, string, sessionpermission.Backend, string, string) error {
	return nil
}
func (evidenceRecorder) RecordBrowserAction(context.Context, string, sessionpermission.Backend, string, string, string) error {
	return nil
}

// NoopEvidenceRecorder suppresses external transcript writes in fake-owner
// integration tests. It is never wired by BAS runtime construction.
func NoopEvidenceRecorder() basaccess.EvidenceRecorder { return evidenceRecorder{} }
