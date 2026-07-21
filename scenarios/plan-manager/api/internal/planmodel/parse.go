package planmodel

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	titleRe               = regexp.MustCompile(`(?m)^#\s+(.+?)\s*$`)
	sectionRe             = regexp.MustCompile(`(?m)^##\s+(.+?)\s*$`)
	subHeadingRe          = regexp.MustCompile(`(?m)^###\s+(.+?)\s*$`)
	phaseRe               = regexp.MustCompile(`(?m)^###\s+Phase\s+(\d+)\s*[—:-]\s*(.+?)\s*$`)
	referenceRe           = regexp.MustCompile(`\[(CODE|REQ|DOC):\s*([^\]]+?)\]`)
	malformedReferenceRe  = regexp.MustCompile(`\[(CODE|REQ|DOC)(?:\s*:)?\s*(?:\]|$)`)
	bulletKeyValueLineRe  = regexp.MustCompile(`(?m)^-\s*([A-Za-z ]+):\s*(.+?)\s*$`)
	contextItemLineRe     = regexp.MustCompile(`^-\s*(.+?)(?:\s+_\((.+)\)_)?\s*$`)
	backtickValueRe       = regexp.MustCompile("`([^`]+)`")
	sectionNumberPrefixRe = regexp.MustCompile(`^\d+[A-Za-z]?\.\s+`)
)

// ParsePlanMarkdown parses a markdown plan into the structured model. It is a
// pure, deterministic adoption helper for the rendered markdown view.
func ParsePlanMarkdown(markdown string) (Plan, error) {
	if strings.TrimSpace(markdown) == "" {
		return Plan{}, ErrInvalidPlan{Reason: "empty markdown"}
	}
	var p Plan
	if m := titleRe.FindStringSubmatch(markdown); m != nil {
		p.Title = strings.TrimSpace(m[1])
	}
	if p.Title == "" {
		return Plan{}, ErrInvalidPlan{Reason: "markdown has no title heading"}
	}
	if err := validateMachineReadableMarkup(markdown); err != nil {
		return Plan{}, err
	}

	ordered := extractOrderedSections(markdown)
	sections := map[string]string{}
	for _, entry := range ordered {
		sections[entry.lower] = entry.body
	}
	// 9-cluster render shape: clusters carry their fields as `###` subsections.
	// Splitting them here lets the same field lookups below serve both the
	// current cluster shape and pre-cluster/legacy flat headings.
	applyClusterSubSections(sections)
	p.Purpose = sections["purpose"]
	p.Definitions = parseDefinitionsTable(sections["definitions"])
	p.Scope = sections["scope"]
	p.Constraints = sections["constraints"]
	p.NonGoals = firstNonEmpty(sections["non-goals"], sections["non goals"])
	p.DefinitionOfDone = firstNonEmpty(sections["definition of done"], sections["definition-of-done"])
	p.ChangeBoundary = ParseChangeBoundaryBlock(firstNonEmpty(sections["change boundary"], sections["acceptance boundary"]))
	p.RegressionAnchor = ParseRegressionAnchorBlock(sections["regression anchor"])
	p.BaselineSet = ParseBaselineSetBlock(firstNonEmpty(sections["regression checks"], sections["baseline set"]))
	// A compact baseline-set projection supersedes the command-wall anchor for
	// new plans. Recreate only the minimum typed anchor intent needed by older
	// quality/validation consumers while keeping the set as the display and
	// execution policy source of truth.
	if p.RegressionAnchor.Strategy == "" && p.BaselineSet.Name != "" {
		p.RegressionAnchor = RegressionAnchor{
			Strategy:     AnchorStrategyChangeBoundary,
			BaselineName: p.BaselineSet.Name,
		}
	}
	// Import upgrade: a legacy plan with no Change Boundary section but a scenario/
	// allowlist anchor gets a boundary DERIVED from that anchor, so imported plans
	// join the boundary model without losing their original blast radius.
	if p.ChangeBoundary.IsZero() {
		p.ChangeBoundary = BoundaryFromLegacyAnchor(p.RegressionAnchor)
	}

	// Professional plan structure (see docs/concepts/PLAN-MODEL.md).
	p.ProblemStatement = firstNonEmpty(sections["problem"], sections["problem / need"], sections["problem/need"], sections["problem need"])
	p.TargetOutcome = firstNonEmpty(sections["outcome"], sections["target outcome"])
	p.Assumptions = sections["assumptions"]
	p.TechnicalApproach = firstNonEmpty(sections["approach & decisions"], sections["technical approach"], sections["technical approach / design rationale"])
	p.ProhibitedApproaches = sections["prohibited approaches"]
	p.RisksHazards = firstNonEmpty(sections["risks / hazards"], sections["risks/hazards"])
	p.ValidationStrategy, p.FinalValidationCommands = parseValidationStrategy(firstNonEmpty(sections["validation strategy"], sections["validation model"]))
	p.Decisions = parsePlanDecisions(sections["decisions"])
	p.AssumptionRisks = parseAssumptionTable(sections["assumptions & risks"])
	p.WorkPosture, p.WorkPostureSource, p.WorkPostureDetail = parseWorkPostureBlock(sections["work posture"])
	p.ImportProvenance = parseImportProvenanceBlock(sections["import provenance"])
	p.PreservedLegacySections = parsePreservedLegacyBlock(sections["preserved legacy sections"])

	// Legacy adoption: map known legacy headings into canonical fields and
	// preserve every other unrecognized section so import is never lossy.
	applyLegacyImport(&p, ordered)

	// Plan-level references are those before the first phase; phase references
	// are recovered per-phase so they do not leak into the plan-level list.
	p.References = parseReferences(prePhaseMarkdown(markdown))
	var err error
	p.RelevantContext, err = parseRelevantContextBlock(sections["global execution setup"], RelevantContextScopeGlobal, "")
	if err != nil {
		return Plan{}, err
	}
	p.RelevantContext = append(p.RelevantContext, migratedRelevantContextFromLines(sections["required reading"], RelevantContextScopeGlobal, "")...)
	p.Phases, err = parsePhases(markdown)
	if err != nil {
		return Plan{}, err
	}
	return p, nil
}

func parseDefinitionsTable(block string) []PlanDefinition {
	var out []PlanDefinition
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") || strings.Contains(line, "---") {
			continue
		}
		line = strings.ReplaceAll(line, "\\|", "\x00")
		parts := strings.Split(line, "|")
		if len(parts) < 4 {
			continue
		}
		term, meaning := strings.TrimSpace(parts[1]), strings.TrimSpace(parts[2])
		if strings.EqualFold(term, "term") || term == "" || meaning == "" {
			continue
		}
		out = append(out, PlanDefinition{Term: strings.ReplaceAll(term, "\x00", "|"), Meaning: strings.ReplaceAll(meaning, "\x00", "|")})
	}
	return out
}

