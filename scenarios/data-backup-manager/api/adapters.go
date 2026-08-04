package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	runsH "data-backup-manager/handlers/runs"

	auditsint "data-backup-manager/internal/audits"
	coverageint "data-backup-manager/internal/coverage"
	destint "data-backup-manager/internal/destinations"
	discoveryint "data-backup-manager/internal/discovery"
	plansint "data-backup-manager/internal/plans"
	restoresint "data-backup-manager/internal/restores"
	runsint "data-backup-manager/internal/runs"
	safetyint "data-backup-manager/internal/safety"
	scenariospecint "data-backup-manager/internal/scenariospec"
	schedint "data-backup-manager/internal/scheduler"
	sourcesint "data-backup-manager/internal/sources"
	targetsint "data-backup-manager/internal/targets"
)

// This file holds the composition-root adapters that satisfy the narrow
// reader/effect seams the runs domain and the scheduler declare, backed by the
// concrete sibling services. Keeping the adapters here (package main) means the
// domains stay decoupled — runs and scheduler depend only on their own
// interfaces, and main is the single place that knows the whole graph.

// planLookup adapts plans.Service to runs.PlanLookup.
type planLookup struct{ svc plansint.Service }

func (a planLookup) PlanForRun(ctx context.Context, planID string) (runsint.PlanForRun, error) {
	p, err := a.svc.Get(ctx, planID)
	if err != nil {
		return runsint.PlanForRun{}, err
	}
	return runsint.PlanForRun{
		ID:             p.ID,
		TargetIDs:      p.TargetIDs,
		DestinationIDs: p.DestinationIDs,
		KeepLatest:     int(p.KeepLatest),
	}, nil
}

// targetLookup adapts targets.Service to runs.TargetLookup.
type targetLookup struct{ svc targetsint.Service }

func (a targetLookup) TargetForRun(ctx context.Context, targetID string) (runsint.TargetForRun, error) {
	t, err := a.svc.Get(ctx, targetID)
	if err != nil {
		return runsint.TargetForRun{}, err
	}
	return runsint.TargetForRun{ID: t.ID, Owner: t.Owner, Name: t.Name, Kind: t.SourceKind, Locator: t.Locator}, nil
}

func (a targetLookup) ActiveTargetIDs(ctx context.Context) ([]string, error) {
	targets, err := a.svc.List(ctx, "", catalogScanLimit)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(targets))
	for _, t := range targets {
		out = append(out, t.ID)
	}
	return out, nil
}

// destinationLookup adapts destinations.Service to runs.DestinationLookup.
type destinationLookup struct{ svc destint.Service }

func (a destinationLookup) DestinationForRun(ctx context.Context, destID string) (runsint.DestinationForRun, error) {
	d, err := a.svc.GetDestination(ctx, destID)
	if err != nil {
		return runsint.DestinationForRun{}, err
	}
	return runsint.DestinationForRun{ID: d.ID, Name: d.Name}, nil
}

func (a destinationLookup) WouldBlock(ctx context.Context, destID string, pendingBytes int64) (bool, string, error) {
	return a.svc.WouldBlock(ctx, destID, pendingBytes)
}

// restoreTargetLookup adapts targets.Service to restores.TargetLookup.
type restoreTargetLookup struct{ svc targetsint.Service }

func (a restoreTargetLookup) TargetForRestore(ctx context.Context, targetID string) (restoresint.TargetForRestore, error) {
	t, err := a.svc.Get(ctx, targetID)
	if err != nil {
		return restoresint.TargetForRestore{}, err
	}
	return restoresint.TargetForRestore{ID: t.ID, Kind: t.SourceKind, Locator: t.Locator}, nil
}

// restoreDestinationLookup adapts destinations.Service to restores.DestinationLookup.
type restoreDestinationLookup struct{ svc destint.Service }

func (a restoreDestinationLookup) DestinationForRestore(ctx context.Context, destID string) (restoresint.DestinationForRestore, error) {
	d, err := a.svc.GetDestination(ctx, destID)
	if err != nil {
		return restoresint.DestinationForRestore{}, err
	}
	return restoresint.DestinationForRestore{ID: d.ID, Name: d.Name}, nil
}

// auditTargetLookup adapts targets.Service to audits.TargetLookup.
type auditTargetLookup struct{ svc targetsint.Service }

