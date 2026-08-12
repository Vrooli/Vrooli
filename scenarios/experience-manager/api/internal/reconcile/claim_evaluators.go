package reconcile

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"experience-manager/internal/spec"
)

type claimEvaluation struct {
	Pass         bool
	AXNodeJSON   string
	Failure      string
	Unverifiable string
}

type claimEvaluatorFunc func(spec.PageDocument, spec.Claim, CaptureTarget, []*AXNode) claimEvaluation

// claimEvaluators is the live set of deterministic structure checkers. It is
// the authority for which claim types can actually pass; ImplementedClaimTypes
// exposes its keys so the capability registry can derive checker coverage
// instead of asserting it.
var claimEvaluators = map[string]claimEvaluatorFunc{
	"no-document-horizontal-overflow": evaluateNoDocumentHorizontalOverflowClaim,
	"viewport-fill":                   evaluateViewportFillClaim,
	"chrome-pinned":                   evaluateChromePinnedClaim,
	"safe-area-tap-targets":           evaluateSafeAreaTapTargetsClaim,
	"single-line-chrome":              evaluateSingleLineChromeClaim,
	"tap-target-size":                 evaluateTapTargetSizeClaim,
	"state-covered":                   evaluateStateCoveredClaim,
	"state-distinct":                  evaluateStateDistinctClaim,
	"element-present":                 evaluateElementPresenceClaim,
	"element-absent":                  evaluateElementAbsentClaim,
	"single-dominant-action":          evaluateSingleDominantActionClaim,
	"keyboard-reachable":              evaluateElementPresenceClaim,
	"accessible-name":                 evaluateAccessibleNameClaim,
	"affordance-present":              evaluateAffordancePresentClaim,
	"announced":                       evaluateAnnouncedClaim,
	"error-association":               evaluateErrorAssociationClaim,
	"focus-contained":                 evaluateFocusContainedClaim,
	"focus-restored":                  evaluateFocusRestoredClaim,
	"heading-hierarchy":               evaluateHeadingHierarchyClaim,
	"layered-dismissal":               evaluateLayeredDismissalClaim,
	"spacing":                         evaluateSpacingClaim,
	"state-contrast":                  evaluateStateContrastClaim,
	"size-parity":                     evaluateSizeParityClaim,
	"visible-without-scroll":          evaluateVisibleWithoutScrollClaim,
	"reading-order":                   evaluateReadingOrderClaim,
}

func claimEvaluator(claimType string) claimEvaluatorFunc {
	return claimEvaluators[claimType]
}

func evaluateElementAbsentClaim(page spec.PageDocument, claim spec.Claim, _ CaptureTarget, nodes []*AXNode) claimEvaluation {
	if len(claim.Elements) != 1 {
		return claimEvaluation{Unverifiable: "element-absent requires exactly one declared element"}
	}
	node := findBoundNode(nodes, page.Bindings.Elements[claim.Elements[0]], elementRole(page, claim.Elements[0]))
	if node == nil {
		return claimEvaluation{Pass: true}
	}
	return claimEvaluation{Pass: false, AXNodeJSON: encodeAXNode(node), Failure: fmt.Sprintf("declared absent element %q is present", claim.Elements[0])}
}

func evaluateSingleDominantActionClaim(page spec.PageDocument, claim spec.Claim, _ CaptureTarget, nodes []*AXNode) claimEvaluation {
	if len(claim.Elements) == 0 {
		return claimEvaluation{Unverifiable: "single-dominant-action requires declared action elements"}
	}
	minimum, ok := numericParam(claim.Params, "minimumAreaRatio", "minAreaRatio")
	if !ok || minimum < 1 {
		minimum = 1.1
	}
	target := findBoundNode(nodes, page.Bindings.Elements[claim.Elements[0]], elementRole(page, claim.Elements[0]))
	if target == nil || target.Bounds == nil || strings.TrimSpace(computedStyleValue(target, "font-size")) == "" {
		return claimEvaluation{Unverifiable: "single-dominant-action requires bounds and computed font-size evidence"}
	}
	targetArea := target.Bounds.Width * target.Bounds.Height
	for _, element := range claim.Elements[1:] {
		other := findBoundNode(nodes, page.Bindings.Elements[element], elementRole(page, element))
		if other == nil || other.Bounds == nil {
			return claimEvaluation{Unverifiable: "single-dominant-action requires bounds for every declared action"}
		}
		otherArea := other.Bounds.Width * other.Bounds.Height
		if targetArea+0.01 < otherArea*minimum {
			return claimEvaluation{Pass: false, AXNodeJSON: encodeAXNode(target), Failure: fmt.Sprintf("action %q is not dominant over %q at %.2fx area", claim.Elements[0], element, targetArea/otherArea)}
		}
	}
	return claimEvaluation{Pass: true, AXNodeJSON: encodeAXNode(target)}
}

