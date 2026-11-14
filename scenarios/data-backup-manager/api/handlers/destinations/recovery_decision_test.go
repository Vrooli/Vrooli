package destinations

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"data-backup-manager/internal/destinationreadiness"
	destdomain "data-backup-manager/internal/destinations"
	destmocks "data-backup-manager/internal/destinations/mocks"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
)

type recoveryDecisionPublisher struct {
	events   []eventbus.DomainEvent
	failNext int
}

func (p *recoveryDecisionPublisher) PublishDomainEvent(_ context.Context, event eventbus.DomainEvent) error {
	if p.failNext > 0 {
		p.failNext--
		return context.DeadlineExceeded
	}
	p.events = append(p.events, event)
	return nil
}

type decisionTestInspector struct {
	identity destinationreadiness.DeviceIdentity
	mounted  *bool
}

func (i decisionTestInspector) Inspect(context.Context, string) (destinationreadiness.Inspection, error) {
	return destinationreadiness.Inspection{Identity: i.identity, Mounted: true, LocationExists: true, LocationIsDirectory: true}, nil
}

func (i decisionTestInspector) InspectDevice(context.Context, destinationreadiness.DeviceIdentity) (destinationreadiness.Inspection, error) {
	return destinationreadiness.Inspection{Identity: i.identity, Mounted: i.mounted != nil && *i.mounted}, nil
}

type decisionTestRemediator struct {
	calls      int
	mounted    *bool
	consistent string
	plans      []destinationreadiness.Plan
}

func (r *decisionTestRemediator) Supported(destinationreadiness.PreparationAction) (bool, string) {
	return true, ""
}

func (r *decisionTestRemediator) Remediate(_ context.Context, plan destinationreadiness.Plan, _ bool) (destinationreadiness.RemediationOutcome, error) {
	r.calls++
	r.plans = append(r.plans, plan)
	if plan.Action == destinationreadiness.ActionMountReadWrite && r.mounted != nil {
		*r.mounted = true
	}
	return destinationreadiness.RemediationOutcome{Status: "changed", Changed: true, Consistent: r.consistent}, nil
}

func setupRecoveryDecision(t *testing.T) (*RecoveryDecisionFlow, *recoveryStore, *decisionTestRemediator, *recoveryDecisionPublisher, string) {
	t.Helper()
	identity := destinationreadiness.DeviceIdentity{
		DevicePath: "/dev/sdz1", Mountpoint: "/media/Elements", Filesystem: "ntfs",
		UUID: "uuid-elements", Serial: "serial-elements", TotalBytes: 100,
	}
	plan := destinationreadiness.Plan{
		ID: "plan-repair-1", Action: destinationreadiness.ActionRepairFilesystem,
		Location: "/media/Elements", TargetPath: "/media/Elements", Identity: identity,
		RequiresConfirm: true, Destructive: true, ConfirmationPhrase: "REPAIR /dev/sdz1 FS ntfs UUID uuid-elements SIZE 100", Supported: true,
	}
	mountPlan := destinationreadiness.Plan{
		ID: "plan-mount-1", Action: destinationreadiness.ActionMountReadWrite,
		Location: "/media/Elements", TargetPath: "/media/Elements", Identity: identity,
		RequiresConfirm: true, ConfirmationPhrase: "MOUNT /dev/sdz1 READ-WRITE SIZE 100", Supported: true,
	}
	journal := destinationreadiness.RecoveryJournal{
		Version: "dbm-volume-recovery-v1", ID: "recovery-1", Location: plan.Location,
		Identity: identity, Current: 0, State: destinationreadiness.RecoveryPaused,
		Steps: []destinationreadiness.RecoveryStep{{Plan: plan, Status: "blocked", Attempts: 1}, {Plan: mountPlan, Status: "pending"}}, UpdatedAt: time.Now().UTC(),
	}
	store := &recoveryStore{journal: journal}
	mounted := false
	remediator := &decisionTestRemediator{mounted: &mounted}
	readiness := destinationreadiness.NewService(decisionTestInspector{identity: identity, mounted: &mounted}, nil).WithRemediator(remediator)
	publisher := &recoveryDecisionPublisher{}
	tasks := &destinationreadiness.FileDecisionStore{Root: filepath.Join(t.TempDir(), "asks")}
	flow := newRecoveryDecisionFlow(readiness, store, tasks, publisher, true, nil)
	if err := flow.RaiseRepairApproval(context.Background(), journal); err != nil {
		t.Fatalf("RaiseRepairApproval: %v", err)
	}
	return flow, store, remediator, publisher, operatordecision.CorrelationID("dbm-repair", journal.ID, plan.ID, "0")
}

