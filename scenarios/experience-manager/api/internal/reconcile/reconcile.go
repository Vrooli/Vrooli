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

// ErrCaptureUnavailable means the capture mechanism could not provide an AX
// snapshot. Reconciliation maps it to skipped evidence, not a failed claim.
var ErrCaptureUnavailable = errors.New("accessibility capture unavailable")

// Check runs structure reconciliation as a checks.Check.
type Check struct {
	Capturer   Capturer
	Repository EvidenceRepository
	Now        func() time.Time
}

// Capturer returns one single-location accessibility snapshot for a page route.
type Capturer interface {
	CaptureAccessibility(ctx context.Context, target CaptureTarget) (Snapshot, error)
}

// CaptureTarget identifies the scenario UI surface to inspect.
type CaptureTarget struct {
	Scenario string
	Route    string
	PageID   string
	SettleMs int
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
		if status != "active" || !hasDefaultMachineClaim(page) {
			continue
		}
		capturer := c.Capturer
		if capturer == nil {
			capturer = BASCapturer{}
		}
		target := CaptureTarget{
			Scenario: report.Scenario,
			Route:    firstRoute(page.Page.Routes),
			PageID:   page.Page.ID,
			SettleMs: defaultSettleMs,
		}
		snapshot, err := capturer.CaptureAccessibility(ctx, target)
		if err != nil || snapshot.Contract != snapshotContract || len(snapshot.Flatten()) == 0 {
			findings = append(findings, spec.Finding{
				Code:       spec.CodeCaptureUnavailable,
				Severity:   spec.SeverityInfo,
				Message:    fmt.Sprintf("accessibility capture unavailable for active page %q", page.Page.ID),
				Locations:  []string{loc},
				Suggestion: "Start browser-automation-studio and the target UI, then rerun the experience phase.",
			})
			findings = append(findings, c.persistEvidence(ctx, report.Scenario, loc, page, target, Snapshot{}, skippedEvidence(page, "capture unavailable"))...)
			continue
		}
		result := reconcileActivePage(loc, page, snapshot)
		findings = append(findings, result.Findings...)
		findings = append(findings, c.persistEvidence(ctx, report.Scenario, loc, page, target, snapshot, result.Evidence)...)
	}
	sortFindings(findings)
	return findings
}

