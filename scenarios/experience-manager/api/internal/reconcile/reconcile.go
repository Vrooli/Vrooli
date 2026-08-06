// Package reconcile checks parsed experience specs against captured
// accessibility-tree evidence.
package reconcile

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
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
	ID               string
	MatrixID         string
	Width            int
	Height           int
	Aliases          []string
	ColorScheme      string
	Locale           string
	MotionPreference string
	InteractionState string
}

// DefaultCaptureProfiles is retained for callers that construct a Check
// without a scenario path. Normal runs load the bounded baseline matrix from
// capabilities/axes.json via captureProfiles.
var DefaultCaptureProfiles = []CaptureProfile{
	{ID: "desktop", MatrixID: "desktop-light-en-rest", Aliases: []string{"wide"}, Width: 1280, Height: 720, ColorScheme: "light", Locale: "en", MotionPreference: "no-preference", InteractionState: "rest"},
	{ID: "mobile", MatrixID: "mobile-dark-en-reduce", Width: 390, Height: 844, ColorScheme: "dark", Locale: "en", MotionPreference: "reduce", InteractionState: "rest"},
}

// ErrCaptureUnavailable means the capture mechanism could not provide an AX
// snapshot. Reconciliation maps it to skipped evidence, not a failed claim.
var ErrCaptureUnavailable = errors.New("accessibility capture unavailable")

// Check runs structure reconciliation as a checks.Check.
type Check struct {
	Capturer           Capturer
	Repository         EvidenceRepository
	Now                func() time.Time
	CaptureProfiles    []CaptureProfile
	CaptureConcurrency int
}

type captureResult struct {
	target   CaptureTarget
	snapshot Snapshot
	err      error
}

func (c Check) captureAll(ctx context.Context, capturer Capturer, targets []CaptureTarget) []captureResult {
	results := make([]captureResult, len(targets))
	workers := c.CaptureConcurrency
	if workers <= 0 {
		workers = 2
	}
	if workers > len(targets) {
		workers = len(targets)
	}
	if workers == 0 {
		return results
	}
	jobs := make(chan int)
	done := make(chan struct{})
	for worker := 0; worker < workers; worker++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for index := range jobs {
				snapshot, err := capturer.CaptureAccessibility(ctx, targets[index])
				results[index] = captureResult{target: targets[index], snapshot: snapshot, err: err}
			}
		}()
	}
	for index := range targets {
		jobs <- index
	}
	close(jobs)
	for worker := 0; worker < workers; worker++ {
		<-done
	}
	return results
}

func captureTargetKey(target CaptureTarget) string {
	return strings.Join([]string{target.DocumentKind, target.PageID, target.ComponentID, target.Route, target.ExampleName, target.StateID, target.ViewportID, target.ColorScheme, target.Locale, target.MotionPreference, target.InteractionState}, "\x00")
}

func (c Check) captureMatrix(ctx context.Context, capturer Capturer, targets []CaptureTarget) map[string]captureResult {
	results := make(map[string]captureResult, len(targets))
	for _, result := range c.captureAll(ctx, capturer, targets) {
		results[captureTargetKey(result.target)] = result
		c.persistCaptureTiming(ctx, result.target, result.snapshot)
	}
	return results
}

func (c Check) persistCaptureTiming(ctx context.Context, target CaptureTarget, snapshot Snapshot) {
	repository, ok := c.Repository.(CaptureTimingRepository)
	if !ok || snapshot.Timing.TotalMilliseconds <= 0 {
		return
	}
	now := c.Now
	if now == nil {
		now = time.Now
	}
	_ = repository.SaveCaptureTiming(ctx, CaptureTargetTiming{
		Scenario:                  target.Scenario,
		DocumentKind:              target.DocumentKind,
		PageID:                    target.PageID,
		ComponentID:               target.ComponentID,
		Route:                     target.Route,
		StateID:                   target.StateID,
		ViewportID:                target.ViewportID,
		ViewportWidth:             target.ViewportWidth,
		ViewportHeight:            target.ViewportHeight,
		TotalMilliseconds:         snapshot.Timing.TotalMilliseconds,
		NavigationMilliseconds:    snapshot.Timing.NavigationMilliseconds,
		ReadinessWaitMilliseconds: snapshot.Timing.ReadinessWaitMilliseconds,
		Strategy:                  snapshot.Timing.Strategy,
		Outcome:                   snapshot.Timing.Outcome,
		CapturedAt:                now().UTC().Format(evidenceTimeFormat),
	})
}

