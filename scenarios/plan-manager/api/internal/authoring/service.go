package authoring

import (
	"context"
	"strings"
	"sync"

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
	SubmitFields(ctx context.Context, sessionID string, writes []FieldWrite) (Session, []FieldWriteResult, GuidedStep, error)
	Next(ctx context.Context, sessionID string) (Section, GuidedStep, bool, error)
	ContinueAuthoring(ctx context.Context, sessionID string) (Session, Section, PhaseDraft, bool, []StructureViolation, GuidedStep, error)
	ValidateStructure(ctx context.Context, sessionID string) (bool, []StructureViolation, GuidedStep, error)
	Autofill(ctx context.Context, sessionID string, sources []AutofillSource) (Session, []AutofillResult, GuidedStep, error)
	SubmitRelevantContextItem(ctx context.Context, sessionID, phaseID string, item planmodel.RelevantContextItem) (Session, planmodel.RelevantContextItem, []StructureViolation, GuidedStep, error)
	ListRelevantContext(ctx context.Context, sessionID, phaseID string) ([]planmodel.RelevantContextItem, GuidedStep, error)
	UpdateRelevantContextItem(ctx context.Context, sessionID, phaseID, itemID string, item planmodel.RelevantContextItem) (Session, planmodel.RelevantContextItem, []StructureViolation, GuidedStep, error)
	RemoveRelevantContextItem(ctx context.Context, sessionID, phaseID, itemID string) (Session, []StructureViolation, GuidedStep, error)
	DiscoverSkillPack(ctx context.Context, sessionID string, concepts []string, complexity string) (Session, SkillPackResult, []planmodel.RelevantContextItem, []planmodel.RelevantContextItem, []StructureViolation, GuidedStep, error)
	AddPhase(ctx context.Context, sessionID string, title, intent string) (Session, PhaseDraft, []StructureViolation, GuidedStep, error)
	MovePhase(ctx context.Context, sessionID, phaseID, beforePhaseID, afterPhaseID string) (Session, PhaseDraft, []StructureViolation, GuidedStep, error)
	GetPhase(ctx context.Context, sessionID, phaseID string) (PhaseDraft, GuidedStep, error)
	SubmitPhaseField(ctx context.Context, sessionID, phaseID string, field PhaseField, content string) (Session, []StructureViolation, GuidedStep, error)
	NextPhase(ctx context.Context, sessionID string) (PhaseDraft, GuidedStep, bool, error)
	PreviewPlan(ctx context.Context, sessionID string) (string, GuidedStep, error)
	Finalize(ctx context.Context, sessionID string, opts FinalizeOptions) (FinalizeResult, GuidedStep, error)
}

