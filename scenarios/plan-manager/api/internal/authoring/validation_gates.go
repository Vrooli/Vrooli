package authoring

import (
	"context"
	"fmt"
	"strings"

	"github.com/vrooli/api-core/markedrefs"

	planmodel "plan-manager/internal/planmodel"
)

const referencesGateMessage = "references must include at least one [CODE:], [REQ:], or [DOC:] reference, or a NO_CODE_REFS: reason"

// boundaryGateMessage is the single message for the change-boundary requirement.
// The boundary is mandatory but satisfiable by an OPERATOR_ONLY: reason, so its
// requirement is enforced by this gate (not the generic empty-mandatory message).
const boundaryGateMessage = "change boundary must declare acceptance_allow paths (one glob per line), or an OPERATOR_ONLY: reason for no-code/operator-only work"

// boundaryGateViolations enforces the change-boundary invariants on a submitted
// acceptance-boundary section: an allow list (or operator-only reason) is
// required and no glob may contain an unresolved placeholder.
func boundaryGateViolations(content string) []StructureViolation {
	b := planmodel.ParseBoundarySection(content)
	if b.IsZero() {
		return []StructureViolation{{SectionKey: SectionAcceptanceBoundary, Message: boundaryGateMessage}}
	}
	var out []StructureViolation
	for _, problem := range planmodel.ValidateBoundary(b, true) {
		out = append(out, StructureViolation{SectionKey: SectionAcceptanceBoundary, Message: problem})
	}
	return out
}

// anchorPlaceholderViolations rejects unresolved authoring placeholders in the
// parsed regression-anchor's scenario, allowlist, and derived commands. The
// HEAD-sha field is exempt: "<captured at execution start>" is intentional intent
// the executor fills with a real sha when execution begins.
func anchorPlaceholderViolations(content string) []StructureViolation {
	if strings.TrimSpace(content) == "" {
		return nil
	}
	anchor := planmodel.ParseRegressionAnchorBlock(content)
	var out []StructureViolation
	check := func(field, value string) {
		if tokens := planmodel.UnresolvedPlaceholders(value); len(tokens) > 0 {
			out = append(out, StructureViolation{
				SectionKey: SectionRegressionAnchor,
				Message:    "regression anchor " + field + " has unresolved placeholder(s) " + strings.Join(tokens, ", "),
			})
		}
	}
	check("scenario", anchor.Scenario)
	for _, p := range anchor.AllowlistPaths {
		check("allowlist", p)
	}
	for _, c := range anchor.Commands {
		check("command", c)
	}
	return out
}

// decisionsGateViolations enforces the pinned-decision line format at submit
// time: every non-empty line needs `<title>: <statement>` with both sides
// present, so a rendered D-list is never missing its handle or its content.
func decisionsGateViolations(content string) []StructureViolation {
	var out []StructureViolation
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
		if line == "" {
			continue
		}
		title, statement, found := strings.Cut(line, ":")
		if !found || strings.TrimSpace(title) == "" || strings.TrimSpace(statement) == "" {
			out = append(out, StructureViolation{
				SectionKey: SectionDecisions,
				Message:    fmt.Sprintf("decision line %q must be '<title>: <statement>' with both parts present", line),
			})
		}
	}
	return out
}

func structureViolations(sections []Section) []StructureViolation {
	var out []StructureViolation
	for _, sec := range sections {
		// References and the change boundary are mandatory, but each owns its gate
		// (allowing NO_CODE_REFS / OPERATOR_ONLY) — skip the generic empty-mandatory
		// message to avoid double-reporting.
		if sec.Key == SectionReferences || sec.Key == SectionAcceptanceBoundary {
			continue
		}
		if sec.Mandatory && strings.TrimSpace(sec.Content) == "" {
			out = append(out, StructureViolation{
				SectionKey: sec.Key,
				Message:    "mandatory section " + string(sec.Key) + " must not be empty",
			})
		}
	}
	if strings.TrimSpace(contentOf(sections, SectionRegressionAnchor)) == "" &&
		!hasMandatoryViolation(out, SectionRegressionAnchor) {
		out = append(out, StructureViolation{
			SectionKey: SectionRegressionAnchor,
			Message:    "regression anchor must be captured before finalizing",
		})
	}
	return out
}