// Capturer returns one single-location accessibility snapshot for a page route.
type Capturer interface {
	CaptureAccessibility(ctx context.Context, target CaptureTarget) (Snapshot, error)
}

// CaptureTarget identifies the scenario UI surface to inspect.
type CaptureTarget struct {
	Scenario          string
	Route             string
	DocumentKind      string
	PageID            string
	ComponentID       string
	ComponentTitle    string
	ExampleName       string
	StateID           string
	StateFingerprints map[string]string
	ViewportID        string
	ViewportAliases   []string
	ViewportWidth     int
	ViewportHeight    int
	ColorScheme       string
	Locale            string
	MotionPreference  string
	InteractionState  string
	SettleMs          int
}

// Name implements checks.Check.
func (c Check) Name() string { return "reconcile.structure" }

// Run implements checks.Check.
func (c Check) Run(ctx context.Context, report spec.Report) []spec.Finding {
	if report.Spec == nil {
		return nil
	}
	var findings []spec.Finding
	capturer := c.Capturer
	if capturer == nil {
		capturer = BASCapturer{}
	}
	profiles := c.captureProfiles(report.TargetPath)
	statusByPage := pageStatuses(report.Spec.Index.Pages)
	for pageID, page := range report.Spec.Pages {
		status := statusByPage[pageID]
		loc := "experience/pages/" + pageID + ".json"
		if status == "draft" {
			findings = append(findings, expectedDraftFindings(loc, page)...)
			continue
		}
		page = pageWithBaselineClaims(page)
		if status != "active" || !hasMachineClaim(page) {
			continue
		}
		targetsByProfile := make(map[string][]CaptureTarget, len(profiles))
		var matrixTargets []CaptureTarget
		for _, profile := range profiles {
			targets := captureTargetsForProfile(report.Scenario, page, profile)
			targetsByProfile[profileKey(profile)] = targets
			matrixTargets = append(matrixTargets, targets...)
		}
		captures := c.captureMatrix(ctx, capturer, matrixTargets)
		for _, profile := range profiles {
			targets := targetsByProfile[profileKey(profile)]
			snapshots := map[string]Snapshot{}
			fingerprints := map[string]string{}
			for _, target := range targets {
				captured := captures[captureTargetKey(target)]
				snapshot, err := captured.snapshot, captured.err
				if err != nil || snapshot.Contract != snapshotContract || len(snapshot.Flatten()) == 0 {
					findings = append(findings, spec.Finding{
						Code:       spec.CodeCaptureUnavailable,
						Severity:   spec.SeverityInfo,
						Message:    fmt.Sprintf("accessibility capture unavailable for active page %q state %q at viewport %q", page.Page.ID, target.StateID, target.ViewportID),
						Locations:  []string{loc},
						Suggestion: "Start browser-automation-studio and the target UI, then rerun the experience phase.",
					})
					findings = append(findings, c.persistEvidence(ctx, report.Scenario, loc, page, target, Snapshot{}, skippedEvidence(page, target, "capture unavailable"))...)
					continue
				}
				snapshots[target.StateID] = snapshot
				fingerprints[target.StateID] = snapshotFingerprint(snapshot)
			}
			for _, target := range targets {
				snapshot, ok := snapshots[target.StateID]
				if !ok {
					continue
				}
				target.StateFingerprints = fingerprints
				result := reconcileActivePage(loc, page, target, snapshot)
				findings = append(findings, result.Findings...)
				findings = append(findings, c.persistEvidence(ctx, report.Scenario, loc, page, target, snapshot, result.Evidence)...)
			}
		}
		findings = append(findings, unverifiableOutOfMatrixFindings(loc, page, profiles)...)
		findings = append(findings, unverifiableStateSetupFindings(loc, page)...)
	}
	statusByComponent := pageStatuses(report.Spec.Index.Components)
	for componentID, component := range report.Spec.Components {
		status := statusByComponent[componentID]
		loc := "experience/components/" + componentID + ".json"
		if status != "active" {
			continue
		}
		findings = append(findings, advisoryComponentFindings(componentSourceFindings(report, loc, component))...)
		page := componentAsPage(componentWithBaselineClaims(component))
		if !hasMachineClaim(page) {
			continue
		}
		targetsByProfile := make(map[string][]CaptureTarget, len(profiles))
		var matrixTargets []CaptureTarget
		for _, profile := range profiles {
			targets := captureTargetsForComponentProfile(report.Scenario, component, profile)
			targetsByProfile[profileKey(profile)] = targets
			matrixTargets = append(matrixTargets, targets...)
		}
		captures := c.captureMatrix(ctx, capturer, matrixTargets)
		for _, profile := range profiles {
			targets := targetsByProfile[profileKey(profile)]
			snapshots := map[string]Snapshot{}
			fingerprints := map[string]string{}
			for _, target := range targets {
				captured := captures[captureTargetKey(target)]
				snapshot, err := captured.snapshot, captured.err
				if err != nil || snapshot.Contract != snapshotContract || len(snapshot.Flatten()) == 0 {
					findings = append(findings, spec.Finding{
						Code:       spec.CodeCaptureUnavailable,
						Severity:   spec.SeverityInfo,
						Message:    fmt.Sprintf("accessibility capture unavailable for active component %q example %q at viewport %q", component.Component.ID, target.ExampleName, target.ViewportID),
						Locations:  []string{loc},
						Suggestion: "Start browser-automation-studio and the RCL UI, then rerun the experience phase.",
					})
					findings = append(findings, c.persistEvidence(ctx, report.Scenario, loc, page, target, Snapshot{}, skippedEvidence(page, target, "capture unavailable"))...)
					continue
				}
				snapshots[target.StateID] = snapshot
				fingerprints[target.StateID] = snapshotFingerprint(snapshot)
			}
			for _, target := range targets {
				snapshot, ok := snapshots[target.StateID]
				if !ok {
					continue
				}
				target.StateFingerprints = fingerprints
				result := reconcileActivePage(loc, page, target, snapshot)
				findings = append(findings, advisoryComponentFindings(result.Findings)...)
				findings = append(findings, c.persistEvidence(ctx, report.Scenario, loc, page, target, snapshot, result.Evidence)...)
			}
		}
		findings = append(findings, advisoryComponentFindings(unverifiableOutOfMatrixFindings(loc, page, profiles))...)
	}
	sortFindings(findings)
	return findings
}

