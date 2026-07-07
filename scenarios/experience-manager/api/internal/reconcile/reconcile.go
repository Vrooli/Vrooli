// Package reconcile checks parsed experience specs against captured
// accessibility-tree evidence.
package reconcile

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"

	"experience-manager/internal/spec"
)

const (
	basScenarioID    = "browser-automation-studio"
	snapshotContract = "bas-accessibility-snapshot/v1"
	defaultSettleMs  = 3000
)

// CaptureProfile names one viewport in the reconciliation matrix.
type CaptureProfile struct {
	ID      string
	Width   int
	Height  int
	Aliases []string
}

// DefaultCaptureProfiles is the minimum viewport matrix for machine evidence.
var DefaultCaptureProfiles = []CaptureProfile{
	{ID: "desktop", Width: 1280, Height: 720, Aliases: []string{"wide"}},
	{ID: "mobile", Width: 390, Height: 844},
}

// ErrCaptureUnavailable means the capture mechanism could not provide an AX
// snapshot. Reconciliation maps it to skipped evidence, not a failed claim.
var ErrCaptureUnavailable = errors.New("accessibility capture unavailable")

// Check runs structure reconciliation as a checks.Check.
type Check struct {
	Capturer        Capturer
	Repository      EvidenceRepository
	Now             func() time.Time
	CaptureProfiles []CaptureProfile
}

// Capturer returns one single-location accessibility snapshot for a page route.
type Capturer interface {
	CaptureAccessibility(ctx context.Context, target CaptureTarget) (Snapshot, error)
}

// CaptureTarget identifies the scenario UI surface to inspect.
type CaptureTarget struct {
	Scenario        string
	Route           string
	PageID          string
	StateID         string
	ViewportID      string
	ViewportAliases []string
	ViewportWidth   int
	ViewportHeight  int
	SettleMs        int
}

// Name implements checks.Check.
func (c Check) Name() string { return "reconcile.structure" }

// Run implements checks.Check.
func (c Check) Run(ctx context.Context, report spec.Report) []spec.Finding {
	if report.Spec == nil {
		return nil
	}
	var findings []spec.Finding
	statusByPage := pageStatuses(report.Spec.Index.Pages)
	for pageID, page := range report.Spec.Pages {
		status := statusByPage[pageID]
		loc := "experience/pages/" + pageID + ".json"
		if status == "draft" {
			findings = append(findings, expectedDraftFindings(loc, page)...)
			continue
		}
		page = pageWithBaselineClaims(page)
		if status != "active" || !hasDefaultMachineClaim(page) {
			continue
		}
		capturer := c.Capturer
		if capturer == nil {
			capturer = BASCapturer{}
		}
		profiles := c.captureProfiles()
		for _, profile := range profiles {
			target := CaptureTarget{
				Scenario:        report.Scenario,
				Route:           firstRoute(page.Page.Routes),
				PageID:          page.Page.ID,
				StateID:         "default",
				ViewportID:      profile.ID,
				ViewportAliases: append([]string(nil), profile.Aliases...),
				ViewportWidth:   profile.Width,
				ViewportHeight:  profile.Height,
				SettleMs:        defaultSettleMs,
			}
			if !pageHasMachineClaimForTarget(page, target) {
				continue
			}
			snapshot, err := capturer.CaptureAccessibility(ctx, target)
			if err != nil || snapshot.Contract != snapshotContract || len(snapshot.Flatten()) == 0 {
				findings = append(findings, spec.Finding{
					Code:       spec.CodeCaptureUnavailable,
					Severity:   spec.SeverityInfo,
					Message:    fmt.Sprintf("accessibility capture unavailable for active page %q at viewport %q", page.Page.ID, target.ViewportID),
					Locations:  []string{loc},
					Suggestion: "Start browser-automation-studio and the target UI, then rerun the experience phase.",
				})
				findings = append(findings, c.persistEvidence(ctx, report.Scenario, loc, page, target, Snapshot{}, skippedEvidence(page, target, "capture unavailable"))...)
				continue
			}
			result := reconcileActivePage(loc, page, target, snapshot)
			findings = append(findings, result.Findings...)
			findings = append(findings, c.persistEvidence(ctx, report.Scenario, loc, page, target, snapshot, result.Evidence)...)
		}
		findings = append(findings, unverifiableOutOfMatrixFindings(loc, page, profiles)...)
	}
	sortFindings(findings)
	return findings
}