func sessionViolations(sess Session) []StructureViolation {
	out := structureViolations(sess.Sections)
	refsContent := contentOf(sess.Sections, SectionReferences)
	if !hasReferencesOrNoCodeReason(refsContent) {
		out = append(out, StructureViolation{SectionKey: SectionReferences, Message: referencesGateMessage})
	}
	out = append(out, referencesContentKindViolations(refsContent)...)
	out = append(out, boundaryGateViolations(contentOf(sess.Sections, SectionAcceptanceBoundary))...)
	out = append(out, anchorPlaceholderViolations(contentOf(sess.Sections, SectionRegressionAnchor))...)
	out = append(out, postureConflictViolations(sess)...)
	if n := pendingContextCandidates(sess); n > 0 {
		out = append(out, StructureViolation{
			SectionKey: SectionRelevantContext,
			Message:    fmt.Sprintf("%d discovery candidate(s) are undispositioned; accept (context-accept) or reject with a reason (context-reject) every candidate before finalizing", n),
		})
	}
	if globalContextResolved(sess) && !globalSkillContextResolved(sess) {
		out = append(out, StructureViolation{
			SectionKey: SectionRelevantContext,
			Message:    "skill setup needs evidence of a sweep: run context-discover for 2-5 decomposed concepts and disposition every candidate, or record NO_SKILL_CONTEXT: <reason> when no relevant internal skill exists",
		})
	}
	for _, phase := range sess.PhaseDrafts {
		out = append(out, phaseViolations(phase)...)
	}
	return out
}

// greenfieldContradictions are tokens an author should never put in a greenfield
// plan's constraints/prohibited approaches — the posture already forbids them, so
// authoring them is a contradiction the renderer must not echo (the Greenfield
// block is injected by posture, not authored). The default posture is greenfield,
// so this is the conservative check until a brownfield override exists.
var greenfieldContradictions = []string{
	"compatibility shim", "compat shim", "backward compat", "backwards compat",
	"legacy wrapper", "compatibility layer",
}

// postureConflictViolations flags authored constraints/prohibited-approaches that
// contradict the default greenfield posture, so the rendered plan never shows
// guidance that fights the injected Greenfield block.
func postureConflictViolations(sess Session) []StructureViolation {
	var out []StructureViolation
	for _, key := range []SectionKey{SectionConstraints, SectionProhibitedApproaches} {
		lower := strings.ToLower(contentOf(sess.Sections, key))
		if strings.TrimSpace(lower) == "" {
			continue
		}
		for _, token := range greenfieldContradictions {
			if strings.Contains(lower, token) {
				out = append(out, StructureViolation{
					SectionKey: key,
					Message:    "section " + string(key) + " contradicts the greenfield work posture (mentions \"" + token + "\"); greenfield plans forbid compatibility shims/legacy wrappers — remove it or record a brownfield override",
				})
				break
			}
		}
	}
	return out
}

// normalizeForCompare lowercases, trims, and collapses internal whitespace so two
// strings that differ only cosmetically compare equal (used to reject a phase
// acceptance that merely restates its validation).
func normalizeForCompare(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), " ")
}

func phaseViolations(phase PhaseDraft) []StructureViolation {
	var out []StructureViolation
	prefix := "phase"
	if phase.Order > 0 {
		prefix = fmt.Sprintf("phase %d", phase.Order)
	}
	if strings.TrimSpace(phase.Title) == "" {
		out = append(out, StructureViolation{SectionKey: SectionPhases, Message: prefix + " title must not be empty"})
	}
	if strings.TrimSpace(phase.Intent) == "" {
		out = append(out, StructureViolation{SectionKey: SectionPhases, Message: prefix + " intent must not be empty"})
	}
	if len(phase.Steps) == 0 {
		out = append(out, StructureViolation{SectionKey: SectionPhases, Message: prefix + " must include at least one ordered implementation step"})
	}
	if strings.TrimSpace(phase.Validation) == "" {
		out = append(out, StructureViolation{SectionKey: SectionPhases, Message: prefix + " must include phase validation (the method of checking it)"})
	}
	if strings.TrimSpace(phase.Acceptance) == "" {
		out = append(out, StructureViolation{SectionKey: SectionPhases, Message: prefix + " acceptance must not be empty"})
	}
	if a, v := normalizeForCompare(phase.Acceptance), normalizeForCompare(phase.Validation); a != "" && a == v {
		out = append(out, StructureViolation{
			SectionKey: SectionPhases,
			Message:    prefix + " acceptance must not be identical to its validation: acceptance is the outcome gate, validation is the checking method",
		})
	}
	if len(phase.References) == 0 && strings.TrimSpace(phase.NoCodeRefsReason) == "" {
		out = append(out, StructureViolation{
			SectionKey: SectionPhases,
			Message:    prefix + " must include references or a no_code_refs_reason",
		})
	}
	out = append(out, phaseReferenceKindViolations(phase.References, prefix)...)
	if !hasPhaseContextOrNoContextReason(phase) {
		out = append(out, StructureViolation{
			SectionKey: SectionPhases,
			Message:    prefix + " must include phase relevant_context or a NO_CONTEXT: reason",
		})
	}
	return out
}

