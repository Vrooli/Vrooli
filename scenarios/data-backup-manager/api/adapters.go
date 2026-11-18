package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	runsH "data-backup-manager/handlers/runs"

	auditsint "data-backup-manager/internal/audits"
	coverageint "data-backup-manager/internal/coverage"
	readinessint "data-backup-manager/internal/destinationreadiness"
	destint "data-backup-manager/internal/destinations"
	topology "data-backup-manager/internal/destinationtopology"
	discoveryint "data-backup-manager/internal/discovery"
	drillsint "data-backup-manager/internal/drills"
	engineint "data-backup-manager/internal/engine"
	plansint "data-backup-manager/internal/plans"
	restoresint "data-backup-manager/internal/restores"
	runsint "data-backup-manager/internal/runs"
	safetyint "data-backup-manager/internal/safety"
	scenariospecint "data-backup-manager/internal/scenariospec"
	schedint "data-backup-manager/internal/scheduler"
	sourcesint "data-backup-manager/internal/sources"
	targetsint "data-backup-manager/internal/targets"
)

func derefDeviceIdentity(identity *readinessint.DeviceIdentity) readinessint.DeviceIdentity {
	if identity == nil {
		return readinessint.DeviceIdentity{}
	}
	return *identity
}

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
		KeepHourly:     int(p.KeepHourly), KeepDaily: int(p.KeepDaily), KeepWeekly: int(p.KeepWeekly), KeepMonthly: int(p.KeepMonthly), KeepAnnual: int(p.KeepAnnual),
	}, nil
}

// targetLookup adapts targets.Service to runs.TargetLookup.
type targetLookup struct{ svc targetsint.Service }

// planCriticalTargetPolicy adapts the target classification owned by the
// targets catalog to the plans service. Critical plans must not be created
// from an unclassified target.
type planCriticalTargetPolicy struct{ svc targetsint.Service }

func (a planCriticalTargetPolicy) IsCritical(ctx context.Context, targetID string) (bool, error) {
	t, err := a.svc.Get(ctx, targetID)
	if err != nil {
		return false, err
	}
	return t.Critical, nil
}

type planCriticalDestinationPolicy struct {
	targets      targetsint.Service
	destinations destint.Service
	readiness    *readinessint.Service
	// hostID is supplied only by a trusted host identity source. Empty means
	// local placement can be observed, but cannot be compared across hosts.
	hostID string
}

type drillPlanLookup struct{ svc plansint.Service }

func (a drillPlanLookup) PlanForDrill(ctx context.Context, id string) (drillsint.Plan, error) {
	p, err := a.svc.Get(ctx, id)
	if err != nil {
		return drillsint.Plan{}, err
	}
	return drillsint.Plan{ID: p.ID, TargetIDs: p.TargetIDs, DestinationIDs: p.DestinationIDs, Enabled: p.Enabled, DrillSchedule: p.RecoveryDrillSchedule}, nil
}

func (a drillPlanLookup) SchedulableDrillPlans(ctx context.Context) ([]drillsint.Plan, error) {
	ps, err := a.svc.SchedulablePlans(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]drillsint.Plan, 0, len(ps))
	for _, p := range ps {
		out = append(out, drillsint.Plan{ID: p.ID, TargetIDs: p.TargetIDs, DestinationIDs: p.DestinationIDs, Enabled: p.Enabled, DrillSchedule: p.DrillSchedule})
	}
	return out, nil
}

type drillSnapshotLookup struct{ svc runsint.Service }

func (a drillSnapshotLookup) LatestSuccessfulSnapshot(ctx context.Context, planID, targetID, destinationID string) (drillsint.Snapshot, bool, error) {
	runs, err := a.svc.ListRuns(ctx, planID, 1000)
	if err != nil {
		return drillsint.Snapshot{}, false, err
	}
	var best drillsint.Snapshot
	for _, r := range runs {
		for _, o := range r.Outcomes {
			if o.TargetID != targetID || o.DestinationID != destinationID || o.Status != runsint.OutcomeSucceeded || strings.TrimSpace(o.SnapshotID) == "" {
				continue
			}
			at := r.FinishedAt
			if at.After(best.CompletedAt) {
				best = drillsint.Snapshot{ID: o.SnapshotID, CompletedAt: at}
			}
		}
	}
	return best, best.ID != "", nil
}

