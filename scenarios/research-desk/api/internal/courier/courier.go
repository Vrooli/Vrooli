// Package courier carries needs-you items to Notification Hub and answers
// back into OPERATOR.md (SYSTEM.md §7). One pass every minute in the instance
// holding the write lock; no watcher, no pending store: every pass re-reads
// the files and the Hub. Decisions and notices go out as events through Vrooli
// Events; a decision carries its page inline (pages.go). An unchanged item is the
// same request, so a repeated send is harmless. An item's ask carries a Hub
// subject key, so the Hub keeps one open decision per item: the courier updates
// it when the item changes and withdraws it when the item is answered in the
// desk or leaves the files.
package courier

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"maps"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/operatordecision"
	"github.com/vrooli/api-core/schedule"

	"research-desk/internal/home"
	"research-desk/internal/writer"
)

// Key prefixes. An item's key is its identity: commission + item ID.
const (
	AskPrefix    = "research-desk/"
	NoticePrefix = "research-desk-notice/"
	TestPrefix   = "research-desk-test/"
)

// SwitchFile holds the operator's courier switches in the desk state folder.
const SwitchFile = "courier.json"

// SightingsFile records when the courier first found each needs-you item
// (FeedItem.Key) in the files: a map from key to time. Three rules read it:
//   - an item first found at or before its switch was turned on was there
//     already, and is never sent;
//   - an item whose scheduler delivers by itself waits SchedulerGrace from
//     its first sighting;
//   - an answer or delivery older than the first sighting belongs to an
//     earlier item that used the same ID.
//
// An item already there when the record began has the zero time: its first
// appearance is unknown, so it counts as there before any switch, its grace
// is over and it is never called reused. An item that leaves is forgotten.
// Deleting the file starts a new record.
const SightingsFile = "first-sightings.json"

// SchedulerGrace is how long the courier leaves an item to a scheduler that
// delivers by itself: a commission whose charter lacks the Decisions sentence.
// Such a scheduler adds `Delivered:` when it delivers; the courier sends only
// what is still undelivered after the grace.
const SchedulerGrace = 2 * time.Hour

// Switch is one courier switch. Turning it on records a first sighting of
// every item present at that moment, so none of them is ever sent: turning
// the courier on never notifies the operator about old items.
type Switch struct {
	On bool      `json:"on"`
	At time.Time `json:"at"`
}

// Sightings are the first sightings of the items in the files.
type Sightings struct {
	First map[string]time.Time
	// found is false before the record began; the first record marks every
	// item present as already there.
	found bool
}

// at is when an item was first sighted; ok is false before its first sighting.
func (s Sightings) at(f *home.FeedItem) (time.Time, bool) {
	t, ok := s.First[f.Key()]
	return t, ok
}

// before reports whether an item was sighted at or before t.
func (s Sightings) before(f *home.FeedItem, t time.Time) bool {
	first, ok := s.at(f)
	return ok && !first.After(t)
}

// graceEnd is when the courier stops waiting for an item's own scheduler;
// an item not sighted yet is sighted now.
func (s Sightings) graceEnd(f *home.FeedItem, now time.Time) time.Time {
	first, ok := s.at(f)
	if !ok {
		first = now
	}
	return first.Add(SchedulerGrace)
}

// Switches are the operator's courier choices. A commission without an entry
// is on only when the desk launched it (every item it has appeared after the
// launch). A level's decisions are on: the courier is their route. The global
// switch lets the operator turn the supervisor's (ATTENTION.md) off.
type Switches struct {
	Global      Switch            `json:"global"`
	Commissions map[string]Switch `json:"commissions"`
}

// For says whether the courier carries an item, and why; present is true
// when the item was there before its switch was turned on.
func (s Switches) For(f *home.FeedItem, sightings Sightings) (on, present bool, why string) {
	switch {
	case f.CommissionID == "":
		// A level's decision. Its record of first sightings began before the
		// item (or the switch) did, else the item was already there: never sent.
		on, why = true, "a topic's decision: the courier is its route"
		var since time.Time
		if f.Topic == "" {
			on, why = s.Supervisor()
			since = s.Global.At
		}
		if on && (!sightings.found || sightings.before(f, since)) {
			if since.IsZero() {
				return false, true, "this item was already there when the courier began to carry a level's own decisions; it is never sent"
			}
			return false, true, "this item was already there when the courier was turned on at " + stamp(since) + "; it is never sent"
		}
		return on, false, why
	case f.Commission == nil:
		return false, false, "it names a commission with no folder"
	}
	on, why = s.ForCommission(f.Commission)
	if sw, ok := s.Commissions[f.Commission.ID]; on && ok && sightings.before(f, sw.At) {
		return false, true, "this item was already there when the courier was turned on at " + stamp(sw.At) + "; it is never sent"
	}
	return on, false, why
}

// Supervisor says whether the courier carries the manager's decisions
// (ATTENTION.md items without a `Commission:` line), and why: on unless the
// operator turned the global switch off.
func (s Switches) Supervisor() (bool, string) {
	switch {
	case s.Global.At.IsZero():
		return true, "the manager's decisions: on (courier off --global turns them off)"
	case s.Global.On:
		return true, "the manager's decisions: turned on by the operator at " + stamp(s.Global.At)
	}
	return false, "the manager's decisions: turned off by the operator at " + stamp(s.Global.At)
}