func TestRecoveryDecisionRaisesExactRiskFactsAndOperatorActionsPath(t *testing.T) {
	_, _, remediator, publisher, _ := setupRecoveryDecision(t)
	if remediator.calls != 0 {
		t.Fatalf("repair ran while the decision was only raised; calls=%d", remediator.calls)
	}
	if len(publisher.events) != 1 {
		t.Fatalf("published %d events, want the decision request", len(publisher.events))
	}
	event := publisher.events[0]
	if event.EventType != operatordecision.DecisionRequestedEventType(recoveryDecisionSource, recoveryDecisionKind) {
		t.Fatalf("event type = %q", event.EventType)
	}
	payload := event.Payload
	if payload["recommended"] != "approve_repair" || payload["reversible"] != false {
		t.Fatalf("decision recommendation/risk facts = %#v", payload)
	}
	if !strings.Contains(payload["technical_details"].(string), operatorActionsFileFullPath) {
		t.Fatalf("decision omits operator actions path: %#v", payload["technical_details"])
	}
}

func TestRecoveryDecisionRequiresVerifiedHumanBeforeRepair(t *testing.T) {
	flow, _, remediator, publisher, askID := setupRecoveryDecision(t)
	handled, err := flow.Apply(context.Background(), operatordecision.DecisionResolution{
		AskID: askID, State: operatordecision.StateAnswered, Answer: "approve_repair",
		AnsweredBy: "agent", AnsweredByIdentityKind: "agent", AnsweredByIdentitySource: "test",
		SourceEventType: operatordecision.DecisionRequestedEventType(recoveryDecisionSource, recoveryDecisionKind),
		Correlation:     map[string]string{"recovery_id": "recovery-1", "plan_id": "plan-repair-1", "step_index": "0"},
	})
	if err != nil || !handled {
		t.Fatalf("Apply = (%v, %v), want handled no-op", handled, err)
	}
	if remediator.calls != 0 {
		t.Fatalf("non-human approval ran repair; calls=%d", remediator.calls)
	}
	if len(publisher.events) != 2 || publisher.events[1].EventType != recoveryNotificationType {
		t.Fatalf("events = %+v, want a follow-up failure notification", publisher.events)
	}
}

