package reconcile

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"experience-manager/internal/spec"
)

// archetypeFloors is data-driven so a floor set can be reviewed and changed
// without editing evaluator logic. The empty archetype intentionally retains
// the conservative legacy set while contracts migrate to explicit archetypes.
//
//go:embed archetype_floors.json
var archetypeFloors []byte

type claimEvaluation struct {
	Pass         bool
	AXNodeJSON   string
	Measurement  *ClaimMeasurement
	Failure      string
	Unverifiable string
}

// ClaimMeasurement is the structured, numeric evidence behind a machine
// claim. It is deliberately independent of the human-facing failure message:
// consumers can render, compare, and overlay these values without parsing
// prose.
type ClaimMeasurement struct {
	Metric     string            `json:"metric"`
	Observed   *float64          `json:"observed,omitempty"`
	Required   *float64          `json:"required,omitempty"`
	Unit       string            `json:"unit,omitempty"`
	Comparator string            `json:"comparator,omitempty"`
	Subjects   []MeasuredSubject `json:"subjects"`
}

type MeasuredSubject struct {
	ElementID string  `json:"elementId"`
	TestID    string  `json:"testId,omitempty"`
	Bounds    *Bounds `json:"bounds,omitempty"`
	ContextID string  `json:"contextId,omitempty"`
	Value     string  `json:"value,omitempty"`
}

func measurement(metric, unit, comparator string, observed, required *float64, subjects []MeasuredSubject) *ClaimMeasurement {
	return &ClaimMeasurement{Metric: metric, Unit: unit, Comparator: comparator, Observed: observed, Required: required, Subjects: subjects}
}

func measuredSubjects(page spec.PageDocument, elements []string, nodes []*AXNode) []MeasuredSubject {
	subjects := make([]MeasuredSubject, 0, len(elements))
	for _, elementID := range elements {
		node := findBoundNode(nodes, page.Bindings.Elements[elementID], elementRole(page, elementID))
		subject := MeasuredSubject{ElementID: elementID}
		if node != nil {
			subject.TestID = node.DOM.TestID
			subject.Bounds = node.Bounds
		}
		subjects = append(subjects, subject)
	}
	return subjects
}

func measuredNodeSubject(elementID string, node *AXNode) []MeasuredSubject {
	if node == nil {
		return []MeasuredSubject{{ElementID: elementID}}
	}
	return []MeasuredSubject{{ElementID: elementID, TestID: node.DOM.TestID, Bounds: node.Bounds}}
}

func measurementFailure(m *ClaimMeasurement) string {
	if m == nil {
		return "claim failed without a structured measurement"
	}
	if m.Observed != nil && m.Required != nil {
		comparison := "did not satisfy"
		if m.Comparator == "gte" {
			comparison = "below"
		} else if m.Comparator == "lte" {
			comparison = "above"
		}
		return fmt.Sprintf("%s claim failed: observed %.2f%s %s required %.2f%s", m.Metric, *m.Observed, m.Unit, comparison, *m.Required, m.Unit)
	}
	return fmt.Sprintf("%s claim failed: measured subjects did not satisfy the declared requirement", m.Metric)
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
	"content-not-clipped":             evaluateContentNotClippedClaim,
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
	"font-size":                       evaluateFontSizeClaim,
	"focus-contained":                 evaluateFocusContainedClaim,
	"focus-containment":               evaluateFocusContainedClaim,
	"focus-order":                     evaluateFocusOrderClaim,
	"focus-restored":                  evaluateFocusRestoredClaim,
	"heading-hierarchy":               evaluateHeadingHierarchyClaim,
	"layered-dismissal":               evaluateLayeredDismissalClaim,
	"motion-duration":                 evaluateMotionDurationClaim,
	"no-layout-animation":             evaluateNoLayoutAnimationClaim,
	"responsive-transformation":       evaluateResponsiveTransformationClaim,
	"spacing":                         evaluateSpacingClaim,
	"state-contrast":                  evaluateStateContrastClaim,
	"size-parity":                     evaluateSizeParityClaim,
	"visible-without-scroll":          evaluateVisibleWithoutScrollClaim,
	"transition-class-conformance":    evaluateTransitionClassConformanceClaim,
	"reading-order":                   evaluateReadingOrderClaim,
	"differential":                    evaluateDifferentialWithoutContexts,
	"dark-parity":                     evaluateDifferentialWithoutContexts,
	"token-resolution":                evaluateTokenResolutionClaim,
	"portal-boundary":                 evaluatePortalBoundaryClaim,
	"chrome-color-agreement":          evaluateChromeColorAgreementClaim,
}

