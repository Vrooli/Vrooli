package authoring

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/vrooli/api-core/markedrefs"

	"plan-manager/internal/clock"
	planmodel "plan-manager/internal/planmodel"

	"github.com/google/uuid"
)

// Service is the authoring application surface — the guided composer wizard.
type Service interface {
	StartSession(ctx context.Context, title, slug, templateID string) (Session, GuidedStep, error)
	GetSession(ctx context.Context, sessionID string) (Session, GuidedStep, error)
	GetSection(ctx context.Context, sessionID string, key SectionKey) (Section, GuidedStep, error)
	SubmitSection(ctx context.Context, sessionID string, key SectionKey, content string) (Session, []StructureViolation, GuidedStep, error)
	Next(ctx context.Context, sessionID string) (Section, GuidedStep, bool, error)
	ContinueAuthoring(ctx context.Context, sessionID string) (Session, Section, PhaseDraft, bool, []StructureViolation, GuidedStep, error)
	ValidateStructure(ctx context.Context, sessionID string) (bool, []StructureViolation, GuidedStep, error)
	Autofill(ctx context.Context, sessionID string, sources []AutofillSource) (Session, []AutofillResult, GuidedStep, error)
	SubmitRelevantContextItem(ctx context.Context, sessionID, phaseID string, item planmodel.RelevantContextItem) (Session, planmodel.RelevantContextItem, []StructureViolation, GuidedStep, error)
	ListRelevantContext(ctx context.Context, sessionID, phaseID string) ([]planmodel.RelevantContextItem, GuidedStep, error)
	UpdateRelevantContextItem(ctx context.Context, sessionID, phaseID, itemID string, item planmodel.RelevantContextItem) (Session, planmodel.RelevantContextItem, []StructureViolation, GuidedStep, error)
	RemoveRelevantContextItem(ctx context.Context, sessionID, phaseID, itemID string) (Session, []StructureViolation, GuidedStep, error)
	DiscoverContextCandidates(ctx context.Context, sessionID string, concepts []string, complexity string) (Session, []ContextCandidate, GuidedStep, error)
	AcceptContextCandidate(ctx context.Context, sessionID, candidateID, phaseID string) (Session, ContextCandidate, planmodel.RelevantContextItem, []StructureViolation, GuidedStep, error)
	RejectContextCandidate(ctx context.Context, sessionID, candidateID, reason string) (Session, ContextCandidate, GuidedStep, error)
	SuggestReferences(ctx context.Context, sessionID string) (Session, []ReferenceCandidate, GuidedStep, error)
	ListReferenceCandidates(ctx context.Context, sessionID string) ([]ReferenceCandidate, GuidedStep, error)
	AcceptReferenceCandidate(ctx context.Context, sessionID, candidateID string, edit *planmodel.Reference) (Session, ReferenceCandidate, []StructureViolation, GuidedStep, error)
	RejectReferenceCandidate(ctx context.Context, sessionID, candidateID, reason string) (Session, ReferenceCandidate, GuidedStep, error)
	AddPhase(ctx context.Context, sessionID string, title, intent string) (Session, PhaseDraft, []StructureViolation, GuidedStep, error)
	MovePhase(ctx context.Context, sessionID, phaseID, beforePhaseID, afterPhaseID string) (Session, PhaseDraft, []StructureViolation, GuidedStep, error)
	GetPhase(ctx context.Context, sessionID, phaseID string) (PhaseDraft, GuidedStep, error)
	SubmitPhaseField(ctx context.Context, sessionID, phaseID string, field PhaseField, content string) (Session, []StructureViolation, GuidedStep, error)
	NextPhase(ctx context.Context, sessionID string) (PhaseDraft, GuidedStep, bool, error)
	PreviewPlan(ctx context.Context, sessionID string) (string, GuidedStep, error)
	Finalize(ctx context.Context, sessionID string) (planmodel.Plan, GuidedStep, error)
}

type service struct {
	store        SessionStore
	writer       PlanWriter
	reader       PlanReader
	anchor       AnchorIntentDeriver
	suggester    ReferenceSuggester
	context      ContextDiscoverer
	commands     CommandReferenceValidator
	templateSeed TemplateSeeder
	renderer     PlanRenderer
	posture      PosturePreparer
	clock        clock.Clock
	lockMu       sync.Mutex
	sessionLocks map[string]*sync.Mutex
}

// PosturePreparer stamps the derived work posture onto a draft plan before
// preview renders it, so the preview render matches the persisted render (which
// applies the SAME derivation on Create). A nil preparer leaves the draft's
// default posture (the wizard's review marker), which is why preview previously
// disagreed with finalize for brownfield scenarios.
type PosturePreparer interface {
	PreparePosture(ctx context.Context, p planmodel.Plan) planmodel.Plan
}

