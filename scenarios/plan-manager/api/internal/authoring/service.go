package authoring

import (
	"context"
	"strings"

	"plan-manager/internal/clock"
	internalplans "plan-manager/internal/plans"

	"github.com/google/uuid"
)

// Service is the authoring application surface — the guided composer wizard.
type Service interface {
	StartSession(ctx context.Context, title, slug, templateID string) (Session, error)
	GetSection(ctx context.Context, sessionID string, key SectionKey) (Section, error)
	SubmitSection(ctx context.Context, sessionID string, key SectionKey, content string) (Session, []StructureViolation, error)
	Next(ctx context.Context, sessionID string) (Section, bool, error)
	ValidateStructure(ctx context.Context, sessionID string) (bool, []StructureViolation, error)
	Autofill(ctx context.Context, sessionID string, sources []AutofillSource) (Session, []AutofillResult, error)
	Finalize(ctx context.Context, sessionID string) (internalplans.Plan, error)
}

type service struct {
	store        SessionStore
	writer       PlanWriter
	anchor       AnchorAutofiller
	reading      RequiredReadingSource
	references   ReferenceExtractor
	templateSeed TemplateSeeder
	clock        clock.Clock
}

// Deps wires the authoring Service. Store + Writer are required (a nil store
// cannot persist sessions; a nil writer cannot finalize). The autofill seams are
// optional — a nil seam degrades that source honestly (the section is left for
// the author, never a false fill). TemplateSeeder is optional (nil => the default
// skeleton; a template id is then ignored).
type Deps struct {
	Store          SessionStore
	Writer         PlanWriter
	Anchor         AnchorAutofiller
	RequiredRead   RequiredReadingSource
	References     ReferenceExtractor
	TemplateSeeder TemplateSeeder
	Clock          clock.Clock
}

// TemplateSeeder pre-scaffolds the section skeleton from a template id. Optional;
// production may wire the plans domain's templates. A nil seeder (or an unknown
// id) falls back to the default skeleton.
type TemplateSeeder interface {
	Skeleton(ctx context.Context, templateID string) ([]Section, bool)
}

// NewService constructs the authoring Service.
func NewService(d Deps) Service {
	clk := d.Clock
	if clk == nil {
		clk = clock.System{}
	}
	return &service{
		store:        d.Store,
		writer:       d.Writer,
		anchor:       d.Anchor,
		reading:      d.RequiredRead,
		references:   d.References,
		templateSeed: d.TemplateSeeder,
		clock:        clk,
	}
}

var _ Service = (*service)(nil)

func (s *service) StartSession(ctx context.Context, title, slug, templateID string) (Session, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Session{}, ErrInvalidSession{Reason: "title is required"}
	}
	sections := newSkeleton()
	if templateID != "" && s.templateSeed != nil {
		if seeded, ok := s.templateSeed.Skeleton(ctx, templateID); ok && len(seeded) > 0 {
			sections = seeded
		}
	}
	now := s.now()
	sess := Session{
		ID:                uuid.NewString(),
		Title:             title,
		Slug:              strings.TrimSpace(slug),
		Sections:          sections,
		CurrentSectionKey: firstUnfilledMandatory(sections),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := s.store.Save(ctx, sess); err != nil {
		return Session{}, err
	}
	return sess, nil
}

func (s *service) GetSection(ctx context.Context, sessionID string, key SectionKey) (Section, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Section{}, err
	}
	idx := indexOf(sess.Sections, key)
	if idx < 0 {
		return Section{}, ErrSectionNotFound{SessionID: sessionID, SectionKey: string(key)}
	}
	return sess.Sections[idx], nil
}

func (s *service) SubmitSection(ctx context.Context, sessionID string, key SectionKey, content string) (Session, []StructureViolation, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, nil, err
	}
	idx := indexOf(sess.Sections, key)
	if idx < 0 {
		return Session{}, nil, ErrSectionNotFound{SessionID: sessionID, SectionKey: string(key)}
	}
	sess.Sections[idx].Content = content
	sess.Sections[idx].Filled = strings.TrimSpace(content) != ""
	sess.Sections[idx].Autofilled = false // an author submission supersedes any autofill
	violations := violationsForSection(sess.Sections[idx])
	sess.CurrentSectionKey = firstUnfilledMandatory(sess.Sections)
	sess.UpdatedAt = s.now()
	if err := s.store.Save(ctx, sess); err != nil {
		return Session{}, nil, err
	}
	return sess, violations, nil
}

