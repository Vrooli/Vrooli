package authoring

import (
	"context"
	"fmt"
	"strings"

	"github.com/vrooli/api-core/markedrefs"

	"plan-manager/internal/clock"
	planmodel "plan-manager/internal/planmodel"

	"github.com/google/uuid"
)

// Service is the authoring application surface — the guided composer wizard.
type Service interface {
	StartSession(ctx context.Context, title, slug, templateID string) (Session, GuidedStep, error)
	GetSection(ctx context.Context, sessionID string, key SectionKey) (Section, GuidedStep, error)
	SubmitSection(ctx context.Context, sessionID string, key SectionKey, content string) (Session, []StructureViolation, GuidedStep, error)
	Next(ctx context.Context, sessionID string) (Section, GuidedStep, bool, error)
	ValidateStructure(ctx context.Context, sessionID string) (bool, []StructureViolation, GuidedStep, error)
	Autofill(ctx context.Context, sessionID string, sources []AutofillSource) (Session, []AutofillResult, GuidedStep, error)
	SubmitRelevantContextItem(ctx context.Context, sessionID, phaseID string, item planmodel.RelevantContextItem) (Session, planmodel.RelevantContextItem, []StructureViolation, GuidedStep, error)
	ListRelevantContext(ctx context.Context, sessionID, phaseID string) ([]planmodel.RelevantContextItem, GuidedStep, error)
	DiscoverContextCandidates(ctx context.Context, sessionID string, concepts []string, complexity string) (Session, []ContextCandidate, GuidedStep, error)
	AcceptContextCandidate(ctx context.Context, sessionID, candidateID, phaseID string) (Session, ContextCandidate, planmodel.RelevantContextItem, []StructureViolation, GuidedStep, error)
	RejectContextCandidate(ctx context.Context, sessionID, candidateID, reason string) (Session, ContextCandidate, GuidedStep, error)
	AddPhase(ctx context.Context, sessionID string, title, intent string) (Session, PhaseDraft, []StructureViolation, GuidedStep, error)
	GetPhase(ctx context.Context, sessionID, phaseID string) (PhaseDraft, GuidedStep, error)
	SubmitPhaseField(ctx context.Context, sessionID, phaseID string, field PhaseField, content string) (Session, []StructureViolation, GuidedStep, error)
	NextPhase(ctx context.Context, sessionID string) (PhaseDraft, GuidedStep, bool, error)
	Finalize(ctx context.Context, sessionID string) (planmodel.Plan, GuidedStep, error)
}

type service struct {
	store        SessionStore
	writer       PlanWriter
	anchor       AnchorAutofiller
	reading      RequiredReadingSource
	references   ReferenceExtractor
	context      ContextDiscoverer
	commands     CommandReferenceValidator
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
	Context        ContextDiscoverer
	Commands       CommandReferenceValidator
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
		context:      d.Context,
		commands:     d.Commands,
		templateSeed: d.TemplateSeeder,
		clock:        clk,
	}
}

var _ Service = (*service)(nil)

func (s *service) StartSession(ctx context.Context, title, slug, templateID string) (Session, GuidedStep, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Session{}, GuidedStep{}, ErrInvalidSession{Reason: "title is required"}
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
		return Session{}, GuidedStep{}, err
	}
	return sess, stepForSession(sess), nil
}

func (s *service) GetSection(ctx context.Context, sessionID string, key SectionKey) (Section, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Section{}, GuidedStep{}, err
	}
	idx := indexOf(sess.Sections, key)
	if idx < 0 {
		return Section{}, GuidedStep{}, ErrSectionNotFound{SessionID: sessionID, SectionKey: string(key)}
	}
	return sess.Sections[idx], stepForSection(sess, sess.Sections[idx]), nil
}

func (s *service) SubmitSection(ctx context.Context, sessionID string, key SectionKey, content string) (Session, []StructureViolation, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, nil, GuidedStep{}, err
	}
	idx := indexOf(sess.Sections, key)
	if idx < 0 {
		return Session{}, nil, GuidedStep{}, ErrSectionNotFound{SessionID: sessionID, SectionKey: string(key)}
	}
	sess.Sections[idx].Content = content
	sess.Sections[idx].Filled = strings.TrimSpace(content) != ""
	sess.Sections[idx].Autofilled = false // an author submission supersedes any autofill
	violations := violationsForSection(sess.Sections[idx])
	violations = append(violations, s.commandViolationsForSection(ctx, sess.Sections[idx])...)
	sess.CurrentSectionKey = firstUnfilledMandatory(sess.Sections)
	sess.UpdatedAt = s.now()
	if err := s.store.Save(ctx, sess); err != nil {
		return Session{}, nil, GuidedStep{}, err
	}
	return sess, violations, stepForCurrentSessionState(sess), nil
}

