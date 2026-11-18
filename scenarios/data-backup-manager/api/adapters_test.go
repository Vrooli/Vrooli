package main

import (
	"context"
	"errors"
	"log"
	"strings"
	"testing"
	"time"

	"data-backup-manager/internal/destinationreadiness"
	destinations "data-backup-manager/internal/destinations"
	destmocks "data-backup-manager/internal/destinations/mocks"
	"data-backup-manager/internal/drills"
	"data-backup-manager/internal/engine"
	plans "data-backup-manager/internal/plans"
	planmocks "data-backup-manager/internal/plans/mocks"
	"data-backup-manager/internal/restores"
	restoremocks "data-backup-manager/internal/restores/mocks"
	runsint "data-backup-manager/internal/runs"
	runsmocks "data-backup-manager/internal/runs/mocks"
	"data-backup-manager/internal/targets"
	targetmocks "data-backup-manager/internal/targets/mocks"
	backupmocks "data-backup-manager/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
)

type adapterReadinessInspector struct {
	identity   destinationreadiness.DeviceIdentity
	inspection *destinationreadiness.Inspection
}

func (i adapterReadinessInspector) Inspect(context.Context, string) (destinationreadiness.Inspection, error) {
	if i.inspection != nil {
		return *i.inspection, nil
	}
	return destinationreadiness.Inspection{
		LocationExists:      true,
		LocationIsDirectory: true,
		Identity:            i.identity,
		FreeBytes:           1 << 40,
	}, nil
}

func TestPlanCriticalDestinationPolicyAssessReportsSharedRisk(t *testing.T) {
	targetsSvc := &targetmocks.FakeService{GetOut: targets.Target{ID: "target-1", Locator: "/source"}}
	destSvc := &destmocks.FakeService{GetOut: destinations.Destination{
		ID:          "destination-1",
		BackendKind: destinations.BackendFilesystem,
		Location:    "/backup",
	}}
	readiness := destinationreadiness.NewService(adapterReadinessInspector{
		identity: destinationreadiness.DeviceIdentity{DevicePath: "/dev/sdb1", Mountpoint: "/backup", UUID: "same-volume"},
	}, nil)
	policy := planCriticalDestinationPolicy{targets: targetsSvc, destinations: destSvc, readiness: readiness}

	report, err := policy.Assess(context.Background(), "critical_secondary", []string{"target-1"}, []string{"destination-1", "destination-2"})
	require.NoError(t, err)
	require.False(t, report.PhysicallyIndependent)
	require.Contains(t, strings.Join(report.Warnings, "\n"), "overlapping filesystem root")
	require.Contains(t, strings.Join(report.Warnings, "\n"), "stable volume identity")
	require.Contains(t, strings.Join(report.Warnings, "\n"), "topology is unknown")
}

type drillSchedulerService struct {
	runDueCalls chan struct{}
}

func (s *drillSchedulerService) Preview(context.Context, string, string, string) (drills.Preview, error) {
	return drills.Preview{}, nil
}

func (s *drillSchedulerService) Run(context.Context, string, string, string, string, bool) (drills.Drill, error) {
	return drills.Drill{}, nil
}

func (s *drillSchedulerService) RunMigrationSnapshot(context.Context, string, string, string, string, string) (drills.Drill, error) {
	return drills.Drill{}, nil
}

func (s *drillSchedulerService) Get(context.Context, string) (drills.Drill, error) {
	return drills.Drill{}, nil
}

func (s *drillSchedulerService) Retire(context.Context, string, string) (drills.Drill, error) {
	return drills.Drill{}, nil
}

func (s *drillSchedulerService) List(context.Context, string, string, int) ([]drills.Drill, error) {
	return nil, nil
}

func (s *drillSchedulerService) ListPage(context.Context, string, string, int, string) (drills.DrillPage, error) {
	return drills.DrillPage{}, nil
}

func (s *drillSchedulerService) RunDue(context.Context) error {
	s.runDueCalls <- struct{}{}
	return nil
}
func (s *drillSchedulerService) Reconcile(context.Context) error { return nil }
func (s *drillSchedulerService) Shutdown(context.Context) error  { return nil }