type drillRestoreRunner struct{ svc restoresint.Service }

type drillDestinationReadiness struct {
	destinations destint.Service
	readiness    interface {
		Analyze(context.Context, readinessint.AnalyzeInput) (readinessint.Report, error)
	}
}

func (a drillDestinationReadiness) ReadyForDrill(ctx context.Context, destinationID string) (bool, string, error) {
	d, err := a.destinations.GetDestination(ctx, destinationID)
	if err != nil {
		return false, "", err
	}
	if d.BackendKind != destint.BackendFilesystem || a.readiness == nil {
		return true, "", nil
	}
	input := readinessint.AnalyzeInput{DestinationID: d.ID, Location: d.Location}
	if d.DeviceIdentity != nil {
		input.ExpectedDevice = *d.DeviceIdentity
	}
	report, err := a.readiness.Analyze(ctx, input)
	if err != nil {
		return false, "", err
	}
	// A drill reads the snapshot and writes only to scratch, so an unknown
	// native filesystem-dirty state does not block it. A known failure
	// (unmounted, read-only, dirty/needs-check) still does.
	if report.OverallSeverity != readinessint.SeverityFail {
		return true, "", nil
	}
	for _, check := range report.Checks {
		if check.Severity == readinessint.SeverityFail {
			return false, check.Message, nil
		}
	}
	return false, fmt.Sprintf("destination readiness is %s", report.OverallSeverity), nil
}

func (a drillRestoreRunner) VerifyTarget(ctx context.Context, targetID, destinationID, snapshotID string) (drillsint.Restore, error) {
	r, err := a.svc.VerifyTarget(ctx, targetID, destinationID, snapshotID)
	if err != nil {
		return drillsint.Restore{}, err
	}
	return drillsint.Restore{ID: r.ID, Status: string(r.Status), Error: r.Error}, nil
}

func (a drillRestoreRunner) GetRestore(ctx context.Context, id string) (drillsint.Restore, error) {
	r, err := a.svc.GetRestore(ctx, id)
	if err != nil {
		return drillsint.Restore{}, err
	}
	return drillsint.Restore{ID: r.ID, Status: string(r.Status), Error: r.Error}, nil
}

func (a planCriticalDestinationPolicy) Validate(ctx context.Context, tier plansint.ProtectionTier, targetIDs, destinationIDs []string) error {
	if tier == plansint.TierFullPrimary {
		return nil
	}
	if tier == plansint.TierCriticalSecondary && len(destinationIDs) < 2 {
		return plansint.ErrInvalidPlan{Field: "destination_ids", Reason: "critical_secondary requires at least two independent destinations"}
	}
	locations := make([]string, 0, len(destinationIDs))
	for _, id := range destinationIDs {
		d, err := a.destinations.GetDestination(ctx, id)
		if err != nil {
			return plansint.ErrInvalidPlan{Field: "destination_ids", Reason: fmt.Sprintf("cannot validate destination %q: %v", id, err)}
		}
		if d.BackendKind == destint.BackendFilesystem && a.readiness != nil {
			report, err := a.readiness.Analyze(ctx, readinessint.AnalyzeInput{
				DestinationID:         d.ID,
				ExpectedDevice:        derefDeviceIdentity(d.DeviceIdentity),
				Location:              d.Location,
				CrossPlatformRequired: true,
			})
			if err != nil {
				return plansint.ErrInvalidPlan{Field: "destination_ids", Reason: fmt.Sprintf("destination %q readiness could not be established: %v", id, err)}
			}
			if report.OverallSeverity == readinessint.SeverityFail {
				failures := make([]string, 0)
				for _, check := range report.Checks {
					if check.Severity == readinessint.SeverityFail {
						failures = append(failures, check.Code+": "+check.Message)
					}
				}
				return plansint.ErrInvalidPlan{Field: "destination_ids", Reason: fmt.Sprintf("destination %q is not suitable for critical protection: %s", id, strings.Join(failures, "; "))}
			}
		}
		location := strings.TrimSpace(d.Location)
		if location == "" {
			return plansint.ErrInvalidPlan{Field: "destination_ids", Reason: fmt.Sprintf("destination %q has no independent location", id)}
		}
		locations = append(locations, filepath.Clean(location))
	}
	for i := range locations {
		for j := i + 1; j < len(locations); j++ {
			if pathsOverlap(locations[i], locations[j]) {
				return plansint.ErrInvalidPlan{Field: "destination_ids", Reason: fmt.Sprintf("destinations %q and %q share an overlapping filesystem root; choose independent destinations", destinationIDs[i], destinationIDs[j])}
			}
		}
	}
	// A critical repository must not sit inside the source it is protecting.
	for _, targetID := range targetIDs {
		t, err := a.targets.Get(ctx, targetID)
		if err != nil {
			return plansint.ErrInvalidPlan{Field: "target_ids", Reason: fmt.Sprintf("cannot validate target %q against destinations: %v", targetID, err)}
		}
		if t.Locator == "" {
			continue
		}
		for _, location := range locations {
			if pathsOverlap(filepath.Clean(t.Locator), location) {
				return plansint.ErrInvalidPlan{Field: "destination_ids", Reason: fmt.Sprintf("destination overlaps critical target %q; separate source and recovery roots", targetID)}
			}
		}
	}
	return nil
}