func (s *service) Next(ctx context.Context, sessionID string) (Section, GuidedStep, bool, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Section{}, GuidedStep{}, false, err
	}
	key := firstUnfilledMandatory(sess.Sections)
	if key == "" {
		return Section{}, stepForReview(sess), true, nil
	}
	idx := indexOf(sess.Sections, key)
	return sess.Sections[idx], stepForSection(sess, sess.Sections[idx]), false, nil
}

func (s *service) ValidateStructure(ctx context.Context, sessionID string) (bool, []StructureViolation, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return false, nil, GuidedStep{}, err
	}
	violations := sessionViolations(sess)
	violations = append(violations, s.commandViolationsForSections(ctx, sess.Sections)...)
	valid := len(violations) == 0
	return valid, violations, stepForValidation(sess, valid, violations), nil
}

func (s *service) Autofill(ctx context.Context, sessionID string, sources []AutofillSource) (Session, []AutofillResult, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, nil, GuidedStep{}, err
	}
	if len(sources) == 0 {
		sources = []AutofillSource{AutofillRegressionAnchor, AutofillReferences}
	}
	results := make([]AutofillResult, 0, len(sources))
	for _, src := range sources {
		results = append(results, s.runAutofill(ctx, &sess, src))
	}
	sess.CurrentSectionKey = firstUnfilledMandatory(sess.Sections)
	sess.UpdatedAt = s.now()
	if err := s.store.Save(ctx, sess); err != nil {
		return Session{}, nil, GuidedStep{}, err
	}
	return sess, results, stepForCurrentSessionState(sess), nil
}

func (s *service) SubmitRelevantContextItem(ctx context.Context, sessionID, phaseID string, item planmodel.RelevantContextItem) (Session, planmodel.RelevantContextItem, []StructureViolation, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, err
	}
	item = normalizeContextItem(item, phaseID)
	var violations []StructureViolation
	if phaseID != "" {
		idx := indexOfDraft(sess.PhaseDrafts, phaseID)
		if idx < 0 {
			return Session{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + phaseID}
		}
		item.Scope = planmodel.RelevantContextScopePhase
		item.PhaseID = sess.PhaseDrafts[idx].ID
		violations = contextItemViolations(item)
		if len(violations) == 0 {
			sess.PhaseDrafts[idx].RelevantContext = append(sess.PhaseDrafts[idx].RelevantContext, item)
		}
		sess.CurrentPhaseID = nextIncompletePhaseID(sess.PhaseDrafts)
		sess = syncPhaseSection(sess)
	} else {
		item.Scope = planmodel.RelevantContextScopeGlobal
		item.PhaseID = ""
		violations = contextItemViolations(item)
		if len(violations) == 0 {
			sess.RelevantContext = append(sess.RelevantContext, item)
			sess = syncContextSection(sess)
		}
	}
	sess.UpdatedAt = s.now()
	if len(violations) == 0 {
		if err := s.store.Save(ctx, sess); err != nil {
			return Session{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, err
		}
	}
	return sess, item, violations, stepForCurrentSessionState(sess), nil
}

func (s *service) ListRelevantContext(ctx context.Context, sessionID, phaseID string) ([]planmodel.RelevantContextItem, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return nil, GuidedStep{}, err
	}
	if phaseID == "" {
		return append([]planmodel.RelevantContextItem(nil), sess.RelevantContext...), stepForCurrentSessionState(sess), nil
	}
	phase, ok := findDraft(sess.PhaseDrafts, phaseID)
	if !ok {
		return nil, GuidedStep{}, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + phaseID}
	}
	return append([]planmodel.RelevantContextItem(nil), phase.RelevantContext...), stepForPhase(sess, phase), nil
}

func (s *service) DiscoverContextCandidates(ctx context.Context, sessionID string, concepts []string, complexity string) (Session, []ContextCandidate, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, nil, GuidedStep{}, err
	}
	var candidates []ContextCandidate
	if s.context == nil {
		candidates = degradedContextCandidates(sess.Title, concepts, "context discovery unavailable")
	} else {
		candidates, err = s.context.DiscoverContext(ctx, sess.Title, concepts, complexity)
		if err != nil {
			candidates = degradedContextCandidates(sess.Title, concepts, err.Error())
		}
	}
	for i := range candidates {
		candidates[i] = normalizeContextCandidate(candidates[i])
	}
	sess.ContextCandidates = append(sess.ContextCandidates, candidates...)
	sess.UpdatedAt = s.now()
	if err := s.store.Save(ctx, sess); err != nil {
		return Session{}, nil, GuidedStep{}, err
	}
	return sess, candidates, stepForContextDiscovery(sess), nil
}