// violationsForSection returns the gate violations specific to one submitted
// section (empty when it passes). A mandatory or regression-anchor section with
// empty content fails.
func violationsForSection(sec Section) []StructureViolation {
	var out []StructureViolation
	empty := strings.TrimSpace(sec.Content) == ""
	if sec.Key == SectionReferences {
		// References uses its own gate (which allows a NO_CODE_REFS: reason)
		// rather than the generic empty-mandatory message.
		if !hasReferencesOrNoCodeReason(sec.Content) {
			out = append(out, StructureViolation{SectionKey: SectionReferences, Message: referencesGateMessage})
		}
		// Semantic kind/path gate: a docs path tagged [CODE:] (or vice versa) is
		// rejected at submit time, not silently accepted into session state.
		out = append(out, referencesContentKindViolations(sec.Content)...)
		return out
	}
	if sec.Key == SectionAcceptanceBoundary {
		// The boundary uses its own gate (which allows an OPERATOR_ONLY: reason and
		// rejects unresolved placeholders) rather than the generic empty message.
		return boundaryGateViolations(sec.Content)
	}
	if sec.Key == SectionDecisions {
		return decisionsGateViolations(sec.Content)
	}
	if sec.Mandatory && empty {
		out = append(out, StructureViolation{
			SectionKey: sec.Key,
			Message:    "mandatory section " + string(sec.Key) + " must not be empty",
		})
	}
	if sec.Key == SectionRegressionAnchor && empty && !sec.Mandatory {
		out = append(out, StructureViolation{
			SectionKey: SectionRegressionAnchor,
			Message:    "regression anchor must be captured before finalizing",
		})
	}
	return out
}

func (s *service) commandViolationsForSections(ctx context.Context, sections []Section) []StructureViolation {
	var out []StructureViolation
	for _, sec := range sections {
		out = append(out, s.commandViolationsForSection(ctx, sec)...)
	}
	return out
}

func (s *service) commandViolationsForSection(ctx context.Context, sec Section) []StructureViolation {
	if strings.TrimSpace(sec.Content) == "" {
		return nil
	}
	refs := commandRefsInSection(sec)
	if len(refs) == 0 {
		return nil
	}
	if s.commands == nil {
		return []StructureViolation{{
			SectionKey: sec.Key,
			Message:    "command reference validation unavailable: CLI Health command validator is not configured",
		}}
	}
	var out []StructureViolation
	for _, ref := range refs {
		if !markedrefs.RequiresExistence(ref) {
			continue
		}
		result, err := s.commands.ValidateCommandReference(ctx, CommandReferenceRequest{
			CommandText: ref.Value,
			Qualifiers:  append([]string(nil), ref.Qualifiers...),
		})
		if err != nil {
			out = append(out, StructureViolation{
				SectionKey: sec.Key,
				Message:    fmt.Sprintf("command reference %q could not be validated through CLI Health: %v", ref.Value, err),
			})
			continue
		}
		switch strings.ToLower(result.Verdict) {
		case "valid", "partial", "skipped":
			continue
		default:
			out = append(out, StructureViolation{
				SectionKey: sec.Key,
				Message:    commandReferenceViolationMessage(ref.Value, result),
			})
		}
	}
	return out
}

func commandRefsInSection(sec Section) []markedrefs.Reference {
	var out []markedrefs.Reference
	for lineNumber, line := range strings.Split(sec.Content, "\n") {
		for _, ref := range markedrefs.ParseInlineCode(line, lineNumber+1) {
			if ref.Marker == markedrefs.MarkerCLI {
				out = append(out, ref)
			}
		}
	}
	return out
}

func commandReferenceViolationMessage(command string, result CommandReferenceResult) string {
	var parts []string
	for _, issue := range result.Issues {
		if issue.Code != "" && issue.Message != "" {
			parts = append(parts, issue.Code+": "+issue.Message)
		} else if issue.Message != "" {
			parts = append(parts, issue.Message)
		}
	}
	for _, suggestion := range result.Suggestions {
		if suggestion != "" {
			parts = append(parts, "suggestion: "+suggestion)
		}
	}
	parts = append(parts, result.Guidance...)
	if len(parts) == 0 {
		detail := strings.TrimSpace(strings.Join([]string{result.Verdict, result.ValidationLevel}, " "))
		if detail == "" {
			detail = "CLI Health returned no validation detail"
		}
		parts = append(parts, detail)
	}
	return fmt.Sprintf("command reference %q is not a valid current command: %s", command, strings.Join(parts, "; "))
}