// ForCommission says whether the courier is on for a commission, and why.
func (s Switches) ForCommission(c *home.Commission) (bool, string) {
	if sw, ok := s.Commissions[c.ID]; ok {
		if sw.On {
			return true, "turned on by the operator at " + stamp(sw.At)
		}
		return false, "turned off by the operator"
	}
	if c.LaunchedBy() == "desk" {
		return true, "on: the desk launched this commission"
	}
	return false, "off: the desk did not launch this commission; the operator turns delivery on for it"
}

func stamp(t time.Time) string { return t.UTC().Format(time.RFC3339) }

// Action is what a pass does for one step.
type Action string

// Actions.
const (
	ActNone            Action = "none"
	ActAsk             Action = "ask"
	ActNotice          Action = "notice"
	ActRecordDelivered Action = "record-delivered"
	ActRecordAnswer    Action = "record-answer"
	// ActRefreshAsk raises an open ask again because its item changed, or
	// because it was raised before asks carried a subject key: the Hub updates
	// or replaces the one open decision for the item.
	ActRefreshAsk Action = "refresh-ask"
	// ActWithdraw closes an open ask the operator no longer needs to answer.
	ActWithdraw Action = "withdraw"
	// ActLandAnswer records the operator's answer to an ask whose item is in no
	// file in its level's inbox (afterlife.go).
	ActLandAnswer Action = "land-answer"
	// ActWritePage starts or moves on the decision-page writer's job for a
	// decision whose page does not exist; the ask goes once the page is
	// ready (pagewriter.go).
	ActWritePage Action = "write-page"
)

// Step is the courier's decision for one item or one orphaned ask.
type Step struct {
	// Item is nil for an open ask whose item is no longer in any file.
	Item     *home.FeedItem
	Key      string
	Delivery home.Delivery
	On       bool
	Switch   string
	Ask      *Ask
	Notice   *Notice
	Action   Action
	// Inboxes are the levels whose INBOX.md gets an answer this step records.
	Inboxes []*home.Level
	// Why explains the action, or why there is none; Reason is the same as
	// a code.
	Why    string
	Reason home.DeliveryReason
	// Reused is set when the item's ID already carries an earlier item's
	// answer, closed ask or delivery; those belong to the earlier item.
	Reused bool
	// Done and Err are set by a pass that carried the action out.
	Done bool
	Err  error
}

// Waiting is whether the item still waits on the operator: it has no answer of
// its own in OPERATOR.md and its Hub ask was not answered. An expired,
// dismissed or withdrawn ask leaves the item waiting; it is answered in the desk.
func (s Step) Waiting() bool {
	switch {
	case s.Item == nil:
		return false
	case s.Reused:
		// The answer and ask on record belong to an earlier item.
		return true
	case s.Item.Answer != nil:
		return false
	default:
		return s.Ask == nil || s.Ask.State != "answered"
	}
}

// Plan is one pass's decisions; a dry run returns it without acting.
type Plan struct {
	At       time.Time
	Switches Switches
	Steps    []Step
	// HubRead is nil when the Hub's asks were read; otherwise why not.
	HubRead error
	// Unreadable lists needs-you and answer files that exist but cannot be
	// read; nothing is sent, recorded or forgotten for their owners.
	Unreadable []home.Evidence
	// Recorded is the newest delivery the files record: a `delivered`
	// entry in any OPERATOR.md. It survives a restart, which the courier's
	// own memory of what it sent does not.
	Recorded  time.Time
	blocked   func(owner string) string
	sightings Sightings
	// pages are the items' explainer pages, read for this plan.
	pages Pages
	// misses are the page misses recorded so far (PageMissesFile).
	misses map[string]PageMiss
}

// keep reports whether a remembered key's owner is blocked, so its record
// is kept as it was rather than forgotten as gone.
func (p Plan) keep(key string) bool {
	owner, _, _ := strings.Cut(key, "/")
	return p.blocked != nil && p.blocked(owner) != ""
}

// Courier runs passes.
type Courier struct {
	Reader home.Snapshotter
	Writer *writer.Writer
	Hub    Hub
	// PageWriter makes the page of a decision whose raiser left it out; nil
	// leaves such a decision unsent.
	PageWriter PageWriter
	Clock      schedule.NowSource
	Logger     *log.Logger

	mu          sync.Mutex
	lastProblem string

	hmu    sync.Mutex
	health Health
}

// Health is the courier's real state, shown in the overview:
// whether its last pass could read and send, when it last read the Hub's
// answers back and when it last delivered.
type Health struct {
	// LastPass is when the last pass ended; zero before the first.
	LastPass time.Time
	// Problem is why the last pass could not read or send; empty when it could.
	Problem string
	// LastReadBack is when the Hub's asks were last read.
	LastReadBack time.Time
	// LastDelivery is the later of when this process last sent an ask or a
	// notice and the newest delivery recorded in the files.
	LastDelivery time.Time
	// PageMisses says which decisions are held unsent because the
	// decision-page writer could not make their page; empty when none are.
	PageMisses string
}

// Health returns the courier's state after its last pass.
func (c *Courier) Health() Health {
	c.hmu.Lock()
	defer c.hmu.Unlock()
	return c.health
}

