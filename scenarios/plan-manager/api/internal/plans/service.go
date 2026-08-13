package plans

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"plan-manager/internal/planmodel"

	"github.com/vrooli/api-core/schedule"

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

	// ExtendChangeBoundary appends allow globs to a plan's change boundary. It is
	// the ONLY mutation permitted while an execution is non-terminal, and it is
	// deliberately narrower than a candidate revision: append-only, one field,
	// never a removal. See the method comment for why that is safe.
	ExtendChangeBoundary(ctx context.Context, planID string, workspace WorkspaceScope, globs []string) (Plan, []string, error)

	AddPhase(ctx context.Context, planID string, workspace WorkspaceScope, phase Phase) (Plan, error)
	AddPhaseWithImpact(ctx context.Context, planID string, workspace WorkspaceScope, phase Phase, allowRegression bool) (Plan, MutationImpact, error)
	UpdatePhase(ctx context.Context, planID string, workspace WorkspaceScope, phase Phase) (Plan, error)
	ReplacePhaseWithImpact(ctx context.Context, planID string, workspace WorkspaceScope, phase Phase, allowRegression bool) (Plan, MutationImpact, error)
	PatchPhase(ctx context.Context, planID string, workspace WorkspaceScope, phase Phase, paths []string, allowRegression bool) (Plan, MutationImpact, error)
	ListRelevantContext(ctx context.Context, idOrSlug string, workspace WorkspaceScope, phaseID string) ([]RelevantContextItem, error)
	AddRelevantContext(ctx context.Context, idOrSlug string, workspace WorkspaceScope, phaseID string, item RelevantContextItem, allowRegression bool) (Plan, MutationImpact, error)
	UpdateRelevantContext(ctx context.Context, idOrSlug string, workspace WorkspaceScope, phaseID, itemID string, item RelevantContextItem) (Plan, error)
	UpdateRelevantContextWithImpact(ctx context.Context, idOrSlug string, workspace WorkspaceScope, phaseID, itemID string, item RelevantContextItem, allowRegression bool) (Plan, MutationImpact, error)
	RemoveRelevantContext(ctx context.Context, idOrSlug string, workspace WorkspaceScope, phaseID, itemID string) (Plan, error)
	RemoveRelevantContextWithImpact(ctx context.Context, idOrSlug string, workspace WorkspaceScope, phaseID, itemID string, allowRegression bool) (Plan, MutationImpact, error)
	ListReferences(ctx context.Context, idOrSlug string, workspace WorkspaceScope, phaseID string) ([]Reference, error)
	AddReference(ctx context.Context, idOrSlug string, workspace WorkspaceScope, phaseID string, ref Reference, allowRegression bool) (Plan, MutationImpact, error)
	UpdateReference(ctx context.Context, idOrSlug string, workspace WorkspaceScope, phaseID, referenceID string, ref Reference) (Plan, error)
	UpdateReferenceWithImpact(ctx context.Context, idOrSlug string, workspace WorkspaceScope, phaseID, referenceID string, ref Reference, allowRegression bool) (Plan, MutationImpact, error)
	RemoveReference(ctx context.Context, idOrSlug string, workspace WorkspaceScope, phaseID, referenceID string) (Plan, error)
	RemoveReferenceWithImpact(ctx context.Context, idOrSlug string, workspace WorkspaceScope, phaseID, referenceID string, allowRegression bool) (Plan, MutationImpact, error)

	GetGraph(ctx context.Context, planID string) ([]PlanEdge, error)
	LinkSupersession(ctx context.Context, supersedingID, supersededID string) (Plan, error)
	LinkDependency(ctx context.Context, dependingID, dependencyID string) (Plan, error)

	ListTemplates(ctx context.Context) ([]PlanTemplate, error)
	CreateFromTemplate(ctx context.Context, templateID, title, slug string) (Plan, error)

	Import(ctx context.Context, sourcePath, markdown, title, slug string, workspace WorkspaceScope) (Plan, error)
	ImportSuperseding(ctx context.Context, sourcePath, markdown, title, slug string, workspace WorkspaceScope, supersede string) (Plan, Plan, error)
	Migrate(ctx context.Context, idOrSlug string) (Plan, error)
	Reconcile(ctx context.Context, req ReconcileRequest) (ReconcileResult, error)

	CreateCandidate(ctx context.Context, candidate CandidateRevision) (CandidateRevision, error)
	GetCandidate(ctx context.Context, id string) (CandidateRevision, error)
	PreviewCandidate(ctx context.Context, id string) (CandidateRevisionPreview, error)
	ValidateCandidate(ctx context.Context, id string) (CandidateRevisionPreview, error)
	ApplyCandidate(ctx context.Context, id, expectedBaseHash string, acknowledgeQualityImpact bool) (CandidateRevision, Plan, CandidateRevisionPreview, error)
	DiscardCandidate(ctx context.Context, id, reason string) (CandidateRevision, error)
}

