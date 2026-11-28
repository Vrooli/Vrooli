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
	"data-backup-manager/internal/repohealth"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
)

// The operator must never see a stale or duplicate DBM decision: one open
// decision per situation (a subject key per destination or recovery), updated
// in place while the situation changes, and withdrawn when it clears.

const (
	wantBackupKey = "backup-failure:dest-1"
	legacyEventID = "legacy-requester-ref"
)

func (h *backupHarness) withdrawals() []eventbus.DomainEvent {
	return h.events(operatordecision.DecisionWithdrawnEventType(recoveryDecisionSource, backupFailureKind))
}

func (h *backupHarness) pendingAsks() []*operatordecision.PendingAsk {
	h.t.Helper()
	all, err := h.flow.asks.ListAsks(context.Background())
	if err != nil {
		h.t.Fatal(err)
	}
	var out []*operatordecision.PendingAsk
	for _, ask := range all {
		if ask.State == operatordecision.PendingStatePending {
			out = append(out, ask)
		}
	}
	return out
}

func (h *backupHarness) askOf(inc repohealth.Incident) *operatordecision.PendingAsk {
	h.t.Helper()
	ask, err := h.flow.asks.Get(context.Background(), backupAskID(inc.ID, inc.Generation))
	if err != nil || ask == nil {
		h.t.Fatalf("ask for %s generation %d = %v, %v", inc.ID, inc.Generation, ask, err)
	}
	return ask
}

// failAgain records a fresh failed run, far enough on that the destination is
// diagnosed again.
func (h *backupHarness) failAgain(runID string) {
	h.now = h.now.Add(backupMinDiagnosisGap + time.Minute)
	h.history.runs["plan-elements"] = RunRecord{
		ID: runID, Status: "failed", Terminal: true, FinishedAt: h.now,
		Failures: []DestinationFailure{{DestinationID: "dest-1", Code: "repository_invalid"}},
	}
}

func requireKey(t *testing.T, events []eventbus.DomainEvent, want string) {
	t.Helper()
	for i, e := range events {
		if got := e.Payload["subject_key"]; got != want {
			t.Fatalf("event %d subject_key = %v, want %q", i, got, want)
		}
	}
}

func TestBackupDecisionsCarryOneSubjectKeyPerDestinationAndAreReasserted(t *testing.T) {
	h := newBackupHarness(t, tempA)
	for i := 0; i < 4; i++ {
		h.now = h.now.Add(2 * time.Minute)
		h.reconcile()
	}
	decisions := h.decisions()
	if len(decisions) != 1 {
		t.Fatalf("decision events = %d, want one however many sweeps pass", len(decisions))
	}
	requireKey(t, decisions, wantBackupKey)
	if got := decisions[0].Payload["reassert_within_seconds"]; got != int64(decisionReassertWithin/time.Second) {
		t.Fatalf("reassert_within_seconds = %v, want the hub's safety net of %s", got, decisionReassertWithin)
	}
	if decisions[0].Payload["asserted_at"] == nil {
		t.Fatal("the decision must say when DBM last saw it was still needed")
	}

	// Still blocked two hours on: the same decision is re-sent so the hub keeps
	// it open, never a second one.
	h.now = h.now.Add(decisionReassertEvery + time.Minute)
	h.reconcile()
	decisions = h.decisions()
	if len(decisions) != 2 {
		t.Fatalf("decision events = %d, want the one decision re-asserted once", len(decisions))
	}
	requireKey(t, decisions, wantBackupKey)
	if decisions[0].Payload["asserted_at"] == decisions[1].Payload["asserted_at"] {
		t.Fatal("a re-assertion must carry a fresh assertion time")
	}
	if len(h.pendingAsks()) != 1 || len(h.withdrawals()) != 0 {
		t.Fatalf("pending asks = %d, withdrawals = %d, want one open and none withdrawn", len(h.pendingAsks()), len(h.withdrawals()))
	}
}

