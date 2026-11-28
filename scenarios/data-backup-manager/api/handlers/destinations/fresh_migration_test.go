package destinations

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"data-backup-manager/internal/repohealth"
	"github.com/vrooli/api-core/operatordecision"
)

type freshMigrationStub struct {
	plan         FreshMigrationPlan
	candidates   []FreshMigrationCandidate
	registerArgs []string
	beginArgs    []string
	resumeCalls  int
	resumeErr    error
}

func (s *freshMigrationStub) Candidates(context.Context, repohealth.Incident) (FreshMigrationPlan, []FreshMigrationCandidate, error) {
	return s.plan, s.candidates, nil
}

func (s *freshMigrationStub) RegisterLocalDestination(_ context.Context, migrationID, planID, oldDestinationID, location string, targets []string) (FreshMigrationCandidate, error) {
	s.registerArgs = []string{migrationID, planID, oldDestinationID, location, fmt.Sprint(targets)}
	candidate := FreshMigrationCandidate{DestinationID: "registered-local", Name: "dbm-fresh-test", Location: location}
	s.candidates = append(s.candidates, candidate)
	return candidate, nil
}

func (s *freshMigrationStub) Begin(_ context.Context, id, planID, oldID, newID string, targets []string) error {
	s.beginArgs = []string{id, planID, oldID, newID, fmt.Sprint(targets)}
	return nil
}

func (s *freshMigrationStub) Resume(context.Context, string) (FreshMigrationProgress, error) {
	s.resumeCalls++
	return FreshMigrationProgress{Stage: "complete", Complete: true}, s.resumeErr
}

