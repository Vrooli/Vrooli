package operatordecision

import (
	"context"
	"strings"
	"testing"
	"time"
)

func subjectFacts(question string) Facts {
	return Facts{
		Subject: "Backups blocked", Question: question, Recommended: "retry", Urgency: "high",
		Options:    []Option{{Key: "retry", Label: "Retry"}, {Key: "pause", Label: "Pause"}},
		SubjectKey: "backup-failure:dest-1",
	}
}

func TestFactsSubjectKeyPayloadAndValidation(t *testing.T) {
	facts := subjectFacts("Retry the backup?")
	facts.ReassertWithin = 2 * time.Hour
	facts.AssertedAt = time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	if err := facts.Validate(); err != nil {
		t.Fatalf("valid subject facts refused: %v", err)
	}
	payload := facts.Payload()
	if payload["subject_key"] != "backup-failure:dest-1" || payload["reassert_within_seconds"] != int64(7200) || payload["asserted_at"] != "2026-11-13T09:00:00Z" {
		t.Fatalf("subject fields missing from payload: %#v", payload)
	}

	plain := subjectFacts("Retry the backup?")
	plain.SubjectKey = ""
	for _, key := range []string{"subject_key", "reassert_within_seconds", "asserted_at"} {
		if _, present := plain.Payload()[key]; present {
			t.Fatalf("%s must be omitted when unset, so keyless producers emit unchanged bytes", key)
		}
	}

	for name, mutate := range map[string]func(*Facts){
		"bad key":              func(f *Facts) { f.SubjectKey = "-starts-with-dash" },
		"key too long":         func(f *Facts) { f.SubjectKey = strings.Repeat("a", 129) },
		"reassert without key": func(f *Facts) { f.SubjectKey = ""; f.ReassertWithin = time.Hour },
		"reassert too short":   func(f *Facts) { f.ReassertWithin = time.Second },
		"reassert too long":    func(f *Facts) { f.ReassertWithin = 31 * 24 * time.Hour },
	} {
		f := subjectFacts("Retry the backup?")
		mutate(&f)
		if err := f.Validate(); err == nil {
			t.Fatalf("%s: expected a validation error", name)
		}
	}
}

func TestWithdrawalEventShape(t *testing.T) {
	if got := DecisionWithdrawnEventType("data-backup-manager", "backup_failure"); got != "data-backup-manager.backup_failure.decision_withdrawn.v1" {
		t.Fatalf("withdrawn type = %q", got)
	}
	if got := RequestEventTypeFor("a.b.decision_withdrawn.v1"); got != "a.b.decision_requested.v1" {
		t.Fatalf("request type = %q", got)
	}
	if RequestEventTypeFor("a.b.notification.v1") != "" {
		t.Fatalf("a non-withdrawal type must map to empty")
	}
	occurred := time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)
	event := BuildWithdrawalEventTyped("agent-manager", "agent_manager.usage_mode.decision_requested.v1", occurred, Withdrawal{SubjectKey: "usage-mode:um-1", Reason: "The mode was turned off."})
	if event.EventType != "agent_manager.usage_mode.decision_withdrawn.v1" || event.Source != "agent-manager" {
		t.Fatalf("event identity wrong: %#v", event)
	}
	if event.Payload["source_event_type"] != "agent_manager.usage_mode.decision_requested.v1" || event.Payload["subject_key"] != "usage-mode:um-1" || event.Payload["asserted_at"] != "2026-11-13T10:00:00Z" {
		t.Fatalf("payload wrong: %#v", event.Payload)
	}
	if _, present := event.Payload["requester_ref"]; present {
		t.Fatalf("requester_ref must be omitted when unset")
	}
	for name, w := range map[string]Withdrawal{
		"no identity": {Reason: "gone"},
		"no reason":   {SubjectKey: "k"},
		"long reason": {SubjectKey: "k", Reason: strings.Repeat("x", MaxWithdrawalReason+1)},
		"bad key":     {SubjectKey: "bad key", Reason: "gone"},
	} {
		if w.Validate() == nil {
			t.Fatalf("%s: expected a validation error", name)
		}
	}
}