func (a auditTargetLookup) TargetForAudit(ctx context.Context, targetID string) (auditsint.TargetForAudit, error) {
	t, err := a.svc.Get(ctx, targetID)
	if err != nil {
		return auditsint.TargetForAudit{}, err
	}
	return auditsint.TargetForAudit{ID: t.ID, Kind: t.SourceKind, Locator: t.Locator}, nil
}

// auditDestinationLookup adapts destinations.Service to audits.DestinationLookup.
type auditDestinationLookup struct{ svc destint.Service }

func (a auditDestinationLookup) DestinationForAudit(ctx context.Context, destID string) (auditsint.DestinationForAudit, error) {
	d, err := a.svc.GetDestination(ctx, destID)
	if err != nil {
		return auditsint.DestinationForAudit{}, err
	}
	return auditsint.DestinationForAudit{ID: d.ID, Name: d.Name}, nil
}

// verifiedLookup adapts restores.Service to the runs handler's VerifiedLookup
// seam: it rolls the latest successful verify per target into a map so
// ListTargetStatus can report proven-restorable posture in one call. This is
// the composition seam that keeps the runs domain from importing restores.
type verifiedLookup struct{ svc restoresint.Service }

func (a verifiedLookup) LastVerifiedByTarget(ctx context.Context) (map[string]runsH.VerifiedInfo, error) {
	statuses, err := a.svc.LastVerifiedByTarget(ctx, nil)
	if err != nil {
		return nil, err
	}
	out := make(map[string]runsH.VerifiedInfo, len(statuses))
	for _, s := range statuses {
		out[s.TargetID] = runsH.VerifiedInfo{LastVerifiedAt: s.LastVerifiedAt, SnapshotID: s.SnapshotID}
	}
	return out, nil
}

// --- Baseline Modes safety substrate seams --------------------------------
//
// The safety domain composes the destinations/targets/plans/runs services to
// provision the reserved safety destination and take pre-promote scenario
// snapshots. These adapters back its narrow seams with the concrete services so
// the safety orchestration stays decoupled and unit-testable.

// safetyDestinations adapts destinations.Service to safety.Destinations.
type safetyDestinations struct{ svc destint.Service }

func (a safetyDestinations) List(ctx context.Context) ([]safetyint.DestinationRef, error) {
	ds, err := a.svc.ListDestinations(ctx, catalogScanLimit)
	if err != nil {
		return nil, err
	}
	out := make([]safetyint.DestinationRef, 0, len(ds))
	for _, d := range ds {
		out = append(out, safetyDestRef(d))
	}
	return out, nil
}

func (a safetyDestinations) CreateSafety(ctx context.Context, name, location string, capBytes int64) (safetyint.DestinationRef, error) {
	d, err := a.svc.CreateDestination(ctx, destint.CreateInput{
		Name:      name,
		Backend:   destint.BackendFilesystem,
		Location:  location,
		CapBytes:  capBytes,
		CapPolicy: destint.CapPolicyAlertBlock,
	})
	if err != nil {
		return safetyint.DestinationRef{}, err
	}
	return safetyDestRef(d), nil
}

func safetyDestRef(d destint.Destination) safetyint.DestinationRef {
	return safetyint.DestinationRef{
		ID:                 d.ID,
		Name:               d.Name,
		Location:           d.Location,
		RepositoryLocation: d.RepositoryLocation,
	}
}

// safetyTargets adapts targets.Service to safety.Targets.
type safetyTargets struct{ svc targetsint.Service }

func (a safetyTargets) ListByOwner(ctx context.Context, owner string) ([]safetyint.TargetRef, error) {
	ts, err := a.svc.List(ctx, owner, catalogScanLimit)
	if err != nil {
		return nil, err
	}
	out := make([]safetyint.TargetRef, 0, len(ts))
	for _, t := range ts {
		out = append(out, safetyint.TargetRef{ID: t.ID, Owner: t.Owner, Name: t.Name})
	}
	return out, nil
}

func (a safetyTargets) Register(ctx context.Context, owner, name string, kind sourcesint.SourceKind, locator string) error {
	_, err := a.svc.Register(ctx, targetsint.RegisterInput{Owner: owner, Name: name, SourceKind: kind, Locator: locator})
	return err
}

