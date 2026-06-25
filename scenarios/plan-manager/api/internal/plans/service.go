package plans

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"plan-manager/internal/clock"

	"github.com/google/uuid"
)

// Service is the plans application surface — the structured-plan SSOT.
type Service interface {
	Create(ctx context.Context, p Plan) (Plan, error)
	Update(ctx context.Context, p Plan) (Plan, error)
	Get(ctx context.Context, idOrSlug string) (Plan, error)
	List(ctx context.Context, filter ListFilter) ([]Plan, error)
	Archive(ctx context.Context, idOrSlug string) (Plan, error)
	Render(ctx context.Context, idOrSlug string) (string, error)

	AddPhase(ctx context.Context, planID string, phase Phase) (Plan, error)
	UpdatePhase(ctx context.Context, planID string, phase Phase) (Plan, error)

	GetGraph(ctx context.Context, planID string) ([]PlanEdge, error)
	LinkSupersession(ctx context.Context, supersedingID, supersededID string) (Plan, error)

	ListTemplates(ctx context.Context) ([]PlanTemplate, error)
	CreateFromTemplate(ctx context.Context, templateID, title, slug string) (Plan, error)

	Import(ctx context.Context, sourcePath, markdown string) (Plan, error)
	Migrate(ctx context.Context, idOrSlug string) (Plan, error)
}

type service struct {
	repo   Repository
	clock  clock.Clock
	reader SourceReader
}

// Deps wires the plans Service. Reader is optional (nil disables reading
// markdown from disk; Import then requires inline markdown).
type Deps struct {
	Repo   Repository
	Clock  clock.Clock
	Reader SourceReader
}

// NewService constructs the plans Service.
func NewService(d Deps) Service {
	clk := d.Clock
	if clk == nil {
		clk = clock.System{}
	}
	return &service{repo: d.Repo, clock: clk, reader: d.Reader}
}

var _ Service = (*service)(nil)

func (s *service) Create(ctx context.Context, p Plan) (Plan, error) {
	p.Title = strings.TrimSpace(p.Title)
	if p.Title == "" {
		return Plan{}, ErrInvalidPlan{Reason: "title is required"}
	}
	p.ID = uuid.NewString()
	slug, err := s.uniqueSlug(ctx, p.Slug, p.Title)
	if err != nil {
		return Plan{}, err
	}
	p.Slug = slug
	now := s.now()
	p.CreatedAt = now
	p.UpdatedAt = now
	p.Phases = normalizePhases(p.Phases)
	p.Status = computePlanStatus(p.Phases)
	p.ContentHash = contentHash(p)
	if err := s.repo.Save(ctx, p); err != nil {
		return Plan{}, err
	}
	return p, nil
}

func (s *service) Update(ctx context.Context, p Plan) (Plan, error) {
	if strings.TrimSpace(p.ID) == "" {
		return Plan{}, ErrInvalidPlan{Reason: "id is required for update"}
	}
	existing, ok, err := s.repo.Get(ctx, p.ID)
	if err != nil {
		return Plan{}, err
	}
	if !ok {
		return Plan{}, ErrPlanNotFound{ID: p.ID}
	}
	// Preserve computed/identity fields; the caller only owns authored fields.
	p.Slug = existing.Slug
	p.CreatedAt = existing.CreatedAt
	p.UpdatedAt = s.now()
	if strings.TrimSpace(p.Title) == "" {
		p.Title = existing.Title
	}
	p.Phases = normalizePhases(p.Phases)
	if existing.Status == PlanStatusArchived {
		p.Status = PlanStatusArchived
	} else {
		p.Status = computePlanStatus(p.Phases)
	}
	// Supersession edges are managed via LinkSupersession, not free-form update.
	p.Supersedes = existing.Supersedes
	p.SupersededBy = existing.SupersededBy
	p.ContentHash = contentHash(p)
	if err := s.repo.Save(ctx, p); err != nil {
		return Plan{}, err
	}
	return p, nil
}

func (s *service) Get(ctx context.Context, idOrSlug string) (Plan, error) {
	p, ok, err := s.repo.Get(ctx, strings.TrimSpace(idOrSlug))
	if err != nil {
		return Plan{}, err
	}
	if !ok {
		return Plan{}, ErrPlanNotFound{ID: idOrSlug}
	}
	return p, nil
}