func TestRecoveryDecisionOtherAndDismissalNeverRepair(t *testing.T) {
	cases := []struct {
		name       string
		state      string
		answer     string
		note       string
		wantDetail string
	}{
		{name: "other", state: operatordecision.StateAnswered, answer: "other", note: "Keep the drive unchanged and use the second destination.", wantDetail: "Keep the drive unchanged and use the second destination."},
		{name: "dismissed", state: operatordecision.StateDismissed, wantDetail: "dismissed the repair decision without answering"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			flow, store, remediator, publisher, askID := setupRecoveryDecision(t)
			resolution := operatordecision.DecisionResolution{
				AskID: askID, State: tc.state, Answer: tc.answer, Note: tc.note,
				AnsweredBy: "operator", AnsweredByIdentityKind: "verified_human", AnsweredByIdentitySource: "cloudflare_access",
				SourceEventType: operatordecision.DecisionRequestedEventType(recoveryDecisionSource, recoveryDecisionKind),
				Correlation:     map[string]string{"recovery_id": store.journal.ID, "plan_id": store.journal.Steps[0].Plan.ID, "step_index": "0"},
			}
			handled, err := flow.Apply(context.Background(), resolution)
			if err != nil || !handled {
				t.Fatalf("Apply = (%v, %v), want handled", handled, err)
			}
			if remediator.calls != 0 {
				t.Fatalf("non-approval triggered host remediation %d times", remediator.calls)
			}
			if len(publisher.events) != 2 || publisher.events[1].Payload["category"] != "volume-recovery-failed" {
				t.Fatalf("events = %+v, want a non-approval follow-up notification", publisher.events)
			}
			if !strings.Contains(publisher.events[1].Payload["body"].(string), tc.wantDetail) {
				t.Fatalf("follow-up omitted operator disposition %q: %#v", tc.wantDetail, publisher.events[1].Payload["body"])
			}
		})
	}
}

func TestRecoveryDecisionVerifiedHumanExecutesOnceAndNotifies(t *testing.T) {
	flow, store, remediator, publisher, askID := setupRecoveryDecision(t)
	resolution := operatordecision.DecisionResolution{
		AskID: askID, State: operatordecision.StateAnswered, Answer: "approve_repair",
		AnsweredBy: "operator", AnsweredByIdentityKind: "verified_human", AnsweredByIdentitySource: "cloudflare_access",
		SourceEventType: operatordecision.DecisionRequestedEventType(recoveryDecisionSource, recoveryDecisionKind),
		Correlation:     map[string]string{"recovery_id": "recovery-1", "plan_id": "plan-repair-1", "step_index": "0"},
	}
	handled, err := flow.Apply(context.Background(), resolution)
	if err != nil || !handled {
		t.Fatalf("Apply = (%v, %v), want handled", handled, err)
	}
	if remediator.calls != 2 || store.journal.State != destinationreadiness.RecoveryCompleted {
		t.Fatalf("repair calls=%d journal=%+v", remediator.calls, store.journal)
	}
	ask, err := flow.asks.Get(context.Background(), askID)
	if err != nil || ask == nil || ask.State != operatordecision.PendingStateApproved {
		t.Fatalf("resolved ask=(%+v,%v)", ask, err)
	}
	if len(publisher.events) != 2 || publisher.events[1].Payload["category"] != "volume-recovery-completed" {
		t.Fatalf("events = %+v, want success follow-up notification", publisher.events)
	}
	if _, err := flow.Apply(context.Background(), resolution); err != nil {
		t.Fatalf("duplicate Apply: %v", err)
	}
	if remediator.calls != 2 {
		t.Fatalf("duplicate callback repeated repair; calls=%d", remediator.calls)
	}
}

func TestRecoveryDecisionRetriesResultNoticeWithoutRepeatingRepair(t *testing.T) {
	flow, _, remediator, publisher, askID := setupRecoveryDecision(t)
	resolution := operatordecision.DecisionResolution{
		AskID: askID, State: operatordecision.StateAnswered, Answer: "approve_repair",
		AnsweredBy: "operator", AnsweredByIdentityKind: "verified_human", AnsweredByIdentitySource: "cloudflare_access",
		SourceEventType: operatordecision.DecisionRequestedEventType(recoveryDecisionSource, recoveryDecisionKind),
		Correlation:     map[string]string{"recovery_id": "recovery-1", "plan_id": "plan-repair-1", "step_index": "0"},
	}
	publisher.failNext = 1
	if handled, err := flow.Apply(context.Background(), resolution); err == nil || !handled {
		t.Fatalf("first Apply = (%v, %v), want handled with retryable notification error", handled, err)
	}
	if remediator.calls != 2 {
		t.Fatalf("first callback remediation calls=%d, want repair plus mount", remediator.calls)
	}
	if handled, err := flow.Apply(context.Background(), resolution); err != nil || !handled {
		t.Fatalf("duplicate Apply = (%v, %v), want outbox retry", handled, err)
	}
	if remediator.calls != 2 {
		t.Fatalf("outbox retry repeated remediation; calls=%d", remediator.calls)
	}
	if len(publisher.events) != 2 || publisher.events[1].EventType != recoveryNotificationType {
		t.Fatalf("events after retry = %+v, want one decision and one result notice", publisher.events)
	}
	ask, err := flow.asks.Get(context.Background(), askID)
	if err != nil || ask == nil || ask.Extra["result_notification_pending"] != "false" {
		t.Fatalf("result outbox = (%+v, %v), want delivered", ask, err)
	}
}

