package deletiongrant

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
	"github.com/vrooli/api-core/schedule"
)

const (
	requestedType = "storage-manager.deletion-grant.decision_requested.v1"
	withdrawnType = "storage-manager.deletion-grant.decision_withdrawn.v1"
)

// recorder is the Vrooli Events the Raiser publishes to; failing makes publishing fail.
type recorder struct {
	events  []eventbus.DomainEvent
	failing bool
}

func (r *recorder) PublishDomainEvent(_ context.Context, e eventbus.DomainEvent) error {
	if r.failing {
		return errors.New("vrooli-events is down")
	}
	r.events = append(r.events, e)
	return nil
}

func (r *recorder) ofType(eventType string) []eventbus.DomainEvent {
	var out []eventbus.DomainEvent
	for _, e := range r.events {
		if e.EventType == eventType {
			out = append(out, e)
		}
	}
	return out
}

type rig struct {
	store  *Store
	events *recorder
	clock  *schedule.Fake
	raiser operatordecision.Raiser
	keyed  KeyedRaiser
	paths  func(string) PathState
}

func newRig(t *testing.T) *rig {
	t.Helper()
	store := newStore(t)
	clock := schedule.NewFake(time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC))
	events := &recorder{}
	raiser := operatordecision.Raiser{Source: Source, Kind: Kind, Store: store, Publisher: events, Clock: clock, MaxOutstanding: 10, Window: 7 * 24 * time.Hour}
	return &rig{store: store, events: events, clock: clock, raiser: raiser, keyed: KeyedRaiser{Inner: raiser, Store: store, Now: clock.Now}, paths: func(string) PathState { return PathPresent }}
}

func (r *rig) request(t *testing.T, paths ...string) string {
	t.Helper()
	id, raised, err := Request(context.Background(), r.keyed, nil, RequestInput{Paths: paths, RequestedBy: "run-1", RaisedByRunID: "run-1", Reason: "cleanup", TTL: time.Hour})
	if err != nil || !raised {
		t.Fatalf("raised=%v err=%v", raised, err)
	}
	return id
}

