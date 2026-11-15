package visualhealth

import (
	"fmt"
	"strings"

	visualpb "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth"
)

// geometryFindings is the sole owner of browser-layout geometry rules. It
// accepts the versioned VisualStepArtifact layout shape, which other
// evaluators can produce from a captured accessibility snapshot without
// reimplementing these thresholds.
func geometryFindings(step *visualpb.VisualStepArtifact, snap layoutSnapshot) []*visualpb.VisualFinding {
	var findings []*visualpb.VisualFinding
	if !snap.hasViewport() {
		findings = append(findings, &visualpb.VisualFinding{
			Code: "visual_geometry_unmeasured", Severity: visualpb.VisualSeverity_VISUAL_SEVERITY_INFO, Category: categoryLayout,
			Message: "browser layout geometry is incomplete", Location: locationFor(step),
			Evidence: "a positive viewport is required", Remediation: "Retain viewport dimensions in layout_json or the VisualStepArtifact viewport.", StepId: step.GetStepId(),
		})
		return findings
	}
	if snap.ViewportWidth > 0 && snap.DocumentWidth > 0 {
		overflow := max(0, snap.DocumentWidth-snap.ViewportWidth)
		if overflow > snap.OverflowSlackPx {
			findings = append(findings, &visualpb.VisualFinding{
				Code: "visual_viewport_overflow", Severity: severityError, Category: categoryLayout,
				Message: "layout width exceeds the visible viewport", Location: locationFor(step),
				Evidence:    fmt.Sprintf("document width %.0fpx exceeds viewport width %.0fpx", snap.DocumentWidth, snap.ViewportWidth),
				Remediation: "Constrain page width to the viewport and move wide content into an intentional scroll container.",
				StepId:      step.GetStepId(), Metrics: []*visualpb.VisualMetric{metric("overflow_px", overflow)},
			})
		}
	}
	if len(snap.Elements) == 0 {
		if snap.DocumentWidth == 0 {
			findings = append(findings, &visualpb.VisualFinding{
				Code: "visual_geometry_unmeasured", Severity: visualpb.VisualSeverity_VISUAL_SEVERITY_INFO, Category: categoryLayout,
				Message: "browser layout geometry is incomplete", Location: locationFor(step),
				Evidence: "element bounds and document width are absent", Remediation: "Retain element bounds or document width in layout_json.", StepId: step.GetStepId(),
			})
		}
		return findings
	}

	chromeDescendants := chromeDescendantIndexes(snap.Elements, snap.ChromeScope)
	for index, el := range snap.Elements {
		if !el.visible() {
			continue
		}
		if el.isInteractive() && !el.InScrollContainer && snap.hasViewport() && el.offscreen(snap) && !snap.sameNodeReachedByKeyboard(el) {
			findings = append(findings, &visualpb.VisualFinding{
				Code: "visual_offscreen_interactive", Severity: severityError, Category: categoryLayout,
				Message:  "interactive element is outside the visible viewport",
				Location: firstNonEmpty(el.NodeKey, el.Selector, locationFor(step)), Evidence: el.describeRect(),
				Remediation: "Keep interactive controls inside the visible viewport or inside reachable scroll containers.", StepId: step.GetStepId(),
			})
		}
		if snap.hasViewport() && !el.InScrollContainer && el.overflowsViewport(snap) {
			findings = append(findings, &visualpb.VisualFinding{
				Code: "visual_viewport_overflow", Severity: severityError, Category: categoryLayout,
				Message:  "visible element extends beyond the captured viewport",
				Location: firstNonEmpty(el.Selector, locationFor(step)), Evidence: el.describeRect(),
				Remediation: "Constrain page width to the viewport and move wide content into an intentional scroll container.", StepId: step.GetStepId(),
			})
		}
		if snap.hasViewport() && isChromeElement(el, snap.ChromeScope) && !boundsInsideLayoutViewport(el, snap) && layoutIntersectsViewport(el, snap) {
			findings = append(findings, &visualpb.VisualFinding{
				Code: "visual_chrome_outside_viewport", Severity: severityError, Category: categoryLayout,
				Message:  "application chrome extends beyond the captured viewport",
				Location: firstNonEmpty(el.NodeKey, el.Selector, locationFor(step)), Evidence: el.describeRect(),
				Remediation: "Keep application navigation and chrome inside the captured viewport.", StepId: step.GetStepId(),
			})
		}
		if snap.mobileLike() && el.isInteractive() && !el.Disabled && (el.Width < minMobileTapTargetPx || el.Height < minMobileTapTargetPx) {
			findings = append(findings, &visualpb.VisualFinding{
				Code: "visual_touch_target_too_small", Severity: severityError, Category: categoryLayout,
				Message:     "interactive control is smaller than the mobile touch target floor",
				Location:    firstNonEmpty(el.NodeKey, el.Selector, locationFor(step)),
				Evidence:    fmt.Sprintf("target %.1fx%.1fpx; minimum %dpx in both dimensions", el.Width, el.Height, minMobileTapTargetPx),
				Remediation: "Give mobile controls at least 44 by 44 CSS pixels of interactive area, including icon-only controls.", StepId: step.GetStepId(),
			})
		}
		scope := snap.UnsafeEdgeScope
		if edge := el.unsafeSafeAreaEdge(snap); edge != "" && (scope != "app-chrome-controls" || isAppChromeControlTestID(el.TestID)) {
			findings = append(findings, &visualpb.VisualFinding{
				Code: "visual_unsafe_edge_tap_zone", Severity: severityError, Category: categoryLayout,
				Message:  "interactive element overlaps an unsafe edge or notch zone",
				Location: firstNonEmpty(el.Selector, locationFor(step)), Evidence: fmt.Sprintf("%s overlaps %s safe area", el.describeRect(), edge),
				Remediation: "Move interactive controls out of unsafe edge and notch zones or pad the layout with env(safe-area-inset-*).", StepId: step.GetStepId(),
			})
		}
		if isChromeLabel(el, chromeDescendants[index]) && (strings.Contains(el.Text, "\n") || el.Height > 72) {
			findings = append(findings, &visualpb.VisualFinding{
				Code: "visual_multiline_chrome_label", Severity: visualpb.VisualSeverity_VISUAL_SEVERITY_WARNING, Category: categoryLayout,
				Message:  "navigation or chrome label appears wrapped or too tall for a single-line control",
				Location: firstNonEmpty(el.NodeKey, el.Selector, locationFor(step)), Evidence: el.describeRect(),
				Remediation: "Keep navigation and chrome labels on one line; shorten or restructure controls when labels do not fit.", StepId: step.GetStepId(),
			})
		}
	}

	root := layoutSurfaceRoot(snap.Elements)
	if root == nil {
		findings = append(findings, &visualpb.VisualFinding{
			Code: "visual_surface_root_unmeasured", Severity: visualpb.VisualSeverity_VISUAL_SEVERITY_INFO, Category: categoryLayout,
			Message: "root application surface geometry was not identified", Location: locationFor(step),
			Evidence: "no surfaceRoot, document, html, or body element has measured bounds", Remediation: "Identify the root application surface in layout_json to measure viewport fill.", StepId: step.GetStepId(),
		})
	} else if root.Width < snap.ViewportWidth*0.98 || root.Height < snap.ViewportHeight*0.98 {
		findings = append(findings, &visualpb.VisualFinding{
			Code: "visual_surface_does_not_fill_viewport", Severity: severityError, Category: categoryLayout,
			Message:  "root application surface does not fill the captured viewport",
			Location: firstNonEmpty(root.NodeKey, root.Selector, locationFor(step)), Evidence: root.describeRect(),
			Remediation: "Make the root application surface fill at least 98% of the captured viewport in both dimensions.", StepId: step.GetStepId(),
		})
	}
	return findings
}