func TestRecoveryDecisionReplaysExactApprovalMisclassifiedByPreviousBuild(t *testing.T) {
	flow, store, remediator, _, askID := setupRecoveryDecision(t)
	ask, err := flow.asks.Get(context.Background(), askID)
	if err != nil || ask == nil {
		t.Fatalf("load pending ask=(%+v, %v)", ask, err)
	}
	ask.State = operatordecision.PendingStateRejected
	ask.Extra["result_title"] = "Backup drive recovery needs attention"
	ask.Extra["result_body"] = "The repair decision did not include a verified human approval. The drive was left unchanged."
	ask.Extra["result_category"] = "volume-recovery-failed"
	if err := flow.asks.Put(context.Background(), ask); err != nil {
		t.Fatalf("seed previous-build state: %v", err)
	}
	resolution := operatordecision.DecisionResolution{
		AskID: askID, State: operatordecision.StateAnswered, Answer: "approve_repair",
		AnsweredBy: "operator", AnsweredByIdentityKind: "verified_human", AnsweredByIdentitySource: "cloudflare_access",
		SourceEventType: operatordecision.DecisionRequestedEventType(recoveryDecisionSource, recoveryDecisionKind),
		Correlation:     map[string]string{"recovery_id": "recovery-1", "plan_id": "plan-repair-1", "step_index": "0"},
	}
	if handled, err := flow.Apply(context.Background(), resolution); err != nil || !handled {
		t.Fatalf("Apply replay = (%v, %v), want handled", handled, err)
	}
	if remediator.calls != 2 || store.journal.State != destinationreadiness.RecoveryCompleted {
		t.Fatalf("repair calls=%d journal=%+v, want approved repair and mount", remediator.calls, store.journal)
	}
}

