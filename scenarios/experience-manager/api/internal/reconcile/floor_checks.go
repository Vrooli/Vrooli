package reconcile

import (
	"fmt"
	"strings"

	"experience-manager/internal/spec"
)

func claimFailureCode(claimType string) string {
	switch claimType {
	case "no-document-horizontal-overflow":
		return spec.CodeFloorNoDocOverflow
	case "viewport-fill":
		return spec.CodeFloorViewportFill
	case "chrome-pinned":
		return spec.CodeFloorChromePinned
	case "safe-area-tap-targets":
		return spec.CodeFloorSafeArea
	case "single-line-chrome":
		return spec.CodeFloorSingleLine
	case "tap-target-size":
		return spec.CodeFloorTapTargetSize
	case "affordance-present":
		return spec.CodeAffordanceMissing
	default:
		return spec.CodeClaimFailed
	}
}

func isBaselineFloorType(claimType string) bool {
	return claimFailureCode(claimType) != spec.CodeClaimFailed
}

func snapshotRoot(nodes []*AXNode) *AXNode {
	if len(nodes) == 0 {
		return nil
	}
	return nodes[0]
}

func firstHorizontalOverflowNode(nodes []*AXNode, target CaptureTarget) *AXNode {
	if target.ViewportWidth <= 0 {
		return nil
	}
	limit := float64(target.ViewportWidth) + 2
	for _, node := range nodes {
		if node == nil || node.Bounds == nil || node.Bounds.Width <= 0 || node.Bounds.Height <= 0 || isTextOnlyNode(node) || isPreviewWorkspaceScrollNode(node) || !verticallyIntersectsViewport(node.Bounds, target) {
			continue
		}
		if node.Bounds.X < -2 || node.Bounds.X+node.Bounds.Width > limit {
			return node
		}
	}
	return nil
}

func viewportFill(root *AXNode, target CaptureTarget) bool {
	if root == nil || root.Bounds == nil || target.ViewportHeight <= 0 || target.ViewportWidth <= 0 {
		return false
	}
	return root.Bounds.Width >= float64(target.ViewportWidth)*0.98 && root.Bounds.Height >= float64(target.ViewportHeight)*0.98
}

func firstUnpinnedChromeNode(nodes []*AXNode, target CaptureTarget) *AXNode {
	if target.ViewportHeight <= 0 || target.ViewportWidth <= 0 {
		return nil
	}
	for _, node := range nodes {
		if !isChromeNode(node) || node.Bounds == nil {
			continue
		}
		if !intersectsViewport(node.Bounds, target) {
			continue
		}
		if node.Bounds.X < -1 || node.Bounds.Y < -1 || node.Bounds.X+node.Bounds.Width > float64(target.ViewportWidth)+1 || node.Bounds.Y+node.Bounds.Height > float64(target.ViewportHeight)+1 {
			return node
		}
	}
	return nil
}

func firstUnsafeAreaTapTarget(nodes []*AXNode, target CaptureTarget) *AXNode {
	if target.ViewportID != "mobile" || target.ViewportHeight <= 0 {
		return nil
	}
	const bottomUnsafeInset = 24.0
	limit := float64(target.ViewportHeight) - bottomUnsafeInset
	for _, node := range nodes {
		if !isInteractiveNode(node) || node.Bounds == nil || !intersectsViewport(node.Bounds, target) {
			continue
		}
		if !isAppChromeControlTestID(node.DOM.TestID) {
			continue
		}
		if node.Bounds.Y+node.Bounds.Height > limit {
			return node
		}
	}
	return nil
}

func firstMultilineChromeLabel(nodes []*AXNode, target CaptureTarget) *AXNode {
	for _, node := range nodes {
		if !isChromeLabelNode(node, target) || node.Bounds == nil {
			continue
		}
		if strings.Contains(node.Name, "\n") || node.Bounds.Height > 72 {
			return node
		}
	}
	return nil
}

func firstUndersizedTapTarget(nodes []*AXNode, target CaptureTarget) *AXNode {
	if target.ViewportID != "mobile" {
		return nil
	}
	const minTarget = 44.0
	for _, node := range nodes {
		if !isInteractiveNode(node) || node.Bounds == nil || !intersectsViewport(node.Bounds, target) {
			continue
		}
		if node.Bounds.Width < minTarget || node.Bounds.Height < minTarget {
			return node
		}
	}
	return nil
}