func evaluateSpacingClaim(page spec.PageDocument, claim spec.Claim, _ CaptureTarget, nodes []*AXNode) claimEvaluation {
	if len(claim.Elements) != 2 {
		return claimEvaluation{Unverifiable: "spacing requires exactly two declared elements"}
	}
	minimum, ok := numericParam(claim.Params, "minSeparation", "minGap")
	if !ok || minimum < 0 {
		return claimEvaluation{Unverifiable: "spacing requires params.minSeparation"}
	}
	first := findBoundNode(nodes, page.Bindings.Elements[claim.Elements[0]], elementRole(page, claim.Elements[0]))
	second := findBoundNode(nodes, page.Bindings.Elements[claim.Elements[1]], elementRole(page, claim.Elements[1]))
	if first == nil || second == nil {
		return claimEvaluation{Unverifiable: "spacing requires both declared elements"}
	}
	axis := strings.ToLower(strings.TrimSpace(claimParamString(claim.Params, "axis")))
	if axis == "" {
		axis = "inline"
	}
	horizontal, vertical := 0.0, 0.0
	if first.Bounds != nil && second.Bounds != nil {
		horizontal = intervalGap(first.Bounds.X, first.Bounds.X+first.Bounds.Width, second.Bounds.X, second.Bounds.X+second.Bounds.Width)
		vertical = intervalGap(first.Bounds.Y, first.Bounds.Y+first.Bounds.Height, second.Bounds.Y, second.Bounds.Y+second.Bounds.Height)
	} else if gap, ok := cssPixels(computedStyleValue(first, "gap")); ok {
		horizontal, vertical = gap, gap
	} else {
		return claimEvaluation{Unverifiable: "spacing requires bounds or computed gap evidence for both declared elements"}
	}
	gap := horizontal
	if axis == "block" || (axis == "any" && vertical > horizontal) {
		gap = vertical
	}
	if axis != "inline" && axis != "block" && axis != "any" {
		return claimEvaluation{Unverifiable: fmt.Sprintf("spacing axis %q is unsupported", axis)}
	}
	if gap+0.01 < minimum {
		return claimEvaluation{Pass: false, AXNodeJSON: encodeAXNode(second), Failure: fmt.Sprintf("declared elements %q and %q are separated by %.1fpx, below %.1fpx", claim.Elements[0], claim.Elements[1], gap, minimum)}
	}
	return claimEvaluation{Pass: true, AXNodeJSON: encodeAXNode(second)}
}

func evaluateStateContrastClaim(page spec.PageDocument, claim spec.Claim, target CaptureTarget, nodes []*AXNode) claimEvaluation {
	if len(claim.Elements) == 0 {
		return claimEvaluation{Unverifiable: "state-contrast requires a declared control element"}
	}
	state := claimParamString(claim.Params, "state")
	if state == "" {
		state = target.StateID
	}
	minimum, ok := numericParam(claim.Params, "minContrastRatio", "minContrast")
	if !ok || minimum < 1 {
		return claimEvaluation{Unverifiable: "state-contrast requires params.minContrastRatio"}
	}
	control := findBoundNode(nodes, page.Bindings.Elements[claim.Elements[0]], elementRole(page, claim.Elements[0]))
	if control == nil {
		return claimEvaluation{Unverifiable: "state-contrast requires computed appearance evidence for the control"}
	}
	foreground, background := computedStyleValue(control, "color"), computedStyleValue(control, "background-color")
	if appearance, exists := appearanceState(control, state); exists {
		foreground, background = firstNonEmpty(appearance.Foreground, foreground), firstNonEmpty(appearance.Background, background)
	}
	backgroundRef := claimParamString(claim.Params, "background", "backgroundElement")
	if backgroundNode := findBoundNode(nodes, page.Bindings.Elements[backgroundRef], elementRole(page, backgroundRef)); backgroundNode != nil {
		background = firstNonEmpty(computedStyleValue(backgroundNode, "background-color"), background)
	}
	if strings.TrimSpace(foreground) == "" || strings.TrimSpace(background) == "" {
		return claimEvaluation{Unverifiable: fmt.Sprintf("computed appearance evidence for state %q is incomplete", state)}
	}
	ratio, err := contrastRatio(foreground, background)
	if err != nil {
		return claimEvaluation{Unverifiable: fmt.Sprintf("computed appearance colors are invalid: %v", err)}
	}
	if ratio+0.001 < minimum {
		return claimEvaluation{Pass: false, AXNodeJSON: encodeAXNode(control), Failure: fmt.Sprintf("state %q contrast is %.2f:1, below %.2f:1", state, ratio, minimum)}
	}
	return claimEvaluation{Pass: true, AXNodeJSON: encodeAXNode(control)}
}

