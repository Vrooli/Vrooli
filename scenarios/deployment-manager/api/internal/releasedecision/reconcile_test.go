package releasedecision

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
)

const withdrawnType = "deployment-manager.release.decision_withdrawn.v1"

type movingClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *movingClock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *movingClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.mu.Unlock()
}

type eventLog struct {
	events  []eventbus.DomainEvent
	failing bool
}

func (l *eventLog) PublishDomainEvent(_ context.Context, e eventbus.DomainEvent) error {
	if l.failing {
		return errors.New("vrooli-events is down")
	}
	l.events = append(l.events, e)
	return nil
}

func (l *eventLog) withdrawals() []eventbus.DomainEvent {
	var out []eventbus.DomainEvent
	for _, e := range l.events {
		if e.EventType == withdrawnType {
			out = append(out, e)
		}
	}
	return out
}

type rig struct {
	store  *SQLStore
	events *eventLog
	clock  *movingClock
	raiser operatordecision.Raiser
	review map[string]ReviewVerdict
}

func newRig(t *testing.T) *rig {
	t.Helper()
	store := newStore(t)
	clock := &movingClock{now: time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)}
	events := &eventLog{}
	return &rig{store: store, events: events, clock: clock, review: map[string]ReviewVerdict{},
		raiser: operatordecision.Raiser{Source: "deployment-manager", Kind: "release", Store: store, Publisher: events, Clock: clock, MaxOutstanding: 10, Window: 24 * time.Hour}}
}

func (r *rig) reconciler() Reconciler {
	return Reconciler{Store: r.store, Withdrawn: r.raiser, Now: r.clock.Now, Review: func(_ context.Context, key string) (ReviewVerdict, error) {
		if v, ok := r.review[key]; ok {
			return v, nil
		}
		return ReviewOpen, nil
	}}
}