func (c Check) captureProfiles(scenarioDir string) []CaptureProfile {
	if len(c.CaptureProfiles) > 0 {
		return c.CaptureProfiles
	}
	if strings.TrimSpace(scenarioDir) != "" {
		if profiles, err := CaptureProfilesFromAxes(filepath.Join(scenarioDir, "capabilities", "axes.json"), defaultCaptureBudget); err == nil && len(profiles) > 0 {
			return profiles
		}
	}
	return DefaultCaptureProfiles
}

func profileKey(profile CaptureProfile) string {
	if strings.TrimSpace(profile.MatrixID) != "" {
		return profile.MatrixID
	}
	return profile.ID + "-" + profile.ColorScheme + "-" + profile.Locale + "-" + profile.MotionPreference + "-" + profile.InteractionState
}

func captureTargetsForProfile(scenario string, page spec.PageDocument, profile CaptureProfile) []CaptureTarget {
	var targets []CaptureTarget
	for _, stateID := range capturableStateIDs(page) {
		state := pageState(page, stateID)
		target := CaptureTarget{
			Scenario:        scenario,
			Route:           routeForState(page, state),
			DocumentKind:    "page",
			PageID:          page.Page.ID,
			StateID:         stateID,
			ViewportID:      profile.ID,
			ViewportAliases: append([]string(nil), profile.Aliases...),
			ViewportWidth:   profile.Width,
			ViewportHeight:  profile.Height,
			ColorScheme:     profile.ColorScheme, Locale: profile.Locale, MotionPreference: profile.MotionPreference, InteractionState: profile.InteractionState,
			SettleMs: settleMsForState(state),
		}
		if !pageHasMachineClaimForTarget(page, target) {
			continue
		}
		targets = append(targets, target)
	}
	return targets
}

