package main

import (
	"context"
	"strings"
	"testing"

	"data-backup-manager/internal/destinationreadiness"
	destinations "data-backup-manager/internal/destinations"
	"data-backup-manager/internal/destinations/mocks"
	plans "data-backup-manager/internal/plans"
	"data-backup-manager/internal/targets"
	targetmocks "data-backup-manager/internal/targets/mocks"

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
	destSvc := &mocks.FakeService{GetOut: destinations.Destination{
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
	require.Contains(t, strings.Join(report.Warnings, "\n"), "same physical volume")
}

func TestPlanCriticalDestinationPolicyRejectsOverlappingCriticalDestination(t *testing.T) {
	targetsSvc := &targetmocks.FakeService{GetOut: targets.Target{ID: "target-1", Critical: true, Locator: "/source"}}
	destSvc := &mocks.FakeService{GetOut: destinations.Destination{
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
	destSvc := &mocks.FakeService{GetOut: destinations.Destination{
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
	destSvc := &mocks.FakeService{GetOut: destinations.Destination{
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