func (s *service) AcceptContextCandidate(ctx context.Context, sessionID, candidateID, phaseID string) (Session, ContextCandidate, planmodel.RelevantContextItem, []StructureViolation, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, ContextCandidate{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, err
	}
	idx := indexOfCandidate(sess.ContextCandidates, candidateID)
	if idx < 0 {
		return Session{}, ContextCandidate{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "context candidate not found: " + candidateID}
	}
	candidate := sess.ContextCandidates[idx]
	if candidate.Status == ContextCandidateRejected {
		return Session{}, ContextCandidate{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "context candidate was rejected: " + candidateID}
	}
	item := normalizeContextItem(candidate.Item, phaseID)
	var violations []StructureViolation
	if phaseID != "" {
		phaseIdx := indexOfDraft(sess.PhaseDrafts, phaseID)
		if phaseIdx < 0 {
			return Session{}, ContextCandidate{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + phaseID}
		}
		item.Scope = planmodel.RelevantContextScopePhase
		item.PhaseID = sess.PhaseDrafts[phaseIdx].ID
		item.RepeatPolicy = defaultRepeatForScope(item.Scope, item.RepeatPolicy)
		violations = contextItemViolations(item)
		if len(violations) == 0 {
			sess.PhaseDrafts[phaseIdx].RelevantContext = append(sess.PhaseDrafts[phaseIdx].RelevantContext, item)
			sess = syncPhaseSection(sess)
		}
	} else {
		item.Scope = planmodel.RelevantContextScopeGlobal
		item.PhaseID = ""
		item.RepeatPolicy = defaultRepeatForScope(item.Scope, item.RepeatPolicy)
		violations = contextItemViolations(item)
		if len(violations) == 0 {
			sess.RelevantContext = append(sess.RelevantContext, item)
			sess = syncContextSection(sess)
		}
	}
	if len(violations) > 0 {
		return sess, candidate, item, violations, stepForContextDiscovery(sess), nil
	}
	candidate.Status = ContextCandidateAccepted
	candidate.Item = item
	sess.ContextCandidates[idx] = candidate
	sess.UpdatedAt = s.now()
	if err := s.store.Save(ctx, sess); err != nil {
		return Session{}, ContextCandidate{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, err
	}
	return sess, candidate, item, nil, stepForCurrentSessionState(sess), nil
}

func (s *service) RejectContextCandidate(ctx context.Context, sessionID, candidateID, reason string) (Session, ContextCandidate, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, ContextCandidate{}, GuidedStep{}, err
	}
	idx := indexOfCandidate(sess.ContextCandidates, candidateID)
	if idx < 0 {
		return Session{}, ContextCandidate{}, GuidedStep{}, ErrInvalidSession{Reason: "context candidate not found: " + candidateID}
	}
	candidate := sess.ContextCandidates[idx]
	candidate.Status = ContextCandidateRejected
	candidate.RejectionReason = strings.TrimSpace(reason)
	sess.ContextCandidates[idx] = candidate
	sess.UpdatedAt = s.now()
	if err := s.store.Save(ctx, sess); err != nil {
		return Session{}, ContextCandidate{}, GuidedStep{}, err
	}
	return sess, candidate, stepForContextDiscovery(sess), nil
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

func (s *service) AddPhase(ctx context.Context, sessionID string, title, intent string) (Session, PhaseDraft, []StructureViolation, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, PhaseDraft{}, nil, GuidedStep{}, err
	}
	title = strings.TrimSpace(title)
	intent = strings.TrimSpace(intent)
	phase := PhaseDraft{
		ID:     uuid.NewString(),
		Order:  len(sess.PhaseDrafts) + 1,
		Title:  title,
		Intent: intent,
	}
	sess.PhaseDrafts = append(sess.PhaseDrafts, phase)
	if sess.CurrentPhaseID == "" {
		sess.CurrentPhaseID = phase.ID
	}
	sess = syncPhaseSection(sess)
	sess.UpdatedAt = s.now()
	violations := phaseViolations(phase)
	if err := s.store.Save(ctx, sess); err != nil {
		return Session{}, PhaseDraft{}, nil, GuidedStep{}, err
	}
	return sess, phase, violations, stepForPhase(sess, phase), nil
}

func (s *service) GetPhase(ctx context.Context, sessionID, phaseID string) (PhaseDraft, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return PhaseDraft{}, GuidedStep{}, err
	}
	phase, ok := findDraft(sess.PhaseDrafts, phaseID)
	if !ok {
		return PhaseDraft{}, GuidedStep{}, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + phaseID}
	}
	return phase, stepForPhase(sess, phase), nil
}

func (s *service) SubmitPhaseField(ctx context.Context, sessionID, phaseID string, field PhaseField, content string) (Session, []StructureViolation, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, nil, GuidedStep{}, err
	}
	idx := indexOfDraft(sess.PhaseDrafts, phaseID)
	if idx < 0 {
		return Session{}, nil, GuidedStep{}, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + phaseID}
	}
	if err := applyPhaseField(&sess.PhaseDrafts[idx], field, content); err != nil {
		return Session{}, nil, GuidedStep{}, err
	}
	sess.CurrentPhaseID = nextIncompletePhaseID(sess.PhaseDrafts)
	sess = syncPhaseSection(sess)
	sess.UpdatedAt = s.now()
	violations := phaseViolations(sess.PhaseDrafts[idx])
	if err := s.store.Save(ctx, sess); err != nil {
		return Session{}, nil, GuidedStep{}, err
	}
	return sess, violations, stepForNextPhaseState(sess, sess.PhaseDrafts[idx]), nil
}