func evaluateSizeParityClaim(page spec.PageDocument, claim spec.Claim, _ CaptureTarget, nodes []*AXNode) claimEvaluation {
	if len(claim.Elements) != 2 {
		return claimEvaluation{Unverifiable: "size-parity requires exactly two declared elements"}
	}
	first := findBoundNode(nodes, page.Bindings.Elements[claim.Elements[0]], elementRole(page, claim.Elements[0]))
	second := findBoundNode(nodes, page.Bindings.Elements[claim.Elements[1]], elementRole(page, claim.Elements[1]))
	if first == nil || second == nil {
		return claimEvaluation{Unverifiable: "size-parity requires both declared elements"}
	}
	tolerance := 1.0
	if value, exists := numericParam(claim.Params, "tolerance"); exists {
		tolerance = value
	}
	firstHeight, firstOK := nodeHeight(first)
	secondHeight, secondOK := nodeHeight(second)
	if !firstOK || !secondOK {
		return claimEvaluation{Unverifiable: "size-parity requires bounds or computed height evidence for both declared elements"}
	}
	delta := math.Abs(firstHeight - secondHeight)
	if delta > tolerance+0.01 {
		return claimEvaluation{Pass: false, AXNodeJSON: encodeAXNode(second), Failure: fmt.Sprintf("declared elements %q and %q differ by %.1fpx in height, above %.1fpx tolerance", claim.Elements[0], claim.Elements[1], delta, tolerance)}
	}
	return claimEvaluation{Pass: true, AXNodeJSON: encodeAXNode(second)}
}

func evaluateAnnouncedClaim(page spec.PageDocument, claim spec.Claim, _ CaptureTarget, nodes []*AXNode) claimEvaluation {
	if len(claim.Elements) != 1 {
		return claimEvaluation{Unverifiable: "announced requires exactly one live-region element"}
	}
	node := findBoundNode(nodes, page.Bindings.Elements[claim.Elements[0]], elementRole(page, claim.Elements[0]))
	if node == nil {
		return claimEvaluation{Unverifiable: "announced requires a bound live-region element"}
	}
	if node.Role != "status" && node.Role != "alert" && !hasStatePrefix(node, "live=") {
		return claimEvaluation{Pass: false, AXNodeJSON: encodeAXNode(node), Failure: "announced element is not a status, alert, or live region"}
	}
	return claimEvaluation{Pass: true, AXNodeJSON: encodeAXNode(node)}
}

func evaluateErrorAssociationClaim(page spec.PageDocument, claim spec.Claim, _ CaptureTarget, nodes []*AXNode) claimEvaluation {
	if len(claim.Elements) != 2 {
		return claimEvaluation{Unverifiable: "error-association requires a control and error element"}
	}
	control := findBoundNode(nodes, page.Bindings.Elements[claim.Elements[0]], elementRole(page, claim.Elements[0]))
	errorNode := findBoundNode(nodes, page.Bindings.Elements[claim.Elements[1]], elementRole(page, claim.Elements[1]))
	if control == nil || errorNode == nil {
		return claimEvaluation{Unverifiable: "error-association requires both bound elements"}
	}
	if !hasStatePrefix(control, "invalid=") && !hasState(control, "invalid") {
		return claimEvaluation{Pass: false, AXNodeJSON: encodeAXNode(control), Failure: "control does not expose an invalid state"}
	}
	if errorNode.Role != "alert" && errorNode.Role != "status" && strings.TrimSpace(errorNode.Name) == "" && strings.TrimSpace(errorNode.Description) == "" {
		return claimEvaluation{Pass: false, AXNodeJSON: encodeAXNode(errorNode), Failure: "associated error has no accessible message"}
	}
	return claimEvaluation{Pass: true, AXNodeJSON: encodeAXNode(errorNode)}
}

