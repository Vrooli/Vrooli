package destinations

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"data-backup-manager/internal/destinationreadiness"
	"data-backup-manager/internal/destinations/mocks"

	"github.com/vrooli/api-core/operatordecision"

	destinationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/destinations"
)

type recoveryStore struct {
	journal destinationreadiness.RecoveryJournal
}

func (s *recoveryStore) Load(context.Context, string) (destinationreadiness.RecoveryJournal, error) {
	return s.journal, nil
}

func (s *recoveryStore) Save(_ context.Context, journal destinationreadiness.RecoveryJournal) error {
	s.journal = journal
	return nil
}

type recoveryHandlerInspector struct {
	identity destinationreadiness.DeviceIdentity
}

func (i recoveryHandlerInspector) Inspect(context.Context, string) (destinationreadiness.Inspection, error) {
	return destinationreadiness.Inspection{Identity: i.identity, Mounted: true, LocationExists: true, LocationIsDirectory: true}, nil
}

func (i recoveryHandlerInspector) InspectDevice(context.Context, destinationreadiness.DeviceIdentity) (destinationreadiness.Inspection, error) {
	return destinationreadiness.Inspection{Identity: i.identity, Mounted: false}, nil
}

type recoveryHandlerRemediator struct{ calls int }

func (r *recoveryHandlerRemediator) Supported(destinationreadiness.PreparationAction) (bool, string) {
	return true, ""
}

func (r *recoveryHandlerRemediator) Remediate(context.Context, destinationreadiness.Plan, bool) (destinationreadiness.RemediationOutcome, error) {
	r.calls++
	return destinationreadiness.RemediationOutcome{Status: "verified"}, nil
}

func TestVolumeRecoveryRPCsPersistAndResumeWithoutHostEffectsOnDryRun(t *testing.T) {
	identity := destinationreadiness.DeviceIdentity{DevicePath: "/dev/sdz1", UUID: "uuid-1", Filesystem: "ntfs", TotalBytes: 100}
	plan := destinationreadiness.Plan{
		ID: "step-1", Action: destinationreadiness.ActionCheckFilesystem,
		Location: "/media/Elements", TargetPath: "/media/Elements",
		Identity: identity, RequiresConfirm: true, ConfirmationPhrase: "CHECK", Supported: true,
	}
	store := &recoveryStore{}
	remediator := &recoveryHandlerRemediator{}
	readiness := destinationreadiness.NewService(recoveryHandlerInspector{identity: identity}, nil).WithRemediator(remediator)
	h := NewConnectHandler(Deps{Service: &mocks.FakeService{}, Readiness: readiness, RecoveryStore: store})

	started, err := h.StartVolumeRecovery(context.Background(), connect.NewRequest(&destinationsv1.StartVolumeRecoveryRequest{
		Location: "/media/Elements", Identity: deviceIdentityToProto(identity), Plans: []*destinationsv1.DestinationPreparationPlan{preparationPlanToProto(plan)},
	}))
	if err != nil {
		t.Fatalf("StartVolumeRecovery: %v", err)
	}
	if started.Msg.Journal.Id == "" || started.Msg.Journal.State != string(destinationreadiness.RecoveryPending) {
		t.Fatalf("journal = %+v", started.Msg.Journal)
	}

	got, err := h.GetVolumeRecovery(context.Background(), connect.NewRequest(&destinationsv1.GetVolumeRecoveryRequest{Id: started.Msg.Journal.Id}))
	if err != nil || got.Msg.Journal.Id != started.Msg.Journal.Id {
		t.Fatalf("GetVolumeRecovery journal=%+v err=%v", got.Msg.Journal, err)
	}

	resumed, err := h.ResumeVolumeRecovery(context.Background(), connect.NewRequest(&destinationsv1.ResumeVolumeRecoveryRequest{
		Id: started.Msg.Journal.Id, Confirmations: map[int32]string{0: "CHECK"}, DryRun: boolPtr(true),
	}))
	if err != nil {
		t.Fatalf("ResumeVolumeRecovery dry-run: %v", err)
	}
	if resumed.Msg.Journal.Current != 0 || resumed.Msg.Journal.State != string(destinationreadiness.RecoveryPending) {
		t.Fatalf("dry-run advanced journal: %+v", resumed.Msg.Journal)
	}
	if remediator.calls != 1 {
		t.Fatalf("remediator calls = %d, want one dry-run validation", remediator.calls)
	}
}

func TestResumeVolumeRecoveryCannotBypassNotificationDecisionWithCallerConfirmation(t *testing.T) {
	identity := destinationreadiness.DeviceIdentity{DevicePath: "/dev/sdz1", UUID: "uuid-1", Filesystem: "ntfs", TotalBytes: 100}
	plan := destinationreadiness.Plan{
		ID: "repair-plan", Action: destinationreadiness.ActionRepairFilesystem,
		Location: "/media/Elements", TargetPath: "/media/Elements", Identity: identity,
		RequiresConfirm: true, Destructive: true, ConfirmationPhrase: "REPAIR", Supported: true,
	}
	journal := destinationreadiness.RecoveryJournal{
		Version: "dbm-volume-recovery-v1", ID: "recovery-repair", Location: plan.Location,
		Identity: identity, Current: 0, State: destinationreadiness.RecoveryPending,
		Steps: []destinationreadiness.RecoveryStep{{Plan: plan, Status: "pending"}},
	}
	store := &recoveryStore{journal: journal}
	remediator := &recoveryHandlerRemediator{}
	readiness := destinationreadiness.NewService(recoveryHandlerInspector{identity: identity}, nil).WithRemediator(remediator)
	publisher := &recoveryDecisionPublisher{}
	decisions := newRecoveryDecisionFlow(readiness, store,
		&destinationreadiness.FileDecisionStore{Root: filepath.Join(t.TempDir(), "asks")}, publisher, true, nil)
	h := NewConnectHandler(Deps{Readiness: readiness, RecoveryStore: store, DecisionFlow: decisions})

	response, err := h.ResumeVolumeRecovery(context.Background(), connect.NewRequest(&destinationsv1.ResumeVolumeRecoveryRequest{
		Id: journal.ID, Confirmations: map[int32]string{0: "REPAIR"}, AcknowledgeDataLoss: true,
		DryRun: boolPtr(false),
	}))
	if err != nil {
		t.Fatalf("ResumeVolumeRecovery: %v", err)
	}
	if remediator.calls != 0 {
		t.Fatalf("caller-supplied repair confirmation executed host effect; calls=%d", remediator.calls)
	}
	if response.Msg.Journal.Steps[0].Status != "blocked" || !strings.Contains(response.Msg.Journal.LastError, "pending a verified Notification Hub decision") {
		t.Fatalf("journal = %+v, want a pending approval", response.Msg.Journal)
	}
	if len(publisher.events) != 1 || publisher.events[0].EventType != operatordecision.DecisionRequestedEventType(recoveryDecisionSource, recoveryDecisionKind) {
		t.Fatalf("published events = %+v, want repair decision request", publisher.events)
	}
}

func boolPtr(v bool) *bool { return &v }
