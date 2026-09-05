// Package reconcile checks parsed experience specs against captured
// accessibility-tree evidence.
package reconcile

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

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
	Direction        string
	Orientation      string
	Pointer          string
	MotionPreference string
	InteractionState string
}

// DefaultCaptureProfiles is retained for callers that construct a Check
// without a scenario path. Normal runs load the bounded baseline matrix from
// capabilities/axes.json via captureProfiles.
var DefaultCaptureProfiles = []CaptureProfile{
	{ID: "desktop", MatrixID: "desktop-light-en-rest", Aliases: []string{"wide"}, Width: 1280, Height: 720, ColorScheme: "light", Locale: "en", Orientation: "landscape", Pointer: "fine-hover", MotionPreference: "no-preference", InteractionState: "rest"},
	{ID: "mobile", MatrixID: "mobile-dark-en-reduce", Width: 390, Height: 844, ColorScheme: "dark", Locale: "en", Orientation: "portrait", Pointer: "coarse-no-hover", MotionPreference: "reduce", InteractionState: "rest"},
	{ID: "mobile-landscape", MatrixID: "mobile-landscape-dark-en-reduce", Aliases: []string{"mobile"}, Width: 844, Height: 390, ColorScheme: "dark", Locale: "en", Orientation: "landscape", Pointer: "coarse-no-hover", MotionPreference: "reduce", InteractionState: "rest"},
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
		// Capture engines share a browser/driver resource budget. Serializing
		// the default matrix keeps a slow page capture from starving another
		// session and being misreported as an unavailable authored surface.
		workers = 1
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
	return strings.Join([]string{target.DocumentKind, target.PageID, target.ComponentID, target.Route, target.ExampleName, target.StateID, target.ViewportID, target.ColorScheme, target.Locale, target.Orientation, target.Pointer, target.MotionPreference, target.InteractionState}, "\x00")
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
	Direction         string
	Orientation       string
	Pointer           string
	MotionPreference  string
	InteractionState  string
	SettleMs          int
}

// Name implements checks.Check.
type pageReconciliation struct {
	Findings []spec.Finding
	Evidence []Evidence
}

func reconcileActivePage(loc string, page spec.PageDocument, target CaptureTarget, snapshot Snapshot, contextSnapshots ...map[string]Snapshot) pageReconciliation {
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
				Code:       spec.CodeBindingsUnjoined,
				Severity:   spec.SeverityError,
				Message:    fmt.Sprintf("accessibility capture for %s %q state %q joined zero of %d declared bindings", page.Kind, page.Page.ID, target.StateID, len(activeElements)),
				Locations:  []string{loc},
				Suggestion: "Correct the declared bindings to match the rendered UI; a completed capture with no joined bindings is an authoring defect.",
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
		claimEvidence, ok := evaluateClaim(page, claim, target, nodes, contextSnapshots...)
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

func evaluateClaim(page spec.PageDocument, claim spec.Claim, target CaptureTarget, nodes []*AXNode, contextSnapshots ...map[string]Snapshot) (Evidence, bool) {
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
	var result claimEvaluation
	if (claim.Type == "differential" || claim.Type == "dark-parity") && len(contextSnapshots) > 0 {
		result = evaluateDifferentialClaim(page, claim, contextSnapshots[0])
	} else {
		evaluator := claimEvaluator(claim.Type)
		if evaluator == nil {
			evidence.Verdict = "unverifiable"
			evidence.Message = "claim type has no deterministic structure checker"
			return evidence, false
		}
		result = evaluator(page, claim, target, nodes)
	}
	if !result.Pass && result.Measurement == nil {
		subjects := measuredSubjects(page, claim.Elements, nodes)
		if len(subjects) == 0 {
			subjects = []MeasuredSubject{{ElementID: claim.ID}}
		}
		result.Measurement = measurement(claim.Type, "", "", nil, nil, subjects)
	}
	if result.AXNodeJSON != "" {
		evidence.AXNodeJSON = result.AXNodeJSON
	}
	if result.Measurement != nil {
		if encoded, err := json.Marshal(result.Measurement); err == nil {
			evidence.MeasurementJSON = string(encoded)
		}
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
		} else {
			evidence.Message = measurementFailure(result.Measurement)
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
		item.MeasurementJSON = withCaptureContext(item.MeasurementJSON, target)
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

// withCaptureContext keeps the independent browser context beside the AX
// measurement. The evidence table predates direction and motion as first-class
// axes, so embedding this small typed envelope is backwards compatible with
// existing measurements while allowing differential gates to pair captures
// without guessing from a viewport or story name.
func withCaptureContext(raw string, target CaptureTarget) string {
	var value map[string]any
	if strings.TrimSpace(raw) == "" || strings.TrimSpace(raw) == "{}" {
		value = map[string]any{}
	} else if json.Unmarshal([]byte(raw), &value) != nil || value == nil {
		value = map[string]any{"legacy": json.RawMessage(raw)}
	}
	value["captureContext"] = map[string]string{
		"colorScheme":      target.ColorScheme,
		"locale":           target.Locale,
		"direction":        target.Direction,
		"motionPreference": target.MotionPreference,
		"interactionState": target.InteractionState,
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return raw
	}
	return string(encoded)
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
func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