func TestRecoveryDecisionCompletesMountedVolumeAfterFilesystemAliasVerificationFailure(t *testing.T) {
	identity := destinationreadiness.DeviceIdentity{
		DevicePath: "/dev/sdz1", Mountpoint: "/media/Elements", Filesystem: "ntfs",
		UUID: "uuid-elements", Serial: "serial-elements", TotalBytes: 100,
	}
	repairPlan := destinationreadiness.Plan{
		ID: "plan-repair-1", Action: destinationreadiness.ActionRepairFilesystem,
		Location: identity.Mountpoint, TargetPath: identity.Mountpoint, Identity: identity,
		RequiresConfirm: true, Destructive: true, ConfirmationPhrase: "REPAIR /dev/sdz1 FS ntfs UUID uuid-elements SIZE 100", Supported: true,
	}
	mountPlan := destinationreadiness.Plan{
		ID: "plan-mount-1", Action: destinationreadiness.ActionMountReadWrite,
		Location: identity.Mountpoint, TargetPath: identity.Mountpoint, Identity: identity,
		RequiresConfirm: true, ConfirmationPhrase: "MOUNT /dev/sdz1 READ-WRITE SIZE 100", Supported: true,
	}
	journal := destinationreadiness.RecoveryJournal{
		Version: "dbm-volume-recovery-v1", ID: "recovery-1", Location: identity.Mountpoint,
		Identity: identity, Current: 0, State: destinationreadiness.RecoveryPaused,
		Steps: []destinationreadiness.RecoveryStep{{Plan: repairPlan, Status: "blocked"}, {Plan: mountPlan, Status: "pending"}},
	}
	store := &recoveryStore{journal: journal}
	// The stable-identity lens can miss a mount that the location readiness
	// observer sees. Completion must use and identity-check that location lens.
	mounted := false
	observedIdentity := identity
	observedIdentity.Filesystem = "ntfs3"
	readiness := destinationreadiness.NewService(decisionTestInspector{identity: observedIdentity, mounted: &mounted}, nil)
	publisher := &recoveryDecisionPublisher{}
	flow := newRecoveryDecisionFlow(readiness, store,
		&destinationreadiness.FileDecisionStore{Root: filepath.Join(t.TempDir(), "asks")}, publisher, true, nil)
	if err := flow.RaiseRepairApproval(context.Background(), journal); err != nil {
		t.Fatalf("RaiseRepairApproval: %v", err)
	}
	askID := operatordecision.CorrelationID("dbm-repair", journal.ID, repairPlan.ID, "0")
	ask, err := flow.asks.Get(context.Background(), askID)
	if err != nil || ask == nil {
		t.Fatalf("load ask=(%+v, %v)", ask, err)
	}
	ask.State = operatordecision.PendingStateRejected
	ask.ResolvedBy = "operator"
	ask.Extra["result_title"] = "Filesystem repair needs attention"
	ask.Extra["result_body"] = "preparation refused: mount action completed without a verified read/write mount"
	ask.Extra["result_category"] = "volume-recovery-failed"
	if err := flow.asks.Put(context.Background(), ask); err != nil {
		t.Fatalf("seed prior verification failure: %v", err)
	}
	store.journal.Steps[0].Status = "completed"
	store.journal.Steps[1].Status = "blocked"
	store.journal.Current = 1
	store.journal.State = destinationreadiness.RecoveryPaused
	resolution := operatordecision.DecisionResolution{
		AskID: askID, State: operatordecision.StateAnswered, Answer: "approve_repair",
		AnsweredBy: "operator", AnsweredByIdentityKind: "verified_human", AnsweredByIdentitySource: "cloudflare_access",
		SourceEventType: operatordecision.DecisionRequestedEventType(recoveryDecisionSource, recoveryDecisionKind),
		Correlation:     map[string]string{"recovery_id": journal.ID, "plan_id": repairPlan.ID, "step_index": "0"},
	}
	if handled, err := flow.Apply(context.Background(), resolution); err != nil || !handled {
		t.Fatalf("Apply replay = (%v, %v), want handled", handled, err)
	}
	if store.journal.State != destinationreadiness.RecoveryCompleted || store.journal.Current != len(store.journal.Steps) {
		t.Fatalf("journal = %+v, want completed after read-only mount verification", store.journal)
	}
	if len(publisher.events) != 2 || publisher.events[1].Payload["category"] != "volume-recovery-completed" {
		t.Fatalf("events = %+v, want one completion follow-up after the original ask", publisher.events)
	}
}

func TestVolumeRecoveryWorkerMountsOnlyAfterPositiveReadOnlyCheck(t *testing.T) {
	identity := destinationreadiness.DeviceIdentity{DevicePath: "/dev/sdz1", Mountpoint: "/media/Elements", UUID: "uuid-elements", Filesystem: "ntfs", TotalBytes: 100}
	mounted := false
	inspector := decisionTestInspector{identity: identity, mounted: &mounted}
	remediator := &decisionTestRemediator{mounted: &mounted, consistent: "yes"}
	readiness := destinationreadiness.NewService(inspector, nil).WithRemediator(remediator)
	dest := destdomain.Destination{ID: "dest-1", Name: "elements", BackendKind: destdomain.BackendFilesystem, Location: "/media/Elements", DeviceIdentity: &identity}
	worker := newVolumeRecoveryWorker(&destmocks.FakeService{ListOut: []destdomain.Destination{dest}}, readiness,
		&destinationreadiness.FileRecoveryStore{Root: filepath.Join(t.TempDir(), "journals")}, nil, nil)
	if err := worker.reconcileDestination(context.Background(), dest); err != nil {
		t.Fatalf("reconcileDestination: %v", err)
	}
	if remediator.calls != 2 || !mounted {
		t.Fatalf("remediation calls=%d mounted=%v, want check then mount", remediator.calls, mounted)
	}
}

