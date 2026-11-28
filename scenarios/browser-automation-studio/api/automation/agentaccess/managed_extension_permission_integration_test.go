package agentaccess

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
)

func newManagedPermissionProbeFixture(t *testing.T) (*extensionArtifactHarness, ExtensionArtifact) {
	t.Helper()
	artifact, archive := makeManagedExtensionArtifact(t)
	h := newExtensionArtifactHarness(t, artifact, nil)
	artifactOwner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/extension/download/"+artifact.ArtifactRef {
			http.Error(w, "unexpected artifact request", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(archive)
	}))
	t.Cleanup(artifactOwner.Close)
	stagingRoot := t.TempDir()
	t.Cleanup(func() { makeExtensionTreeWritable(stagingRoot) })
	controller, err := NewManagedExtensionController(
		h.broker.manager,
		artifactOwner.URL,
		WithManagedExtensionStagingRoot(filepath.Join(stagingRoot, "staged-extensions")),
	)
	if err != nil {
		t.Fatal(err)
	}
	h.broker.extensions = controller
	ctx := withBridgeContextCaller(context.Background(), h.callerID)
	if _, err := h.broker.InstallExtension(ctx, h.entry.sessionID, "", artifact.ArtifactRef, artifact.Digest); err != nil {
		t.Fatalf("install exact permission-probe artifact: %v", err)
	}
	return h, artifact
}

func managedProbeEvidence(observedAt time.Time) *sessionpermission.ExtensionPermissionEvidence {
	evidence := permissionEvidence(sessionpermission.ExtensionPermissionEvidenceEffective, sessionpermission.ExtensionPermissionSourceChromeGetAll, observedAt.UTC())
	return &evidence
}

func TestManagedPermissionProbeOnlyReturnsEffectiveForExactEnabledArtifactAndOwner(t *testing.T) {
	h, artifact := newManagedPermissionProbeFixture(t)
	// A runtime grant set may be narrower than the manifest declaration. Keep
	// this deliberately empty to prove the Broker does not substitute declared
	// values when the exact same-context query reports no effective grants.
	h.driverInventory.override.Store(&managedDriverInventoryOverride{
		Permissions:     []string{},
		HostPermissions: []string{},
	})
	ctx := withBridgeContextCaller(context.Background(), h.callerID)
	inventory, err := h.broker.SelectedTabExtensions(ctx, h.entry.sessionID)
	if err != nil || len(inventory.Extensions) != 1 {
		t.Fatalf("inventory=%#v err=%v", inventory, err)
	}
	item := inventory.Extensions[0]
	binding, err := h.broker.extensionSessionBinding(h.entry)
	if err != nil {
		t.Fatal(err)
	}
	if inventory.Binding != binding || inventory.SessionID != binding.SessionID || item.ID != artifact.ExtensionID || item.PermissionEvidence.Status != sessionpermission.ExtensionPermissionEvidenceEffective || item.PermissionEvidence.Source != extensionPermissionProbeChromeGetAllV1 || item.PermissionEvidence.Scope != sessionpermission.ExtensionPermissionScopeGrantSetOnly || item.PermissionEvidence.PerTabSiteAccess != sessionpermission.ExtensionPerTabSiteAccessNotMeasured || item.PermissionEvidence.ObservedAt.IsZero() {
		t.Fatalf("effective result was not tied to exact current session/profile/owner/artifact: inventory=%#v item=%#v binding=%#v", inventory, item, binding)
	}
	if !item.Enabled || !item.ArtifactVerified || item.ArtifactSHA256 != artifact.Digest || item.ManifestSHA256 != artifact.ManifestSHA256 || len(item.Permissions) != 0 || len(item.HostPermissions) != 0 || !containsString(item.DeclaredPermissions, "storage") || !containsString(item.DeclaredHostPermissions, "https://example.test/*") {
		t.Fatalf("effective result did not match the installed enabled artifact: %#v", item)
	}
}

func TestManagedPermissionProbeRejectsWrongArtifactStateAndStaleReport(t *testing.T) {
	tests := []struct {
		name     string
		override *managedDriverInventoryOverride
	}{
		{
			name:     "wrong artifact path",
			override: &managedDriverInventoryOverride{Path: filepath.Join(t.TempDir(), "another-artifact"), Evidence: managedProbeEvidence(time.Now())},
		},
		{
			name:     "wrong artifact version",
			override: &managedDriverInventoryOverride{Version: "9.9.9", Evidence: managedProbeEvidence(time.Now())},
		},
		{
			name:     "disabled extension",
			override: &managedDriverInventoryOverride{Enabled: managedProbeBoolPtr(false), Evidence: managedProbeEvidence(time.Now())},
		},
		{
			name:     "stale permission report",
			override: &managedDriverInventoryOverride{Evidence: managedProbeEvidence(time.Now().Add(-2 * time.Minute))},
		},
		{
			name: "unsupported report provenance",
			override: &managedDriverInventoryOverride{Evidence: &sessionpermission.ExtensionPermissionEvidence{
				Status: sessionpermission.ExtensionPermissionEvidenceEffective, Source: "manifest.declared_permissions",
				Scope: sessionpermission.ExtensionPermissionScopeGrantSetOnly, PerTabSiteAccess: sessionpermission.ExtensionPerTabSiteAccessNotMeasured, ObservedAt: time.Now().UTC(),
			}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, _ := newManagedPermissionProbeFixture(t)
			h.driverInventory.override.Store(tc.override)
			ctx := withBridgeContextCaller(context.Background(), h.callerID)
			inventory, err := h.broker.SelectedTabExtensions(ctx, h.entry.sessionID)
			if err == nil || len(inventory.Extensions) != 0 {
				t.Fatalf("invalid runtime permission report was released: inventory=%#v err=%v", inventory, err)
			}
		})
	}
}

