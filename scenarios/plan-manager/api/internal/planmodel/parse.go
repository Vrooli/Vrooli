package planmodel

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	titleRe              = regexp.MustCompile(`(?m)^#\s+(.+?)\s*$`)
	sectionRe            = regexp.MustCompile(`(?m)^##\s+(.+?)\s*$`)
	subHeadingRe         = regexp.MustCompile(`(?m)^###\s+(.+?)\s*$`)
	phaseRe              = regexp.MustCompile(`(?m)^###\s+Phase\s+(\d+)\s*[—:-]\s*(.+?)\s*$`)
	referenceRe          = regexp.MustCompile(`\[(CODE|REQ|DOC):\s*([^\]]+?)\]`)
	malformedReferenceRe = regexp.MustCompile(`\[(CODE|REQ|DOC)(?:\s*:)?\s*(?:\]|$)`)
	bulletKeyValueLineRe = regexp.MustCompile(`(?m)^-\s*([A-Za-z ]+):\s*(.+?)\s*$`)
	contextItemLineRe    = regexp.MustCompile(`^-\s*(.+?)(?:\s+_\((.+)\)_)?\s*$`)
	backtickValueRe      = regexp.MustCompile("`([^`]+)`")
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
	p.Purpose = sections["purpose"]
	p.Scope = sections["scope"]
	p.Constraints = sections["constraints"]
	p.NonGoals = firstNonEmpty(sections["non-goals"], sections["non goals"])
	p.DefinitionOfDone = firstNonEmpty(sections["definition of done"], sections["definition-of-done"])
	p.RegressionAnchor = ParseRegressionAnchorBlock(sections["regression anchor"])

	// Professional plan structure (see docs/concepts/PLAN-MODEL.md).
	p.ProblemStatement = firstNonEmpty(sections["problem / need"], sections["problem/need"], sections["problem need"])
	p.TargetOutcome = sections["target outcome"]
	p.Assumptions = sections["assumptions"]
	p.TechnicalApproach = firstNonEmpty(sections["technical approach"], sections["technical approach / design rationale"])
	p.ProhibitedApproaches = sections["prohibited approaches"]
	p.RisksHazards = firstNonEmpty(sections["risks / hazards"], sections["risks/hazards"])
	p.ValidationStrategy, p.FinalValidationCommands = parseValidationStrategy(firstNonEmpty(sections["validation strategy"], sections["validation model"]))
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
			"git-control-tower baseline snapshot status --scenario " + anchor.Scenario + " --name " + anchor.BaselineName,
			"git-control-tower baseline diff --scenario " + anchor.Scenario + " --name " + anchor.BaselineName,
		}
	case "head_sha_allowlist":
		if anchor.HeadSha == "" {
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

func anchorPresent(a RegressionAnchor) bool {
	return a.Strategy != "" || a.Scenario != "" || a.BaselineName != "" || a.HeadSha != "" ||
		len(a.AllowlistPaths) > 0 || len(a.Commands) > 0 || a.CapturedAt != "" || a.Unavailable
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
			lower:   strings.ToLower(heading),
			body:    strings.TrimSpace(body),
		})
	}
	return out
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
			case "acceptance":
				ph.Acceptance = val
			case "status":
				ph.Status = phaseStatusFromLabel(val)
			}
		}
		ph.References = parseReferences(body)
		ph.AffectedAreas = listItemsFromBlock(extractPhaseBlock(body, markerAffectedAreas))
		ph.Steps = listItemsFromBlock(extractPhaseBlock(body, markerOrderedSteps))
		ph.ExpectedOutputs = listItemsFromBlock(extractPhaseBlock(body, markerExpectedOutputs))
		ph.RisksHazards = listItemsFromBlock(extractPhaseBlock(body, markerRisksHazards))
		ph.Validation = extractPhaseBlock(body, markerPhaseValidation)
		ph.HandoffNotes = extractPhaseBlock(body, markerHandoffNotes)
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
	markerReminders       = "**Reminders:**"
	markerBaselineScope   = "**Baseline scope:**"
	markerPhaseReferences = "**References:**"
	markerRequiredReading = "**Required Reading:**"
)