func (r *rig) reconcile(t *testing.T) Result {
	t.Helper()
	res, err := Reconciler{Store: r.store, Withdrawn: r.raiser, Now: r.clock.Now, PathState: r.paths}.Reconcile(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func (r *rig) ask(t *testing.T, id string) *operatordecision.PendingAsk {
	t.Helper()
	ask, err := r.store.Get(context.Background(), id)
	if err != nil || ask == nil {
		t.Fatalf("ask %s: %v %v", id, ask, err)
	}
	return ask
}

// A deletion request is one decision per path set: the request carries its id as
// the Hub subject key, so an agent asking again updates the open decision.
func TestRequestIsKeyedByItsPathSet(t *testing.T) {
	r := newRig(t)
	id := r.request(t, "/home/u/.cache/x")
	sent := r.events.ofType(requestedType)
	if len(sent) != 1 || sent[0].Payload["subject_key"] != id {
		t.Fatalf("the request is keyed by its id %q: %+v", id, sent)
	}
	if r.ask(t, id).Extra[extraSubjectKey] != id {
		t.Fatal("the ask records the key it was raised with, so a withdrawal can use it")
	}

	// The same agent asks again with another reason: the open decision updates in
	// place, so the Hub sees the same subject, and nothing else is opened.
	if _, _, err := Request(context.Background(), r.keyed, nil, RequestInput{Paths: []string{"/home/u/.cache/x"}, RequestedBy: "run-1", RaisedByRunID: "run-1", Reason: "still needed for the cleanup", TTL: time.Hour}); err != nil {
		t.Fatal(err)
	}
	for _, e := range r.events.ofType(requestedType) {
		if e.Payload["subject_key"] != id {
			t.Fatalf("every send for the path set names one subject: %+v", e.Payload)
		}
	}
}

// An ask nobody answered within its window is closed with a plain reason, by its
// subject key, and the agent can ask again.
func TestALapsedAskIsWithdrawnAndCanBeAskedAgain(t *testing.T) {
	r := newRig(t)
	id := r.request(t, "/home/u/.cache/x")
	r.clock.Advance(6 * 24 * time.Hour)
	if res := r.reconcile(t); res.Lapsed != 0 || res.Gone != 0 {
		t.Fatalf("an ask inside its window is left alone: %+v", res)
	}
	r.clock.Advance(25 * time.Hour)
	if res := r.reconcile(t); res.Lapsed != 1 {
		t.Fatalf("the lapsed ask is withdrawn: %+v", res)
	}
	sent := r.events.ofType(withdrawnType)
	if len(sent) != 1 || sent[0].Payload["subject_key"] != id || !strings.Contains(sent[0].Payload["reason"].(string), "Nobody answered within 168h0m0s") {
		t.Fatalf("withdrawal published with a plain reason: %+v", sent)
	}
	if got := r.ask(t, id); got.State != operatordecision.PendingStateWithdrawn {
		t.Fatalf("the ask is recorded as withdrawn: %s", got.State)
	}
	if rules, _ := r.store.ActiveGrantRoots(context.Background(), r.clock.Now()); len(rules) != 0 {
		t.Fatalf("a withdrawn ask grants nothing: %+v", rules)
	}
	r.reconcile(t)
	if len(r.events.ofType(withdrawnType)) != 1 {
		t.Fatal("a withdrawn ask is withdrawn once")
	}

	r.request(t, "/home/u/.cache/x")
	if r.ask(t, id).State != operatordecision.PendingStatePending {
		t.Fatal("asking again after a withdrawal opens a new decision")
	}
}

// An ask whose paths are all gone no longer needs the operator; one path still
// there, or one that cannot be told from an unplugged drive, keeps it open.
func TestAnAskWhosePathsAreGoneIsWithdrawn(t *testing.T) {
	dir := t.TempDir()
	gone, kept := filepath.Join(dir, "gone"), filepath.Join(dir, "kept")
	if err := os.WriteFile(kept, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	r := newRig(t)
	r.paths = StatPath
	allGone := r.request(t, gone, filepath.Join(dir, "gone2"))
	partly := r.request(t, gone, kept)
	unplugged := r.request(t, filepath.Join(dir, "no-such-drive", "x"))

	res := r.reconcile(t)
	if res.Gone != 1 {
		t.Fatalf("only the ask whose paths are all gone is withdrawn: %+v", res)
	}
	if r.ask(t, allGone).State != operatordecision.PendingStateWithdrawn || r.ask(t, partly).State != operatordecision.PendingStatePending || r.ask(t, unplugged).State != operatordecision.PendingStatePending {
		t.Fatal("a path still there, or whose parent is missing, keeps its ask open")
	}
	sent := r.events.ofType(withdrawnType)
	if len(sent) != 1 || sent[0].Payload["subject_key"] != allGone || !strings.Contains(sent[0].Payload["reason"].(string), "already gone") {
		t.Fatalf("withdrawn with a plain reason: %+v", sent)
	}
}

// An ask raised before subject keys has no subject on the Hub: it is closed by
// the request event id the row holds, and a repeat request does not open a second.
func TestAnAskRaisedBeforeSubjectKeysIsClosedByItsRequestEventID(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	id := r.request(t, "/home/u/.cache/x")
	// Make the row what the earlier code wrote: the unkeyed event's id, no key.
	legacy := r.ask(t, id)
	delete(legacy.Extra, extraSubjectKey)
	legacy.EventID = "event-id-before-subject-keys"
	if err := r.store.Put(ctx, legacy); err != nil {
		t.Fatal(err)
	}
	before := len(r.events.events)

	if _, _, err := Request(ctx, r.keyed, nil, RequestInput{Paths: []string{"/home/u/.cache/x"}, RequestedBy: "run-1", RaisedByRunID: "run-1", Reason: "still needed", TTL: time.Hour}); err != nil {
		t.Fatal(err)
	}
	for _, e := range r.events.events[before:] {
		if _, keyed := e.Payload["subject_key"]; keyed {
			t.Fatalf("a repeat of an unkeyed ask stays unkeyed, or the Hub would hold two decisions: %+v", e.Payload)
		}
	}
	legacy = r.ask(t, id)

	r.clock.Advance(8 * 24 * time.Hour)
	r.reconcile(t)
	sent := r.events.ofType(withdrawnType)
	if len(sent) != 1 || sent[0].Payload["requester_ref"] != legacy.EventID {
		t.Fatalf("the legacy ask is closed by its request event id %q: %+v", legacy.EventID, sent)
	}
	if _, keyed := sent[0].Payload["subject_key"]; keyed {
		t.Fatalf("no subject key to name: %+v", sent[0].Payload)
	}
}

// A withdrawal that could not be published is recorded and sent again, once.
func TestAWithdrawalThatCouldNotBePublishedIsSentAgain(t *testing.T) {
	r := newRig(t)
	id := r.request(t, "/home/u/.cache/x")
	r.clock.Advance(8 * 24 * time.Hour)
	r.events.failing = true
	if _, err := (Reconciler{Store: r.store, Withdrawn: r.raiser, Now: r.clock.Now, PathState: r.paths}).Reconcile(context.Background()); err == nil {
		t.Fatal("the failed publish is reported")
	}
	if r.ask(t, id).State != operatordecision.PendingStateWithdrawn {
		t.Fatal("the ask is withdrawn locally even though the Hub was not told")
	}
	r.events.failing = false
	if res := r.reconcile(t); res.Republished != 1 {
		t.Fatalf("the next pass sends it: %+v", res)
	}
	sent := r.events.ofType(withdrawnType)
	if len(sent) != 1 || sent[0].Payload["subject_key"] != id {
		t.Fatalf("one withdrawal reached the Hub: %+v", sent)
	}
	if r.reconcile(t); len(r.events.ofType(withdrawnType)) != 1 {
		t.Fatal("it is not sent a third time")
	}
}

// A withdrawn resolution is no answer: it records no grant, and a late answer to
// a withdrawn ask is ignored. A superseded one leaves the ask for its successor.
func TestAWithdrawnResolutionIsNeverApproval(t *testing.T) {
	ctx := context.Background()
	r := newRig(t)
	resolver := NewResolver(r.store).WithClock(r.clock.Now)
	resolve := func(id string, res operatordecision.DecisionResolution) {
		t.Helper()
		res.Correlation = map[string]string{correlationAskID: id}
		if handled, err := resolver.Apply(ctx, res); err != nil || !handled {
			t.Fatalf("handled=%v err=%v", handled, err)
		}
	}

	superseded := r.request(t, "/home/u/a")
	resolve(superseded, operatordecision.DecisionResolution{State: operatordecision.StateWithdrawn, SupersededByAskID: "newer"})
	if r.ask(t, superseded).State != operatordecision.PendingStatePending {
		t.Fatal("a superseded decision leaves the ask pending for its successor")
	}

	closed := r.request(t, "/home/u/b")
	resolve(closed, operatordecision.DecisionResolution{State: operatordecision.StateWithdrawn})
	if got := r.ask(t, closed); got.State != operatordecision.PendingStateWithdrawn || got.Extra[extraAnswer] != "" {
		t.Fatalf("a withdrawn decision is recorded as withdrawn with no answer: %+v", got)
	}
	resolve(closed, operatordecision.DecisionResolution{State: operatordecision.StateAnswered, Answer: "approve", AnsweredBy: "operator"})
	if rules, _ := r.store.ActiveGrantRoots(ctx, r.clock.Now()); len(rules) != 0 {
		t.Fatalf("an answer after a withdrawal grants nothing: %+v", rules)
	}
}