func TestStartDrillSchedulerRunsDueWorkAtBootstrap(t *testing.T) {
	t.Setenv("DBM_DRILL_SCHEDULER_INTERVAL", "1h")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service := &drillSchedulerService{runDueCalls: make(chan struct{}, 1)}
	require.NoError(t, startDrillScheduler(ctx, service, log.Default()))

	select {
	case <-service.runDueCalls:
	case <-time.After(time.Second):
		t.Fatal("drill scheduler did not run due work during bootstrap")
	}
	cancel()
}

func TestPlanCriticalDestinationPolicyUsesObservedHostAndDeviceTopology(t *testing.T) {
	targetsSvc := &targetmocks.FakeService{GetOut: targets.Target{ID: "target-1", Locator: "/source"}}
	destSvc := &destmocks.FakeService{GetOut: destinations.Destination{
		ID: "destination", BackendKind: destinations.BackendFilesystem, Location: "/backup",
	}}
	readiness := destinationreadiness.NewService(adapterReadinessInspector{
		identity: destinationreadiness.DeviceIdentity{Serial: "physical-disk-7", UUID: "volume-7"},
	}, nil)
	policy := planCriticalDestinationPolicy{targets: targetsSvc, destinations: destSvc, readiness: readiness, hostID: "confirmed-host-1"}

	report, err := policy.Assess(context.Background(), plans.TierCriticalSecondary, nil, []string{"destination-a", "destination-b"})
	require.NoError(t, err)
	require.False(t, report.PhysicallyIndependent)
	require.Contains(t, strings.Join(report.Warnings, "\n"), "same_device")

	policy.hostID = ""
	report, err = policy.Assess(context.Background(), plans.TierCriticalSecondary, nil, []string{"destination-a", "destination-b"})
	require.NoError(t, err)
	require.False(t, report.PhysicallyIndependent)
	require.Contains(t, strings.Join(report.Warnings, "\n"), "unknown")
}

func TestPlanCriticalDestinationPolicyDoesNotInferS3Hosting(t *testing.T) {
	destSvc := &destmocks.FakeService{GetOut: destinations.Destination{
		ID: "s3", BackendKind: destinations.BackendS3, Location: "bucket/prefix",
	}}
	policy := planCriticalDestinationPolicy{destinations: destSvc, hostID: "confirmed-host-1"}
	report, err := policy.Assess(context.Background(), plans.TierCriticalPrimary, nil, []string{"s3"})
	require.NoError(t, err)
	require.False(t, report.PhysicallyIndependent)
	require.Contains(t, strings.Join(report.Warnings, "\n"), "endpoint hosting evidence is unavailable")
}

func TestCoveragePlanCatalogUsesEnabledTargetDestinationPairs(t *testing.T) {
	svc := &planmocks.FakeService{ListOut: []plans.Plan{
		{ID: "enabled", Enabled: true, TargetIDs: []string{"t1", "t2"}, DestinationIDs: []string{"d1", "d2"}},
		{ID: "disabled", Enabled: false, TargetIDs: []string{"t1"}, DestinationIDs: []string{"d3"}},
	}}
	catalog := coveragePlanCatalog{svc: svc}
	planned, err := catalog.PlannedTargetIDs(context.Background())
	require.NoError(t, err)
	require.Contains(t, planned, "t1")
	require.Contains(t, planned, "t2")

	destinations, err := catalog.DestinationIDsByTarget(context.Background(), []string{"t1", "t2"})
	require.NoError(t, err)
	require.Equal(t, map[string][]string{"t1": {"d1", "d2"}, "t2": {"d1", "d2"}}, destinations)
	filtered, err := catalog.DestinationIDsByTarget(context.Background(), []string{"t2"})
	require.NoError(t, err)
	require.Equal(t, map[string][]string{"t2": {"d1", "d2"}}, filtered)
}

