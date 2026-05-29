package migration

import (
	"context"
	"sort"
	"strings"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"

	"github.com/google/uuid"
)

// Analytics event kinds emitted by the tracker. Kept as plain strings in
// this domain (the durable analytics_events enum is generalized when the
// migration handler is wired — Phase 5); the recorder seam is nil-safe so
// production can run without analytics, mirroring conflicts.NewService.
const (
	EventFindingIngested  = "finding_ingested"
	EventFindingResolved  = "finding_resolved"
	EventFindingValidated = "finding_validated"
	EventFindingRegressed = "finding_regressed"
	EventMigrationCreated = "migration_created"
	EventMigrationClosed  = "migration_closed"
)

// AnalyticsRecorder is the slim seam between tracker lifecycle transitions
// and the analytics event log. Production wires an adapter (Phase 5); tests
// use a fake; nil means transitions are silent.
type AnalyticsRecorder interface {
	Record(ctx context.Context, scenario, kind, stableID string, payload map[string]any)
}

// Service is the application-layer surface for the migration tracker.
type Service interface {
	// Create opens a migration for a scenario and ingests the initial
	// findings (all start `detected`). name is optional.
	Create(ctx context.Context, scenario, name string, findings []*architecturev1.ArchitectureFinding) (Status, error)
	// Status returns the migration plus every tracked finding and rollups.
	Status(ctx context.Context, id string) (Status, error)
	// Next returns the prioritized, dependency-aware worklist of open
	// findings (regressions first, then cycles, then severity).
	Next(ctx context.Context, id string) ([]Finding, error)
	// Resolve marks one finding resolved with an operator note.
	Resolve(ctx context.Context, id, stableID, note string) (Finding, error)
	// Reaudit reconciles a fresh findings set against the tracked set by
	// stable id: absent→validated, present→stay, (re)appeared→regression.
	Reaudit(ctx context.Context, id string, fresh []*architecturev1.ArchitectureFinding) (ReauditResult, error)
	// Close marks the migration closed.
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
	m := Migration{
		ID:       uuid.NewString(),
		Scenario: scenario,
		Name:     strings.TrimSpace(name),
		Status:   MigrationOpen,
	}
	if err := s.repo.CreateMigration(ctx, m); err != nil {
		return Status{}, err
	}
	s.record(ctx, scenario, EventMigrationCreated, "", map[string]any{"migration_id": m.ID, "name": m.Name})

	for _, pf := range findings {
		f := fromProto(scenario, pf)
		if f.StableID == "" {
			continue
		}
		if err := s.repo.UpsertFinding(ctx, m.ID, f); err != nil {
			return Status{}, err
		}
		s.record(ctx, scenario, EventFindingIngested, f.StableID, map[string]any{
			"source": f.Source, "code": f.Code, "severity": f.Severity,
		})
	}
	return s.Status(ctx, m.ID)
}

func (s *service) Status(ctx context.Context, id string) (Status, error) {
	m, err := s.repo.GetMigration(ctx, id)
	if err != nil {
		return Status{}, err
	}
	findings, err := s.repo.ListFindings(ctx, id)
	if err != nil {
		return Status{}, err
	}
	return buildStatus(m, findings), nil
}

func (s *service) Next(ctx context.Context, id string) ([]Finding, error) {
	if _, err := s.repo.GetMigration(ctx, id); err != nil {
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
	sortWorklist(open)
	return open, nil
}

func (s *service) Resolve(ctx context.Context, id, stableID, note string) (Finding, error) {
	stableID = strings.TrimSpace(stableID)
	if stableID == "" {
		return Finding{}, ErrInvalidInput{Reason: "finding stable id is required"}
	}
	m, err := s.repo.GetMigration(ctx, id)
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
	s.record(ctx, m.Scenario, EventFindingResolved, f.StableID, map[string]any{"note": f.ResolutionNote})
	return s.repo.GetFinding(ctx, id, stableID)
}

func (s *service) Reaudit(ctx context.Context, id string, fresh []*architecturev1.ArchitectureFinding) (ReauditResult, error) {
	m, err := s.repo.GetMigration(ctx, id)
	if err != nil {
		return ReauditResult{}, err
	}
	tracked, err := s.repo.ListFindings(ctx, id)
	if err != nil {
		return ReauditResult{}, err
	}

	// Index the fresh photograph by canonical afid.
	freshByID := make(map[string]Finding, len(fresh))
	for _, pf := range fresh {
		f := fromProto(m.Scenario, pf)
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
				s.record(ctx, m.Scenario, EventFindingRegressed, f.StableID, map[string]any{"reason": "reappeared_after_terminal"})
				result.Regressions = append(result.Regressions, f)
			} else {
				result.StillOpen = append(result.StillOpen, f)
			}
			continue
		}
		// Absent from the fresh photograph → fixed. Validate it (unless it
		// was already validated/committed).
		switch f.Status {
		case StatusValidated, StatusCommitted:
			// already terminal-and-gone; leave as is
		default:
			f.Status = StatusValidated
			f.Regressed = false
			if err := s.repo.UpsertFinding(ctx, id, f); err != nil {
				return ReauditResult{}, err
			}
			s.record(ctx, m.Scenario, EventFindingValidated, f.StableID, nil)
			result.Validated = append(result.Validated, f)
		}
	}

	// 2. Fresh findings not previously tracked → brand-new, introduced
	// while the migration was in flight. Add as detected + flag regression.
	for sid, f := range freshByID {
		if _, known := trackedByID[sid]; known {
			continue
		}
		f.Status = StatusDetected
		f.Regressed = true
		if err := s.repo.UpsertFinding(ctx, id, f); err != nil {
			return ReauditResult{}, err
		}
		s.record(ctx, m.Scenario, EventFindingRegressed, sid, map[string]any{"reason": "new_during_migration"})
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
	m, err := s.repo.GetMigration(ctx, id)
	if err != nil {
		return Status{}, err
	}
	if err := s.repo.UpdateMigrationStatus(ctx, id, MigrationClosed); err != nil {
		return Status{}, err
	}
	s.record(ctx, m.Scenario, EventMigrationClosed, "", map[string]any{"migration_id": id})
	return s.Status(ctx, id)
}

// buildStatus computes the rollup projection.
func buildStatus(m Migration, findings []Finding) Status {
	st := Status{
		Migration:  m,
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

// severityRank orders severities for the worklist (higher = more urgent).
func severityRank(sev string) int {
	switch sev {
	case "blocker":
		return 4
	case "error":
		return 3
	case "warn":
		return 2
	case "info":
		return 1
	default:
		return 0
	}
}

// sortWorklist orders open findings for the agent: regressions first (the
// refactor broke something), then import cycles (they block dependent
// moves — fix first), then by severity desc, then code, then stable id for
// determinism.
func sortWorklist(findings []Finding) {
	isCycle := func(f Finding) bool { return strings.HasPrefix(f.Code, "cycle") }
	sort.SliceStable(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a.Regressed != b.Regressed {
			return a.Regressed
		}
		if ac, bc := isCycle(a), isCycle(b); ac != bc {
			return ac
		}
		if ra, rb := severityRank(a.Severity), severityRank(b.Severity); ra != rb {
			return ra > rb
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		return a.StableID < b.StableID
	})
}