func claimEvaluator(claimType string) claimEvaluatorFunc {
	return claimEvaluators[claimType]
}

func evaluateDifferentialWithoutContexts(_ spec.PageDocument, _ spec.Claim, _ CaptureTarget, _ []*AXNode) claimEvaluation {
	return claimEvaluation{Unverifiable: "differential claims require their paired render contexts"}
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
		m := measurement("inline-gap", "px", "gte", &gap, &minimum, measuredSubjects(page, claim.Elements, nodes))
		return claimEvaluation{Pass: false, AXNodeJSON: encodeAXNode(second), Measurement: m, Failure: measurementFailure(m)}
	}
	m := measurement("inline-gap", "px", "gte", &gap, &minimum, measuredSubjects(page, claim.Elements, nodes))
	return claimEvaluation{Pass: true, AXNodeJSON: encodeAXNode(second), Measurement: m}
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
		m := measurement("state-contrast", "ratio", "gte", &ratio, &minimum, measuredSubjects(page, claim.Elements, nodes))
		return claimEvaluation{Pass: false, AXNodeJSON: encodeAXNode(control), Measurement: m, Failure: measurementFailure(m)}
	}
	m := measurement("state-contrast", "ratio", "gte", &ratio, &minimum, measuredSubjects(page, claim.Elements, nodes))
	return claimEvaluation{Pass: true, AXNodeJSON: encodeAXNode(control), Measurement: m}
}

func evaluateSizeParityClaim(page spec.PageDocument, claim spec.Claim, _ CaptureTarget, nodes []*AXNode) claimEvaluation {
	if len(claim.Elements) == 1 && claim.Elements[0] == "control" {
		for _, node := range nodes {
			if node == nil || node.Bounds == nil || node.DOM.Attributes["data-rcl-control"] != "true" {
				continue
			}
			expected := map[string]float64{"xs": 32, "sm": 36, "md": 40, "default": 40, "lg": 44, "xl": 48, "icon": 40}[node.DOM.Attributes["data-control-size"]]
			if expected == 0 {
				return claimEvaluation{Unverifiable: "floor-control-geometry requires a known data-control-size rung"}
			}
			tolerance := 1.0
			if value, exists := numericParam(claim.Params, "tolerance"); exists {
				tolerance = value
			}
			delta := math.Max(math.Abs(node.Bounds.Height-expected), math.Abs(node.Bounds.Width-expected))
			observed := math.Min(node.Bounds.Height, node.Bounds.Width)
			m := measurement("size-parity", "px", "lte", &delta, &tolerance, measuredNodeSubject("control", node))
			if delta > tolerance+0.01 {
				return claimEvaluation{Pass: false, AXNodeJSON: encodeAXNode(node), Measurement: m, Failure: fmt.Sprintf("control rung %q measures %.1fx%.1fpx; expected %.0fpx", node.DOM.Attributes["data-control-size"], node.Bounds.Width, node.Bounds.Height, expected)}
			}
			_ = observed
			return claimEvaluation{Pass: true, AXNodeJSON: encodeAXNode(node), Measurement: m}
		}
		return claimEvaluation{Pass: true}
	}
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
		m := measurement("size-parity", "px", "lte", &delta, &tolerance, measuredSubjects(page, claim.Elements, nodes))
		return claimEvaluation{Pass: false, AXNodeJSON: encodeAXNode(second), Measurement: m, Failure: measurementFailure(m)}
	}
	m := measurement("size-parity", "px", "lte", &delta, &tolerance, measuredSubjects(page, claim.Elements, nodes))
	return claimEvaluation{Pass: true, AXNodeJSON: encodeAXNode(second), Measurement: m}
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