// safetyScenarioInspector adapts scenariospec.Inspector to
// safety.ScenarioInspector, mapping the package's Facts onto the safety seam's
// ScenarioFacts so the orchestration never imports scenariospec directly.
type safetyScenarioInspector struct{ insp *scenariospecint.Inspector }

func (a safetyScenarioInspector) Inspect(ctx context.Context, scenario string) (safetyint.ScenarioFacts, error) {
	f, err := a.insp.Inspect(ctx, scenario)
	if err != nil {
		return safetyint.ScenarioFacts{}, err
	}
	return safetyint.ScenarioFacts{
		UsesPostgres:   f.UsesPostgres,
		DataDir:        f.DataDir,
		DataDirPresent: f.DataDirPresent,
	}, nil
}

// safetyPlans adapts plans.Service to safety.Plans. The ephemeral plan is
// always created/updated with AllowIncompleteCoverage=true (it deliberately
// scopes to one scenario's targets, not the full default-coverage set) and an
// empty schedule (manual-only — the scheduler never fires it).
type safetyPlans struct{ svc plansint.Service }

func (a safetyPlans) List(ctx context.Context) ([]safetyint.PlanRef, error) {
	ps, err := a.svc.List(ctx, catalogScanLimit)
	if err != nil {
		return nil, err
	}
	out := make([]safetyint.PlanRef, 0, len(ps))
	for _, p := range ps {
		out = append(out, safetyint.PlanRef{ID: p.ID, Name: p.Name})
	}
	return out, nil
}

func (a safetyPlans) Create(ctx context.Context, name string, targetIDs, destinationIDs []string, keepLatest int32) (safetyint.PlanRef, error) {
	p, err := a.svc.Create(ctx, plansint.CreateInput{
		Name:                    name,
		TargetIDs:               targetIDs,
		DestinationIDs:          destinationIDs,
		Schedule:                "",
		KeepLatest:              keepLatest,
		Enabled:                 true,
		AllowIncompleteCoverage: true,
	})
	if err != nil {
		return safetyint.PlanRef{}, err
	}
	return safetyint.PlanRef{ID: p.ID, Name: p.Name}, nil
}

func (a safetyPlans) Update(ctx context.Context, id, name string, targetIDs, destinationIDs []string, keepLatest int32) (safetyint.PlanRef, error) {
	p, err := a.svc.Update(ctx, plansint.UpdateInput{
		ID:                      id,
		Name:                    name,
		TargetIDs:               targetIDs,
		DestinationIDs:          destinationIDs,
		Schedule:                "",
		KeepLatest:              keepLatest,
		Enabled:                 true,
		AllowIncompleteCoverage: true,
	})
	if err != nil {
		return safetyint.PlanRef{}, err
	}
	return safetyint.PlanRef{ID: p.ID, Name: p.Name}, nil
}

// safetyRuns adapts runs.Service to safety.Runs (manual-triggered safety run +
// the run reads PopulateShadow needs to find each target's restorable snapshot).
type safetyRuns struct{ svc runsint.Service }

func (a safetyRuns) TriggerManual(ctx context.Context, planID string) (safetyint.RunRef, error) {
	r, err := a.svc.TriggerRun(ctx, planID, runsint.TriggerManual)
	if err != nil {
		return safetyint.RunRef{}, err
	}
	return safetyint.RunRef{ID: r.ID, PlanID: r.PlanID, Status: string(r.Status)}, nil
}

func (a safetyRuns) GetRun(ctx context.Context, runID string) (safetyint.RunDetail, error) {
	r, err := a.svc.GetRun(ctx, runID)
	if err != nil {
		return safetyint.RunDetail{}, err
	}
	return safetyRunDetail(r), nil
}

// safetyRunListLimit bounds the run scan LatestTerminalRun does; the ephemeral
// safety plan accrues few runs, so the newest terminal one is well within this.
const safetyRunListLimit = 50

func (a safetyRuns) LatestTerminalRun(ctx context.Context, planID string) (safetyint.RunDetail, bool, error) {
	// ListRuns returns newest-first, so the first terminal run is the latest.
	rs, err := a.svc.ListRuns(ctx, planID, safetyRunListLimit)
	if err != nil {
		return safetyint.RunDetail{}, false, err
	}
	for _, r := range rs {
		if isTerminalRunStatus(r.Status) {
			return safetyRunDetail(r), true, nil
		}
	}
	return safetyint.RunDetail{}, false, nil
}