// ParseBaselineSetBlock recovers the baseline intent from current concise
// regression-check prose or the legacy declarative projection. The capture
// command is used only to retain the durable intent when a rendered plan is
// re-imported; GCT still owns operation details and execution state.
func ParseBaselineSetBlock(block string) BaselineSetIntent {
	if strings.TrimSpace(block) == "" {
		return BaselineSetIntent{}
	}
	var out BaselineSetIntent
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- "))
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "name":
			out.Name = trimMarkdownValue(value)
		case "capture policy":
			out.CapturePolicy = trimMarkdownValue(value)
		case "behavioral scenario coverage":
			out.ScenarioTargets = backtickValues(value)
		case "source changes for review (informational)":
			out.RepoPaths = backtickValues(value)
		}
	}
	if out.Name == "" && len(out.ScenarioTargets) == 0 && len(out.RepoPaths) == 0 {
		out = parseBaselineCollectionCapture(block)
	}
	if out.Name == "" && len(out.ScenarioTargets) == 0 && len(out.RepoPaths) == 0 {
		return BaselineSetIntent{}
	}
	if out.CapturePolicy == "" {
		out.CapturePolicy = BaselineCapturePolicyExecutionStart
	}
	out.Compatibility = BaselineSetCompatibilityCurrent
	return out
}

func parseBaselineCollectionCapture(block string) BaselineSetIntent {
	const prefix = "git-control-tower baseline collection capture"
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		fields := strings.Fields(line)
		var out BaselineSetIntent
		for i := 0; i+1 < len(fields); i++ {
			switch fields[i] {
			case "--name":
				out.Name = fields[i+1]
			case "--member":
				out.ScenarioTargets = append(out.ScenarioTargets, fields[i+1])
			case "--path":
				out.RepoPaths = append(out.RepoPaths, fields[i+1])
			}
		}
		if out.Name != "" || len(out.ScenarioTargets) > 0 || len(out.RepoPaths) > 0 {
			return out
		}
	}
	return BaselineSetIntent{}
}

func backtickValues(value string) []string {
	values := backtickValueRe.FindAllStringSubmatch(value, -1)
	out := make([]string, 0, len(values))
	for _, match := range values {
		if len(match) == 2 && strings.TrimSpace(match[1]) != "" {
			out = append(out, strings.TrimSpace(match[1]))
		}
	}
	return out
}

// ParseChangeBoundaryBlock recovers a ChangeBoundary from the rendered
// "## Change Boundary" section: the `**Acceptance allow:**` / `**Acceptance
// deny:**` backticked-bullet lists and an optional `- Operator-only:` line. The
// result is normalized so render -> parse -> render is idempotent.
func ParseChangeBoundaryBlock(block string) ChangeBoundary {
	block = strings.TrimSpace(block)
	if block == "" {
		return ChangeBoundary{}
	}
	var b ChangeBoundary
	current := "" // "allow" | "deny" | ""
	for _, line := range strings.Split(block, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		switch {
		case strings.EqualFold(trimmed, "**Acceptance allow:**"):
			current = "allow"
			continue
		case strings.EqualFold(trimmed, "**Acceptance deny:**"):
			current = "deny"
			continue
		}
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "- operator-only:") {
			b.OperatorOnlyReason = strings.TrimSpace(trimmed[len("- Operator-only:"):])
			current = ""
			continue
		}
		if strings.HasPrefix(trimmed, "- ") {
			val := trimMarkdownValue(strings.TrimSpace(trimmed[len("- "):]))
			if val == "" {
				continue
			}
			switch current {
			case "allow":
				b.AcceptanceAllow = append(b.AcceptanceAllow, val)
			case "deny":
				b.AcceptanceDeny = append(b.AcceptanceDeny, val)
			}
		}
	}
	return b.Normalized()
}

// parsePhaseChangeBoundary recovers a phase's compact boundary refinement block
// (Allow/Deny comma lists + optional Operator-only line).
func parsePhaseChangeBoundary(block string) ChangeBoundary {
	block = strings.TrimSpace(block)
	if block == "" {
		return ChangeBoundary{}
	}
	var b ChangeBoundary
	for _, line := range strings.Split(block, "\n") {
		trimmed := strings.TrimSpace(line)
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
		lower := strings.ToLower(trimmed)
		switch {
		case strings.HasPrefix(lower, "allow:"):
			b.AcceptanceAllow = splitCommaList(trimmed[len("Allow:"):])
		case strings.HasPrefix(lower, "deny:"):
			b.AcceptanceDeny = splitCommaList(trimmed[len("Deny:"):])
		case strings.HasPrefix(lower, "operator-only:"):
			b.OperatorOnlyReason = strings.TrimSpace(trimmed[len("Operator-only:"):])
		}
	}
	return b.Normalized()
}

func parsePhaseValidationScope(block string) ValidationScope {
	block = strings.TrimSpace(block)
	if block == "" {
		return ValidationScope{}
	}
	var scope ValidationScope
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
		lower := strings.ToLower(line)
		switch {
		case strings.HasPrefix(lower, "mode:"):
			scope.Mode = ValidationScopeMode(strings.TrimSpace(line[len("Mode:"):]))
		case strings.HasPrefix(lower, "rationale:"):
			scope.Rationale = strings.TrimSpace(line[len("Rationale:"):])
		case strings.HasPrefix(lower, "allow:"):
			scope.Boundary.AcceptanceAllow = splitCommaList(line[len("Allow:"):])
		case strings.HasPrefix(lower, "deny:"):
			scope.Boundary.AcceptanceDeny = splitCommaList(line[len("Deny:"):])
		}
	}
	scope.Boundary = scope.Boundary.Normalized()
	if scope.Mode != ValidationScopeNarrow && scope.Mode != ValidationScopeFullPlan {
		return ValidationScope{}
	}
	return scope
}

// ParseRegressionAnchorBlock converts the rendered Regression Anchor section, or
// a legacy prose anchor, into typed anchor fields. New plans should render the
// structured bullet form; legacy prose remains readable but is marked explicitly
// so validation cannot silently treat arbitrary text as an oracle.
func ParseRegressionAnchorBlock(block string) RegressionAnchor {
	block = strings.TrimSpace(block)
	if block == "" {
		return RegressionAnchor{}
	}
	var anchor RegressionAnchor
	var legacy []string
	for _, line := range strings.Split(block, "\n") {
		applyRegressionAnchorLine(&anchor, &legacy, line)
	}
	anchor.Strategy = inferredRegressionAnchorStrategy(anchor)
	if len(anchor.Commands) == 0 {
		anchor.Commands = RegressionAnchorCommands(anchor)
	}
	if anchorPresent(anchor) {
		return anchor
	}
	return legacyRegressionAnchor(legacy)
}