func TestRaiserSubjectKeyedFactsChangePublishNewRevision(t *testing.T) {
	clock := &fakeClock{t: time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)}
	store := newMemStore()
	pub := &capturePublisher{}
	r := Raiser{Source: "data-backup-manager", Kind: "backup_failure", Store: store, Publisher: pub, Clock: clock}
	ask := PendingAsk{ID: "dbm-1", Subject: "dest-1"}

	if _, err := r.Raise(context.Background(), ask, subjectFacts("Retry the backup?")); err != nil {
		t.Fatal(err)
	}
	if got := pub.events[0].Payload["asserted_at"]; got != "2026-11-13T12:00:00Z" {
		t.Fatalf("a subject-keyed raise must carry the generation's asserted_at, got %v", got)
	}
	// The same facts again: idempotent.
	if _, err := r.Raise(context.Background(), ask, subjectFacts("Retry the backup?")); err != nil || len(pub.events) != 1 {
		t.Fatalf("same facts must not republish: events=%d err=%v", len(pub.events), err)
	}
	// Changed facts inside the generation: a keyless ask would refuse; a keyed
	// ask publishes the new revision and stores its event id.
	if _, err := r.Raise(context.Background(), ask, subjectFacts("Retry the backup now that the drive is back?")); err != nil {
		t.Fatalf("changed subject-keyed facts refused: %v", err)
	}
	if len(pub.events) != 2 {
		t.Fatalf("changed facts must publish a revision, got %d events", len(pub.events))
	}
	stored, _ := store.Get(context.Background(), ask.ID)
	if stored.EventID != EventID(pub.events[1]) || !stored.Published {
		t.Fatalf("stored generation must bind the latest revision: %#v", stored)
	}
}

func TestRaiserWithdrawClosesOutstandingAsk(t *testing.T) {
	clock := &fakeClock{t: time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)}
	store := newMemStore()
	pub := &capturePublisher{}
	r := Raiser{Source: "data-backup-manager", Kind: "backup_failure", Store: store, Publisher: pub, Clock: clock}
	ctx := context.Background()

	if changed, err := r.Withdraw(ctx, "missing", "backup-failure:dest-1", "recovered"); changed || err != nil {
		t.Fatalf("unknown ask must be a no-op: %v %v", changed, err)
	}
	keyless := subjectFacts("Retry?")
	keyless.SubjectKey = ""
	if _, err := r.Raise(ctx, PendingAsk{ID: "keyless", Subject: "dest-2"}, keyless); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Raise(ctx, PendingAsk{ID: "keyed", Subject: "dest-1"}, subjectFacts("Retry?")); err != nil {
		t.Fatal(err)
	}

	changed, err := r.Withdraw(ctx, "keyed", "backup-failure:dest-1", "Backups to the drive succeeded again.")
	if err != nil || !changed {
		t.Fatalf("withdraw keyed: %v %v", changed, err)
	}
	last := pub.events[len(pub.events)-1]
	if last.EventType != "data-backup-manager.backup_failure.decision_withdrawn.v1" || last.Payload["subject_key"] != "backup-failure:dest-1" || last.Payload["reason"] != "Backups to the drive succeeded again." {
		t.Fatalf("keyed withdrawal event wrong: %#v", last)
	}
	if stored, _ := store.Get(ctx, "keyed"); stored.State != PendingStateWithdrawn || stored.ResolvedAt == nil || stored.Outstanding(clock.Now()) {
		t.Fatalf("withdrawn ask must be terminal: %#v", stored)
	}
	// A repeat is a no-op and publishes nothing.
	before := len(pub.events)
	if changed, err := r.Withdraw(ctx, "keyed", "backup-failure:dest-1", "again"); changed || err != nil || len(pub.events) != before {
		t.Fatalf("repeat withdraw must be a no-op")
	}

	// Without a subject key, the withdrawal names the request event id.
	stored, _ := store.Get(ctx, "keyless")
	if _, err := r.Withdraw(ctx, "keyless", "", "Destination removed."); err != nil {
		t.Fatal(err)
	}
	last = pub.events[len(pub.events)-1]
	if last.Payload["requester_ref"] != stored.EventID {
		t.Fatalf("keyless withdrawal must carry the request event id %q: %#v", stored.EventID, last.Payload)
	}

	// A withdrawn ask may be raised again when the situation returns.
	clock.t = clock.t.Add(time.Hour)
	if raised, err := r.Raise(ctx, PendingAsk{ID: "keyed", Subject: "dest-1"}, subjectFacts("Retry?")); err != nil || !raised {
		t.Fatalf("re-raise after withdraw: %v %v", raised, err)
	}
	if got := pub.events[len(pub.events)-1].Payload["asserted_at"]; got != "2026-11-13T13:00:00Z" {
		t.Fatalf("re-raise must assert after the withdrawal, got %v", got)
	}
}
