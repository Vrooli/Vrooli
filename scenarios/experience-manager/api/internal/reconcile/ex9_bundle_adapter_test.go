package reconcile

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"experience-manager/internal/spec"
	"github.com/stretchr/testify/require"
	uxcapturepb "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/uxcapture"
)

func TestSnapshotFromEX9BundleAdaptsDigestBoundAccessibilityEvidence(t *testing.T) {
	target := CaptureTarget{Scenario: "demo", Route: "/settings", ViewportWidth: 390, ViewportHeight: 844}
	accessibility := []byte(`{"contract":"bas-accessibility-snapshot/v1","url":"http://localhost/settings","root":{"role":"main","name":"Settings","bounds":{"x":0,"y":0,"width":390,"height":844},"dom":{"testid":"root"},"children":[{"role":"button","name":"Save","states":["focusable"],"bounds":{"x":300,"y":790,"width":72,"height":40},"dom":{"testid":"save","tag":"button","attributes":{"type":"submit"}},"computedStyle":{"color":"rgb(1, 2, 3)"}}]}}`)
	pixels := []byte("png-bytes")
	accessibilityDigest := sha256.Sum256(accessibility)
	pixelsDigest := sha256.Sum256(pixels)
	bundle := &uxcapturepb.Bundle{
		SchemaVersion: "ux-capture.v1", Id: "bundle-1",
		Source:   &uxcapturepb.Source{Kind: "scenario", Scenario: "demo", Route: "/settings"},
		Session:  &uxcapturepb.Session{Id: "session-1", Browser: "chromium"},
		Viewport: &uxcapturepb.Viewport{Width: 390, Height: 844},
		Lenses:   completeEX9TestLenses(),
		Privacy:  &uxcapturepb.Privacy{RedactionStatus: "complete"},
		Artifacts: []*uxcapturepb.Artifact{
			{Kind: ex9AccessibilityKind, Path: "accessibility.json", SizeBytes: int64(len(accessibility)), Sha256: hex.EncodeToString(accessibilityDigest[:])},
			{Kind: ex9PixelsKind, Path: "screen.png", SizeBytes: int64(len(pixels)), Sha256: hex.EncodeToString(pixelsDigest[:])},
		},
	}

	snapshot, err := SnapshotFromEX9Bundle(target, bundle, EX9ArtifactPayloads{"accessibility.json": accessibility, "screen.png": pixels})
	require.NoError(t, err)
	require.Equal(t, snapshotContract, snapshot.Contract)
	require.Equal(t, "http://localhost/settings", snapshot.URL)
	require.Equal(t, "main", snapshot.Root.Role)
	require.Equal(t, "Save", snapshot.Flatten()[1].Name)
	require.True(t, snapshot.Flatten()[1].KeyboardReachable())
	flattened := snapshot.Flatten()
	require.Equal(t, "rgb(1, 2, 3)", computedStyleValue(flattened[1], "color"))
	require.Equal(t, "ux-capture://bundle-1/screen.png#sha256="+hex.EncodeToString(pixelsDigest[:]), snapshot.ScreenshotRef)
}

func TestSnapshotFromEX9BundleRejectsUnusableOrUntrustedEvidence(t *testing.T) {
	accessibility := []byte(`{"contract":"bas-accessibility-snapshot/v1","root":{"role":"main"}}`)
	digest := sha256.Sum256(accessibility)
	base := func() *uxcapturepb.Bundle {
		return &uxcapturepb.Bundle{
			SchemaVersion: "ux-capture.v1", Id: "bundle-1",
			Source:    &uxcapturepb.Source{Kind: "scenario", Scenario: "demo", Route: "/"},
			Session:   &uxcapturepb.Session{Id: "session-1"},
			Viewport:  &uxcapturepb.Viewport{Width: 1280, Height: 720},
			Lenses:    completeEX9TestLenses(),
			Privacy:   &uxcapturepb.Privacy{RedactionStatus: "complete"},
			Artifacts: []*uxcapturepb.Artifact{{Kind: ex9AccessibilityKind, Path: "a11y.json", SizeBytes: int64(len(accessibility)), Sha256: hex.EncodeToString(digest[:])}},
		}
	}
	target := CaptureTarget{Scenario: "demo", Route: "/", ViewportWidth: 1280, ViewportHeight: 720}
	payloads := EX9ArtifactPayloads{"a11y.json": accessibility}

	t.Run("wrong scenario cannot be joined", func(t *testing.T) {
		bundle := base()
		bundle.Source.Scenario = "other"
		_, err := SnapshotFromEX9Bundle(target, bundle, payloads)
		require.ErrorContains(t, err, "source does not match")
	})
	t.Run("partial redaction is not machine evidence", func(t *testing.T) {
		bundle := base()
		bundle.Privacy.RedactionStatus = "partial"
		_, err := SnapshotFromEX9Bundle(target, bundle, payloads)
		require.ErrorContains(t, err, "complete redaction")
	})
	t.Run("digest mismatch cannot become a passing snapshot", func(t *testing.T) {
		bundle := base()
		_, err := SnapshotFromEX9Bundle(target, bundle, EX9ArtifactPayloads{"a11y.json": []byte(`{"contract":"bas-accessibility-snapshot/v1","root":{"role":"xxxx"}}`)})
		require.ErrorContains(t, err, "digest mismatch")
	})
	t.Run("untyped accessibility payload schema is rejected", func(t *testing.T) {
		bundle := base()
		payload := []byte(`{"schema":"unknown/v1","tree":{}}`)
		digest := sha256.Sum256(payload)
		bundle.Artifacts[0].SizeBytes = int64(len(payload))
		bundle.Artifacts[0].Sha256 = hex.EncodeToString(digest[:])
		_, err := SnapshotFromEX9Bundle(target, bundle, EX9ArtifactPayloads{"a11y.json": payload})
		require.ErrorContains(t, err, "rooted")
	})
	t.Run("unsafe artifact path is rejected", func(t *testing.T) {
		bundle := base()
		bundle.Artifacts[0].Path = "../a11y.json"
		_, err := SnapshotFromEX9Bundle(target, bundle, payloads)
		require.ErrorContains(t, err, "invalid artifact path")
	})
}