// RegressionAnchorCommands derives the canonical check commands implied by a
// typed anchor. Baseline diffs are verdict oracles; sha allowlist diffs are
// informational and intentionally still included for operator review.
func RegressionAnchorCommands(anchor RegressionAnchor) []string {
	switch anchor.Strategy {
	case "scenario_baseline":
		if anchor.Scenario == "" || anchor.BaselineName == "" || strings.ContainsAny(anchor.BaselineName, " \t\r\n") {
			return nil
		}
		return []string{
			"git-control-tower baseline snapshot status --scenario " + anchor.Scenario + " --name " + anchor.BaselineName + " --wait --json",
			"git-control-tower baseline diff --scenario " + anchor.Scenario + " --name " + anchor.BaselineName + " --wait",
		}
	case "head_sha_allowlist":
		if anchor.HeadSha == "" || isExecutionStartHeadSentinel(anchor.HeadSha) {
			return nil
		}
		cmd := "git diff --stat " + anchor.HeadSha
		if len(anchor.AllowlistPaths) > 0 {
			cmd += " -- " + strings.Join(anchor.AllowlistPaths, " ")
		}
		return []string{cmd}
	default:
		return nil
	}
}

func isExecutionStartHeadSentinel(v string) bool {
	v = strings.Trim(strings.TrimSpace(v), "`<>")
	return strings.EqualFold(v, "captured at execution start")
}

func anchorPresent(a RegressionAnchor) bool {
	return a.Strategy != "" || a.Scenario != "" || a.BaselineName != "" || a.HeadSha != "" ||
		len(a.AllowlistPaths) > 0 || len(a.Commands) > 0 || a.CapturedAt != "" ||
		a.CaptureStatus != "" || a.CaptureReason != "" || a.Fallback != "" || a.Unavailable
}

func applyRegressionAnchorLine(anchor *RegressionAnchor, legacy *[]string, line string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return
	}
	if strings.HasPrefix(trimmed, "- ") {
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
	}
	if applyRegressionAnchorField(anchor, trimmed) {
		return
	}
	*legacy = append(*legacy, trimmed)
}

func applyRegressionAnchorField(anchor *RegressionAnchor, trimmed string) bool {
	lower := strings.ToLower(trimmed)
	switch {
	case strings.Contains(lower, "anchor autofill was unavailable"):
		anchor.Unavailable = true
	case strings.HasPrefix(lower, "strategy:"):
		anchor.Strategy = strings.TrimSpace(trimmed[len("Strategy:"):])
	case strings.HasPrefix(lower, "scenario baseline:"):
		applyScenarioBaselineAnchor(anchor, trimmed[len("Scenario baseline:"):])
	case strings.HasPrefix(lower, "baseline name:"):
		anchor.BaselineName = trimMarkdownValue(trimmed[len("Baseline name:"):])
	case strings.HasPrefix(lower, "head sha:"):
		anchor.HeadSha = trimMarkdownValue(trimmed[len("HEAD sha:"):])
	case strings.HasPrefix(lower, "allowlist:"):
		anchor.AllowlistPaths = splitCommaList(trimmed[len("Allowlist:"):])
	case strings.HasPrefix(lower, "captured at:"):
		anchor.CapturedAt = trimMarkdownValue(trimmed[len("Captured at:"):])
	case strings.HasPrefix(lower, "capture status:"):
		anchor.CaptureStatus = strings.TrimSpace(trimmed[len("Capture status:"):])
	case strings.HasPrefix(lower, "capture reason:"):
		anchor.CaptureReason = strings.TrimSpace(trimmed[len("Capture reason:"):])
	case strings.HasPrefix(lower, "fallback:"):
		anchor.Fallback = strings.TrimSpace(trimmed[len("Fallback:"):])
	case strings.HasPrefix(trimmed, "`") && strings.HasSuffix(trimmed, "`"):
		anchor.Commands = append(anchor.Commands, strings.Trim(trimmed, "`"))
	default:
		return false
	}
	return true
}

func applyScenarioBaselineAnchor(anchor *RegressionAnchor, raw string) {
	rest := strings.TrimSpace(raw)
	values := backtickValueRe.FindAllStringSubmatch(rest, -1)
	if len(values) > 0 {
		anchor.Scenario = strings.TrimSpace(values[0][1])
	} else {
		anchor.Scenario = strings.TrimSpace(strings.Split(rest, "(")[0])
	}
	if len(values) > 1 {
		anchor.BaselineName = strings.TrimSpace(values[1][1])
	}
}

func inferredRegressionAnchorStrategy(anchor RegressionAnchor) string {
	if anchor.Strategy != "" {
		return anchor.Strategy
	}
	if anchor.Scenario != "" || anchor.BaselineName != "" {
		return "scenario_baseline"
	}
	if anchor.HeadSha != "" || len(anchor.AllowlistPaths) > 0 {
		return "head_sha_allowlist"
	}
	return ""
}

func legacyRegressionAnchor(legacy []string) RegressionAnchor {
	legacyText := strings.TrimSpace(strings.Join(legacy, "\n"))
	if legacyText == "" {
		return RegressionAnchor{}
	}
	anchor := RegressionAnchor{Strategy: "legacy_prose", BaselineName: legacyText}
	if strings.ContainsAny(legacyText, " \t\r\n") {
		anchor.Unavailable = true
	}
	return anchor
}

func trimMarkdownValue(v string) string {
	v = strings.TrimSpace(v)
	if matches := backtickValueRe.FindStringSubmatch(v); len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}
	return strings.Trim(v, "` ")
}