func TestFreshMigrationRequiresSecondVerifiedDestinationChoiceAndQueuesCompletionNotice(t *testing.T) {
	h := newBackupHarness(t, tempA)
	targets := make([]string, 14)
	for i := range targets {
		targets[i] = fmt.Sprintf("target-%02d", i)
	}
	migration := &freshMigrationStub{plan: FreshMigrationPlan{PlanID: "plan-14", Targets: targets}, candidates: []FreshMigrationCandidate{{DestinationID: "new-destination", Name: "backup-offhost", Location: "/mnt/backups/repositories/dbm"}}}
	h.flow.migrations = migration
	h.reconcile()
	incident := h.openIncident()
	h.apply(h.resolution(incident, repohealth.OptFresh))
	if migration.beginArgs != nil {
		t.Fatal("fresh approval alone started migration without destination callback")
	}
	migrationID := repohealth.FreshMigrationID(incident.ID, incident.Generation)
	task, err := h.flow.asks.Get(context.Background(), freshDestinationAskID(migrationID, 1))
	if err != nil || task == nil || task.State != operatordecision.PendingStatePending {
		t.Fatalf("destination choice ask not persisted: ask=%+v err=%v", task, err)
	}
	selected := operatordecision.DecisionResolution{
		AskID: task.ID, State: operatordecision.StateAnswered, Answer: "dest:new-destination",
		AnsweredBy: "operator", AnsweredByIdentityKind: "verified_human", AnsweredByIdentitySource: "cloudflare_access",
		SourceEventType: operatordecision.DecisionRequestedEventType(recoveryDecisionSource, backupFailureKind),
		Correlation:     map[string]string{"decision_type": freshDestinationDecisionType, "incident_id": incident.ID, "generation": fmt.Sprint(incident.Generation), "migration_id": migrationID, "selection_generation": "1"},
	}
	if handled, err := h.flow.Apply(context.Background(), selected); err != nil || !handled {
		t.Fatalf("destination selection Apply=(%v,%v)", handled, err)
	}
	got, found, err := h.incidents.Get(context.Background(), incident.ID)
	if err != nil || !found {
		t.Fatalf("read selected incident: found=%v err=%v", found, err)
	}
	if got.FreshMigrationStage != "migrating" || got.FreshMigrationDestinationID != "new-destination" || got.FreshMigrationPlanID != "plan-14" || len(got.FreshMigrationTargets) != 14 || got.FreshMigrationSelectionAskID != task.ID {
		t.Fatalf("verified selection was not durably bound: %+v", got)
	}
	if len(migration.beginArgs) != 5 || migration.beginArgs[3] != "new-destination" {
		t.Fatalf("migration journal inputs = %v", migration.beginArgs)
	}
	// Simulate Notification Hub being unavailable after the migration journal
	// has completed. The durable ask result must remain replayable.
	h.publisher.failNext = 1
	if err := h.flow.Settle(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := h.flow.WaitFreshMigrations(ctx); err != nil {
		t.Fatal(err)
	}
	got, _, _ = h.incidents.Get(context.Background(), incident.ID)
	if got.FreshMigrationStage != "complete" || migration.resumeCalls == 0 {
		t.Fatalf("migration completion not reconciled: incident=%+v resumes=%d", got, migration.resumeCalls)
	}
	task, err = h.flow.asks.Get(context.Background(), task.ID)
	if err != nil || task.Extra["result_notification_pending"] != "true" {
		t.Fatalf("undelivered completion result was not retained in the outbox: ask=%+v err=%v", task, err)
	}
	if err := h.flow.RetryPendingResultNotices(context.Background()); err != nil {
		t.Fatalf("retry durable completion notice: %v", err)
	}
	task, err = h.flow.asks.Get(context.Background(), task.ID)
	if err != nil || task.Extra["result_notification_pending"] != "false" {
		t.Fatalf("completion outbox was not acknowledged: ask=%+v err=%v", task, err)
	}
	if notes := h.notices(categoryBackupRecovered + "-fresh-migration-complete"); len(notes) != 1 {
		t.Fatalf("completion result notice count=%d, want one", len(notes))
	}
}

func TestFreshMigrationDeferWaitsBeforeReaskingAndNeverStartsEffects(t *testing.T) {
	h := newBackupHarness(t, tempA)
	targets := make([]string, 14)
	for i := range targets {
		targets[i] = fmt.Sprintf("target-%02d", i)
	}
	migration := &freshMigrationStub{plan: FreshMigrationPlan{PlanID: "plan-14", Targets: targets}, candidates: []FreshMigrationCandidate{{DestinationID: "new-destination", Name: "backup-offhost"}}}
	h.flow.migrations = migration
	h.reconcile()
	incident := h.openIncident()
	h.apply(h.resolution(incident, repohealth.OptFresh))
	id := repohealth.FreshMigrationID(incident.ID, incident.Generation)
	ask, err := h.flow.asks.Get(context.Background(), freshDestinationAskID(id, 1))
	if err != nil || ask == nil {
		t.Fatalf("read selection decision: ask=%+v err=%v", ask, err)
	}
	deferred := operatordecision.DecisionResolution{
		AskID: ask.ID, State: operatordecision.StateAnswered, Answer: "defer",
		AnsweredBy: "operator", AnsweredByIdentityKind: "verified_human", AnsweredByIdentitySource: "cloudflare_access",
		SourceEventType: operatordecision.DecisionRequestedEventType(recoveryDecisionSource, backupFailureKind),
		Correlation:     map[string]string{"decision_type": freshDestinationDecisionType, "incident_id": incident.ID, "generation": fmt.Sprint(incident.Generation), "migration_id": id, "selection_generation": "1"},
	}
	if handled, err := h.flow.Apply(context.Background(), deferred); err != nil || !handled {
		t.Fatalf("defer callback Apply=(%v,%v)", handled, err)
	}
	got, _, _ := h.incidents.Get(context.Background(), incident.ID)
	if got.FreshMigrationStage != "selection_deferred" || got.FreshMigrationSelectionRetryAt.Sub(h.now) != 24*time.Hour || migration.beginArgs != nil {
		t.Fatalf("defer state=%+v begin=%v", got, migration.beginArgs)
	}
	if err := h.flow.Settle(context.Background()); err != nil {
		t.Fatal(err)
	}
	if ask, _ := h.flow.asks.Get(context.Background(), freshDestinationAskID(id, 2)); ask != nil {
		t.Fatal("destination was re-asked before the defer interval elapsed")
	}
	h.now = h.now.Add(24 * time.Hour)
	if err := h.flow.Settle(context.Background()); err != nil {
		t.Fatal(err)
	}
	if ask, _ := h.flow.asks.Get(context.Background(), freshDestinationAskID(id, 2)); ask == nil || ask.State != operatordecision.PendingStatePending {
		t.Fatalf("destination selection was not reissued after defer interval: %+v", ask)
	}
	if migration.beginArgs != nil {
		t.Fatalf("defer unexpectedly started migration: %v", migration.beginArgs)
	}
}

func TestFreshMigrationEmptyMenuRequiresPathProposalThenVerifiedRegistrationAndSelection(t *testing.T) {
	h := newBackupHarness(t, tempA)
	targets := make([]string, 14)
	for i := range targets {
		targets[i] = fmt.Sprintf("target-%02d", i)
	}
	migration := &freshMigrationStub{plan: FreshMigrationPlan{PlanID: "plan-14", Targets: targets}}
	h.flow.migrations = migration
	h.reconcile()
	incident := h.openIncident()
	h.apply(h.resolution(incident, repohealth.OptFresh))
	id := repohealth.FreshMigrationID(incident.ID, incident.Generation)
	selection, err := h.flow.asks.Get(context.Background(), freshDestinationAskID(id, 1))
	if err != nil || selection == nil || selection.State != operatordecision.PendingStatePending {
		t.Fatalf("empty eligible menu did not remain actionable: ask=%+v err=%v", selection, err)
	}
	propose := operatordecision.DecisionResolution{
		AskID: selection.ID, State: operatordecision.StateAnswered, Answer: "register-local", Note: "/mnt/recovery-backups",
		AnsweredBy: "operator", AnsweredByIdentityKind: "verified_human", AnsweredByIdentitySource: "cloudflare_access",
		SourceEventType: operatordecision.DecisionRequestedEventType(recoveryDecisionSource, backupFailureKind),
		Correlation:     map[string]string{"decision_type": freshDestinationDecisionType, "incident_id": incident.ID, "generation": fmt.Sprint(incident.Generation), "migration_id": id, "selection_generation": "1"},
	}
	if handled, err := h.flow.Apply(context.Background(), propose); err != nil || !handled {
		t.Fatalf("path proposal Apply=(%v,%v)", handled, err)
	}
	if migration.registerArgs != nil {
		t.Fatal("path text provisioned a repository without a separate verified confirmation")
	}
	registration, err := h.flow.asks.Get(context.Background(), freshDestinationRegistrationAskID(id, 1))
	if err != nil || registration == nil || registration.State != operatordecision.PendingStatePending {
		t.Fatalf("registration confirmation was not raised: ask=%+v err=%v", registration, err)
	}
	confirm := operatordecision.DecisionResolution{
		AskID: registration.ID, State: operatordecision.StateAnswered, Answer: "register",
		AnsweredBy: "operator", AnsweredByIdentityKind: "verified_human", AnsweredByIdentitySource: "cloudflare_access",
		SourceEventType: operatordecision.DecisionRequestedEventType(recoveryDecisionSource, backupFailureKind),
		Correlation:     map[string]string{"decision_type": freshDestinationRegistrationDecisionType, "incident_id": incident.ID, "generation": fmt.Sprint(incident.Generation), "migration_id": id, "selection_generation": "1"},
	}
	if handled, err := h.flow.Apply(context.Background(), confirm); err != nil || !handled {
		t.Fatalf("registration confirmation Apply=(%v,%v)", handled, err)
	}
	if len(migration.registerArgs) != 5 || migration.registerArgs[1] != "plan-14" || migration.registerArgs[3] != "/mnt/recovery-backups" || len(targets) != 14 || migration.beginArgs != nil {
		t.Fatalf("registration owner inputs=%v migration begin=%v", migration.registerArgs, migration.beginArgs)
	}
	registeredSelection, err := h.flow.asks.Get(context.Background(), freshDestinationAskID(id, 2))
	if err != nil || registeredSelection == nil || registeredSelection.State != operatordecision.PendingStatePending {
		t.Fatalf("registered destination was not offered for explicit selection: ask=%+v err=%v", registeredSelection, err)
	}
	selectLocal := operatordecision.DecisionResolution{
		AskID: registeredSelection.ID, State: operatordecision.StateAnswered, Answer: "dest:registered-local",
		AnsweredBy: "operator", AnsweredByIdentityKind: "verified_human", AnsweredByIdentitySource: "cloudflare_access",
		SourceEventType: operatordecision.DecisionRequestedEventType(recoveryDecisionSource, backupFailureKind),
		Correlation:     map[string]string{"decision_type": freshDestinationDecisionType, "incident_id": incident.ID, "generation": fmt.Sprint(incident.Generation), "migration_id": id, "selection_generation": "2"},
	}
	if handled, err := h.flow.Apply(context.Background(), selectLocal); err != nil || !handled {
		t.Fatalf("registered destination selection Apply=(%v,%v)", handled, err)
	}
	if len(migration.beginArgs) != 5 || migration.beginArgs[3] != "registered-local" {
		t.Fatalf("migration did not use separately selected registered destination: %v", migration.beginArgs)
	}
}

func TestFreshMigrationLocalRegistrationRejectsProviderAndCredentialText(t *testing.T) {
	for _, invalid := range []string{"s3://bucket/prefix", "secret://credential-value", "relative/path"} {
		t.Run(invalid, func(t *testing.T) {
			h := newBackupHarness(t, tempA)
			targets := make([]string, 14)
			for i := range targets {
				targets[i] = fmt.Sprintf("target-%02d", i)
			}
			migration := &freshMigrationStub{plan: FreshMigrationPlan{PlanID: "plan-14", Targets: targets}}
			h.flow.migrations = migration
			h.reconcile()
			incident := h.openIncident()
			h.apply(h.resolution(incident, repohealth.OptFresh))
			id := repohealth.FreshMigrationID(incident.ID, incident.Generation)
			ask, _ := h.flow.asks.Get(context.Background(), freshDestinationAskID(id, 1))
			if ask == nil {
				t.Fatal("fresh destination ask missing")
			}
			proposal := operatordecision.DecisionResolution{
				AskID: ask.ID, State: operatordecision.StateAnswered, Answer: "register-local", Note: invalid,
				AnsweredBy: "operator", AnsweredByIdentityKind: "verified_human", AnsweredByIdentitySource: "cloudflare_access",
				SourceEventType: operatordecision.DecisionRequestedEventType(recoveryDecisionSource, backupFailureKind),
				Correlation:     map[string]string{"decision_type": freshDestinationDecisionType, "incident_id": incident.ID, "generation": fmt.Sprint(incident.Generation), "migration_id": id, "selection_generation": "1"},
			}
			if handled, err := h.flow.Apply(context.Background(), proposal); err != nil || !handled {
				t.Fatalf("invalid path proposal Apply=(%v,%v)", handled, err)
			}
			if migration.registerArgs != nil {
				t.Fatalf("invalid path text reached registration owner: %v", migration.registerArgs)
			}
			if ask, _ := h.flow.asks.Get(context.Background(), freshDestinationRegistrationAskID(id, 1)); ask != nil {
				t.Fatal("invalid path text raised a provisioning confirmation")
			}
		})
	}
}

func TestFreshSetQuestionsArriveAsPagesTheOperatorCanAnswerCold(t *testing.T) {
	inc := repohealth.Incident{
		ID: "bf-1", Generation: 2, FreshMigrationID: "fm-1", FreshMigrationProposedLocation: "/media/u/Spare/vrooli-backups",
		Diagnosis: repohealth.Diagnosis{Volume: repohealth.VolumeState{Label: "Elements"}},
	}
	plan := FreshMigrationPlan{PlanID: "plan-1", Targets: make([]string, 14)}
	facts, err := freshSelectionFacts(inc, plan, []FreshMigrationCandidate{{DestinationID: "dest-spare", Name: "spare-local", Location: "/media/u/Spare"}})
	if err != nil {
		t.Fatal(err)
	}
	facts.Correlation = map[string]string{"k": "v"}
	if err := facts.Validate(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(facts.Question+facts.BigPicture, "plan-1") || !strings.Contains(facts.Question, "the Elements drive") {
		t.Errorf("question %q names the plan id or not the drive", facts.Question)
	}
	for _, want := range []string{
		"Items to copy", ">14<", "Ready destinations", "spare-local", "/media/u/Spare",
		"Restore each copy to a scratch folder", `data-od-answer="dest:dest-spare"`, `data-od-answer="register-local"`, "If you do not answer:",
	} {
		if !strings.Contains(facts.Explainer, want) {
			t.Errorf("selection page is missing %q", want)
		}
	}

	confirm := freshRegistrationFacts(inc)
	confirm.Correlation = map[string]string{"k": "v"}
	if err := confirm.Validate(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(confirm.Question, "“vrooli-backups”") {
		t.Errorf("confirmation question %q should name the folder", confirm.Question)
	}
	for _, f := range []operatordecision.Facts{facts, confirm} {
		if problems := operatordecision.CoverProblems(f); len(problems) > 0 {
			t.Errorf("a cold reader cannot parse %q: %v", f.Question, problems)
		}
	}
	for _, want := range []string{"/media/u/Spare/vrooli-backups", "Encryption", `data-od-answer="register"`} {
		if !strings.Contains(confirm.Explainer, want) {
			t.Errorf("confirmation page is missing %q", want)
		}
	}
}