func TestVolumeRecoveryWorkerVerifiesAndNotifiesSafeRemount(t *testing.T) {
	identity := destinationreadiness.DeviceIdentity{DevicePath: "/dev/sdz1", Mountpoint: "/media/Elements", UUID: "uuid-elements", Filesystem: "ntfs", TotalBytes: 100}
	mounted := false
	remediator := &decisionTestRemediator{mounted: &mounted, consistent: "yes"}
	readiness := destinationreadiness.NewService(decisionTestInspector{identity: identity, mounted: &mounted}, nil).
		WithRemediator(remediator)
	publisher := &recoveryDecisionPublisher{}
	flow := newRecoveryDecisionFlow(readiness, &recoveryStore{}, &destinationreadiness.FileDecisionStore{Root: filepath.Join(t.TempDir(), "asks")}, publisher, true, nil)
	dest := destdomain.Destination{ID: "dest-safe", Name: "elements", BackendKind: destdomain.BackendFilesystem, Location: "/media/Elements/vrooli-backups", DeviceIdentity: &identity}
	worker := newVolumeRecoveryWorker(&destmocks.FakeService{ListOut: []destdomain.Destination{dest}}, readiness,
		&destinationreadiness.FileRecoveryStore{Root: filepath.Join(t.TempDir(), "journals")}, flow, nil)
	if err := worker.reconcileDestination(context.Background(), dest); err != nil {
		t.Fatalf("reconcileDestination: %v", err)
	}
	if !mounted {
		t.Fatal("safe remount did not reach the mounted state")
	}
	if len(remediator.plans) != 2 || remediator.plans[1].Location != "/media/Elements" {
		t.Fatalf("mount plan location = %+v, want volume mount root rather than nested backup directory", remediator.plans)
	}
	if len(publisher.events) != 1 || publisher.events[0].Payload["category"] != "volume-recovery-completed" {
		t.Fatalf("events = %+v, want verified remount completion notice", publisher.events)
	}
}

func TestVolumeRecoveryWorkerRaisesDecisionWhenCheckCannotProveCleanState(t *testing.T) {
	identity := destinationreadiness.DeviceIdentity{DevicePath: "/dev/sdz1", Mountpoint: "/media/Elements", UUID: "uuid-elements", Filesystem: "ntfs", TotalBytes: 100}
	mounted := false
	inspector := decisionTestInspector{identity: identity, mounted: &mounted}
	remediator := &decisionTestRemediator{mounted: &mounted, consistent: "unknown"}
	readiness := destinationreadiness.NewService(inspector, nil).WithRemediator(remediator)
	publisher := &recoveryDecisionPublisher{}
	recoveryStore := &destinationreadiness.FileRecoveryStore{Root: filepath.Join(t.TempDir(), "journals")}
	flow := newRecoveryDecisionFlow(readiness,
		recoveryStore,
		&destinationreadiness.FileDecisionStore{Root: filepath.Join(t.TempDir(), "asks")}, publisher, true, nil)
	dest := destdomain.Destination{ID: "dest-2", Name: "elements", BackendKind: destdomain.BackendFilesystem, Location: "/media/Elements", DeviceIdentity: &identity}
	worker := newVolumeRecoveryWorker(&destmocks.FakeService{ListOut: []destdomain.Destination{dest}}, readiness, recoveryStore, flow, nil)
	if err := worker.reconcileDestination(context.Background(), dest); err != nil {
		t.Fatalf("reconcileDestination: %v", err)
	}
	if remediator.calls != 1 || mounted {
		t.Fatalf("remediation calls=%d mounted=%v, want only check and no mount", remediator.calls, mounted)
	}
	if len(publisher.events) != 2 || publisher.events[0].EventType != recoveryNotificationType || publisher.events[1].EventType != operatordecision.DecisionRequestedEventType(recoveryDecisionSource, recoveryDecisionKind) {
		t.Fatalf("events = %+v, want mount-failure alert followed by repair decision", publisher.events)
	}
}