func TestBackupDecisionIsWithdrawnWhenBackupsRecover(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	h.now = h.now.Add(time.Hour)
	h.history.runs["plan-elements"] = RunRecord{ID: "run-ok", Status: "completed", Terminal: true, FinishedAt: h.now}
	h.reconcile()

	withdrawals := h.withdrawals()
	if len(withdrawals) != 1 {
		t.Fatalf("withdrawals = %d, want one", len(withdrawals))
	}
	payload := withdrawals[0].Payload
	reason, _ := payload["reason"].(string)
	if payload["subject_key"] != wantBackupKey || payload["requester_ref"] != nil ||
		payload["source_event_type"] != operatordecision.DecisionRequestedEventType(recoveryDecisionSource, backupFailureKind) {
		t.Fatalf("withdrawal = %#v, want the destination's subject key and the request type", payload)
	}
	for _, want := range []string{"the Elements drive", "succeeded again at 13:00 UTC"} {
		if !strings.Contains(reason, want) {
			t.Errorf("reason %q omits %q", reason, want)
		}
	}
	if len(h.pendingAsks()) != 0 {
		t.Fatal("a recovered situation left a pending ask")
	}
	if len(h.notices(categoryBackupRecovered)) != 1 {
		t.Fatal("the 'working again' notice is kept, but it must not be the only signal")
	}
	h.now = h.now.Add(time.Hour)
	h.reconcile()
	if len(h.withdrawals()) != 1 {
		t.Fatalf("withdrawals = %d after another sweep, want the withdrawal to be idempotent", len(h.withdrawals()))
	}
}

func TestBackupDecisionIsWithdrawnWhenEveryCheckPassesAgain(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	h.fs.heal(tempA)
	h.failAgain("run-2")
	h.reconcile()
	withdrawals := h.withdrawals()
	if len(withdrawals) != 1 || !strings.Contains(withdrawals[0].Payload["reason"].(string), "no longer blocked") {
		t.Fatalf("withdrawals = %+v, want one saying the drive is no longer blocked", withdrawals)
	}
}

func TestBackupSignatureChangeRevisesTheSameSubjectInsteadOfWithdrawing(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	first := h.openIncident()
	h.fs.breakEntry(tempB)
	h.failAgain("run-2")
	h.reconcile()

	second := h.openIncident()
	if second.ID == first.ID {
		t.Fatal("a changed set of unreadable files must be a new condition")
	}
	decisions := h.decisions()
	if len(decisions) != 2 {
		t.Fatalf("decision events = %d, want the original and its revision", len(decisions))
	}
	requireKey(t, decisions, wantBackupKey)
	if len(h.withdrawals()) != 0 {
		t.Fatalf("withdrawals = %+v: a condition that still needs a decision is a revision, not a withdrawal", h.withdrawals())
	}
	pending := h.pendingAsks()
	if len(pending) != 1 || pending[0].ID != second.AskID {
		t.Fatalf("pending asks = %+v, want only the revised one", pending)
	}
	if old, _, _ := h.incidents.Get(context.Background(), first.ID); old.State != repohealth.IncidentSuperseded {
		t.Fatalf("earlier incident = %s, want superseded", old.State)
	}
}

func TestBackupSameConditionDiagnosedAgainUpdatesTheOpenQuestionInPlace(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	before := h.openIncident()
	h.now = h.now.Add(40 * 24 * time.Hour)
	h.failAgain("run-2")
	h.reconcile()
	after := h.openIncident()
	if after.ID != before.ID || after.Generation != before.Generation {
		t.Fatalf("incident changed from %s/%d to %s/%d", before.ID, before.Generation, after.ID, after.Generation)
	}
	decisions := h.decisions()
	if len(decisions) < 2 {
		t.Fatalf("decision events = %d, want the refreshed text published as a revision", len(decisions))
	}
	requireKey(t, decisions, wantBackupKey)
	if decisions[0].Payload["subject"] == decisions[len(decisions)-1].Payload["subject"] {
		t.Fatalf("the question still reads %q after 40 more days", decisions[0].Payload["subject"])
	}
	if len(h.pendingAsks()) != 1 {
		t.Fatalf("pending asks = %d, want exactly one", len(h.pendingAsks()))
	}
}