func (s *service) List(ctx context.Context, filter ListFilter) ([]Plan, error) {
	return s.repo.List(ctx, filter)
}

func (s *service) Archive(ctx context.Context, idOrSlug string) (Plan, error) {
	p, err := s.Get(ctx, idOrSlug)
	if err != nil {
		return Plan{}, err
	}
	p.Status = PlanStatusArchived
	p.UpdatedAt = s.now()
	if err := s.repo.Save(ctx, p); err != nil {
		return Plan{}, err
	}
	return p, nil
}

func (s *service) Render(ctx context.Context, idOrSlug string) (string, error) {
	p, err := s.Get(ctx, idOrSlug)
	if err != nil {
		return "", err
	}
	return RenderMarkdown(p), nil
}

func (s *service) AddPhase(ctx context.Context, planID string, phase Phase) (Plan, error) {
	p, err := s.Get(ctx, planID)
	if err != nil {
		return Plan{}, err
	}
	if phase.ID == "" {
		phase.ID = uuid.NewString()
	}
	phase.Status = defaultPhaseStatus(phase.Status)
	phase.Order = len(p.Phases) + 1
	p.Phases = append(p.Phases, phase)
	return s.saveRecomputed(ctx, p)
}

func (s *service) UpdatePhase(ctx context.Context, planID string, phase Phase) (Plan, error) {
	p, err := s.Get(ctx, planID)
	if err != nil {
		return Plan{}, err
	}
	idx := -1
	for i := range p.Phases {
		if p.Phases[i].ID == phase.ID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return Plan{}, ErrPhaseNotFound{PlanID: planID, PhaseID: phase.ID}
	}
	// Preserve order + identity; the caller owns the authored/status fields.
	phase.Order = p.Phases[idx].Order
	phase.Status = defaultPhaseStatus(phase.Status)
	p.Phases[idx] = phase
	return s.saveRecomputed(ctx, p)
}

func (s *service) GetGraph(ctx context.Context, planID string) ([]PlanEdge, error) {
	return s.repo.ListEdges(ctx, strings.TrimSpace(planID))
}

func (s *service) LinkSupersession(ctx context.Context, supersedingID, supersededID string) (Plan, error) {
	superseding, err := s.Get(ctx, supersedingID)
	if err != nil {
		return Plan{}, err
	}
	superseded, err := s.Get(ctx, supersededID)
	if err != nil {
		return Plan{}, err
	}
	if err := s.repo.SaveEdge(ctx, PlanEdge{
		FromPlanID: superseding.ID,
		ToPlanID:   superseded.ID,
		Kind:       "supersedes",
	}); err != nil {
		return Plan{}, err
	}
	superseding.Supersedes = appendUnique(superseding.Supersedes, superseded.ID)
	superseding.UpdatedAt = s.now()
	if err := s.repo.Save(ctx, superseding); err != nil {
		return Plan{}, err
	}
	superseded.SupersededBy = appendUnique(superseded.SupersededBy, superseding.ID)
	superseded.UpdatedAt = s.now()
	if err := s.repo.Save(ctx, superseded); err != nil {
		return Plan{}, err
	}
	return superseding, nil
}

func (s *service) ListTemplates(ctx context.Context) ([]PlanTemplate, error) {
	out := make([]PlanTemplate, 0, len(builtinTemplates))
	for _, t := range builtinTemplates {
		out = append(out, t.PlanTemplate)
	}
	return out, nil
}

func (s *service) CreateFromTemplate(ctx context.Context, templateID, title, slug string) (Plan, error) {
	tmpl, ok := builtinTemplates[templateID]
	if !ok {
		return Plan{}, ErrTemplateNotFound{ID: templateID}
	}
	p := Plan{
		Title:       title,
		Slug:        slug,
		Purpose:     tmpl.Purpose,
		Scope:       tmpl.Scope,
		Constraints: tmpl.Constraints,
		Phases:      clonePhases(tmpl.Phases),
	}
	return s.Create(ctx, p)
}

// saveRecomputed recomputes status + content_hash and persists.
func (s *service) saveRecomputed(ctx context.Context, p Plan) (Plan, error) {
	if p.Status != PlanStatusArchived {
		p.Status = computePlanStatus(p.Phases)
	}
	p.UpdatedAt = s.now()
	p.ContentHash = contentHash(p)
	if err := s.repo.Save(ctx, p); err != nil {
		return Plan{}, err
	}
	return p, nil
}