func isAppChromeControlTestID(testID string) bool {
	return strings.HasPrefix(testID, "layout-sidebar-link-") ||
		strings.HasPrefix(testID, "layout-bottom-nav-link-") ||
		strings.HasPrefix(testID, "mobile-nav-")
}

func layoutSurfaceRoot(elements []layoutElement) *layoutElement {
	for index := range elements {
		if elements[index].SurfaceRoot {
			return &elements[index]
		}
	}
	for index := range elements {
		if elements[index].Role == "document" || elements[index].Tag == "html" || elements[index].Tag == "body" {
			return &elements[index]
		}
	}
	return nil
}

func isChromeElement(el layoutElement, scope string) bool {
	if isAppChromeTestID(el.TestID) {
		return true
	}
	switch el.Role {
	case "banner", "navigation", "menubar", "toolbar", "tablist", "contentinfo":
		return true
	}
	switch el.Tag {
	case "nav", "footer":
		return true
	case "header":
		return scope != "application-chrome"
	}
	return false
}

func isChromeLabel(el layoutElement, insideChrome bool) bool {
	if strings.TrimSpace(el.Text) == "" || el.Width <= 0 || el.Height <= 0 {
		return false
	}
	if isAppChromeTestID(el.TestID) {
		return true
	}
	if !insideChrome {
		return false
	}
	switch el.Role {
	case "button", "link", "tab", "menuitem":
		return true
	default:
		return false
	}
}

func isAppChromeTestID(testID string) bool {
	switch testID {
	case "layout-top-bar", "layout-sidebar", "layout-bottom-nav", "status-header", "mobile-header", "mobile-nav", "workspace-header":
		return true
	}
	return strings.HasPrefix(testID, "layout-sidebar-link-") ||
		strings.HasPrefix(testID, "layout-bottom-nav-link-") ||
		strings.HasPrefix(testID, "mobile-nav-")
}

func chromeDescendantIndexes(elements []layoutElement, scope string) map[int]bool {
	bySelector := make(map[string]int, len(elements))
	for index, el := range elements {
		if el.Selector != "" {
			bySelector[el.Selector] = index
		}
	}
	chrome := make(map[int]bool)
	for index, el := range elements {
		if isChromeElement(el, scope) {
			chrome[index] = true
		}
	}
	for index, el := range elements {
		parent := el.ParentSelector
		for steps := 0; parent != "" && steps < len(elements); steps++ {
			ancestorIndex, ok := bySelector[parent]
			if !ok {
				break
			}
			if chrome[ancestorIndex] {
				chrome[index] = true
				break
			}
			parent = elements[ancestorIndex].ParentSelector
		}
	}
	return chrome
}

func boundsInsideLayoutViewport(el layoutElement, snap layoutSnapshot) bool {
	return el.X >= -1 && el.Y >= -1 && el.X+el.Width <= snap.ViewportWidth+1 && el.Y+el.Height <= snap.ViewportHeight+1
}

func layoutIntersectsViewport(el layoutElement, snap layoutSnapshot) bool {
	return el.X+el.Width > 0 && el.Y+el.Height > 0 && el.X < snap.ViewportWidth && el.Y < snap.ViewportHeight
}