func splitCommaList(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = trimMarkdownValue(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func validateMachineReadableMarkup(markdown string) error {
	if m := malformedReferenceRe.FindString(markdown); m != "" {
		return ErrInvalidPlan{Reason: "malformed reference marker " + m}
	}
	for _, line := range strings.Split(markdown, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "### Phase") && !phaseRe.MatchString(trimmed) {
			return ErrInvalidPlan{Reason: "malformed phase heading " + trimmed}
		}
	}
	return nil
}

// sectionEntry is one top-level `##` section with its original-case heading
// (preserved so legacy import keeps the source heading verbatim) and trimmed body.
type sectionEntry struct {
	heading string
	lower   string
	body    string
}

func extractOrderedSections(markdown string) []sectionEntry {
	locs := sectionRe.FindAllStringSubmatchIndex(markdown, -1)
	out := make([]sectionEntry, 0, len(locs))
	for i, loc := range locs {
		heading := strings.TrimSpace(markdown[loc[2]:loc[3]])
		bodyStart := loc[1]
		bodyEnd := len(markdown)
		if i+1 < len(locs) {
			bodyEnd = locs[i+1][0]
		}
		body := markdown[bodyStart:bodyEnd]
		if idx := phaseRe.FindStringIndex(body); idx != nil {
			body = body[:idx[0]]
		}
		out = append(out, sectionEntry{
			heading: heading,
			lower:   normalizeSectionKey(heading),
			body:    strings.TrimSpace(body),
		})
	}
	return out
}

// clusterHeadings are the 9-cluster render headings whose fields arrive as
// `###` subsections. Their subsection bodies are lifted into the flat section
// map so field lookups serve both the cluster shape and legacy flat headings.
var clusterHeadings = []string{"boundaries", "assumptions & risks", "verification", "approach & decisions", "execution setup"}

// applyClusterSubSections lifts each cluster's `###` subsections into the flat
// section map (never clobbering an existing flat section). The Approach &
// Decisions cluster keeps its leading prose as its own body (the technical
// approach); Execution Setup aliases to the pre-cluster global-execution-setup
// key so the relevant-context block parser reads it unchanged.
func applyClusterSubSections(sections map[string]string) {
	for _, cluster := range clusterHeadings {
		body, ok := sections[cluster]
		if !ok {
			continue
		}
		lead, subs := splitSubSections(body)
		if cluster == "approach & decisions" || cluster == "assumptions & risks" {
			// These clusters own their leading prose: the technical approach and
			// the assumption/mitigation table respectively.
			sections[cluster] = strings.TrimSpace(lead)
		}
		if cluster == "execution setup" {
			if strings.TrimSpace(sections["global execution setup"]) == "" {
				sections["global execution setup"] = body
			}
			continue
		}
		for _, sub := range subs {
			key := normalizeSectionKey(sub.heading)
			if _, exists := sections[key]; !exists {
				sections[key] = sub.body
			}
		}
	}
}

// parsePlanDecisions recovers the ordered D1..Dn list rendered under
// Approach & Decisions → Decisions: `- **D<n> — <title>:** <statement>`.
// Order comes from list position; the D-number label is presentation.
func parsePlanDecisions(block string) []PlanDecision {
	block = strings.TrimSpace(block)
	if block == "" {
		return nil
	}
	var out []PlanDecision
	for _, line := range strings.Split(block, "\n") {
		m := planDecisionLineRe.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		out = append(out, PlanDecision{Title: strings.TrimSpace(m[1]), Statement: strings.TrimSpace(m[2])})
	}
	return out
}

var planDecisionLineRe = regexp.MustCompile(`^-\s*\*\*D\d+\s*[—:-]\s*(.+?):\*\*\s*(.+)$`)

// parseAssumptionTable recovers the two-column Assumptions & Risks table:
// `| <assumption> | <if wrong → mitigation> |` rows (header/divider skipped).
func parseAssumptionTable(block string) []PlanAssumption {
	block = strings.TrimSpace(block)
	if block == "" {
		return nil
	}
	var out []PlanAssumption
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cells := splitTableRow(line)
		if len(cells) != 2 {
			continue
		}
		if strings.EqualFold(cells[0], "assumption") || strings.HasPrefix(cells[0], "---") {
			continue
		}
		out = append(out, PlanAssumption{Statement: cells[0], Mitigation: cells[1]})
	}
	return out
}

// splitTableRow splits one `| a | b |` row into unescaped cells.
func splitTableRow(line string) []string {
	line = strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(line), "|"), "|")
	placeholder := "\x00"
	line = strings.ReplaceAll(line, "\\|", placeholder)
	parts := strings.Split(line, "|")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(strings.ReplaceAll(part, placeholder, "|"))
		out = append(out, part)
	}
	return out
}

// splitSubSections splits a cluster body into its leading prose and ordered
// `###` subsections.
func splitSubSections(body string) (string, []sectionEntry) {
	locs := subHeadingRe.FindAllStringSubmatchIndex(body, -1)
	if len(locs) == 0 {
		return body, nil
	}
	lead := body[:locs[0][0]]
	out := make([]sectionEntry, 0, len(locs))
	for i, loc := range locs {
		heading := strings.TrimSpace(body[loc[2]:loc[3]])
		bodyStart := loc[1]
		bodyEnd := len(body)
		if i+1 < len(locs) {
			bodyEnd = locs[i+1][0]
		}
		out = append(out, sectionEntry{
			heading: heading,
			lower:   normalizeSectionKey(heading),
			body:    strings.TrimSpace(body[bodyStart:bodyEnd]),
		})
	}
	return lead, out
}

func normalizeSectionKey(heading string) string {
	heading = strings.TrimSpace(heading)
	heading = sectionNumberPrefixRe.ReplaceAllString(heading, "")
	return strings.ToLower(strings.TrimSpace(heading))
}

// prePhaseMarkdown returns the markdown before the first phase heading, so
// plan-level reference scanning never absorbs phase-scoped references.
func prePhaseMarkdown(markdown string) string {
	if idx := phaseRe.FindStringIndex(markdown); idx != nil {
		return markdown[:idx[0]]
	}
	return markdown
}

func parseReferences(markdown string) []Reference {
	matches := referenceRe.FindAllStringSubmatch(markdown, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := map[string]bool{}
	out := make([]Reference, 0, len(matches))
	for _, m := range matches {
		kind := referenceKindFromMarker(m[1])
		target := strings.TrimSpace(m[2])
		key := string(kind) + "|" + target
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, Reference{Kind: kind, Target: target})
	}
	return out
}

func parsePhases(markdown string) ([]Phase, error) {
	locs := phaseRe.FindAllStringSubmatchIndex(markdown, -1)
	if len(locs) == 0 {
		return nil, nil
	}
	out := make([]Phase, 0, len(locs))
	for i, loc := range locs {
		order, _ := strconv.Atoi(markdown[loc[2]:loc[3]])
		title := strings.TrimSpace(markdown[loc[4]:loc[5]])
		bodyStart := loc[1]
		bodyEnd := len(markdown)
		if i+1 < len(locs) {
			bodyEnd = locs[i+1][0]
		}
		body := markdown[bodyStart:bodyEnd]
		ph := Phase{Order: order, Title: title, Status: PhaseStatusTodo}
		for _, kv := range bulletKeyValueLineRe.FindAllStringSubmatch(body, -1) {
			key := strings.ToLower(strings.TrimSpace(kv[1]))
			val := strings.TrimSpace(kv[2])
			switch key {
			case "intent":
				ph.Intent = val
			case "validation":
				ph.Validation = firstNonEmpty(ph.Validation, val)
			case "acceptance":
				ph.Acceptance = val
			case "definition of done":
				ph.Acceptance = firstNonEmpty(ph.Acceptance, val)
			case "status":
				ph.Status = phaseStatusFromLabel(val)
			case "context":
				// The compact NO_CONTEXT render: `- Context: none needed — <reason>`.
				// Reconstruct the typed skip note so the quality gate still sees an
				// explicit NO_CONTEXT decision and re-render reproduces the line.
				if item, ok := noContextItemFromCompactLine(val, ph.ID); ok {
					ph.RelevantContext = append(ph.RelevantContext, item)
				}
			}
		}
		ph.References = parseReferences(body)
		ph.AffectedAreas = listItemsFromBlock(extractPhaseBlock(body, markerAffectedAreas))
		ph.Steps = listItemsFromBlock(extractPhaseBlock(body, markerOrderedSteps))
		ph.ExpectedOutputs = listItemsFromBlock(extractPhaseBlock(body, markerExpectedOutputs))
		ph.RisksHazards = listItemsFromBlock(extractPhaseBlock(body, markerRisksHazards))
		ph.Validation = extractPhaseBlock(body, markerPhaseValidation)
		ph.HandoffNotes = extractPhaseBlock(body, markerHandoffNotes)
		ph.ChangeBoundary = parsePhaseChangeBoundary(extractPhaseBlock(body, markerChangeBoundary))
		ph.ValidationScope = parsePhaseValidationScope(extractPhaseBlock(body, markerValidationScope))
		applyLegacyPhaseSections(&ph, body)
		contextBody := extractPhaseContextSetup(body)
		if contextBody != "" {
			context, err := parseRelevantContextBlock(contextBody, RelevantContextScopePhase, ph.ID)
			if err != nil {
				return nil, err
			}
			ph.RelevantContext = context
		}
		if legacyRequiredReading := extractPhaseRequiredReading(body); legacyRequiredReading != "" {
			ph.RequiredReading = requiredReadingLines(legacyRequiredReading)
			ph.RelevantContext = append(ph.RelevantContext, migratedRelevantContextFromLines(legacyRequiredReading, RelevantContextScopePhase, ph.ID)...)
		}
		out = append(out, ph)
	}
	return out, nil
}