func TestVolumeRecoveryWorkerRaisesFreshDecisionAfterCompletedEpisodeAndAvoidsDuplicateAsks(t *testing.T) {
	ctx := context.Background()
	identity := destinationreadiness.DeviceIdentity{DevicePath: "/dev/sdz1", Mountpoint: "/media/Elements", UUID: "uuid-elements", Serial: "serial-elements", Filesystem: "ntfs", TotalBytes: 100}
	mounted := false
	remediator := &decisionTestRemediator{mounted: &mounted, consistent: "unknown"}
	readiness := destinationreadiness.NewService(decisionTestInspector{identity: identity, mounted: &mounted}, nil).WithRemediator(remediator)
	publisher := &recoveryDecisionPublisher{}
	recoveryStore := &destinationreadiness.FileRecoveryStore{Root: filepath.Join(t.TempDir(), "journals")}
	flow := newRecoveryDecisionFlow(readiness, recoveryStore, &destinationreadiness.FileDecisionStore{Root: filepath.Join(t.TempDir(), "asks")}, publisher, true, nil)
	dest := destdomain.Destination{ID: "dest-repeat", Name: "elements", BackendKind: destdomain.BackendFilesystem, Location: "/media/Elements/vrooli-backups", DeviceIdentity: &identity}
	worker := newVolumeRecoveryWorker(&destmocks.FakeService{ListOut: []destdomain.Destination{dest}}, readiness, recoveryStore, flow, nil)

	if err := worker.reconcileDestination(ctx, dest); err != nil {
		t.Fatalf("first outage reconcile: %v", err)
	}
	firstEpisode, err := recoveryStore.ObserveEpisode(ctx, dest.ID, identity, false, "")
	if err != nil {
		t.Fatalf("read first episode: %v", err)
	}
	firstJournal, err := recoveryStore.Load(ctx, firstEpisode.RecoveryID)
	if err != nil {
		t.Fatalf("load first journal: %v", err)
	}
	firstPlan := firstJournal.Steps[0].Plan
	firstAskID := operatordecision.CorrelationID("dbm-repair", firstJournal.ID, firstPlan.ID, "0")
	resolution := operatordecision.DecisionResolution{
		AskID: firstAskID, State: operatordecision.StateAnswered, Answer: "approve_repair",
		AnsweredBy: "operator", AnsweredByIdentityKind: "verified_human", AnsweredByIdentitySource: "test",
		SourceEventType: operatordecision.DecisionRequestedEventType(recoveryDecisionSource, recoveryDecisionKind),
		Correlation:     map[string]string{"recovery_id": firstJournal.ID, "plan_id": firstPlan.ID, "step_index": "0"},
	}
	if handled, err := flow.Apply(ctx, resolution); err != nil || !handled {
		t.Fatalf("complete first approved episode=(%v, %v)", handled, err)
	}
	completed, err := recoveryStore.Load(ctx, firstJournal.ID)
	if err != nil || completed.State != destinationreadiness.RecoveryCompleted {
		t.Fatalf("first episode journal=(%+v, %v), want completed", completed, err)
	}

	// Simulate a later unplug before the worker observes an intervening mounted
	// poll. The completed journal must advance to a new ask instead of suppressing
	// recovery forever.
	mounted = false
	if err := worker.reconcileDestination(ctx, dest); err != nil {
		t.Fatalf("second outage reconcile: %v", err)
	}
	secondEpisode, err := recoveryStore.ObserveEpisode(ctx, dest.ID, identity, false, "")
	if err != nil || secondEpisode.RecoveryID == firstJournal.ID {
		t.Fatalf("second episode=(%+v, %v), want fresh ID after completed first episode", secondEpisode, err)
	}
	secondJournal, err := recoveryStore.Load(ctx, secondEpisode.RecoveryID)
	if err != nil || secondJournal.State != destinationreadiness.RecoveryPaused {
		t.Fatalf("second episode journal=(%+v, %v), want a newly paused repair decision", secondJournal, err)
	}
	decisionEvents := 0
	for _, event := range publisher.events {
		if event.EventType == operatordecision.DecisionRequestedEventType(recoveryDecisionSource, recoveryDecisionKind) {
			decisionEvents++
		}
	}
	if decisionEvents != 2 {
		t.Fatalf("decision events=%d, want one per outage episode; events=%+v", decisionEvents, publisher.events)
	}

	if err := worker.reconcileDestination(ctx, dest); err != nil {
		t.Fatalf("repeated second outage reconcile: %v", err)
	}
	decisionEvents = 0
	for _, event := range publisher.events {
		if event.EventType == operatordecision.DecisionRequestedEventType(recoveryDecisionSource, recoveryDecisionKind) {
			decisionEvents++
		}
	}
	if decisionEvents != 2 {
		t.Fatalf("repeated poll raised duplicate decision: count=%d events=%+v", decisionEvents, publisher.events)
	}
}

