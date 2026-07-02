package authoring

import (
	"fmt"
	"strings"

	planmodel "plan-manager/internal/planmodel"

	"github.com/google/uuid"
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

// referenceSuggestionQuery builds the broad search-hub query for reference
// discovery from the rich authoring inputs (title + scope + technical approach).
// Broad on purpose: search-hub federates/ranks and the locator-shape routing is
// the Answer-projection filter, so we never need a brittle taxonomy gate here.
func referenceSuggestionQuery(sess Session) string {
	parts := []string{sess.Title}
	parts = append(parts, contentOf(sess.Sections, SectionScope))
	parts = append(parts, contentOf(sess.Sections, SectionTechnicalApproach))
	var b strings.Builder
	for _, part := range parts {
		if p := strings.TrimSpace(part); p != "" {
			if b.Len() > 0 {
				b.WriteString(" ")
			}
			b.WriteString(p)
		}
	}
	return b.String()
}

// normalizeReferenceCandidate fills ids and the pending default so a suggester
// (or test fake) need not set bookkeeping fields.
func normalizeReferenceCandidate(candidate ReferenceCandidate) ReferenceCandidate {
	candidate.ID = strings.TrimSpace(candidate.ID)
	if candidate.ID == "" {
		candidate.ID = uuid.NewString()
	}
	candidate.Reference.ID = strings.TrimSpace(candidate.Reference.ID)
	if candidate.Reference.ID == "" {
		candidate.Reference.ID = uuid.NewString()
	}
	candidate.Reference.Target = strings.TrimSpace(candidate.Reference.Target)
	candidate.Source = strings.TrimSpace(candidate.Source)
	candidate.Detail = strings.TrimSpace(candidate.Detail)
	candidate.Handle = strings.TrimSpace(candidate.Handle)
	candidate.BatchID = strings.TrimSpace(candidate.BatchID)
	candidate.Tier = strings.TrimSpace(candidate.Tier)
	candidate.RejectionReason = strings.TrimSpace(candidate.RejectionReason)
	if len(candidate.Corroboration) == 0 && (candidate.Source != "" || candidate.Confidence != 0) {
		candidate.Corroboration = []ProbeHit{{Probe: candidate.Source, Score: candidate.Confidence}}
	}
	if candidate.Status == "" {
		candidate.Status = ReferenceCandidatePending
	}
	return candidate
}

func indexOfReferenceCandidate(candidates []ReferenceCandidate, id string) int {
	id = strings.TrimSpace(id)
	for i := range candidates {
		if candidates[i].ID == id || candidates[i].Handle == id {
			return i
		}
	}
	return -1
}

// appendAcceptedReference appends one reviewed locator line to the references
// section content and marks it filled (author-reviewed, not autofilled).
func appendAcceptedReference(sess *Session, ref planmodel.Reference) {
	idx := indexOf(sess.Sections, SectionReferences)
	if idx < 0 {
		return
	}
	line := "[" + referenceMarker(ref.Kind) + ": " + ref.Target + "]"
	existing := strings.TrimRight(sess.Sections[idx].Content, "\n")
	if strings.Contains(existing, line) {
		return
	}
	if strings.TrimSpace(existing) == "" {
		sess.Sections[idx].Content = line
	} else {
		sess.Sections[idx].Content = existing + "\n" + line
	}
	sess.Sections[idx].Filled = true
	sess.Sections[idx].Autofilled = false
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

// globalContextResolved reports whether the plan-wide relevant-context checkpoint
// has been addressed: at least one accepted/submitted global context item, or an
// explicit NO_CONTEXT skip reason recorded in the relevant-context section.
func globalContextResolved(sess Session) bool {
	if len(sess.RelevantContext) > 0 {
		return true
	}
	return noContextReason(contentOf(sess.Sections, SectionRelevantContext)) != ""
}

// globalSkillContextResolved is the skill checkpoint gate v3: it demands
// evidence of a batch-level sweep, not per-candidate bookkeeping. Satisfied when
// the latest discovery batch was applied, when legacy pre-batch candidates were
// fully dispositioned, or when an explicit NO_SKILL_CONTEXT/NO_CONTEXT skip
// reason is recorded.
func globalSkillContextResolved(sess Session) bool {
	if _, ok := latestAppliedDiscoveryBatch(sess); ok {
		return true
	}
	if len(sess.DiscoveryBatches) == 0 && len(sess.ContextCandidates) > 0 && pendingLegacyContextCandidates(sess) == 0 {
		return true
	}
	content := contentOf(sess.Sections, SectionRelevantContext)
	return noSkillContextReason(content) != "" || noContextReason(content) != ""
}

// pendingLegacyContextCandidates counts pre-batch discovery candidates still
// awaiting an accept/reject decision. Current batches are gated by batch status,
// and longlist candidates are intentionally not disposition-required.
func pendingLegacyContextCandidates(sess Session) int {
	n := 0
	for _, c := range sess.ContextCandidates {
		if c.BatchID == "" && c.Status == ContextCandidatePending {
			n++
		}
	}
	return n
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