// Withdraw is said wherever the operator sees the courier.
const Withdraw = "an ask closes when its item is answered in the desk or leaves the files"

// Say is the courier's health in the operator's words: whether it works,
// and the times behind that.
func (h Health) Say() (bool, string) {
	when := func(t time.Time) string {
		if t.IsZero() {
			return "never"
		}
		return stamp(t)
	}
	misses := ""
	if h.PageMisses != "" {
		misses = "; " + h.PageMisses
	}
	switch {
	case h.LastPass.IsZero():
		return false, "no pass yet; " + Withdraw
	case h.Problem != "":
		return false, fmt.Sprintf("last pass %s failed: %s; answers last read back %s; last delivery %s%s; %s", stamp(h.LastPass), h.Problem, when(h.LastReadBack), when(h.LastDelivery), misses, Withdraw)
	}
	return true, fmt.Sprintf("last pass %s; answers read back %s; last delivery %s%s; %s", stamp(h.LastPass), when(h.LastReadBack), when(h.LastDelivery), misses, Withdraw)
}

// problem is why a pass could not read or send everything, or "".
func problem(plan Plan, err error) string {
	switch {
	case err != nil:
		return err.Error()
	case plan.HubRead != nil:
		return "notification-hub unreadable: " + plan.HubRead.Error()
	case len(plan.Unreadable) > 0:
		return fmt.Sprintf("%d needs-you or answer files cannot be read, so nothing is sent or recorded for their items; first: %s", len(plan.Unreadable), plan.Unreadable[0].Reason)
	}
	var failed []string
	for _, s := range plan.Steps {
		if s.Err != nil {
			failed = append(failed, fmt.Sprintf("%s %s failed: %v", s.Action, s.Key, s.Err))
		}
	}
	for _, s := range plan.Steps {
		if s.Item == nil && s.Action == ActNone && s.Ask != nil && s.Ask.State == "answered" {
			failed = append(failed, fmt.Sprintf("the operator's answer to %s (ask %s) is not recorded yet: %s", s.Key, s.Ask.ID, s.Why))
		}
	}
	return strings.Join(failed, "; ")
}

// note records a finished pass in Health.
func (c *Courier) note(plan Plan, err error) {
	c.hmu.Lock()
	defer c.hmu.Unlock()
	h := &c.health
	h.LastPass, h.Problem = c.Clock.Now(), problem(plan, err)
	if plan.misses != nil {
		h.PageMisses = sayMisses(held(plan, h.LastPass))
	}
	if err == nil && plan.HubRead == nil {
		h.LastReadBack = plan.At
	}
	for _, s := range plan.Steps {
		if s.Done && (s.Action == ActAsk || s.Action == ActNotice) {
			h.LastDelivery = h.LastPass
		}
	}
	if plan.Recorded.After(h.LastDelivery) {
		h.LastDelivery = plan.Recorded
	}
}

// Switches reads the operator's switches; a missing file is all defaults.
func (c *Courier) Switches() (Switches, error) {
	var s Switches
	if _, err := c.Writer.ReadState(SwitchFile, &s); err != nil {
		return Switches{}, fmt.Errorf("%w; the courier sends nothing until it can read its switches", err)
	}
	if s.Commissions == nil {
		s.Commissions = map[string]Switch{}
	}
	return s, nil
}

// Set records one switch. An empty commission sets the global switch.
// Turning a switch on first records a sighting, at this moment, of every item
// not sighted yet, so the items present now are never sent.
func (c *Courier) Set(ctx context.Context, commission string, on bool) (Switches, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	s, err := c.Switches()
	if err != nil {
		return s, err
	}
	sw := Switch{On: on, At: c.Clock.Now().UTC()}
	if on {
		snap, err := c.Reader.Snapshot(ctx, home.Options{})
		if err != nil {
			return s, fmt.Errorf("read the items present now, which turning the courier on leaves unsent: %w", err)
		}
		feed := snap.NeedsYou()
		if why := feed.Blocked(commission); why != "" {
			return s, fmt.Errorf("%w: the switch is unchanged: the items it covers cannot be read, so those there now could not be marked as never sent (%s)", home.ErrRefused, why)
		}
		sightings, err := c.sightings()
		if err != nil {
			return s, err
		}
		next := maps.Clone(sightings.First)
		for i := range feed.Items {
			if _, ok := next[feed.Items[i].Key()]; !ok {
				next[feed.Items[i].Key()] = sightingTime(sightings, sw.At)
			}
		}
		if err := c.writeSightings(sightings, next); err != nil {
			return s, err
		}
	}
	if commission == "" {
		s.Global = sw
	} else {
		s.Commissions[commission] = sw
	}
	_, err = c.Writer.WriteState(SwitchFile, s)
	return s, err
}

// sightingTime is the first sighting recorded now: the zero time ("already
// there") in the first record, else now.
func sightingTime(s Sightings, now time.Time) time.Time {
	if !s.found {
		return time.Time{}
	}
	return now
}

// sightings reads the first sightings.
func (c *Courier) sightings() (Sightings, error) {
	s := Sightings{First: map[string]time.Time{}}
	found, err := c.Writer.ReadState(SightingsFile, &s.First)
	if err != nil {
		return s, fmt.Errorf("%w; the courier sends nothing until it can read when each item first appeared", err)
	}
	s.found = found
	return s, nil
}