func (r *rig) reconcile(t *testing.T) Result {
	t.Helper()
	res, err := r.reconciler().Reconcile(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// raise raises a release decision the way the release handler does: keyed by
// profile, channel and targets, its own ask per commit.
func (r *rig) raise(t *testing.T, commit string, keyed bool) string {
	t.Helper()
	targets := []string{"linux-x64"}
	key := SubjectKey("p1", "stable", targets)
	id := operatordecision.CorrelationID("release", "p1", commit, "sha256:"+commit, "linux-x64")
	extra := map[string]string{"review_key": "rr-" + commit, "profile_id": "p1", "candidate_commit": commit, "channel": "stable", "targets": "linux-x64"}
	facts := operatordecision.Facts{
		Question: "Approve shipping p1 at " + commit + "?", Recommended: "reject",
		Options:     []operatordecision.Option{{Key: "approve", Label: "Approve"}, {Key: "reject", Label: "Reject"}},
		Correlation: map[string]string{"review_key": "rr-" + commit},
	}
	if keyed {
		extra[ExtraSubjectKey], facts.SubjectKey = key, key
	}
	if _, err := r.raiser.Raise(context.Background(), operatordecision.PendingAsk{ID: id, Subject: "rr-" + commit, Extra: extra}, facts); err != nil {
		t.Fatal(err)
	}
	return id
}

func (r *rig) ask(t *testing.T, id string) *operatordecision.PendingAsk {
	t.Helper()
	ask, err := r.store.Get(context.Background(), id)
	if err != nil || ask == nil {
		t.Fatalf("ask %s: %v %v", id, ask, err)
	}
	return ask
}

func TestSubjectKeyIsOnePerProfileChannelAndTargets(t *testing.T) {
	base := SubjectKey("p1", "stable", []string{"linux-x64", "darwin-arm64"})
	if SubjectKey("p1", "stable", []string{"darwin-arm64", "linux-x64"}) != base {
		t.Error("target order does not matter")
	}
	for name, other := range map[string]string{
		"profile": SubjectKey("p2", "stable", []string{"linux-x64", "darwin-arm64"}),
		"channel": SubjectKey("p1", "beta", []string{"linux-x64", "darwin-arm64"}),
		"targets": SubjectKey("p1", "stable", []string{"linux-x64"}),
	} {
		if other == base {
			t.Errorf("a different %s is another subject", name)
		}
	}
	for _, key := range []string{base, SubjectKey("a b/c#", "ch an", nil), SubjectKey(strings.Repeat("p", 200), "stable", nil)} {
		if !operatordecision.SubjectKeyRule.MatchString(key) {
			t.Errorf("%q breaks the Hub's key rule", key)
		}
	}
	if SubjectKey("a b", "x", nil) == SubjectKey("a_b", "x", nil) {
		t.Error("two different profiles never share a subject")
	}
}

// An ask nobody answered within its window is closed by its subject key with a plain reason.
func TestALapsedReleaseAskIsWithdrawnOnce(t *testing.T) {
	r := newRig(t)
	id := r.raise(t, "abc", true)
	r.clock.Advance(23 * time.Hour)
	if res := r.reconcile(t); res != (Result{}) {
		t.Fatalf("inside its window nothing happens: %+v", res)
	}
	r.clock.Advance(2 * time.Hour)
	if res := r.reconcile(t); res.Lapsed != 1 {
		t.Fatalf("the lapsed ask is withdrawn: %+v", res)
	}
	got := r.events.withdrawals()
	if len(got) != 1 || got[0].Payload["subject_key"] != SubjectKey("p1", "stable", []string{"linux-x64"}) || !strings.Contains(got[0].Payload["reason"].(string), "Nobody answered within 24h0m0s") {
		t.Fatalf("withdrawal with a plain reason: %+v", got)
	}
	if r.ask(t, id).State != operatordecision.PendingStateWithdrawn {
		t.Fatal("the ask is recorded as withdrawn")
	}
	r.reconcile(t)
	if len(r.events.withdrawals()) != 1 {
		t.Fatal("withdrawn once")
	}
}

// A review approved elsewhere, or superseded, closes the ask; an open or unknown one does not.
func TestAReleaseAskWhoseReviewIsDecidedElsewhereIsWithdrawn(t *testing.T) {
	r := newRig(t)
	approved, superseded, open, unknown := r.raise(t, "a1", true), r.raise(t, "a2", true), r.raise(t, "a3", true), r.raise(t, "a4", true)
	// Four commits of one release are one subject, so only the newest speaks; use
	// separate subjects by giving each its own channel.
	for id, channel := range map[string]string{approved: "c1", superseded: "c2", open: "c3", unknown: "c4"} {
		ask := r.ask(t, id)
		ask.Extra["channel"], ask.Extra[ExtraSubjectKey] = channel, "release:"+channel
		if err := r.store.Put(context.Background(), ask); err != nil {
			t.Fatal(err)
		}
	}
	r.review["rr-a1"], r.review["rr-a2"], r.review["rr-a3"] = ReviewApproved, ReviewSuperseded, ReviewOpen
	rec := r.reconciler()
	rec.Review = func(_ context.Context, key string) (ReviewVerdict, error) {
		if key == "rr-a4" {
			return ReviewUnknown, errors.New("review store unreadable")
		}
		return r.review[key], nil
	}
	res, err := rec.Reconcile(context.Background())
	if err == nil || !strings.Contains(err.Error(), "review store unreadable") {
		t.Fatalf("an unreadable review is reported, not guessed: %v", err)
	}
	if res.Cleared != 2 {
		t.Fatalf("approved and superseded reviews clear their asks: %+v", res)
	}
	if r.ask(t, approved).State != operatordecision.PendingStateWithdrawn || r.ask(t, superseded).State != operatordecision.PendingStateWithdrawn {
		t.Fatal("approved and superseded asks are withdrawn")
	}
	if r.ask(t, open).State != operatordecision.PendingStatePending || r.ask(t, unknown).State != operatordecision.PendingStatePending {
		t.Fatal("an open or unreadable review keeps its ask")
	}
	reasons := ""
	for _, e := range r.events.withdrawals() {
		reasons += e.Payload["reason"].(string) + "|"
	}
	if !strings.Contains(reasons, "already been approved") || !strings.Contains(reasons, "newer readiness review") {
		t.Fatalf("reasons: %s", reasons)
	}
}

// A newer release of the same profile, channel and targets replaces the older ask
// locally, and publishes nothing: the Hub already updated or replaced the
// decision, and a withdrawal naming the subject would close the newer one.
func TestANewerReleaseReplacesTheOlderAskWithoutClosingTheHubDecision(t *testing.T) {
	r := newRig(t)
	older := r.raise(t, "aaa", true)
	r.clock.Advance(time.Hour)
	newer := r.raise(t, "bbb", true)

	if res := r.reconcile(t); res.Replaced != 1 {
		t.Fatalf("the older ask is replaced: %+v", res)
	}
	if r.ask(t, older).State != operatordecision.PendingStateWithdrawn || r.ask(t, newer).State != operatordecision.PendingStatePending {
		t.Fatal("the older ask is withdrawn locally and the newer one stays")
	}
	if len(r.events.withdrawals()) != 0 {
		t.Fatalf("nothing is published for a keyed replacement: %+v", r.events.withdrawals())
	}
}

// While the Hub has not been told about the newer ask, the older decision is still
// the one the operator can answer, so it is kept.
func TestTheOlderAskIsKeptUntilTheNewerOneReachedTheHub(t *testing.T) {
	r := newRig(t)
	older := r.raise(t, "aaa", true)
	r.clock.Advance(time.Hour)
	r.events.failing = true
	id := operatordecision.CorrelationID("release", "p1", "bbb", "sha256:bbb", "linux-x64")
	_, _ = r.raiser.Raise(context.Background(), operatordecision.PendingAsk{ID: id, Subject: "rr-bbb", Extra: map[string]string{"profile_id": "p1", "channel": "stable", "targets": "linux-x64", ExtraSubjectKey: SubjectKey("p1", "stable", []string{"linux-x64"})}},
		operatordecision.Facts{Question: "Approve bbb?", Recommended: "reject", SubjectKey: SubjectKey("p1", "stable", []string{"linux-x64"}), Options: []operatordecision.Option{{Key: "approve"}, {Key: "reject"}}})
	r.events.failing = false
	if res := r.reconcile(t); res.Replaced != 0 {
		t.Fatalf("not replaced while the newer ask is unpublished: %+v", res)
	}
	if r.ask(t, older).State != operatordecision.PendingStatePending {
		t.Fatal("the older ask is still answerable")
	}
}

// An ask raised before subject keys is its own decision at the Hub: a newer release
// closes it by the request event id its row holds.
func TestAReleaseAskRaisedBeforeSubjectKeysIsClosedByItsRequestEventID(t *testing.T) {
	r := newRig(t)
	legacy := r.raise(t, "aaa", false)
	r.clock.Advance(time.Hour)
	r.raise(t, "bbb", true)
	stored := r.ask(t, legacy)

	if res := r.reconcile(t); res.Replaced != 1 {
		t.Fatalf("the legacy ask is replaced: %+v", res)
	}
	got := r.events.withdrawals()
	if len(got) != 1 || got[0].Payload["requester_ref"] != stored.EventID || stored.EventID == "" {
		t.Fatalf("closed by its request event id %q: %+v", stored.EventID, got)
	}
	if _, keyed := got[0].Payload["subject_key"]; keyed {
		t.Fatalf("no subject key to name: %+v", got[0].Payload)
	}
	if r.ask(t, legacy).Extra[ExtraWithdrawalUnpublished] != "" {
		t.Fatal("the published withdrawal is not marked for retry")
	}
}

// A repeat request for an ask raised before subject keys stays unkeyed while it is
// outstanding, so the Hub never holds two decisions for it.
func TestKeyedRaiserKeepsAnOutstandingLegacyAskUnkeyed(t *testing.T) {
	r := newRig(t)
	legacy := r.raise(t, "aaa", false)
	before := len(r.events.events)
	keyed := KeyedRaiser{Inner: r.raiser, Store: r.store, Now: r.clock.Now}
	key := SubjectKey("p1", "stable", []string{"linux-x64"})
	repeat := operatordecision.Facts{Question: "Approve shipping p1 at aaa (changed)?", Recommended: "reject", SubjectKey: key,
		Options: []operatordecision.Option{{Key: "approve", Label: "Approve"}, {Key: "reject", Label: "Reject"}}}
	if _, err := keyed.Raise(context.Background(), operatordecision.PendingAsk{ID: legacy, Subject: "rr-aaa", Extra: map[string]string{ExtraSubjectKey: key}}, repeat); err != nil {
		t.Fatal(err)
	}
	for _, e := range r.events.events[before:] {
		if _, has := e.Payload["subject_key"]; has {
			t.Fatalf("a repeat of an unkeyed ask is unkeyed: %+v", e.Payload)
		}
	}
	// After it lapsed, a new request is keyed.
	r.clock.Advance(25 * time.Hour)
	if _, err := keyed.Raise(context.Background(), operatordecision.PendingAsk{ID: legacy, Subject: "rr-aaa", Extra: map[string]string{ExtraSubjectKey: key}}, repeat); err != nil {
		t.Fatal(err)
	}
	if last := r.events.events[len(r.events.events)-1]; last.Payload["subject_key"] != key {
		t.Fatalf("a request after the ask lapsed is keyed: %+v", last.Payload)
	}
}

// A withdrawal that could not be published is recorded and sent again, once.
func TestAReleaseWithdrawalThatCouldNotBePublishedIsSentAgain(t *testing.T) {
	r := newRig(t)
	id := r.raise(t, "abc", true)
	r.clock.Advance(25 * time.Hour)
	r.events.failing = true
	if _, err := r.reconciler().Reconcile(context.Background()); err == nil {
		t.Fatal("the failed publish is reported")
	}
	if r.ask(t, id).State != operatordecision.PendingStateWithdrawn {
		t.Fatal("withdrawn locally even though the Hub was not told")
	}
	r.events.failing = false
	if res := r.reconcile(t); res.Republished != 1 {
		t.Fatalf("the next pass sends it: %+v", res)
	}
	if got := r.events.withdrawals(); len(got) != 1 || got[0].Payload["subject_key"] != SubjectKey("p1", "stable", []string{"linux-x64"}) {
		t.Fatalf("one withdrawal reached the Hub: %+v", got)
	}
	if r.reconcile(t); len(r.events.withdrawals()) != 1 {
		t.Fatal("not sent a third time")
	}
}