// Deps wires the authoring Service. Store + Writer are required (a nil store
// cannot persist sessions; a nil writer cannot finalize). The autofill seams are
// optional — a nil seam degrades that source honestly (the section is left for
// the author, never a false fill). TemplateSeeder is optional (nil => the default
// skeleton; a template id is then ignored).
type Deps struct {
	Store          SessionStore
	Writer         PlanWriter
	Reader         PlanReader
	Anchor         AnchorIntentDeriver
	Suggester      ReferenceSuggester
	Context        ContextDiscoverer
	Commands       CommandReferenceValidator
	TemplateSeeder TemplateSeeder
	Renderer       PlanRenderer
	Posture        PosturePreparer
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
		reader:       d.Reader,
		anchor:       d.Anchor,
		suggester:    d.Suggester,
		context:      d.Context,
		commands:     d.Commands,
		templateSeed: d.TemplateSeeder,
		renderer:     d.Renderer,
		posture:      d.Posture,
		clock:        clk,
		sessionLocks: map[string]*sync.Mutex{},
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
	prefillWorkPosture(sections)
	now := s.now()
	sessionSlug, err := s.uniqueSessionSlug(ctx, slug, title)
	if err != nil {
		return Session{}, GuidedStep{}, err
	}
	sess := Session{
		ID:                uuid.NewString(),
		Title:             title,
		Slug:              sessionSlug,
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

// GetSession is the explicit full-state read. Mutations return only focused
// progress/summary, so a UI/operator that needs the whole session graph asks for
// it deliberately here (read-after-write) rather than relying on a session echo.
func (s *service) GetSession(ctx context.Context, sessionID string) (Session, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, GuidedStep{}, err
	}
	return sess, stepForCurrentSessionState(sess), nil
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

func (s *service) ContinueAuthoring(ctx context.Context, sessionID string) (Session, Section, PhaseDraft, bool, []StructureViolation, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, Section{}, PhaseDraft{}, false, nil, GuidedStep{}, err
	}
	if sess.Finalized {
		return sess, Section{}, PhaseDraft{}, false, nil, onlyRecommendedAction(stepForFinalizedPlan(sess, sess.PlanID, sess.Slug)), nil
	}
	if key := firstUnfilledMandatory(sess.Sections); key != "" {
		idx := indexOf(sess.Sections, key)
		return sess, sess.Sections[idx], PhaseDraft{}, false, nil, onlyRecommendedAction(stepForSection(sess, sess.Sections[idx])), nil
	}
	// Global relevant-context checkpoint: the continue loop must NOT silently
	// bypass plan-wide setup context. It is resolved by accepting/adding at least
	// one global context item, or by explicitly recording a NO_CONTEXT reason.
	if !globalContextResolved(sess) {
		idx := indexOf(sess.Sections, SectionRelevantContext)
		sec := Section{Key: SectionRelevantContext, Label: "Relevant context"}
		if idx >= 0 {
			sec = sess.Sections[idx]
		}
		return sess, sec, PhaseDraft{}, false, nil, onlyRecommendedAction(stepForGlobalContextCheckpoint(sess)), nil
	}
	if id := nextIncompletePhaseID(sess.PhaseDrafts); id != "" {
		phase, ok := findDraft(sess.PhaseDrafts, id)
		if !ok {
			return Session{}, Section{}, PhaseDraft{}, false, nil, GuidedStep{}, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + id}
		}
		return sess, Section{}, phase, false, phaseViolations(phase), onlyRecommendedAction(stepForPhase(sess, phase)), nil
	}
	violations := sessionViolations(sess)
	violations = append(violations, s.commandViolationsForSections(ctx, sess.Sections)...)
	if len(violations) > 0 {
		return sess, Section{}, PhaseDraft{}, false, violations, onlyRecommendedAction(stepForValidation(sess, false, violations)), nil
	}
	return sess, Section{}, PhaseDraft{}, true, nil, onlyRecommendedAction(stepForReview(sess)), nil
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
		sources = []AutofillSource{AutofillRegressionAnchor}
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
		// A phase-scoped item repeats on phase entry, not once-per-execution —
		// apply the scope default here (mirrors AcceptContextCandidate) so an unset
		// or contradictory once_per_execution policy is corrected at submit time.
		item.RepeatPolicy = defaultRepeatForScope(item.Scope, item.RepeatPolicy)
		violations = contextItemViolations(item)
		if len(violations) == 0 {
			sess.PhaseDrafts[idx].RelevantContext = append(sess.PhaseDrafts[idx].RelevantContext, item)
		}
		sess.CurrentPhaseID = nextIncompletePhaseID(sess.PhaseDrafts)
		sess = syncPhaseSection(sess)
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

// UpdateRelevantContextItem replaces one accepted context item in place (by id)
// so a bad item discovered in preview is corrected without deleting the whole
// phase/session. Legal only before finalize. On a content violation the session
// is left unchanged (mirrors SubmitRelevantContextItem).
func (s *service) UpdateRelevantContextItem(ctx context.Context, sessionID, phaseID, itemID string, item planmodel.RelevantContextItem) (Session, planmodel.RelevantContextItem, []StructureViolation, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, err
	}
	if sess.Finalized {
		return Session{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "relevant context cannot be edited after finalize"}
	}
	itemID = strings.TrimSpace(itemID)
	if itemID == "" {
		return Session{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "item_id is required"}
	}
	item.ID = itemID
	item = normalizeContextItem(item, phaseID)
	if phaseID != "" {
		phaseIdx := indexOfDraft(sess.PhaseDrafts, phaseID)
		if phaseIdx < 0 {
			return Session{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + phaseID}
		}
		item.Scope = planmodel.RelevantContextScopePhase
		item.PhaseID = sess.PhaseDrafts[phaseIdx].ID
		item.RepeatPolicy = defaultRepeatForScope(item.Scope, item.RepeatPolicy)
		pos := indexOfContextItem(sess.PhaseDrafts[phaseIdx].RelevantContext, itemID)
		if pos < 0 {
			return Session{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "phase context item not found: " + itemID}
		}
		if violations := contextItemViolations(item); len(violations) > 0 {
			return sess, item, violations, stepForCurrentSessionState(sess), nil
		}
		sess.PhaseDrafts[phaseIdx].RelevantContext[pos] = item
		sess = syncPhaseSection(sess)
	} else {
		item.Scope = planmodel.RelevantContextScopeGlobal
		item.PhaseID = ""
		item.RepeatPolicy = defaultRepeatForScope(item.Scope, item.RepeatPolicy)
		pos := indexOfContextItem(sess.RelevantContext, itemID)
		if pos < 0 {
			return Session{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "global context item not found: " + itemID}
		}
		if violations := contextItemViolations(item); len(violations) > 0 {
			return sess, item, violations, stepForCurrentSessionState(sess), nil
		}
		sess.RelevantContext[pos] = item
		sess = syncContextSection(sess)
	}
	sess.UpdatedAt = s.now()
	if err := s.store.Save(ctx, sess); err != nil {
		return Session{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, err
	}
	return sess, item, nil, stepForCurrentSessionState(sess), nil
}

// RemoveRelevantContextItem deletes one accepted context item (by id) before
// finalize, recomputing structure violations so a resulting gate (e.g. a phase
// left with no context) is reported with its recovery step.
func (s *service) RemoveRelevantContextItem(ctx context.Context, sessionID, phaseID, itemID string) (Session, []StructureViolation, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, nil, GuidedStep{}, err
	}
	if sess.Finalized {
		return Session{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "relevant context cannot be edited after finalize"}
	}
	itemID = strings.TrimSpace(itemID)
	if itemID == "" {
		return Session{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "item_id is required"}
	}
	var violations []StructureViolation
	if phaseID != "" {
		phaseIdx := indexOfDraft(sess.PhaseDrafts, phaseID)
		if phaseIdx < 0 {
			return Session{}, nil, GuidedStep{}, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + phaseID}
		}
		pos := indexOfContextItem(sess.PhaseDrafts[phaseIdx].RelevantContext, itemID)
		if pos < 0 {
			return Session{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "phase context item not found: " + itemID}
		}
		sess.PhaseDrafts[phaseIdx].RelevantContext = removeContextItemAt(sess.PhaseDrafts[phaseIdx].RelevantContext, pos)
		sess.CurrentPhaseID = nextIncompletePhaseID(sess.PhaseDrafts)
		sess = syncPhaseSection(sess)
		violations = phaseViolations(sess.PhaseDrafts[phaseIdx])
	} else {
		pos := indexOfContextItem(sess.RelevantContext, itemID)
		if pos < 0 {
			return Session{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "global context item not found: " + itemID}
		}
		sess.RelevantContext = removeContextItemAt(sess.RelevantContext, pos)
		sess = syncContextSection(sess)
	}
	sess.UpdatedAt = s.now()
	if err := s.store.Save(ctx, sess); err != nil {
		return Session{}, nil, GuidedStep{}, err
	}
	step := stepForCurrentSessionState(sess)
	if phaseID == "" && !globalContextResolved(sess) {
		step = stepForGlobalContextCheckpoint(sess)
	}
	return sess, violations, step, nil
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
	)
	switch src {
	case AutofillRegressionAnchor:
		key = SectionRegressionAnchor
		if s.anchor == nil {
			return degraded(src, key, "anchor intent deriver unavailable")
		}
		boundary := planmodel.ParseBoundarySection(contentOf(sess.Sections, SectionAcceptanceBoundary))
		content = s.anchor.DeriveAnchorIntent(ctx, sess.Title, sess.Slug, boundary)
	default:
		return AutofillResult{Source: src, Degraded: true, Detail: "unknown autofill source"}
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

// SuggestReferences queries search-hub's Answer projection from the session's
// title + scope + technical approach and stores reviewable reference candidates
// (routed by locator shape) on the session. It NEVER writes the references
// section — only Accept finalizes a reviewed candidate. A nil seam / error /
// empty result degrades honestly to no candidates (the references step still
// offers manual entry and NO_CODE_REFS).
func (s *service) SuggestReferences(ctx context.Context, sessionID string) (Session, []ReferenceCandidate, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, nil, GuidedStep{}, err
	}
	query := referenceSuggestionQuery(sess)
	var candidates []ReferenceCandidate
	if s.suggester == nil {
		candidates = nil
	} else if found, suggestErr := s.suggester.Suggest(ctx, query); suggestErr == nil {
		candidates = found
	}
	for i := range candidates {
		candidates[i] = normalizeReferenceCandidate(candidates[i])
	}
	sess.ReferenceCandidates = append(sess.ReferenceCandidates, candidates...)
	sess.UpdatedAt = s.now()
	if err := s.store.Save(ctx, sess); err != nil {
		return Session{}, nil, GuidedStep{}, err
	}
	return sess, candidates, stepForReferenceCandidates(sess), nil
}