func TestBackupLapseAndRenewalNeverLeaveTwoOpenDecisions(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	for round := 0; round < 3; round++ {
		inc := h.openIncident()
		h.apply(h.resolution(inc, "", func(r *operatordecision.DecisionResolution) {
			r.State = operatordecision.StateExpired
			r.AnsweredBy, r.AnsweredByIdentityKind, r.AnsweredByIdentitySource = "", "", ""
		}))
		h.now = h.now.Add(time.Minute)
		h.reconcile()
		if got := len(h.pendingAsks()); got != 1 {
			t.Fatalf("round %d: pending asks = %d, want exactly one open", round, got)
		}
	}
	decisions := h.decisions()
	if len(decisions) != 4 {
		t.Fatalf("decision events = %d, want the first plus one renewal per expiry", len(decisions))
	}
	requireKey(t, decisions, wantBackupKey)
	if len(h.withdrawals()) != 0 {
		t.Fatalf("withdrawals = %+v: a renewal must update the open subject, not close it", h.withdrawals())
	}
}

func TestBackupWithdrawnResolutionIsNeverAnAnswer(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	inc := h.openIncident()

	// The hub's safety net closed the decision: the answer fields carry what the
	// approval would have said, and none of it may be acted on.
	h.apply(h.resolution(inc, repohealth.OptAccept, func(r *operatordecision.DecisionResolution) {
		r.State = operatordecision.StateWithdrawn
	}))
	if len(h.exceptions()) != 0 || len(h.history.triggered) != 0 || h.remed.calls != 0 {
		t.Fatal("a withdrawn resolution changed state")
	}
	if len(h.notices(categoryBackupNeedsAttention)) != 0 {
		t.Fatal("a withdrawal is not a refusal; no 'needs attention' notice is wanted")
	}
	if h.askOf(inc).State != operatordecision.PendingStateWithdrawn {
		t.Fatal("the closed decision must no longer count as open")
	}
	// The problem is still there, so the next sweep raises it again.
	h.now = h.now.Add(time.Minute)
	h.reconcile()
	if len(h.decisions()) != 2 || len(h.pendingAsks()) != 1 {
		t.Fatalf("decisions = %d, pending = %d, want the situation raised again once", len(h.decisions()), len(h.pendingAsks()))
	}
	requireKey(t, h.decisions(), wantBackupKey)
}

func TestBackupReplacementWithdrawalLeavesTheCurrentQuestionOpen(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	inc := h.openIncident()
	h.apply(h.resolution(inc, "", func(r *operatordecision.DecisionResolution) {
		r.State, r.SupersededByAskID = operatordecision.StateWithdrawn, "newer-ask"
	}))
	if h.askOf(inc).State != operatordecision.PendingStatePending {
		t.Fatal("the hub replaced the decision with a newer one; DBM's question must stay open")
	}
	h.now = h.now.Add(time.Minute)
	h.reconcile()
	if len(h.decisions()) != 1 {
		t.Fatalf("decision events = %d, want no extra copy for a replacement", len(h.decisions()))
	}
}

func TestBackupAnswerToAWithdrawnQuestionChangesNothingAndTellsTheOperatorOnce(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	inc := h.openIncident()
	h.now = h.now.Add(time.Hour)
	h.history.runs["plan-elements"] = RunRecord{ID: "run-ok", Status: "completed", Terminal: true, FinishedAt: h.now}
	h.reconcile() // recovered: the question is withdrawn

	late := h.resolution(inc, repohealth.OptRepair)
	h.apply(late)
	h.apply(late) // a replayed delivery
	if h.remed.calls != 0 || len(h.exceptions()) != 0 {
		t.Fatal("an answer to a withdrawn question was carried out")
	}
	if got := len(h.notices(categoryBackupNeedsAttention)); got != 1 {
		t.Fatalf("late-answer notices = %d, want exactly one however often it is replayed", got)
	}
}