func safetyRunDetail(r runsint.Run) safetyint.RunDetail {
	outcomes := make([]safetyint.TargetSnapshot, 0, len(r.Outcomes))
	for _, o := range r.Outcomes {
		outcomes = append(outcomes, safetyint.TargetSnapshot{
			TargetID:      o.TargetID,
			DestinationID: o.DestinationID,
			SnapshotID:    o.SnapshotID,
			Succeeded:     o.Status == runsint.OutcomeSucceeded,
		})
	}
	return safetyint.RunDetail{
		ID:       r.ID,
		PlanID:   r.PlanID,
		Status:   string(r.Status),
		Terminal: isTerminalRunStatus(r.Status),
		Outcomes: outcomes,
	}
}

func isTerminalRunStatus(s runsint.RunStatus) bool {
	switch s {
	case runsint.RunCompleted, runsint.RunPartialFailed, runsint.RunFailed:
		return true
	default:
		return false
	}
}

// safetyRestores adapts restores.Service to safety.Restores — PopulateShadow
// restores each target's safety snapshot into its shadow namespace.
type safetyRestores struct{ svc restoresint.Service }

func (a safetyRestores) RestoreTarget(ctx context.Context, targetID, destinationID, snapshotID, location string) (safetyint.RestoreRef, error) {
	r, err := a.svc.RestoreTarget(ctx, targetID, destinationID, snapshotID, location)
	if err != nil {
		return safetyint.RestoreRef{}, err
	}
	return safetyint.RestoreRef{ID: r.ID, Status: string(r.Status)}, nil
}

// catalogScanLimit is a generous upper bound for the "list everything" catalog
// reads the discovery filters need; the catalogs are small (one row per
// registered target/destination), so a fixed cap avoids paging ceremony.
const catalogScanLimit = 100000

// discoveryTargetCatalog adapts targets.Service to discovery.TargetCatalog so
// already-registered sources are filtered out of target suggestions.
type discoveryTargetCatalog struct{ svc targetsint.Service }

func (a discoveryTargetCatalog) ListAll(ctx context.Context) ([]discoveryint.ExistingTarget, error) {
	ts, err := a.svc.List(ctx, "", catalogScanLimit)
	if err != nil {
		return nil, err
	}
	out := make([]discoveryint.ExistingTarget, 0, len(ts))
	for _, t := range ts {
		out = append(out, discoveryint.ExistingTarget{Owner: t.Owner, Name: t.Name, Locator: t.Locator})
	}
	return out, nil
}

// discoveryDestCatalog adapts destinations.Service to discovery.DestinationCatalog
// so already-used locations are filtered out of destination suggestions.
type discoveryDestCatalog struct{ svc destint.Service }

func (a discoveryDestCatalog) ListAll(ctx context.Context) ([]discoveryint.ExistingDestination, error) {
	ds, err := a.svc.ListDestinations(ctx, catalogScanLimit)
	if err != nil {
		return nil, err
	}
	out := make([]discoveryint.ExistingDestination, 0, len(ds))
	for _, d := range ds {
		out = append(out, discoveryint.ExistingDestination{Location: d.Location})
	}
	return out, nil
}

// discoveryProtectedPaths satisfies discovery.ProtectedPaths. The destinations
// service's own protectedRoot (just SCENARIO_DATA_DIR) is too narrow for
// destination filtering, so discovery's set is the resolved runtime root + every
// registered destination location + every registered target locator (Contract
// Decision D4). A volume overlapping any of these is flagged unsafe.
type discoveryProtectedPaths struct {
	runtimeRoot string
	targets     targetsint.Service
	dests       destint.Service
}

func (a discoveryProtectedPaths) ProtectedPaths(ctx context.Context) ([]string, error) {
	paths := make([]string, 0, 8)
	if strings.TrimSpace(a.runtimeRoot) != "" {
		paths = append(paths, a.runtimeRoot)
	}
	ds, err := a.dests.ListDestinations(ctx, catalogScanLimit)
	if err != nil {
		return nil, err
	}
	for _, d := range ds {
		if d.Location != "" {
			paths = append(paths, d.Location)
		}
	}
	ts, err := a.targets.List(ctx, "", catalogScanLimit)
	if err != nil {
		return nil, err
	}
	for _, t := range ts {
		if t.Locator != "" {
			paths = append(paths, t.Locator)
		}
	}
	return paths, nil
}