func evaluateTokenResolutionClaim(page spec.PageDocument, claim spec.Claim, _ CaptureTarget, nodes []*AXNode) claimEvaluation {
	if len(claim.Elements) != 1 {
		return claimEvaluation{Unverifiable: "token-resolution requires exactly one declared specimen element"}
	}
	properties := paramStringSlice(claim.Params, "properties")
	if len(properties) == 0 {
		return claimEvaluation{Unverifiable: "token-resolution requires params.properties naming the custom properties to measure"}
	}
	node := findBoundNode(nodes, page.Bindings.Elements[claim.Elements[0]], elementRole(page, claim.Elements[0]))
	if node == nil {
		return claimEvaluation{Pass: false, Failure: "token-resolution specimen was not captured"}
	}
	subjects := make([]MeasuredSubject, 0, len(properties))
	for _, property := range properties {
		property = strings.TrimSpace(property)
		if !strings.HasPrefix(property, "--") {
			return claimEvaluation{Unverifiable: "token-resolution properties must be CSS custom property names beginning with --"}
		}
		value := strings.TrimSpace(computedStyleValue(node, property))
		subjects = append(subjects, MeasuredSubject{ElementID: claim.Elements[0], TestID: node.DOM.TestID, Value: value})
		if value == "" || strings.Contains(value, "var(") {
			return claimEvaluation{Pass: false, AXNodeJSON: encodeAXNode(node), Measurement: measurement("token-resolution", "", "non-empty", nil, nil, subjects), Failure: fmt.Sprintf("custom property %s did not resolve to a concrete value", property)}
		}
	}
	return claimEvaluation{Pass: true, AXNodeJSON: encodeAXNode(node), Measurement: measurement("token-resolution", "", "non-empty", nil, nil, subjects)}
}

func evaluatePortalBoundaryClaim(page spec.PageDocument, claim spec.Claim, _ CaptureTarget, nodes []*AXNode) claimEvaluation {
	if len(claim.Elements) != 2 {
		return claimEvaluation{Unverifiable: "portal-boundary requires application-root and floating-surface elements"}
	}
	app := findBoundNode(nodes, page.Bindings.Elements[claim.Elements[0]], elementRole(page, claim.Elements[0]))
	surface := findBoundNode(nodes, page.Bindings.Elements[claim.Elements[1]], elementRole(page, claim.Elements[1]))
	if app == nil || surface == nil {
		return claimEvaluation{Pass: false, Failure: "portal-boundary requires both bound elements in the accessibility snapshot"}
	}
	if axContainsNode(app, surface) {
		return claimEvaluation{Pass: false, AXNodeJSON: encodeAXNode(surface), Failure: "floating surface remains inside the declared application root"}
	}
	return claimEvaluation{Pass: true, AXNodeJSON: encodeAXNode(surface)}
}

func axContainsNode(root, target *AXNode) bool {
	if root == nil || target == nil {
		return false
	}
	if root == target || (root.DOM.TestID != "" && root.DOM.TestID == target.DOM.TestID) {
		return true
	}
	for index := range root.Children {
		if axContainsNode(&root.Children[index], target) {
			return true
		}
	}
	return false
}