// ListReferenceCandidates returns the session's reference candidates without
// changing wizard position.
func (s *service) ListReferenceCandidates(ctx context.Context, sessionID string) ([]ReferenceCandidate, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return nil, GuidedStep{}, err
	}
	return append([]ReferenceCandidate(nil), sess.ReferenceCandidates...), stepForReferenceCandidates(sess), nil
}

// AcceptReferenceCandidate promotes one pending reference candidate into the
// references section (with an optional inline edit of the locator). The accepted
// locator is appended to the section so the references gate (which reads the
// section for [CODE:]/[DOC:]/[REQ:] locators) passes only on reviewed state. A
// kind/path mismatch is rejected before the locator enters the section.
func (s *service) AcceptReferenceCandidate(ctx context.Context, sessionID, candidateID string, edit *planmodel.Reference) (Session, ReferenceCandidate, []StructureViolation, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, ReferenceCandidate{}, nil, GuidedStep{}, err
	}
	idx := indexOfReferenceCandidate(sess.ReferenceCandidates, candidateID)
	if idx < 0 {
		return Session{}, ReferenceCandidate{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "reference candidate not found: " + candidateID}
	}
	candidate := sess.ReferenceCandidates[idx]
	if candidate.Status == ReferenceCandidateRejected {
		return Session{}, ReferenceCandidate{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "reference candidate was rejected: " + candidateID}
	}
	ref := candidate.Reference
	if edit != nil {
		if edit.Kind != "" {
			ref.Kind = edit.Kind
		}
		if strings.TrimSpace(edit.Target) != "" {
			ref.Target = strings.TrimSpace(edit.Target)
		}
		ref.Future = edit.Future
	}
	if ref.Kind == "" {
		ref.Kind = planmodel.ReferenceCode
	}
	if strings.TrimSpace(ref.Target) == "" {
		return sess, candidate, []StructureViolation{{SectionKey: SectionReferences, Message: "reference candidate has no target locator"}}, stepForReferenceCandidates(sess), nil
	}
	if msg := referenceKindMismatch(ref.Kind, ref.Target); msg != "" {
		return sess, candidate, []StructureViolation{{SectionKey: SectionReferences, Message: msg}}, stepForReferenceCandidates(sess), nil
	}
	candidate.Reference = ref
	candidate.Status = ReferenceCandidateAccepted
	sess.ReferenceCandidates[idx] = candidate
	appendAcceptedReference(&sess, ref)
	sess.CurrentSectionKey = firstUnfilledMandatory(sess.Sections)
	sess.UpdatedAt = s.now()
	if err := s.store.Save(ctx, sess); err != nil {
		return Session{}, ReferenceCandidate{}, nil, GuidedStep{}, err
	}
	return sess, candidate, nil, stepForCurrentSessionState(sess), nil
}