// Assess is deliberately read-only. It reports what the service can prove
// about the selected topology and keeps uncertainty explicit. A clean path is
// not enough to claim physical independence: when the platform cannot expose
// a stable volume identity, the plan remains usable but carries a warning.
func (a planCriticalDestinationPolicy) Assess(ctx context.Context, tier plansint.ProtectionTier, targetIDs, destinationIDs []string) (plansint.DestinationRiskReport, error) {
	report := plansint.DestinationRiskReport{PhysicallyIndependent: true}
	if len(destinationIDs) == 0 {
		return plansint.DestinationRiskReport{Warnings: []string{"no destinations selected; physical independence is not proven"}}, nil
	}

	locations := make([]string, 0, len(destinationIDs))
	evidence := make([]topology.Evidence, 0, len(destinationIDs))
	for _, id := range destinationIDs {
		d, err := a.destinations.GetDestination(ctx, id)
		if err != nil {
			return plansint.DestinationRiskReport{}, fmt.Errorf("destination %q: %w", id, err)
		}
		location := strings.TrimSpace(d.Location)
		if location == "" {
			report.PhysicallyIndependent = false
			evidence = append(evidence, topology.Evidence{Endpoint: topology.EndpointUnknown})
			report.Warnings = append(report.Warnings, fmt.Sprintf("destination %q has no location; physical independence is not proven", id))
			continue
		}
		locations = append(locations, filepath.Clean(location))

		if d.BackendKind != destint.BackendFilesystem {
			report.PhysicallyIndependent = false
			evidence = append(evidence, topology.Evidence{Endpoint: topology.EndpointUnknown})
			report.Warnings = append(report.Warnings, fmt.Sprintf("destination %q uses %s; endpoint hosting evidence is unavailable", id, d.BackendKind))
			continue
		}
		observed := topology.Evidence{HostID: a.hostID, Endpoint: topology.EndpointLocal}
		if a.readiness == nil {
			report.PhysicallyIndependent = false
			evidence = append(evidence, observed)
			report.Warnings = append(report.Warnings, fmt.Sprintf("destination %q has no volume identity inspector; physical independence is not proven", id))
			continue
		}
		readiness, err := a.readiness.Analyze(ctx, readinessint.AnalyzeInput{
			DestinationID:         d.ID,
			ExpectedDevice:        derefDeviceIdentity(d.DeviceIdentity),
			Location:              location,
			CrossPlatformRequired: true,
		})
		if err != nil {
			report.PhysicallyIndependent = false
			evidence = append(evidence, observed)
			report.Warnings = append(report.Warnings, fmt.Sprintf("destination %q identity unavailable: %v", id, err))
			continue
		}
		// A device serial is the only currently observed field that names the
		// physical device. A filesystem UUID or device path can identify a
		// volume or assignment, so neither is promoted to physical identity.
		observed.DeviceID = strings.TrimSpace(readiness.Identity.Serial)
		evidence = append(evidence, observed)
		if observed.DeviceID == "" {
			report.PhysicallyIndependent = false
			report.Warnings = append(report.Warnings, fmt.Sprintf("destination %q has no stable volume identity; physical independence is not proven", id))
		}
		if readiness.OverallSeverity == readinessint.SeverityFail {
			report.PhysicallyIndependent = false
			for _, check := range readiness.Checks {
				if check.Severity == readinessint.SeverityFail {
					report.Warnings = append(report.Warnings, fmt.Sprintf("destination %q readiness %s: %s", id, check.Code, check.Message))
				}
			}
		} else if readiness.OverallSeverity == readinessint.SeverityWarning {
			for _, check := range readiness.Checks {
				if check.Severity == readinessint.SeverityWarning {
					report.Warnings = append(report.Warnings, fmt.Sprintf("destination %q suitability %s: %s", id, check.Code, check.Message))
				}
			}
		}
	}

	for i := range locations {
		for j := i + 1; j < len(locations); j++ {
			if pathsOverlap(locations[i], locations[j]) {
				report.PhysicallyIndependent = false
				report.Warnings = append(report.Warnings, fmt.Sprintf("destinations %q and %q share an overlapping filesystem root", destinationIDs[i], destinationIDs[j]))
			}
		}
	}
	for i := range evidence {
		for j := i + 1; j < len(evidence); j++ {
			relation := topology.Compare(evidence[i], evidence[j])
			if relation != topology.RelationIndependent {
				report.PhysicallyIndependent = false
				report.Warnings = append(report.Warnings, fmt.Sprintf("destinations %q and %q topology is %s; independent placement is not proven", destinationIDs[i], destinationIDs[j], relation))
			}
		}
	}

	for _, targetID := range targetIDs {
		t, err := a.targets.Get(ctx, targetID)
		if err != nil {
			return plansint.DestinationRiskReport{}, fmt.Errorf("target %q: %w", targetID, err)
		}
		if strings.TrimSpace(t.Locator) == "" {
			continue
		}
		for i, location := range locations {
			if pathsOverlap(filepath.Clean(t.Locator), location) {
				report.PhysicallyIndependent = false
				report.Warnings = append(report.Warnings, fmt.Sprintf("destination %q overlaps protected target %q", destinationIDs[i], targetID))
			}
		}
	}
	if tier == plansint.TierFullPrimary && len(report.Warnings) == 0 {
		// A full-primary plan is not an independence claim against another
		// tier, but its selected root has passed the same risk checks.
		report.PhysicallyIndependent = true
	}
	return report, nil
}