func (c Check) captureProfiles() []CaptureProfile {
	if len(c.CaptureProfiles) > 0 {
		return c.CaptureProfiles
	}
	return DefaultCaptureProfiles
}

func expectedDraftFindings(loc string, page spec.PageDocument) []spec.Finding {
	expected := expectedFailures(page)
	var findings []spec.Finding
	for _, claim := range page.Claims {
		if !expected[claim.ID] {
			continue
		}
		code := claimFailureCode(claim.Type)
		severity := spec.SeverityWarning
		if claim.Tier == "manual" {
			code = spec.CodeClaimUnproven
			severity = spec.SeverityInfo
		}
		findings = append(findings, spec.Finding{
			Code:       code,
			Severity:   severity,
			Message:    fmt.Sprintf("draft calibration expects claim %q to remain unproven", claim.ID),
			Locations:  []string{loc},
			Suggestion: "Use this draft finding as reconciliation calibration until the page becomes active.",
		})
	}
	return findings
}

func expectedFailures(page spec.PageDocument) map[string]bool {
	raw := page.Extensions["x-spike"]
	if len(raw) == 0 {
		return nil
	}
	var spike struct {
		Expected struct {
			Machine []string `json:"machine"`
			Manual  []string `json:"manual"`
		} `json:"expected-reconciliation-failures"`
	}
	if err := json.Unmarshal(raw, &spike); err != nil {
		return nil
	}
	out := map[string]bool{}
	for _, id := range spike.Expected.Machine {
		out[id] = true
	}
	for _, id := range spike.Expected.Manual {
		out[id] = true
	}
	return out
}

type pageReconciliation struct {
	Findings []spec.Finding
	Evidence []Evidence
}

func reconcileActivePage(loc string, page spec.PageDocument, target CaptureTarget, snapshot Snapshot) pageReconciliation {
	nodes := snapshot.Flatten()
	var findings []spec.Finding
	var evidence []Evidence
	resolvedBindings := 0
	activeElements := activeMachineElementIDs(page, target)
	for elementID, binding := range page.Bindings.Elements {
		if !activeElements[elementID] {
			continue
		}
		if findBoundNode(nodes, binding, elementRole(page, elementID)) == nil {
			findings = append(findings, spec.Finding{
				Code:       spec.CodeBindingUnresolved,
				Severity:   spec.SeverityError,
				Message:    fmt.Sprintf("binding for element %q did not resolve in captured accessibility tree", elementID),
				Locations:  []string{loc},
				Suggestion: "Align the data-testid/selector binding with the rendered UI.",
			})
			continue
		}
		resolvedBindings++
	}
	if len(activeElements) > 0 && resolvedBindings == 0 {
		return pageReconciliation{
			Findings: []spec.Finding{{
				Code:       spec.CodeCaptureUnavailable,
				Severity:   spec.SeverityInfo,
				Message:    fmt.Sprintf("accessibility capture for page %q did not join any declared bindings", page.Page.ID),
				Locations:  []string{loc},
				Suggestion: "Capture the intended page/state before treating structure claims as failed.",
			}},
			Evidence: skippedEvidence(page, target, "no declared bindings joined the accessibility tree"),
		}
	}
	for _, claim := range page.Claims {
		if claim.Tier != "machine" {
			continue
		}
		applies, reason := claimAppliesToTarget(claim, target)
		if !applies {
			if claim.Type == "state-covered" || claim.Type == "state-distinct" {
				claimEvidence := unreachableEvidence(page, claim, target, reason)
				evidence = append(evidence, claimEvidence)
				findings = append(findings, spec.Finding{
					Code:       spec.CodeClaimUnverifiable,
					Severity:   spec.SeverityWarning,
					Message:    fmt.Sprintf("machine claim %q is unverifiable: %s", claim.ID, claimEvidence.Message),
					Locations:  []string{loc},
					Suggestion: "Use a captured default-state/viewport claim, provide manual attestation, or extend the capture matrix.",
				})
			}
			continue
		}
		claimEvidence, ok := evaluateClaim(page, claim, target, nodes)
		evidence = append(evidence, claimEvidence)
		if ok {
			continue
		}
		code := claimFailureCode(claim.Type)
		severity := spec.SeverityError
		message := fmt.Sprintf("machine claim %q was not proven by the accessibility snapshot", claim.ID)
		suggestion := "Update the UI, binding, or claim tier so structure evidence matches intent."
		if claimEvidence.Verdict == "unverifiable" {
			code = spec.CodeClaimUnverifiable
			severity = spec.SeverityWarning
			message = fmt.Sprintf("machine claim %q is unverifiable: %s", claim.ID, claimEvidence.Message)
			suggestion = "Use a supported claim type, retier the claim, or add deterministic checker coverage."
		} else if claimEvidence.Message != "" && claimEvidence.Message != "claim was not proven by accessibility snapshot" {
			message = fmt.Sprintf("machine claim %q failed: %s", claim.ID, claimEvidence.Message)
		}
		findings = append(findings, spec.Finding{
			Code:       code,
			Severity:   severity,
			Message:    message,
			Locations:  []string{loc},
			Suggestion: suggestion,
		})
	}
	return pageReconciliation{Findings: findings, Evidence: evidence}
}