// --- Coverage composition seams -------------------------------------------
//
// Coverage reads every fact through a seam and owns no scanner/catalog logic;
// these adapters back its seams with the concrete sibling services. Discovery
// is the single authority for "what durable state is worth protecting", so both
// the coverage report and the plan guard read suggestions from it.

// coverageSuggestions adapts discovery.Service to coverage.SuggestionSource.
// Discovery already filters out registered and dismissed suggestions, so the
// recommended/sensitive split is the only classification coverage adds.
type coverageSuggestions struct{ svc discoveryint.Service }

func (a coverageSuggestions) ListTargetSuggestions(ctx context.Context) ([]coverageint.Suggestion, error) {
	sugs, err := a.svc.ListTargetSuggestions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]coverageint.Suggestion, 0, len(sugs))
	for _, s := range sugs {
		out = append(out, coverageint.Suggestion{
			ID:          s.ID,
			Owner:       s.Owner,
			Name:        s.Name,
			SourceKind:  s.SourceKind,
			Locator:     s.Locator,
			Rationale:   s.Rationale,
			ApproxBytes: s.ApproxBytes,
			Sensitive:   s.Sensitive,
			Warning:     discoverySuggestionWarning(s.Sensitive, len(s.Findings)),
		})
	}
	return out, nil
}

// discoverySensitiveWarning mirrors the discovery handler's sensitive warning so
// the coverage surface shows the same caution without importing the handler.
func discoverySensitiveWarning(sensitive bool) string {
	if !sensitive {
		return ""
	}
	return "Includes credentials/tokens — review before backing up; restoring stale tokens can silently break auth."
}

func discoverySuggestionWarning(sensitive bool, findings int) string {
	warning := discoverySensitiveWarning(sensitive)
	if findings == 0 {
		return warning
	}
	if warning == "" {
		return "Owner storage metadata has declaration findings; review before registering this target."
	}
	return warning + " Owner storage metadata also has declaration findings; review before registering."
}

// coverageTargetCatalog adapts targets.Service to coverage.TargetCatalog: it
// lists registered targets and registers accepted suggestions (idempotent
// upsert, locators only — no content read).
type coverageTargetCatalog struct{ svc targetsint.Service }

func (a coverageTargetCatalog) List(ctx context.Context) ([]coverageint.CatalogTarget, error) {
	ts, err := a.svc.List(ctx, "", catalogScanLimit)
	if err != nil {
		return nil, err
	}
	out := make([]coverageint.CatalogTarget, 0, len(ts))
	for _, t := range ts {
		out = append(out, coverageint.CatalogTarget{
			ID:         t.ID,
			Owner:      t.Owner,
			Name:       t.Name,
			SourceKind: t.SourceKind,
			Locator:    t.Locator,
		})
	}
	return out, nil
}

func (a coverageTargetCatalog) Register(ctx context.Context, in coverageint.RegisterInput) (coverageint.CatalogTarget, error) {
	t, err := a.svc.Register(ctx, targetsint.RegisterInput{
		Owner:      in.Owner,
		Name:       in.Name,
		SourceKind: in.SourceKind,
		Locator:    in.Locator,
	})
	if err != nil {
		return coverageint.CatalogTarget{}, err
	}
	return coverageint.CatalogTarget{
		ID:         t.ID,
		Owner:      t.Owner,
		Name:       t.Name,
		SourceKind: t.SourceKind,
		Locator:    t.Locator,
	}, nil
}

// coveragePlanCatalog adapts plans.Service to coverage.PlanCatalog: it rolls
// every plan's target membership into the set of planned target ids.
type coveragePlanCatalog struct{ svc plansint.Service }

func (a coveragePlanCatalog) PlannedTargetIDs(ctx context.Context) (map[string]struct{}, error) {
	ps, err := a.svc.List(ctx, catalogScanLimit)
	if err != nil {
		return nil, err
	}
	out := make(map[string]struct{})
	for _, p := range ps {
		for _, id := range p.TargetIDs {
			out[id] = struct{}{}
		}
	}
	return out, nil
}