func samePhysicalVolume(a, b readinessint.DeviceIdentity) bool {
	if a.UUID != "" && b.UUID != "" && strings.EqualFold(a.UUID, b.UUID) {
		return true
	}
	if a.Serial != "" && b.Serial != "" && a.Serial == b.Serial {
		return true
	}
	return a.DevicePath != "" && b.DevicePath != "" && a.Mountpoint != "" && b.Mountpoint != "" &&
		strings.EqualFold(filepath.Clean(a.DevicePath), filepath.Clean(b.DevicePath)) &&
		strings.EqualFold(filepath.Clean(a.Mountpoint), filepath.Clean(b.Mountpoint))
}

func pathsOverlap(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	return a == b || strings.HasPrefix(a, b+string(filepath.Separator)) || strings.HasPrefix(b, a+string(filepath.Separator))
}

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
	// Planning reads catalog identity; it must not require write-ready state or
	// perform a metadata repair. The write gate re-resolves strictly (or admits
	// a probe-backed degraded write) immediately before the snapshot.
	d, err := a.svc.GetDestination(ctx, destID)
	if err != nil {
		return runsint.DestinationForRun{}, err
	}
	return runsint.DestinationForRun{
		ID:             d.ID,
		Name:           d.Name,
		BackendKind:    string(d.BackendKind),
		Location:       d.Location,
		DeviceIdentity: d.DeviceIdentity,
	}, nil
}

func (a destinationLookup) WithResolvedDestination(ctx context.Context, destID string, use func(runsint.DestinationForRun) error) error {
	return a.WithResolvedDestinationForWrite(ctx, destID, false, use)
}

func (a destinationLookup) WithResolvedDestinationForRead(ctx context.Context, destID string, use func(runsint.DestinationForRun) error) error {
	if use == nil {
		return nil
	}
	return a.svc.WithResolvedDestinationForRead(ctx, destID, func(d destint.Destination) error {
		return use(runsint.DestinationForRun{ID: d.ID, Name: d.Name, BackendKind: string(d.BackendKind), Location: d.Location, DeviceIdentity: d.DeviceIdentity})
	})
}