// writeSightings writes next when it differs from what was read.
func (c *Courier) writeSightings(old Sightings, next map[string]time.Time) error {
	if old.found && maps.EqualFunc(old.First, next, time.Time.Equal) {
		return nil
	}
	_, err := c.Writer.WriteState(SightingsFile, next)
	return err
}

// Plan computes a pass without acting: the dry run.
func (c *Courier) Plan(ctx context.Context) (Plan, *home.Snapshot, error) {
	snap, err := c.Reader.Snapshot(ctx, home.Options{})
	if err != nil {
		return Plan{}, nil, err
	}
	sw, err := c.Switches()
	if err != nil {
		return Plan{}, nil, err
	}
	feed := snap.NeedsYou()
	plan := Plan{At: snap.Now, Switches: sw, Unreadable: feed.Unreadable, blocked: feed.Blocked, Recorded: recordedDelivery(snap)}
	var (
		asks    []Ask
		notices map[string]Notice
	)
	switch {
	case c.Hub == nil:
		plan.HubRead = errors.New("notification-hub is not configured for this instance")
	default:
		if plan.HubRead = c.Hub.Sendable(ctx); plan.HubRead != nil {
			break
		}
		plan.pages = readPages(feed, snap.ResearchDir, snap.Now)
		c.readJobs(&plan.pages)
		if asks, plan.HubRead = c.Hub.Asks(ctx, AskPrefix); plan.HubRead == nil {
			notices, plan.HubRead = c.Hub.Notices(ctx)
		}
	}
	if plan.sightings, err = c.sightings(); err != nil {
		return Plan{}, nil, err
	}
	if plan.misses, err = c.misses(); err != nil {
		return Plan{}, nil, fmt.Errorf("%w; the courier sends nothing until it can read which decisions went without a written page", err)
	}
	plan.Steps = decide(c.Hub, feed, sw, asks, notices, plan.HubRead, plan.sightings, plan.pages, ownersOf(snap), snap.Now)
	return plan, snap, nil
}

// recordedDelivery is the newest `delivered` entry in any commission's
// OPERATOR.md; zero when none is recorded.
func recordedDelivery(snap *home.Snapshot) time.Time {
	var newest time.Time
	for _, c := range snap.Commissions {
		if c.Operator == nil {
			continue
		}
		for _, e := range c.Operator.Entries {
			if e.Kind == home.EntryDelivered && e.At.After(newest) {
				newest = e.At
			}
		}
	}
	return newest
}

// selfDelivering reports whether the item's scheduler delivers it by itself:
// a commission whose charter predates the Decisions sentence.
func selfDelivering(f *home.FeedItem) bool {
	return f.Commission != nil && !f.Commission.Decisions.Value
}