// Phase block markers — the bold-header blocks the renderer emits inside a phase.
// extractPhaseBlock cuts a block at the next marker so blocks never bleed.
const (
	markerAffectedAreas   = "**Affected Areas:**"
	markerPhaseContext    = "**Phase Context Setup:**"
	markerOrderedSteps    = "**Ordered Steps:**"
	markerExpectedOutputs = "**Expected Outputs:**"
	markerPhaseValidation = "**Phase Validation:**"
	markerRisksHazards    = "**Risks / Hazards:**"
	markerHandoffNotes    = "**Handoff Notes:**"
	markerChangeBoundary  = "**Change Boundary:**"
	markerValidationScope = "**Validation Scope:**"
	markerReminders       = "**Reminders:**"
	markerBaselineScope   = "**Baseline scope:**"
	markerPhaseReferences = "**References:**"
	markerRequiredReading = "**Required Reading:**"
)

var phaseBlockMarkers = []string{
	markerAffectedAreas, markerPhaseContext, markerOrderedSteps, markerExpectedOutputs,
	markerPhaseValidation, markerRisksHazards, markerHandoffNotes, markerChangeBoundary, markerValidationScope,
	markerReminders, markerBaselineScope, markerPhaseReferences, markerRequiredReading,
}

// phaseScalarBulletTerminators are the scalar phase bullets (rendered between
// blocks, e.g. "- Acceptance:") that must also terminate a preceding text block
// so a multi-line block (Phase Validation) never absorbs a following bullet.
var phaseScalarBulletTerminators = []string{"\n- Acceptance:", "\n- Status:", "\n- Intent:", "\n- Context:"}

// extractPhaseBlock returns the content of one bold-header block in a phase body,
// terminated by the next phase block marker or scalar bullet (so adjacent
// fields never merge).
func extractPhaseBlock(body, marker string) string {
	idx := strings.Index(body, marker)
	if idx < 0 {
		return ""
	}
	rest := body[idx+len(marker):]
	end := len(rest)
	for _, m := range phaseBlockMarkers {
		if m == marker {
			continue
		}
		if found := strings.Index(rest, m); found >= 0 && found < end {
			end = found
		}
	}
	for _, term := range phaseScalarBulletTerminators {
		if found := strings.Index(rest, term); found >= 0 && found < end {
			end = found
		}
	}
	return strings.TrimSpace(rest[:end])
}

// noContextItemFromCompactLine reconstructs the typed NO_CONTEXT skip note from
// the compact `- Context: none needed — <reason>` phase line.
func noContextItemFromCompactLine(value, phaseID string) (RelevantContextItem, bool) {
	value = strings.TrimSpace(value)
	lower := strings.ToLower(value)
	const marker = "none needed"
	if !strings.HasPrefix(lower, marker) {
		return RelevantContextItem{}, false
	}
	reason := strings.TrimSpace(strings.TrimLeft(value[len(marker):], " —–-:"))
	if reason == "" {
		reason = "no phase-specific setup context."
	}
	label := "NO_CONTEXT: " + reason
	return RelevantContextItem{
		Kind:         RelevantContextNote,
		Scope:        RelevantContextScopePhase,
		PhaseID:      phaseID,
		Label:        label,
		Instruction:  label,
		RepeatPolicy: RelevantContextPhaseEntry,
		Source:       RelevantContextSourceAuthored,
		Status:       RelevantContextStatusReady,
	}, true
}

func extractPhaseContextSetup(body string) string {
	return extractPhaseBlock(body, markerPhaseContext)
}

func extractPhaseRequiredReading(body string) string {
	return extractPhaseBlock(body, markerRequiredReading)
}

// listItemsFromBlock parses a bullet/numbered block body into ordered items,
// stripping a leading "- " or "N. " from each line.
func listItemsFromBlock(block string) []string {
	if strings.TrimSpace(block) == "" {
		return nil
	}
	var out []string
	for _, line := range strings.Split(block, "\n") {
		item := strings.TrimSpace(numberedListPrefix.ReplaceAllString(strings.TrimSpace(line), ""))
		item = strings.TrimSpace(strings.TrimPrefix(item, "-"))
		if item != "" {
			out = append(out, strings.TrimSpace(item))
		}
	}
	return out
}

var numberedListPrefix = regexp.MustCompile(`^\d+\.\s+`)

func applyLegacyPhaseSections(ph *Phase, body string) {
	sections := legacyPhaseSections(body)
	if ph.Intent == "" {
		ph.Intent = strings.TrimSpace(firstNonEmpty(sections["objective"], sections["intent"]))
	}
	if len(ph.Steps) == 0 {
		ph.Steps = listItemsFromBlock(firstNonEmpty(sections["checklist"], sections["steps"], sections["ordered steps"]))
	}
	if len(ph.ExpectedOutputs) == 0 {
		ph.ExpectedOutputs = listItemsFromBlock(firstNonEmpty(sections["expected outputs"], sections["outputs"]))
	}
	if strings.TrimSpace(ph.Validation) == "" {
		ph.Validation = strings.TrimSpace(firstNonEmpty(sections["validation"], sections["testing"]))
	}
	if strings.TrimSpace(ph.Acceptance) == "" {
		ph.Acceptance = strings.TrimSpace(firstNonEmpty(sections["definition of done"], sections["acceptance"]))
	}
}