func captureTargetsForComponentProfile(scenario string, component spec.ComponentDocument, profile CaptureProfile) []CaptureTarget {
	var targets []CaptureTarget
	page := componentAsPage(componentWithBaselineClaims(component))
	version := componentVersion(componentCatalogRef(component))
	for _, state := range component.States {
		if strings.TrimSpace(state.ID) == "" || strings.TrimSpace(state.Example) == "" || !hasMachineClaimForState(page, state.ID) {
			continue
		}
		target := CaptureTarget{
			Scenario:        "react-component-library",
			Route:           componentHarnessRoute(scenario, component, version, state.Example),
			DocumentKind:    "component",
			PageID:          component.Component.ID,
			ComponentID:     component.Component.ID,
			ComponentTitle:  component.Component.Title,
			ExampleName:     state.Example,
			StateID:         state.ID,
			ViewportID:      profile.ID,
			ViewportAliases: append([]string(nil), profile.Aliases...),
			ViewportWidth:   profile.Width,
			ViewportHeight:  profile.Height,
			ColorScheme:     profile.ColorScheme, Locale: profile.Locale, MotionPreference: profile.MotionPreference, InteractionState: profile.InteractionState,
			SettleMs: defaultSettleMs,
		}
		if !pageHasMachineClaimForTarget(page, target) {
			continue
		}
		targets = append(targets, target)
	}
	return targets
}

func componentAsPage(component spec.ComponentDocument) spec.PageDocument {
	return spec.PageDocument{
		Page: spec.PageIdentity{
			ID:      component.Component.ID,
			Title:   component.Component.Title,
			Routes:  []string{"/"},
			Purpose: component.Component.Purpose,
		},
		States:       componentStatesAsPageStates(component.States),
		Elements:     component.Elements,
		Claims:       component.Claims,
		Bindings:     component.Bindings,
		FloorOptOuts: component.FloorOptOuts,
	}
}

func componentStatesAsPageStates(states []spec.ComponentState) []spec.State {
	out := make([]spec.State, 0, len(states))
	for _, state := range states {
		out = append(out, spec.State{ID: state.ID, Description: state.Description})
	}
	return out
}

func capturableStateIDs(page spec.PageDocument) []string {
	ids := []string{"default"}
	seen := map[string]bool{"default": true}
	for _, state := range page.States {
		id := strings.TrimSpace(state.ID)
		if id == "" || id == "default" || seen[id] || !hasStateSetup(state) {
			continue
		}
		if !hasMachineClaimForState(page, id) {
			continue
		}
		ids = append(ids, id)
		seen[id] = true
	}
	return ids
}

func hasMachineClaimForState(page spec.PageDocument, stateID string) bool {
	for _, claim := range page.Claims {
		if claim.Tier == "machine" && claimTargetsState(claim, stateID) {
			return true
		}
	}
	return false
}

func pageState(page spec.PageDocument, stateID string) spec.State {
	for _, state := range page.States {
		if state.ID == stateID {
			return state
		}
	}
	return spec.State{ID: "default"}
}

func hasStateSetup(state spec.State) bool {
	return strings.TrimSpace(state.Setup.Route) != "" ||
		len(state.Setup.Query) > 0 ||
		strings.TrimSpace(state.Setup.Hash) != "" ||
		state.Setup.SettleMs > 0
}

func routeForState(page spec.PageDocument, state spec.State) string {
	route := firstRoute(page.Page.Routes)
	if strings.TrimSpace(state.Setup.Route) != "" {
		route = state.Setup.Route
	}
	if len(state.Setup.Query) == 0 && strings.TrimSpace(state.Setup.Hash) == "" {
		return route
	}
	parsed, err := url.Parse(route)
	if err != nil {
		return route
	}
	query := parsed.Query()
	keys := make([]string, 0, len(state.Setup.Query))
	for key := range state.Setup.Query {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		query.Set(key, state.Setup.Query[key])
	}
	parsed.RawQuery = query.Encode()
	if hash := strings.TrimSpace(state.Setup.Hash); hash != "" {
		parsed.Fragment = strings.TrimPrefix(hash, "#")
	}
	return parsed.String()
}