func (s *service) NextPhase(ctx context.Context, sessionID string) (PhaseDraft, GuidedStep, bool, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return PhaseDraft{}, GuidedStep{}, false, err
	}
	id := nextIncompletePhaseID(sess.PhaseDrafts)
	if id == "" {
		return PhaseDraft{}, stepForReview(sess), true, nil
	}
	phase, ok := findDraft(sess.PhaseDrafts, id)
	if !ok {
		return PhaseDraft{}, GuidedStep{}, false, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + id}
	}
	return phase, stepForPhase(sess, phase), false, nil
}

func (s *service) Finalize(ctx context.Context, sessionID string) (planmodel.Plan, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return planmodel.Plan{}, GuidedStep{}, err
	}
	if violations := sessionViolations(sess); len(violations) > 0 {
		return planmodel.Plan{}, GuidedStep{}, ErrStructureGate{Violations: violations}
	}
	if violations := s.commandViolationsForSections(ctx, sess.Sections); len(violations) > 0 {
		return planmodel.Plan{}, GuidedStep{}, ErrStructureGate{Violations: violations}
	}
	draft, err := sessionToPlan(sess)
	if err != nil {
		return planmodel.Plan{}, GuidedStep{}, err
	}
	plan, err := s.writer.CreatePlan(ctx, draft)
	if err != nil {
		return planmodel.Plan{}, GuidedStep{}, err
	}
	sess.Finalized = true
	sess.PlanID = plan.ID
	sess.UpdatedAt = s.now()
	if err := s.store.Save(ctx, sess); err != nil {
		return planmodel.Plan{}, GuidedStep{}, err
	}
	return plan, stepForFinalizedPlan(sess, plan.ID, plan.Slug), nil
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

func stepForCurrentSessionState(sess Session) GuidedStep {
	if sess.CurrentSectionKey == "" {
		return stepForReview(sess)
	}
	if sec, ok := sectionByKey(sess.Sections, sess.CurrentSectionKey); ok {
		return stepForSection(sess, sec)
	}
	return stepForReview(sess)
}

func stepForNextPhaseState(sess Session, fallback PhaseDraft) GuidedStep {
	if id := nextIncompletePhaseID(sess.PhaseDrafts); id != "" {
		if phase, ok := findDraft(sess.PhaseDrafts, id); ok {
			return stepForPhase(sess, phase)
		}
	}
	return stepForPhase(sess, fallback)
}

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