func legacyPhaseSections(body string) map[string]string {
	sections := map[string]string{}
	current := ""
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if current != "" {
				sections[current] += "\n"
			}
			continue
		}
		if heading, ok := legacyPhaseSectionHeading(trimmed); ok {
			current = heading
			continue
		}
		if isRenderedPhaseStructureLine(trimmed) {
			current = ""
			continue
		}
		if current != "" {
			sections[current] += line + "\n"
		}
	}
	for key, value := range sections {
		sections[key] = strings.TrimSpace(value)
	}
	return sections
}

func legacyPhaseSectionHeading(line string) (string, bool) {
	trimmed := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "- "), ":"))
	key := strings.ToLower(trimmed)
	switch key {
	case "objective", "intent", "checklist", "steps", "ordered steps", "expected outputs", "outputs", "validation", "testing", "definition of done", "acceptance":
		return key, true
	default:
		return "", false
	}
}

func isRenderedPhaseStructureLine(line string) bool {
	if strings.HasPrefix(line, "**") || strings.HasPrefix(line, "### ") {
		return true
	}
	lower := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "-")))
	return strings.HasPrefix(lower, "status:") ||
		strings.HasPrefix(lower, "intent:") ||
		strings.HasPrefix(lower, "acceptance:") ||
		strings.HasPrefix(lower, "validation:") ||
		strings.HasPrefix(lower, "definition of done:")
}

func requiredReadingLines(block string) []string {
	return legacySetupLines(block)
}

func legacySetupLines(block string) []string {
	var out []string
	lines := strings.Split(block, "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		switch {
		case line == "":
			continue
		case strings.HasPrefix(line, "```"):
			fenceLines, next := collectFenceLines(lines, i+1)
			out = append(out, splitSetupFenceCommands(fenceLines)...)
			i = next
		case strings.HasPrefix(line, "-"):
			item := strings.TrimSpace(strings.TrimPrefix(line, "-"))
			if item != "" {
				out = append(out, item)
			}
		case looksLikeSetupCommand(line) || looksLikeSetupTarget(line):
			out = append(out, line)
		}
	}
	return out
}

func collectFenceLines(lines []string, start int) ([]string, int) {
	var out []string
	for i := start; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
			return out, i
		}
		out = append(out, strings.TrimSpace(lines[i]))
	}
	return out, len(lines) - 1
}

func splitSetupFenceCommands(lines []string) []string {
	var out []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func looksLikeSetupTarget(line string) bool {
	lower := strings.ToLower(strings.TrimSpace(line))
	return strings.HasPrefix(lower, "docs/") ||
		strings.HasPrefix(lower, "scenarios/") ||
		strings.HasPrefix(lower, "packages/") ||
		strings.HasSuffix(lower, ".md") ||
		strings.HasPrefix(lower, "[req:") ||
		strings.HasPrefix(lower, "req:") ||
		strings.HasPrefix(lower, "[code:") ||
		strings.HasPrefix(lower, "code:")
}

func looksLikeSetupCommand(line string) bool {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return false
	}
	switch fields[0] {
	case "prompt-manager", "search-hub", "cli-health", "swarm-manager", "plan-manager", "vrooli", "test-genie", "git-control-tower", "sed":
		return true
	default:
		return strings.HasPrefix(strings.ToLower(line), "cli:")
	}
}

func migratedRelevantContextFromLines(block string, scope RelevantContextScope, phaseID string) []RelevantContextItem {
	lines := requiredReadingLines(block)
	if len(lines) == 0 {
		return nil
	}
	out := make([]RelevantContextItem, 0, len(lines))
	for _, line := range lines {
		out = append(out, RelevantContextItemFromSetupLine(line, scope, phaseID, "Migrated from legacy Required Reading."))
	}
	return out
}

func lastField(line string) string {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return ""
	}
	return strings.Trim(fields[len(fields)-1], "`'\"")
}

func parseRelevantContextBlock(block string, scope RelevantContextScope, phaseID string) ([]RelevantContextItem, error) {
	block = strings.TrimSpace(block)
	if block == "" {
		return nil, nil
	}
	lines := strings.Split(block, "\n")
	items := make([]RelevantContextItem, 0)
	currentKind := RelevantContextKind("")
	var current *RelevantContextItem
	for i := 0; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], " \t")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "### ") {
			kind, ok := relevantContextKindFromHeading(strings.TrimSpace(strings.TrimPrefix(trimmed, "### ")))
			if !ok {
				currentKind = ""
				current = nil
				continue
			}
			currentKind = kind
			current = nil
			continue
		}
		if currentKind == "" {
			continue
		}
		if strings.HasPrefix(line, "- ") {
			item, err := parseRelevantContextItemLine(line, currentKind, scope, phaseID)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
			current = &items[len(items)-1]
			continue
		}
		if current == nil {
			return nil, ErrInvalidPlan{Reason: "malformed relevant context line " + trimmed}
		}
		switch {
		case strings.HasPrefix(trimmed, "- Reason:"):
			current.Reason = strings.TrimSpace(strings.TrimPrefix(trimmed, "- Reason:"))
		case strings.HasPrefix(trimmed, "- Instruction:"):
			current.Instruction = strings.TrimSpace(strings.TrimPrefix(trimmed, "- Instruction:"))
		case strings.HasPrefix(trimmed, "- Status:"):
			current.StatusDetail = strings.TrimSpace(strings.TrimPrefix(trimmed, "- Status:"))
		case trimmed == "```bash":
			command, next, err := parseContextCommandFence(lines, i+1)
			if err != nil {
				return nil, err
			}
			current.Command = command
			applyRelevantContextCommandInference(current)
			i = next
		default:
			return nil, ErrInvalidPlan{Reason: "malformed relevant context line " + trimmed}
		}
	}
	return items, nil
}

func parseRelevantContextItemLine(line string, kind RelevantContextKind, scope RelevantContextScope, phaseID string) (RelevantContextItem, error) {
	m := contextItemLineRe.FindStringSubmatch(line)
	if m == nil {
		return RelevantContextItem{}, ErrInvalidPlan{Reason: "malformed relevant context item " + strings.TrimSpace(line)}
	}
	label := strings.TrimSpace(m[1])
	label, inlineCommand := splitInlineRelevantContextCommand(label)
	item := RelevantContextItem{
		Kind:         kind,
		Scope:        scope,
		PhaseID:      phaseID,
		Label:        label,
		Required:     false,
		RepeatPolicy: defaultRelevantContextRepeatPolicy(scope),
		Source:       RelevantContextSourceAuthored,
		Status:       RelevantContextStatusReady,
	}
	if kind == RelevantContextNote {
		item.Instruction = label
	} else {
		item.Target = label
	}
	if kind == RelevantContextReqRef || kind == RelevantContextCodeRef {
		item.Target = targetFromReferenceLikeLabel(label)
		item.Kind = inferReferenceContextKind(label, item.Target)
	}
	for _, annotation := range strings.Split(m[2], ",") {
		applyRelevantContextAnnotation(&item, strings.TrimSpace(annotation))
	}
	if inlineCommand != "" {
		item.Command = inlineCommand
		applyRelevantContextCommandInference(&item)
	}
	return item, nil
}