// RejectReferenceCandidate records why a suggested reference is not relevant. The
// rejected candidate stays as an authoring audit trail; it never enters the
// references section.
func (s *service) RejectReferenceCandidate(ctx context.Context, sessionID, candidateID, reason string) (Session, ReferenceCandidate, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, ReferenceCandidate{}, GuidedStep{}, err
	}
	idx := indexOfReferenceCandidate(sess.ReferenceCandidates, candidateID)
	if idx < 0 {
		return Session{}, ReferenceCandidate{}, GuidedStep{}, ErrInvalidSession{Reason: "reference candidate not found: " + candidateID}
	}
	candidate := sess.ReferenceCandidates[idx]
	candidate.Status = ReferenceCandidateRejected
	candidate.RejectionReason = strings.TrimSpace(reason)
	sess.ReferenceCandidates[idx] = candidate
	sess.UpdatedAt = s.now()
	if err := s.store.Save(ctx, sess); err != nil {
		return Session{}, ReferenceCandidate{}, GuidedStep{}, err
	}
	return sess, candidate, stepForReferenceCandidates(sess), nil
}

func (s *service) AddPhase(ctx context.Context, sessionID string, title, intent string) (Session, PhaseDraft, []StructureViolation, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, PhaseDraft{}, nil, GuidedStep{}, err
	}
	unlock := s.lockSession(sess.ID)
	defer unlock()
	sess, err = s.load(ctx, sess.ID)
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

