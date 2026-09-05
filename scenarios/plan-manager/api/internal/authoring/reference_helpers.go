package authoring

import (
	"fmt"
	"strings"

	planmodel "plan-manager/internal/planmodel"
)

func hasMandatoryViolation(violations []StructureViolation, key SectionKey) bool {
	for _, v := range violations {
		if v.SectionKey == key {
			return true
		}
	}
	return false
}

// firstUnfilledMandatory returns the key of the first mandatory section that
// still needs author input, or "" when every mandatory section is filled.
func firstUnfilledMandatory(sections []Section) SectionKey {
	for _, sec := range sections {
		if sec.Mandatory && strings.TrimSpace(sec.Content) == "" {
			return sec.Key
		}
	}
	return ""
}

func indexOf(sections []Section, key SectionKey) int {
	for i := range sections {
		if sections[i].Key == key {
			return i
		}
	}
	return -1
}

func contentOf(sections []Section, key SectionKey) string {
	if i := indexOf(sections, key); i >= 0 {
		return sections[i].Content
	}
	return ""
}

func degraded(src AutofillSource, key SectionKey, detail string) AutofillResult {
	return AutofillResult{Source: src, SectionKey: key, Filled: false, Degraded: true, Detail: detail}
}

func hasReferencesOrNoCodeReason(content string) bool {
	if strings.TrimSpace(noCodeRefsReason(content)) != "" {
		return true
	}
	return hasReferenceMarker(content)
}

func hasPhaseContextOrNoContextReason(phase PhaseDraft) bool {
	for _, item := range phase.RelevantContext {
		if isNoContextItem(item) {
			return true
		}
		if strings.TrimSpace(item.Label) != "" || strings.TrimSpace(item.Target) != "" ||
			strings.TrimSpace(item.Command) != "" || strings.TrimSpace(item.Instruction) != "" {
			return true
		}
	}
	for _, raw := range phase.RequiredReading {
		if strings.TrimSpace(raw) != "" {
			return true
		}
	}
	return false
}

func isNoContextItem(item planmodel.RelevantContextItem) bool {
	for _, value := range []string{item.Label, item.Reason, item.Instruction, item.Target} {
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(value)), "NO_CONTEXT:") {
			return true
		}
	}
	return false
}

func hasReferenceMarker(content string) bool {
	upper := strings.ToUpper(content)
	return strings.Contains(upper, "[CODE:") ||
		strings.Contains(upper, "[REQ:") ||
		strings.Contains(upper, "[DOC:")
}

// codeFileExts are the source-file extensions used to catch the most common
// reference-kind mistake. Intentionally narrow — only an obvious mismatch is
// rejected, so a legitimate edge case (a doc that ends in an unusual extension,
// a code generator that emits markdown) is never blocked.
var codeFileExts = []string{
	".go", ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs", ".py", ".rs", ".java",
	".rb", ".c", ".h", ".hpp", ".cc", ".cpp", ".cs", ".kt", ".swift", ".proto",
	".sql", ".sh", ".bash", ".yaml", ".yml", ".json", ".toml",
}

func isCodeReferencePath(target string) bool {
	lower := strings.ToLower(strings.TrimSpace(target))
	for _, ext := range codeFileExts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

func isDocReferencePath(target string) bool {
	lower := strings.ToLower(strings.TrimSpace(target))
	if lower == "" {
		return false
	}
	if strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".mdx") || strings.HasSuffix(lower, ".rst") {
		return true
	}
	// A docs/ path segment that does not also resolve to a source file.
	if (strings.Contains(lower, "/docs/") || strings.HasPrefix(lower, "docs/")) && !isCodeReferencePath(lower) {
		return true
	}
	return false
}

// referenceKindMismatch returns an actionable message when a reference's declared
// kind obviously contradicts its target path (a docs path tagged [CODE:], or a
// source file tagged [DOC:]). It returns "" when the kind is plausible, so a
// REQ id, a bare scenario path, or any ambiguous target is left to the author.
func referenceKindMismatch(kind planmodel.ReferenceKind, target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}
	switch kind {
	case planmodel.ReferenceCode:
		if isDocReferencePath(target) && !isCodeReferencePath(target) {
			return fmt.Sprintf("reference %q is marked [CODE:] but points at a documentation path; use [DOC:] for docs", target)
		}
	case planmodel.ReferenceDoc:
		if isCodeReferencePath(target) && !isDocReferencePath(target) {
			return fmt.Sprintf("reference %q is marked [DOC:] but points at a source file; use [CODE:] for code", target)
		}
	}
	return ""
}