func settleMsForState(state spec.State) int {
	if state.Setup.SettleMs > 0 {
		return state.Setup.SettleMs
	}
	return defaultSettleMs
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
			if (claim.Type == "state-covered" || claim.Type == "state-distinct") && !strings.Contains(reason, "outside captured state") {
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
		Route:          targetEvidenceRoute(page, target),
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
			Route:          targetEvidenceRoute(page, target),
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
		Route:          targetEvidenceRoute(page, target),
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

func targetEvidenceRoute(page spec.PageDocument, target CaptureTarget) string {
	if strings.TrimSpace(target.Route) != "" {
		return target.Route
	}
	return firstRoute(page.Page.Routes)
}

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

func advisoryComponentFindings(findings []spec.Finding) []spec.Finding {
	for i := range findings {
		if findings[i].Severity == spec.SeverityError {
			findings[i].Severity = spec.SeverityWarning
		}
		if findings[i].Suggestion == "" {
			findings[i].Suggestion = "Use component reconciliation as advisory evidence until the component contract graduates to gating."
		}
	}
	return findings
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
		if node == nil || node.Bounds == nil || isTextOnlyNode(node) || isPreviewWorkspaceScrollNode(node) || !verticallyIntersectsViewport(node.Bounds, target) {
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

func (c Check) persistEvidence(ctx context.Context, scenario, loc string, page spec.PageDocument, target CaptureTarget, snapshot Snapshot, evidence []Evidence) []spec.Finding {
	if c.Repository == nil || len(evidence) == 0 {
		return nil
	}
	now := time.Now().UTC()
	if c.Now != nil {
		now = c.Now().UTC()
	}
	checkedAt := now.Format(evidenceTimeFormat)
	captureRef := firstNonEmpty(snapshot.ScreenshotRef, snapshot.URL)
	if strings.TrimSpace(captureRef) == "" {
		captureRef = fmt.Sprintf("scenario=%s,path=%s", target.Scenario, target.Route)
	}
	for _, item := range evidence {
		item.Scenario = scenario
		if item.DocumentKind == "" {
			item.DocumentKind = target.DocumentKind
		}
		if item.PageID == "" {
			item.PageID = page.Page.ID
		}
		if item.ComponentID == "" {
			item.ComponentID = target.ComponentID
		}
		if item.ComponentTitle == "" {
			item.ComponentTitle = target.ComponentTitle
		}
		if item.ExampleName == "" {
			item.ExampleName = target.ExampleName
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

func snapshotFingerprint(snapshot Snapshot) string {
	nodes := snapshot.Flatten()
	type fingerprintNode struct {
		Role   string  `json:"role,omitempty"`
		Name   string  `json:"name,omitempty"`
		Value  string  `json:"value,omitempty"`
		TestID string  `json:"testid,omitempty"`
		Tag    string  `json:"tag,omitempty"`
		Bounds *Bounds `json:"bounds,omitempty"`
	}
	normalized := make([]fingerprintNode, 0, len(nodes))
	for _, node := range nodes {
		if node == nil {
			continue
		}
		normalized = append(normalized, fingerprintNode{
			Role:   node.Role,
			Name:   node.Name,
			Value:  node.Value,
			TestID: node.DOM.TestID,
			Tag:    node.DOM.Tag,
			Bounds: node.Bounds,
		})
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:])
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
		if err != nil {
			return Snapshot{}, fmt.Errorf("%w: resolve BAS endpoint: %v", ErrCaptureUnavailable, err)
		}
		return Snapshot{}, fmt.Errorf("%w: BAS endpoint is empty", ErrCaptureUnavailable)
	}
	// Preserve the scenario identity instead of pre-resolving localhost. BAS
	// uses this shorthand to obtain the Experience Manager-owned readiness
	// profile and wait for a terminal ExperienceSurface state.
	targetURL := "scenario=" + target.Scenario + ",path=" + firstRoute([]string{target.Route})

	type waitForPayload struct {
		TimeoutMs int `json:"timeout_ms"`
	}
	type dimensionsPayload struct {
		Width  int `json:"width,omitempty"`
		Height int `json:"height,omitempty"`
	}
	type fingerprintPayload struct {
		Locale      string `json:"locale,omitempty"`
		ColorScheme string `json:"colorScheme,omitempty"`
	}
	type browserProfilePayload struct {
		Fingerprint      *fingerprintPayload `json:"fingerprint,omitempty"`
		MotionPreference string              `json:"motionPreference,omitempty"`
		InteractionState string              `json:"interactionState,omitempty"`
	}
	type captureRequestPayload struct {
		URL string `json:"url"`
		// CaptureService is a Connect endpoint, whose canonical JSON field name
		// is lowerCamelCase. The response has always accepted both shapes, but a
		// snake_case request silently loses this optional capture flag.
		InlineAccessibility bool                   `json:"inlineAccessibility"`
		Label               string                 `json:"label"`
		Dimensions          dimensionsPayload      `json:"dimensions,omitempty"`
		WaitFor             *waitForPayload        `json:"wait_for,omitempty"`
		InteractionFlowJSON string                 `json:"interaction_flow_json,omitempty"`
		InlineComputedStyle bool                   `json:"inlineComputedStyle,omitempty"`
		BrowserProfile      *browserProfilePayload `json:"browserProfile,omitempty"`
		InteractionState    string                 `json:"interactionState,omitempty"`
	}
	payload := captureRequestPayload{
		URL:                 targetURL,
		InlineAccessibility: true,
		InlineComputedStyle: true,
		Label:               "experience-manager structure reconciliation",
	}
	if target.ColorScheme != "" || target.Locale != "" || target.MotionPreference != "" || target.InteractionState != "" {
		payload.BrowserProfile = &browserProfilePayload{
			Fingerprint:      &fingerprintPayload{Locale: target.Locale, ColorScheme: target.ColorScheme},
			MotionPreference: target.MotionPreference,
			InteractionState: target.InteractionState,
		}
		payload.InteractionState = target.InteractionState
	}
	if target.ViewportWidth > 0 && target.ViewportHeight > 0 {
		payload.Dimensions = dimensionsPayload{Width: target.ViewportWidth, Height: target.ViewportHeight}
	}
	// No explicit caller wait: BAS first applies declared semantic readiness
	// and falls back to its compatibility settle delay only when no profile is
	// available. SettleMs remains part of CaptureTarget for legacy callers and
	// state metadata, but it must not mask a declared readiness surface.

	encoded, err := json.Marshal(payload)
	if err != nil {
		return Snapshot{}, fmt.Errorf("%w: encode BAS capture request: %v", ErrCaptureUnavailable, err)
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(baseURL, "/")+"/browser_automation_studio.v1.capture.CaptureService/Capture",
		strings.NewReader(string(encoded)),
	)
	if err != nil {
		return Snapshot{}, fmt.Errorf("%w: create BAS capture request: %v", ErrCaptureUnavailable, err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return Snapshot{}, fmt.Errorf("%w: call BAS CaptureService: %v", ErrCaptureUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Snapshot{}, fmt.Errorf("%w: BAS CaptureService returned HTTP %d", ErrCaptureUnavailable, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Snapshot{}, fmt.Errorf("%w: read BAS CaptureService response: %v", ErrCaptureUnavailable, err)
	}
	var decoded struct {
		AccessibilityJSON      string `json:"accessibility_json"`
		AccessibilityJSONCamel string `json:"accessibilityJson"`
		Artifacts              []struct {
			Type      any               `json:"type"`
			Path      string            `json:"path"`
			SizeBytes int64             `json:"size_bytes"`
			Metadata  map[string]string `json:"metadata"`
		} `json:"artifacts"`
		Readiness struct {
			DurationMS                   json.RawMessage `json:"duration_ms"`
			DurationMSCamel              json.RawMessage `json:"durationMs"`
			NavigationDurationMS         json.RawMessage `json:"navigation_duration_ms"`
			NavigationDurationMSCamel    json.RawMessage `json:"navigationDurationMs"`
			ReadinessWaitDurationMS      json.RawMessage `json:"readiness_wait_duration_ms"`
			ReadinessWaitDurationMSCamel json.RawMessage `json:"readinessWaitDurationMs"`
			SelectedStrategy             string          `json:"selected_strategy"`
			SelectedStrategyCamel        string          `json:"selectedStrategy"`
			Outcome                      string          `json:"outcome"`
		} `json:"readiness"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return Snapshot{}, fmt.Errorf("%w: decode BAS CaptureService response: %v", ErrCaptureUnavailable, err)
	}
	if strings.TrimSpace(decoded.AccessibilityJSON) == "" {
		decoded.AccessibilityJSON = decoded.AccessibilityJSONCamel
	}
	if strings.TrimSpace(decoded.AccessibilityJSON) == "" {
		return Snapshot{}, fmt.Errorf("%w: BAS response omitted inline accessibility data", ErrCaptureUnavailable)
	}
	var snapshot Snapshot
	if err := json.Unmarshal([]byte(decoded.AccessibilityJSON), &snapshot); err != nil {
		return Snapshot{}, fmt.Errorf("%w: decode accessibility snapshot: %v", ErrCaptureUnavailable, err)
	}
	if snapshot.Contract != snapshotContract {
		return Snapshot{}, fmt.Errorf("%w: unsupported accessibility snapshot contract %q", ErrCaptureUnavailable, snapshot.Contract)
	}
	snapshot.ScreenshotRef = screenshotRefFromArtifacts(decoded.Artifacts)
	snapshot.Timing = CaptureTiming{
		TotalMilliseconds:         firstNonZero(parseCaptureMilliseconds(decoded.Readiness.DurationMS), parseCaptureMilliseconds(decoded.Readiness.DurationMSCamel)),
		NavigationMilliseconds:    firstNonZero(parseCaptureMilliseconds(decoded.Readiness.NavigationDurationMS), parseCaptureMilliseconds(decoded.Readiness.NavigationDurationMSCamel)),
		ReadinessWaitMilliseconds: firstNonZero(parseCaptureMilliseconds(decoded.Readiness.ReadinessWaitDurationMS), parseCaptureMilliseconds(decoded.Readiness.ReadinessWaitDurationMSCamel)),
		Strategy:                  firstNonEmpty(decoded.Readiness.SelectedStrategy, decoded.Readiness.SelectedStrategyCamel),
		Outcome:                   decoded.Readiness.Outcome,
	}
	return snapshot, nil
}

// parseCaptureMilliseconds accepts standard JSON numbers and the quoted int64
// representation emitted by protobuf's canonical JSON mapping.
func parseCaptureMilliseconds(raw json.RawMessage) int64 {
	if len(raw) == 0 || string(raw) == "null" {
		return 0
	}
	var number int64
	if err := json.Unmarshal(raw, &number); err == nil {
		return number
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return 0
	}
	number, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return 0
	}
	return number
}

func firstNonZero(values ...int64) int64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func screenshotRefFromArtifacts(artifacts []struct {
	Type      any               `json:"type"`
	Path      string            `json:"path"`
	SizeBytes int64             `json:"size_bytes"`
	Metadata  map[string]string `json:"metadata"`
},
) string {
	for _, artifact := range artifacts {
		if !isScreenshotArtifactType(artifact.Type) {
			continue
		}
		if artifact.SizeBytes == 0 || strings.TrimSpace(artifact.Metadata["unavailable"]) != "" || strings.TrimSpace(artifact.Metadata["reason"]) != "" {
			continue
		}
		if ref := dataURLFromFile(artifact.Path); ref != "" {
			return ref
		}
	}
	return ""
}

func isScreenshotArtifactType(raw any) bool {
	switch value := raw.(type) {
	case string:
		normalized := strings.ToLower(strings.TrimSpace(value))
		return normalized == "capture_type_screenshot" || normalized == "screenshot" || normalized == "1"
	case float64:
		return value == 1
	default:
		return false
	}
}

func dataURLFromFile(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return ""
	}
	contentType := "application/octet-stream"
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		contentType = "image/png"
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".webp":
		contentType = "image/webp"
	}
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func (c BASCapturer) resolve(ctx context.Context) (string, error) {
	if c.Resolve != nil {
		return c.Resolve(ctx)
	}
	return discovery.ResolveScenarioURLDefault(ctx, basScenarioID)
}

func (c BASCapturer) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	// Full-fidelity capture evaluates the declared viewport/state matrix and
	// may drive several isolated browser sessions. Keep the default long
	// enough for that matrix to finish; callers can still inject a tighter
	// client for bounded operations and tests.
	return &http.Client{Timeout: 10 * time.Minute}
}

func hasMachineClaim(page spec.PageDocument) bool {
	for _, claim := range page.Claims {
		if claim.Tier != "machine" || len(claim.Locales) > 0 || len(claim.Extensions) > 0 {
			continue
		}
		return true
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

func claimAppliesToTarget(claim spec.Claim, target CaptureTarget) (bool, string) {
	if len(claim.Locales) > 0 {
		return false, "locale-scoped claims require locale capture support"
	}
	if len(claim.Extensions) > 0 {
		return false, "extension-scoped claims require a deterministic extension checker"
	}
	if !claimTargetsState(claim, target.StateID) {
		return false, fmt.Sprintf("claim states %v are outside captured state %q", claim.States, target.StateID)
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
	if len(claim.States) == 0 {
		return true
	}
	if stateID != "default" {
		for _, state := range claim.States {
			if state == stateID {
				return true
			}
		}
		return false
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

func unverifiableStateSetupFindings(loc string, page spec.PageDocument) []spec.Finding {
	setups := map[string]bool{"default": true}
	for _, state := range page.States {
		if state.ID != "" && hasStateSetup(state) {
			setups[state.ID] = true
		}
	}
	var findings []spec.Finding
	for _, claim := range page.Claims {
		if claim.Tier != "machine" || len(claim.States) == 0 || isBaselineFloorType(claim.Type) {
			continue
		}
		if claimTargetsState(claim, "default") {
			continue
		}
		var missing []string
		for _, state := range claim.States {
			stateID := state
			if stateID == "" {
				stateID = "default"
			}
			if stateID != "default" && !setups[stateID] {
				missing = append(missing, stateID)
			}
		}
		if len(missing) == 0 {
			continue
		}
		findings = append(findings, spec.Finding{
			Code:       spec.CodeClaimUnverifiable,
			Severity:   spec.SeverityWarning,
			Message:    fmt.Sprintf("machine claim %q targets states without deterministic setup %v", claim.ID, missing),
			Locations:  []string{loc},
			Suggestion: "Add states[].setup for those states, retier the claim, or keep the claim on captured states only.",
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

func componentHarnessRoute(scenario string, component spec.ComponentDocument, version, example string) string {
	catalogID := componentCatalogID(scenario, component)
	route := "/preview/" + url.PathEscape(catalogID) + "/harness.html"
	query := url.Values{}
	if strings.TrimSpace(version) != "" {
		query.Set("version", version)
	}
	if strings.TrimSpace(example) != "" {
		// RCL's harness contract calls the selected story "story". Keep the
		// provider-generated route identical to the public preview route so
		// readiness and accessibility capture observe the requested specimen.
		query.Set("story", example)
	}
	if encoded := query.Encode(); encoded != "" {
		route += "?" + encoded
	}
	return route
}

func componentCatalogID(scenario string, component spec.ComponentDocument) string {
	refParts := strings.Split(filepath.ToSlash(componentCatalogRef(component)), "/")
	for i := 0; i+1 < len(refParts); i++ {
		if refParts[i] == "components" && refParts[i+1] != "" {
			return scenario + ":" + refParts[i+1]
		}
	}
	if strings.TrimSpace(component.Component.Title) != "" {
		return scenario + ":" + strings.TrimSpace(component.Component.Title)
	}
	return scenario + ":" + component.Component.ID
}

func componentVersion(examplesRef string) string {
	parts := strings.Split(filepath.ToSlash(examplesRef), "/")
	for i := 0; i+1 < len(parts); i++ {
		if parts[i] == "versions" {
			return parts[i+1]
		}
	}
	return ""
}

func componentCatalogRef(component spec.ComponentDocument) string {
	if strings.TrimSpace(component.Component.StoryRef) != "" {
		return component.Component.StoryRef
	}
	return component.Component.ExamplesRef
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
	if fallback := findTextSlotNode(nodes, binding, role); fallback != nil {
		for i, node := range nodes {
			if node == fallback {
				return i
			}
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
	return findTextSlotNode(nodes, binding, role)
}

// BAS accessibility snapshots expose visible inline label text as a
// StaticText node and do not carry the source span's data-testid onto that
// node. A declared x-label binding still has a deterministic structural
// target: the first StaticText descendant of the interactive control in the
// same specimen. Keep this fallback narrow to the conventional label slot so
// arbitrary missing bindings remain honest failures.
func findTextSlotNode(nodes []*AXNode, binding spec.Binding, role string) *AXNode {
	if role != "x-label" || !strings.HasSuffix(strings.TrimSpace(binding.TestID), "-label") {
		return nil
	}
	var textDescendant func(node *AXNode) *AXNode
	textDescendant = func(node *AXNode) *AXNode {
		if node == nil {
			return nil
		}
		if isTextOnlyNode(node) && node.Bounds != nil {
			return node
		}
		for index := range node.Children {
			if found := textDescendant(&node.Children[index]); found != nil {
				return found
			}
		}
		return nil
	}
	for _, node := range nodes {
		if isInteractiveNode(node) {
			if found := textDescendant(node); found != nil {
				return found
			}
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
	if strings.Contains(selector, "aria-label=") {
		return ariaLabelValue(selector) == node.Name
	}
	if !strings.ContainsAny(selector, "#.[] >:+~") {
		return strings.EqualFold(node.DOM.Tag, selector) || strings.EqualFold(node.Role, selector)
	}
	return false
}

func ariaLabelValue(selector string) string {
	idx := strings.Index(selector, "aria-label=")
	if idx < 0 {
		return ""
	}
	value := strings.TrimLeft(selector[idx+len("aria-label="):], `'")`)
	end := strings.IndexAny(value, `'"]`)
	if end >= 0 {
		value = value[:end]
	}
	return value
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