func (s *service) Next(ctx context.Context, sessionID string) (Section, bool, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Section{}, false, err
	}
	key := firstUnfilledMandatory(sess.Sections)
	if key == "" {
		return Section{}, true, nil
	}
	idx := indexOf(sess.Sections, key)
	return sess.Sections[idx], false, nil
}

func (s *service) ValidateStructure(ctx context.Context, sessionID string) (bool, []StructureViolation, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return false, nil, err
	}
	violations := structureViolations(sess.Sections)
	return len(violations) == 0, violations, nil
}

func (s *service) Autofill(ctx context.Context, sessionID string, sources []AutofillSource) (Session, []AutofillResult, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, nil, err
	}
	if len(sources) == 0 {
		sources = []AutofillSource{AutofillRegressionAnchor, AutofillRequiredReading, AutofillReferences}
	}
	results := make([]AutofillResult, 0, len(sources))
	for _, src := range sources {
		results = append(results, s.runAutofill(ctx, &sess, src))
	}
	sess.CurrentSectionKey = firstUnfilledMandatory(sess.Sections)
	sess.UpdatedAt = s.now()
	if err := s.store.Save(ctx, sess); err != nil {
		return Session{}, nil, err
	}
	return sess, results, nil
}

// runAutofill runs one source against the session in place. It NEVER fabricates a
// fill: a nil seam or an error leaves the section untouched and returns
// Degraded=true with the honest reason.
func (s *service) runAutofill(ctx context.Context, sess *Session, src AutofillSource) AutofillResult {
	var (
		key     SectionKey
		content string
		err     error
	)
	switch src {
	case AutofillRegressionAnchor:
		key = SectionRegressionAnchor
		if s.anchor == nil {
			return degraded(src, key, "git-control-tower unavailable")
		}
		content, err = s.anchor.Anchor(ctx, sess.Title, sess.Slug)
	case AutofillRequiredReading:
		key = SectionRequiredReading
		if s.reading == nil {
			return degraded(src, key, "prompt-manager unavailable")
		}
		content, err = s.reading.RequiredReading(ctx, sess.Title)
	case AutofillReferences:
		key = SectionReferences
		if s.references == nil {
			return degraded(src, key, "code-facts unavailable")
		}
		content, err = s.references.References(ctx, sess.Title, contentOf(sess.Sections, SectionScope))
	default:
		return AutofillResult{Source: src, Degraded: true, Detail: "unknown autofill source"}
	}
	if err != nil {
		return degraded(src, key, err.Error())
	}
	if strings.TrimSpace(content) == "" {
		return degraded(src, key, "source returned no content")
	}
	idx := indexOf(sess.Sections, key)
	if idx < 0 {
		return degraded(src, key, "section not present in this session")
	}
	sess.Sections[idx].Content = content
	sess.Sections[idx].Filled = true
	sess.Sections[idx].Autofilled = true
	return AutofillResult{Source: src, SectionKey: key, Filled: true, Detail: "autofilled"}
}

func (s *service) Finalize(ctx context.Context, sessionID string) (internalplans.Plan, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return internalplans.Plan{}, err
	}
	if violations := structureViolations(sess.Sections); len(violations) > 0 {
		return internalplans.Plan{}, ErrStructureGate{Violations: violations}
	}
	plan, err := s.writer.CreatePlan(ctx, sessionToPlan(sess))
	if err != nil {
		return internalplans.Plan{}, err
	}
	sess.Finalized = true
	sess.PlanID = plan.ID
	sess.UpdatedAt = s.now()
	if err := s.store.Save(ctx, sess); err != nil {
		return internalplans.Plan{}, err
	}
	return plan, nil
}