// coverageRunStatus adapts runs.Service to coverage.RunStatusSource: the last
// successful backup time per target.
type coverageRunStatus struct{ svc runsint.Service }

func (a coverageRunStatus) LastSuccessByTarget(ctx context.Context, targetIDs []string) (map[string]time.Time, error) {
	statuses, err := a.svc.ListTargetStatus(ctx, targetIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[string]time.Time, len(statuses))
	for _, s := range statuses {
		if !s.LastSuccessAt.IsZero() {
			out[s.TargetID] = s.LastSuccessAt
		}
	}
	return out, nil
}

// coverageVerified adapts restores.Service to coverage.VerifiedSource: the last
// verified-restore time per target.
type coverageVerified struct{ svc restoresint.Service }

func (a coverageVerified) LastVerifiedByTarget(ctx context.Context, targetIDs []string) (map[string]time.Time, error) {
	statuses, err := a.svc.LastVerifiedByTarget(ctx, targetIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[string]time.Time, len(statuses))
	for _, s := range statuses {
		if !s.LastVerifiedAt.IsZero() {
			out[s.TargetID] = s.LastVerifiedAt
		}
	}
	return out, nil
}

// planCoverageGuard adapts the coverage service to plans.CoverageGuard. The
// plans service consults it before persisting a create/update so a plan cannot
// silently omit non-sensitive recommended default coverage. It reads only the
// non-sensitive recommendations — sensitive suggestions never block a plan.
type planCoverageGuard struct{ svc coverageint.Service }

func (a planCoverageGuard) UnregisteredDefaultTargets(ctx context.Context) ([]plansint.MissingTarget, error) {
	recs, err := a.svc.UnregisteredDefaultTargets(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]plansint.MissingTarget, 0, len(recs))
	for _, r := range recs {
		out = append(out, plansint.MissingTarget{Owner: r.Owner, Name: r.Name, Locator: r.Locator})
	}
	return out, nil
}

// runTrigger adapts runs.Service to scheduler.RunTrigger (scheduler-fired runs).
type runTrigger struct{ svc runsint.Service }

func (a runTrigger) TriggerRun(ctx context.Context, planID string) error {
	_, err := a.svc.TriggerRun(ctx, planID, runsint.TriggerScheduler)
	return err
}

// planSource adapts plans.Service to scheduler.PlanSource.
type planSource struct{ svc plansint.Service }

func (a planSource) SchedulablePlans(ctx context.Context) ([]schedint.DuePlan, error) {
	ps, err := a.svc.SchedulablePlans(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]schedint.DuePlan, 0, len(ps))
	for _, p := range ps {
		out = append(out, schedint.DuePlan{ID: p.ID, Schedule: p.Schedule, Enabled: p.Enabled})
	}
	return out, nil
}

// nextScheduleAdapter satisfies runs.NextScheduleSource by joining the plans
// service (which plans target a given schedule) with the in-process scheduler
// (when each plan next fires). The scheduler is set after construction because
// it depends on the runs service (runTrigger), which in turn carries this
// adapter — a late bind breaks the cycle. The pointer is always populated
// before any request reaches ListTargetStatus.
type nextScheduleAdapter struct {
	plans plansint.Service
	sched *schedint.Scheduler
}

func (a *nextScheduleAdapter) NextScheduledByTarget(ctx context.Context) (map[string]time.Time, error) {
	if a.sched == nil {
		return nil, nil
	}
	plans, err := a.plans.SchedulablePlans(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]time.Time)
	for _, p := range plans {
		if !p.Enabled || strings.TrimSpace(p.Schedule) == "" {
			continue
		}
		next, ok := a.sched.NextFire(p.ID, p.Schedule)
		if !ok {
			continue
		}
		for _, tid := range p.TargetIDs {
			if cur, exists := out[tid]; !exists || next.Before(cur) {
				out[tid] = next
			}
		}
	}
	return out, nil
}

// logEventSink is the production runs.EventSink: it logs backup outcomes for
// platform monitoring (infra-health / system-monitor) to pick up. A routed
// notification sink (OT-P1-006) replaces this later.
type logEventSink struct{ logger *log.Logger }