func evaluateFocusOrderClaim(page spec.PageDocument, claim spec.Claim, target CaptureTarget, nodes []*AXNode) claimEvaluation {
	candidates := claimNodes(page, claim, nodes)
	if len(candidates) == 1 {
		candidates = flattenAXNode(candidates[0])
	}
	focusable := make([]*AXNode, 0, len(candidates))
	for _, node := range candidates {
		if node != nil && node.Bounds != nil && node.KeyboardReachable() {
			focusable = append(focusable, node)
		}
	}
	if len(focusable) < 2 {
		return claimEvaluation{Unverifiable: "focus-order requires at least two captured keyboard-reachable elements with bounds"}
	}
	rtl := target.Direction == "rtl" || strings.HasPrefix(strings.ToLower(target.Locale), "ar")
	for index := 1; index < len(focusable); index++ {
		previous, current := focusable[index-1], focusable[index]
		sameRow := math.Abs(previous.Bounds.Y-current.Bounds.Y) < math.Min(previous.Bounds.Height, current.Bounds.Height)/2
		if sameRow {
			ordered := current.Bounds.X >= previous.Bounds.X
			if rtl {
				ordered = current.Bounds.X <= previous.Bounds.X
			}
			if !ordered {
				return claimEvaluation{Pass: false, AXNodeJSON: encodeAXNode(current), Failure: "sequential focus order does not match the captured horizontal reading direction"}
			}
			continue
		}
		if current.Bounds.Y+0.01 < previous.Bounds.Y {
			return claimEvaluation{Pass: false, AXNodeJSON: encodeAXNode(current), Failure: "sequential focus order moves upward before completing the visual reading order"}
		}
	}
	return claimEvaluation{Pass: true, AXNodeJSON: encodeAXNode(focusable[0])}
}

func flattenAXNode(root *AXNode) []*AXNode {
	var out []*AXNode
	var walk func(*AXNode)
	walk = func(node *AXNode) {
		if node == nil {
			return
		}
		out = append(out, node)
		for index := range node.Children {
			walk(&node.Children[index])
		}
	}
	walk(root)
	return out
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

var layoutAnimationProperties = map[string]bool{
	"all": true, "block-size": true, "bottom": true, "height": true,
	"inline-size": true, "left": true, "margin": true, "margin-block": true,
	"margin-inline": true, "padding": true, "padding-block": true,
	"padding-inline": true, "right": true, "top": true, "width": true,
}

func evaluateTransitionClassConformanceClaim(page spec.PageDocument, claim spec.Claim, _ CaptureTarget, nodes []*AXNode) claimEvaluation {
	checked, offending := inspectTransitionProperties(page, claim, nodes)
	if checked == 0 {
		return claimEvaluation{Unverifiable: "transition-class-conformance requires captured transition-property evidence"}
	}
	if len(offending) > 0 {
		return claimEvaluation{Pass: false, Failure: "transition animates layout-affecting properties: " + strings.Join(offending, ", ")}
	}
	return claimEvaluation{Pass: true}
}

func evaluateNoLayoutAnimationClaim(page spec.PageDocument, claim spec.Claim, _ CaptureTarget, nodes []*AXNode) claimEvaluation {
	checked, offending := inspectTransitionProperties(page, claim, nodes)
	if checked == 0 {
		return claimEvaluation{Unverifiable: "no-layout-animation requires captured transition-property evidence"}
	}
	if len(offending) > 0 {
		return claimEvaluation{Pass: false, Failure: "layout-affecting animation properties are present: " + strings.Join(offending, ", ")}
	}
	return claimEvaluation{Pass: true}
}

func inspectTransitionProperties(page spec.PageDocument, claim spec.Claim, nodes []*AXNode) (int, []string) {
	checked := 0
	seen := map[string]bool{}
	for _, node := range claimNodes(page, claim, nodes) {
		value := strings.TrimSpace(computedStyleValue(node, "transition-property"))
		if value == "" {
			continue
		}
		checked++
		for _, property := range strings.Split(value, ",") {
			property = strings.TrimSpace(property)
			if layoutAnimationProperties[property] && !seen[property] {
				seen[property] = true
			}
		}
	}
	offending := make([]string, 0, len(seen))
	for property := range seen {
		offending = append(offending, property)
	}
	sort.Strings(offending)
	return checked, offending
}

func evaluateMotionDurationClaim(page spec.PageDocument, claim spec.Claim, target CaptureTarget, nodes []*AXNode) claimEvaluation {
	maximum := 500.0
	if value, ok := numericParam(claim.Params, "maxMs", "maximumMs"); ok {
		maximum = value
	}
	if target.MotionPreference == "reduce" {
		maximum = 1
	}
	checked := 0
	observed := 0.0
	for _, node := range claimNodes(page, claim, nodes) {
		for _, property := range []string{"transition-duration", "animation-duration"} {
			value := strings.TrimSpace(computedStyleValue(node, property))
			if value == "" {
				continue
			}
			checked++
			for _, item := range strings.Split(value, ",") {
				if duration := cssDurationMilliseconds(item); duration > observed {
					observed = duration
				}
			}
		}
	}
	if checked == 0 {
		return claimEvaluation{Unverifiable: "motion-duration requires captured transition or animation duration evidence"}
	}
	m := measurement("motion-duration", "ms", "lte", &observed, &maximum, nil)
	if observed > maximum+0.01 {
		return claimEvaluation{Pass: false, Measurement: m, Failure: measurementFailure(m)}
	}
	return claimEvaluation{Pass: true, Measurement: m}
}

func cssDurationMilliseconds(value string) float64 {
	value = strings.TrimSpace(value)
	if strings.HasSuffix(value, "ms") {
		parsed, _ := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(value, "ms")), 64)
		return parsed
	}
	if strings.HasSuffix(value, "s") {
		parsed, _ := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(value, "s")), 64)
		return parsed * 1000
	}
	return 0
}

