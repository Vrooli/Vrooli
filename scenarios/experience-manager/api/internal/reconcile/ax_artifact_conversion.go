package reconcile

import (
	"strings"

	"experience-manager/internal/spec"
	"github.com/vrooli/api-core/readinessprofile"
)

func baselineGeometryFloor(floor string) (geometryRule string, ok bool) {
	return readinessprofile.GeometryRuleForFloor(floor)
}

func isBaselineFloorType(claimType string) bool {
	_, ok := baselineGeometryFloor(claimType)
	return ok
}

func geometryRuleForFloor(floor string) string {
	geometryRule, _ := baselineGeometryFloor(floor)
	return geometryRule
}

// claimFailureCode maps non-geometry claims to their finding vocabulary. The
// inherited geometry floors are emitted only by ui-health's canonical rules.
func claimFailureCode(claimType string) string {
	if claimType == "affordance-present" {
		return spec.CodeAffordanceMissing
	}
	return spec.CodeClaimFailed
}

func axParents(nodes []*AXNode) map[*AXNode]*AXNode {
	parents := make(map[*AXNode]*AXNode, len(nodes))
	for _, node := range nodes {
		if node == nil {
			continue
		}
		for index := range node.Children {
			parents[&node.Children[index]] = node
		}
	}
	return parents
}

func outsideViewportHorizontally(bounds *Bounds, target CaptureTarget) bool {
	return bounds.X < -2 || bounds.X+bounds.Width > float64(target.ViewportWidth)+2
}

// clippedHorizontallyByAncestor transfers AX overflow evidence into the
// shared artifact's inScrollContainer bit. It does not judge geometry; that is
// owned by ui-health after conversion.
func clippedHorizontallyByAncestor(node *AXNode, parents map[*AXNode]*AXNode, target CaptureTarget) bool {
	for ancestor := parents[node]; ancestor != nil; ancestor = parents[ancestor] {
		if ancestor.Bounds == nil || !clipsHorizontally(ancestor) {
			continue
		}
		return !outsideViewportHorizontally(ancestor.Bounds, target) || clippedHorizontallyByAncestor(ancestor, parents, target)
	}
	return false
}

func clipsHorizontally(node *AXNode) bool {
	value := strings.TrimSpace(computedStyleValue(node, "overflow-x"))
	if value == "" {
		value = strings.TrimSpace(computedStyleValue(node, "overflow"))
	}
	if fields := strings.Fields(strings.ToLower(value)); len(fields) > 0 {
		value = fields[0]
	}
	switch value {
	case "auto", "scroll", "hidden", "clip", "overlay":
		return true
	default:
		return false
	}
}

// isPreviewWorkspaceScrollNode identifies RCL's emulated-device canvas. Its
// child coordinates are device-space evidence; the adapter carries that fact
// as inScrollContainer so the canonical evaluator does not mistake it for page
// overflow.
func isPreviewWorkspaceScrollNode(node *AXNode) bool {
	if node == nil {
		return false
	}
	testID := node.DOM.TestID
	if strings.EqualFold(node.DOM.Tag, "header") {
		return true
	}
	switch testID {
	case "components-editor-gallery", "components-editor-preview-frame", "components-editor-story-picker-item", "components-emulator-viewport", "components-emulator-viewport-frame", "components-emulator-viewport-canvas":
		return true
	}
	return strings.HasPrefix(testID, "components-editor-gallery") || strings.HasPrefix(testID, "components-editor-example-")
}

func isAppChromeControlTestID(testID string) bool {
	return strings.HasPrefix(testID, "layout-sidebar-link-") ||
		strings.HasPrefix(testID, "layout-bottom-nav-link-") ||
		strings.HasPrefix(testID, "mobile-nav-")
}

func isInteractiveNode(node *AXNode) bool {
	if node == nil {
		return false
	}
	switch strings.ToLower(node.Role) {
	case "button", "link", "tab", "checkbox", "combobox", "menuitem", "switch", "textbox":
		return true
	default:
		return false
	}
}

func isTextOnlyNode(node *AXNode) bool {
	if node == nil {
		return false
	}
	switch strings.ToLower(node.Role) {
	case "statictext", "inlinetextbox", "text":
		return true
	default:
		return false
	}
}

func boundsInsideViewport(bounds *Bounds, target CaptureTarget) bool {
	return bounds != nil && target.ViewportWidth > 0 && target.ViewportHeight > 0 &&
		bounds.X >= 0 && bounds.Y >= 0 &&
		bounds.X+bounds.Width <= float64(target.ViewportWidth) &&
		bounds.Y+bounds.Height <= float64(target.ViewportHeight)
}
