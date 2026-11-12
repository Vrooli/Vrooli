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

// WaitFile records when the courier first saw each item it holds back for
// its scheduler (see SchedulerGrace). Deleting it restarts the wait.
const WaitFile = "courier-wait.json"

// SchedulerGrace is how long the courier leaves an item to a scheduler that
// delivers by itself: a commission whose charter lacks the Decisions sentence,
// and every global item. Such a scheduler adds `Delivered:` when it delivers;
// the courier sends only what is still undelivered after the grace (S4c).
const SchedulerGrace = 2 * time.Hour

// SeenFile records when each needs-you item (FeedItem.Key) was first seen in
// the files, so an ID that reappears after an earlier item's answer is shown
// as reused instead of silently counted as answered (rulings F11). An item
// already there when the record began has the zero time: its first sight is
// unknown, so it is never called reused. An item that leaves is forgotten.
const SeenFile = "courier-seen.json"

// Switch is one courier switch. Turning it on records the items present at
// that moment as its baseline; they are never sent, so turning the courier on
// never notifies the operator about old items (F6).
type Switch struct {
	On bool      `json:"on"`
	At time.Time `json:"at"`
	// Baseline holds the item keys (FeedItem.Key) present when it was turned on.
	Baseline []string `json:"baseline"`
}

// Switches are the operator's courier choices. A commission without an entry
// is on only when the desk launched it (every item it has appeared after the
// launch); the global ATTENTION.md switch is off until turned on.
type Switches struct {
	Global      Switch            `json:"global"`
	Commissions map[string]Switch `json:"commissions"`
}

// For says whether the courier carries an item, and why.
func (s Switches) For(f *home.FeedItem) (bool, string) {
	if f.Commission == nil {
		if !s.Global.On {
			return false, "global items: off until the operator turns delivery on for them"
		}
		if s.Global.baselined(f) {
			return false, "global items: this item was already there when the courier was turned on at " + stamp(s.Global.At) + "; it is never sent"
		}
		return true, "global items: turned on by the operator"
	}
	on, why := s.ForCommission(f.Commission)
	if sw, ok := s.Commissions[f.Commission.ID]; on && ok && sw.baselined(f) {
		return false, "this item was already there when the courier was turned on at " + stamp(sw.At) + "; it is never sent"
	}
	return on, why
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

func (sw Switch) baselined(f *home.FeedItem) bool { return slices.Contains(sw.Baseline, f.Key()) }

// baselined reports whether the item was present when its switch was turned
// on, so it is never sent (rulings F6).
func (s Switches) baselined(f *home.FeedItem) bool {
	if f.Commission == nil {
		return s.Global.On && s.Global.baselined(f)
	}
	sw, ok := s.Commissions[f.Commission.ID]
	return ok && sw.On && sw.baselined(f)
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
	// a code (rulings P5).
	Why    string
	Reason home.DeliveryReason
	// Reused is set when the item's ID already carries an earlier item's
	// answer, closed ask or delivery (F11); those belong to the earlier item.
	Reused bool
	// Done and Err are set by a pass that carried the action out.
	Done bool
	Err  error
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
	// read; nothing is sent, recorded or forgotten for their owners (E1).
	Unreadable []home.Evidence
	blocked    func(owner string) string
}

// keep reports whether a remembered key's owner is blocked, so its record
// is kept as it was rather than forgotten as gone.
func (p Plan) keep(key string) bool {
	owner, _, _ := strings.Cut(key, "/")
	return p.blocked != nil && p.blocked(owner) != ""
}

// Courier runs passes.
type Courier struct {
	Reader interface {
		Snapshot(ctx context.Context, opts home.Options) (*home.Snapshot, error)
	}
	Writer *writer.Writer
	Hub    Hub
	Clock  schedule.NowSource
	Logger *log.Logger

	mu      sync.Mutex
	lastErr string

	hmu    sync.Mutex
	health Health
}

// Health is the courier's real state, shown in the overview (rulings E6):
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
	// LastDelivery is when the courier last sent an ask or a notice.
	LastDelivery time.Time
}