func unreachableEvidence(page spec.PageDocument, claim spec.Claim, target CaptureTarget, reason string) Evidence {
	if reason == "" {
		reason = "claim target is outside the active capture matrix"
	}
	return Evidence{
		PageID:         page.Page.ID,
		Route:          firstRoute(page.Page.Routes),
		StateID:        target.StateID,
		ViewportID:     target.ViewportID,
		ViewportWidth:  target.ViewportWidth,
		ViewportHeight: target.ViewportHeight,
		ClaimID:        claim.ID,
		ClaimType:      claim.Type,
		Verdict:        "unverifiable",
		Message:        reason,
	}
}

func skippedEvidence(page spec.PageDocument, target CaptureTarget, message string) []Evidence {
	var out []Evidence
	for _, claim := range page.Claims {
		if applies, _ := claimAppliesToTarget(claim, target); claim.Tier != "machine" || !applies {
			continue
		}
		out = append(out, Evidence{
			PageID:         page.Page.ID,
			Route:          firstRoute(page.Page.Routes),
			StateID:        target.StateID,
			ViewportID:     target.ViewportID,
			ViewportWidth:  target.ViewportWidth,
			ViewportHeight: target.ViewportHeight,
			ClaimID:        claim.ID,
			ClaimType:      claim.Type,
			Verdict:        "skipped",
			Message:        message,
		})
	}
	return out
}

func evaluateClaim(page spec.PageDocument, claim spec.Claim, target CaptureTarget, nodes []*AXNode) (Evidence, bool) {
	evidence := Evidence{
		PageID:         page.Page.ID,
		Route:          firstRoute(page.Page.Routes),
		StateID:        target.StateID,
		ViewportID:     target.ViewportID,
		ViewportWidth:  target.ViewportWidth,
		ViewportHeight: target.ViewportHeight,
		ClaimID:        claim.ID,
		ClaimType:      claim.Type,
		Verdict:        "passed",
		Message:        "claim proven by accessibility snapshot",
	}
	evaluator := claimEvaluator(claim.Type)
	if evaluator == nil {
		evidence.Verdict = "unverifiable"
		evidence.Message = "claim type has no deterministic structure checker"
		return evidence, false
	}
	result := evaluator(page, claim, target, nodes)
	if result.AXNodeJSON != "" {
		evidence.AXNodeJSON = result.AXNodeJSON
	}
	if result.Unverifiable != "" {
		evidence.Verdict = "unverifiable"
		evidence.Message = result.Unverifiable
		return evidence, false
	}
	if !result.Pass {
		evidence.Verdict = "failed"
		evidence.Message = "claim was not proven by accessibility snapshot"
		if result.Failure != "" {
			evidence.Message = result.Failure
		}
	}
	return evidence, result.Pass
}

type claimEvaluation struct {
	Pass         bool
	AXNodeJSON   string
	Failure      string
	Unverifiable string
}