func (s *service) MovePhase(ctx context.Context, sessionID, phaseID, beforePhaseID, afterPhaseID string) (Session, PhaseDraft, []StructureViolation, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, PhaseDraft{}, nil, GuidedStep{}, err
	}
	unlock := s.lockSession(sess.ID)
	defer unlock()
	sess, err = s.load(ctx, sess.ID)
	if err != nil {
		return Session{}, PhaseDraft{}, nil, GuidedStep{}, err
	}
	phaseID = strings.TrimSpace(phaseID)
	beforePhaseID = strings.TrimSpace(beforePhaseID)
	afterPhaseID = strings.TrimSpace(afterPhaseID)
	if phaseID == "" {
		return Session{}, PhaseDraft{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "phase id is required"}
	}
	if (beforePhaseID == "") == (afterPhaseID == "") {
		return Session{}, PhaseDraft{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "provide exactly one of before or after phase id"}
	}
	from := indexOfDraft(sess.PhaseDrafts, phaseID)
	if from < 0 {
		return Session{}, PhaseDraft{}, nil, GuidedStep{}, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + phaseID}
	}
	targetID := firstNonEmpty(beforePhaseID, afterPhaseID)
	target := indexOfDraft(sess.PhaseDrafts, targetID)
	if target < 0 {
		return Session{}, PhaseDraft{}, nil, GuidedStep{}, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + targetID}
	}
	if from == target {
		phase := sess.PhaseDrafts[from]
		return sess, phase, phaseViolations(phase), stepForPhase(sess, phase), nil
	}
	phase := sess.PhaseDrafts[from]
	remaining := append([]PhaseDraft{}, sess.PhaseDrafts[:from]...)
	remaining = append(remaining, sess.PhaseDrafts[from+1:]...)
	target = indexOfDraft(remaining, targetID)
	if target < 0 {
		return Session{}, PhaseDraft{}, nil, GuidedStep{}, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + targetID}
	}
	insertAt := target
	if afterPhaseID != "" {
		insertAt = target + 1
	}
	reordered := append([]PhaseDraft{}, remaining[:insertAt]...)
	reordered = append(reordered, phase)
	reordered = append(reordered, remaining[insertAt:]...)
	renumberPhaseDrafts(reordered)
	sess.PhaseDrafts = reordered
	sess.CurrentPhaseID = nextIncompletePhaseID(sess.PhaseDrafts)
	sess = syncPhaseSection(sess)
	sess.UpdatedAt = s.now()
	moved, _ := findDraft(sess.PhaseDrafts, phase.ID)
	violations := phaseViolations(moved)
	if err := s.store.Save(ctx, sess); err != nil {
		return Session{}, PhaseDraft{}, nil, GuidedStep{}, err
	}
	return sess, moved, violations, stepForPhase(sess, moved), nil
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
	unlock := s.lockSession(sess.ID)
	defer unlock()
	sess, err = s.load(ctx, sess.ID)
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