func splitInlineRelevantContextCommand(label string) (string, string) {
	trimmed := strings.TrimSpace(label)
	if !strings.HasSuffix(trimmed, "`") {
		return label, ""
	}
	end := strings.LastIndex(trimmed[:len(trimmed)-1], "`")
	if end < 0 {
		return label, ""
	}
	command := strings.TrimSpace(trimmed[end+1 : len(trimmed)-1])
	before := strings.TrimSpace(trimmed[:end])
	before = strings.TrimSpace(strings.TrimSuffix(before, "—"))
	before = strings.TrimSpace(strings.TrimSuffix(before, "-"))
	if before == "" || command == "" {
		return label, ""
	}
	return before, command
}

func parseContextCommandFence(lines []string, start int) (string, int, error) {
	var command []string
	for i := start; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "```" {
			return strings.TrimSpace(strings.Join(command, "\n")), i, nil
		}
		command = append(command, strings.TrimSpace(lines[i]))
	}
	return "", len(lines), ErrInvalidPlan{Reason: "unterminated relevant context command fence"}
}

func relevantContextKindFromHeading(heading string) (RelevantContextKind, bool) {
	switch strings.ToLower(strings.TrimSpace(heading)) {
	case "load skills":
		return RelevantContextSkill, true
	case "read docs":
		return RelevantContextDoc, true
	case "run discovery searches":
		return RelevantContextSearch, true
	case "run commands":
		return RelevantContextCommand, true
	case "inspect references":
		return RelevantContextCodeRef, true
	case "operator notes":
		return RelevantContextNote, true
	default:
		return "", false
	}
}

func defaultRelevantContextRepeatPolicy(scope RelevantContextScope) RelevantContextRepeatPolicy {
	if scope == RelevantContextScopePhase {
		return RelevantContextPhaseEntry
	}
	return RelevantContextOncePerExecution
}

func applyRelevantContextAnnotation(item *RelevantContextItem, annotation string) {
	switch strings.ToLower(annotation) {
	case "":
		return
	case "required":
		item.Required = true
	case "run on resume":
		item.RepeatPolicy = RelevantContextOnResume
	case "run every phase":
		item.RepeatPolicy = RelevantContextEveryPhase
	case "as needed":
		item.RepeatPolicy = RelevantContextAsNeeded
	case "authored":
		item.Source = RelevantContextSourceAuthored
	case "discovered":
		item.Source = RelevantContextSourceDiscovered
	case "migrated":
		item.Source = RelevantContextSourceMigrated
	case "autofilled":
		item.Source = RelevantContextSourceAutofilled
	case "ready":
		item.Status = RelevantContextStatusReady
	case "degraded":
		item.Status = RelevantContextStatusDegraded
	case "unresolved":
		item.Status = RelevantContextStatusUnresolved
	}
}

func applyRelevantContextCommandInference(item *RelevantContextItem) {
	if item.Kind == RelevantContextSkill {
		// A repeated `prompt-manager skill read` prefix is unambiguous corruption
		// (a skill slug is a single token and can never contain the prefix): a
		// renderer defect once emitted the doubled command into mirrors, so
		// re-importing such a mirror repairs it deterministically here.
		item.Command = collapseRepeatedSkillReadPrefix(item.Command)
	}
	fields := strings.Fields(item.Command)
	if len(fields) == 0 {
		return
	}
	if item.Kind == RelevantContextSkill && len(fields) >= 4 &&
		fields[0] == "prompt-manager" && fields[1] == "skill" && fields[2] == "read" {
		item.Target = strings.Join(fields[3:], " ")
	}
	if (item.Kind == RelevantContextDoc || item.Kind == RelevantContextCodeRef || item.Kind == RelevantContextReqRef) &&
		len(fields) >= 4 && fields[0] == "sed" {
		item.Target = fields[len(fields)-1]
	}
	if item.Kind == RelevantContextSearch && item.Target == "" {
		item.Target = item.Command
	}
}

// collapseRepeatedSkillReadPrefix rewrites `prompt-manager skill read
// prompt-manager skill read <slug>` (any repetition depth) to a single-prefix
// command. Only the exact repeated-prefix shape is touched.
func collapseRepeatedSkillReadPrefix(command string) string {
	const prefix = "prompt-manager skill read "
	trimmed := strings.TrimSpace(command)
	if !strings.HasPrefix(trimmed, prefix) {
		return command
	}
	rest := strings.TrimPrefix(trimmed, prefix)
	for strings.HasPrefix(rest, prefix) {
		rest = strings.TrimPrefix(rest, prefix)
	}
	return prefix + rest
}

func targetFromReferenceLikeLabel(label string) string {
	if m := referenceRe.FindStringSubmatch(label); m != nil {
		return strings.TrimSpace(m[2])
	}
	return label
}

func inferReferenceContextKind(label, target string) RelevantContextKind {
	upper := strings.ToUpper(label)
	switch {
	case strings.Contains(upper, "[REQ:") || strings.HasPrefix(strings.ToLower(target), "req:") || strings.Contains(target, "requirements/"):
		return RelevantContextReqRef
	default:
		return RelevantContextCodeRef
	}
}

func referenceKindFromMarker(marker string) ReferenceKind {
	switch strings.ToUpper(strings.TrimSpace(marker)) {
	case "REQ":
		return ReferenceReq
	case "DOC":
		return ReferenceDoc
	default:
		return ReferenceCode
	}
}

// parseValidationStrategy splits a Validation Strategy section into its prose and
// the trailing "**Final validation commands:**" list (backticked commands).
func parseValidationStrategy(block string) (string, []string) {
	block = strings.TrimSpace(block)
	if block == "" {
		return "", nil
	}
	const marker = "**Final validation commands:**"
	idx := strings.Index(block, marker)
	if idx < 0 {
		return block, nil
	}
	prose := strings.TrimSpace(block[:idx])
	var commands []string
	for _, line := range strings.Split(block[idx+len(marker):], "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
		if cmd := strings.Trim(strings.TrimSpace(line), "`"); cmd != "" {
			commands = append(commands, cmd)
		}
	}
	return prose, commands
}

// parseWorkPostureBlock recovers the autofilled posture from the rendered Work
// Posture bullets so render -> parse -> render is idempotent.
func parseWorkPostureBlock(block string) (WorkPosture, WorkPostureSource, string) {
	block = strings.TrimSpace(block)
	if block == "" {
		return WorkPostureUnspecified, WorkPostureSourceUnspecified, ""
	}
	var posture WorkPosture
	var source WorkPostureSource
	var detail string
	for _, line := range strings.Split(block, "\n") {
		trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
		lower := strings.ToLower(trimmed)
		switch {
		case strings.HasPrefix(lower, "posture:"):
			posture = WorkPosture(strings.ToLower(strings.Trim(strings.TrimSpace(trimmed[len("Posture:"):]), "* ")))
		case strings.HasPrefix(lower, "source:"):
			source = WorkPostureSource(strings.TrimSpace(trimmed[len("Source:"):]))
		case strings.HasPrefix(lower, "detail:"):
			detail = strings.TrimSpace(trimmed[len("Detail:"):])
		}
	}
	return posture, source, detail
}