func TestBackupAnswerNamingAnOlderGenerationIsNotCarriedOutAndNotSilent(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	first := h.openIncident()
	// The operator declined, and the question came back as generation 2.
	h.apply(h.resolution(first, repohealth.OptLeave))
	h.failAgain("run-2")
	h.now = h.now.Add(backupReaskAfter)
	h.reconcile()
	current := h.openIncident()
	if current.Generation != 2 {
		t.Fatalf("generation = %d, want the question asked again", current.Generation)
	}
	// A late tap on the first card still names generation 1.
	stale := h.resolution(first, repohealth.OptAccept)
	h.apply(stale)
	h.apply(stale)
	if len(h.exceptions()) != 0 || len(h.history.triggered) != 0 {
		t.Fatal("an answer to an older generation was carried out against the current question")
	}
	if h.askOf(current).State != operatordecision.PendingStatePending {
		t.Fatal("the current question must stay open after a stale answer")
	}
}

func TestBackupDecisionWithoutASubjectKeyIsAdoptedAndItsKeylessCopyClosed(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	inc := h.openIncident()
	h.makeKeyless(inc, legacyEventID)
	h.publisher.events = nil

	h.now = h.now.Add(time.Minute)
	h.reconcile()
	decisions, withdrawals := h.decisions(), h.withdrawals()
	if len(decisions) != 1 {
		t.Fatalf("decision events = %d, want the keyed decision published", len(decisions))
	}
	requireKey(t, decisions, wantBackupKey)
	if len(withdrawals) != 1 || withdrawals[0].Payload["requester_ref"] != legacyEventID || withdrawals[0].Payload["subject_key"] != nil {
		t.Fatalf("withdrawals = %+v, want the keyless copy closed by its requester ref only", withdrawals)
	}
	if ask := h.askOf(inc); ask.Extra[extraSubjectKey] != wantBackupKey || ask.Extra[extraLegacyRef] != "" {
		t.Fatalf("ask bookkeeping = %+v", ask.Extra)
	}
	h.now = h.now.Add(time.Minute)
	h.reconcile()
	if len(h.decisions()) != 1 || len(h.withdrawals()) != 1 {
		t.Fatal("adoption must be idempotent: a second sweep published again")
	}
}

func TestBackupAdoptionRetriesTheKeylessWithdrawalWhenThePublishFails(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	inc := h.openIncident()
	h.makeKeyless(inc, legacyEventID)
	h.publisher.events = nil
	// The hub refuses the keyless withdrawal for the first two attempts: the
	// adoption's own and the same sweep's settle.
	h.publisher.failType, h.publisher.failTypeTimes = operatordecision.DecisionWithdrawnEventType(recoveryDecisionSource, backupFailureKind), 2

	h.now = h.now.Add(time.Minute)
	h.reconcile()
	if len(h.withdrawals()) != 0 {
		t.Fatal("the failed publish was recorded as sent")
	}
	if h.askOf(inc).Extra[extraLegacyRef] != legacyEventID {
		t.Fatal("the keyless reference was forgotten before it was withdrawn")
	}
	h.now = h.now.Add(time.Minute)
	h.reconcile()
	if w := h.withdrawals(); len(w) != 1 || w[0].Payload["requester_ref"] != legacyEventID {
		t.Fatalf("withdrawals = %+v, want the keyless copy closed on the next sweep", w)
	}
	if len(h.decisions()) != 1 {
		t.Fatalf("decision events = %d, want the keyed decision published once", len(h.decisions()))
	}
}

func TestBackupKeylessDecisionIsClosedWhenTheSituationAlreadyCleared(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	inc := h.openIncident()
	h.makeKeyless(inc, legacyEventID)
	h.publisher.events = nil

	h.now = h.now.Add(time.Hour)
	h.history.runs["plan-elements"] = RunRecord{ID: "run-ok", Status: "completed", Terminal: true, FinishedAt: h.now}
	h.reconcile()
	if len(h.decisions()) != 0 {
		t.Fatal("a cleared situation was raised again")
	}
	w := h.withdrawals()
	if len(w) != 1 || w[0].Payload["requester_ref"] != legacyEventID || !strings.Contains(w[0].Payload["reason"].(string), "succeeded again") {
		t.Fatalf("withdrawals = %+v, want the keyless copy closed with the recovery reason", w)
	}
}