func isChromeNode(node *AXNode) bool {
	if node == nil {
		return false
	}
	if isAppChromeContainerTestID(node.DOM.TestID) {
		return true
	}
	switch strings.ToLower(node.Role) {
	case "banner", "navigation", "menubar", "toolbar", "tablist", "contentinfo":
		return true
	case "sectionheader":
		// Card and specimen headers are common inside scrollable preview
		// workspaces. Only a named application chrome marker is a floor target.
		return isAppChromeContainerTestID(node.DOM.TestID)
	default:
		return strings.EqualFold(node.DOM.Tag, "nav") || strings.EqualFold(node.DOM.Tag, "footer")
	}
}

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
	// These nodes are descendants of the intentionally scrollable preview
	// canvas. Their bounds are measured in the emulated device coordinate
	// space, not the document viewport, so they must not be reported as page
	// overflow or off-screen controls.
	return strings.HasPrefix(testID, "components-editor-gallery") ||
		strings.HasPrefix(testID, "components-editor-example-")
}

func isChromeLabelNode(node *AXNode, target CaptureTarget) bool {
	if node == nil || strings.TrimSpace(node.Name) == "" || node.Bounds == nil {
		return false
	}
	if isAppChromeControlTestID(node.DOM.TestID) {
		return true
	}
	switch strings.ToLower(node.Role) {
	case "button", "link", "tab", "menuitem":
		return node.Bounds.Y <= 120
	default:
		return false
	}
}

func isAppChromeContainerTestID(testID string) bool {
	switch testID {
	case "layout-top-bar", "layout-sidebar", "layout-bottom-nav", "status-header", "mobile-header", "mobile-nav", "workspace-header":
		return true
	default:
		return false
	}
}

func isAppChromeControlTestID(testID string) bool {
	return strings.HasPrefix(testID, "layout-sidebar-link-") ||
		strings.HasPrefix(testID, "layout-bottom-nav-link-") ||
		strings.HasPrefix(testID, "mobile-nav-")
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

func intersectsViewport(bounds *Bounds, target CaptureTarget) bool {
	if bounds == nil || target.ViewportHeight <= 0 || target.ViewportWidth <= 0 {
		return false
	}
	return bounds.X+bounds.Width > 0 &&
		bounds.Y+bounds.Height > 0 &&
		bounds.X < float64(target.ViewportWidth) &&
		bounds.Y < float64(target.ViewportHeight)
}

func verticallyIntersectsViewport(bounds *Bounds, target CaptureTarget) bool {
	if bounds == nil || target.ViewportHeight <= 0 {
		return false
	}
	return bounds.Y+bounds.Height > 0 && bounds.Y < float64(target.ViewportHeight)
}

func boundsInsideViewport(bounds *Bounds, target CaptureTarget) bool {
	if bounds == nil || target.ViewportHeight <= 0 || target.ViewportWidth <= 0 {
		return false
	}
	return bounds.X >= -1 &&
		bounds.Y >= -1 &&
		bounds.X+bounds.Width <= float64(target.ViewportWidth)+1 &&
		bounds.Y+bounds.Height <= float64(target.ViewportHeight)+1
}

func describeBoundsFailure(prefix string, node *AXNode, target CaptureTarget) string {
	if node == nil || node.Bounds == nil {
		return prefix
	}
	name := strings.TrimSpace(node.Name)
	if name == "" {
		name = strings.TrimSpace(node.DOM.TestID)
	}
	if name == "" {
		name = strings.TrimSpace(node.DOM.Tag)
	}
	if name == "" {
		name = strings.TrimSpace(node.Role)
	}
	return fmt.Sprintf("%s: %s bounds x=%.1f y=%.1f w=%.1f h=%.1f viewport=%dx%d",
		prefix,
		name,
		node.Bounds.X,
		node.Bounds.Y,
		node.Bounds.Width,
		node.Bounds.Height,
		target.ViewportWidth,
		target.ViewportHeight,
	)
}