func (s *service) load(ctx context.Context, sessionID string) (Session, error) {
	sess, ok, err := s.store.Get(ctx, strings.TrimSpace(sessionID))
	if err != nil {
		return Session{}, err
	}
	if !ok {
		return Session{}, ErrSessionNotFound{ID: sessionID}
	}
	return sess, nil
}

func (s *service) now() string { return s.clock.Now().UTC().Format(sessionTimeFormat) }

// --- pure helpers (no I/O) ---

// structureViolations is the structure-validation gate (PM-AUTHOR-002): every
// mandatory section must be non-empty, and the regression-anchor section must not
// be empty (it is a distinct violation even when not otherwise mandatory).
func structureViolations(sections []Section) []StructureViolation {
	var out []StructureViolation
	for _, sec := range sections {
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

// violationsForSection returns the gate violations specific to one submitted
// section (empty when it passes). A mandatory or regression-anchor section with
// empty content fails.
func violationsForSection(sec Section) []StructureViolation {
	var out []StructureViolation
	empty := strings.TrimSpace(sec.Content) == ""
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

// sessionToPlan maps a finalized session's sections into the structured plans
// model. The prose sections map directly to the matching plan fields; the
// references and phases sections carry authored markup (the same [CODE:]/[REQ:]/
// [DOC:] locators and `### Phase N — Title` headings the plans renderer emits),
// so they are parsed through the plans-domain markdown parser — the one SSOT for
// that grammar — by assembling a minimal markdown view and re-reading the
// references/phases it recovers. The prose fields are taken verbatim from the
// sections (not re-extracted) so authored content is never lossily reshaped. The
// regression anchor is carried as captured prose; the validation domain derives
// the anchor's commands from the plan's references later.
func sessionToPlan(sess Session) internalplans.Plan {
	parsed := parseReferencesAndPhases(sess)
	p := internalplans.Plan{
		Title:            sess.Title,
		Slug:             sess.Slug,
		Purpose:          contentOf(sess.Sections, SectionPurpose),
		Scope:            contentOf(sess.Sections, SectionScope),
		Constraints:      contentOf(sess.Sections, SectionConstraints),
		NonGoals:         contentOf(sess.Sections, SectionNonGoals),
		DefinitionOfDone: contentOf(sess.Sections, SectionDefinitionOfDone),
		References:       parsed.References,
		Phases:           parsed.Phases,
	}
	if anchor := strings.TrimSpace(contentOf(sess.Sections, SectionRegressionAnchor)); anchor != "" {
		// The regression-anchor section content is the authored/auto-filled
		// "before" capture; carry it forward as captured prose. Commands are
		// derived later by the validation domain from the plan's references.
		p.RegressionAnchor = internalplans.RegressionAnchor{
			Strategy:     "captured",
			BaselineName: anchor,
		}
	}
	return p
}

// parseReferencesAndPhases recovers the structured references[] and phases[] from
// the authored references/phases section markup via the plans-domain parser (the
// SSOT for the [CODE:]/[REQ:]/[DOC:] + `### Phase N — Title` grammar). It feeds a
// minimal markdown view — a synthetic title (so the parser accepts it) plus the
// references and phases sections — and returns only the recovered structured
// lists; the prose fields are taken verbatim by the caller. A parse failure
// degrades to empty lists rather than failing finalize.
func parseReferencesAndPhases(sess Session) internalplans.Plan {
	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(sess.Title)
	b.WriteString("\n\n")
	if refs := strings.TrimSpace(contentOf(sess.Sections, SectionReferences)); refs != "" {
		b.WriteString("## References\n\n")
		b.WriteString(refs)
		b.WriteString("\n\n")
	}
	if phases := strings.TrimSpace(contentOf(sess.Sections, SectionPhases)); phases != "" {
		b.WriteString("## Phases\n\n")
		b.WriteString(phases)
		b.WriteString("\n")
	}
	parsed, err := internalplans.ParsePlanMarkdown(b.String())
	if err != nil {
		return internalplans.Plan{}
	}
	return parsed
}