func evaluateFocusContainedClaim(page spec.PageDocument, claim spec.Claim, _ CaptureTarget, nodes []*AXNode) claimEvaluation {
	if len(claim.Elements) != 1 {
		return claimEvaluation{Unverifiable: "focus-contained requires exactly one focus scope"}
	}
	scope := findBoundNode(nodes, page.Bindings.Elements[claim.Elements[0]], elementRole(page, claim.Elements[0]))
	if scope == nil || scope.Bounds == nil {
		return claimEvaluation{Unverifiable: "focus-contained requires bounds for the focus scope"}
	}
	focused := 0
	for _, node := range nodes {
		if !hasState(node, "focused") {
			continue
		}
		focused++
		if node.Bounds == nil || !containsBounds(scope.Bounds, node.Bounds) {
			return claimEvaluation{Pass: false, AXNodeJSON: encodeAXNode(node), Failure: "focused node escapes the declared focus scope"}
		}
	}
	if focused == 0 {
		return claimEvaluation{Unverifiable: "focus-contained requires a captured focused node"}
	}
	return claimEvaluation{Pass: true, AXNodeJSON: encodeAXNode(scope)}
}

func evaluateFocusRestoredClaim(page spec.PageDocument, claim spec.Claim, target CaptureTarget, nodes []*AXNode) claimEvaluation {
	if len(claim.Elements) != 1 {
		return claimEvaluation{Unverifiable: "focus-restored requires the restoring control"}
	}
	control := findBoundNode(nodes, page.Bindings.Elements[claim.Elements[0]], elementRole(page, claim.Elements[0]))
	if control == nil {
		return claimEvaluation{Unverifiable: "focus-restored requires a bound restoring control"}
	}
	if target.InteractionState != "rest" && target.InteractionState != "focus-visible" {
		return claimEvaluation{Unverifiable: "focus-restored requires the post-dismissal capture state"}
	}
	if !hasState(control, "focused") && target.InteractionState != "focus-visible" {
		return claimEvaluation{Pass: false, AXNodeJSON: encodeAXNode(control), Failure: "focus was not restored to the declared control"}
	}
	return claimEvaluation{Pass: true, AXNodeJSON: encodeAXNode(control)}
}

func evaluateHeadingHierarchyClaim(_ spec.PageDocument, _ spec.Claim, _ CaptureTarget, nodes []*AXNode) claimEvaluation {
	previous := 0
	var first *AXNode
	for _, node := range nodes {
		if node.Role != "heading" {
			continue
		}
		level, ok := stateInt(node, "level=")
		if !ok {
			return claimEvaluation{Unverifiable: "heading-hierarchy requires heading levels in the accessibility tree"}
		}
		if first == nil {
			first = node
		}
		if previous > 0 && level > previous+1 {
			return claimEvaluation{Pass: false, AXNodeJSON: encodeAXNode(node), Failure: fmt.Sprintf("heading level jumps from %d to %d", previous, level)}
		}
		previous = level
	}
	if first == nil {
		return claimEvaluation{Unverifiable: "heading-hierarchy requires at least one heading"}
	}
	return claimEvaluation{Pass: true, AXNodeJSON: encodeAXNode(first)}
}

func evaluateLayeredDismissalClaim(page spec.PageDocument, claim spec.Claim, _ CaptureTarget, nodes []*AXNode) claimEvaluation {
	if len(claim.Elements) < 1 {
		return claimEvaluation{Unverifiable: "layered-dismissal requires one or more dismissible layers"}
	}
	visibleDialogs := 0
	for _, node := range nodes {
		if node.Role == "dialog" && node.Bounds != nil {
			visibleDialogs++
		}
	}
	if visibleDialogs == 0 {
		return claimEvaluation{Unverifiable: "layered-dismissal requires a captured dialog layer"}
	}
	return claimEvaluation{Pass: true, AXNodeJSON: encodeAXNode(findBoundNode(nodes, page.Bindings.Elements[claim.Elements[0]], elementRole(page, claim.Elements[0])))}
}

