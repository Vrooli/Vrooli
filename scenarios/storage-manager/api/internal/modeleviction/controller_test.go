package modeleviction

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/hubsender"
	"github.com/vrooli/api-core/operatordecision"

	"storage-manager/internal/providers"
)

var tickNow = time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)

type scriptedDecider struct {
	decision providers.ModelEvictionDecision
	policy   providers.ModelRetentionPolicy
	applied  [][]providers.ModelCandidate
}

func (s *scriptedDecider) Decide(context.Context) providers.ModelEvictionDecision { return s.decision }
func (s *scriptedDecider) Policy() providers.ModelRetentionPolicy            { return s.policy }
func (s *scriptedDecider) ApplyApproved(_ context.Context, approved []providers.ModelCandidate, _, _ string) []providers.ModelEvictionOutcome {
	s.applied = append(s.applied, approved)
	var out []providers.ModelEvictionOutcome
	for _, c := range approved {
		out = append(out, providers.ModelEvictionOutcome{Item: providers.ModelItemID(c), Model: c.Name, Evicted: true, FreedBytes: c.UniqueBytes, Reacquire: "resource-ollaya models pull --model " + c.Name})
	}
	return out
}

type recordedNotices struct{ sent []hubsender.Notice }

func (r *recordedNotices) Send(_ context.Context, n hubsender.Notice) error {
	r.sent = append(r.sent, n)
	return nil
}

type recordedEvents struct{ events []eventbus.DomainEvent }

func (r *recordedEvents) PublishDomainEvent(_ context.Context, e eventbus.DomainEvent) error {
	r.events = append(r.events, e)
	return nil
}

func (r *recordedEvents) types() []string {
	var out []string
	for _, e := range r.events {
		out = append(out, e.EventType)
	}
	return out
}

const gb = int64(1) << 30

func overBudgetDecision(selected ...string) providers.ModelEvictionDecision {
	d := providers.ModelEvictionDecision{
		ObservedAt: tickNow, Band: providers.ModelBandOK,
		Filesystems: []providers.ModelFilesystemSummary{{FilesystemSpace: providers.FilesystemSpace{ID: "dev:1", Path: "/models", FreeBytes: 400 * gb, TotalBytes: 1000 * gb}, FreePercent: 40, Band: providers.ModelBandOK}},
		Stores:      []providers.ModelStoreSummary{{Resource: "ollaya", StoreBytes: 46 * gb, BudgetBytes: 20 * gb, OverBudgetBytes: 26 * gb}},
	}
	for _, name := range selected {
		c := providers.ModelCandidate{Resource: "ollaya", Name: name, Digest: "sha256:" + name, Bytes: 9 * gb, UniqueBytes: 9 * gb, Eligible: true, ReacquireExact: true, SelectedFor: "budget", IdleFor: 20 * 24 * time.Hour, Reacquire: []string{"resource-ollaya", "models", "pull", "--model", name}}
		d.Candidates = append(d.Candidates, c)
		d.Selected = append(d.Selected, c)
		d.SelectedBytes += c.UniqueBytes
	}
	return d
}

func newController(t *testing.T, d providers.ModelEvictionDecision) (*Controller, *scriptedDecider, *recordedNotices, *recordedEvents) {
	t.Helper()
	decider := &scriptedDecider{decision: d, policy: providers.ModelRetentionPolicy{CheckInterval: 15 * time.Minute, Approval: providers.ModelApprovalOperator}}
	notices, events := &recordedNotices{}, &recordedEvents{}
	store := NewFileStore(filepath.Join(t.TempDir(), "model-eviction.json"))
	clock := func() time.Time { return tickNow }
	return &Controller{
		Decider: decider, Store: store, Notices: notices, Now: clock,
		Raiser: operatordecision.Raiser{Source: Source, Kind: Kind, Store: store, Publisher: events, Window: AskWindow, Clock: fixedNow{tickNow}},
	}, decider, notices, events
}

type fixedNow struct{ t time.Time }

func (f fixedNow) Now() time.Time { return f.t }