func TestVerifiedLookupRequiresEveryEnabledDestinationAndUsesOldestReceipt(t *testing.T) {
	base := time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)
	planSvc := &planmocks.FakeService{ListOut: []plans.Plan{
		{ID: "enabled", Enabled: true, TargetIDs: []string{"t1"}, DestinationIDs: []string{"d1", "d2"}},
		{ID: "disabled", Enabled: false, TargetIDs: []string{"t1"}, DestinationIDs: []string{"d3"}},
	}}
	restoreSvc := &restoremocks.FakeService{VerifiedDestinationOut: []restores.DestinationVerifiedStatus{
		{TargetID: "t1", DestinationID: "d1", LastVerifiedAt: base, SnapshotID: "s1"},
		{TargetID: "t1", DestinationID: "d2", LastVerifiedAt: base.Add(time.Hour), SnapshotID: "s2"},
	}}
	lookup := verifiedLookup{svc: restoreSvc, plans: planSvc}
	got, err := lookup.LastVerifiedByTarget(context.Background())
	require.NoError(t, err)
	require.Equal(t, base, got["t1"].LastVerifiedAt, "freshness should reflect the oldest copy verification")
	require.Equal(t, "all-enabled-destinations", got["t1"].SnapshotID)

	restoreSvc.VerifiedDestinationOut = restoreSvc.VerifiedDestinationOut[:1]
	got, err = lookup.LastVerifiedByTarget(context.Background())
	require.NoError(t, err)
	require.NotContains(t, got, "t1", "one good copy cannot qualify an unverified destination")
}

func TestPlanCriticalDestinationPolicyRejectsOverlappingCriticalDestination(t *testing.T) {
	targetsSvc := &targetmocks.FakeService{GetOut: targets.Target{ID: "target-1", Critical: true, Locator: "/source"}}
	destSvc := &destmocks.FakeService{GetOut: destinations.Destination{
		ID:          "destination-1",
		BackendKind: destinations.BackendFilesystem,
		Location:    "/backup",
	}}
	policy := planCriticalDestinationPolicy{targets: targetsSvc, destinations: destSvc}

	err := policy.Validate(context.Background(), "critical_secondary", []string{"target-1"}, []string{"destination-1", "destination-2"})
	var invalid plans.ErrInvalidPlan
	require.ErrorAs(t, err, &invalid)
	require.Contains(t, invalid.Reason, "overlapping filesystem root")
}

func TestPlanCriticalDestinationPolicyUsesCriticalTargetLocator(t *testing.T) {
	targetsSvc := &targetmocks.FakeService{GetOut: targets.Target{ID: "target-1", Critical: true, Locator: "/backup/source"}}
	destSvc := &destmocks.FakeService{GetOut: destinations.Destination{
		ID:          "destination-1",
		BackendKind: destinations.BackendFilesystem,
		Location:    "/backup",
	}}
	policy := planCriticalDestinationPolicy{targets: targetsSvc, destinations: destSvc}

	err := policy.Validate(context.Background(), "critical_primary", []string{"target-1"}, []string{"destination-1"})
	var invalid plans.ErrInvalidPlan
	require.ErrorAs(t, err, &invalid)
	require.Contains(t, invalid.Reason, "overlaps critical target")
}

func TestPlanCriticalDestinationPolicyRejectsKnownUnreadableDestination(t *testing.T) {
	targetsSvc := &targetmocks.FakeService{GetOut: targets.Target{ID: "target-1", Critical: true, Locator: "/source"}}
	destSvc := &destmocks.FakeService{GetOut: destinations.Destination{
		ID:          "destination-1",
		BackendKind: destinations.BackendFilesystem,
		Location:    "/backup",
	}}
	readiness := destinationreadiness.NewService(adapterReadinessInspector{
		inspection: &destinationreadiness.Inspection{
			LocationExists:      false,
			LocationIsDirectory: false,
			Identity: destinationreadiness.DeviceIdentity{
				DevicePath: "/dev/sdb1",
				Mountpoint: "/backup",
				Filesystem: "ntfs",
			},
		},
	}, nil)
	policy := planCriticalDestinationPolicy{targets: targetsSvc, destinations: destSvc, readiness: readiness}

	err := policy.Validate(context.Background(), plans.TierCriticalPrimary, []string{"target-1"}, []string{"destination-1"})
	var invalid plans.ErrInvalidPlan
	require.ErrorAs(t, err, &invalid)
	require.Contains(t, invalid.Reason, "destination_missing")
}

func TestBackupPostureDetectsDisconnectedRepositoryForEnabledPlan(t *testing.T) {
	destSvc := &destmocks.FakeService{GetOut: destinations.Destination{ID: "d1", Name: "elements-local"}}
	planSvc := &planmocks.FakeService{ListOut: []plans.Plan{{ID: "p1", Enabled: true, DestinationIDs: []string{"d1"}}}}
	engine := &backupmocks.FakeKopiaEngine{
		RepoStatusFn: func(context.Context, string) (engine.RepoStatus, error) {
			return engine.RepoStatus{}, errors.New("repository unavailable")
		},
	}
	posture := backupPosture{plans: planSvc, destinations: destSvc, engine: engine}

	degraded, detail, err := posture.OverdueOrFailed(context.Background())
	require.NoError(t, err)
	require.True(t, degraded)
	require.Contains(t, detail, "repository elements-local unavailable")
	require.NotContains(t, detail, "repository unavailable", "raw engine errors should not escape into health output")
}