var phaseBlockMarkers = []string{
	markerAffectedAreas, markerPhaseContext, markerOrderedSteps, markerExpectedOutputs,
	markerPhaseValidation, markerRisksHazards, markerHandoffNotes, markerReminders,
	markerBaselineScope, markerPhaseReferences, markerRequiredReading,
}

// phaseScalarBulletTerminators are the scalar phase bullets (rendered between
// blocks, e.g. "- Acceptance:") that must also terminate a preceding text block
// so a multi-line block (Phase Validation) never absorbs a following bullet.
var phaseScalarBulletTerminators = []string{"\n- Acceptance:", "\n- Status:", "\n- Intent:"}

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

func requiredReadingLines(block string) []string {
	var out []string
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func migratedRelevantContextFromLines(block string, scope RelevantContextScope, phaseID string) []RelevantContextItem {
	lines := requiredReadingLines(block)
	if len(lines) == 0 {
		return nil
	}
	out := make([]RelevantContextItem, 0, len(lines))
	for _, line := range lines {
		out = append(out, migratedRelevantContextItem(line, scope, phaseID))
	}
	return out
}

func migratedRelevantContextItem(line string, scope RelevantContextScope, phaseID string) RelevantContextItem {
	item := RelevantContextItem{
		Kind:         RelevantContextNote,
		Scope:        scope,
		PhaseID:      phaseID,
		Label:        line,
		Reason:       "Migrated from legacy Required Reading.",
		Instruction:  "Load or inspect this context before implementation work.",
		Target:       line,
		Required:     true,
		RepeatPolicy: defaultRelevantContextRepeatPolicy(scope),
		Source:       RelevantContextSourceMigrated,
		Status:       RelevantContextStatusReady,
	}
	if scope == RelevantContextScopePhase {
		item.RepeatPolicy = RelevantContextPhaseEntry
	}
	lower := strings.ToLower(line)
	switch {
	case strings.HasPrefix(lower, "prompt-manager skill read "):
		item.Kind = RelevantContextSkill
		item.Command = line
		item.Argv = strings.Fields(line)
		item.Target = strings.TrimSpace(strings.TrimPrefix(line, "prompt-manager skill read "))
		item.Instruction = "Load this internal skill before implementation."
	case strings.HasPrefix(lower, "search-hub "):
		item.Kind = RelevantContextSearch
		item.Command = line
		item.Argv = strings.Fields(line)
		item.Instruction = "Run this discovery search before implementation."
	case strings.HasPrefix(lower, "cli:"):
		item.Kind = RelevantContextCommand
		item.Command = strings.TrimSpace(line[len("cli:"):])
		item.Argv = strings.Fields(item.Command)
		item.Target = ""
		item.Instruction = "Run this command before implementation."
	case strings.HasPrefix(lower, "docs/") || strings.HasSuffix(lower, ".md"):
		item.Kind = RelevantContextDoc
		item.Instruction = "Read this document before implementation."
	case strings.HasPrefix(lower, "[req:") || strings.HasPrefix(lower, "req:") || strings.Contains(lower, "requirements/"):
		item.Kind = RelevantContextReqRef
		item.Target = targetFromReferenceLikeLabel(line)
		item.Instruction = "Inspect this requirement before implementation."
	case strings.HasPrefix(lower, "[code:") || strings.HasPrefix(lower, "code:"):
		item.Kind = RelevantContextCodeRef
		item.Target = targetFromReferenceLikeLabel(line)
		item.Instruction = "Inspect this code reference before implementation."
	}
	return item
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
	return item, nil
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
		case strings.HasPrefix(lower, "note:"):
			prov.Note = strings.TrimSpace(trimmed[len("Note:"):])
		}
	}
	if prov.SourcePath == "" && prov.OriginalFormat == "" && prov.ImportedAt == "" && prov.Note == "" {
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
	"prohibited approaches": true, "global execution setup": true, "references": true,
	"regression anchor": true, "validation strategy": true, "validation model": true,
	"definition of done": true, "definition-of-done": true, "risks / hazards": true,
	"risks/hazards": true, "import provenance": true, "preserved legacy sections": true,
	"plan graph": true, "phases": true, "required reading": true,
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
