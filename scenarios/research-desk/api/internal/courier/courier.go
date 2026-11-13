// Package courier carries needs-you items to Notification Hub and answers
// back into OPERATOR.md (SYSTEM.md §7). One pass every minute in the instance
// holding the write lock; no watcher, no pending store: every pass re-reads
// the files and the Hub. Items go out as decision and notification events
// through Vrooli Events; an unchanged item is the same event, so a repeated
// send is harmless.
package courier

import (
	"context"
	"errors"
	"fmt"
	"log"
	"maps"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

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

// Files of the earlier form, read once to build SightingsFile and then
// removed: first sightings, first holds for a scheduler, and each switch's
// list of the items present when it was turned on.
const (
	oldSightingsFile = "courier-seen.json"
	oldHoldsFile     = "courier-wait.json"
)

// SchedulerGrace is how long the courier leaves an item to a scheduler that
// delivers by itself: a commission whose charter lacks the Decisions sentence,
// and every global item. Such a scheduler adds `Delivered:` when it delivers;
// the courier sends only what is still undelivered after the grace.
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
	// fromOldFiles is true when the record was built from the earlier files,
	// which are replaced on the next write.
	fromOldFiles bool
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
// launch); the global ATTENTION.md switch is off until turned on.
type Switches struct {
	Global      Switch            `json:"global"`
	Commissions map[string]Switch `json:"commissions"`
}

// For says whether the courier carries an item, and why; present is true
// when the item was there before its switch was turned on.
func (s Switches) For(f *home.FeedItem, sightings Sightings) (on, present bool, why string) {
	if f.Commission == nil {
		if !s.Global.On {
			return false, false, "global items: off until the operator turns delivery on for them"
		}
		if sightings.before(f, s.Global.At) {
			return false, true, "global items: this item was already there when the courier was turned on at " + stamp(s.Global.At) + "; it is never sent"
		}
		return true, false, "global items: turned on by the operator"
	}
	on, why = s.ForCommission(f.Commission)
	if sw, ok := s.Commissions[f.Commission.ID]; on && ok && sightings.before(f, sw.At) {
		return false, true, "this item was already there when the courier was turned on at " + stamp(sw.At) + "; it is never sent"
	}
	return on, false, why
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

// any reports whether the Hub needs reading at all.
func (s Switches) any(snap *home.Snapshot) bool {
	if s.Global.On {
		return true
	}
	for _, sw := range s.Commissions {
		if sw.On {
			return true
		}
	}
	for _, c := range snap.Commissions {
		if c.State != home.StateEnded && c.LaunchedBy() == "desk" {
			return true
		}
	}
	return false
}

// Action is what a pass does for one step.
type Action string

// Actions.
const (
	ActNone            Action = "none"
	ActAsk             Action = "ask"
	ActNotice          Action = "notice"
	ActRecordDelivered Action = "record-delivered"
	ActRecordAnswer    Action = "record-answer"
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
// its own in OPERATOR.md and its Hub ask was not answered. An expired or
// withdrawn ask leaves the item waiting; it is answered in the desk.
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
	// HubSkipped is true when no switch is on, so the Hub was not read.
	HubSkipped bool
	// Unreadable lists needs-you and answer files that exist but cannot be
	// read; nothing is sent, recorded or forgotten for their owners.
	Unreadable []home.Evidence
	// Recorded is the newest delivery the files record: a `delivered`
	// entry in any OPERATOR.md. It survives a restart, which the courier's
	// own memory of what it sent does not.
	Recorded  time.Time
	blocked   func(owner string) string
	sightings Sightings
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
	Clock  schedule.NowSource
	Logger *log.Logger

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
	// Idle is true when no switch is on, so the last pass read nothing.
	Idle bool
	// LastReadBack is when the Hub's asks were last read.
	LastReadBack time.Time
	// LastDelivery is the later of when this process last sent an ask or a
	// notice and the newest delivery recorded in the files.
	LastDelivery time.Time
}

// Health returns the courier's state after its last pass.
func (c *Courier) Health() Health {
	c.hmu.Lock()
	defer c.hmu.Unlock()
	return c.health
}

// Withdraw is said wherever the operator sees the courier.
const Withdraw = "the desk cannot withdraw an ask once sent; an answered or expired ask closes it"

// Say is the courier's health in the operator's words: whether it works,
// and the times behind that.
func (h Health) Say() (bool, string) {
	when := func(t time.Time) string {
		if t.IsZero() {
			return "never"
		}
		return stamp(t)
	}
	switch {
	case h.LastPass.IsZero():
		return false, "no pass yet; " + Withdraw
	case h.Problem != "":
		return false, fmt.Sprintf("last pass %s failed: %s; answers last read back %s; last delivery %s; %s", stamp(h.LastPass), h.Problem, when(h.LastReadBack), when(h.LastDelivery), Withdraw)
	case h.Idle:
		return true, fmt.Sprintf("last pass %s: no switch on, nothing is sent; last delivery %s; %s", stamp(h.LastPass), when(h.LastDelivery), Withdraw)
	}
	return true, fmt.Sprintf("last pass %s; answers read back %s; last delivery %s; %s", stamp(h.LastPass), when(h.LastReadBack), when(h.LastDelivery), Withdraw)
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
	return strings.Join(failed, "; ")
}

// note records a finished pass in Health.
func (c *Courier) note(plan Plan, err error) {
	c.hmu.Lock()
	defer c.hmu.Unlock()
	h := &c.health
	h.LastPass, h.Idle, h.Problem = c.Clock.Now(), plan.HubSkipped, problem(plan, err)
	if err == nil && plan.HubRead == nil && !plan.HubSkipped {
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

// sightings reads the first sightings. Before the file exists it builds them
// once from the earlier files: their first sightings, their first holds for a
// scheduler (whichever is earlier wins), and each switch's list of items
// present when it was turned on (sighted no later than the switch).
func (c *Courier) sightings() (Sightings, error) {
	s := Sightings{First: map[string]time.Time{}}
	found, err := c.Writer.ReadState(SightingsFile, &s.First)
	if err != nil {
		return s, fmt.Errorf("%w; the courier sends nothing until it can read when each item first appeared", err)
	}
	if found {
		s.found = true
		return s, nil
	}
	earlier := func(key string, t time.Time) {
		if old, ok := s.First[key]; !ok || t.Before(old) {
			s.First[key] = t
		}
	}
	for _, name := range []string{oldSightingsFile, oldHoldsFile} {
		old := map[string]time.Time{}
		ok, err := c.Writer.ReadState(name, &old)
		if err != nil {
			return s, fmt.Errorf("%w; the courier sends nothing until it can read when each item first appeared", err)
		}
		s.found = s.found || ok
		for key, t := range old {
			earlier(key, t)
		}
	}
	type oldSwitch struct {
		On       bool      `json:"on"`
		At       time.Time `json:"at"`
		Baseline []string  `json:"baseline"`
	}
	var old struct {
		Global      oldSwitch            `json:"global"`
		Commissions map[string]oldSwitch `json:"commissions"`
	}
	if _, err := c.Writer.ReadState(SwitchFile, &old); err != nil {
		return s, fmt.Errorf("%w; the courier sends nothing until it can read its switches", err)
	}
	for _, sw := range append(slices.Collect(maps.Values(old.Commissions)), old.Global) {
		for _, key := range sw.Baseline {
			s.found = true
			if _, ok := s.First[key]; !ok {
				s.First[key] = time.Time{}
			} else if sw.On {
				earlier(key, sw.At)
			}
		}
	}
	s.fromOldFiles = s.found
	return s, nil
}

// writeSightings writes next when it differs from what was read; a record
// built from the earlier files replaces them, and the switches are written
// again without their item lists.
func (c *Courier) writeSightings(old Sightings, next map[string]time.Time) error {
	if old.found && !old.fromOldFiles && maps.EqualFunc(old.First, next, time.Time.Equal) {
		return nil
	}
	if _, err := c.Writer.WriteState(SightingsFile, next); err != nil {
		return err
	}
	if !old.fromOldFiles {
		return nil
	}
	s, err := c.Switches()
	if err != nil {
		return err
	}
	if _, err := c.Writer.WriteState(SwitchFile, s); err != nil {
		return err
	}
	return errors.Join(c.Writer.RemoveState(oldSightingsFile), c.Writer.RemoveState(oldHoldsFile))
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
	case !sw.any(snap):
		plan.HubSkipped = true
	case c.Hub == nil:
		plan.HubRead = errors.New("notification-hub is not configured for this instance")
	default:
		if plan.HubRead = c.Hub.Sendable(ctx); plan.HubRead != nil {
			break
		}
		if asks, plan.HubRead = c.Hub.Asks(ctx, AskPrefix); plan.HubRead == nil {
			notices, plan.HubRead = c.Hub.Notices(ctx)
		}
	}
	if plan.sightings, err = c.sightings(); err != nil {
		return Plan{}, nil, err
	}
	plan.Steps = decide(feed, sw, asks, notices, plan.HubRead, plan.HubSkipped, plan.sightings, snap.Now)
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

// selfDelivering reports whether the item's scheduler delivers it by itself.
func selfDelivering(f *home.FeedItem) bool {
	return f.Commission == nil || !f.Commission.Decisions.Value
}

// decide is the pure core of a pass.
func decide(feed home.Feed, sw Switches, asks []Ask, notices map[string]Notice, hubErr error, hubSkipped bool, sightings Sightings, now time.Time) []Step {
	// An item whose text changed while its ask was open has two asks; the
	// open one, then the newest, speaks for it.
	byKey := make(map[string]*Ask, len(asks))
	for i := range asks {
		a := &asks[i]
		if old := byKey[a.Key]; old == nil || (a.Open() && !old.Open()) || (a.Open() == old.Open() && a.CreatedAt.After(old.CreatedAt)) {
			byKey[a.Key] = a
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
		if f.Commission != nil {
			s.Key = AskPrefix + f.CommissionID + "/" + f.Item.ID
			s.Ask = byKey[s.Key]
			listed[s.Key] = true
		}
		if n, ok := notices[NoticePrefix+owner(f)+"/"+f.Item.ID]; ok {
			s.Notice = &n
		}
		answerable, _ := f.Answerable()
		first, _ := sightings.at(f)
		reusedWhy := reused(f, s.Ask, first)
		switch a := s.Ask; {
		case feed.Blocked(f.CommissionID) != "":
			s.Reason, s.Why = home.ReasonUnreadable, "nothing is sent or recorded while the desk cannot read this item's files: "+feed.Blocked(f.CommissionID)
		case reusedWhy != "":
			s.Reused, s.Reason, s.Why = true, home.ReasonIDReused, reusedWhy
		case a != nil && a.State == "answered" && f.Answer == nil && answerable:
			s.Action, s.Reason, s.Why = ActRecordAnswer, home.ReasonHubAnswered, "answered in Notification Hub"
		case a != nil && a.Open() && (f.Answer != nil || f.Item.Delivered != ""):
			s.Reason, s.Why = home.ReasonInHub, stillOpen(a, "answered or delivered elsewhere")
		case a != nil && !a.FirstDelivered.IsZero() && f.Delivered == nil && f.Answer == nil:
			s.Action, s.Reason, s.Why = ActRecordDelivered, home.ReasonHubDelivered, "the Hub delivered the ask"
		case a != nil:
			s.Reason, s.Why = home.ReasonInHub, "Hub ask "+a.State
			if !a.Open() && a.State != "answered" {
				s.Why += "; answer it in the desk"
			}
		case s.Delivery == home.DeliverNone:
		case present:
			s.Reason, s.Why = home.ReasonBaseline, s.Switch
		case !s.On:
			s.Reason, s.Why = home.ReasonSwitchOff, s.Switch
		case selfDelivering(f) && now.Before(sightings.graceEnd(f, now)):
			s.Reason, s.Why = home.ReasonSchedulerGrace, "its scheduler delivers it by itself; sent only if it has no `Delivered:` line by "+stamp(sightings.graceEnd(f, now))
		case hubSkipped:
			s.Reason, s.Why = home.ReasonHubUnavailable, "the Hub was not read"
		case hubErr != nil:
			s.Reason, s.Why = home.ReasonHubUnavailable, "not sent: "+hubErr.Error()
		case s.Delivery == home.DeliverAsk:
			s.Action = ActAsk
		case f.Delivered != nil:
			s.Reason, s.Why = home.ReasonHubDelivered, "notice delivered "+f.Delivered.At.UTC().Format(time.RFC3339)
		default:
			s.Key = NoticePrefix + owner(f) + "/" + f.Item.ID
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
		steps = append(steps, s)
	}
	for _, a := range byKey {
		if listed[a.Key] || !a.Open() {
			continue
		}
		why := "its item left STATE.md and ATTENTION.md, or its commission ended"
		if commission, _, _ := strings.Cut(strings.TrimPrefix(a.Key, AskPrefix), "/"); feed.Blocked(commission) != "" {
			why = "its item's files cannot be read: " + feed.Blocked(commission)
		}
		steps = append(steps, Step{Key: a.Key, Ask: a, Delivery: home.DeliverNone, Action: ActNone, Reason: home.ReasonOrphaned, Why: stillOpen(a, why)})
	}
	n := len(feed.Items)
	sort.SliceStable(steps[n:], func(i, j int) bool { return steps[n+i].Key < steps[n+j].Key })
	return steps
}

// reused says why an item's ID was reused, or "": an answer, a
// closed Hub ask or a delivered entry for the same commission and ID dates
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
	what := "answer it in the desk"
	if f.Commission != nil {
		what = "answer it in the desk, or tell its supervisor to give the item a new ID"
	}
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

// stillOpen explains an open ask the desk no longer needs: Notification Hub
// lets only its producer withdraw through a verified identity or an Agent
// Manager event, so the ask stays until its deadline.
func stillOpen(a *Ask, why string) string {
	out := why + "; the Hub ask stays open"
	if !a.Deadline.IsZero() {
		out += " until " + a.Deadline.UTC().Format(time.RFC3339)
	}
	return out + " (the desk has no way to withdraw it); an answer there is ignored"
}

func owner(f *home.FeedItem) string {
	if f.CommissionID == "" {
		return "global"
	}
	return f.CommissionID
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
		s.Err = c.act(ctx, s, now)
		s.Done = s.Err == nil
	}
	return plan, c.recordSightings(plan)
}

func (c *Courier) act(ctx context.Context, s *Step, now time.Time) error {
	f := s.Item
	switch s.Action {
	case ActAsk:
		return c.Hub.Ask(ctx, askSpec(f))
	case ActNotice:
		return c.Hub.Notify(ctx, noticeSpec(f, s.Key))
	case ActRecordDelivered:
		if s.Ask != nil {
			return c.record(f, home.EntryDelivered, now, "hub ask "+s.Ask.ID)
		}
		return c.record(f, home.EntryDelivered, now, "hub notice "+s.Notice.ID)
	case ActRecordAnswer:
		return c.record(f, home.EntryAnswer, now, home.AnswerBody(s.Ask.Answer, s.Ask.AnswerLabel, s.Ask.Note))
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

const questionLimit = 600

// askSpec is a pure function of the item, so an unchanged item is the same
// event on every pass; the Hub's default deadline applies.
func askSpec(f *home.FeedItem) AskSpec {
	return AskSpec{
		Key:         AskPrefix + f.CommissionID + "/" + f.Item.ID,
		Question:    clip(f.CommissionID+" · "+f.Item.ID+" — "+f.Item.Title, questionLimit),
		BigPicture:  clip(f.Item.Body, 4000),
		Options:     f.Item.Choices(),
		Recommended: f.Item.Recommended.Key,
		Reason:      clip(f.Item.Recommended.Label, 300),
	}
}

func noticeSpec(f *home.FeedItem, key string) NoticeSpec {
	title := owner(f) + " · " + f.Item.ID + " — " + f.Item.Title
	body := "Needs you, in " + f.Sources[0] + "."
	if f.Item.Secret != "" {
		body += " A secret goes where the item says: " + f.Item.Secret + ". Never type it into the desk or a file."
	}
	return NoticeSpec{Key: key, Title: clip(title, 200), Body: clip(body, 1000)}
}

func clip(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s
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
		Key:         result.Key,
		Question:    testQuestion + stamp + " — no answer needed; it expires unanswered in 15 minutes.",
		Options:     []home.Option{{Key: "seen", Label: "Seen"}, {Key: "not-seen", Label: "Not seen"}},
		Recommended: "seen",
		Reason:      "This only checks that research-desk decisions reach you.",
		Deadline:    result.Expires,
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