func TestOverBudgetStoreWithNothingEligibleNotifiesOnceAndRaisesNoDecision(t *testing.T) {
	c, _, notices, events := newController(t, overBudgetDecision())
	for i := 0; i < 3; i++ {
		if _, err := c.TickResult(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if len(notices.sent) != 1 {
		t.Fatalf("one notice per situation per day, got %d", len(notices.sent))
	}
	n := notices.sent[0]
	if n.Title != "A model store is over its disk budget" || !strings.Contains(n.Body, "ollaya models use 46.0 GiB, 26.0 GiB over their 20.0 GiB budget") || !strings.Contains(n.Body, "No model can be removed yet") || !strings.Contains(n.Body, "every 15 minutes") {
		t.Fatalf("notice = %+v", n)
	}
	if len(events.events) != 0 {
		t.Fatalf("nothing eligible means no decision, got %v", events.types())
	}
}

func TestSelectionBecomesOneHubDecisionThatTracksTheSelection(t *testing.T) {
	c, decider, notices, events := newController(t, overBudgetDecision("kev:4b", "coba:1b"))
	res, err := c.TickResult(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !res.Raised || len(events.events) != 1 || events.events[0].EventType != "storage-manager.model-eviction.decision_requested.v1" {
		t.Fatalf("raise = %+v events=%v", res, events.types())
	}
	payload := events.events[0].Payload
	if payload["recommended"] != "approve" || payload["subject_key"] != SubjectKey || !strings.Contains(payload["question"].(string), "kev:4b (ollaya): 9.0 GiB, unused for 20 days") {
		t.Fatalf("payload = %+v", payload)
	}
	if !strings.Contains(notices.sent[0].Body, "a decision with the list is waiting for you") {
		t.Fatalf("notice must point at the decision: %s", notices.sent[0].Body)
	}
	// Re-ticking the same selection publishes nothing new.
	if _, err := c.TickResult(context.Background()); err != nil || len(events.events) != 1 {
		t.Fatalf("idempotent re-tick published %v err=%v", events.types(), err)
	}
	// coba is used again: the selection changes, the old ask is superseded and a
	// new revision goes out under the same subject key.
	decider.decision = overBudgetDecision("kev:4b")
	res2, err := c.TickResult(context.Background())
	if err != nil || res2.AskID == res.AskID || len(events.events) != 2 {
		t.Fatalf("changed selection: res=%+v events=%v err=%v", res2, events.types(), err)
	}
	old, _ := c.Store.Get(context.Background(), res.AskID)
	if old.State != operatordecision.PendingStateWithdrawn {
		t.Fatalf("superseded ask state = %s", old.State)
	}
	// Space recovers: the open decision is withdrawn at the Hub.
	decider.decision = overBudgetDecision()
	decider.decision.Stores[0].OverBudgetBytes = 0
	res3, err := c.TickResult(context.Background())
	if err != nil || res3.Withdrawn != 1 || events.events[2].EventType != "storage-manager.model-eviction.decision_withdrawn.v1" {
		t.Fatalf("withdraw: res=%+v events=%v err=%v", res3, events.types(), err)
	}
}

func TestApprovedDecisionEvictsTheApprovedSelectionAndReportsIt(t *testing.T) {
	c, decider, notices, _ := newController(t, overBudgetDecision("kev:4b"))
	res, err := c.TickResult(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	r := &Resolver{Decider: decider, Store: c.Store, Notices: notices, Now: c.Now}
	if !r.Handles("storage-manager.model-eviction.decision_requested.v1") || r.Handles("storage-manager.deletion-grant.decision_requested.v1") {
		t.Fatal("resolver must own only model-eviction decisions")
	}
	resolution := operatordecision.DecisionResolution{State: "answered", Answer: "approve", AnsweredBy: "operator", Correlation: map[string]string{correlationAskID: res.AskID}}
	if handled, err := r.Apply(context.Background(), resolution); !handled || err != nil {
		t.Fatalf("apply handled=%v err=%v", handled, err)
	}
	if len(decider.applied) != 1 || decider.applied[0][0].Name != "kev:4b" || decider.applied[0][0].Digest != "sha256:kev:4b" {
		t.Fatalf("applied = %+v", decider.applied)
	}
	last := notices.sent[len(notices.sent)-1]
	if !strings.HasPrefix(last.Title, "Removed 1 of 1 idle AI models") || !strings.Contains(last.Body, "resource-ollaya models pull --model kev:4b") {
		t.Fatalf("outcome notice = %+v", last)
	}
	// A redelivered answer does nothing more.
	if _, err := r.Apply(context.Background(), resolution); err != nil || len(decider.applied) != 1 {
		t.Fatalf("redelivery re-applied: %d err=%v", len(decider.applied), err)
	}
	// The answered selection is not re-asked on the next tick.
	if res2, err := c.TickResult(context.Background()); err != nil || res2.Raised {
		t.Fatalf("answered selection re-raised: %+v err=%v", res2, err)
	}
}

func TestDeclinedDecisionEvictsNothingAndIsNotReasked(t *testing.T) {
	c, decider, _, events := newController(t, overBudgetDecision("kev:4b"))
	res, _ := c.TickResult(context.Background())
	r := &Resolver{Decider: decider, Store: c.Store, Now: c.Now}
	if _, err := r.Apply(context.Background(), operatordecision.DecisionResolution{State: "answered", Answer: "decline", Correlation: map[string]string{correlationAskID: res.AskID}}); err != nil {
		t.Fatal(err)
	}
	if len(decider.applied) != 0 {
		t.Fatal("a decline must not evict")
	}
	if _, err := c.TickResult(context.Background()); err != nil || len(events.events) != 1 {
		t.Fatalf("declined selection was re-asked: %v err=%v", events.types(), err)
	}
}

func TestLongSelectionsStayWithinHubLimits(t *testing.T) {
	var names []string
	for i := 0; i < 40; i++ {
		names = append(names, strings.Repeat("m", 30)+string(rune('a'+i%26))+":"+strings.Repeat("t", 10))
	}
	d := overBudgetDecision(names...)
	facts := DecisionFacts(d, selectionID(d), selectionReason(d))
	if err := facts.Validate(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(facts.BigPicture, names[39]) {
		t.Fatal("an overlong list moves to the big picture rather than being dropped")
	}
}