// PreviewPlan renders the in-progress session to its markdown review artifact
// WITHOUT persisting — the render-preview the wizard offers before finalize. It
// maps the session through the same sessionToPlan path Finalize uses, so the
// preview matches what will be saved (posture is filled in on save; the preview
// shows the default greenfield block). Malformed authored markup surfaces as a
// typed error so the agent fixes it before finalizing.
func (s *service) PreviewPlan(ctx context.Context, sessionID string) (string, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return "", GuidedStep{}, err
	}
	if s.renderer == nil {
		return "", GuidedStep{}, ErrInvalidSession{Reason: "render preview unavailable: no renderer configured"}
	}
	draft, err := sessionToPlan(sess)
	if err != nil {
		return "", GuidedStep{}, err
	}
	// Apply the same posture derivation finalize/Create uses so the preview render
	// agrees with the persisted render (greenfield default OR brownfield from
	// scenario maturity) instead of always showing the default greenfield block.
	if s.posture != nil {
		draft = s.posture.PreparePosture(ctx, draft)
	}
	return s.renderer.Render(draft), stepForReview(sess), nil
}

func (s *service) Finalize(ctx context.Context, sessionID string) (planmodel.Plan, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return planmodel.Plan{}, GuidedStep{}, err
	}
	unlock := s.lockSession(sess.ID)
	defer unlock()
	sess, err = s.load(ctx, sess.ID)
	if err != nil {
		return planmodel.Plan{}, GuidedStep{}, err
	}
	if sess.Finalized && strings.TrimSpace(sess.PlanID) != "" {
		plan, err := s.readFinalizedPlan(ctx, planmodel.Plan{ID: sess.PlanID})
		if err != nil {
			return planmodel.Plan{}, GuidedStep{}, err
		}
		return plan, stepForFinalizedPlan(sess, plan.ID, plan.Slug), nil
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
	verified, err := s.readFinalizedPlan(ctx, plan)
	if err != nil {
		return planmodel.Plan{}, GuidedStep{}, err
	}
	sess.Finalized = true
	sess.PlanID = verified.ID
	sess.UpdatedAt = s.now()
	if err := s.store.Save(ctx, sess); err != nil {
		return planmodel.Plan{}, GuidedStep{}, err
	}
	return verified, stepForFinalizedPlan(sess, verified.ID, verified.Slug), nil
}

func (s *service) readFinalizedPlan(ctx context.Context, fallback planmodel.Plan) (planmodel.Plan, error) {
	idOrSlug := fallback.ID
	if s.reader == nil {
		return fallback, nil
	}
	plan, err := s.reader.GetPlan(ctx, idOrSlug)
	if err != nil {
		return planmodel.Plan{}, ErrFinalizeReadback{PlanID: idOrSlug, Cause: err}
	}
	if strings.TrimSpace(plan.ID) == "" {
		return planmodel.Plan{}, ErrFinalizeReadback{PlanID: idOrSlug, Cause: fmt.Errorf("resolved plan has empty id")}
	}
	if _, err := s.reader.RenderPlan(ctx, plan.ID); err != nil {
		return planmodel.Plan{}, ErrFinalizeReadback{PlanID: plan.ID, Cause: err}
	}
	return plan, nil
}

var sessionSlugNonWord = regexp.MustCompile(`[^a-z0-9]+`)

