package reconcile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"

	uxcapturepb "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/uxcapture"
)

const (
	ex9BundleSchema       = "ux-capture.v1"
	ex9AccessibilityKind  = "accessibility-snapshot"
	ex9PixelsKind         = "pixels"
	ex9FullPagePixelsKind = "full-page-pixels"
)

// EX9ArtifactPayloads contains bytes resolved for the declared paths in one
// EX9 manifest. Keeping the manifest and payloads separate makes the
// digest-bound artifact boundary explicit and testable without a live BAS.
type EX9ArtifactPayloads map[string][]byte

// SnapshotFromEX9Bundle adapts an EX9 capture into the evidence shape that
// experience-manager's retained spec checkers consume. It deliberately
// refuses implicit schema coercions: the accessibility artifact must carry
// the existing BAS snapshot contract until EX9 formalizes that lens payload.
func SnapshotFromEX9Bundle(target CaptureTarget, bundle *uxcapturepb.Bundle, payloads EX9ArtifactPayloads) (Snapshot, error) {
	if bundle == nil {
		return Snapshot{}, fmt.Errorf("EX9 bundle is required")
	}
	if bundle.GetSchemaVersion() != ex9BundleSchema {
		return Snapshot{}, fmt.Errorf("unsupported EX9 bundle schema %q", bundle.GetSchemaVersion())
	}
	if strings.TrimSpace(bundle.GetId()) == "" || bundle.GetSource() == nil || bundle.GetSession() == nil || strings.TrimSpace(bundle.GetSession().GetId()) == "" || bundle.GetViewport() == nil {
		return Snapshot{}, fmt.Errorf("EX9 bundle identity, source, session identity, and viewport are required")
	}
	if bundle.GetViewport().GetWidth() <= 0 || bundle.GetViewport().GetHeight() <= 0 {
		return Snapshot{}, fmt.Errorf("EX9 bundle viewport dimensions must be positive")
	}
	source := bundle.GetSource()
	if source.GetKind() != "scenario" || source.GetScenario() != target.Scenario || normalizeCaptureRoute(source.GetRoute()) != normalizeCaptureRoute(target.Route) {
		return Snapshot{}, fmt.Errorf("EX9 bundle source does not match target scenario and route")
	}
	if target.ViewportWidth > 0 && int(bundle.GetViewport().GetWidth()) != target.ViewportWidth || target.ViewportHeight > 0 && int(bundle.GetViewport().GetHeight()) != target.ViewportHeight {
		return Snapshot{}, fmt.Errorf("EX9 bundle viewport does not match target %dx%d", target.ViewportWidth, target.ViewportHeight)
	}
	if bundle.GetPrivacy() == nil || bundle.GetPrivacy().GetRedactionStatus() != "complete" || bundle.GetPrivacy().GetExternal() {
		return Snapshot{}, fmt.Errorf("EX9 bundle must have complete redaction before machine evaluation")
	}
	for _, lensName := range []string{"L0", "L1", "L2", "L3", "L4", "L5", "L6"} {
		lens, exists := bundle.GetLenses()[lensName]
		if !exists || lens == nil || (lens.GetStatus() != "present" && lens.GetStatus() != "partial" && lens.GetStatus() != "absent") {
			return Snapshot{}, fmt.Errorf("EX9 bundle has no valid %s lens status", lensName)
		}
		if lens.GetStatus() != "present" && strings.TrimSpace(lens.GetReason()) == "" {
			return Snapshot{}, fmt.Errorf("EX9 bundle %s lens omits its absence/partial reason", lensName)
		}
	}

	var accessibility *uxcapturepb.Artifact
	var pixels *uxcapturepb.Artifact
	seen := make(map[string]struct{}, len(bundle.GetArtifacts()))
	for _, artifact := range bundle.GetArtifacts() {
		if artifact == nil || !safeEX9ArtifactPath(artifact.GetPath()) {
			return Snapshot{}, fmt.Errorf("EX9 bundle contains an invalid artifact path")
		}
		if _, duplicate := seen[artifact.GetPath()]; duplicate {
			return Snapshot{}, fmt.Errorf("EX9 bundle declares duplicate artifact path %q", artifact.GetPath())
		}
		seen[artifact.GetPath()] = struct{}{}
		switch artifact.GetKind() {
		case ex9AccessibilityKind:
			if accessibility != nil {
				return Snapshot{}, fmt.Errorf("EX9 bundle declares multiple accessibility snapshots")
			}
			accessibility = artifact
		case ex9PixelsKind:
			if pixels == nil {
				pixels = artifact
			}
		case ex9FullPagePixelsKind:
			if pixels == nil {
				pixels = artifact
			}
		}
	}
	if accessibility == nil {
		return Snapshot{}, fmt.Errorf("EX9 bundle has no accessibility-snapshot artifact")
	}
	accessibilityBytes, err := verifiedEX9Payload(accessibility, payloads)
	if err != nil {
		return Snapshot{}, fmt.Errorf("resolve EX9 accessibility snapshot: %w", err)
	}
	var snapshot Snapshot
	if err := json.Unmarshal(accessibilityBytes, &snapshot); err != nil {
		return Snapshot{}, fmt.Errorf("decode EX9 accessibility snapshot: %w", err)
	}
	if snapshot.Contract != snapshotContract || snapshot.Root.Role == "" {
		return Snapshot{}, fmt.Errorf("EX9 accessibility artifact must contain a rooted %q snapshot", snapshotContract)
	}
	if strings.TrimSpace(snapshot.URL) == "" {
		snapshot.URL = source.GetUrl()
		if snapshot.URL == "" {
			snapshot.URL = source.Route
		}
	}
	if pixels != nil {
		if _, err := verifiedEX9Payload(pixels, payloads); err != nil {
			return Snapshot{}, fmt.Errorf("resolve EX9 screenshot artifact: %w", err)
		}
		snapshot.ScreenshotRef = ex9ArtifactReference(bundle.GetId(), pixels)
	}
	return snapshot, nil
}

func verifiedEX9Payload(artifact *uxcapturepb.Artifact, payloads EX9ArtifactPayloads) ([]byte, error) {
	if artifact == nil {
		return nil, fmt.Errorf("artifact declaration is missing")
	}
	content, ok := payloads[artifact.GetPath()]
	if !ok {
		return nil, fmt.Errorf("payload %q is missing", artifact.GetPath())
	}
	if int64(len(content)) != artifact.GetSizeBytes() {
		return nil, fmt.Errorf("payload %q size mismatch", artifact.GetPath())
	}
	digest := sha256.Sum256(content)
	if !strings.EqualFold(hex.EncodeToString(digest[:]), artifact.GetSha256()) {
		return nil, fmt.Errorf("payload %q digest mismatch", artifact.GetPath())
	}
	return content, nil
}

func safeEX9ArtifactPath(value string) bool {
	if value == "" || strings.ContainsRune(value, '\\') || strings.HasPrefix(value, "/") || path.Clean(value) != value {
		return false
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

func ex9ArtifactReference(bundleID string, artifact *uxcapturepb.Artifact) string {
	segments := strings.Split(artifact.GetPath(), "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}
	return "ux-capture://" + url.PathEscape(bundleID) + "/" + strings.Join(segments, "/") + "#sha256=" + strings.ToLower(artifact.GetSha256())
}

func normalizeCaptureRoute(route string) string {
	route = strings.TrimSpace(route)
	if route == "" {
		return "/"
	}
	return route
}