func (a destinationLookup) WithResolvedDestinationForWrite(ctx context.Context, destID string, allowDegraded bool, use func(runsint.DestinationForRun) error) error {
	if use == nil {
		return nil
	}
	return a.svc.WithResolvedDestinationForWrite(ctx, destID, allowDegraded, func(d destint.Destination) error {
		return use(runsint.DestinationForRun{ID: d.ID, Name: d.Name, BackendKind: string(d.BackendKind), Location: d.Location, DeviceIdentity: d.DeviceIdentity})
	})
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
	// A restore/verify reads the snapshot from the destination and writes only
	// to a caller-supplied target or scratch. Planning reads catalog identity;
	// the read gate proves access immediately before the Kopia operation.
	d, err := a.svc.GetDestination(ctx, destID)
	if err != nil {
		return restoresint.DestinationForRestore{}, err
	}
	return restoresint.DestinationForRestore{ID: d.ID, Name: d.Name}, nil
}

func (a restoreDestinationLookup) WithResolvedDestination(ctx context.Context, destID string, use func(restoresint.DestinationForRestore) error) error {
	return a.WithResolvedDestinationForRead(ctx, destID, use)
}

func (a restoreDestinationLookup) WithResolvedDestinationForRead(ctx context.Context, destID string, use func(restoresint.DestinationForRestore) error) error {
	if use == nil {
		return nil
	}
	return a.svc.WithResolvedDestinationForRead(ctx, destID, func(d destint.Destination) error {
		return use(restoresint.DestinationForRestore{ID: d.ID, Name: d.Name})
	})
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

func (a auditDestinationLookup) WithResolvedDestination(ctx context.Context, destID string, use func(auditsint.DestinationForAudit) error) error {
	return a.WithResolvedDestinationForRead(ctx, destID, use)
}

func (a auditDestinationLookup) WithResolvedDestinationForRead(ctx context.Context, destID string, use func(auditsint.DestinationForAudit) error) error {
	if use == nil {
		return nil
	}
	return a.svc.WithResolvedDestinationForRead(ctx, destID, func(d destint.Destination) error {
		return use(auditsint.DestinationForAudit{ID: d.ID, Name: d.Name})
	})
}

// verifiedLookup combines enabled plan bindings with destination-specific
// restore receipts. A target is marked verified only when every enabled copy
// has a successful scratch-restore verification.
type verifiedLookup struct {
	svc   restoresint.Service
	plans plansint.Service
}

func (a verifiedLookup) LastVerifiedByTarget(ctx context.Context) (map[string]runsH.VerifiedInfo, error) {
	if a.plans == nil {
		return nil, fmt.Errorf("enabled plan bindings are required to compute target verification")
	}
	plans, err := a.plans.List(ctx, catalogScanLimit)
	if err != nil {
		return nil, err
	}
	bindings := map[string]map[string]struct{}{}
	for _, plan := range plans {
		if !plan.Enabled {
			continue
		}
		for _, targetID := range plan.TargetIDs {
			if bindings[targetID] == nil {
				bindings[targetID] = map[string]struct{}{}
			}
			for _, destinationID := range plan.DestinationIDs {
				if strings.TrimSpace(destinationID) != "" {
					bindings[targetID][destinationID] = struct{}{}
				}
			}
		}
	}
	targetIDs := make([]string, 0, len(bindings))
	for targetID := range bindings {
		targetIDs = append(targetIDs, targetID)
	}
	statuses, err := a.svc.LastVerifiedByTargetDestination(ctx, targetIDs)
	if err != nil {
		return nil, err
	}
	verified := make(map[string]restoresint.DestinationVerifiedStatus, len(statuses))
	for _, status := range statuses {
		verified[coverageTargetDestinationKey(status.TargetID, status.DestinationID)] = status
	}
	out := make(map[string]runsH.VerifiedInfo, len(bindings))
	for targetID, destinationSet := range bindings {
		if len(destinationSet) == 0 {
			continue
		}
		var oldest time.Time
		complete := true
		for destinationID := range destinationSet {
			status, ok := verified[coverageTargetDestinationKey(targetID, destinationID)]
			if !ok || status.LastVerifiedAt.IsZero() {
				complete = false
				break
			}
			if oldest.IsZero() || status.LastVerifiedAt.Before(oldest) {
				oldest = status.LastVerifiedAt
			}
		}
		if complete {
			out[targetID] = runsH.VerifiedInfo{LastVerifiedAt: oldest, SnapshotID: "all-enabled-destinations"}
		}
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

// plansTargetCatalog lets a selector plan name every registered target. It reads
// the catalog in full: a silent cap would leave targets out of a plan whose
// promise is that registered means backed up.
type plansTargetCatalog struct{ svc targetsint.Service }

func (a plansTargetCatalog) RegisteredTargetIDs(ctx context.Context) ([]string, error) {
	ts, err := a.svc.List(ctx, "", catalogScanLimit)
	if err != nil {
		return nil, err
	}
	if len(ts) >= catalogScanLimit {
		return nil, fmt.Errorf("the target catalog has %d or more targets, which is more than a selector plan reads; raise catalogScanLimit", catalogScanLimit)
	}
	ids := make([]string, 0, len(ts))
	for _, t := range ts {
		ids = append(ids, t.ID)
	}
	return ids, nil
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

func (a coverageSuggestions) ListTargetCandidates(ctx context.Context) ([]coverageint.DiscoveredTarget, error) {
	candidates, err := a.svc.ListTargetCandidates(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]coverageint.DiscoveredTarget, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, coverageint.DiscoveredTarget{
			Owner: candidate.Owner, Name: candidate.Name, Locator: candidate.Locator,
			Reacquirable: candidate.Reacquirable, ReacquisitionCommand: candidate.ReacquisitionCommand,
			ReacquisitionEvidence:      candidate.ReacquisitionEvidence,
			ReacquisitionNeedsOperator: candidate.ReacquisitionNeedsOperator,
			ReacquisitionAuthReason:    candidate.ReacquisitionAuthReason,
		})
	}
	return out, nil
}

func (a coverageSuggestions) ListTargetSuggestions(ctx context.Context) ([]coverageint.Suggestion, error) {
	sugs, err := a.svc.ListTargetSuggestions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]coverageint.Suggestion, 0, len(sugs))
	for _, s := range sugs {
		out = append(out, coverageint.Suggestion{
			ID:                         s.ID,
			Owner:                      s.Owner,
			OwnerKind:                  s.OwnerKind,
			Name:                       s.Name,
			SourceKind:                 s.SourceKind,
			Locator:                    s.Locator,
			Rationale:                  s.Rationale,
			ApproxBytes:                s.ApproxBytes,
			Sensitive:                  s.Sensitive,
			Critical:                   s.Critical,
			Warning:                    discoverySuggestionWarning(s.Sensitive, len(s.Findings)),
			Reacquirable:               s.Reacquirable,
			ReacquisitionCommand:       s.ReacquisitionCommand,
			ReacquisitionEvidence:      s.ReacquisitionEvidence,
			ReacquisitionNeedsOperator: s.ReacquisitionNeedsOperator,
			ReacquisitionAuthReason:    s.ReacquisitionAuthReason,
			BudgetMaxBytes:             s.BudgetMaxBytes,
			BudgetMaxAge:               s.BudgetMaxAge,
			Retention:                  s.Retention,
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
			Critical:   t.Critical,
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
		Critical:   in.Critical,
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
		Critical:   t.Critical,
	}, nil
}

// coveragePlanCatalog adapts plans.Service to coverage.PlanCatalog: it rolls
// every plan's target membership into the set of planned target ids.
type coveragePlanCatalog struct {
	svc      plansint.Service
	topology plansint.CriticalDestinationReporter
}

func (a coveragePlanCatalog) PlannedTargetIDs(ctx context.Context) (map[string]struct{}, error) {
	ps, err := a.svc.List(ctx, catalogScanLimit)
	if err != nil {
		return nil, err
	}
	out := make(map[string]struct{})
	for _, p := range ps {
		if !p.Enabled {
			continue
		}
		for _, id := range p.TargetIDs {
			out[id] = struct{}{}
		}
	}
	return out, nil
}

func (a coveragePlanCatalog) DestinationIDsByTarget(ctx context.Context, targetIDs []string) (map[string][]string, error) {
	ps, err := a.svc.List(ctx, catalogScanLimit)
	if err != nil {
		return nil, err
	}
	wanted := map[string]struct{}{}
	for _, id := range targetIDs {
		wanted[id] = struct{}{}
	}
	sets := map[string]map[string]struct{}{}
	for _, plan := range ps {
		if !plan.Enabled {
			continue
		}
		for _, targetID := range plan.TargetIDs {
			if _, ok := wanted[targetID]; !ok {
				continue
			}
			if sets[targetID] == nil {
				sets[targetID] = map[string]struct{}{}
			}
			for _, destinationID := range plan.DestinationIDs {
				if strings.TrimSpace(destinationID) != "" {
					sets[targetID][destinationID] = struct{}{}
				}
			}
		}
	}
	out := make(map[string][]string, len(sets))
	for targetID, set := range sets {
		for destinationID := range set {
			out[targetID] = append(out[targetID], destinationID)
		}
		sort.Strings(out[targetID])
	}
	return out, nil
}

func (a coveragePlanCatalog) PhysicalIndependenceByTarget(ctx context.Context, destinations map[string][]string) (map[string]bool, map[string][]string, error) {
	proven := make(map[string]bool, len(destinations))
	warnings := make(map[string][]string, len(destinations))
	for targetID, destinationIDs := range destinations {
		if len(destinationIDs) < 2 {
			warnings[targetID] = []string{"fewer than two enabled destinations are configured"}
			continue
		}
		if a.topology == nil {
			warnings[targetID] = []string{"destination topology evidence is unavailable"}
			continue
		}
		report, err := a.topology.Assess(ctx, plansint.TierCriticalSecondary, []string{targetID}, destinationIDs)
		if err != nil {
			return nil, nil, fmt.Errorf("target %q: %w", targetID, err)
		}
		proven[targetID] = report.PhysicallyIndependent
		warnings[targetID] = append([]string(nil), report.Warnings...)
	}
	return proven, warnings, nil
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

func (a coverageRunStatus) LastSuccessByTargetDestination(ctx context.Context, targetIDs []string) (map[string]time.Time, error) {
	statuses, err := a.svc.ListTargetDestinationStatus(ctx, targetIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[string]time.Time, len(statuses))
	for _, status := range statuses {
		if !status.LastSuccessAt.IsZero() {
			out[coverageTargetDestinationKey(status.TargetID, status.DestinationID)] = status.LastSuccessAt
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

func (a coverageVerified) LastVerifiedByTargetDestination(ctx context.Context, targetIDs []string) (map[string]time.Time, error) {
	statuses, err := a.svc.LastVerifiedByTargetDestination(ctx, targetIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[string]time.Time, len(statuses))
	for _, status := range statuses {
		if !status.LastVerifiedAt.IsZero() {
			out[coverageTargetDestinationKey(status.TargetID, status.DestinationID)] = status.LastVerifiedAt
		}
	}
	return out, nil
}

func coverageTargetDestinationKey(targetID, destinationID string) string {
	return targetID + "\x00" + destinationID
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
type planSource struct {
	svc  plansint.Service
	runs runsint.Service
}

func (a planSource) SchedulablePlans(ctx context.Context) ([]schedint.DuePlan, error) {
	ps, err := a.svc.SchedulablePlans(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]schedint.DuePlan, 0, len(ps))
	for _, p := range ps {
		due := schedint.DuePlan{ID: p.ID, Schedule: p.Schedule, Enabled: p.Enabled}
		if a.runs != nil {
			history, err := a.runs.ListRuns(ctx, p.ID, 1)
			if err != nil {
				return nil, fmt.Errorf("load latest run for scheduled plan %q: %w", p.ID, err)
			}
			if len(history) > 0 {
				due.LastRunAt = history[0].StartedAt
			}
		}
		out = append(out, due)
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

// backupPosture adapts backup freshness and enabled-plan repository connection
// checks to health.BackupPosture. It counts the per-target Overdue flag the
// runs service computes so health and `runs status` share one freshness rule.
type backupPosture struct {
	runs         runsint.Service
	plans        plansint.Service
	destinations destint.Service
	engine       repositoryPostureEngine
}

type repositoryPostureEngine interface {
	RepoStatus(context.Context, string) (engineint.RepoStatus, error)
}

func (a backupPosture) OverdueOrFailed(ctx context.Context) (bool, string, error) {
	const maxNamed = 10
	var named []string
	bad := 0
	represented := 0
	targetReasons := map[string]int{}
	if a.runs != nil {
		statuses, err := a.runs.ListTargetStatus(ctx, nil)
		if err != nil {
			return false, "", err
		}
		for _, s := range statuses {
			if !s.Overdue {
				continue
			}
			bad++
			targetReasons[overdueReason(s)]++
		}
	}
	reasons := make([]string, 0, len(targetReasons))
	for reason := range targetReasons {
		reasons = append(reasons, reason)
	}
	sort.Strings(reasons)
	for _, reason := range reasons {
		count := targetReasons[reason]
		if len(named) >= maxNamed {
			break
		}
		named = append(named, fmt.Sprintf("%d target(s) %s", count, reason))
		represented += count
	}

	// Check repository connections used by enabled plans as well as target
	// freshness. Manifest enumeration is intentionally left to the per-run
	// preflight: it can exceed the health probe's short timeout on large repos.
	if a.plans != nil && a.destinations != nil && a.engine != nil {
		plans, err := a.plans.List(ctx, 100)
		if err != nil {
			return false, "", fmt.Errorf("load backup plans for health: %w", err)
		}
		checked := map[string]bool{}
		for _, plan := range plans {
			if !plan.Enabled {
				continue
			}
			for _, id := range plan.DestinationIDs {
				if checked[id] {
					continue
				}
				checked[id] = true
				destinationName := id
				resolved := false
				// Health proves repository *connectivity*, which is a read. Using
				// the write gate here reported a mounted destination with an
				// unobservable native filesystem state as "unavailable" even though
				// its snapshots were readable.
				err := a.destinations.WithResolvedDestinationForRead(ctx, id, func(destination destint.Destination) error {
					if destination.ID != id {
						return fmt.Errorf("destination identity changed")
					}
					destinationName, resolved = destination.Name, true
					_, statusErr := a.engine.RepoStatus(ctx, destination.Name)
					return statusErr
				})
				if err != nil {
					bad++
					if len(named) < maxNamed {
						if resolved {
							named = append(named, fmt.Sprintf("repository %s unavailable", destinationName))
						} else {
							named = append(named, fmt.Sprintf("destination %s unavailable", id))
						}
						represented++
					}
				}
			}
		}
	}
	if bad == 0 {
		return false, "", nil
	}
	detail := fmt.Sprintf("%d backup posture issue(s): %s", bad, strings.Join(named, ", "))
	if bad > represented {
		detail += fmt.Sprintf(", +%d more", bad-represented)
	}
	return true, detail, nil
}

// overdueReason classifies why a target is overdue, for the /health detail
// enumeration. Mirrors runs.isOverdue's branches.
func overdueReason(s runsint.TargetStatus) string {
	switch s.LastRunStatus {
	case runsint.RunFailed:
		return lastFailureReason(s, "last run failed")
	case runsint.RunPartialFailed:
		return lastFailureReason(s, "last run partial-failed")
	}
	if s.LastSuccessAt.IsZero() {
		return "never backed up"
	}
	return "stale"
}

func lastFailureReason(s runsint.TargetStatus, fallback string) string {
	if s.LastFailureCategory == "" || s.LastFailureCode == "" {
		return fallback
	}
	return fmt.Sprintf("%s (%s/%s)", fallback, s.LastFailureCategory, s.LastFailureCode)
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

func startDrillScheduler(ctx context.Context, svc drillsint.Service, logger *log.Logger) error {
	interval := 60 * time.Second
	if raw, ok := lookupEnvTrimmed("DBM_DRILL_SCHEDULER_INTERVAL"); ok {
		d, err := time.ParseDuration(raw)
		if err != nil || d < 0 {
			return fmt.Errorf("DBM_DRILL_SCHEDULER_INTERVAL must be zero or a positive Go duration, got %q", raw)
		}
		if d == 0 {
			logger.Printf("recovery-drill scheduler: disabled via DBM_DRILL_SCHEDULER_INTERVAL=0")
			return nil
		}
		interval = d
	}
	go func() {
		// Reconcile due work immediately after owner bootstrap. Waiting for the
		// first ticker edge left a restart without scheduled coverage for up to
		// one full scheduler interval.
		if err := svc.RunDue(ctx); err != nil {
			logger.Printf("recovery-drill scheduler bootstrap: %v", err)
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := svc.RunDue(ctx); err != nil {
					logger.Printf("recovery-drill scheduler tick: %v", err)
				}
			}
		}
	}()
	return nil
}