func TestEX9AdaptedSnapshotPreservesRetainedClaimsAndFloorWaivers(t *testing.T) {
	accessibility := []byte(`{"contract":"bas-accessibility-snapshot/v1","root":{"role":"WebArea","children":[{"role":"button","states":["focusable"],"bounds":{"x":24,"y":80,"width":120,"height":48},"dom":{"testid":"primary-action","tag":"button"}}]}}`)
	digest := sha256.Sum256(accessibility)
	target := CaptureTarget{Scenario: "demo", Route: "/", DocumentKind: "page", PageID: "home", StateID: "default", ViewportID: "desktop", ViewportWidth: 1280, ViewportHeight: 720}
	bundle := &uxcapturepb.Bundle{
		SchemaVersion: "ux-capture.v1", Id: "bundle-1",
		Source:    &uxcapturepb.Source{Kind: "scenario", Scenario: "demo", Route: "/"},
		Session:   &uxcapturepb.Session{Id: "session-1"},
		Viewport:  &uxcapturepb.Viewport{Width: 1280, Height: 720},
		Lenses:    completeEX9TestLenses(),
		Privacy:   &uxcapturepb.Privacy{RedactionStatus: "complete"},
		Artifacts: []*uxcapturepb.Artifact{{Kind: ex9AccessibilityKind, Path: "accessibility/snapshot.json", SizeBytes: int64(len(accessibility)), Sha256: hex.EncodeToString(digest[:])}},
	}
	snapshot, err := SnapshotFromEX9Bundle(target, bundle, EX9ArtifactPayloads{"accessibility/snapshot.json": accessibility})
	require.NoError(t, err)

	report := activeReport("primary-action", spec.Binding{TestID: "primary-action"})
	page := report.Spec.Pages["home"]
	page.FloorOptOuts = []spec.FloorOptOut{{Floor: "viewport-fill", Reason: "This short fixture intentionally does not fill its viewport."}}
	page = pageWithBaselineClaims(page)
	result := reconcileActivePageWithGeometry("experience/pages/home.json", page, target, snapshot, []spec.Finding{})
	require.Empty(t, result.Findings)
	var claimEvidence *Evidence
	for i := range result.Evidence {
		if result.Evidence[i].ClaimID == "primary-action-present" {
			claimEvidence = &result.Evidence[i]
		}
		require.NotEqual(t, "viewport-fill", result.Evidence[i].ClaimType, "authored waiver must suppress only its named baseline floor")
	}
	require.NotNil(t, claimEvidence)
	require.Equal(t, "passed", claimEvidence.Verdict)
	waiverFindings := floorWaiverFindings("experience/pages/home.json", page.FloorOptOuts)
	require.Len(t, waiverFindings, 1)
	require.Equal(t, spec.SeverityWarning, waiverFindings[0].Severity)
	require.True(t, strings.Contains(waiverFindings[0].Message, "short fixture"))
}

func completeEX9TestLenses() map[string]*uxcapturepb.Lens {
	lenses := make(map[string]*uxcapturepb.Lens, 7)
	for _, name := range []string{"L0", "L1", "L2", "L3", "L4", "L5", "L6"} {
		lenses[name] = &uxcapturepb.Lens{Status: "absent", Reason: "fixture has no optional lens payload"}
	}
	lenses["L0"] = &uxcapturepb.Lens{Status: "present"}
	lenses["L2"] = &uxcapturepb.Lens{Status: "present"}
	return lenses
}