func hasState(node *AXNode, wanted string) bool {
	for _, state := range node.States {
		if state == wanted {
			return true
		}
	}
	return false
}

func hasStatePrefix(node *AXNode, prefix string) bool {
	for _, state := range node.States {
		if strings.HasPrefix(state, prefix) {
			return true
		}
	}
	return false
}

func stateInt(node *AXNode, prefix string) (int, bool) {
	for _, state := range node.States {
		if strings.HasPrefix(state, prefix) {
			value, err := strconv.Atoi(strings.TrimPrefix(state, prefix))
			return value, err == nil
		}
	}
	return 0, false
}

func containsBounds(container, child *Bounds) bool {
	return child.X >= container.X && child.Y >= container.Y && child.X+child.Width <= container.X+container.Width+0.01 && child.Y+child.Height <= container.Y+container.Height+0.01
}

func intervalGap(firstStart, firstEnd, secondStart, secondEnd float64) float64 {
	if firstEnd < secondStart {
		return secondStart - firstEnd
	}
	if secondEnd < firstStart {
		return firstStart - secondEnd
	}
	return 0
}

func computedStyleValue(node *AXNode, property string) string {
	if node == nil {
		return ""
	}
	if node.ComputedStyle != nil {
		if value := strings.TrimSpace(node.ComputedStyle[property]); value != "" {
			return value
		}
	}
	if node.Appearance == nil {
		return ""
	}
	switch property {
	case "color":
		return node.Appearance.Foreground
	case "background-color":
		return node.Appearance.Background
	case "font-size":
		return node.Appearance.FontSize
	case "line-height":
		return node.Appearance.LineHeight
	case "margin":
		return node.Appearance.Margin
	case "padding":
		return node.Appearance.Padding
	case "font-weight":
		return node.Appearance.FontWeight
	default:
		return ""
	}
}

func appearanceState(node *AXNode, state string) (AppearanceState, bool) {
	if node == nil || node.Appearance == nil {
		return AppearanceState{}, false
	}
	appearance, ok := node.Appearance.States[strings.ToLower(strings.TrimSpace(state))]
	return appearance, ok
}

func cssPixels(value string) (float64, bool) {
	value = strings.TrimSpace(value)
	if value == "" || value == "normal" {
		return 0, false
	}
	value = strings.TrimSuffix(value, "px")
	number, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return number, err == nil
}

func nodeHeight(node *AXNode) (float64, bool) {
	if node == nil {
		return 0, false
	}
	if node.Bounds != nil {
		return node.Bounds.Height, true
	}
	return cssPixels(computedStyleValue(node, "height"))
}

func numericParam(params map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		switch value := params[key].(type) {
		case float64:
			return value, true
		case float32:
			return float64(value), true
		case int:
			return float64(value), true
		case int64:
			return float64(value), true
		case json.Number:
			parsed, err := value.Float64()
			if err == nil {
				return parsed, true
			}
		}
	}
	return 0, false
}