func parseImportProvenanceBlock(block string) *ImportProvenance {
	block = strings.TrimSpace(block)
	if block == "" {
		return nil
	}
	prov := &ImportProvenance{}
	for _, line := range strings.Split(block, "\n") {
		trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
		lower := strings.ToLower(trimmed)
		switch {
		case strings.HasPrefix(lower, "source:"):
			prov.SourcePath = trimMarkdownValue(trimmed[len("Source:"):])
		case strings.HasPrefix(lower, "original format:"):
			prov.OriginalFormat = strings.TrimSpace(trimmed[len("Original format:"):])
		case strings.HasPrefix(lower, "imported at:"):
			prov.ImportedAt = trimMarkdownValue(trimmed[len("Imported at:"):])
		case strings.HasPrefix(lower, "workspace id:"):
			prov.WorkspaceID = trimMarkdownValue(trimmed[len("Workspace ID:"):])
		case strings.HasPrefix(lower, "workspace root:"):
			prov.WorkspaceRoot = trimMarkdownValue(trimmed[len("Workspace root:"):])
		case strings.HasPrefix(lower, "note:"):
			prov.Note = strings.TrimSpace(trimmed[len("Note:"):])
		}
	}
	if prov.SourcePath == "" && prov.OriginalFormat == "" && prov.ImportedAt == "" && prov.Note == "" && prov.WorkspaceID == "" && prov.WorkspaceRoot == "" {
		return nil
	}
	return prov
}

// parsePreservedLegacyBlock recovers preserved legacy sections from the rendered
// "Preserved Legacy Sections" block (its `### Heading` subsections plus the
// Mapped-to / Preservation-reason bullets and verbatim content).
func parsePreservedLegacyBlock(block string) []LegacySection {
	block = strings.TrimSpace(block)
	if block == "" {
		return nil
	}
	locs := subHeadingRe.FindAllStringSubmatchIndex(block, -1)
	if len(locs) == 0 {
		return nil
	}
	var out []LegacySection
	for i, loc := range locs {
		heading := strings.TrimSpace(block[loc[2]:loc[3]])
		bodyStart := loc[1]
		bodyEnd := len(block)
		if i+1 < len(locs) {
			bodyEnd = locs[i+1][0]
		}
		body := block[bodyStart:bodyEnd]
		sec := LegacySection{Heading: heading, PreservationReason: PreservationReasonUnmapped}
		var content []string
		for _, line := range strings.Split(body, "\n") {
			trimmed := strings.TrimSpace(line)
			lower := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(trimmed, "-")))
			switch {
			case strings.HasPrefix(lower, "mapped to:"):
				sec.MappedTo = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(strings.TrimPrefix(trimmed, "-")), "Mapped to:"))
			case strings.HasPrefix(lower, "preservation reason:"):
				sec.PreservationReason = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(strings.TrimPrefix(trimmed, "-")), "Preservation reason:"))
			default:
				content = append(content, line)
			}
		}
		sec.Content = strings.TrimSpace(strings.Join(content, "\n"))
		out = append(out, sec)
	}
	return out
}

// legacyImportMap maps known legacy 13-section headings to the canonical field
// they adopt into. Mapped sections become first-class fields; everything else
// unrecognized is preserved verbatim (never silently dropped).
var legacyImportMap = map[string]func(*Plan, string){
	"problem statement":               func(p *Plan, v string) { p.ProblemStatement = firstNonEmpty(p.ProblemStatement, v) },
	"target end state":                func(p *Plan, v string) { p.TargetOutcome = firstNonEmpty(p.TargetOutcome, v) },
	"implementation strategy":         func(p *Plan, v string) { p.TechnicalApproach = firstNonEmpty(p.TechnicalApproach, v) },
	"testing plan":                    func(p *Plan, v string) { p.ValidationStrategy = firstNonEmpty(p.ValidationStrategy, v) },
	"risks and mitigations":           func(p *Plan, v string) { p.RisksHazards = firstNonEmpty(p.RisksHazards, v) },
	"risks + mitigations":             func(p *Plan, v string) { p.RisksHazards = firstNonEmpty(p.RisksHazards, v) },
	"risks & mitigations":             func(p *Plan, v string) { p.RisksHazards = firstNonEmpty(p.RisksHazards, v) },
	"non-goals / prohibited patterns": func(p *Plan, v string) { p.NonGoals = firstNonEmpty(p.NonGoals, v) },
}

// canonicalConsumedHeadings is the set of headings the parser already maps into
// a canonical field or handles structurally; they are never preserved as legacy.
var canonicalConsumedHeadings = map[string]bool{
	"purpose": true, "problem / need": true, "problem/need": true, "problem need": true,
	"target outcome": true, "work posture": true, "scope": true, "non-goals": true,
	"non goals": true, "assumptions": true, "technical approach": true,
	"technical approach / design rationale": true, "constraints": true,
	"prohibited approaches": true, "global execution setup": true, "execution feedback": true, "references": true,
	"change boundary": true, "acceptance boundary": true,
	"regression anchor": true, "validation strategy": true, "validation model": true,
	"definition of done": true, "definition-of-done": true, "risks / hazards": true,
	"risks/hazards": true, "import provenance": true, "preserved legacy sections": true,
	"plan graph": true, "phases": true, "required reading": true,
	// 9-cluster headings (contract decision D1).
	"problem": true, "outcome": true, "approach & decisions": true, "boundaries": true,
	"assumptions & risks": true, "verification": true, "execution setup": true,
}

// applyLegacyImport maps recognized legacy headings into canonical fields and
// preserves every other unrecognized, non-empty section as a LegacySection.
func applyLegacyImport(p *Plan, ordered []sectionEntry) {
	for _, entry := range ordered {
		lower := entry.lower
		if mapper, ok := legacyImportMap[lower]; ok {
			mapper(p, entry.body)
			continue
		}
		if canonicalConsumedHeadings[lower] {
			continue
		}
		if strings.TrimSpace(entry.body) == "" {
			continue
		}
		p.PreservedLegacySections = append(p.PreservedLegacySections, LegacySection{
			Heading:            entry.heading,
			Content:            entry.body,
			PreservationReason: PreservationReasonUnmapped,
		})
	}
}

func phaseStatusFromLabel(s string) PhaseStatus {
	switch strings.ToLower(strings.Trim(strings.TrimSpace(s), "*")) {
	case "active":
		return PhaseStatusActive
	case "done":
		return PhaseStatusDone
	case "blocked":
		return PhaseStatusBlocked
	default:
		return PhaseStatusTodo
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
