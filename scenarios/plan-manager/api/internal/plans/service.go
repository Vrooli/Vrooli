package plans

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"plan-manager/internal/clock"
	"plan-manager/internal/planmodel"

	"github.com/google/uuid"
)

// Service is the plans application surface — the structured-plan SSOT.
type Service interface {
	Create(ctx context.Context, p Plan) (Plan, error)
	Update(ctx context.Context, p Plan) (Plan, error)
	Get(ctx context.Context, idOrSlug string, workspace WorkspaceScope) (Plan, error)
	List(ctx context.Context, filter ListFilter) ([]Plan, error)
	Archive(ctx context.Context, idOrSlug string, workspace WorkspaceScope) (Plan, error)
	Render(ctx context.Context, idOrSlug string, workspace WorkspaceScope, opts RenderOptions) (RenderResult, error)

	AddPhase(ctx context.Context, planID string, phase Phase) (Plan, error)
	UpdatePhase(ctx context.Context, planID string, phase Phase) (Plan, error)

	GetGraph(ctx context.Context, planID string) ([]PlanEdge, error)
	LinkSupersession(ctx context.Context, supersedingID, supersededID string) (Plan, error)
	LinkDependency(ctx context.Context, dependingID, dependencyID string) (Plan, error)

	ListTemplates(ctx context.Context) ([]PlanTemplate, error)
	CreateFromTemplate(ctx context.Context, templateID, title, slug string) (Plan, error)

	Import(ctx context.Context, sourcePath, markdown, title, slug string, workspace WorkspaceScope) (Plan, error)
	Migrate(ctx context.Context, idOrSlug string) (Plan, error)
	Reconcile(ctx context.Context, req ReconcileRequest) (ReconcileResult, error)
}

type service struct {
	repo     Repository
	clock    clock.Clock
	reader   SourceReader
	mirror   MirrorStore
	maturity MaturityReader
}

// Deps wires the plans Service. Reader is optional (nil disables reading
// markdown from disk; Import then requires inline markdown). Maturity is optional
// (nil resolves every plan's work posture to Greenfield by default — never a
// false Brownfield).
type Deps struct {
	Repo     Repository
	Clock    clock.Clock
	Reader   SourceReader
	Mirror   MirrorStore
	Maturity MaturityReader
}

// NewService constructs the plans Service.
func NewService(d Deps) Service {
	clk := d.Clock
	if clk == nil {
		clk = clock.System{}
	}
	mirror := d.Mirror
	if mirror == nil {
		mirror = noMirrorStore{}
	}
	return &service{repo: d.Repo, clock: clk, reader: d.Reader, mirror: mirror, maturity: d.Maturity}
}

// applyPosture derives and stamps the work posture onto a plan (autofilled;
// never agent-authored). It is idempotent and honors an explicit override or an
// imported posture via ResolvePosture.
func (s *service) applyPosture(ctx context.Context, p *Plan) {
	posture, source, detail := ResolvePosture(ctx, *p, s.maturity)
	p.WorkPosture = posture
	p.WorkPostureSource = source
	p.WorkPostureDetail = detail
}

var _ Service = (*service)(nil)

func renderQualitySummary(p Plan) (string, []string) {
	report := planmodel.AssessPlanQuality(p, "")
	out := make([]string, 0, len(report.Findings))
	for _, finding := range report.Findings {
		out = append(out, fmt.Sprintf("%s %s at %s: %s", finding.Severity, finding.Code, finding.Location, finding.Message))
	}
	return report.Status, out
}