func claimParamString(params map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := params[key].(string); ok {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

var colorComponentRE = regexp.MustCompile(`[-+]?(?:\d*\.\d+|\d+\.?\d*)%?`)

func contrastRatio(foreground, background string) (float64, error) {
	fg, err := parseColor(foreground)
	if err != nil {
		return 0, err
	}
	bg, err := parseColor(background)
	if err != nil {
		return 0, err
	}
	fgL := relativeLuminance(fg)
	bgL := relativeLuminance(bg)
	if fgL < bgL {
		fgL, bgL = bgL, fgL
	}
	return (fgL + 0.05) / (bgL + 0.05), nil
}

type rgbColor struct{ r, g, b float64 }

func parseColor(raw string) (rgbColor, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if strings.HasPrefix(value, "#") {
		hex := strings.TrimPrefix(value, "#")
		if len(hex) == 3 {
			hex = fmt.Sprintf("%c%c%c%c%c%c", hex[0], hex[0], hex[1], hex[1], hex[2], hex[2])
		}
		if len(hex) != 6 {
			return rgbColor{}, fmt.Errorf("unsupported hex color %q", raw)
		}
		var channels [3]uint64
		for i := range channels {
			parsed, err := strconv.ParseUint(hex[i*2:i*2+2], 16, 8)
			if err != nil {
				return rgbColor{}, fmt.Errorf("unsupported color %q", raw)
			}
			channels[i] = parsed
		}
		return rgbColor{float64(channels[0]) / 255, float64(channels[1]) / 255, float64(channels[2]) / 255}, nil
	}
	if strings.HasPrefix(value, "rgb(") || strings.HasPrefix(value, "rgba(") {
		start, end := strings.IndexByte(value, '('), strings.LastIndexByte(value, ')')
		if start < 0 || end <= start {
			return rgbColor{}, fmt.Errorf("unsupported color %q", raw)
		}
		matches := colorComponentRE.FindAllString(value[start+1:end], -1)
		if len(matches) < 3 {
			return rgbColor{}, fmt.Errorf("unsupported color %q", raw)
		}
		var channels [3]float64
		for i := range channels {
			parsed, err := strconv.ParseFloat(strings.TrimSuffix(matches[i], "%"), 64)
			if err != nil {
				return rgbColor{}, fmt.Errorf("unsupported color %q", raw)
			}
			if strings.HasSuffix(matches[i], "%") {
				parsed *= 2.55
			}
			channels[i] = parsed / 255
		}
		return rgbColor{channels[0], channels[1], channels[2]}, nil
	}
	return rgbColor{}, fmt.Errorf("unsupported color %q", raw)
}

func relativeLuminance(color rgbColor) float64 {
	linear := func(channel float64) float64 {
		if channel <= 0.03928 {
			return channel / 12.92
		}
		return math.Pow((channel+0.055)/1.055, 2.4)
	}
	return 0.2126*linear(color.r) + 0.7152*linear(color.g) + 0.0722*linear(color.b)
}

func evaluateNoDocumentHorizontalOverflowClaim(_ spec.PageDocument, _ spec.Claim, target CaptureTarget, nodes []*AXNode) claimEvaluation {
	node := firstHorizontalOverflowNode(nodes, target)
	if node == nil {
		return claimEvaluation{Pass: true}
	}
	return claimEvaluation{Pass: false, AXNodeJSON: encodeAXNode(node), Failure: describeBoundsFailure("node extends horizontally outside the viewport", node, target)}
}

func evaluateViewportFillClaim(_ spec.PageDocument, _ spec.Claim, target CaptureTarget, nodes []*AXNode) claimEvaluation {
	root := snapshotRoot(nodes)
	if viewportFill(root, target) {
		return claimEvaluation{Pass: true, AXNodeJSON: encodeAXNode(root)}
	}
	return claimEvaluation{Pass: false, AXNodeJSON: encodeAXNode(root), Failure: describeBoundsFailure("root surface does not fill the viewport", root, target)}
}

func evaluateChromePinnedClaim(_ spec.PageDocument, _ spec.Claim, target CaptureTarget, nodes []*AXNode) claimEvaluation {
	node := firstUnpinnedChromeNode(nodes, target)
	if node == nil {
		return claimEvaluation{Pass: true}
	}
	return claimEvaluation{Pass: false, AXNodeJSON: encodeAXNode(node), Failure: describeBoundsFailure("chrome node sits outside the captured viewport", node, target)}
}

func evaluateSafeAreaTapTargetsClaim(_ spec.PageDocument, _ spec.Claim, target CaptureTarget, nodes []*AXNode) claimEvaluation {
	node := firstUnsafeAreaTapTarget(nodes, target)
	if node == nil {
		return claimEvaluation{Pass: true}
	}
	return claimEvaluation{Pass: false, AXNodeJSON: encodeAXNode(node), Failure: describeBoundsFailure("interactive target overlaps the mobile unsafe bottom area", node, target)}
}

func evaluateSingleLineChromeClaim(_ spec.PageDocument, _ spec.Claim, target CaptureTarget, nodes []*AXNode) claimEvaluation {
	node := firstMultilineChromeLabel(nodes, target)
	if node == nil {
		return claimEvaluation{Pass: true}
	}
	return claimEvaluation{Pass: false, AXNodeJSON: encodeAXNode(node), Failure: describeBoundsFailure("chrome label appears wrapped or too tall for a single-line control", node, target)}
}

func evaluateTapTargetSizeClaim(_ spec.PageDocument, _ spec.Claim, target CaptureTarget, nodes []*AXNode) claimEvaluation {
	node := firstUndersizedTapTarget(nodes, target)
	if node == nil {
		return claimEvaluation{Pass: true}
	}
	return claimEvaluation{Pass: false, AXNodeJSON: encodeAXNode(node), Failure: describeBoundsFailure("mobile interactive target is smaller than 44px", node, target)}
}

func evaluateStateCoveredClaim(_ spec.PageDocument, claim spec.Claim, target CaptureTarget, _ []*AXNode) claimEvaluation {
	return claimEvaluation{Pass: claimTargetsState(claim, target.StateID)}
}

func evaluateStateDistinctClaim(_ spec.PageDocument, claim spec.Claim, target CaptureTarget, _ []*AXNode) claimEvaluation {
	if len(claim.States) < 2 {
		return claimEvaluation{}
	}
	seen := map[string]string{}
	for _, state := range claim.States {
		stateID := state
		if stateID == "" {
			stateID = "default"
		}
		fingerprint := target.StateFingerprints[stateID]
		if fingerprint == "" {
			return claimEvaluation{Unverifiable: "state " + stateID + " was not captured for distinct-state comparison"}
		}
		if other, ok := seen[fingerprint]; ok {
			return claimEvaluation{Pass: false, Failure: fmt.Sprintf("states %q and %q produced the same accessibility fingerprint", other, stateID)}
		}
		seen[fingerprint] = stateID
	}
	return claimEvaluation{Pass: true}
}

func evaluateElementPresenceClaim(page spec.PageDocument, claim spec.Claim, _ CaptureTarget, nodes []*AXNode) claimEvaluation {
	if len(claim.Elements) == 0 {
		return claimEvaluation{}
	}
	pass := true
	var axNodeJSON string
	for _, elementID := range claim.Elements {
		node := findBoundNode(nodes, page.Bindings.Elements[elementID], elementRole(page, elementID))
		if node == nil {
			pass = false
			continue
		}
		axNodeJSON = encodeAXNode(node)
		if claim.Type == "keyboard-reachable" && !node.KeyboardReachable() {
			pass = false
		}
	}
	return claimEvaluation{Pass: pass, AXNodeJSON: axNodeJSON}
}

// evaluateAccessibleNameClaim proves only explicitly declared name intent. It
// does not scan for generic WCAG defects; the expected label comes from the
// element contract (or claim.params.name for an intentional override).
func evaluateAccessibleNameClaim(page spec.PageDocument, claim spec.Claim, _ CaptureTarget, nodes []*AXNode) claimEvaluation {
	if len(claim.Elements) == 0 {
		return claimEvaluation{Unverifiable: "accessible-name requires at least one declared element"}
	}
	override := paramString(claim.Params, "name")
	for _, elementID := range claim.Elements {
		node := findBoundNode(nodes, page.Bindings.Elements[elementID], elementRole(page, elementID))
		if node == nil {
			return claimEvaluation{Pass: false, Failure: "declared element " + elementID + " was not found in the accessibility snapshot"}
		}
		expected := override
		if expected == "" {
			for _, element := range page.Elements {
				if element.ID == elementID {
					expected = element.Name
					break
				}
			}
		}
		if strings.TrimSpace(expected) == "" {
			return claimEvaluation{Unverifiable: "accessible-name requires element.name or claim.params.name"}
		}
		if !strings.EqualFold(strings.TrimSpace(node.Name), strings.TrimSpace(expected)) {
			return claimEvaluation{Pass: false, AXNodeJSON: encodeAXNode(node), Failure: fmt.Sprintf("accessible name %q does not match declared name %q", node.Name, expected)}
		}
	}
	return claimEvaluation{Pass: true}
}

func evaluateVisibleWithoutScrollClaim(page spec.PageDocument, claim spec.Claim, target CaptureTarget, nodes []*AXNode) claimEvaluation {
	if len(claim.Elements) == 0 {
		return claimEvaluation{}
	}
	pass := true
	var axNodeJSON string
	for _, elementID := range claim.Elements {
		node := findBoundNode(nodes, page.Bindings.Elements[elementID], elementRole(page, elementID))
		if node == nil || node.Bounds == nil || !boundsInsideViewport(node.Bounds, target) {
			pass = false
			continue
		}
		axNodeJSON = encodeAXNode(node)
	}
	return claimEvaluation{Pass: pass, AXNodeJSON: axNodeJSON}
}

func evaluateReadingOrderClaim(page spec.PageDocument, claim spec.Claim, _ CaptureTarget, nodes []*AXNode) claimEvaluation {
	if len(claim.Elements) <= 1 {
		return claimEvaluation{}
	}
	pass := true
	last := -1
	var axNodeJSON string
	for _, elementID := range claim.Elements {
		idx := findBoundIndex(nodes, page.Bindings.Elements[elementID], elementRole(page, elementID))
		if idx < 0 || idx < last {
			pass = false
		}
		if idx >= 0 {
			axNodeJSON = encodeAXNode(nodes[idx])
		}
		last = idx
	}
	return claimEvaluation{Pass: pass, AXNodeJSON: axNodeJSON}
}

func containsState(node *AXNode, state string) bool {
	for _, candidate := range node.States {
		if strings.EqualFold(candidate, state) {
			return true
		}
	}
	return false
}

func pageWithBaselineClaims(page spec.PageDocument) spec.PageDocument {
	optedOut := map[string]bool{}
	for _, optOut := range page.FloorOptOuts {
		optedOut[optOut.Floor] = true
	}
	for _, floor := range baselineFloorClaims() {
		if optedOut[floor.Type] || hasClaimType(page, floor.Type) {
			continue
		}
		page.Claims = append(page.Claims, floor)
	}
	return page
}

func baselineFloorClaims() []spec.Claim {
	return []spec.Claim{
		{ID: "floor-no-document-horizontal-overflow", Type: "no-document-horizontal-overflow", Statement: "The page never creates document-level horizontal scrolling at captured viewports.", Tier: "machine", States: []string{"default"}},
		{ID: "floor-viewport-fill", Type: "viewport-fill", Statement: "The page surface fills the captured viewport instead of collapsing short content.", Tier: "machine", States: []string{"default"}},
		{ID: "floor-chrome-pinned", Type: "chrome-pinned", Statement: "Application chrome remains inside the viewport at captured viewports.", Tier: "machine", States: []string{"default"}},
		{ID: "floor-safe-area-tap-targets", Type: "safe-area-tap-targets", Statement: "Mobile tap targets stay outside unsafe device-edge interaction zones.", Tier: "machine", States: []string{"default"}, Viewports: []string{"mobile"}},
		{ID: "floor-single-line-chrome", Type: "single-line-chrome", Statement: "Navigation and chrome labels render as single-line controls at captured viewports.", Tier: "machine", States: []string{"default"}},
		{ID: "floor-tap-target-size", Type: "tap-target-size", Statement: "Mobile interactive controls expose comfortable touch targets.", Tier: "machine", States: []string{"default"}, Viewports: []string{"mobile"}},
	}
}

func componentWithBaselineClaims(component spec.ComponentDocument) spec.ComponentDocument {
	optedOut := map[string]bool{}
	for _, optOut := range component.FloorOptOuts {
		optedOut[optOut.Floor] = true
	}
	for _, floor := range componentBaselineFloorClaims() {
		if optedOut[floor.Type] || componentHasClaimType(component, floor.Type) {
			continue
		}
		component.Claims = append(component.Claims, floor)
	}
	return component
}

func componentBaselineFloorClaims() []spec.Claim {
	return []spec.Claim{
		{ID: "floor-no-component-horizontal-overflow", Type: "no-document-horizontal-overflow", Statement: "The component harness stage does not create horizontal overflow at captured viewports.", Tier: "machine"},
		{ID: "floor-component-tap-target-size", Type: "tap-target-size", Statement: "Interactive component examples expose comfortable mobile touch targets.", Tier: "machine", Viewports: []string{"mobile"}},
	}
}

func hasClaimType(page spec.PageDocument, claimType string) bool {
	for _, claim := range page.Claims {
		if claim.Type == claimType {
			return true
		}
	}
	return false
}

func componentHasClaimType(component spec.ComponentDocument, claimType string) bool {
	for _, claim := range component.Claims {
		if claim.Type == claimType {
			return true
		}
	}
	return false
}
