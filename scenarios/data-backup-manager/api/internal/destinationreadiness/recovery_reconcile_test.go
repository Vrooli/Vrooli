package destinationreadiness_test

import (
	"context"
	"testing"

	"data-backup-manager/internal/destinationreadiness"
	"data-backup-manager/internal/sysmounts"
)

type recoveryStateInspector struct {
	identity        destinationreadiness.DeviceIdentity
	mounted         bool
	readOnly        bool
	filesystemState sysmounts.FilesystemState
}

func (i recoveryStateInspector) Inspect(context.Context, string) (destinationreadiness.Inspection, error) {
	return destinationreadiness.Inspection{Identity: i.identity, Mounted: i.mounted, ReadOnly: i.readOnly, FilesystemState: i.filesystemState}, nil
}

func (i recoveryStateInspector) InspectDevice(context.Context, destinationreadiness.DeviceIdentity) (destinationreadiness.Inspection, error) {
	return destinationreadiness.Inspection{Identity: i.identity, Mounted: i.mounted, ReadOnly: i.readOnly, FilesystemState: i.filesystemState}, nil
}

func TestResumeRecoveryStepReconcilesProvenUnmountAfterRestart(t *testing.T) {
	_, journal, remediator := recoveryFixture(t)
	journal.Steps[0].Status = destinationreadiness.RecoveryStepInFlight
	store := &memoryRecoveryStore{journal: journal}
	identity := journal.Identity
	identity.Mountpoint = ""
	svc := destinationreadiness.NewService(recoveryStateInspector{identity: identity, mounted: false}, nil).WithRemediator(remediator)

	got, err := svc.ResumeRecoveryStep(context.Background(), store, journal, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.Current != 1 || got.State != destinationreadiness.RecoveryPending || got.Steps[0].Status != destinationreadiness.RecoveryStepCompleted || remediator.calls != 0 {
		t.Fatalf("journal=%+v calls=%d; want proven unmount receipted without replay", got, remediator.calls)
	}
}

func TestResumeRecoveryStepReconcilesProvenReadWriteMountAfterRestart(t *testing.T) {
	_, journal, remediator := recoveryFixture(t)
	journal.Current = 1
	journal.Steps[0].Status = destinationreadiness.RecoveryStepCompleted
	journal.Steps[1].Status = destinationreadiness.RecoveryStepInFlight
	store := &memoryRecoveryStore{journal: journal}
	identity := journal.Identity
	identity.Mountpoint = journal.Location
	svc := destinationreadiness.NewService(recoveryStateInspector{identity: identity, mounted: true, filesystemState: sysmounts.FilesystemStateClean}, nil).WithRemediator(remediator)

	got, err := svc.ResumeRecoveryStep(context.Background(), store, journal, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.Current != 2 || got.State != destinationreadiness.RecoveryCompleted || got.Steps[1].Status != destinationreadiness.RecoveryStepCompleted || remediator.calls != 0 {
		t.Fatalf("journal=%+v calls=%d; want exact read/write mount receipted without replay", got, remediator.calls)
	}
}

func TestResumeRecoveryStillPausesWhenInterruptedMountFilesystemIsUnknown(t *testing.T) {
	_, journal, remediator := recoveryFixture(t)
	journal.Current = 1
	journal.Steps[0].Status = destinationreadiness.RecoveryStepCompleted
	journal.Steps[1].Status = destinationreadiness.RecoveryStepInFlight
	store := &memoryRecoveryStore{journal: journal}
	identity := journal.Identity
	identity.Mountpoint = journal.Location
	svc := destinationreadiness.NewService(recoveryStateInspector{identity: identity, mounted: true, filesystemState: sysmounts.FilesystemStateUnknown}, nil).WithRemediator(remediator)

	got, err := svc.ResumeRecoveryStep(context.Background(), store, journal, nil, false, false)
	if err == nil || got.State != destinationreadiness.RecoveryPaused || got.Current != 1 || remediator.calls != 0 {
		t.Fatalf("journal=%+v calls=%d err=%v; want pause until filesystem health is proven", got, remediator.calls, err)
	}
}

func TestResumeRecoveryStillPausesWhenInterruptedMountIdentityIsUnproven(t *testing.T) {
	_, journal, remediator := recoveryFixture(t)
	journal.Steps[0].Status = destinationreadiness.RecoveryStepInFlight
	store := &memoryRecoveryStore{journal: journal}
	identity := journal.Identity
	identity.UUID = "replacement-device"
	svc := destinationreadiness.NewService(recoveryStateInspector{identity: identity, mounted: false}, nil).WithRemediator(remediator)

	got, err := svc.ResumeRecoveryStep(context.Background(), store, journal, nil, false, false)
	if err == nil || got.State != destinationreadiness.RecoveryPaused || got.Current != 0 || remediator.calls != 0 {
		t.Fatalf("journal=%+v calls=%d err=%v; want fail-closed pause", got, remediator.calls, err)
	}
}