// Health returns the courier's state after its last pass.
func (c *Courier) Health() Health {
	c.hmu.Lock()
	defer c.hmu.Unlock()
	return c.health
}

// Withdraw is said wherever the operator sees the courier (rulings F15).
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

// note records a finished pass in Health.
func (c *Courier) note(plan Plan, err error) {
	c.hmu.Lock()
	defer c.hmu.Unlock()
	h := &c.health
	h.LastPass, h.Idle, h.Problem = c.Clock.Now(), plan.HubSkipped, ""
	switch {
	case err != nil:
		h.Problem = err.Error()
	case plan.HubRead != nil:
		h.Problem = "notification-hub unreadable: " + plan.HubRead.Error()
	case !plan.HubSkipped:
		h.LastReadBack = plan.At
	}
	if n := len(plan.Unreadable); n > 0 && h.Problem == "" {
		h.Problem = fmt.Sprintf("%d needs-you or answer files cannot be read, so nothing is sent or recorded for their items; first: %s", n, plan.Unreadable[0].Reason)
	}
	for _, s := range plan.Steps {
		if s.Err != nil && h.Problem == "" {
			h.Problem = fmt.Sprintf("%s %s failed: %v", s.Action, s.Key, s.Err)
		}
		if s.Done && (s.Action == ActAsk || s.Action == ActNotice) {
			h.LastDelivery = h.LastPass
		}
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
// Turning a switch on records the items it covers at this moment as its
// baseline; turning it off keeps nothing.
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
		for _, f := range feed.Items {
			if f.CommissionID == commission {
				sw.Baseline = append(sw.Baseline, f.Key())
			}
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
	plan := Plan{At: snap.Now, Switches: sw, Unreadable: feed.Unreadable, blocked: feed.Blocked}
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
	waits, err := c.waits()
	if err != nil {
		return Plan{}, nil, err
	}
	seen := map[string]time.Time{}
	if _, err := c.Writer.ReadState(SeenFile, &seen); err != nil {
		return Plan{}, nil, fmt.Errorf("%w; the courier sends nothing until it can tell a reused ID from an answered one", err)
	}
	plan.Steps = decide(feed, sw, asks, notices, plan.HubRead, plan.HubSkipped, waits, seen, snap.Now)
	return plan, snap, nil
}

func (c *Courier) waits() (map[string]time.Time, error) {
	waits := map[string]time.Time{}
	if _, err := c.Writer.ReadState(WaitFile, &waits); err != nil {
		return nil, err
	}
	return waits, nil
}

// selfDelivering reports whether the item's scheduler delivers it by itself.
func selfDelivering(f *home.FeedItem) bool {
	return f.Commission == nil || !f.Commission.Decisions.Value
}

// decide is the pure core of a pass.
func decide(feed home.Feed, sw Switches, asks []Ask, notices map[string]Notice, hubErr error, hubSkipped bool, waits, seen map[string]time.Time, now time.Time) []Step {
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
		s.On, s.Switch = sw.For(f)
		if f.Commission != nil {
			s.Key = AskPrefix + f.CommissionID + "/" + f.Item.ID
			s.Ask = byKey[s.Key]
			listed[s.Key] = true
		}
		if n, ok := notices[NoticePrefix+owner(f)+"/"+f.Item.ID]; ok {
			s.Notice = &n
		}
		answerable, _ := f.Answerable()
		switch a := s.Ask; {
		case feed.Blocked(f.CommissionID) != "":
			s.Reason, s.Why = home.ReasonUnreadable, "nothing is sent or recorded while the desk cannot read this item's files: "+feed.Blocked(f.CommissionID)
		case reused(f, a, seen[f.Key()]) != "":
			s.Reused, s.Reason, s.Why = true, home.ReasonIDReused, reused(f, a, seen[f.Key()])
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
		case !s.On && sw.baselined(f):
			s.Reason, s.Why = home.ReasonBaseline, s.Switch
		case !s.On:
			s.Reason, s.Why = home.ReasonSwitchOff, s.Switch
		case selfDelivering(f) && !graceOver(waits, f, now):
			s.Reason, s.Why = home.ReasonSchedulerGrace, "its scheduler delivers it by itself; sent only if it has no `Delivered:` line by "+graceEnd(waits, f, now).UTC().Format(time.RFC3339)
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

// reused says why an item's ID was reused, or "" (rulings F11): an answer, a
// closed Hub ask or a delivered entry for the same commission and ID dates
// from before the item was first seen, so it belongs to an earlier item. An
// answer given after the item appeared settles it.
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

// recordSeen keeps the first sight of each item in the feed and forgets the
// rest. The first record marks every item present as already there.
func (c *Courier) recordSeen(plan Plan) error {
	old := map[string]time.Time{}
	found, err := c.Writer.ReadState(SeenFile, &old)
	if err != nil {
		return err
	}
	seen := map[string]time.Time{}
	for key, first := range old {
		if plan.keep(key) {
			seen[key] = first
		}
	}
	for _, s := range plan.Steps {
		if s.Item == nil || plan.keep(s.Item.Key()) {
			continue
		}
		key := s.Item.Key()
		switch first, ok := old[key]; {
		case ok:
			seen[key] = first
		case found:
			seen[key] = plan.At
		default:
			seen[key] = time.Time{}
		}
	}
	if found && maps.EqualFunc(old, seen, time.Time.Equal) {
		return nil
	}
	_, err = c.Writer.WriteState(SeenFile, seen)
	return err
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

func graceEnd(waits map[string]time.Time, f *home.FeedItem, now time.Time) time.Time {
	first, ok := waits[f.Key()]
	if !ok {
		first = now
	}
	return first.Add(SchedulerGrace)
}

func graceOver(waits map[string]time.Time, f *home.FeedItem, now time.Time) bool {
	return !now.Before(graceEnd(waits, f, now))
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
	return plan, errors.Join(c.recordWaits(plan), c.recordSeen(plan))
}

// recordWaits keeps the first-seen time of each item the courier is holding
// for its scheduler, and forgets every other item.
func (c *Courier) recordWaits(plan Plan) error {
	old, err := c.waits()
	if err != nil {
		return err
	}
	waits := map[string]time.Time{}
	for key, first := range old {
		if plan.keep(key) {
			waits[key] = first
		}
	}
	for _, s := range plan.Steps {
		f := s.Item
		if f == nil || plan.keep(f.Key()) || !s.On || s.Ask != nil || f.Delivered != nil || s.Delivery == home.DeliverNone || !selfDelivering(f) {
			continue
		}
		if first, ok := old[f.Key()]; ok {
			waits[f.Key()] = first
		} else {
			waits[f.Key()] = plan.At
		}
	}
	if maps.EqualFunc(old, waits, time.Time.Equal) {
		return nil
	}
	_, err = c.Writer.WriteState(WaitFile, waits)
	return err
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
	// refused, so the answer is not lost (F9).
	_, err := c.Writer.AppendOperator(f.CommissionID, home.Entry{At: at, Kind: kind, Subject: f.Item.ID, Via: via, Body: home.FitEntryText(body)})
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

// Run passes every interval until ctx ends. A failure is logged once until
// it changes.
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
	msg := ""
	switch {
	case err != nil:
		msg = err.Error()
	case plan.HubRead != nil:
		msg = "notification-hub unreadable: " + plan.HubRead.Error()
	}
	var acted []string
	for _, s := range plan.Steps {
		if s.Err != nil {
			acted = append(acted, fmt.Sprintf("%s %s failed: %v", s.Action, s.Key, s.Err))
		} else if s.Done {
			acted = append(acted, fmt.Sprintf("%s %s", s.Action, s.Key))
		}
	}
	sort.Strings(acted)
	for _, line := range acted {
		c.Logger.Printf("courier: %s", line)
	}
	if msg != c.lastErr && msg != "" {
		c.Logger.Printf("courier: %s", msg)
	}
	c.lastErr = msg
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