// referenceKindViolations flags every declared reference whose kind contradicts
// its target path.
func referenceKindViolations(refs []planmodel.Reference, key SectionKey) []StructureViolation {
	var out []StructureViolation
	for _, ref := range refs {
		if msg := referenceKindMismatch(ref.Kind, ref.Target); msg != "" {
			out = append(out, StructureViolation{SectionKey: key, Message: msg})
		}
	}
	return out
}

// phaseReferenceKindViolations flags a phase's reference kind/path mismatches,
// prefixing each message with the phase label (e.g. "phase 2 reference …").
func phaseReferenceKindViolations(refs []planmodel.Reference, prefix string) []StructureViolation {
	var out []StructureViolation
	for _, ref := range refs {
		if msg := referenceKindMismatch(ref.Kind, ref.Target); msg != "" {
			out = append(out, StructureViolation{SectionKey: SectionPhases, Message: prefix + " " + msg})
		}
	}
	return out
}

// contextItemKindMismatch returns a reference-kind/path mismatch message for a
// code_ref/doc context item, or "" when the kind is plausible.
func contextItemKindMismatch(item planmodel.RelevantContextItem) string {
	switch item.Kind {
	case planmodel.RelevantContextCodeRef:
		return referenceKindMismatch(planmodel.ReferenceCode, item.Target)
	case planmodel.RelevantContextDoc:
		return referenceKindMismatch(planmodel.ReferenceDoc, item.Target)
	default:
		return ""
	}
}

// referencesContentKindViolations parses a references-section body and flags any
// kind/path mismatch. A markup parse error returns no violations here — that case
// is owned by the authored-markup gate (parseReferencesAndPhases) at finalize, so
// the same error is never double-reported as both "invalid markup" and "wrong
// kind".
func referencesContentKindViolations(content string) []StructureViolation {
	if strings.TrimSpace(content) == "" {
		return nil
	}
	refs, err := parseReferencesContent(content)
	if err != nil {
		return nil
	}
	return referenceKindViolations(refs, SectionReferences)
}

func noCodeRefsReason(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(line), "NO_CODE_REFS:") {
			return strings.TrimSpace(line[len("NO_CODE_REFS:"):])
		}
	}
	return ""
}

// noContextReason returns the explicit "NO_CONTEXT:" skip reason recorded in the
// global relevant-context section, or "" when none is present.
func noContextReason(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(line), "NO_CONTEXT:") {
			return strings.TrimSpace(line[len("NO_CONTEXT:"):])
		}
	}
	return ""
}

func noSkillContextReason(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(line), "NO_SKILL_CONTEXT:") {
			return strings.TrimSpace(line[len("NO_SKILL_CONTEXT:"):])
		}
	}
	return ""
}

// globalContextResolved is intentionally lenient: missing plan-wide context is
// advisory, not a finalization blocker. Accepted context still renders and
// executes, but authors are no longer forced through NO_CONTEXT paperwork.
func globalContextResolved(sess Session) bool {
	return true
}

// globalSkillContextResolved reports whether the author has made an explicit
// skill-context decision: a global skill item exists (usually via DiscoverSkillPack)
// or the relevant-context section records a NO_SKILL_CONTEXT:/NO_CONTEXT: skip
// reason. It steers navigation and progress only — finalize never blocks on it
// (missing skill context stays a readiness warning, not a violation).
func globalSkillContextResolved(sess Session) bool {
	if hasSkillContext(sess.RelevantContext) {
		return true
	}
	content := contentOf(sess.Sections, SectionRelevantContext)
	return noSkillContextReason(content) != "" || noContextReason(content) != ""
}

func hasSkillContext(items []planmodel.RelevantContextItem) bool {
	for _, item := range items {
		if item.Kind == planmodel.RelevantContextSkill {
			return true
		}
	}
	return false
}

func parseReferencesContent(content string) ([]planmodel.Reference, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, nil
	}
	var b strings.Builder
	b.WriteString("# References\n\n## References\n\n")
	b.WriteString(content)
	b.WriteString("\n")
	parsed, err := planmodel.ParsePlanMarkdown(b.String())
	if err != nil {
		return nil, err
	}
	return parsed.References, nil
}