// makeKeyless rewrites an incident's ask into the shape published before subject
// keys existed: no key, and its event id is what the hub holds as requester_ref.
func (h *backupHarness) makeKeyless(inc repohealth.Incident, eventID string) {
	h.t.Helper()
	ask := h.askOf(inc)
	delete(ask.Extra, extraSubjectKey)
	delete(ask.Extra, extraAssertedAt)
	ask.EventID, ask.Published = eventID, true
	if err := h.flow.asks.Put(context.Background(), ask); err != nil {
		h.t.Fatal(err)
	}
}

func TestBackupDecisionIsWithdrawnWhenTheDestinationIsRemoved(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	h.worker.dests.(*destmocks.FakeService).ListOut = nil
	h.now = h.now.Add(time.Minute)
	h.reconcile()
	w := h.withdrawals()
	if len(w) != 1 || w[0].Payload["subject_key"] != wantBackupKey || !strings.Contains(w[0].Payload["reason"].(string), "was removed") {
		t.Fatalf("withdrawals = %+v, want one saying the destination was removed", w)
	}
	if len(h.pendingAsks()) != 0 {
		t.Fatal("a removed destination left a pending ask")
	}
}

func TestBackupDecisionIsWithdrawnWhenThePlanIsSwitchedOffAndRaisedAgainWhenItReturns(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	h.history.plans[0].Enabled = false
	h.now = h.now.Add(time.Minute)
	h.reconcile()
	w := h.withdrawals()
	if len(w) != 1 || !strings.Contains(w[0].Payload["reason"].(string), "switched on") {
		t.Fatalf("withdrawals = %+v, want one saying no plan is switched on", w)
	}
	withdrawnAt := parseAssertedAt(w[0].Payload["asserted_at"].(string))

	h.history.plans[0].Enabled = true
	h.failAgain("run-2")
	h.reconcile()
	decisions := h.decisions()
	if len(decisions) != 2 || len(h.pendingAsks()) != 1 {
		t.Fatalf("decisions = %d, pending = %d, want the situation raised again once it is real again", len(decisions), len(h.pendingAsks()))
	}
	requireKey(t, decisions, wantBackupKey)
	if raisedAt := parseAssertedAt(decisions[1].Payload["asserted_at"].(string)); raisedAt.Before(withdrawnAt) {
		t.Fatalf("the new decision (%s) is asserted before the withdrawal (%s), so the hub would drop it as stale", raisedAt, withdrawnAt)
	}
}

func TestBackupDecisionIsWithdrawnWhenTheDriveIsNoLongerMounted(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	*h.mounted = false
	h.failAgain("run-2")
	h.reconcile()
	w := h.withdrawals()
	if len(w) != 1 || !strings.Contains(w[0].Payload["reason"].(string), "not mounted") {
		t.Fatalf("withdrawals = %+v, want the backup question closed because drive recovery owns the problem", w)
	}
}