type claimEvaluatorFunc func(spec.PageDocument, spec.Claim, CaptureTarget, []*AXNode) claimEvaluation

func claimEvaluator(claimType string) claimEvaluatorFunc {
	evaluators := map[string]claimEvaluatorFunc{
		"no-document-horizontal-overflow": evaluateNoDocumentHorizontalOverflowClaim,
		"viewport-fill":                   evaluateViewportFillClaim,
		"chrome-pinned":                   evaluateChromePinnedClaim,
		"safe-area-tap-targets":           evaluateSafeAreaTapTargetsClaim,
		"single-line-chrome":              evaluateSingleLineChromeClaim,
		"tap-target-size":                 evaluateTapTargetSizeClaim,
		"state-covered":                   evaluateStateCoveredClaim,
		"state-distinct":                  evaluateStateDistinctClaim,
		"element-present":                 evaluateElementPresenceClaim,
		"single-dominant-action":          evaluateElementPresenceClaim,
		"keyboard-reachable":              evaluateElementPresenceClaim,
		"visible-without-scroll":          evaluateVisibleWithoutScrollClaim,
		"reading-order":                   evaluateReadingOrderClaim,
	}
	return evaluators[claimType]
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

func evaluateStateDistinctClaim(_ spec.PageDocument, claim spec.Claim, _ CaptureTarget, _ []*AXNode) claimEvaluation {
	if len(claim.States) < 2 {
		return claimEvaluation{}
	}
	for _, state := range claim.States {
		if state != "" && state != "default" {
			return claimEvaluation{Unverifiable: "state setup capture is not implemented for non-default state " + strconv.Quote(state)}
		}
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

func hasClaimType(page spec.PageDocument, claimType string) bool {
	for _, claim := range page.Claims {
		if claim.Type == claimType {
			return true
		}
	}
	return false
}

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
		if node == nil || node.Bounds == nil || isTextOnlyNode(node) || !verticallyIntersectsViewport(node.Bounds, target) {
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
		if strings.Contains(node.Name, "\n") || node.Bounds.Height > 56 {
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
	default:
		return strings.EqualFold(node.DOM.Tag, "nav") || strings.EqualFold(node.DOM.Tag, "header") || strings.EqualFold(node.DOM.Tag, "footer")
	}
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
	return testID == "layout-top-bar" || testID == "layout-sidebar" || testID == "layout-bottom-nav"
}

func isAppChromeControlTestID(testID string) bool {
	return strings.HasPrefix(testID, "layout-sidebar-link-") || strings.HasPrefix(testID, "layout-bottom-nav-link-")
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

func (c Check) persistEvidence(ctx context.Context, scenario, loc string, page spec.PageDocument, target CaptureTarget, snapshot Snapshot, evidence []Evidence) []spec.Finding {
	if c.Repository == nil || len(evidence) == 0 {
		return nil
	}
	now := time.Now().UTC()
	if c.Now != nil {
		now = c.Now().UTC()
	}
	checkedAt := now.Format(evidenceTimeFormat)
	captureRef := snapshot.URL
	if strings.TrimSpace(captureRef) == "" {
		captureRef = fmt.Sprintf("scenario=%s,path=%s", target.Scenario, target.Route)
	}
	for _, item := range evidence {
		item.Scenario = scenario
		if item.PageID == "" {
			item.PageID = page.Page.ID
		}
		if item.Route == "" {
			item.Route = target.Route
		}
		if item.StateID == "" {
			item.StateID = "default"
		}
		if item.ViewportID == "" {
			item.ViewportID = target.ViewportID
		}
		if item.ViewportWidth == 0 {
			item.ViewportWidth = target.ViewportWidth
		}
		if item.ViewportHeight == 0 {
			item.ViewportHeight = target.ViewportHeight
		}
		item.CaptureRef = captureRef
		item.CheckedAt = checkedAt
		if strings.TrimSpace(item.AXNodeJSON) == "" {
			item.AXNodeJSON = "{}"
		}
		item.ID = evidenceID(item)
		if err := c.Repository.SaveEvidence(ctx, item); err != nil {
			return []spec.Finding{{
				Code:       spec.CodeClaimUnproven,
				Severity:   spec.SeverityWarning,
				Message:    fmt.Sprintf("reconciliation evidence could not be persisted for claim %q: %v", item.ClaimID, err),
				Locations:  []string{loc},
				Suggestion: "Check the experience-manager SQLite schema and retry the validation run.",
			}}
		}
	}
	return nil
}

func defaultStateID(claim spec.Claim) string {
	return "default"
}

func encodeAXNode(node *AXNode) string {
	if node == nil {
		return "{}"
	}
	data, err := json.Marshal(node)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func evidenceID(e Evidence) string {
	key := strings.Join([]string{e.Scenario, e.PageID, e.StateID, e.ViewportID, e.ClaimID, e.CaptureRef, e.CheckedAt}, "\x00")
	sum := sha256.Sum256([]byte(key))
	return fmt.Sprintf("ev-%x", sum[:12])
}

// BASCapturer calls Browser Automation Studio's CaptureService.
type BASCapturer struct {
	Resolve       func(ctx context.Context) (string, error)
	ResolveTarget func(ctx context.Context, target CaptureTarget) (string, error)
	HTTPClient    *http.Client
}

// CaptureAccessibility implements Capturer.
func (c BASCapturer) CaptureAccessibility(ctx context.Context, target CaptureTarget) (Snapshot, error) {
	baseURL, err := c.resolve(ctx)
	if err != nil || strings.TrimSpace(baseURL) == "" {
		return Snapshot{}, ErrCaptureUnavailable
	}
	targetURL, err := c.resolveTarget(ctx, target)
	if err != nil || strings.TrimSpace(targetURL) == "" {
		return Snapshot{}, ErrCaptureUnavailable
	}

	type waitForPayload struct {
		TimeoutMs int `json:"timeout_ms"`
	}
	type dimensionsPayload struct {
		Width  int `json:"width,omitempty"`
		Height int `json:"height,omitempty"`
	}
	type captureRequestPayload struct {
		URL                 string            `json:"url"`
		InlineAccessibility bool              `json:"inline_accessibility"`
		Label               string            `json:"label"`
		Dimensions          dimensionsPayload `json:"dimensions,omitempty"`
		WaitFor             *waitForPayload   `json:"wait_for,omitempty"`
		InteractionFlowJSON string            `json:"interaction_flow_json,omitempty"`
	}
	payload := captureRequestPayload{
		URL:                 targetURL,
		InlineAccessibility: true,
		Label:               "experience-manager structure reconciliation",
	}
	if target.ViewportWidth > 0 && target.ViewportHeight > 0 {
		payload.Dimensions = dimensionsPayload{Width: target.ViewportWidth, Height: target.ViewportHeight}
	}
	if target.SettleMs > 0 {
		payload.WaitFor = &waitForPayload{TimeoutMs: target.SettleMs}
		payload.InteractionFlowJSON = settleInteractionFlow(target.SettleMs)
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return Snapshot{}, ErrCaptureUnavailable
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(baseURL, "/")+"/browser_automation_studio.v1.capture.CaptureService/Capture",
		strings.NewReader(string(encoded)),
	)
	if err != nil {
		return Snapshot{}, ErrCaptureUnavailable
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return Snapshot{}, ErrCaptureUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Snapshot{}, ErrCaptureUnavailable
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Snapshot{}, ErrCaptureUnavailable
	}
	var decoded struct {
		AccessibilityJSON      string `json:"accessibility_json"`
		AccessibilityJSONCamel string `json:"accessibilityJson"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return Snapshot{}, ErrCaptureUnavailable
	}
	if strings.TrimSpace(decoded.AccessibilityJSON) == "" {
		decoded.AccessibilityJSON = decoded.AccessibilityJSONCamel
	}
	if strings.TrimSpace(decoded.AccessibilityJSON) == "" {
		return Snapshot{}, ErrCaptureUnavailable
	}
	var snapshot Snapshot
	if err := json.Unmarshal([]byte(decoded.AccessibilityJSON), &snapshot); err != nil {
		return Snapshot{}, fmt.Errorf("%w: decode accessibility snapshot: %v", ErrCaptureUnavailable, err)
	}
	return snapshot, nil
}

func settleInteractionFlow(durationMs int) string {
	if durationMs <= 0 {
		return ""
	}
	type waitParams struct {
		DurationMs int `json:"duration_ms"`
	}
	type action struct {
		Type     string     `json:"type"`
		Wait     waitParams `json:"wait"`
		Metadata struct {
			Label string `json:"label"`
		} `json:"metadata"`
	}
	type node struct {
		ID     string `json:"id"`
		Action action `json:"action"`
	}
	payload := struct {
		Nodes []node `json:"nodes"`
		Edges []any  `json:"edges"`
	}{
		Nodes: []node{{
			ID: "experience-manager-settle",
			Action: action{
				Type: "ACTION_TYPE_WAIT",
				Wait: waitParams{DurationMs: durationMs},
				Metadata: struct {
					Label string `json:"label"`
				}{Label: "Settle before accessibility capture"},
			},
		}},
		Edges: []any{},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(data)
}

func (c BASCapturer) resolve(ctx context.Context) (string, error) {
	if c.Resolve != nil {
		return c.Resolve(ctx)
	}
	return discovery.ResolveScenarioURLDefault(ctx, basScenarioID)
}

func (c BASCapturer) resolveTarget(ctx context.Context, target CaptureTarget) (string, error) {
	if c.ResolveTarget != nil {
		return c.ResolveTarget(ctx, target)
	}
	resolver := discovery.NewResolver(discovery.ResolverConfig{})
	port, err := resolver.ResolveScenarioPort(ctx, target.Scenario, "UI_PORT")
	if err != nil || port <= 0 {
		return "", ErrCaptureUnavailable
	}
	return fmt.Sprintf("http://localhost:%d%s", port, firstRoute([]string{target.Route})), nil
}

func (c BASCapturer) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 120 * time.Second}
}

func hasDefaultMachineClaim(page spec.PageDocument) bool {
	for _, claim := range page.Claims {
		if claim.Tier == "machine" && claimTargetsDefaultState(claim) {
			return true
		}
	}
	return false
}

func pageHasMachineClaimForTarget(page spec.PageDocument, target CaptureTarget) bool {
	for _, claim := range page.Claims {
		if applies, _ := claimAppliesToTarget(claim, target); claim.Tier == "machine" && applies {
			return true
		}
	}
	return false
}

func activeMachineElementIDs(page spec.PageDocument, target CaptureTarget) map[string]bool {
	out := map[string]bool{}
	for _, claim := range page.Claims {
		if applies, _ := claimAppliesToTarget(claim, target); claim.Tier != "machine" || !applies {
			continue
		}
		for _, elementID := range claim.Elements {
			out[elementID] = true
		}
	}
	return out
}

func claimTargetsDefaultState(claim spec.Claim) bool {
	if len(claim.Locales) > 0 || len(claim.Extensions) > 0 {
		return false
	}
	if len(claim.States) == 0 {
		return true
	}
	for _, state := range claim.States {
		if state == "" || state == "default" {
			return true
		}
	}
	return false
}

func claimAppliesToTarget(claim spec.Claim, target CaptureTarget) (bool, string) {
	if len(claim.Locales) > 0 {
		return false, "locale-scoped claims require locale capture support"
	}
	if len(claim.Extensions) > 0 {
		return false, "extension-scoped claims require a deterministic extension checker"
	}
	if !claimTargetsState(claim, target.StateID) {
		return false, "state setup capture is not implemented for non-default states"
	}
	if !claimTargetsViewport(claim, target.ViewportID, target.ViewportAliases) {
		return false, fmt.Sprintf("claim viewports %v are outside captured viewport %q", claim.Viewports, target.ViewportID)
	}
	return true, ""
}

func claimTargetsState(claim spec.Claim, stateID string) bool {
	if stateID == "" {
		stateID = "default"
	}
	if stateID != "default" {
		for _, state := range claim.States {
			if state == stateID {
				return true
			}
		}
		return false
	}
	if len(claim.States) == 0 {
		return true
	}
	for _, state := range claim.States {
		if state == "" || state == "default" {
			return true
		}
	}
	return false
}

func claimTargetsViewport(claim spec.Claim, viewportID string, aliases []string) bool {
	if len(claim.Viewports) == 0 {
		return true
	}
	for _, viewport := range claim.Viewports {
		if viewport == viewportID {
			return true
		}
		for _, alias := range aliases {
			if viewport == alias {
				return true
			}
		}
	}
	return false
}

func unverifiableOutOfMatrixFindings(loc string, page spec.PageDocument, profiles []CaptureProfile) []spec.Finding {
	var findings []spec.Finding
	captured := map[string]bool{}
	for _, profile := range profiles {
		captured[profile.ID] = true
		for _, alias := range profile.Aliases {
			captured[alias] = true
		}
	}
	for _, claim := range page.Claims {
		if claim.Tier != "machine" || len(claim.Viewports) == 0 || isBaselineFloorType(claim.Type) {
			continue
		}
		var missing []string
		for _, viewport := range claim.Viewports {
			if !captured[viewport] {
				missing = append(missing, viewport)
			}
		}
		if len(missing) == 0 {
			continue
		}
		findings = append(findings, spec.Finding{
			Code:       spec.CodeClaimUnverifiable,
			Severity:   spec.SeverityWarning,
			Message:    fmt.Sprintf("machine claim %q targets uncaptured viewports %v", claim.ID, missing),
			Locations:  []string{loc},
			Suggestion: "Add the viewport to the capture matrix, retier the claim, or remove the viewport scope if it should apply everywhere.",
		})
	}
	return findings
}

func pageStatuses(refs []spec.DocumentRef) map[string]string {
	out := map[string]string{}
	for _, ref := range refs {
		out[ref.ID] = ref.Status
	}
	return out
}

func firstRoute(routes []string) string {
	if len(routes) == 0 || strings.TrimSpace(routes[0]) == "" {
		return "/"
	}
	return routes[0]
}

func elementRole(page spec.PageDocument, elementID string) string {
	for _, el := range page.Elements {
		if el.ID == elementID {
			return el.Role
		}
	}
	return ""
}

func findBoundIndex(nodes []*AXNode, binding spec.Binding, role string) int {
	for i, node := range nodes {
		if nodeMatches(node, binding, role) {
			return i
		}
	}
	return -1
}

func findBoundNode(nodes []*AXNode, binding spec.Binding, role string) *AXNode {
	for _, node := range nodes {
		if nodeMatches(node, binding, role) {
			return node
		}
	}
	return nil
}

func nodeMatches(node *AXNode, binding spec.Binding, role string) bool {
	if node == nil {
		return false
	}
	if binding.TestID != "" && node.DOM.TestID != binding.TestID {
		return false
	}
	if binding.Selector != "" && !selectorMatches(node, binding.Selector) {
		return false
	}
	if role != "" && !strings.HasPrefix(role, "x-") && node.Role != role {
		return false
	}
	return binding.TestID != "" || binding.Selector != ""
}

func selectorMatches(node *AXNode, selector string) bool {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return false
	}
	if strings.Contains(selector, "[role='") || strings.Contains(selector, "[role=\"") {
		return selectorContains(selector, "role", node.Role)
	}
	if strings.Contains(selector, "data-testid^=") {
		prefix := selectorValue(selector)
		return prefix != "" && strings.HasPrefix(node.DOM.TestID, prefix)
	}
	if strings.Contains(selector, "data-testid=") {
		return selectorValue(selector) == node.DOM.TestID
	}
	return false
}

func selectorContains(selector, attr, value string) bool {
	return strings.Contains(selector, attr+"='"+value+"'") || strings.Contains(selector, attr+"=\""+value+"\"")
}

func selectorValue(selector string) string {
	for _, token := range []string{"data-testid^=", "data-testid="} {
		idx := strings.Index(selector, token)
		if idx < 0 {
			continue
		}
		value := strings.TrimLeft(selector[idx+len(token):], `'"`)
		end := strings.IndexAny(value, `'" ]`)
		if end >= 0 {
			value = value[:end]
		}
		return value
	}
	return ""
}

func sortFindings(findings []spec.Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Code != findings[j].Code {
			return findings[i].Code < findings[j].Code
		}
		return first(findings[i].Locations) < first(findings[j].Locations)
	})
}

func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
