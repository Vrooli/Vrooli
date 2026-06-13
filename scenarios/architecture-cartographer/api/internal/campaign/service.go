package campaign

import (
	"context"
	"strings"

	campaignv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/campaign"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"

	"github.com/google/uuid"
)

// Analytics event kinds emitted by the tracker. Kept as plain strings in
// this domain; the recorder seam is nil-safe so production can run without
// analytics, mirroring conflicts.NewService.
const (
	EventFindingIngested  = "finding_ingested"
	EventFindingResolved  = "finding_resolved"
	EventFindingValidated = "finding_validated"
	EventFindingRegressed = "finding_regressed"
	EventCampaignCreated  = "campaign_created"
	EventCampaignClosed   = "campaign_closed"
)

// AnalyticsRecorder is the slim seam between tracker lifecycle transitions
// and the analytics event log. Production wires an adapter; tests use a
// fake; nil means transitions are silent.
type AnalyticsRecorder interface {
	Record(ctx context.Context, scenario, kind, stableID string, payload map[string]any)
}

// Service is the application-layer surface for the campaign tracker.
type Service interface {
	// Create opens a campaign for a scenario and ingests the initial
	// findings (all start `detected`). name is optional.
	Create(ctx context.Context, scenario, name string, findings []*architecturev1.ArchitectureFinding) (Status, error)
	// Status returns the campaign plus every tracked finding and rollups.
	Status(ctx context.Context, id string) (Status, error)
	// List returns the campaign headers for a scenario (newest first); an
	// empty scenario returns every campaign.
	List(ctx context.Context, scenario string) ([]Campaign, error)
	// Next returns the profile-ranked worklist of open findings. profile
	// selects the ordering strategy (BALANCED when unspecified).
	Next(ctx context.Context, id string, profile campaignv1.RankProfile) ([]Finding, error)
	// Resolve marks one finding resolved with an operator note.
	Resolve(ctx context.Context, id, stableID, note string) (Finding, error)
	// Reaudit reconciles a fresh findings set against the tracked set by
	// stable id: absent→validated, present→stay, (re)appeared→regression.
	// coveredSources scopes which sources the fresh photograph actually
	// observed (findingid tokens); a tracked finding whose source is not
	// covered is left untouched and reported in NotReaudited. An EMPTY
	// coveredSources means all sources are covered (full-suite semantics).
	Reaudit(ctx context.Context, id string, fresh []*architecturev1.ArchitectureFinding, coveredSources []string) (ReauditResult, error)
	// Close marks the campaign closed.
	Close(ctx context.Context, id string) (Status, error)
}

type service struct {
	repo     Repository
	recorder AnalyticsRecorder
}

// NewService constructs the production Service without analytics (silent).
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// NewServiceWithAnalytics constructs the Service with an analytics recorder
// so every lifecycle transition emits an event.
func NewServiceWithAnalytics(repo Repository, recorder AnalyticsRecorder) Service {
	return &service{repo: repo, recorder: recorder}
}

var _ Service = (*service)(nil)

func (s *service) record(ctx context.Context, scenario, kind, stableID string, payload map[string]any) {
	if s.recorder == nil {
		return
	}
	s.recorder.Record(ctx, scenario, kind, stableID, payload)
}

func (s *service) Create(ctx context.Context, scenario, name string, findings []*architecturev1.ArchitectureFinding) (Status, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return Status{}, ErrInvalidInput{Reason: "scenario is required"}
	}
	// Defense in depth behind the CLI check: an empty campaign tracks nothing
	// and is never useful.
	if !hasStampableFinding(scenario, findings) {
		return Status{}, ErrInvalidInput{Reason: "no findings to ingest — an empty campaign is never useful"}
	}
	c := Campaign{
		ID:       uuid.NewString(),
		Scenario: scenario,
		Name:     strings.TrimSpace(name),
		Status:   CampaignOpen,
	}
	if err := s.repo.CreateCampaign(ctx, c); err != nil {
		return Status{}, err
	}
	s.record(ctx, scenario, EventCampaignCreated, "", map[string]any{"campaign_id": c.ID, "name": c.Name})

	for _, pf := range findings {
		f := fromProto(scenario, pf)
		if f.StableID == "" {
			continue
		}
		if err := s.repo.UpsertFinding(ctx, c.ID, f); err != nil {
			return Status{}, err
		}
		s.record(ctx, scenario, EventFindingIngested, f.StableID, map[string]any{
			"source": f.Source, "code": f.Code, "severity": f.Severity,
		})
	}
	return s.Status(ctx, c.ID)
}

func (s *service) Status(ctx context.Context, id string) (Status, error) {
	c, err := s.repo.GetCampaign(ctx, id)
	if err != nil {
		return Status{}, err
	}
	findings, err := s.repo.ListFindings(ctx, id)
	if err != nil {
		return Status{}, err
	}
	return buildStatus(c, findings), nil
}

func (s *service) List(ctx context.Context, scenario string) ([]Campaign, error) {
	return s.repo.ListCampaigns(ctx, strings.TrimSpace(scenario))
}

func (s *service) Next(ctx context.Context, id string, profile campaignv1.RankProfile) ([]Finding, error) {
	if _, err := s.repo.GetCampaign(ctx, id); err != nil {
		return nil, err
	}
	findings, err := s.repo.ListFindings(ctx, id)
	if err != nil {
		return nil, err
	}
	var open []Finding
	for _, f := range findings {
		if f.Status.IsOpen() {
			open = append(open, f)
		}
	}
	Order(open, profile)
	return open, nil
}