func (s *service) Create(ctx context.Context, p Plan) (Plan, error) {
	p.Title = strings.TrimSpace(p.Title)
	if p.Title == "" {
		return Plan{}, ErrInvalidPlan{Reason: "title is required"}
	}
	stampCanonicalWorkspace(&p, workspaceFromPlan(p))
	p.ID = uuid.NewString()
	slug, err := s.uniqueSlug(ctx, p.Slug, p.Title, workspaceFromPlan(p))
	if err != nil {
		return Plan{}, err
	}
	p.Slug = slug
	now := s.now()
	p.CreatedAt = now
	p.UpdatedAt = now
	p.Phases = normalizePhases(p.Phases)
	p.Status = computePlanStatus(p.Phases)
	s.applyPosture(ctx, &p)
	p.ContentHash = contentHash(p)
	hashMatches, err := s.contentHashMatches(ctx, p)
	if err != nil {
		return Plan{}, err
	}
	for _, match := range hashMatches {
		p.Supersedes = appendUnique(p.Supersedes, match.ID)
	}
	if err := s.repo.WithTx(ctx, func(repo Repository) error {
		if err := repo.Save(ctx, p); err != nil {
			return err
		}
		for _, match := range hashMatches {
			match.SupersededBy = appendUnique(match.SupersededBy, p.ID)
			match.UpdatedAt = s.now()
			if err := repo.SaveEdge(ctx, PlanEdge{FromPlanID: p.ID, ToPlanID: match.ID, Kind: EdgeKindSupersedes}); err != nil {
				return err
			}
			if err := repo.Save(ctx, match); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return Plan{}, err
	}
	return s.publishMirror(ctx, p)
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
	p.WorkspaceID = existing.WorkspaceID
	p.WorkspaceRoot = existing.WorkspaceRoot
	p.CreatedAt = existing.CreatedAt
	p.UpdatedAt = s.now()
	if strings.TrimSpace(p.Title) == "" {
		p.Title = existing.Title
	}
	p.Phases = normalizePhases(reconcilePhaseIDs(existing.Phases, p.Phases))
	if existing.Status == PlanStatusArchived {
		p.Status = PlanStatusArchived
	} else {
		p.Status = computePlanStatus(p.Phases)
	}
	// Supersession edges are managed by graph APIs and content-hash derivation,
	// not free-form update payload fields.
	p.Supersedes = existing.Supersedes
	p.SupersededBy = existing.SupersededBy
	// Import provenance and preserved legacy sections are governance lineage, not
	// authored fields: a normal authored-field update that omits them must not drop
	// them. They are preserved unless the caller explicitly supplies a replacement.
	if p.ImportProvenance == nil {
		p.ImportProvenance = existing.ImportProvenance
	}
	if len(p.PreservedLegacySections) == 0 {
		p.PreservedLegacySections = existing.PreservedLegacySections
	}
	s.applyPosture(ctx, &p)
	p.ContentHash = contentHash(p)
	hashMatches, err := s.contentHashMatches(ctx, p)
	if err != nil {
		return Plan{}, err
	}
	for _, match := range hashMatches {
		p.Supersedes = appendUnique(p.Supersedes, match.ID)
	}
	if err := s.repo.WithTx(ctx, func(repo Repository) error {
		if err := repo.Save(ctx, p); err != nil {
			return err
		}
		for _, match := range hashMatches {
			match.SupersededBy = appendUnique(match.SupersededBy, p.ID)
			match.UpdatedAt = s.now()
			if err := repo.SaveEdge(ctx, PlanEdge{FromPlanID: p.ID, ToPlanID: match.ID, Kind: EdgeKindSupersedes}); err != nil {
				return err
			}
			if err := repo.Save(ctx, match); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return Plan{}, err
	}
	return s.publishMirror(ctx, p)
}

func (s *service) Get(ctx context.Context, idOrSlug string, workspace WorkspaceScope) (Plan, error) {
	p, err := s.resolvePlan(ctx, idOrSlug, workspace)
	if err != nil {
		return Plan{}, err
	}
	p.Mirror = s.ensureMirrorPath(ctx, p)
	return p, nil
}

func (s *service) List(ctx context.Context, filter ListFilter) ([]Plan, error) {
	return s.repo.List(ctx, filter)
}

func (s *service) Archive(ctx context.Context, idOrSlug string, workspace WorkspaceScope) (Plan, error) {
	p, err := s.Get(ctx, idOrSlug, workspace)
	if err != nil {
		return Plan{}, err
	}
	p.Status = PlanStatusArchived
	p.UpdatedAt = s.now()
	if err := s.repo.Save(ctx, p); err != nil {
		return Plan{}, err
	}
	return s.publishMirror(ctx, p)
}

func (s *service) Render(ctx context.Context, idOrSlug string, workspace WorkspaceScope, opts RenderOptions) (RenderResult, error) {
	p, err := s.Get(ctx, idOrSlug, workspace)
	if err != nil {
		return RenderResult{}, err
	}
	qualityStatus, qualityFindings := renderQualitySummary(p)
	if opts.Compact {
		return RenderResult{
			Markdown:        RenderMarkdownWithOptions(p, opts),
			Mirror:          p.Mirror,
			Plan:            p,
			QualityStatus:   qualityStatus,
			QualityFindings: qualityFindings,
		}, nil
	}
	data, meta, err := s.mirror.Read(ctx, p)
	if err == nil && meta.Status == RenderedMirrorStatusFresh {
		p.Mirror = meta
		return RenderResult{Markdown: string(data), Mirror: meta, Plan: p, QualityStatus: qualityStatus, QualityFindings: qualityFindings}, nil
	}
	repaired, err := s.publishMirror(ctx, p)
	if err != nil {
		markdown := RenderMarkdownWithOptions(p, opts)
		if meta.Status == "" {
			meta = repaired.Mirror
		}
		if meta.Status == "" {
			meta.Status = RenderedMirrorStatusWriteFailed
		}
		p.Mirror = meta
		return RenderResult{Markdown: markdown, Mirror: meta, Repaired: false, Plan: p, QualityStatus: qualityStatus, QualityFindings: qualityFindings}, nil
	}
	return RenderResult{Markdown: RenderMarkdownWithOptions(repaired, opts), Mirror: repaired.Mirror, Repaired: true, Plan: repaired, QualityStatus: qualityStatus, QualityFindings: qualityFindings}, nil
}

func (s *service) AddPhase(ctx context.Context, planID string, phase Phase) (Plan, error) {
	p, err := s.Get(ctx, planID, WorkspaceScope{})
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
	p, err := s.Get(ctx, planID, WorkspaceScope{})
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
	superseding, err := s.Get(ctx, supersedingID, WorkspaceScope{})
	if err != nil {
		return Plan{}, err
	}
	superseded, err := s.Get(ctx, supersededID, WorkspaceScope{})
	if err != nil {
		return Plan{}, err
	}
	superseding.Supersedes = appendUnique(superseding.Supersedes, superseded.ID)
	superseding.UpdatedAt = s.now()
	superseded.SupersededBy = appendUnique(superseded.SupersededBy, superseding.ID)
	superseded.UpdatedAt = s.now()

	// The edge and both plans' updated edge-lists are one logical change — commit
	// them atomically so a mid-sequence failure can't leave a dangling edge or a
	// one-sided supersession link.
	if err := s.repo.WithTx(ctx, func(repo Repository) error {
		if err := repo.SaveEdge(ctx, PlanEdge{
			FromPlanID: superseding.ID,
			ToPlanID:   superseded.ID,
			Kind:       EdgeKindSupersedes,
		}); err != nil {
			return err
		}
		if err := repo.Save(ctx, superseding); err != nil {
			return err
		}
		return repo.Save(ctx, superseded)
	}); err != nil {
		return Plan{}, err
	}
	return superseding, nil
}

func (s *service) LinkDependency(ctx context.Context, dependingID, dependencyID string) (Plan, error) {
	depending, err := s.Get(ctx, dependingID, WorkspaceScope{})
	if err != nil {
		return Plan{}, err
	}
	dependency, err := s.Get(ctx, dependencyID, WorkspaceScope{})
	if err != nil {
		return Plan{}, err
	}
	if depending.ID == dependency.ID {
		return Plan{}, ErrInvalidPlan{Reason: "a plan cannot depend on itself"}
	}
	depending.UpdatedAt = s.now()
	if err := s.repo.WithTx(ctx, func(repo Repository) error {
		if err := repo.SaveEdge(ctx, PlanEdge{
			FromPlanID: depending.ID,
			ToPlanID:   dependency.ID,
			Kind:       EdgeKindDependsOn,
		}); err != nil {
			return err
		}
		return repo.Save(ctx, depending)
	}); err != nil {
		return Plan{}, err
	}
	return depending, nil
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
		Title:              title,
		Slug:               slug,
		Purpose:            tmpl.Purpose,
		ProblemStatement:   tmpl.ProblemStatement,
		TargetOutcome:      tmpl.TargetOutcome,
		Scope:              tmpl.Scope,
		Constraints:        tmpl.Constraints,
		TechnicalApproach:  tmpl.TechnicalApproach,
		ValidationStrategy: tmpl.ValidationStrategy,
		Phases:             clonePhases(tmpl.Phases),
	}
	return s.Create(ctx, p)
}

// saveRecomputed recomputes status + content_hash and persists.
func (s *service) saveRecomputed(ctx context.Context, p Plan) (Plan, error) {
	if p.Status != PlanStatusArchived {
		p.Status = computePlanStatus(p.Phases)
	}
	s.applyPosture(ctx, &p)
	p.UpdatedAt = s.now()
	p.ContentHash = contentHash(p)
	hashMatches, err := s.contentHashMatches(ctx, p)
	if err != nil {
		return Plan{}, err
	}
	for _, match := range hashMatches {
		p.Supersedes = appendUnique(p.Supersedes, match.ID)
	}
	if err := s.repo.WithTx(ctx, func(repo Repository) error {
		if err := repo.Save(ctx, p); err != nil {
			return err
		}
		for _, match := range hashMatches {
			match.SupersededBy = appendUnique(match.SupersededBy, p.ID)
			match.UpdatedAt = s.now()
			if err := repo.SaveEdge(ctx, PlanEdge{FromPlanID: p.ID, ToPlanID: match.ID, Kind: EdgeKindSupersedes}); err != nil {
				return err
			}
			if err := repo.Save(ctx, match); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return Plan{}, err
	}
	return s.publishMirror(ctx, p)
}

func (s *service) publishMirror(ctx context.Context, p Plan) (Plan, error) {
	markdown := []byte(RenderMarkdown(p))
	meta, err := s.mirror.Publish(ctx, p, markdown, s.now())
	if err != nil {
		if fallback, pathErr := s.mirror.PathFor(ctx, p); pathErr == nil {
			meta = fallback
		}
		meta.Status = RenderedMirrorStatusWriteFailed
		meta.RenderVersion = RendererVersion
		meta.LastError = err.Error()
		p.Mirror = meta
		_ = s.repo.Save(ctx, p)
		return p, nil
	}
	p.Mirror = meta
	if err := s.repo.Save(ctx, p); err != nil {
		return Plan{}, err
	}
	return p, nil
}

func (s *service) ensureMirrorPath(ctx context.Context, p Plan) RenderedPlanMirror {
	meta, err := s.mirror.PathFor(ctx, p)
	if err != nil {
		p.Mirror.Status = RenderedMirrorStatusUnknown
		p.Mirror.LastError = err.Error()
		p.Mirror.RenderVersion = RendererVersion
		return p.Mirror
	}
	if p.Mirror.Path == "" {
		p.Mirror.Path = meta.Path
	}
	if p.Mirror.RelativePath == "" {
		p.Mirror.RelativePath = meta.RelativePath
	}
	if p.Mirror.RenderVersion == "" {
		p.Mirror.RenderVersion = RendererVersion
	}
	if p.Mirror.Status == "" {
		p.Mirror.Status = RenderedMirrorStatusUnknown
	}
	return p.Mirror
}

func (s *service) contentHashMatches(ctx context.Context, p Plan) ([]Plan, error) {
	if strings.TrimSpace(p.ContentHash) == "" {
		return nil, nil
	}
	all, err := s.repo.List(ctx, ListFilter{IncludeArchived: true})
	if err != nil {
		return nil, err
	}
	matches := make([]Plan, 0)
	for _, candidate := range all {
		if candidate.ID == p.ID || candidate.Status == PlanStatusArchived {
			continue
		}
		if candidate.ContentHash == p.ContentHash {
			matches = append(matches, candidate)
		}
	}
	return matches, nil
}

func (s *service) now() string { return s.clock.Now().UTC().Format(planTimeFormat) }

// uniqueSlug derives a filename-safe slug from the given slug-or-title and
// disambiguates against existing plans by appending -2, -3, ….
func (s *service) uniqueSlug(ctx context.Context, slug, title string, workspace WorkspaceScope) (string, error) {
	base := slugify(slug)
	if base == "" {
		base = slugify(title)
	}
	if base == "" {
		base = "plan"
	}
	// Cap DERIVED slugs at a typeable length (word-boundary truncation); the
	// collision suffix below may exceed it by its own length only. The cap is
	// derivation-time only — stored long slugs keep resolving and keep their
	// mirror paths (slugify itself is uncapped).
	base = planmodel.TruncateSlug(base, planmodel.MaxSlugLength)
	candidate := base
	for i := 2; ; i++ {
		ok, err := s.slugExistsInWorkspace(ctx, candidate, workspace)
		if err != nil {
			return "", err
		}
		if !ok {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
}

func (s *service) resolvePlan(ctx context.Context, idOrSlug string, workspace WorkspaceScope) (Plan, error) {
	ref := strings.TrimSpace(idOrSlug)
	if ref == "" {
		return Plan{}, ErrPlanNotFound{ID: idOrSlug}
	}
	if !workspaceSpecified(workspace) {
		p, ok, err := s.repo.Get(ctx, ref)
		if err != nil {
			return Plan{}, err
		}
		if !ok {
			return Plan{}, ErrPlanNotFound{ID: idOrSlug}
		}
		return p, nil
	}
	all, err := s.repo.List(ctx, ListFilter{IncludeArchived: true, WorkspaceID: workspace.ID, WorkspaceRoot: workspace.Root})
	if err != nil {
		return Plan{}, err
	}
	for _, p := range all {
		if p.ID == ref || p.Slug == ref {
			return p, nil
		}
	}
	return Plan{}, ErrPlanNotFound{ID: idOrSlug}
}

func (s *service) slugExistsInWorkspace(ctx context.Context, slug string, workspace WorkspaceScope) (bool, error) {
	all, err := s.repo.List(ctx, ListFilter{IncludeArchived: true})
	if err != nil {
		return false, err
	}
	for _, p := range all {
		if p.Slug == slug && planMatchesWorkspace(p, workspace) {
			return true, nil
		}
	}
	return false, nil
}

func workspaceFromPlan(p Plan) WorkspaceScope {
	scope := WorkspaceScope{ID: strings.TrimSpace(p.WorkspaceID), Root: strings.TrimSpace(p.WorkspaceRoot)}
	if !workspaceSpecified(scope) && p.ImportProvenance != nil {
		scope.ID = strings.TrimSpace(p.ImportProvenance.WorkspaceID)
		scope.Root = strings.TrimSpace(p.ImportProvenance.WorkspaceRoot)
	}
	return normalizeWorkspaceScope(scope)
}

func stampCanonicalWorkspace(p *Plan, workspace WorkspaceScope) {
	workspace = normalizeWorkspaceScope(workspace)
	if !workspaceSpecified(workspace) {
		return
	}
	p.WorkspaceID = workspace.ID
	p.WorkspaceRoot = workspace.Root
	if p.ImportProvenance != nil {
		if p.ImportProvenance.WorkspaceID == "" {
			p.ImportProvenance.WorkspaceID = workspace.ID
		}
		if p.ImportProvenance.WorkspaceRoot == "" {
			p.ImportProvenance.WorkspaceRoot = workspace.Root
		}
	}
}

func planMatchesWorkspace(p Plan, workspace WorkspaceScope) bool {
	workspace = normalizeWorkspaceScope(workspace)
	if !workspaceSpecified(workspace) {
		return true
	}
	planScope := workspaceFromPlan(p)
	if workspace.ID != "" {
		return planScope.ID == workspace.ID
	}
	return planScope.Root != "" && planScope.Root == workspace.Root
}

func workspaceSpecified(workspace WorkspaceScope) bool {
	return strings.TrimSpace(workspace.ID) != "" || strings.TrimSpace(workspace.Root) != ""
}

func normalizeWorkspaceScope(workspace WorkspaceScope) WorkspaceScope {
	workspace.ID = strings.TrimSpace(workspace.ID)
	workspace.Root = strings.TrimSpace(workspace.Root)
	if workspace.Root != "" {
		workspace.Root = filepath.Clean(workspace.Root)
	}
	return workspace
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

// reconcilePhaseIDs preserves existing phase identity across an Update that may
// have dropped phase IDs (e.g. a caller round-tripping a plan through a surface
// that does not echo IDs). An incoming phase with no ID is matched to an existing
// phase by Title (first unused match) and inherits its ID, so the executions,
// decisions, and findings that reference that phase id are not orphaned by a
// silent re-key. A phase with no title match is genuinely new and gets a fresh id
// from normalizePhases.
func reconcilePhaseIDs(existing, incoming []Phase) []Phase {
	byTitle := make(map[string][]string, len(existing))
	for _, ex := range existing {
		if strings.TrimSpace(ex.ID) == "" {
			continue
		}
		key := strings.TrimSpace(ex.Title)
		byTitle[key] = append(byTitle[key], ex.ID)
	}
	used := make(map[string]bool, len(existing))
	out := make([]Phase, len(incoming))
	copy(out, incoming)
	for i := range out {
		if id := strings.TrimSpace(out[i].ID); id != "" {
			used[id] = true
			continue
		}
		for _, id := range byTitle[strings.TrimSpace(out[i].Title)] {
			if !used[id] {
				out[i].ID = id
				used[id] = true
				break
			}
		}
	}
	return out
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
		cp.AffectedAreas = append([]string(nil), ph.AffectedAreas...)
		cp.Steps = append([]string(nil), ph.Steps...)
		cp.ExpectedOutputs = append([]string(nil), ph.ExpectedOutputs...)
		cp.RisksHazards = append([]string(nil), ph.RisksHazards...)
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
		Title                   string                `json:"title"`
		Purpose                 string                `json:"purpose"`
		ProblemStatement        string                `json:"problem_statement"`
		TargetOutcome           string                `json:"target_outcome"`
		Scope                   string                `json:"scope"`
		NonGoals                string                `json:"non_goals"`
		Assumptions             string                `json:"assumptions"`
		Constraints             string                `json:"constraints"`
		ProhibitedApproaches    string                `json:"prohibited_approaches"`
		TechnicalApproach       string                `json:"technical_approach"`
		ValidationStrategy      string                `json:"validation_strategy"`
		FinalValidationCommands []string              `json:"final_validation_commands"`
		RisksHazards            string                `json:"risks_hazards"`
		Decisions               []PlanDecision        `json:"decisions"`
		AssumptionRisks         []PlanAssumption      `json:"assumption_risks"`
		DefinitionOfDone        string                `json:"definition_of_done"`
		References              []Reference           `json:"references"`
		Phases                  []Phase               `json:"phases"`
		RelevantContext         []RelevantContextItem `json:"relevant_context"`
	}{
		Title:                   p.Title,
		Purpose:                 p.Purpose,
		ProblemStatement:        p.ProblemStatement,
		TargetOutcome:           p.TargetOutcome,
		Scope:                   p.Scope,
		NonGoals:                p.NonGoals,
		Assumptions:             p.Assumptions,
		Constraints:             p.Constraints,
		ProhibitedApproaches:    p.ProhibitedApproaches,
		TechnicalApproach:       p.TechnicalApproach,
		ValidationStrategy:      p.ValidationStrategy,
		FinalValidationCommands: p.FinalValidationCommands,
		RisksHazards:            p.RisksHazards,
		Decisions:               p.Decisions,
		AssumptionRisks:         p.AssumptionRisks,
		DefinitionOfDone:        p.DefinitionOfDone,
		References:              stripRefIDs(p.References),
		Phases:                  stripPhaseIDs(p.Phases),
		RelevantContext:         stripRelevantContextIDs(p.RelevantContext),
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
		ph.RelevantContext = stripRelevantContextIDs(ph.RelevantContext)
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

func stripRelevantContextIDs(items []RelevantContextItem) []RelevantContextItem {
	out := make([]RelevantContextItem, len(items))
	for i, item := range items {
		item.ID = ""
		out[i] = item
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