func TestBackupPostureDoesNotProbeStaleRepositoryWhenDestinationCannotResolve(t *testing.T) {
	repoStatusCalls := 0
	destSvc := &destmocks.FakeService{GetOut: destinations.Destination{ID: "d1", Name: "elements-local"}, ResolveErr: errors.New("filesystem readiness unknown")}
	planSvc := &planmocks.FakeService{ListOut: []plans.Plan{{ID: "p1", Enabled: true, DestinationIDs: []string{"d1"}}}}
	engine := &backupmocks.FakeKopiaEngine{RepoStatusFn: func(context.Context, string) (engine.RepoStatus, error) {
		repoStatusCalls++
		return engine.RepoStatus{Connected: true}, nil
	}}
	posture := backupPosture{plans: planSvc, destinations: destSvc, engine: engine}
	degraded, detail, err := posture.OverdueOrFailed(context.Background())
	require.NoError(t, err)
	require.True(t, degraded)
	require.Contains(t, detail, "destination d1 unavailable")
	require.Equal(t, 0, repoStatusCalls, "a stale engine connection cannot make unknown destination readiness look healthy")
}

func TestBackupPostureGroupsTargetFailuresByRedactedCause(t *testing.T) {
	runs := &runsmocks.FakeService{StatusOut: []runsint.TargetStatus{
		{TargetID: "secret-target-id-1", LastRunStatus: runsint.RunFailed, LastFailureCategory: "destination", LastFailureCode: "destination_inaccessible", Overdue: true},
		{TargetID: "secret-target-id-2", LastRunStatus: runsint.RunFailed, LastFailureCategory: "destination", LastFailureCode: "destination_inaccessible", Overdue: true},
		{TargetID: "secret-target-id-3", Overdue: true},
	}}
	degraded, detail, err := (backupPosture{runs: runs}).OverdueOrFailed(context.Background())
	require.NoError(t, err)
	require.True(t, degraded)
	require.Contains(t, detail, "2 target(s) last run failed (destination/destination_inaccessible)")
	require.Contains(t, detail, "1 target(s) never backed up")
	require.NotContains(t, detail, "secret-target-id", "health should group by cause instead of exposing opaque target IDs")
}

func TestBackupPostureMoreCountCountsIssuesNotSummaryRows(t *testing.T) {
	statuses := make([]runsint.TargetStatus, 0, 30)
	for i := 0; i < 21; i++ {
		statuses = append(statuses, runsint.TargetStatus{TargetID: "missing", LastRunStatus: runsint.RunFailed, LastFailureCategory: "destination", LastFailureCode: "destination_missing", Overdue: true})
	}
	for i := 0; i < 9; i++ {
		statuses = append(statuses, runsint.TargetStatus{TargetID: "unknown", LastRunStatus: runsint.RunFailed, LastFailureCategory: "destination", LastFailureCode: "destination_inaccessible", Overdue: true})
	}
	runService := &runsmocks.FakeService{StatusOut: statuses}
	planService := &planmocks.FakeService{ListOut: []plans.Plan{{ID: "plan", Enabled: true, DestinationIDs: []string{"d1", "d2", "d3"}}}}
	destinationService := &destmocks.FakeService{ResolveErr: errors.New("filesystem readiness unknown")}
	engine := &backupmocks.FakeKopiaEngine{}
	degraded, detail, err := (backupPosture{runs: runService, plans: planService, destinations: destinationService, engine: engine}).OverdueOrFailed(context.Background())
	require.NoError(t, err)
	require.True(t, degraded)
	require.Contains(t, detail, "33 backup posture issue(s)")
	require.Contains(t, detail, "21 target(s)")
	require.Contains(t, detail, "9 target(s)")
	require.Contains(t, detail, "destination d1 unavailable")
	require.NotContains(t, detail, "+28 more", "rendered summary rows are fewer than actual represented issues")
}