type service struct {
	store           SessionStore
	writer          PlanWriter
	reader          PlanReader
	anchor          AnchorIntentDeriver
	skills          SkillPackDiscoverer
	skillSteer      SkillApplicabilityResolver
	commands        CommandReferenceValidator
	sourceEvidence  SourceEvidenceAdvisor
	resolver        ReferenceResolver
	templateSeed    TemplateSeeder
	renderer        PlanRenderer
	posture         PosturePreparer
	storePath       string
	clock           clock.Clock
	lockMu          sync.Mutex
	sessionLocks    map[string]*sync.Mutex
	lifecycleCtx    context.Context
	lifecycleCancel context.CancelFunc
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
	Store         SessionStore
	Writer        PlanWriter
	Reader        PlanReader
	Anchor        AnchorIntentDeriver
	Skills        SkillPackDiscoverer
	SkillResolver SkillApplicabilityResolver
	Commands      CommandReferenceValidator
	// SourceEvidence is optional and advisory. A failed GCT lookup never makes
	// an otherwise deterministic-ready plan fail author validation.
	SourceEvidence SourceEvidenceAdvisor
	Resolver       ReferenceResolver
	TemplateSeeder TemplateSeeder
	Renderer       PlanRenderer
	Posture        PosturePreparer
	// StorePath is the resolved physical path of the plans SQLite store this
	// process writes through. Finalize reports it so a successful response
	// names WHERE the plan row lives (store identity, not just an id). Empty
	// (tests, unwired callers) degrades to "unknown" in the response — never a
	// fabricated path.
	StorePath string
	Clock     clock.Clock
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
	lifecycleCtx, lifecycleCancel := context.WithCancel(context.Background())
	return &service{
		store:           d.Store,
		writer:          d.Writer,
		reader:          d.Reader,
		anchor:          d.Anchor,
		skills:          d.Skills,
		skillSteer:      resolverOrNoop(d.SkillResolver),
		commands:        d.Commands,
		sourceEvidence:  d.SourceEvidence,
		resolver:        d.Resolver,
		templateSeed:    d.TemplateSeeder,
		renderer:        d.Renderer,
		posture:         d.Posture,
		storePath:       strings.TrimSpace(d.StorePath),
		clock:           clk,
		sessionLocks:    map[string]*sync.Mutex{},
		lifecycleCtx:    lifecycleCtx,
		lifecycleCancel: lifecycleCancel,
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
	var violations []StructureViolation
	sess, err := s.withSessionLock(ctx, sessionID, func(sess *Session) (bool, error) {
		idx := indexOf(sess.Sections, key)
		if idx < 0 {
			return false, ErrSectionNotFound{SessionID: sessionID, SectionKey: string(key)}
		}
		violations = s.applySection(ctx, sess, idx, content)
		return true, nil
	})
	if err != nil {
		return Session{}, nil, GuidedStep{}, err
	}
	return sess, violations, stepForCurrentSessionState(sess), nil
}

// applySection writes one section's content in place and returns its
// violations. Shared by SubmitSection and the batch SubmitFields path so both
// apply identical semantics.
func (s *service) applySection(ctx context.Context, sess *Session, idx int, content string) []StructureViolation {
	sess.Sections[idx].Content = content
	sess.Sections[idx].Filled = strings.TrimSpace(content) != ""
	sess.Sections[idx].Autofilled = false // an author submission supersedes any autofill
	violations := violationsForSection(sess.Sections[idx])
	violations = append(violations, s.commandViolationsForSection(ctx, sess.Sections[idx])...)
	sess.CurrentSectionKey = firstUnfilledMandatory(sess.Sections)
	return violations
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
	work := selectWorkItem(sess, nil)
	if work.Kind == WorkItemReview {
		violations, advisory := s.readinessAssessment(ctx, sess)
		violations = append(violations, s.commandViolationsForSections(ctx, sess.Sections)...)
		work = selectWorkItem(sess, violations)
		work.Step = appendSourceEvidenceAdvisory(work.Step, advisory)
	}
	return sess, work.Section, work.Phase, work.Ready, work.Violations, onlyRecommendedAction(work.Step), nil
}

func (s *service) ValidateStructure(ctx context.Context, sessionID string) (bool, []StructureViolation, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return false, nil, GuidedStep{}, err
	}
	violations, advisory := s.readinessAssessment(ctx, sess)
	violations = append(violations, s.commandViolationsForSections(ctx, sess.Sections)...)
	valid := len(violations) == 0
	return valid, violations, appendSourceEvidenceAdvisory(stepForValidation(sess, valid, violations), advisory), nil
}

func (s *service) Autofill(ctx context.Context, sessionID string, sources []AutofillSource) (Session, []AutofillResult, GuidedStep, error) {
	if len(sources) == 0 {
		sources = []AutofillSource{AutofillRegressionAnchor}
	}
	var results []AutofillResult
	sess, err := s.withSessionLock(ctx, sessionID, func(sess *Session) (bool, error) {
		results = make([]AutofillResult, 0, len(sources))
		for _, src := range sources {
			results = append(results, s.runAutofill(ctx, sess, src))
		}
		sess.CurrentSectionKey = firstUnfilledMandatory(sess.Sections)
		return true, nil
	})
	if err != nil {
		return Session{}, nil, GuidedStep{}, err
	}
	return sess, results, stepForCurrentSessionState(sess), nil
}