// decide is the pure core of a pass.
func decide(hub Hub, feed home.Feed, sw Switches, asks []Ask, notices map[string]Notice, hubErr error, sightings Sightings, pages Pages, owners Owners, now time.Time) []Step {
	// An item may have several asks: an earlier one that ended, or an open one
	// raised before asks carried a subject key and a newer one that replaced
	// it. The open one, then the newest, speaks for the item; any other open
	// ask is a duplicate and is withdrawn.
	byKey := make(map[string]*Ask, len(asks))
	for i := range asks {
		a := &asks[i]
		if old := byKey[a.Key]; old == nil || (a.Open() && !old.Open()) || (a.Open() == old.Open() && a.CreatedAt.After(old.CreatedAt)) {
			byKey[a.Key] = a
		}
	}
	var duplicates []*Ask
	for i := range asks {
		if a := &asks[i]; a.Open() && byKey[a.Key] != a {
			duplicates = append(duplicates, a)
		}
	}
	listed := map[string]bool{}
	steps := make([]Step, 0, len(feed.Items))
	for i := range feed.Items {
		f := &feed.Items[i]
		s := Step{Item: f, Action: ActNone}
		s.Delivery, s.Reason, s.Why = f.Delivery()
		on, present, why := sw.For(f, sightings)
		s.On, s.Switch = on, why
		s.Key = owners.askKey(f, byKey)
		s.Ask = byKey[s.Key]
		if s.Ask != nil && s.Ask.State == "withdrawn" {
			// Only the desk withdraws an ask: its item left the files, or was answered in the desk, or a newer ask
			// replaced it. An item that is back and unanswered is a new question, never an "ID reused" (4 live asks
			// sat undelivered for that on 2026-11-20); the operator's own refusal is `dismissed`.
			s.Ask = nil
		}
		listed[s.Key] = true
		if n, ok := notices[NoticePrefix+f.Owner()+"/"+f.Item.ID]; ok {
			s.Notice = &n
		}
		answerable, _ := f.Answerable()
		first, _ := sightings.at(f)
		reusedWhy := reused(f, s.Ask, first)
		switch a := s.Ask; {
		case feed.Blocked(f.Owner()) != "":
			s.Reason, s.Why = home.ReasonUnreadable, "nothing is sent or recorded while the desk cannot read this item's files: "+feed.Blocked(f.Owner())
		case reusedWhy != "":
			s.Reused, s.Reason, s.Why = true, home.ReasonIDReused, reusedWhy
		case a != nil && a.State == "answered" && f.Answer == nil && answerable:
			s.Action, s.Reason, s.Why, s.Inboxes = ActRecordAnswer, home.ReasonHubAnswered, "answered in Notification Hub; recorded in "+answerRecord(f), f.Inboxes
		case a != nil && a.Open() && f.Answer != nil:
			s.Action, s.Reason, s.Why = ActWithdraw, home.ReasonAnswered, withdrawAnswered
		case a != nil && a.Open() && s.On && hubErr == nil && askStale(hub, a, f, pages, now):
			s.Action, s.Reason, s.Why = ActRefreshAsk, home.ReasonInHub, "the Hub ask is not the item's current question; raised again so the Hub updates or replaces it"
		case a != nil && a.Open() && f.Item.Delivered != "":
			s.Reason, s.Why = home.ReasonInHub, "delivered by its scheduler; the Hub ask stays open until it is answered or the item leaves the files"
		case a != nil && !a.FirstDelivered.IsZero() && f.Commission != nil && f.Delivered == nil && f.Answer == nil:
			s.Action, s.Reason, s.Why = ActRecordDelivered, home.ReasonHubDelivered, "the Hub delivered the ask"
		case a != nil:
			s.Reason, s.Why = home.ReasonInHub, "Hub ask "+a.State
			if !a.Open() && a.State != "answered" && f.Answer == nil {
				s.Why += "; answer it in the desk"
			}
		case s.Delivery == home.DeliverNone:
		case present:
			s.Reason, s.Why = home.ReasonBaseline, s.Switch
		case !s.On:
			s.Reason, s.Why = home.ReasonSwitchOff, s.Switch
		case selfDelivering(f) && now.Before(sightings.graceEnd(f, now)):
			s.Reason, s.Why = home.ReasonSchedulerGrace, "its scheduler delivers it by itself; sent only if it has no `Delivered:` line by "+stamp(sightings.graceEnd(f, now))
		case hubErr != nil:
			s.Reason, s.Why = home.ReasonHubUnavailable, "not sent: "+hubErr.Error()
		case s.Delivery == home.DeliverAsk:
			s.Action = ActAsk
		case f.Delivered != nil:
			s.Reason, s.Why = home.ReasonHubDelivered, "notice delivered "+f.Delivered.At.UTC().Format(time.RFC3339)
		default:
			s.Key = NoticePrefix + f.Owner() + "/" + f.Item.ID
			switch n := s.Notice; {
			case n == nil:
				s.Action = ActNotice
			case n.Delivered && f.Commission != nil:
				s.Action, s.Reason, s.Why = ActRecordDelivered, home.ReasonHubDelivered, "the Hub delivered the notice"
			case n.Delivered:
				s.Reason, s.Why = home.ReasonHubDelivered, "notice delivered"
			default:
				s.Reason, s.Why = home.ReasonInHub, "Hub notice "+n.State
			}
		}
		if unready := pages.unready[f.Key()]; unready != "" && (s.Action == ActAsk || s.Action == ActRefreshAsk) {
			// A thin packet cannot reach the operator: its executor would re-derive the decision (O7(f)).
			s.Action, s.Reason = ActNone, home.ReasonPacketNotReady
			s.Why = "not sent until the decision's folder is ready for a cold executor: " + unready + "; check it with needs_you.py validate"
		}
		if job := pages.job(f); job != nil && (s.Action == ActAsk || s.Action == ActRefreshAsk) {
			switch failure := job.missed(); {
			case job.page() != nil:
				s.Why = joinWhy(s.Why, "sent with the page the decision-page writer made, "+job.result.Page)
			case failure != nil:
				// No page restating the item ever reaches the operator.
				s.Action, s.Reason = ActNone, home.ReasonPacketNotReady
				s.Why = "not sent: the decision-page writer could not make its page (" + failure.Error() + "; job " + job.result.Dir +
					"); recorded as a page miss. Write work/" + f.Item.ID + "/page.html beside the brief, or change the brief, and the next pass sends it"
			default:
				// The operator answers in hours and a page takes minutes: the item waits for it.
				s.Action, s.Reason, s.Why = ActWritePage, home.ReasonPacketNotReady, job.waiting()
			}
		}
		if problem := pages.problem(f); problem != "" && (s.Action == ActAsk || s.Action == ActRefreshAsk) {
			s.Why = joinWhy(s.Why, problem)
		}
		var grammar []string
		for _, p := range f.Item.Problems {
			if p != f.Item.RecommendedProblem {
				grammar = append(grammar, p)
			}
		}
		if len(grammar) > 0 && s.Action != ActWithdraw && s.Reason != home.ReasonPacketNotReady {
			// A withdrawal's reason is the operator's to read; the grammar note is the agent's.
			s.Why = joinWhy(s.Why, "the item breaks the grammar ("+strings.Join(grammar, "; ")+"); check it with needs_you.py validate")
		}
		steps = append(steps, s)
	}
	for _, a := range byKey {
		if listed[a.Key] {
			continue
		}
		if a.State == "answered" && strings.HasPrefix(a.Key, AskPrefix) {
			if step, ok := afterAnswer(feed, owners, a); ok {
				steps = append(steps, step)
			}
			continue
		}
		if !a.Open() {
			continue
		}
		if why := feed.Blocked(commissionOf(a.Key)); why != "" {
			steps = append(steps, Step{
				Key: a.Key, Ask: a, Delivery: home.DeliverNone, Action: ActNone, Reason: home.ReasonOrphaned,
				Why: "its item's files cannot be read, so the Hub ask stays open until they can: " + why,
			})
			continue
		}
		steps = append(steps, Step{Key: a.Key, Ask: a, Delivery: home.DeliverNone, Action: ActWithdraw, Reason: home.ReasonOrphaned, Why: withdrawOrphaned})
	}
	for _, a := range duplicates {
		if feed.Blocked(commissionOf(a.Key)) != "" {
			continue
		}
		steps = append(steps, Step{Key: a.Key, Ask: a, Delivery: home.DeliverNone, Action: ActWithdraw, Reason: home.ReasonInHub, Why: withdrawDuplicate})
	}
	n := len(feed.Items)
	sort.SliceStable(steps[n:], func(i, j int) bool {
		if a, b := steps[n+i], steps[n+j]; a.Key != b.Key {
			return a.Key < b.Key
		}
		return steps[n+i].Ask.ID < steps[n+j].Ask.ID
	})
	return steps
}