func evaluateResponsiveTransformationClaim(page spec.PageDocument, claim spec.Claim, target CaptureTarget, nodes []*AXNode) claimEvaluation {
	expectedKey := "desktopPresentation"
	if target.ViewportID == "mobile" || target.ViewportWidth > 0 && target.ViewportWidth < 640 {
		expectedKey = "mobilePresentation"
	}
	expected := paramString(claim.Params, expectedKey)
	for _, node := range claimNodes(page, claim, nodes) {
		presentation := firstNonEmptyString(
			node.DOM.Attributes["data-presentation"],
			node.DOM.Attributes["data-rcl-presentation"],
			node.DOM.Attributes["data-responsive-presentation"],
		)
		if presentation == "" {
			continue
		}
		if expected != "" && presentation != expected {
			return claimEvaluation{Pass: false, AXNodeJSON: encodeAXNode(node), Failure: fmt.Sprintf("viewport %q rendered presentation %q, expected %q", target.ViewportID, presentation, expected)}
		}
		return claimEvaluation{Pass: true, AXNodeJSON: encodeAXNode(node)}
	}
	return claimEvaluation{Unverifiable: "responsive-transformation requires a captured presentation data attribute"}
}

func claimNodes(page spec.PageDocument, claim spec.Claim, nodes []*AXNode) []*AXNode {
	if len(claim.Elements) == 0 {
		return nodes
	}
	out := make([]*AXNode, 0, len(claim.Elements))
	for _, elementID := range claim.Elements {
		if node := findBoundNode(nodes, page.Bindings.Elements[elementID], elementRole(page, elementID)); node != nil {
			out = append(out, node)
		}
	}
	return out
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
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

type rgbColor struct{ r, g, b, a float64 }

func evaluateChromeColorAgreementClaim(page spec.PageDocument, claim spec.Claim, _ CaptureTarget, nodes []*AXNode) claimEvaluation {
	if len(claim.Elements) == 0 {
		return claimEvaluation{Unverifiable: "chrome-color-agreement requires a declared status-bar element"}
	}
	referenceID := claimParamString(claim.Params, "reference")
	if referenceID == "" {
		return claimEvaluation{Unverifiable: "chrome-color-agreement requires params.reference"}
	}
	statusID := claim.Elements[0]
	statusNode := findBoundNode(nodes, page.Bindings.Elements[statusID], elementRole(page, statusID))
	referenceNode := findBoundNode(nodes, page.Bindings.Elements[referenceID], elementRole(page, referenceID))
	if statusNode == nil || referenceNode == nil {
		return claimEvaluation{Unverifiable: "chrome-color-agreement requires both bound elements"}
	}
	statusRaw := computedStyleValue(statusNode, "background-color")
	referenceRaw := computedStyleValue(referenceNode, "background-color")
	status, statusErr := parseColor(statusRaw)
	reference, referenceErr := parseColor(referenceRaw)
	if statusErr != nil || referenceErr != nil || status.a == 0 || reference.a == 0 {
		return claimEvaluation{Unverifiable: "chrome-color-agreement requires opaque, parseable background colors"}
	}
	subjects := []MeasuredSubject{{ElementID: statusID, TestID: statusNode.DOM.TestID, Bounds: statusNode.Bounds, Value: statusRaw}, {ElementID: referenceID, TestID: referenceNode.DOM.TestID, Bounds: referenceNode.Bounds, Value: referenceRaw}}
	measurement := &ClaimMeasurement{Metric: "chrome-color-agreement", Subjects: subjects}
	if status != reference {
		return claimEvaluation{Pass: false, AXNodeJSON: encodeAXNode(statusNode), Measurement: measurement, Failure: fmt.Sprintf("chrome colors differ: %s=%s, %s=%s", statusID, statusRaw, referenceID, referenceRaw)}
	}
	return claimEvaluation{Pass: true, AXNodeJSON: encodeAXNode(statusNode), Measurement: measurement}
}

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
		return rgbColor{float64(channels[0]) / 255, float64(channels[1]) / 255, float64(channels[2]) / 255, 1}, nil
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
		alpha := 1.0
		if len(matches) >= 4 {
			parsed, err := strconv.ParseFloat(strings.TrimSuffix(matches[3], "%"), 64)
			if err != nil {
				return rgbColor{}, fmt.Errorf("unsupported color %q", raw)
			}
			if strings.HasSuffix(matches[3], "%") {
				parsed /= 100
			}
			alpha = parsed
		}
		return rgbColor{channels[0], channels[1], channels[2], alpha}, nil
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

func evaluateTapTargetSizeClaim(_ spec.PageDocument, claim spec.Claim, target CaptureTarget, nodes []*AXNode) claimEvaluation {
	node := firstUndersizedTapTarget(nodes, target)
	if node == nil {
		return claimEvaluation{Pass: true}
	}
	minimum := 44.0
	observed := math.Min(node.Bounds.Width, node.Bounds.Height)
	elementID := "tap-target"
	if len(claim.Elements) > 0 {
		elementID = claim.Elements[0]
	}
	m := measurement("tap-target-size", "px", "gte", &observed, &minimum, measuredNodeSubject(elementID, node))
	// Name the offending control. A bare "observed 34px below required 44px"
	// tells an author a control is too small but not which one, and every other
	// floor evaluator here already reports its node.
	return claimEvaluation{
		Pass:        false,
		AXNodeJSON:  encodeAXNode(node),
		Measurement: m,
		Failure:     describeBoundsFailure(measurementFailure(m), node, target),
	}
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
		// An empty subject list is an authoring mistake, not a measurement. Say
		// so, rather than reporting the generic "measured subjects did not
		// satisfy the declared requirement" that an unsatisfied claim produces.
		return claimEvaluation{Unverifiable: fmt.Sprintf("%s requires at least one declared element; add claim.elements referencing an id in elements[] with a bindings.elements entry", claim.Type)}
	}
	pass := true
	var missing []string
	var axNodeJSON string
	for _, elementID := range claim.Elements {
		node := findBoundNode(nodes, page.Bindings.Elements[elementID], elementRole(page, elementID))
		if node == nil {
			pass = false
			missing = append(missing, elementID)
			continue
		}
		axNodeJSON = encodeAXNode(node)
		if claim.Type == "keyboard-reachable" && !node.KeyboardReachable() {
			pass = false
		}
	}
	evaluation := claimEvaluation{Pass: pass, AXNodeJSON: axNodeJSON}
	if len(missing) > 0 {
		// Naming the unresolved ids separates "the surface is genuinely absent"
		// from "the binding never matched a node", which are different fixes.
		evaluation.Failure = fmt.Sprintf("no accessibility node matched declared element(s) %s; check bindings.elements testid and the elements[] role", strings.Join(missing, ", "))
	}
	return evaluation
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

// evaluateFontSizeClaim proves a minimum readable size from captured computed
// style evidence. The default is intentionally conservative for scaffolded
// input assets; authors can make the requirement explicit with minPx.
func evaluateFontSizeClaim(page spec.PageDocument, claim spec.Claim, _ CaptureTarget, nodes []*AXNode) claimEvaluation {
	if len(claim.Elements) == 0 {
		return claimEvaluation{Unverifiable: "font-size requires at least one declared element"}
	}
	minimum := 16.0
	if value, ok := numericParam(claim.Params, "minPx", "minimumPx", "minimum"); ok {
		minimum = value
	}
	for _, elementID := range claim.Elements {
		node := findBoundNode(nodes, page.Bindings.Elements[elementID], elementRole(page, elementID))
		if node == nil {
			return claimEvaluation{Pass: false, Failure: "declared element " + elementID + " was not found in the accessibility snapshot"}
		}
		observed, ok := cssPixels(computedStyleValue(node, "font-size"))
		if !ok {
			return claimEvaluation{Unverifiable: "font-size requires computed font-size evidence"}
		}
		measurement := measurement("font-size", "px", "gte", &observed, &minimum, measuredNodeSubject(elementID, node))
		if observed < minimum {
			return claimEvaluation{Pass: false, AXNodeJSON: encodeAXNode(node), Measurement: measurement, Failure: measurementFailure(measurement)}
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

func evaluateContentNotClippedClaim(page spec.PageDocument, claim spec.Claim, _ CaptureTarget, nodes []*AXNode) claimEvaluation {
	if len(claim.Elements) != 1 {
		return claimEvaluation{Unverifiable: "content-not-clipped requires exactly one declared element"}
	}
	node := findBoundNode(nodes, page.Bindings.Elements[claim.Elements[0]], elementRole(page, claim.Elements[0]))
	if node == nil {
		return claimEvaluation{Unverifiable: fmt.Sprintf("declared element %q was not found in the accessibility snapshot", claim.Elements[0])}
	}
	clientWidth, widthOK := layoutMetric(node, "clientWidth", "client-width")
	scrollWidth, scrollWidthOK := layoutMetric(node, "scrollWidth", "scroll-width")
	clientHeight, heightOK := layoutMetric(node, "clientHeight", "client-height")
	scrollHeight, scrollHeightOK := layoutMetric(node, "scrollHeight", "scroll-height")
	if !widthOK || !scrollWidthOK || !heightOK || !scrollHeightOK {
		return claimEvaluation{Unverifiable: "content-not-clipped requires client and scroll dimensions in layout evidence"}
	}
	overflowX := firstNonEmpty(computedStyleValue(node, "overflow-x"), computedStyleValue(node, "overflow"))
	overflowY := firstNonEmpty(computedStyleValue(node, "overflow-y"), computedStyleValue(node, "overflow"))
	if strings.TrimSpace(overflowX) == "" || strings.TrimSpace(overflowY) == "" {
		return claimEvaluation{Unverifiable: "content-not-clipped requires computed overflow evidence"}
	}
	widthOverflow := scrollWidth > clientWidth+0.01 && !hasReachableOverflow(overflowX)
	heightOverflow := scrollHeight > clientHeight+0.01 && !hasReachableOverflow(overflowY)
	if widthOverflow || heightOverflow {
		axis := "horizontal"
		if heightOverflow && !widthOverflow {
			axis = "vertical"
		} else if widthOverflow && heightOverflow {
			axis = "horizontal and vertical"
		}
		return claimEvaluation{
			Pass:       false,
			AXNodeJSON: encodeAXNode(node),
			Failure:    fmt.Sprintf("element %q (%s) has %s content clipping: client=%gx%g scroll=%gx%g overflow=%q/%q", claim.Elements[0], firstNonEmpty(node.DOM.TestID, node.Name, node.Role), axis, clientWidth, clientHeight, scrollWidth, scrollHeight, overflowX, overflowY),
		}
	}
	return claimEvaluation{Pass: true, AXNodeJSON: encodeAXNode(node)}
}

func layoutMetric(node *AXNode, keys ...string) (float64, bool) {
	for _, key := range keys {
		value := strings.TrimSpace(computedStyleValue(node, key))
		if value == "" {
			continue
		}
		value = strings.TrimSuffix(value, "px")
		metric, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err == nil && metric >= 0 {
			return metric, true
		}
	}
	return 0, false
}

func hasReachableOverflow(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "visible", "auto", "scroll":
		return true
	default:
		return false
	}
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
	for _, floor := range floorClaimsForArchetype(page.Archetype, false) {
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
	for _, floor := range floorClaimsForArchetype(component.Archetype, true) {
		if optedOut[floor.Type] || componentHasClaimType(component, floor.Type) {
			continue
		}
		component.Claims = append(component.Claims, floor)
	}
	if len(component.Elements) > 0 && !componentHasClaimType(component, "content-not-clipped") && !floorOptedOut(component, "content-not-clipped") {
		component.Claims = append(component.Claims, spec.Claim{ID: "floor-content-not-clipped", Type: "content-not-clipped", Statement: "The primary component surface does not clip its content.", Tier: "machine", Elements: []string{component.Elements[0].ID}})
	}
	if !componentHasClaimType(component, "size-parity") && !floorOptedOut(component, "size-parity") {
		component.Claims = append(component.Claims, spec.Claim{ID: "floor-control-geometry", Type: "size-parity", Statement: "Every rendered RCL control matches its documented size rung.", Tier: "machine", Elements: []string{"control"}, Params: map[string]any{"tolerance": 1}})
	}
	return component
}

func floorOptedOut(component spec.ComponentDocument, floor string) bool {
	for _, optOut := range component.FloorOptOuts {
		if optOut.Floor == floor {
			return true
		}
	}
	return false
}

type floorSetDocument struct {
	Archetypes map[string][]string `json:"archetypes"`
}

func floorClaimsForArchetype(archetype string, component bool) []spec.Claim {
	var document floorSetDocument
	if err := json.Unmarshal(archetypeFloors, &document); err != nil {
		if component {
			return componentBaselineFloorClaims()
		}
		return baselineFloorClaims()
	}
	key := strings.ToLower(strings.TrimSpace(archetype))
	types, ok := document.Archetypes[key]
	if !ok {
		if component {
			return componentBaselineFloorClaims()
		}
		return baselineFloorClaims()
	}
	claims := make([]spec.Claim, 0, len(types))
	for _, floorType := range types {
		candidates := baselineFloorClaims()
		if component {
			candidates = componentBaselineFloorClaims()
		}
		for _, candidate := range candidates {
			if candidate.Type == floorType {
				claims = append(claims, candidate)
				break
			}
		}
	}
	return claims
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