func (s *service) now() string { return s.clock.Now().UTC().Format(planTimeFormat) }

// uniqueSlug derives a filename-safe slug from the given slug-or-title and
// disambiguates against existing plans by appending -2, -3, ….
func (s *service) uniqueSlug(ctx context.Context, slug, title string) (string, error) {
	base := slugify(slug)
	if base == "" {
		base = slugify(title)
	}
	if base == "" {
		base = "plan"
	}
	candidate := base
	for i := 2; ; i++ {
		_, ok, err := s.repo.Get(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !ok {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
}

// computePlanStatus derives plan status from the phase-status set (COMPUTED;
// never free-text). Empty/all-todo => draft; all-done => complete; otherwise
// active. Archived is a separate explicit terminal state handled by callers.
func computePlanStatus(phases []Phase) PlanStatus {
	if len(phases) == 0 {
		return PlanStatusDraft
	}
	allDone := true
	anyMoved := false
	for _, ph := range phases {
		st := defaultPhaseStatus(ph.Status)
		if st != PhaseStatusDone {
			allDone = false
		}
		if st != PhaseStatusTodo {
			anyMoved = true
		}
	}
	switch {
	case allDone:
		return PlanStatusComplete
	case anyMoved:
		return PlanStatusActive
	default:
		return PlanStatusDraft
	}
}

func defaultPhaseStatus(s PhaseStatus) PhaseStatus {
	if s == "" {
		return PhaseStatusTodo
	}
	return s
}

func normalizePhases(phases []Phase) []Phase {
	out := make([]Phase, 0, len(phases))
	for i, ph := range phases {
		if ph.ID == "" {
			ph.ID = uuid.NewString()
		}
		ph.Order = i + 1
		ph.Status = defaultPhaseStatus(ph.Status)
		out = append(out, ph)
	}
	return out
}

func clonePhases(phases []Phase) []Phase {
	out := make([]Phase, 0, len(phases))
	for _, ph := range phases {
		cp := ph
		cp.ID = ""
		cp.RequiredReading = append([]string(nil), ph.RequiredReading...)
		cp.Reminders = append([]string(nil), ph.Reminders...)
		cp.BaselineScope = append([]string(nil), ph.BaselineScope...)
		out = append(out, cp)
	}
	return out
}

// contentHash computes a deterministic SHA-256 over the AUTHORED content of a
// plan (the structured fields that define identity for supersession edges). The
// computed/identity columns (ids, status, timestamps, the hash itself) are
// excluded so the same authored content always hashes the same — two plans with
// identical prose + phases hash identically regardless of their assigned ids.
func contentHash(p Plan) string {
	payload := struct {
		Title            string      `json:"title"`
		Purpose          string      `json:"purpose"`
		Scope            string      `json:"scope"`
		Constraints      string      `json:"constraints"`
		NonGoals         string      `json:"non_goals"`
		DefinitionOfDone string      `json:"definition_of_done"`
		References       []Reference `json:"references"`
		Phases           []Phase     `json:"phases"`
	}{
		Title:            p.Title,
		Purpose:          p.Purpose,
		Scope:            p.Scope,
		Constraints:      p.Constraints,
		NonGoals:         p.NonGoals,
		DefinitionOfDone: p.DefinitionOfDone,
		References:       stripRefIDs(p.References),
		Phases:           stripPhaseIDs(p.Phases),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// stripPhaseIDs returns phases with id fields zeroed so the content hash is
// identity-independent (phase IDs are assigned per-plan, authored content is not).
func stripPhaseIDs(phases []Phase) []Phase {
	out := make([]Phase, len(phases))
	for i, ph := range phases {
		ph.ID = ""
		ph.References = stripRefIDs(ph.References)
		out[i] = ph
	}
	return out
}

func stripRefIDs(refs []Reference) []Reference {
	out := make([]Reference, len(refs))
	for i, r := range refs {
		r.ID = ""
		out[i] = r
	}
	return out
}

var slugNonWord = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugNonWord.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func appendUnique(list []string, v string) []string {
	for _, e := range list {
		if e == v {
			return list
		}
	}
	return append(list, v)
}