func TestBackupSettleClosesOnlyWhatIsNoLongerWanted(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	if err := h.flow.Settle(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(h.withdrawals()) != 0 || len(h.pendingAsks()) != 1 {
		t.Fatalf("settle closed a wanted decision: withdrawals=%d pending=%d", len(h.withdrawals()), len(h.pendingAsks()))
	}
}

// --- volume repair decisions -------------------------------------------------

type volumeHarness struct {
	t         *testing.T
	now       time.Time
	flow      *RecoveryDecisionFlow
	worker    *VolumeRecoveryWorker
	recovery  *destinationreadiness.FileRecoveryStore
	publisher *recoveryDecisionPublisher
	remed     *decisionTestRemediator
	dest      destdomain.Destination
	mounted   *bool
}

func newVolumeHarness(t *testing.T) *volumeHarness {
	t.Helper()
	identity := destinationreadiness.DeviceIdentity{DevicePath: "/dev/sdz1", Mountpoint: "/media/Elements", Label: "Elements", UUID: "uuid-elements", Serial: "serial-elements", Filesystem: "ntfs", TotalBytes: 100}
	mounted := false
	h := &volumeHarness{t: t, now: backupTestNow, mounted: &mounted, publisher: &recoveryDecisionPublisher{}}
	h.remed = &decisionTestRemediator{mounted: &mounted, consistent: "unknown"}
	readiness := destinationreadiness.NewService(decisionTestInspector{identity: identity, mounted: &mounted}, nil).WithRemediator(h.remed)
	dir := t.TempDir()
	h.recovery = &destinationreadiness.FileRecoveryStore{Root: filepath.Join(dir, "journals")}
	h.flow = newRecoveryDecisionFlow(readiness, h.recovery, &destinationreadiness.FileDecisionStore{Root: filepath.Join(dir, "asks")}, h.publisher, true, nil)
	h.flow.now = func() time.Time { return h.now }
	h.dest = destdomain.Destination{ID: "dest-vol", Name: "elements", BackendKind: destdomain.BackendFilesystem, Location: "/media/Elements/vrooli-backups", DeviceIdentity: &identity}
	h.worker = newVolumeRecoveryWorker(&destmocks.FakeService{ListOut: []destdomain.Destination{h.dest}}, readiness, h.recovery, h.flow, nil)
	return h
}

func (h *volumeHarness) reconcile() {
	h.t.Helper()
	h.worker.reconcile(context.Background())
}

func (h *volumeHarness) requests() []eventbus.DomainEvent {
	return h.eventsOf(operatordecision.DecisionRequestedEventType(recoveryDecisionSource, recoveryDecisionKind))
}

func (h *volumeHarness) withdrawals() []eventbus.DomainEvent {
	return h.eventsOf(operatordecision.DecisionWithdrawnEventType(recoveryDecisionSource, recoveryDecisionKind))
}

func (h *volumeHarness) eventsOf(eventType string) []eventbus.DomainEvent {
	var out []eventbus.DomainEvent
	for _, e := range h.publisher.events {
		if e.EventType == eventType {
			out = append(out, e)
		}
	}
	return out
}

func (h *volumeHarness) journalID() string {
	h.t.Helper()
	episode, err := h.recovery.ObserveEpisode(context.Background(), h.dest.ID, *h.dest.DeviceIdentity, false, "", "")
	if err != nil || episode.RecoveryID == "" {
		h.t.Fatalf("episode = %+v, %v", episode, err)
	}
	return episode.RecoveryID
}

func (h *volumeHarness) pending() []*operatordecision.PendingAsk {
	h.t.Helper()
	all, err := h.flow.asks.ListAsks(context.Background())
	if err != nil {
		h.t.Fatal(err)
	}
	var out []*operatordecision.PendingAsk
	for _, ask := range all {
		if ask.State == operatordecision.PendingStatePending {
			out = append(out, ask)
		}
	}
	return out
}

func (h *volumeHarness) approval(askID, journalID string, state string) operatordecision.DecisionResolution {
	journal, err := h.recovery.Load(context.Background(), journalID)
	if err != nil {
		h.t.Fatal(err)
	}
	return operatordecision.DecisionResolution{
		AskID: askID, State: state, Answer: "approve_repair",
		AnsweredBy: "operator", AnsweredByIdentityKind: "verified_human", AnsweredByIdentitySource: "test",
		SourceEventType: operatordecision.DecisionRequestedEventType(recoveryDecisionSource, recoveryDecisionKind),
		Correlation:     map[string]string{"recovery_id": journalID, "plan_id": journal.Steps[0].Plan.ID, "step_index": "0"},
	}
}

func TestVolumeRepairDecisionIsKeyedPerRecoveryAndReasserted(t *testing.T) {
	h := newVolumeHarness(t)
	h.reconcile()
	journalID := h.journalID()
	want := "volume-repair:" + journalID
	for i := 0; i < 3; i++ {
		h.now = h.now.Add(time.Minute)
		h.reconcile()
	}
	if got := h.requests(); len(got) != 1 {
		t.Fatalf("requests = %d, want one however many sweeps pass", len(got))
	}
	requireKey(t, h.requests(), want)
	h.now = h.now.Add(decisionReassertEvery + time.Minute)
	h.reconcile()
	if got := h.requests(); len(got) != 2 || len(h.pending()) != 1 {
		t.Fatalf("requests = %d, pending = %d, want the one decision re-asserted", len(got), len(h.pending()))
	}
	requireKey(t, h.requests(), want)
}

func TestVolumeRepairDecisionIsWithdrawnWhenTheDriveIsMountedAgain(t *testing.T) {
	h := newVolumeHarness(t)
	h.reconcile()
	journalID := h.journalID()
	*h.mounted = true // the operator mounted it in DBM or at the desk
	h.now = h.now.Add(time.Minute)
	h.reconcile()
	w := h.withdrawals()
	if len(w) != 1 || w[0].Payload["subject_key"] != "volume-repair:"+journalID || !strings.Contains(w[0].Payload["reason"].(string), "mounted and writable again") {
		t.Fatalf("withdrawals = %+v, want one closing the repair question because the drive is back", w)
	}
	if len(h.pending()) != 0 {
		t.Fatal("a mounted drive left a pending repair question")
	}
	h.now = h.now.Add(time.Minute)
	h.reconcile()
	if len(h.withdrawals()) != 1 {
		t.Fatal("the withdrawal must be idempotent")
	}
}

func TestVolumeRepairDecisionIsWithdrawnWhenTheOperatorCompletesTheRecoveryInDBM(t *testing.T) {
	h := newVolumeHarness(t)
	h.reconcile()
	journalID := h.journalID()
	h.flow.NotifyRecoveryCompleted(context.Background(), destinationreadiness.RecoveryJournal{ID: journalID, Location: "/media/Elements"})
	if w := h.withdrawals(); len(w) != 1 || !strings.Contains(w[0].Payload["reason"].(string), "recovery finished") {
		t.Fatalf("withdrawals = %+v, want the repair question closed when the recovery finished", w)
	}
	if len(h.pending()) != 0 {
		t.Fatal("a finished recovery left a pending repair question")
	}
}

func TestVolumeRepairDecisionIsWithdrawnWhenTheDestinationIsRemoved(t *testing.T) {
	h := newVolumeHarness(t)
	h.reconcile()
	h.worker.destinations.(*destmocks.FakeService).ListOut = nil
	h.now = h.now.Add(time.Minute)
	h.reconcile()
	if w := h.withdrawals(); len(w) != 1 || !strings.Contains(w[0].Payload["reason"].(string), "was removed") {
		t.Fatalf("withdrawals = %+v, want one saying the destination was removed", w)
	}
	if len(h.pending()) != 0 {
		t.Fatal("a removed destination left a pending repair question")
	}
}

func TestVolumeRepairDecisionWithoutASubjectKeyIsAdopted(t *testing.T) {
	h := newVolumeHarness(t)
	h.reconcile()
	ask := h.pending()[0]
	delete(ask.Extra, extraSubjectKey)
	delete(ask.Extra, extraAssertedAt)
	ask.EventID, ask.Published = legacyEventID, true
	if err := h.flow.asks.Put(context.Background(), ask); err != nil {
		t.Fatal(err)
	}
	h.publisher.events = nil
	h.now = h.now.Add(time.Minute)
	h.reconcile()
	if len(h.requests()) != 1 {
		t.Fatalf("requests = %d, want the keyed decision published", len(h.requests()))
	}
	requireKey(t, h.requests(), "volume-repair:"+h.journalID())
	if w := h.withdrawals(); len(w) != 1 || w[0].Payload["requester_ref"] != legacyEventID {
		t.Fatalf("withdrawals = %+v, want the keyless copy closed by its requester ref", w)
	}
	h.now = h.now.Add(time.Minute)
	h.reconcile()
	if len(h.requests()) != 1 || len(h.withdrawals()) != 1 {
		t.Fatal("adoption must be idempotent")
	}
}

func TestVolumeRepairWithdrawnResolutionIsNeverApprovalAndTheQuestionReturns(t *testing.T) {
	h := newVolumeHarness(t)
	h.reconcile()
	journalID := h.journalID()
	ask := h.pending()[0]
	callsBefore := h.remed.calls
	res := h.approval(ask.ID, journalID, operatordecision.StateWithdrawn)
	if handled, err := h.flow.Apply(context.Background(), res); err != nil || !handled {
		t.Fatalf("Apply = %v, %v", handled, err)
	}
	if h.remed.calls != callsBefore {
		t.Fatal("a withdrawn resolution ran a repair")
	}
	// The drive is still down, so the repair question is raised again.
	h.now = h.now.Add(time.Minute)
	h.reconcile()
	if len(h.requests()) != 2 || len(h.pending()) != 1 {
		t.Fatalf("requests = %d, pending = %d, want the question asked again once", len(h.requests()), len(h.pending()))
	}
}

func TestVolumeRepairAnswerToAWithdrawnQuestionRunsNothingAndTellsTheOperatorOnce(t *testing.T) {
	h := newVolumeHarness(t)
	h.reconcile()
	journalID := h.journalID()
	ask := h.pending()[0]
	*h.mounted = true
	h.now = h.now.Add(time.Minute)
	h.reconcile() // withdrawn: the drive is back
	callsBefore := h.remed.calls
	res := h.approval(ask.ID, journalID, operatordecision.StateAnswered)
	for i := 0; i < 2; i++ {
		if handled, err := h.flow.Apply(context.Background(), res); err != nil || !handled {
			t.Fatalf("Apply = %v, %v", handled, err)
		}
	}
	if h.remed.calls != callsBefore {
		t.Fatal("an approval of a withdrawn question ran a repair")
	}
	notices := 0
	for _, e := range h.publisher.events {
		if e.EventType == recoveryNotificationType && strings.Contains(e.Payload["title"].(string), "already closed") {
			notices++
		}
	}
	if notices != 1 {
		t.Fatalf("late-answer notices = %d, want exactly one", notices)
	}
}

func TestBackupWithdrawalReasonNamesTheStateItActuallyIsIn(t *testing.T) {
	inc := repohealth.Incident{Generation: 4}
	inc.Diagnosis.DestinationName = "the Elements drive"
	cases := []struct {
		state      repohealth.IncidentState
		generation int
		want       string
	}{
		{repohealth.IncidentDeclined, 4, "dismissed or declined"},
		{repohealth.IncidentDeclined, 3, "earlier round"},
		{repohealth.IncidentFailed, 4, "did not work"},
		{repohealth.IncidentLapsed, 4, "expired unanswered"},
		{repohealth.IncidentApplied, 4, "was carried out"},
		{repohealth.IncidentAgentRequested, 4, "asked for an agent"},
		{repohealth.IncidentApplying, 4, "being carried out"},
	}
	for _, c := range cases {
		inc.State = c.state
		got := backupWithdrawalReason(inc, c.generation, true, false)
		if !strings.Contains(got, c.want) || strings.Contains(got, "being handled") {
			t.Errorf("%s gen %d: reason %q, want it to say %q", c.state, c.generation, got, c.want)
		}
	}
}

// The live case: an older round's decision was still pending after the operator
// dismissed the newest round, which left the incident declined.
func TestBackupOlderRoundsPendingDecisionIsClosedWithAnHonestReason(t *testing.T) {
	h := newBackupHarness(t, tempA)
	h.reconcile()
	first := h.openIncident()
	h.apply(h.resolution(first, repohealth.OptLeave)) // declined; the first round's ask is finished
	zombie := *h.askOf(first)
	zombie.State, zombie.ResolvedAt = operatordecision.PendingStatePending, nil
	zombie.Extra = map[string]string{"incident_id": first.ID, "generation": "0", "destination_id": first.DestinationID}
	zombie.ID = backupAskID(first.ID, 0)
	if err := h.flow.asks.Put(context.Background(), &zombie); err != nil {
		t.Fatal(err)
	}
	h.publisher.events = nil
	h.reconcile()
	w := h.withdrawals()
	if len(w) != 1 || strings.Contains(w[0].Payload["reason"].(string), "being handled") {
		t.Fatalf("withdrawals = %+v, want the stale round closed with a reason that names its state", w)
	}
}
