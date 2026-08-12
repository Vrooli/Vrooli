package reconcile

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strings"

	"experience-manager/internal/spec"
)

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
		findings = append(findings, componentSourceFindings(report, loc, component)...)
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
				findings = append(findings, result.Findings...)
				findings = append(findings, c.persistEvidence(ctx, report.Scenario, loc, page, target, snapshot, result.Evidence)...)
			}
		}
		findings = append(findings, unverifiableOutOfMatrixFindings(loc, page, profiles)...)
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
		Kind: "experience-component",
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