// joinWhy adds a note to a step's sentence.
func joinWhy(why, note string) string {
	if why == "" {
		return note
	}
	return why + "; " + note
}

// reused says why an item's ID was reused, or "": an answer, an
// answered, dismissed or expired Hub ask or a delivered entry for the same commission and ID dates
// from before the item's first sighting, so it belongs to an earlier item. An
// answer given after the item appeared settles it; an item with no known
// first sighting (zero) is never called reused.
func reused(f *home.FeedItem, a *Ask, first time.Time) string {
	if first.IsZero() || (f.Answer != nil && !f.Answer.At.Before(first)) {
		return ""
	}
	var earlier string
	switch {
	case f.Answer != nil:
		earlier = "an answer in OPERATOR.md from " + stamp(f.Answer.At)
	case a != nil && !a.Open() && a.CreatedAt.Before(first):
		earlier = "a " + a.State + " Hub ask from " + stamp(a.CreatedAt)
	case f.Delivered != nil && f.Delivered.At.Before(first):
		earlier = "a delivered entry from " + stamp(f.Delivered.At)
	default:
		return ""
	}
	what := "answer it in the desk, or give the item a new ID"
	return fmt.Sprintf("ID reused, not delivered: %s already has %s, before this item first appeared at %s; %s", f.Item.ID, earlier, stamp(first), what)
}

// recordSightings keeps the first sighting of each item in the feed and
// forgets the rest, except the items of owners whose files cannot be read,
// which are kept as they were.
func (c *Courier) recordSightings(plan Plan) error {
	old := plan.sightings
	next := map[string]time.Time{}
	for key, first := range old.First {
		if plan.keep(key) {
			next[key] = first
		}
	}
	for _, s := range plan.Steps {
		if s.Item == nil || plan.keep(s.Item.Key()) {
			continue
		}
		key := s.Item.Key()
		if first, ok := old.First[key]; ok {
			next[key] = first
		} else {
			next[key] = sightingTime(old, plan.At)
		}
	}
	return c.writeSightings(old, next)
}

// Reasons shown to the operator on a closed Hub decision, in plain words.
const (
	withdrawAnswered  = "You answered this in the research desk."
	withdrawOrphaned  = "This item is no longer in the research desk's files, so it no longer needs you."
	withdrawDuplicate = "Replaced by an updated copy of this decision."
)

// commissionOf is the owner an ask key names: a commission, a topic
// (home.TopicOwner) or the supervisor.
func commissionOf(key string) string {
	commission, _, _ := strings.Cut(strings.TrimPrefix(key, AskPrefix), "/")
	return commission
}

// askStale reports whether an open ask is no longer the question the courier
// would raise for the item: its text or answer menu changed. An ask raised
// before asks carried a subject key, and still saying what the item says, is
// left as it is (raising it again would page the operator twice for nothing);
// it is withdrawn by its request id when the item is answered or leaves.
func askStale(hub Hub, a *Ask, f *home.FeedItem, pages Pages, now time.Time) bool {
	spec := askSpec(f, a.Key, pages, now)
	ref := hub.RequestRef(spec)
	return ref != "" && a.RequesterRef != ref && !a.says(spec)
}

// says reports whether the Hub holds the decision spec describes.
func (a Ask) says(spec AskSpec) bool {
	same := strings.TrimSpace
	if same(a.Question) != same(spec.Question) || same(a.BigPicture) != same(spec.BigPicture) || same(a.TechnicalDetails) != same(spec.TechnicalDetails) || a.Recommended != spec.Recommended || len(a.Options) != len(spec.Options) {
		return false
	}
	for i, o := range spec.Options {
		if a.Options[i].Key != o.Key || same(a.Options[i].Label) != same(o.Label) {
			return false
		}
	}
	return true
}