type service struct {
	repo       Repository
	clock      schedule.Clock
	reader     SourceReader
	mirror     MirrorStore
	maturity   MaturityReader
	mutationMu sync.Mutex
}

// MutationImpact makes quality movement explicit on every patch/add surface.
type MutationImpact struct {
	BeforeGrade              string
	AfterGrade               string
	AddedIssueCodes          []string
	ClearedIssueCodes        []string
	ExecutionGradeRegression bool
	RegressionAcknowledged   bool
}

// Deps wires the plans Service. Reader is optional (nil disables reading
// markdown from disk; Import then requires inline markdown). Maturity is optional
// (nil resolves every plan's work posture to Greenfield by default — never a
// false Brownfield).
type Deps struct {
	Repo     Repository
	Clock    schedule.Clock
	Reader   SourceReader
	Mirror   MirrorStore
	Maturity MaturityReader
}

// NewService constructs the plans Service.
func NewService(d Deps) Service {
	clk := d.Clock
	if clk == nil {
		clk = schedule.System()
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

var syntheticItemIDPattern = regexp.MustCompile(`^item-\d+$`)

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
	planmodel.EnsureCurrentBaselineSet(&p)
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
	// The caller owns authored fields only. All identity, computed, governance,
	// and graph fields come from the stored plan; explicit lifecycle rules below
	// may then derive their next values.
	planmodel.PreserveNonAuthoredPlanFields(&p, &existing)
	p.UpdatedAt = s.now()
	if strings.TrimSpace(p.Title) == "" {
		p.Title = existing.Title
	}
	planmodel.EnsureCurrentBaselineSet(&p)
	p.Phases = normalizePhases(reconcilePhaseIDs(existing.Phases, p.Phases))
	if existing.Status == PlanStatusArchived {
		p.Status = PlanStatusArchived
	} else {
		p.Status = computePlanStatus(p.Phases)
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
		current := RenderMarkdownWithOptions(p, opts)
		if string(data) != current {
			repaired, publishErr := s.publishMirror(ctx, p)
			if publishErr == nil {
				return RenderResult{Markdown: RenderMarkdownWithOptions(repaired, opts), Mirror: repaired.Mirror, Repaired: true, Plan: repaired, QualityStatus: qualityStatus, QualityFindings: qualityFindings}, nil
			}
			meta.Status = RenderedMirrorStatusWriteFailed
			p.Mirror = meta
			return RenderResult{Markdown: current, Mirror: meta, Repaired: false, Plan: p, QualityStatus: qualityStatus, QualityFindings: qualityFindings}, nil
		}
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

func (s *service) AddPhase(ctx context.Context, planID string, workspace WorkspaceScope, phase Phase) (Plan, error) {
	p, err := s.Get(ctx, planID, workspace)
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

func (s *service) AddPhaseWithImpact(ctx context.Context, planID string, workspace WorkspaceScope, phase Phase, allowRegression bool) (Plan, MutationImpact, error) {
	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()
	p, err := s.Get(ctx, planID, workspace)
	if err != nil {
		return Plan{}, MutationImpact{}, err
	}
	beforeStatus := p.Status
	beforeReport := planmodel.AssessPlanQuality(p, "")
	if phase.ID == "" {
		phase.ID = uuid.NewString()
	}
	phase.Status = defaultPhaseStatus(phase.Status)
	phase.Order = len(p.Phases) + 1
	p.Phases = append(p.Phases, phase)
	impact, err := assessMutationImpactFromReport(beforeStatus, beforeReport, p, allowRegression)
	if err != nil {
		return Plan{}, impact, err
	}
	updated, err := s.saveRecomputed(ctx, p)
	return updated, impact, err
}

// ExtendChangeBoundary appends globs to the plan's acceptance_allow and returns
// the updated plan plus the globs that were actually new (empty when every glob
// was already covered).
//
// Why this exists as a first-class mutation rather than going through a
// candidate revision: ApplyCandidate deliberately refuses to run while an
// execution is non-terminal, because a whole-plan replacement mid-flight can
// change phases, acceptance, or validation strategy out from under a running
// agent. That guard is correct and stays. But it also made the authored blast
// radius unwidenable during the only period when its inaccuracy is discoverable
// — while the work is being done. The result was that an executing agent facing
// a genuinely-needed edit outside the boundary had exactly two options: write a
// workaround to stay inside a stale estimate, or stop. Both are worse than
// widening the boundary and re-validating.
//
// This mutation is safe where a whole-plan revision is not, because it is:
//   - append-only (an existing allow glob can never be removed, so no in-flight
//     edit is retroactively made illegal);
//   - single-field (phases, acceptance, and validation strategy are untouched);
//   - deny-respecting (the authored prohibition still wins — see below);
//   - monotonic for validation (widening allow only ever adds oracles/paths).
//
// Deny globs are NOT extendable and are re-checked here. acceptance_deny is the
// half of the boundary that expresses a real authored prohibition; an execution
// may discover that the estimate was too narrow, but it may not overrule what
// the plan explicitly forbade.
func (s *service) ExtendChangeBoundary(ctx context.Context, planID string, workspace WorkspaceScope, globs []string) (Plan, []string, error) {
	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()
	p, err := s.Get(ctx, planID, workspace)
	if err != nil {
		return Plan{}, nil, err
	}
	if p.Status == PlanStatusArchived {
		return Plan{}, nil, ErrInvalidPlan{Reason: "cannot extend the change boundary of an archived plan"}
	}
	boundary := p.ChangeBoundary.Normalized()
	existing := make(map[string]struct{}, len(boundary.AcceptanceAllow))
	for _, g := range boundary.AcceptanceAllow {
		existing[g] = struct{}{}
	}
	deny := make(map[string]struct{}, len(boundary.AcceptanceDeny))
	for _, g := range boundary.AcceptanceDeny {
		deny[g] = struct{}{}
	}
	var added []string
	for _, raw := range globs {
		glob := planmodel.NormalizeBoundaryGlob(raw)
		if glob == "" {
			continue
		}
		if tokens := planmodel.UnresolvedPlaceholders(glob); len(tokens) > 0 {
			return Plan{}, nil, ErrInvalidPlan{Reason: "boundary extension glob has unresolved placeholder(s) " + strings.Join(tokens, ", ") + ": " + glob}
		}
		if _, forbidden := deny[glob]; forbidden {
			return Plan{}, nil, ErrInvalidPlan{Reason: "boundary extension glob is explicitly forbidden by acceptance_deny: " + glob}
		}
		if covered, by := planmodel.DenyCovers(boundary.AcceptanceDeny, glob); covered {
			return Plan{}, nil, ErrInvalidPlan{Reason: "boundary extension glob " + glob + " falls under the forbidden path " + by}
		}
		if _, dup := existing[glob]; dup {
			continue
		}
		existing[glob] = struct{}{}
		added = append(added, glob)
	}
	if len(added) == 0 {
		return p, nil, nil
	}
	boundary.AcceptanceAllow = append(boundary.AcceptanceAllow, added...)
	p.ChangeBoundary = boundary.Normalized()
	sort.Strings(added)
	updated, err := s.saveRecomputed(ctx, p)
	if err != nil {
		return Plan{}, nil, err
	}
	return updated, added, nil
}

func (s *service) UpdatePhase(ctx context.Context, planID string, workspace WorkspaceScope, phase Phase) (Plan, error) {
	p, err := s.Get(ctx, planID, workspace)
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

func (s *service) ReplacePhaseWithImpact(ctx context.Context, planID string, workspace WorkspaceScope, phase Phase, allowRegression bool) (Plan, MutationImpact, error) {
	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()
	p, err := s.Get(ctx, planID, workspace)
	if err != nil {
		return Plan{}, MutationImpact{}, err
	}
	idx := phaseIndex(p.Phases, phase.ID)
	if idx < 0 {
		return Plan{}, MutationImpact{}, ErrPhaseNotFound{PlanID: planID, PhaseID: phase.ID}
	}
	beforeStatus := p.Status
	beforeReport := planmodel.AssessPlanQuality(p, "")
	phase.Order = p.Phases[idx].Order
	phase.Status = defaultPhaseStatus(phase.Status)
	p.Phases[idx] = phase
	impact, err := assessMutationImpactFromReport(beforeStatus, beforeReport, p, allowRegression)
	if err != nil {
		return Plan{}, impact, err
	}
	updated, err := s.saveRecomputed(ctx, p)
	return updated, impact, err
}

var allowedPhasePatchPaths = map[string]func(*Phase, Phase){
	"title":            func(dst *Phase, src Phase) { dst.Title = src.Title },
	"intent":           func(dst *Phase, src Phase) { dst.Intent = src.Intent },
	"affected_areas":   func(dst *Phase, src Phase) { dst.AffectedAreas = append([]string(nil), src.AffectedAreas...) },
	"steps":            func(dst *Phase, src Phase) { dst.Steps = append([]string(nil), src.Steps...) },
	"expected_outputs": func(dst *Phase, src Phase) { dst.ExpectedOutputs = append([]string(nil), src.ExpectedOutputs...) },
	"validation":       func(dst *Phase, src Phase) { dst.Validation = src.Validation },
	"acceptance":       func(dst *Phase, src Phase) { dst.Acceptance = src.Acceptance },
	"risks_hazards":    func(dst *Phase, src Phase) { dst.RisksHazards = append([]string(nil), src.RisksHazards...) },
	"handoff_notes":    func(dst *Phase, src Phase) { dst.HandoffNotes = src.HandoffNotes },
	"status":           func(dst *Phase, src Phase) { dst.Status = src.Status },
	"references":       func(dst *Phase, src Phase) { dst.References = append([]Reference(nil), src.References...) },
	"relevant_context": func(dst *Phase, src Phase) {
		dst.RelevantContext = append([]RelevantContextItem(nil), src.RelevantContext...)
	},
	"reminders":        func(dst *Phase, src Phase) { dst.Reminders = append([]string(nil), src.Reminders...) },
	"baseline_scope":   func(dst *Phase, src Phase) { dst.BaselineScope = append([]string(nil), src.BaselineScope...) },
	"change_boundary":  func(dst *Phase, src Phase) { dst.ChangeBoundary = src.ChangeBoundary },
	"validation_scope": func(dst *Phase, src Phase) { dst.ValidationScope = src.ValidationScope },
}

func (s *service) PatchPhase(ctx context.Context, planID string, workspace WorkspaceScope, phase Phase, paths []string, allowRegression bool) (Plan, MutationImpact, error) {
	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()
	if len(paths) == 0 {
		return Plan{}, MutationImpact{}, ErrInvalidPlan{Reason: "phase patch requires at least one update_mask path"}
	}
	p, err := s.Get(ctx, planID, workspace)
	if err != nil {
		return Plan{}, MutationImpact{}, err
	}
	idx := phaseIndex(p.Phases, phase.ID)
	if idx < 0 {
		return Plan{}, MutationImpact{}, ErrPhaseNotFound{PlanID: planID, PhaseID: phase.ID}
	}
	beforeStatus := p.Status
	beforeReport := planmodel.AssessPlanQuality(p, "")
	patched := p.Phases[idx]
	seen := make(map[string]struct{}, len(paths))
	for _, raw := range paths {
		path := strings.TrimSpace(raw)
		apply, ok := allowedPhasePatchPaths[path]
		if !ok {
			return Plan{}, MutationImpact{}, ErrInvalidPlan{Reason: "phase patch path is immutable, computed, or unknown: " + path}
		}
		if _, duplicate := seen[path]; duplicate {
			continue
		}
		seen[path] = struct{}{}
		apply(&patched, phase)
	}
	patched.ID = p.Phases[idx].ID
	patched.Order = p.Phases[idx].Order
	patched.Status = defaultPhaseStatus(patched.Status)
	p.Phases[idx] = patched
	impact, err := assessMutationImpactFromReport(beforeStatus, beforeReport, p, allowRegression)
	if err != nil {
		return Plan{}, impact, err
	}
	updated, err := s.saveRecomputed(ctx, p)
	return updated, impact, err
}

func assessMutationImpactFromReport(beforeStatus PlanStatus, beforeReport planmodel.QualityReport, after Plan, allowRegression bool) (MutationImpact, error) {
	afterReport := planmodel.AssessPlanQuality(after, "")
	beforeCodes := qualityCodes(beforeReport)
	afterCodes := qualityCodes(afterReport)
	impact := MutationImpact{BeforeGrade: beforeReport.Status, AfterGrade: afterReport.Status}
	for code := range afterCodes {
		if _, existed := beforeCodes[code]; !existed {
			impact.AddedIssueCodes = append(impact.AddedIssueCodes, code)
		}
	}
	for code := range beforeCodes {
		if _, remains := afterCodes[code]; !remains {
			impact.ClearedIssueCodes = append(impact.ClearedIssueCodes, code)
		}
	}
	sort.Strings(impact.AddedIssueCodes)
	sort.Strings(impact.ClearedIssueCodes)
	impact.ExecutionGradeRegression = beforeStatus != PlanStatusDraft && beforeReport.ExecutionReady() && !afterReport.ExecutionReady()
	impact.RegressionAcknowledged = impact.ExecutionGradeRegression && allowRegression
	if impact.ExecutionGradeRegression && !allowRegression {
		return impact, ErrInvalidPlan{Reason: "mutation would regress an execution-grade plan; pass allow_quality_regression to acknowledge: " + strings.Join(impact.AddedIssueCodes, ", ")}
	}
	return impact, nil
}

func qualityCodes(report planmodel.QualityReport) map[string]struct{} {
	out := make(map[string]struct{}, len(report.Findings))
	for _, finding := range report.Findings {
		out[finding.Code] = struct{}{}
	}
	return out
}

func (s *service) ListRelevantContext(ctx context.Context, idOrSlug string, workspace WorkspaceScope, phaseID string) ([]RelevantContextItem, error) {
	p, err := s.Get(ctx, idOrSlug, workspace)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(phaseID) == "" {
		return contextWithEffectiveIDs(p.RelevantContext), nil
	}
	idx := phaseIndex(p.Phases, phaseID)
	if idx < 0 {
		return nil, ErrPhaseNotFound{PlanID: p.ID, PhaseID: phaseID}
	}
	return contextWithEffectiveIDs(p.Phases[idx].RelevantContext), nil
}

func (s *service) AddRelevantContext(ctx context.Context, idOrSlug string, workspace WorkspaceScope, phaseID string, item RelevantContextItem, allowRegression bool) (Plan, MutationImpact, error) {
	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()
	p, err := s.Get(ctx, idOrSlug, workspace)
	if err != nil {
		return Plan{}, MutationImpact{}, err
	}
	beforeStatus := p.Status
	beforeReport := planmodel.AssessPlanQuality(p, "")
	item.ID = uuid.NewString()
	item.Source = planmodel.RelevantContextSourceAuthored
	item.Status = planmodel.RelevantContextStatusReady
	if strings.TrimSpace(phaseID) == "" {
		item.Scope, item.PhaseID = planmodel.RelevantContextScopeGlobal, ""
		p.RelevantContext = append(p.RelevantContext, item)
	} else {
		idx := phaseIndex(p.Phases, phaseID)
		if idx < 0 {
			return Plan{}, MutationImpact{}, ErrPhaseNotFound{PlanID: p.ID, PhaseID: phaseID}
		}
		item.Scope, item.PhaseID = planmodel.RelevantContextScopePhase, p.Phases[idx].ID
		p.Phases[idx].RelevantContext = append(p.Phases[idx].RelevantContext, item)
	}
	impact, err := assessMutationImpactFromReport(beforeStatus, beforeReport, p, allowRegression)
	if err != nil {
		return Plan{}, impact, err
	}
	updated, err := s.saveRecomputed(ctx, p)
	return updated, impact, err
}

func (s *service) UpdateRelevantContext(ctx context.Context, idOrSlug string, workspace WorkspaceScope, phaseID, itemID string, item RelevantContextItem) (Plan, error) {
	p, err := s.Get(ctx, idOrSlug, workspace)
	if err != nil {
		return Plan{}, err
	}
	if err := applyRelevantContextUpdate(&p, phaseID, itemID, item); err != nil {
		return Plan{}, err
	}
	return s.saveRecomputed(ctx, p)
}

func (s *service) UpdateRelevantContextWithImpact(ctx context.Context, idOrSlug string, workspace WorkspaceScope, phaseID, itemID string, item RelevantContextItem, allowRegression bool) (Plan, MutationImpact, error) {
	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()
	p, err := s.Get(ctx, idOrSlug, workspace)
	if err != nil {
		return Plan{}, MutationImpact{}, err
	}
	beforeStatus := p.Status
	beforeReport := planmodel.AssessPlanQuality(p, "")
	if err := applyRelevantContextUpdate(&p, phaseID, itemID, item); err != nil {
		return Plan{}, MutationImpact{}, err
	}
	impact, err := assessMutationImpactFromReport(beforeStatus, beforeReport, p, allowRegression)
	if err != nil {
		return Plan{}, impact, err
	}
	updated, err := s.saveRecomputed(ctx, p)
	return updated, impact, err
}

func applyRelevantContextUpdate(p *Plan, phaseID, itemID string, item RelevantContextItem) error {
	itemID = strings.TrimSpace(itemID)
	if itemID == "" {
		return ErrInvalidPlan{Reason: "context item id is required"}
	}
	if strings.TrimSpace(item.ID) == "" {
		item.ID = itemID
	}
	if strings.TrimSpace(item.ID) == "" || isSyntheticItemID(item.ID) {
		item.ID = uuid.NewString()
	}
	if strings.TrimSpace(phaseID) == "" {
		idx := contextItemIndex(p.RelevantContext, itemID)
		if idx < 0 {
			return ErrInvalidPlan{Reason: "global context item not found: " + itemID}
		}
		item.Scope = RelevantContextScopeGlobal
		item.PhaseID = ""
		p.RelevantContext[idx] = item
		return nil
	}
	phaseIdx := phaseIndex(p.Phases, phaseID)
	if phaseIdx < 0 {
		return ErrPhaseNotFound{PlanID: p.ID, PhaseID: phaseID}
	}
	idx := contextItemIndex(p.Phases[phaseIdx].RelevantContext, itemID)
	if idx < 0 {
		return ErrInvalidPlan{Reason: "phase context item not found: " + itemID}
	}
	item.Scope = RelevantContextScopePhase
	item.PhaseID = p.Phases[phaseIdx].ID
	p.Phases[phaseIdx].RelevantContext[idx] = item
	return nil
}

func (s *service) RemoveRelevantContext(ctx context.Context, idOrSlug string, workspace WorkspaceScope, phaseID, itemID string) (Plan, error) {
	p, err := s.Get(ctx, idOrSlug, workspace)
	if err != nil {
		return Plan{}, err
	}
	if err := applyRelevantContextRemove(&p, phaseID, itemID); err != nil {
		return Plan{}, err
	}
	return s.saveRecomputed(ctx, p)
}

func (s *service) RemoveRelevantContextWithImpact(ctx context.Context, idOrSlug string, workspace WorkspaceScope, phaseID, itemID string, allowRegression bool) (Plan, MutationImpact, error) {
	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()
	p, err := s.Get(ctx, idOrSlug, workspace)
	if err != nil {
		return Plan{}, MutationImpact{}, err
	}
	beforeStatus := p.Status
	beforeReport := planmodel.AssessPlanQuality(p, "")
	if err := applyRelevantContextRemove(&p, phaseID, itemID); err != nil {
		return Plan{}, MutationImpact{}, err
	}
	impact, err := assessMutationImpactFromReport(beforeStatus, beforeReport, p, allowRegression)
	if err != nil {
		return Plan{}, impact, err
	}
	updated, err := s.saveRecomputed(ctx, p)
	return updated, impact, err
}

func applyRelevantContextRemove(p *Plan, phaseID, itemID string) error {
	itemID = strings.TrimSpace(itemID)
	if itemID == "" {
		return ErrInvalidPlan{Reason: "context item id is required"}
	}
	if strings.TrimSpace(phaseID) == "" {
		idx := contextItemIndex(p.RelevantContext, itemID)
		if idx < 0 {
			return ErrInvalidPlan{Reason: "global context item not found: " + itemID}
		}
		p.RelevantContext = append(p.RelevantContext[:idx], p.RelevantContext[idx+1:]...)
		return nil
	}
	phaseIdx := phaseIndex(p.Phases, phaseID)
	if phaseIdx < 0 {
		return ErrPhaseNotFound{PlanID: p.ID, PhaseID: phaseID}
	}
	idx := contextItemIndex(p.Phases[phaseIdx].RelevantContext, itemID)
	if idx < 0 {
		return ErrInvalidPlan{Reason: "phase context item not found: " + itemID}
	}
	p.Phases[phaseIdx].RelevantContext = append(p.Phases[phaseIdx].RelevantContext[:idx], p.Phases[phaseIdx].RelevantContext[idx+1:]...)
	return nil
}

func (s *service) ListReferences(ctx context.Context, idOrSlug string, workspace WorkspaceScope, phaseID string) ([]Reference, error) {
	p, err := s.Get(ctx, idOrSlug, workspace)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(phaseID) == "" {
		return referencesWithEffectiveIDs(p.References), nil
	}
	idx := phaseIndex(p.Phases, phaseID)
	if idx < 0 {
		return nil, ErrPhaseNotFound{PlanID: p.ID, PhaseID: phaseID}
	}
	return referencesWithEffectiveIDs(p.Phases[idx].References), nil
}

func (s *service) AddReference(ctx context.Context, idOrSlug string, workspace WorkspaceScope, phaseID string, ref Reference, allowRegression bool) (Plan, MutationImpact, error) {
	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()
	if strings.TrimSpace(ref.Target) == "" {
		return Plan{}, MutationImpact{}, ErrInvalidPlan{Reason: "reference target is required"}
	}
	p, err := s.Get(ctx, idOrSlug, workspace)
	if err != nil {
		return Plan{}, MutationImpact{}, err
	}
	beforeStatus := p.Status
	beforeReport := planmodel.AssessPlanQuality(p, "")
	ref.ID = uuid.NewString()
	ref = authoredReference(ref)
	if strings.TrimSpace(phaseID) == "" {
		for _, existing := range p.References {
			if existing.Kind == ref.Kind && existing.Target == ref.Target {
				return Plan{}, MutationImpact{}, ErrInvalidPlan{Reason: "duplicate global reference target: " + ref.Target}
			}
		}
		p.References = append(p.References, ref)
	} else {
		idx := phaseIndex(p.Phases, phaseID)
		if idx < 0 {
			return Plan{}, MutationImpact{}, ErrPhaseNotFound{PlanID: p.ID, PhaseID: phaseID}
		}
		for _, existing := range p.Phases[idx].References {
			if existing.Kind == ref.Kind && existing.Target == ref.Target {
				return Plan{}, MutationImpact{}, ErrInvalidPlan{Reason: "duplicate phase reference target: " + ref.Target}
			}
		}
		p.Phases[idx].References = append(p.Phases[idx].References, ref)
	}
	impact, err := assessMutationImpactFromReport(beforeStatus, beforeReport, p, allowRegression)
	if err != nil {
		return Plan{}, impact, err
	}
	updated, err := s.saveRecomputed(ctx, p)
	return updated, impact, err
}

func (s *service) UpdateReference(ctx context.Context, idOrSlug string, workspace WorkspaceScope, phaseID, referenceID string, ref Reference) (Plan, error) {
	p, err := s.Get(ctx, idOrSlug, workspace)
	if err != nil {
		return Plan{}, err
	}
	if err := applyReferenceUpdate(&p, phaseID, referenceID, ref); err != nil {
		return Plan{}, err
	}
	return s.saveRecomputed(ctx, p)
}

func (s *service) UpdateReferenceWithImpact(ctx context.Context, idOrSlug string, workspace WorkspaceScope, phaseID, referenceID string, ref Reference, allowRegression bool) (Plan, MutationImpact, error) {
	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()
	p, err := s.Get(ctx, idOrSlug, workspace)
	if err != nil {
		return Plan{}, MutationImpact{}, err
	}
	beforeStatus := p.Status
	beforeReport := planmodel.AssessPlanQuality(p, "")
	if err := applyReferenceUpdate(&p, phaseID, referenceID, ref); err != nil {
		return Plan{}, MutationImpact{}, err
	}
	impact, err := assessMutationImpactFromReport(beforeStatus, beforeReport, p, allowRegression)
	if err != nil {
		return Plan{}, impact, err
	}
	updated, err := s.saveRecomputed(ctx, p)
	return updated, impact, err
}

func applyReferenceUpdate(p *Plan, phaseID, referenceID string, ref Reference) error {
	referenceID = strings.TrimSpace(referenceID)
	if referenceID == "" {
		return ErrInvalidPlan{Reason: "reference id is required"}
	}
	if strings.TrimSpace(ref.ID) == "" {
		ref.ID = referenceID
	}
	if strings.TrimSpace(ref.ID) == "" || isSyntheticItemID(ref.ID) {
		ref.ID = uuid.NewString()
	}
	if strings.TrimSpace(phaseID) == "" {
		idx := referenceIndex(p.References, referenceID)
		if idx < 0 {
			return ErrInvalidPlan{Reason: "global reference not found: " + referenceID}
		}
		p.References[idx] = authoredReference(ref)
		return nil
	}
	phaseIdx := phaseIndex(p.Phases, phaseID)
	if phaseIdx < 0 {
		return ErrPhaseNotFound{PlanID: p.ID, PhaseID: phaseID}
	}
	idx := referenceIndex(p.Phases[phaseIdx].References, referenceID)
	if idx < 0 {
		return ErrInvalidPlan{Reason: "phase reference not found: " + referenceID}
	}
	p.Phases[phaseIdx].References[idx] = authoredReference(ref)
	return nil
}

func (s *service) RemoveReference(ctx context.Context, idOrSlug string, workspace WorkspaceScope, phaseID, referenceID string) (Plan, error) {
	p, err := s.Get(ctx, idOrSlug, workspace)
	if err != nil {
		return Plan{}, err
	}
	if err := applyReferenceRemove(&p, phaseID, referenceID); err != nil {
		return Plan{}, err
	}
	return s.saveRecomputed(ctx, p)
}

func (s *service) RemoveReferenceWithImpact(ctx context.Context, idOrSlug string, workspace WorkspaceScope, phaseID, referenceID string, allowRegression bool) (Plan, MutationImpact, error) {
	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()
	p, err := s.Get(ctx, idOrSlug, workspace)
	if err != nil {
		return Plan{}, MutationImpact{}, err
	}
	beforeStatus := p.Status
	beforeReport := planmodel.AssessPlanQuality(p, "")
	if err := applyReferenceRemove(&p, phaseID, referenceID); err != nil {
		return Plan{}, MutationImpact{}, err
	}
	impact, err := assessMutationImpactFromReport(beforeStatus, beforeReport, p, allowRegression)
	if err != nil {
		return Plan{}, impact, err
	}
	updated, err := s.saveRecomputed(ctx, p)
	return updated, impact, err
}

func applyReferenceRemove(p *Plan, phaseID, referenceID string) error {
	referenceID = strings.TrimSpace(referenceID)
	if referenceID == "" {
		return ErrInvalidPlan{Reason: "reference id is required"}
	}
	if strings.TrimSpace(phaseID) == "" {
		idx := referenceIndex(p.References, referenceID)
		if idx < 0 {
			return ErrInvalidPlan{Reason: "global reference not found: " + referenceID}
		}
		p.References = append(p.References[:idx], p.References[idx+1:]...)
		return nil
	}
	phaseIdx := phaseIndex(p.Phases, phaseID)
	if phaseIdx < 0 {
		return ErrPhaseNotFound{PlanID: p.ID, PhaseID: phaseID}
	}
	idx := referenceIndex(p.Phases[phaseIdx].References, referenceID)
	if idx < 0 {
		return ErrInvalidPlan{Reason: "phase reference not found: " + referenceID}
	}
	p.Phases[phaseIdx].References = append(p.Phases[phaseIdx].References[:idx], p.Phases[phaseIdx].References[idx+1:]...)
	return nil
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

func phaseIndex(phases []Phase, phaseID string) int {
	phaseID = strings.TrimSpace(phaseID)
	for i, ph := range phases {
		if ph.ID == phaseID || fmt.Sprint(ph.Order) == phaseID {
			return i
		}
	}
	return -1
}

func contextWithEffectiveIDs(items []RelevantContextItem) []RelevantContextItem {
	out := make([]RelevantContextItem, 0, len(items))
	for i, item := range items {
		if strings.TrimSpace(item.ID) == "" {
			item.ID = syntheticItemID(i)
		}
		out = append(out, item)
	}
	return out
}

func referencesWithEffectiveIDs(refs []Reference) []Reference {
	out := make([]Reference, 0, len(refs))
	for i, ref := range refs {
		if strings.TrimSpace(ref.ID) == "" {
			ref.ID = syntheticItemID(i)
		}
		out = append(out, ref)
	}
	return out
}

func contextItemIndex(items []RelevantContextItem, id string) int {
	id = strings.TrimSpace(id)
	for i, item := range items {
		if item.ID == id || syntheticItemID(i) == id {
			return i
		}
	}
	return -1
}

func referenceIndex(refs []Reference, id string) int {
	id = strings.TrimSpace(id)
	for i, ref := range refs {
		if ref.ID == id || syntheticItemID(i) == id {
			return i
		}
	}
	return -1
}

func syntheticItemID(index int) string {
	return fmt.Sprintf("item-%d", index+1)
}

func isSyntheticItemID(id string) bool {
	return syntheticItemIDPattern.MatchString(strings.TrimSpace(id))
}

func authoredReference(ref Reference) Reference {
	ref.Resolution = ""
	ref.Staleness = ""
	ref.ChangeFactor = 0
	return ref
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
		Definitions             []PlanDefinition      `json:"definitions"`
		ChangeBoundary          ChangeBoundary        `json:"change_boundary"`
		RegressionAnchor        RegressionAnchor      `json:"regression_anchor"`
		BaselineSet             BaselineSetIntent     `json:"baseline_set"`
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
		Definitions:             p.Definitions,
		ChangeBoundary:          p.ChangeBoundary,
		RegressionAnchor:        p.RegressionAnchor,
		BaselineSet:             p.BaselineSet,
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