func TestManagedPermissionProbeRejectsWrongSessionOrUnownedArtifactID(t *testing.T) {
	t.Run("wrong session", func(t *testing.T) {
		h, _ := newManagedPermissionProbeFixture(t)
		h.driverInventory.override.Store(&managedDriverInventoryOverride{SessionID: "different-session"})
		ctx := withBridgeContextCaller(context.Background(), h.callerID)
		inventory, err := h.broker.SelectedTabExtensions(ctx, h.entry.sessionID)
		if err == nil || len(inventory.Extensions) != 0 {
			t.Fatalf("wrong-session inventory was released: inventory=%#v err=%v", inventory, err)
		}
	})
	t.Run("unowned extension ID", func(t *testing.T) {
		h, artifact := newManagedPermissionProbeFixture(t)
		otherID := strings.Repeat("a", 32)
		if otherID == artifact.ExtensionID {
			otherID = strings.Repeat("b", 32)
		}
		h.driverInventory.override.Store(&managedDriverInventoryOverride{ExtensionID: otherID, Evidence: managedProbeEvidence(time.Now())})
		ctx := withBridgeContextCaller(context.Background(), h.callerID)
		inventory, err := h.broker.SelectedTabExtensions(ctx, h.entry.sessionID)
		if err != nil || len(inventory.Extensions) != 0 {
			t.Fatalf("unowned extension permission report was disclosed: inventory=%#v err=%v", inventory, err)
		}
	})
}

func TestManagedPermissionProbeRechecksProfileRevocationAndHostileURL(t *testing.T) {
	t.Run("wrong profile", func(t *testing.T) {
		h, _ := newManagedPermissionProbeFixture(t)
		h.entry.profile.ProfileID = "another-profile"
		before := h.driverInventory.calls.Load()
		ctx := withBridgeContextCaller(context.Background(), h.callerID)
		if _, err := h.broker.SelectedTabExtensions(ctx, h.entry.sessionID); err == nil {
			t.Fatal("query under a different profile was accepted")
		}
		if h.driverInventory.calls.Load() != before {
			t.Fatal("wrong-profile query reached the browser owner")
		}
	})
	t.Run("wrong caller", func(t *testing.T) {
		h, _ := newManagedPermissionProbeFixture(t)
		before := h.driverInventory.calls.Load()
		ctx := withBridgeContextCaller(context.Background(), "different-trusted-caller")
		if _, err := h.broker.SelectedTabExtensions(ctx, h.entry.sessionID); err == nil {
			t.Fatal("inventory query by another caller was accepted")
		}
		if h.driverInventory.calls.Load() != before {
			t.Fatal("wrong-caller query reached the browser owner")
		}
	})
	t.Run("revoked during query", func(t *testing.T) {
		h, _ := newManagedPermissionProbeFixture(t)
		started := make(chan struct{}, 1)
		release := make(chan struct{})
		h.driverInventory.override.Store(&managedDriverInventoryOverride{HoldStarted: started, HoldRelease: release})
		ctx := withBridgeContextCaller(context.Background(), h.callerID)
		done := make(chan error, 1)
		go func() {
			_, err := h.broker.SelectedTabExtensions(ctx, h.entry.sessionID)
			done <- err
		}()
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatal("query did not reach the exact session owner")
		}
		h.policy.revoke()
		close(release)
		select {
		case err := <-done:
			if err == nil {
				t.Fatal("effective permission report was released after grant revocation")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("inventory query did not complete after its owner was released")
		}
	})
	t.Run("hostile out-of-scope URL", func(t *testing.T) {
		h, _ := newManagedPermissionProbeFixture(t)
		before := h.driverInventory.calls.Load()
		h.url.Store("https://hostile.example/private")
		ctx := withBridgeContextCaller(context.Background(), h.callerID)
		if _, err := h.broker.SelectedTabExtensions(ctx, h.entry.sessionID); err == nil {
			t.Fatal("permission report query ran while the managed tab was outside its URL profile")
		}
		if h.driverInventory.calls.Load() != before {
			t.Fatal("out-of-scope permission query reached browser inventory")
		}
	})
}

func managedProbeBoolPtr(value bool) *bool { return &value }