func (s *service) uniqueSessionSlug(ctx context.Context, slug, title string) (string, error) {
	base := sessionSlugify(slug)
	if base == "" {
		base = sessionSlugify(title)
	}
	if base == "" {
		base = "session"
	}
	candidate := base
	for i := 2; ; i++ {
		_, ok, err := s.store.Get(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !ok {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
}

func sessionSlugify(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	v = sessionSlugNonWord.ReplaceAllString(v, "-")
	return strings.Trim(v, "-")
}

func (s *service) lockSession(sessionID string) func() {
	s.lockMu.Lock()
	mu := s.sessionLocks[sessionID]
	if mu == nil {
		mu = &sync.Mutex{}
		s.sessionLocks[sessionID] = mu
	}
	s.lockMu.Unlock()
	mu.Lock()
	return mu.Unlock
}

// prefillWorkPosture marks the Work Posture section as autofilled+reviewed (never
// agent-authored). The actual posture is derived from scenario maturity when the
// plan is persisted; the section here is an informational review marker so the
// wizard surfaces posture without asking the author to write the Greenfield block.
func prefillWorkPosture(sections []Section) {
	idx := indexOf(sections, SectionWorkPosture)
	if idx < 0 {
		return
	}
	sections[idx].Content = "Work posture is derived automatically from scenario maturity (default: greenfield). Do not author the Greenfield/Brownfield block — the renderer injects it."
	sections[idx].Filled = true
	sections[idx].Autofilled = true
	sections[idx].Mandatory = false
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

// stepForCurrentSessionState is the guided step a normal mutation returns. It
// delegates to nextGuidedStep so a mutation never reports final_review while
// global relevant context is unresolved or a phase is still incomplete — the
// premature-final-review friction. CurrentSectionKey alone is insufficient
// because it tracks only mandatory *sections* (the phases section reads "filled"
// as soon as one draft exists, even an incomplete one), and the global-context
// checkpoint is not a mandatory section at all.
func stepForCurrentSessionState(sess Session) GuidedStep {
	return nextGuidedStep(sess)
}

// nextGuidedStep selects the guided step for the session's true next required
// action, mirroring ContinueAuthoring's resolution order: finalized → first
// unfilled mandatory section → global relevant-context checkpoint → first
// incomplete phase → outstanding structure violation → final review. It is pure
// (no command-reference seam); that seam runs only at ValidateStructure/Finalize,
// so a clean nextGuidedStep is a "structurally ready" hint, never a guarantee.
func nextGuidedStep(sess Session) GuidedStep {
	if sess.Finalized {
		return stepForFinalizedPlan(sess, sess.PlanID, sess.Slug)
	}
	if key := firstUnfilledMandatory(sess.Sections); key != "" {
		if sec, ok := sectionByKey(sess.Sections, key); ok {
			return stepForSection(sess, sec)
		}
	}
	if !globalContextResolved(sess) {
		return stepForGlobalContextCheckpoint(sess)
	}
	if id := nextIncompletePhaseID(sess.PhaseDrafts); id != "" {
		if phase, ok := findDraft(sess.PhaseDrafts, id); ok {
			return stepForPhase(sess, phase)
		}
	}
	if violations := sessionViolations(sess); len(violations) > 0 {
		return stepForValidation(sess, false, violations)
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
// referencesGateMessage is the single message for the references requirement.
// References is mandatory but satisfiable by a NO_CODE_REFS: reason, so its
// requirement is enforced by this gate (not the generic empty-mandatory message),
// which keeps the "unless NO_CODE_REFS" escape in the same sentence.
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
	candidate.RejectionReason = strings.TrimSpace(candidate.RejectionReason)
	if candidate.Status == "" {
		candidate.Status = ReferenceCandidatePending
	}
	return candidate
}

func indexOfReferenceCandidate(candidates []ReferenceCandidate, id string) int {
	id = strings.TrimSpace(id)
	for i := range candidates {
		if candidates[i].ID == id {
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

// globalContextResolved reports whether the plan-wide relevant-context checkpoint
// has been addressed: at least one accepted/submitted global context item, or an
// explicit NO_CONTEXT skip reason recorded in the relevant-context section.
func globalContextResolved(sess Session) bool {
	if len(sess.RelevantContext) > 0 {
		return true
	}
	return noContextReason(contentOf(sess.Sections, SectionRelevantContext)) != ""
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
	case PhaseFieldAffectedAreas:
		phase.AffectedAreas = splitLines(content)
	case PhaseFieldSteps:
		phase.Steps = splitLines(content)
	case PhaseFieldExpectedOutputs:
		phase.ExpectedOutputs = splitLines(content)
	case PhaseFieldValidation:
		phase.Validation = content
	case PhaseFieldRisksHazards:
		phase.RisksHazards = splitLines(content)
	case PhaseFieldHandoffNotes:
		phase.HandoffNotes = content
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
		// Free-form phase context lines are classified as notes only — prose must
		// never become an executable command argv. Executable setup context flows
		// through typed context-submit/candidate acceptance.
		phase.RelevantContext = append(phase.RelevantContext, noteContextItemsFromLines(content, phase.ID)...)
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

func renumberPhaseDrafts(phases []PhaseDraft) {
	for i := range phases {
		phases[i].Order = i + 1
	}
}

// sessionToPlan maps a finalized session's sections into the structured plans
// model. The prose sections map directly to the matching plan fields; the
// references and phases sections carry authored markup (the same [CODE:]/[REQ:]/
// [DOC:] locators and `### Phase N — Title` headings the plans renderer emits),
// so they are parsed through the plans-domain markdown parser — the one SSOT for
// that grammar — by assembling a minimal markdown view and re-reading the
// references/phases it recovers. The prose fields are taken verbatim from the
// sections (not re-extracted) so authored content is never lossily reshaped. The
// regression anchor section is parsed into typed anchor fields when it uses the
// rendered structure; legacy prose remains marked as legacy/degraded.
func sessionToPlan(sess Session) (planmodel.Plan, error) {
	parsed, err := parseReferencesAndPhases(sess)
	if err != nil {
		return planmodel.Plan{}, err
	}
	p := planmodel.Plan{
		Title:                sess.Title,
		Slug:                 sess.Slug,
		Purpose:              contentOf(sess.Sections, SectionPurpose),
		ProblemStatement:     contentOf(sess.Sections, SectionProblemStatement),
		TargetOutcome:        contentOf(sess.Sections, SectionTargetOutcome),
		Scope:                contentOf(sess.Sections, SectionScope),
		NonGoals:             contentOf(sess.Sections, SectionNonGoals),
		Assumptions:          contentOf(sess.Sections, SectionAssumptions),
		TechnicalApproach:    contentOf(sess.Sections, SectionTechnicalApproach),
		Constraints:          contentOf(sess.Sections, SectionConstraints),
		ProhibitedApproaches: contentOf(sess.Sections, SectionProhibitedApproaches),
		ValidationStrategy:   contentOf(sess.Sections, SectionValidationStrategy),
		DefinitionOfDone:     contentOf(sess.Sections, SectionDefinitionOfDone),
		References:           parsed.References,
		Phases:               parsed.Phases,
		RelevantContext:      append([]planmodel.RelevantContextItem(nil), sess.RelevantContext...),
	}
	p.RelevantContext = append(p.RelevantContext, contextItemsFromLines(contentOf(sess.Sections, SectionRequiredReading), planmodel.RelevantContextScopeGlobal, "")...)
	if len(sess.PhaseDrafts) > 0 {
		p.Phases = phaseDraftsToPlanPhases(sess.PhaseDrafts)
	}
	p.ChangeBoundary = planmodel.ParseBoundarySection(contentOf(sess.Sections, SectionAcceptanceBoundary))
	if reason := noCodeRefsReason(contentOf(sess.Sections, SectionReferences)); reason != "" {
		p.Constraints = appendNoCodeRefsReason(p.Constraints, reason)
	}
	if anchor := strings.TrimSpace(contentOf(sess.Sections, SectionRegressionAnchor)); anchor != "" {
		p.RegressionAnchor = planmodel.ParseRegressionAnchorBlock(anchor)
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
			AffectedAreas:   append([]string(nil), draft.AffectedAreas...),
			Steps:           append([]string(nil), draft.Steps...),
			ExpectedOutputs: append([]string(nil), draft.ExpectedOutputs...),
			Validation:      draft.Validation,
			RisksHazards:    append([]string(nil), draft.RisksHazards...),
			HandoffNotes:    draft.HandoffNotes,
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

func indexOfContextItem(items []planmodel.RelevantContextItem, id string) int {
	id = strings.TrimSpace(id)
	for i := range items {
		if strings.TrimSpace(items[i].ID) == id {
			return i
		}
	}
	return -1
}

func removeContextItemAt(items []planmodel.RelevantContextItem, pos int) []planmodel.RelevantContextItem {
	out := make([]planmodel.RelevantContextItem, 0, len(items)-1)
	out = append(out, items[:pos]...)
	out = append(out, items[pos+1:]...)
	return out
}

// noteContextItemsFromLines classifies every free-form line of a phase
// relevant_context submission as a NOTE (never an executable skill/command), so
// prose can no longer silently become a bad `prompt-manager skill read ...` argv.
// Executable setup context must flow through typed context-submit/candidate
// acceptance, which carries an explicit kind/command (contract decision §6). A
// NO_CONTEXT: line is preserved verbatim so the no-context checkpoint still
// recognizes it.
func noteContextItemsFromLines(content, phaseID string) []planmodel.RelevantContextItem {
	var out []planmodel.RelevantContextItem
	for _, line := range splitLines(content) {
		item := planmodel.RelevantContextItem{
			ID:           uuid.NewString(),
			Kind:         planmodel.RelevantContextNote,
			Scope:        planmodel.RelevantContextScopePhase,
			PhaseID:      phaseID,
			Label:        line,
			Reason:       "Authored phase note.",
			Instruction:  line,
			Required:     false,
			RepeatPolicy: planmodel.RelevantContextPhaseEntry,
			Source:       planmodel.RelevantContextSourceAuthored,
			Status:       planmodel.RelevantContextStatusReady,
		}
		out = append(out, item)
	}
	return out
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
		// A code_ref/doc context item whose target obviously belongs to the other
		// kind is the same docs-as-CODE mistake at context scope; reject it so the
		// rendered plan never mislabels a setup reference.
		if msg := contextItemKindMismatch(item); msg != "" {
			add(msg)
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