func expectedDraftFindings(loc string, page spec.PageDocument) []spec.Finding {
	expected := expectedFailures(page)
	var findings []spec.Finding
	for _, claim := range page.Claims {
		if !expected[claim.ID] {
			continue
		}
		code := spec.CodeClaimFailed
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

func reconcileActivePage(loc string, page spec.PageDocument, snapshot Snapshot) pageReconciliation {
	nodes := snapshot.Flatten()
	var findings []spec.Finding
	var evidence []Evidence
	resolvedBindings := 0
	activeElements := activeDefaultMachineElementIDs(page)
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
			Evidence: skippedEvidence(page, "no declared bindings joined the accessibility tree"),
		}
	}
	for _, claim := range page.Claims {
		if claim.Tier != "machine" {
			continue
		}
		if !claimTargetsDefault(claim) {
			if claim.Type == "state-covered" || claim.Type == "state-distinct" {
				claimEvidence := unreachableStateEvidence(page, claim)
				evidence = append(evidence, claimEvidence)
				findings = append(findings, spec.Finding{
					Code:       spec.CodeClaimUnverifiable,
					Severity:   spec.SeverityWarning,
					Message:    fmt.Sprintf("machine claim %q is unverifiable: %s", claim.ID, claimEvidence.Message),
					Locations:  []string{loc},
					Suggestion: "Use a default-state claim, provide manual attestation, or add multi-state capture support.",
				})
			}
			continue
		}
		claimEvidence, ok := evaluateClaim(page, claim, nodes)
		evidence = append(evidence, claimEvidence)
		if ok {
			continue
		}
		code := spec.CodeClaimFailed
		severity := spec.SeverityError
		message := fmt.Sprintf("machine claim %q was not proven by the accessibility snapshot", claim.ID)
		suggestion := "Update the UI, binding, or claim tier so structure evidence matches intent."
		if claimEvidence.Verdict == "unverifiable" {
			code = spec.CodeClaimUnverifiable
			severity = spec.SeverityWarning
			message = fmt.Sprintf("machine claim %q is unverifiable: %s", claim.ID, claimEvidence.Message)
			suggestion = "Use a supported claim type, retier the claim, or add deterministic checker coverage."
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

func unreachableStateEvidence(page spec.PageDocument, claim spec.Claim) Evidence {
	return Evidence{
		PageID:    page.Page.ID,
		Route:     firstRoute(page.Page.Routes),
		StateID:   defaultStateID(claim),
		ClaimID:   claim.ID,
		ClaimType: claim.Type,
		Verdict:   "unverifiable",
		Message:   "v1 BAS reconciliation captures only the default state; non-default states require multi-state capture",
	}
}

func skippedEvidence(page spec.PageDocument, message string) []Evidence {
	var out []Evidence
	for _, claim := range page.Claims {
		if claim.Tier != "machine" || !claimTargetsDefault(claim) {
			continue
		}
		out = append(out, Evidence{
			PageID:    page.Page.ID,
			Route:     firstRoute(page.Page.Routes),
			StateID:   defaultStateID(claim),
			ClaimID:   claim.ID,
			ClaimType: claim.Type,
			Verdict:   "skipped",
			Message:   message,
		})
	}
	return out
}

func evaluateClaim(page spec.PageDocument, claim spec.Claim, nodes []*AXNode) (Evidence, bool) {
	evidence := Evidence{
		PageID:    page.Page.ID,
		Route:     firstRoute(page.Page.Routes),
		StateID:   defaultStateID(claim),
		ClaimID:   claim.ID,
		ClaimType: claim.Type,
		Verdict:   "passed",
		Message:   "claim proven by accessibility snapshot",
	}
	pass := true
	switch claim.Type {
	case "state-covered":
		pass = claimTargetsDefault(claim)
	case "state-distinct":
		if len(claim.States) < 2 {
			pass = false
			break
		}
		for _, state := range claim.States {
			if state != "" && state != "default" {
				evidence.Verdict = "unverifiable"
				evidence.Message = "v1 BAS reconciliation captures only the default state; non-default states require multi-state capture"
				return evidence, false
			}
		}
	case "element-present", "single-dominant-action", "keyboard-reachable":
		if len(claim.Elements) == 0 {
			pass = false
			break
		}
		for _, elementID := range claim.Elements {
			node := findBoundNode(nodes, page.Bindings.Elements[elementID], elementRole(page, elementID))
			if node == nil {
				pass = false
				continue
			}
			evidence.AXNodeJSON = encodeAXNode(node)
			if claim.Type == "keyboard-reachable" && !node.KeyboardReachable() {
				pass = false
			}
		}
	case "reading-order":
		if len(claim.Elements) <= 1 {
			pass = false
			break
		}
		last := -1
		for _, elementID := range claim.Elements {
			idx := findBoundIndex(nodes, page.Bindings.Elements[elementID], elementRole(page, elementID))
			if idx < 0 || idx < last {
				pass = false
			}
			if idx >= 0 {
				evidence.AXNodeJSON = encodeAXNode(nodes[idx])
			}
			last = idx
		}
	default:
		evidence.Verdict = "unverifiable"
		evidence.Message = "claim type has no deterministic structure checker"
		return evidence, false
	}
	if !pass {
		evidence.Verdict = "failed"
		evidence.Message = "claim was not proven by accessibility snapshot"
	}
	return evidence, pass
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
	key := strings.Join([]string{e.Scenario, e.PageID, e.StateID, e.ClaimID, e.CaptureRef, e.CheckedAt}, "\x00")
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
		TimeoutMs string `json:"timeoutMs"`
	}
	type captureRequestPayload struct {
		URL                 string          `json:"url"`
		InlineAccessibility bool            `json:"inlineAccessibility"`
		Label               string          `json:"label"`
		WaitFor             *waitForPayload `json:"waitFor,omitempty"`
		InteractionFlowJSON string          `json:"interactionFlowJson,omitempty"`
	}
	payload := captureRequestPayload{
		URL:                 targetURL,
		InlineAccessibility: true,
		Label:               "experience-manager structure reconciliation",
	}
	if target.SettleMs > 0 {
		payload.WaitFor = &waitForPayload{TimeoutMs: strconv.Itoa(target.SettleMs)}
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
		AccessibilityJSON string `json:"accessibilityJson"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return Snapshot{}, ErrCaptureUnavailable
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
		if claim.Tier == "machine" && claimTargetsDefault(claim) {
			return true
		}
	}
	return false
}

func activeDefaultMachineElementIDs(page spec.PageDocument) map[string]bool {
	out := map[string]bool{}
	for _, claim := range page.Claims {
		if claim.Tier != "machine" || !claimTargetsDefault(claim) {
			continue
		}
		for _, elementID := range claim.Elements {
			out[elementID] = true
		}
	}
	return out
}

func claimTargetsDefault(claim spec.Claim) bool {
	if len(claim.Viewports) > 0 || len(claim.Locales) > 0 || len(claim.Extensions) > 0 {
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