func (s *service) Resolve(ctx context.Context, id, stableID, note string) (Finding, error) {
	stableID = strings.TrimSpace(stableID)
	if stableID == "" {
		return Finding{}, ErrInvalidInput{Reason: "finding stable id is required"}
	}
	c, err := s.repo.GetCampaign(ctx, id)
	if err != nil {
		return Finding{}, err
	}
	f, err := s.repo.GetFinding(ctx, id, stableID)
	if err != nil {
		return Finding{}, err
	}
	f.Status = StatusResolved
	f.ResolutionNote = strings.TrimSpace(note)
	if err := s.repo.UpsertFinding(ctx, id, f); err != nil {
		return Finding{}, err
	}
	s.record(ctx, c.Scenario, EventFindingResolved, f.StableID, map[string]any{"note": f.ResolutionNote})
	return s.repo.GetFinding(ctx, id, stableID)
}

func (s *service) Reaudit(ctx context.Context, id string, fresh []*architecturev1.ArchitectureFinding, coveredSources []string) (ReauditResult, error) {
	c, err := s.repo.GetCampaign(ctx, id)
	if err != nil {
		return ReauditResult{}, err
	}
	// An empty fresh photograph would mass-validate every tracked finding;
	// reject it rather than corrupt the campaign.
	if !hasStampableFinding(c.Scenario, fresh) {
		return ReauditResult{}, ErrInvalidInput{Reason: "reaudit received zero findings — an empty photograph would falsely validate everything"}
	}
	tracked, err := s.repo.ListFindings(ctx, id)
	if err != nil {
		return ReauditResult{}, err
	}

	// Build the coverage set. Empty ⇒ all sources covered (full-suite).
	covered := make(map[string]struct{}, len(coveredSources))
	for _, src := range coveredSources {
		if src = strings.TrimSpace(src); src != "" {
			covered[src] = struct{}{}
		}
	}
	isCovered := func(source string) bool {
		if len(covered) == 0 {
			return true
		}
		_, ok := covered[source]
		return ok
	}

	// Index the fresh photograph by canonical afid.
	freshByID := make(map[string]Finding, len(fresh))
	for _, pf := range fresh {
		f := fromProto(c.Scenario, pf)
		if f.StableID == "" {
			continue
		}
		freshByID[f.StableID] = f
	}
	trackedByID := make(map[string]Finding, len(tracked))
	for _, f := range tracked {
		trackedByID[f.StableID] = f
	}

	var result ReauditResult

	// 1. Tracked findings: reconcile against the fresh photograph.
	for _, f := range tracked {
		if _, present := freshByID[f.StableID]; present {
			// Still present. A finding the agent already marked terminal
			// that reappears is a regression (the fix didn't hold).
			if f.Status.IsTerminal() {
				f.Status = StatusDetected
				f.Regressed = true
				if err := s.repo.UpsertFinding(ctx, id, f); err != nil {
					return ReauditResult{}, err
				}
				s.record(ctx, c.Scenario, EventFindingRegressed, f.StableID, map[string]any{"reason": "reappeared_after_terminal"})
				result.Regressions = append(result.Regressions, f)
			} else {
				result.StillOpen = append(result.StillOpen, f)
			}
			continue
		}
		// Absent from the fresh photograph. Only treat absence as evidence of
		// a fix when this finding's source was actually covered by the audit;
		// otherwise its phase did not run and we must leave it untouched.
		if !isCovered(f.Source) {
			result.NotReaudited = append(result.NotReaudited, f)
			continue
		}
		// Covered and absent → fixed. Validate it (unless it was already
		// validated/committed).
		switch f.Status {
		case StatusValidated, StatusCommitted:
			// already terminal-and-gone; leave as is
		default:
			f.Status = StatusValidated
			f.Regressed = false
			if err := s.repo.UpsertFinding(ctx, id, f); err != nil {
				return ReauditResult{}, err
			}
			s.record(ctx, c.Scenario, EventFindingValidated, f.StableID, nil)
			result.Validated = append(result.Validated, f)
		}
	}

	// 2. Fresh findings not previously tracked → brand-new, introduced
	// while the campaign was in flight. Add as detected + flag regression.
	for sid, f := range freshByID {
		if _, known := trackedByID[sid]; known {
			continue
		}
		f.Status = StatusDetected
		f.Regressed = true
		if err := s.repo.UpsertFinding(ctx, id, f); err != nil {
			return ReauditResult{}, err
		}
		s.record(ctx, c.Scenario, EventFindingRegressed, sid, map[string]any{"reason": "new_during_campaign"})
		result.Regressions = append(result.Regressions, f)
	}

	status, err := s.Status(ctx, id)
	if err != nil {
		return ReauditResult{}, err
	}
	result.Status = status
	return result, nil
}

func (s *service) Close(ctx context.Context, id string) (Status, error) {
	c, err := s.repo.GetCampaign(ctx, id)
	if err != nil {
		return Status{}, err
	}
	if err := s.repo.UpdateCampaignStatus(ctx, id, CampaignClosed); err != nil {
		return Status{}, err
	}
	s.record(ctx, c.Scenario, EventCampaignClosed, "", map[string]any{"campaign_id": id})
	return s.Status(ctx, id)
}

// buildStatus computes the rollup projection.
func buildStatus(c Campaign, findings []Finding) Status {
	st := Status{
		Campaign:   c,
		Findings:   findings,
		Total:      len(findings),
		BySeverity: map[string]int{},
		ByStatus:   map[string]int{},
	}
	for _, f := range findings {
		st.BySeverity[f.Severity]++
		st.ByStatus[string(f.Status)]++
		if f.Status.IsOpen() {
			st.Open++
		}
		if f.Status == StatusResolved {
			st.Resolved++
		}
		if f.Status == StatusValidated {
			st.Validated++
		}
		if f.Regressed {
			st.Regressions++
		}
	}
	return st
}