func (s logEventSink) BackupOutcome(_ context.Context, ev runsint.RunOutcomeEvent) {
	s.logger.Printf("backup-outcome run=%s plan=%s status=%s succeeded=%d failed=%d blocked=%d",
		ev.RunID, ev.PlanID, ev.Status, ev.Succeeded, ev.Failed, ev.Blocked)
}

// BackupPostureDegraded satisfies health.PostureEventSink: it logs a degraded
// backup posture so platform monitoring can flag overdue/failed backups.
func (s logEventSink) BackupPostureDegraded(_ context.Context, detail string) {
	s.logger.Printf("backup-posture degraded: %s", detail)
}

// runConcurrency returns how many target×destination units a single run
// executes in parallel. Configurable via DBM_RUN_CONCURRENCY (a positive
// integer); default 4. An invalid value is an error so misconfiguration is
// loud rather than silently ignored.
func runConcurrency() (int, error) {
	const def = 4
	raw, ok := lookupEnvTrimmed("DBM_RUN_CONCURRENCY")
	if !ok {
		return def, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("DBM_RUN_CONCURRENCY must be a positive integer, got %q", raw)
	}
	return n, nil
}

// overdueAfter returns the age past which a target's last success counts as
// overdue. Configurable via DBM_OVERDUE_AFTER (a Go duration); default 36h.
func overdueAfter() (time.Duration, error) {
	const def = 36 * time.Hour
	if raw, ok := lookupEnvTrimmed("DBM_OVERDUE_AFTER"); ok {
		d, err := time.ParseDuration(raw)
		if err != nil || d <= 0 {
			return 0, fmt.Errorf("DBM_OVERDUE_AFTER must be a positive Go duration, got %q", raw)
		}
		return d, nil
	}
	return def, nil
}

// backupPosture adapts runs.Service to health.BackupPosture. It counts the
// per-target Overdue flag the runs service computes (last run failed/partial,
// never succeeded, or last success older than DBM_OVERDUE_AFTER) so the health
// rollup and `runs status` share one freshness rule rather than each applying
// their own.
type backupPosture struct {
	runs runsint.Service
}

func (a backupPosture) OverdueOrFailed(ctx context.Context) (bool, string, error) {
	statuses, err := a.runs.ListTargetStatus(ctx, nil)
	if err != nil {
		return false, "", err
	}
	const maxNamed = 10
	var named []string
	bad := 0
	for _, s := range statuses {
		if !s.Overdue {
			continue
		}
		bad++
		if len(named) < maxNamed {
			named = append(named, fmt.Sprintf("%s (%s)", s.TargetID, overdueReason(s)))
		}
	}
	if bad == 0 {
		return false, "", nil
	}
	detail := fmt.Sprintf("%d of %d target(s) overdue or last run failed: %s", bad, len(statuses), strings.Join(named, ", "))
	if bad > len(named) {
		detail += fmt.Sprintf(", +%d more", bad-len(named))
	}
	return true, detail, nil
}

// overdueReason classifies why a target is overdue, for the /health detail
// enumeration. Mirrors runs.isOverdue's branches.
func overdueReason(s runsint.TargetStatus) string {
	switch s.LastRunStatus {
	case runsint.RunFailed:
		return "last run failed"
	case runsint.RunPartialFailed:
		return "last run partial-failed"
	}
	if s.LastSuccessAt.IsZero() {
		return "never backed up"
	}
	return "stale"
}

// startScheduler runs the in-process scheduler on a ticker in a background
// goroutine. Interval comes from DBM_SCHEDULER_INTERVAL (a Go duration);
// default 60s. Setting it to "0" disables the ticker (the manual/on-demand
// trigger path still works via the RunsService). With no schedulable plans,
// each tick is a no-op.
func startScheduler(ctx context.Context, sched *schedint.Scheduler, logger *log.Logger) error {
	interval := 60 * time.Second
	if raw, ok := lookupEnvTrimmed("DBM_SCHEDULER_INTERVAL"); ok {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return fmt.Errorf("DBM_SCHEDULER_INTERVAL must be a Go duration, got %q", raw)
		} else if d == 0 {
			logger.Printf("scheduler: disabled via DBM_SCHEDULER_INTERVAL=0")
			return nil
		} else {
			interval = d
		}
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := sched.Tick(ctx); err != nil {
					logger.Printf("scheduler tick: %v", err)
				}
			}
		}
	}()
	return nil
}