func sessionViolations(sess Session) []StructureViolation {
	out := structureViolations(sess.Sections)
	refsContent := contentOf(sess.Sections, SectionReferences)
	if !hasReferencesOrNoCodeReason(refsContent) {
		out = append(out, StructureViolation{
			SectionKey: SectionReferences,
			Message:    "references must include at least one [CODE:], [REQ:], or [DOC:] reference, or a NO_CODE_REFS: reason",
		})
	}
	for _, phase := range sess.PhaseDrafts {
		for _, v := range phaseViolations(phase) {
			out = append(out, v)
		}
	}
	return out
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
	if strings.TrimSpace(phase.Acceptance) == "" {
		out = append(out, StructureViolation{SectionKey: SectionPhases, Message: prefix + " acceptance must not be empty"})
	}
	if len(phase.References) == 0 && strings.TrimSpace(phase.NoCodeRefsReason) == "" {
		out = append(out, StructureViolation{
			SectionKey: SectionPhases,
			Message:    prefix + " must include references or a no_code_refs_reason",
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

func hasReferenceMarker(content string) bool {
	upper := strings.ToUpper(content)
	return strings.Contains(upper, "[CODE:") ||
		strings.Contains(upper, "[REQ:") ||
		strings.Contains(upper, "[DOC:")
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

func splitLines(content string) []string {
	var out []string
	for _, line := range strings.Split(content, "\n") {
		if trimmed := strings.TrimSpace(strings.TrimPrefix(line, "-")); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func applyPhaseField(phase *PhaseDraft, field PhaseField, content string) error {
	content = strings.TrimSpace(content)
	switch field {
	case PhaseFieldTitle:
		phase.Title = content
	case PhaseFieldIntent:
		phase.Intent = content
	case PhaseFieldReferences:
		refs, err := parseReferencesContent(content)
		if err != nil {
			return ErrAuthoredMarkup{SectionKey: SectionPhases, Reason: err.Error()}
		}
		phase.References = refs
		if len(refs) > 0 {
			phase.NoCodeRefsReason = ""
		}
	case PhaseFieldRequiredReading:
		phase.RequiredReading = splitLines(content)
	case PhaseFieldReminders:
		phase.Reminders = splitLines(content)
	case PhaseFieldAcceptance:
		phase.Acceptance = content
	case PhaseFieldNoCodeRefsReason:
		phase.NoCodeRefsReason = content
		if content != "" {
			phase.References = nil
		}
	case PhaseFieldRelevantContext:
		phase.RelevantContext = append(phase.RelevantContext, contextItemsFromLines(content, planmodel.RelevantContextScopePhase, phase.ID)...)
	default:
		return ErrInvalidSession{Reason: "unknown phase field " + string(field)}
	}
	return nil
}

func syncPhaseSection(sess Session) Session {
	if len(sess.PhaseDrafts) == 0 {
		return sess
	}
	idx := indexOf(sess.Sections, SectionPhases)
	if idx < 0 {
		return sess
	}
	sess.Sections[idx].Content = renderPhaseDrafts(sess.PhaseDrafts)
	sess.Sections[idx].Filled = strings.TrimSpace(sess.Sections[idx].Content) != ""
	sess.Sections[idx].Autofilled = false
	sess.CurrentSectionKey = firstUnfilledMandatory(sess.Sections)
	return sess
}

func syncContextSection(sess Session) Session {
	idx := indexOf(sess.Sections, SectionRelevantContext)
	if idx < 0 {
		return sess
	}
	sess.Sections[idx].Content = renderContextItems(sess.RelevantContext)
	sess.Sections[idx].Filled = strings.TrimSpace(sess.Sections[idx].Content) != ""
	sess.Sections[idx].Autofilled = false
	return sess
}

func renderPhaseDrafts(phases []PhaseDraft) string {
	var b strings.Builder
	for i, ph := range phases {
		order := ph.Order
		if order <= 0 {
			order = i + 1
		}
		fmt.Fprintf(&b, "### Phase %d — %s\n", order, ph.Title)
		if ph.Intent != "" {
			fmt.Fprintf(&b, "- Intent: %s\n", ph.Intent)
		}
		if ph.Acceptance != "" {
			fmt.Fprintf(&b, "- Acceptance: %s\n", ph.Acceptance)
		}
		context := append([]planmodel.RelevantContextItem(nil), ph.RelevantContext...)
		context = append(context, contextItemsFromRequiredReading(ph.RequiredReading, ph.ID)...)
		if len(context) > 0 {
			b.WriteString("- Relevant context:\n")
			for _, item := range context {
				fmt.Fprintf(&b, "  - %s\n", renderContextItemSummary(item))
			}
		}
		if len(ph.Reminders) > 0 {
			b.WriteString("- Reminders:\n")
			for _, item := range ph.Reminders {
				fmt.Fprintf(&b, "  - %s\n", item)
			}
		}
		if len(ph.References) > 0 {
			b.WriteString("- References:\n")
			for _, ref := range ph.References {
				fmt.Fprintf(&b, "  - [%s: %s]\n", referenceMarker(ref.Kind), ref.Target)
			}
		}
		if ph.NoCodeRefsReason != "" {
			fmt.Fprintf(&b, "- NO_CODE_REFS: %s\n", ph.NoCodeRefsReason)
		}
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func referenceMarker(k planmodel.ReferenceKind) string {
	switch k {
	case planmodel.ReferenceReq:
		return "REQ"
	case planmodel.ReferenceDoc:
		return "DOC"
	default:
		return "CODE"
	}
}

func findDraft(phases []PhaseDraft, id string) (PhaseDraft, bool) {
	if strings.TrimSpace(id) == "" && len(phases) > 0 {
		return phases[0], true
	}
	for _, ph := range phases {
		if ph.ID == id || fmt.Sprint(ph.Order) == id {
			return ph, true
		}
	}
	return PhaseDraft{}, false
}

func indexOfDraft(phases []PhaseDraft, id string) int {
	if strings.TrimSpace(id) == "" && len(phases) > 0 {
		return 0
	}
	for i, ph := range phases {
		if ph.ID == id || fmt.Sprint(ph.Order) == id {
			return i
		}
	}
	return -1
}

func nextIncompletePhaseID(phases []PhaseDraft) string {
	for _, ph := range phases {
		if len(phaseViolations(ph)) > 0 {
			return ph.ID
		}
	}
	return ""
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
func sessionToPlan(sess Session) (planmodel.Plan, error) {
	parsed, err := parseReferencesAndPhases(sess)
	if err != nil {
		return planmodel.Plan{}, err
	}
	p := planmodel.Plan{
		Title:            sess.Title,
		Slug:             sess.Slug,
		Purpose:          contentOf(sess.Sections, SectionPurpose),
		Scope:            contentOf(sess.Sections, SectionScope),
		Constraints:      contentOf(sess.Sections, SectionConstraints),
		NonGoals:         contentOf(sess.Sections, SectionNonGoals),
		DefinitionOfDone: contentOf(sess.Sections, SectionDefinitionOfDone),
		References:       parsed.References,
		Phases:           parsed.Phases,
		RelevantContext:  append([]planmodel.RelevantContextItem(nil), sess.RelevantContext...),
	}
	p.RelevantContext = append(p.RelevantContext, contextItemsFromLines(contentOf(sess.Sections, SectionRequiredReading), planmodel.RelevantContextScopeGlobal, "")...)
	if len(sess.PhaseDrafts) > 0 {
		p.Phases = phaseDraftsToPlanPhases(sess.PhaseDrafts)
	}
	if reason := noCodeRefsReason(contentOf(sess.Sections, SectionReferences)); reason != "" {
		p.Constraints = appendNoCodeRefsReason(p.Constraints, reason)
	}
	if anchor := strings.TrimSpace(contentOf(sess.Sections, SectionRegressionAnchor)); anchor != "" {
		// The regression-anchor section content is the authored/auto-filled
		// "before" capture; carry it forward as captured prose. Commands are
		// derived later by the validation domain from the plan's references.
		p.RegressionAnchor = planmodel.RegressionAnchor{
			Strategy:     "captured",
			BaselineName: anchor,
		}
	}
	return p, nil
}

func phaseDraftsToPlanPhases(drafts []PhaseDraft) []planmodel.Phase {
	out := make([]planmodel.Phase, 0, len(drafts))
	for i, draft := range drafts {
		order := draft.Order
		if order <= 0 {
			order = i + 1
		}
		phaseID := strings.TrimSpace(draft.ID)
		reminders := append([]string(nil), draft.Reminders...)
		if draft.NoCodeRefsReason != "" {
			reminders = append(reminders, "No connected code references: "+draft.NoCodeRefsReason)
		}
		relevantContext := append([]planmodel.RelevantContextItem(nil), draft.RelevantContext...)
		relevantContext = append(relevantContext, contextItemsFromRequiredReading(draft.RequiredReading, phaseID)...)
		for i := range relevantContext {
			relevantContext[i].Scope = planmodel.RelevantContextScopePhase
			relevantContext[i].PhaseID = phaseID
		}
		out = append(out, planmodel.Phase{
			ID:              phaseID,
			Order:           order,
			Title:           draft.Title,
			Intent:          draft.Intent,
			RequiredReading: append([]string(nil), draft.RequiredReading...),
			Reminders:       reminders,
			Acceptance:      draft.Acceptance,
			References:      append([]planmodel.Reference(nil), draft.References...),
			RelevantContext: relevantContext,
			Status:          planmodel.PhaseStatusTodo,
		})
	}
	return out
}

func normalizeContextItem(item planmodel.RelevantContextItem, phaseID string) planmodel.RelevantContextItem {
	item.ID = strings.TrimSpace(item.ID)
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	item.PhaseID = strings.TrimSpace(item.PhaseID)
	if phaseID != "" {
		item.PhaseID = strings.TrimSpace(phaseID)
	}
	item.Label = strings.TrimSpace(item.Label)
	item.Reason = strings.TrimSpace(item.Reason)
	item.Instruction = strings.TrimSpace(item.Instruction)
	item.Command = strings.TrimSpace(item.Command)
	item.Target = strings.TrimSpace(item.Target)
	if item.Scope == "" {
		if phaseID != "" || item.PhaseID != "" {
			item.Scope = planmodel.RelevantContextScopePhase
		} else {
			item.Scope = planmodel.RelevantContextScopeGlobal
		}
	}
	if item.RepeatPolicy == "" {
		if item.Scope == planmodel.RelevantContextScopePhase {
			item.RepeatPolicy = planmodel.RelevantContextPhaseEntry
		} else {
			item.RepeatPolicy = planmodel.RelevantContextOncePerExecution
		}
	}
	if item.Source == "" {
		item.Source = planmodel.RelevantContextSourceAuthored
	}
	if item.Status == "" {
		item.Status = planmodel.RelevantContextStatusReady
	}
	if item.Label == "" {
		item.Label = firstNonEmpty(item.Target, item.Command, item.Instruction, string(item.Kind))
	}
	return item
}

func normalizeContextCandidate(candidate ContextCandidate) ContextCandidate {
	candidate.ID = strings.TrimSpace(candidate.ID)
	if candidate.ID == "" {
		candidate.ID = uuid.NewString()
	}
	candidate.Concept = strings.TrimSpace(candidate.Concept)
	candidate.Source = strings.TrimSpace(candidate.Source)
	candidate.Detail = strings.TrimSpace(candidate.Detail)
	candidate.RejectionReason = strings.TrimSpace(candidate.RejectionReason)
	if candidate.Status == "" {
		candidate.Status = ContextCandidatePending
	}
	candidate.Item = normalizeContextItem(candidate.Item, "")
	if candidate.Degraded && candidate.Item.Status == planmodel.RelevantContextStatusReady {
		candidate.Item.Status = planmodel.RelevantContextStatusDegraded
	}
	return candidate
}

func degradedContextCandidates(title string, concepts []string, detail string) []ContextCandidate {
	if len(concepts) == 0 {
		concepts = []string{title}
	}
	out := make([]ContextCandidate, 0, len(concepts))
	for _, concept := range concepts {
		concept = strings.TrimSpace(concept)
		if concept == "" {
			continue
		}
		item := planmodel.RelevantContextItem{
			ID:           uuid.NewString(),
			Kind:         planmodel.RelevantContextNote,
			Scope:        planmodel.RelevantContextScopeGlobal,
			Label:        "Context discovery degraded: " + concept,
			Reason:       "Automated relevant-context discovery could not run.",
			Instruction:  "Manually run prompt-manager/search-hub/cli-health discovery for this concept before accepting setup context.",
			Required:     false,
			RepeatPolicy: planmodel.RelevantContextAsNeeded,
			Source:       planmodel.RelevantContextSourceDiscovered,
			Status:       planmodel.RelevantContextStatusDegraded,
		}
		out = append(out, ContextCandidate{
			ID:       uuid.NewString(),
			Item:     item,
			Concept:  concept,
			Source:   "context-discovery",
			Degraded: true,
			Detail:   strings.TrimSpace(detail),
			Status:   ContextCandidatePending,
		})
	}
	return out
}

func defaultRepeatForScope(scope planmodel.RelevantContextScope, current planmodel.RelevantContextRepeatPolicy) planmodel.RelevantContextRepeatPolicy {
	if current != "" && !(scope == planmodel.RelevantContextScopePhase && current == planmodel.RelevantContextOncePerExecution) {
		return current
	}
	if scope == planmodel.RelevantContextScopePhase {
		return planmodel.RelevantContextPhaseEntry
	}
	return planmodel.RelevantContextOncePerExecution
}

func indexOfCandidate(candidates []ContextCandidate, id string) int {
	id = strings.TrimSpace(id)
	for i := range candidates {
		if candidates[i].ID == id {
			return i
		}
	}
	return -1
}

func contextItemViolations(item planmodel.RelevantContextItem) []StructureViolation {
	var out []StructureViolation
	add := func(msg string) {
		out = append(out, StructureViolation{SectionKey: SectionRelevantContext, Message: msg})
	}
	if item.Kind == "" {
		add("relevant context kind is required")
	}
	if item.Scope == planmodel.RelevantContextScopePhase && strings.TrimSpace(item.PhaseID) == "" {
		add("phase-scoped relevant context requires a phase id")
	}
	if item.Required && item.RepeatPolicy == "" {
		add("required relevant context requires a repeat policy")
	}
	switch item.Kind {
	case planmodel.RelevantContextCommand, planmodel.RelevantContextSearch:
		if strings.TrimSpace(item.Command) == "" && len(item.Argv) == 0 {
			add("command/search context requires command or argv")
		}
		if strings.TrimSpace(item.Instruction) == "" {
			add("command/search context requires an instruction")
		}
		if strings.TrimSpace(item.Reason) == "" {
			add("command/search context requires a reason")
		}
	case planmodel.RelevantContextSkill, planmodel.RelevantContextDoc, planmodel.RelevantContextCodeRef, planmodel.RelevantContextReqRef:
		if strings.TrimSpace(item.Target) == "" && strings.TrimSpace(item.Command) == "" && len(item.Argv) == 0 {
			add("reference context requires a target, command, or argv")
		}
	case planmodel.RelevantContextNote:
		if strings.TrimSpace(item.Instruction) == "" && strings.TrimSpace(item.Reason) == "" {
			add("note context requires an instruction or reason")
		}
	}
	return out
}

func contextItemsFromLines(content string, scope planmodel.RelevantContextScope, phaseID string) []planmodel.RelevantContextItem {
	var out []planmodel.RelevantContextItem
	for _, line := range splitLines(content) {
		item := migratedContextItem(line, scope, phaseID)
		if item.Label != "" || item.Target != "" || item.Command != "" || item.Instruction != "" {
			out = append(out, item)
		}
	}
	return out
}

func contextItemsFromRequiredReading(lines []string, phaseID string) []planmodel.RelevantContextItem {
	out := make([]planmodel.RelevantContextItem, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		out = append(out, migratedContextItem(line, planmodel.RelevantContextScopePhase, phaseID))
	}
	return out
}

func migratedContextItem(line string, scope planmodel.RelevantContextScope, phaseID string) planmodel.RelevantContextItem {
	line = strings.TrimSpace(strings.TrimPrefix(line, "-"))
	item := planmodel.RelevantContextItem{
		ID:           uuid.NewString(),
		Scope:        scope,
		PhaseID:      phaseID,
		Label:        line,
		Reason:       "Migrated from required-reading authoring input.",
		Instruction:  "Load or inspect this context before implementation work.",
		Target:       line,
		Required:     true,
		RepeatPolicy: planmodel.RelevantContextPhaseEntry,
		Source:       planmodel.RelevantContextSourceMigrated,
		Status:       planmodel.RelevantContextStatusReady,
	}
	if scope == planmodel.RelevantContextScopeGlobal {
		item.RepeatPolicy = planmodel.RelevantContextOncePerExecution
	}
	lower := strings.ToLower(line)
	switch {
	case strings.HasPrefix(lower, "prompt-manager skill read "):
		item.Kind = planmodel.RelevantContextSkill
		item.Command = line
		item.Argv = strings.Fields(line)
		item.Target = strings.TrimSpace(strings.TrimPrefix(line, "prompt-manager skill read "))
		item.Instruction = "Load this internal skill before implementation."
	case strings.HasPrefix(lower, "search-hub "):
		item.Kind = planmodel.RelevantContextSearch
		item.Command = line
		item.Argv = strings.Fields(line)
		item.Instruction = "Run this discovery search before implementation."
	case strings.HasPrefix(lower, "cli:"):
		item.Kind = planmodel.RelevantContextCommand
		item.Command = strings.TrimSpace(line[len("cli:"):])
		item.Argv = strings.Fields(item.Command)
		item.Target = ""
		item.Instruction = "Run this setup command before implementation."
	case strings.Contains(lower, "docs/") || strings.HasSuffix(lower, ".md"):
		item.Kind = planmodel.RelevantContextDoc
	default:
		item.Kind = planmodel.RelevantContextNote
		item.Instruction = line
		item.Target = ""
	}
	return item
}

func renderContextItems(items []planmodel.RelevantContextItem) string {
	var b strings.Builder
	for _, item := range items {
		fmt.Fprintf(&b, "- %s\n", renderContextItemSummary(item))
	}
	return strings.TrimSpace(b.String())
}

func renderContextItemSummary(item planmodel.RelevantContextItem) string {
	label := firstNonEmpty(item.Label, item.Target, item.Command, item.Instruction, string(item.Kind))
	return fmt.Sprintf("%s: %s", item.Kind, label)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func appendNoCodeRefsReason(constraints, reason string) string {
	line := "No connected code references: " + reason
	if strings.TrimSpace(constraints) == "" {
		return line
	}
	return strings.TrimRight(constraints, "\n") + "\n" + line
}

// parseReferencesAndPhases recovers the structured references[] and phases[] from
// the authored references/phases section markup via the plans-domain parser (the
// SSOT for the [CODE:]/[REQ:]/[DOC:] + `### Phase N — Title` grammar). It feeds a
// minimal markdown view — a synthetic title (so the parser accepts it) plus the
// references and phases sections — and returns only the recovered structured
// lists; the prose fields are taken verbatim by the caller. Because these
// sections are machine-readable, non-empty markup that cannot be parsed is a
// typed authoring error, never an empty-list degradation.
func parseReferencesAndPhases(sess Session) (planmodel.Plan, error) {
	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(sess.Title)
	b.WriteString("\n\n")
	refsContent := strings.TrimSpace(contentOf(sess.Sections, SectionReferences))
	phasesContent := strings.TrimSpace(contentOf(sess.Sections, SectionPhases))
	refsOnlyExplainsNoCode := refsContent != "" && noCodeRefsReason(refsContent) != "" && !hasReferenceMarker(refsContent)
	if refsContent != "" && !refsOnlyExplainsNoCode {
		b.WriteString("## References\n\n")
		b.WriteString(refsContent)
		b.WriteString("\n\n")
	}
	if phasesContent != "" {
		b.WriteString("## Phases\n\n")
		b.WriteString(phasesContent)
		b.WriteString("\n")
	}
	parsed, err := planmodel.ParsePlanMarkdown(b.String())
	if err != nil {
		return planmodel.Plan{}, ErrAuthoredMarkup{SectionKey: markupSectionForError(refsContent, phasesContent, err.Error()), Reason: err.Error()}
	}
	if refsContent != "" && !refsOnlyExplainsNoCode && len(parsed.References) == 0 {
		return planmodel.Plan{}, ErrAuthoredMarkup{SectionKey: SectionReferences, Reason: "expected at least one [CODE:], [REQ:], or [DOC:] reference"}
	}
	if phasesContent != "" && len(parsed.Phases) == 0 {
		return planmodel.Plan{}, ErrAuthoredMarkup{SectionKey: SectionPhases, Reason: "expected at least one '### Phase N - Title' heading"}
	}
	return parsed, nil
}

func markupSectionForError(refsContent, phasesContent, reason string) SectionKey {
	if phasesContent != "" && strings.Contains(reason, "phase") {
		return SectionPhases
	}
	if refsContent != "" && strings.Contains(reason, "reference") {
		return SectionReferences
	}
	if phasesContent != "" {
		return SectionPhases
	}
	return SectionPhases
}