// Pass computes a plan and carries it out. Only the lock holder sends.
func (c *Courier) Pass(ctx context.Context) (plan Plan, err error) {
	defer func() { c.note(plan, err) }()
	if err := c.Writer.Held(); err != nil {
		return Plan{}, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	plan, _, err = c.Plan(ctx)
	if err != nil {
		return plan, err
	}
	now := c.Clock.Now()
	for i := range plan.Steps {
		s := &plan.Steps[i]
		if s.Action == ActNone {
			continue
		}
		s.Err = c.act(ctx, s, now, plan)
		s.Done = s.Err == nil
	}
	if err := c.recordMisses(plan, now); err != nil {
		return plan, err
	}
	return plan, c.recordSightings(plan)
}

func (c *Courier) act(ctx context.Context, s *Step, now time.Time, plan Plan) error {
	f := s.Item
	switch s.Action {
	case ActAsk, ActRefreshAsk:
		return c.Hub.Ask(ctx, askSpec(f, s.Key, plan.pages, now))
	case ActWritePage:
		_, err := c.PageWriter.Advance(ctx, plan.pages.job(f).request)
		return err
	case ActWithdraw:
		return c.Hub.Withdraw(ctx, *s.Ask, s.Why, now)
	case ActNotice:
		return c.Hub.Notify(ctx, noticeSpec(f, s.Key))
	case ActRecordDelivered:
		if s.Ask != nil {
			return c.record(f, home.EntryDelivered, now, "hub ask "+s.Ask.ID)
		}
		return c.record(f, home.EntryDelivered, now, "hub notice "+s.Notice.ID)
	case ActRecordAnswer, ActLandAnswer:
		return c.recordAnswer(s, now)
	}
	return nil
}

func (c *Courier) record(f *home.FeedItem, kind string, at time.Time, body string) error {
	via := home.ViaDesk
	if kind == home.EntryAnswer {
		via = home.ViaHub
	}
	// A Hub answer's note is the operator's phone text; it is fitted, never
	// refused, so the answer is not lost.
	_, err := c.Writer.AppendOperator(f.CommissionID, home.Entry{At: at, Kind: kind, Subject: f.Item.ID, Via: via, Body: c.Writer.FitEntry(body)})
	return err
}

// askSpec is a function of the item and its key, so an unchanged item is the
// same event on every pass (the one exception is its silence default, which is
// dropped once its day is too near for the Hub to apply). The operator reads
// the question and big picture cold on a phone, so they hold only the item's
// title and prose; the desk's key, the commission, the packet and any
// `Details:` line go to the technical details for the agent that acts on the
// answer (FILE-CONTRACT.md §5).
func askSpec(f *home.FeedItem, key string, pages Pages, now time.Time) AskSpec {
	page := pages.of(f)
	reason := f.Item.Reason
	if reason == "" && f.Item.Recommended != nil {
		reason = f.Item.Recommended.Label
	}
	spec := AskSpec{
		Key:              key,
		Subject:          f.Item.Title,
		Question:         operatordecision.Clip(f.Item.Title, operatordecision.MaxQuestion),
		BigPicture:       operatordecision.Clip(f.Item.Prose(), operatordecision.MaxBigPictureCodePoints),
		TechnicalDetails: operatordecision.Clip(technicalDetails(f, key, page, pages.effects[f.Key()]), operatordecision.MaxTechnicalDetailsCodePoints),
		Options:          f.Item.Choices(),
		Recommended:      f.Item.Recommended.Key,
		Reason:           operatordecision.Clip(reason, operatordecision.MaxReason),
		Explainer:        pages.explainer(f),
		Packet:           packetPath(f),
	}
	if silence := silenceFor(f.Item, now); silence != nil {
		spec.Default, spec.Deadline = silence.Key, silence.On
	}
	return spec
}

// minSilenceLead is how far ahead a silence day must be: the Hub applies a default no earlier than 12 hours
// after delivery and refuses a deadline that has passed.
const minSilenceLead = 13 * time.Hour

// silenceFor is the item's `If silent: <key> on <date>` as the Hub's default answer and deadline, so silence
// applies it (R9). Nothing is passed for an item without the line, with `none`, naming an option it does not
// offer, or whose day is too near; such an item stays open until answered.
func silenceFor(it home.NeedsYouItem, now time.Time) *home.Silence {
	if it.IfSilent == nil || !it.IfSilent.On.After(now.Add(minSilenceLead)) {
		return nil
	}
	if _, offered := it.Choice(it.IfSilent.Key); !offered {
		return nil
	}
	return it.IfSilent
}

// packetPath is the item's brief as an absolute path, or "" when it names
// none or only a relative path the desk cannot resolve.
func packetPath(f *home.FeedItem) string {
	if f.Item.Packet == "" || !filepath.IsAbs(f.Item.PacketPath()) {
		return ""
	}
	return f.Item.PacketPath()
}

// technicalDetails says where the item lives and what the next agent needs.
// The hash of the item's own page, or of the packet text its summary page
// shows, is in it, so a page that changed makes the item a changed request
// and the Hub's page is replaced.
func technicalDetails(f *home.FeedItem, key string, page *Page, effects map[string]string) string {
	lines := []string{deskKeyLine(key)}
	switch {
	case f.CommissionID != "":
		lines = append(lines, "Research commission: "+f.CommissionID)
	case f.Topic != "":
		lines = append(lines, "Research topic: "+f.Topic)
	default:
		lines = append(lines, "Raised by: the research manager")
	}
	lines = append(lines, "Needs-you item: "+f.Item.ID)
	if src := f.Item.Source; src != "" {
		lines = append(lines, "Item file: "+src)
	}
	if f.Item.Packet != "" {
		line := "Packet: " + f.Item.PacketPath()
		if page == nil && len(effects) > 0 {
			sum := sha256.Sum256([]byte(fmt.Sprint(effects)))
			line += " (on each answer: sha256 " + hex.EncodeToString(sum[:8]) + ")"
		}
		lines = append(lines, line)
	}
	if len(f.Inboxes) > 0 {
		lines = append(lines, "Answer lands in: "+filepath.Join(f.Inboxes[0].Dir, home.InboxFile))
	}
	switch {
	case page != nil && page.Written != "":
		lines = append(lines, "Page: made by the decision-page writer, "+page.Written+" (sha256 "+page.SHA256[:16]+")")
	case page != nil:
		lines = append(lines, "Explainer: "+f.Item.ExplainerPath()+" (sha256 "+page.SHA256[:16]+")")
	}
	if silence := f.Item.IfSilent; silence != nil {
		lines = append(lines, "If silent: "+silence.Key+" on "+silence.On.Format("2006-01-02"))
	}
	if f.Item.Details != "" {
		lines = append(lines, "Details: "+f.Item.Details)
	}
	return strings.Join(lines, "\n")
}

func noticeSpec(f *home.FeedItem, key string) NoticeSpec {
	title := f.Owner() + " · " + f.Item.ID + " — " + f.Item.Title
	body := "Needs you, in " + f.Sources[0] + "."
	if f.Item.Secret != "" {
		body += " A secret goes where the item says: " + f.Item.Secret + ". Never type it into the desk or a file."
	}
	return NoticeSpec{Key: key, Title: operatordecision.Clip(title, 200), Body: operatordecision.Clip(body, 1000)}
}

// Run passes every interval until ctx ends. Each action is logged; a problem
// is logged once until it changes.
func (c *Courier) Run(ctx context.Context, every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		c.logPass(c.Pass(ctx))
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (c *Courier) logPass(plan Plan, err error) {
	var acted []string
	for _, s := range plan.Steps {
		if s.Done {
			acted = append(acted, fmt.Sprintf("%s %s", s.Action, s.Key))
		}
	}
	sort.Strings(acted)
	for _, line := range acted {
		c.Logger.Printf("courier: %s", line)
	}
	if msg := problem(plan, err); msg != c.lastProblem {
		if msg != "" {
			c.Logger.Printf("courier: %s", msg)
		}
		c.lastProblem = msg
	}
}

// TestResult is the outcome of the delivery test.
type TestResult struct {
	Key string
	// Ask is the Hub's ask once it appeared; its ID is empty until then.
	Ask            Ask
	FirstDelivered time.Time
	// Expires is the test ask's deadline: nothing can withdraw it, so it
	// expires unanswered.
	Expires time.Time
	Waited  time.Duration
}

// testLifetime is the delivery-test ask's deadline.
const testLifetime = 15 * time.Minute

// testPage is the delivery test's page, built the way an item's summary page
// is, so the test shows that a decision's page reaches the operator.
func testPage(stamp string) string {
	return operatordecision.NewPage("Research desk delivery test").
		Lead("This checks that research decisions reach you with their page, like this one. Test "+stamp+".").
		Options(operatordecision.Facts{Options: []operatordecision.Option{{Key: "seen", Label: "Seen"}, {Key: "not-seen", Label: "Not seen"}}, Recommended: "seen"}, map[string]string{
			"seen": "Nothing else happens; research decisions reach you.", "not-seen": "Nothing else happens; tell the agent that ran the test.",
		}).
		IfNoAnswer("No answer is needed. It expires unanswered in 15 minutes.").HTML()
}

// Test raises one research-desk delivery test ask and waits up to wait for a
// device to receive it. Nothing is answered; the ask expires at its deadline.
func (c *Courier) Test(ctx context.Context, wait time.Duration) (TestResult, error) {
	if err := c.Writer.Held(); err != nil {
		return TestResult{}, err
	}
	if c.Hub == nil {
		return TestResult{}, errors.New("notification-hub is not configured for this instance")
	}
	now := c.Clock.Now()
	stamp := now.UTC().Format("20060102T150405Z")
	result := TestResult{Key: TestPrefix + stamp, Expires: now.Add(testLifetime)}
	err := c.Hub.Ask(ctx, AskSpec{
		Key:              result.Key,
		Subject:          "Research desk delivery test",
		Question:         testQuestion + stamp + " — no answer needed; it expires unanswered in 15 minutes.",
		TechnicalDetails: deskKeyLine(result.Key),
		Options:          []home.Option{{Key: "seen", Label: "Seen"}, {Key: "not-seen", Label: "Not seen"}},
		Recommended:      "seen",
		Reason:           "This only checks that research-desk decisions reach you.",
		Deadline:         result.Expires,
		Explainer:        testPage(stamp),
	})
	if err != nil {
		return result, err
	}
	start := time.Now()
	deadline := start.Add(wait)
	for {
		if asks, err := c.Hub.Asks(ctx, result.Key); err == nil && len(asks) > 0 {
			result.Ask, result.FirstDelivered = asks[0], asks[0].FirstDelivered
		}
		if !result.FirstDelivered.IsZero() || !time.Now().Before(deadline) {
			break
		}
		select {
		case <-ctx.Done():
			deadline = time.Now()
		case <-time.After(2 * time.Second):
		}
	}
	result.Waited = time.Since(start)
	return result, nil
}