func TestVolumeRecoveryWorkerSkipsDestinationWithoutStableIdentity(t *testing.T) {
	readiness := destinationreadiness.NewService(decisionTestInspector{}, nil)
	publisher := &recoveryDecisionPublisher{}
	flow := newRecoveryDecisionFlow(readiness, &recoveryStore{},
		&destinationreadiness.FileDecisionStore{Root: filepath.Join(t.TempDir(), "asks")}, publisher, true, nil)
	dest := destdomain.Destination{ID: "legacy-destination", Name: "elements-local", BackendKind: destdomain.BackendFilesystem, Location: "/media/matthalloran8/Elements/vrooli-backups"}
	worker := newVolumeRecoveryWorker(&destmocks.FakeService{ListOut: []destdomain.Destination{dest}}, readiness,
		&destinationreadiness.FileRecoveryStore{Root: filepath.Join(t.TempDir(), "journals")}, flow, nil)
	worker.reconcile(context.Background())
	if len(publisher.events) != 0 {
		t.Fatalf("events = %+v, want no guesses or repeated alerts without stable identity", publisher.events)
	}
}

func TestVolumeRecoveryWorkerIgnoresAvailableLegacyFilesystemDestination(t *testing.T) {
	readiness := destinationreadiness.NewService(decisionTestInspector{}, nil)
	publisher := &recoveryDecisionPublisher{}
	flow := newRecoveryDecisionFlow(readiness, &recoveryStore{},
		&destinationreadiness.FileDecisionStore{Root: filepath.Join(t.TempDir(), "asks")}, publisher, true, nil)
	location := t.TempDir()
	dest := destdomain.Destination{ID: "legacy-local", Name: "local-backup", BackendKind: destdomain.BackendFilesystem, Location: location}
	worker := newVolumeRecoveryWorker(&destmocks.FakeService{ListOut: []destdomain.Destination{dest}}, readiness,
		&destinationreadiness.FileRecoveryStore{Root: filepath.Join(t.TempDir(), "journals")}, flow, nil)
	worker.reconcile(context.Background())
	if len(publisher.events) != 0 {
		t.Fatalf("events = %+v, want no mount warning for an available destination", publisher.events)
	}
}
