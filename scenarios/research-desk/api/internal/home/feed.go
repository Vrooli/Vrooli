package home

import (
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"
)

// FeedItem is one needs-you item (FILE-CONTRACT.md §5). Its identity is its
// owner plus the item ID. The owner is the level that keeps it: a topic
// (topics/<t>/DECISIONS.md), the supervisor (ATTENTION.md without a
// `Commission:` line), or, for items still written the older way, a live
// commission (its STATE.md, and ATTENTION.md copies naming it). Copies are one
// item: a topic's DECISIONS.md copy of a live commission's item on that topic,
// and a supervisor copy of a topic's item.
type FeedItem struct {
	// Item is the first copy read: the commission's STATE.md, then the
	// topic's DECISIONS.md, then ATTENTION.md.
	Item NeedsYouItem
	// Sources are every file holding the item.
	Sources []string
	// CommissionID owns a commission's item; empty for a level's.
	CommissionID string
	// Commission is nil for a level's item or one naming a missing folder.
	Commission *Commission
	// Topic owns a topic's item; empty for a commission's or the supervisor's.
	Topic string
	// Inboxes are the levels whose INBOX.md gets the item's answer: the
	// owning level, or the topics a commission's item works for.
	Inboxes []*Level
	// Answer is the item's answer record: its commission's OPERATOR.md
	// entry, or for a level's item the newest `answer` entry naming it in
	// the level's inbox or history (and, for a topic, the OPERATOR.md of the
	// topic's commissions, where the older way recorded it).
	Answer *Entry
	// Read is whether the answer was read: STATE.md rewritten after it, or
	// the inbox entry drained.
	Read Flag
	// Delivered is the desk's newest `delivered <ID>` entry in a commission's
	// OPERATOR.md; a level's item keeps no delivery receipt.
	Delivered *Entry
}

// Owner is the item's owner key: the commission ID, TopicOwner(topic) or
// SupervisorOwner.
func (f FeedItem) Owner() string {
	switch {
	case f.CommissionID != "":
		return f.CommissionID
	case f.Topic != "":
		return TopicOwner(f.Topic)
	}
	return SupervisorOwner
}

// Key is the item's identity, "<owner>/<ID>".
func (f FeedItem) Key() string { return f.Owner() + "/" + f.Item.ID }

// Answerable reports whether an answer can be written for the item, and why not.
func (f FeedItem) Answerable() (bool, string) {
	switch {
	case f.Item.Secret != "":
		return false, "secrets never go into a file; put it where the item says: " + f.Item.Secret
	case f.CommissionID != "" && f.Commission == nil:
		return false, "the item names commission " + f.CommissionID + ", which has no folder under cohorts/"
	default:
		return true, ""
	}
}

// Delivery is how the courier would carry an item to Notification Hub,
// before the per-commission switch is considered.
type Delivery string

// Delivery kinds.
const (
	// DeliverAsk raises a Hub decision with the item's options.
	DeliverAsk Delivery = "ask"
	// DeliverNotice sends a plain notice pointing at the item, no options.
	DeliverNotice Delivery = "notice"
	// DeliverNone means the courier does not send the item; the reason says why.
	DeliverNone Delivery = "none"
)

// DeliveryReason is why an item is or is not carried, as a code; the
// sentence beside it says the same for a reader.
type DeliveryReason string

// Delivery reasons. The first group is the item's own; the courier adds the
// rest when it plans a pass.
const (
	ReasonSchedulerDelivered DeliveryReason = "scheduler_delivered"
	ReasonAnswered           DeliveryReason = "answered"
	ReasonSecret             DeliveryReason = "secret"
	ReasonNoCommission       DeliveryReason = "no_commission"
	ReasonNoSteering         DeliveryReason = "no_steering"
	ReasonNoRecommendation   DeliveryReason = "no_recommendation"
	ReasonAsk                DeliveryReason = "ask"

	ReasonUnreadable     DeliveryReason = "unreadable"
	ReasonIDReused       DeliveryReason = "id_reused"
	ReasonSwitchOff      DeliveryReason = "switch_off"
	ReasonSchedulerGrace DeliveryReason = "scheduler_grace"
	ReasonHubUnavailable DeliveryReason = "hub_unavailable"
	ReasonInHub          DeliveryReason = "in_hub"
	ReasonHubDelivered   DeliveryReason = "hub_delivered"
	ReasonHubAnswered    DeliveryReason = "hub_answered"
	ReasonOrphaned       DeliveryReason = "orphaned"
	ReasonBaseline       DeliveryReason = "baseline"
	ReasonNotice         DeliveryReason = "notice"
	// ReasonPacketNotReady: a decision whose work/<ID>/ lacks a brief a cold
	// executor can run (or its page) is never sent.
	ReasonPacketNotReady DeliveryReason = "packet_not_ready"
)

// Delivery says how the courier would carry the item, why as a code, and
// why in words.
func (f FeedItem) Delivery() (Delivery, DeliveryReason, string) {
	switch {
	case f.Item.Delivered != "" && f.Delivered != nil && f.deskReceipt():
		return DeliverNone, ReasonHubDelivered, "delivered by the desk at " + f.Delivered.At.UTC().Format(time.RFC3339) + " (" + f.Delivered.Body + ")"
	case f.Item.Delivered != "" && f.deskReceipt():
		return DeliverNone, ReasonHubDelivered, "delivered by the desk (Delivered: " + f.Item.Delivered + "); its OPERATOR.md holds no receipt for it"
	case f.Item.Delivered != "":
		return DeliverNone, ReasonSchedulerDelivered, "the scheduler delivered it itself (Delivered: " + f.Item.Delivered + ")"
	case f.Answer != nil:
		return DeliverNone, ReasonAnswered, "already answered (" + f.Answer.Heading() + ")"
	case f.Item.Secret != "":
		return DeliverNotice, ReasonSecret, "a secret item is sent as a notice saying where the secret goes; it is never answered into a file"
	case f.CommissionID != "" && f.Commission == nil:
		return DeliverNotice, ReasonNoCommission, "it names a commission with no folder, so no one would read an answer; sent as a notice"
	case f.Commission != nil && !f.Commission.Steering.Value:
		return DeliverNotice, ReasonNoSteering, "the charter does not tell its scheduler to read OPERATOR.md, so an answer would not be read; sent as a notice until steering is enabled"
	case f.Item.Recommended == nil && f.Item.RecommendedProblem != "":
		return DeliverNotice, ReasonNoRecommendation, f.Item.RecommendedProblem + "; sent as a notice without options"
	case f.Item.Recommended == nil:
		return DeliverNotice, ReasonNoRecommendation, `no "Recommended: <key> — <label>" line; sent as a notice without options`
	default:
		return DeliverAsk, ReasonAsk, "sent as a Notification Hub decision with the item's options"
	}
}

// DeskRoute begins a `Delivered:` line copied from one of the desk's own
// deliveries.
const DeskRoute = "research-desk"

// deskReceipt reports whether the item's `Delivered:` line is the desk's own
// delivery: it names the Hub ask or notice of the desk's receipt in
// OPERATOR.md, or its route is the desk. A scheduler copies the desk's
// receipt into that line, so the line alone does not mean it delivered.
func (f FeedItem) deskReceipt() bool {
	line := strings.TrimSpace(f.Item.Delivered)
	if f.Delivered != nil {
		if fields := strings.Fields(f.Delivered.Body); len(fields) > 0 && strings.Contains(line, fields[len(fields)-1]) {
			return true
		}
	}
	route, _, _ := strings.Cut(line, " ")
	return strings.EqualFold(route, DeskRoute)
}

// Feed is the merged needs-you feed and what could not be read for it.
type Feed struct {
	Items []FeedItem
	// Legacy is text in a needs-you section that is not an item.
	Legacy []Legacy
	// Unreadable lists the files behind the feed that exist but cannot be
	// read: their items and answers are unknown, not absent.
	Unreadable []Evidence
	// blocked maps an owner to the reasons its items are unknown; the ""
	// owner is ATTENTION.md, whose items may belong to any owner.
	blocked map[string][]string
}

// Blocked says why an owner's items (FeedItem.Owner) cannot be known, or ""
// when they can. An unreadable ATTENTION.md blocks every owner.
func (f Feed) Blocked(owner string) string {
	reasons := append(append([]string(nil), f.blocked[""]...), f.blocked[owner]...)
	return strings.Join(reasons, "; ")
}

// NeedsYou is the merged feed: live commissions' items first, then topics',
// then the supervisor's. Items owned by ended commissions are left out; a
// topic's items stay whatever its commissions do.
func (s *Snapshot) NeedsYou() Feed {
	out := Feed{blocked: map[string][]string{}}
	byKey := map[string]int{}
	// byKey also maps "copy:<topic>/<ID>" of a live commission's item, and
	// "copy:/<ID>" of a topic's, to the item a copy of it joins.
	block := func(owner string, list []Evidence) {
		for _, e := range list {
			out.Unreadable = append(out.Unreadable, e)
			out.blocked[owner] = append(out.blocked[owner], e.Reason)
		}
	}
	add := func(f FeedItem, copyOf ...string) int {
		for _, key := range append([]string{f.Key()}, copyOf...) {
			if i, ok := byKey[key]; ok {
				out.Items[i].Sources = append(out.Items[i].Sources, f.Item.Source)
				return i
			}
		}
		byKey[f.Key()] = len(out.Items)
		f.Sources = []string{f.Item.Source}
		f.Answer, f.Read = s.answer(&f)
		out.Items = append(out.Items, f)
		return len(out.Items) - 1
	}
	for _, c := range s.Commissions {
		if c.State == StateEnded {
			continue
		}
		block(c.ID, c.Unreadable)
		for _, l := range c.Legacy {
			l.CommissionID = c.ID
			out.Legacy = append(out.Legacy, l)
		}
		for _, item := range c.NeedsYou {
			f := FeedItem{Item: item, CommissionID: c.ID, Commission: c, Delivered: c.Operator.Newest(EntryDelivered, item.ID)}
			for _, t := range c.AnswerTopics(&item) {
				if l := s.Level(TopicOwner(t)); l != nil {
					f.Inboxes = append(f.Inboxes, l)
				}
			}
			i := add(f)
			for _, t := range c.Topics {
				byKey["copy:"+t.Topic+"/"+item.ID] = i
			}
		}
	}
	for _, l := range s.Levels {
		block(l.Owner(), l.Unreadable)
		out.Legacy = append(out.Legacy, l.Legacy...)
		for _, item := range l.Items {
			i := add(FeedItem{Item: item, Topic: l.Topic, Inboxes: []*Level{l}}, "copy:"+l.Topic+"/"+item.ID)
			if _, ok := byKey["copy:/"+item.ID]; !ok {
				byKey["copy:/"+item.ID] = i
			}
		}
	}
	for _, e := range s.Unreadable {
		if filepath.Base(e.Source) == AttentionFile {
			block("", []Evidence{e})
		}
	}
	out.Legacy = append(out.Legacy, s.AttentionLegacy...)
	supervisor := s.Level(SupervisorOwner)
	for _, item := range s.Attention {
		if item.Commission == "" {
			f := FeedItem{Item: item}
			if supervisor != nil {
				f.Inboxes = []*Level{supervisor}
			}
			add(f, "copy:/"+item.ID)
			continue
		}
		c := s.Commission(item.Commission)
		if c != nil && c.State == StateEnded {
			continue
		}
		f := FeedItem{Item: item, CommissionID: item.Commission, Commission: c}
		if c != nil {
			f.Delivered = c.Operator.Newest(EntryDelivered, item.ID)
		}
		add(f)
	}
	rank := func(f FeedItem) int {
		switch {
		case f.CommissionID != "":
			return 0
		case f.Topic != "":
			return 1
		}
		return 2
	}
	sort.SliceStable(out.Items, func(i, j int) bool { return rank(out.Items[i]) < rank(out.Items[j]) })
	return out
}

// answer is the item's answer record and whether it was read (FeedItem.Answer).
func (s *Snapshot) answer(f *FeedItem) (*Entry, Flag) {
	if f.Commission != nil {
		e := f.Commission.Operator.Newest(EntryAnswer, f.Item.ID)
		if e == nil {
			return nil, Flag{}
		}
		return e, f.Commission.ReadSince(e)
	}
	if f.CommissionID != "" {
		return nil, Flag{}
	}
	var best *Entry
	read := Flag{}
	for _, l := range f.Inboxes {
		if a := l.AnswerTo(f.Item.ID); a != nil && (best == nil || a.At.After(best.At)) {
			e := a.Entry
			e.Subject = f.Item.ID
			best = &e
			read = Flag{Value: a.Drained, Evidence: From(a.Source, a.At)}
		}
	}
	for _, c := range s.Commissions {
		if f.Topic == "" || !slices.ContainsFunc(c.Topics, func(t TopicRef) bool { return t.Topic == f.Topic }) {
			continue
		}
		if e := c.Operator.Newest(EntryAnswer, f.Item.ID); e != nil && (best == nil || e.At.After(best.At)) {
			best, read = e, c.ReadSince(e)
		}
	}
	return best, read
}

// AnswerTopics are the topics an answer to the commission's item belongs to:
// its one topic; else the topic whose folder holds the item's packet or page;
// else all of its topics, so the answer is seen whichever carries the work on.
func (c *Commission) AnswerTopics(it *NeedsYouItem) []string {
	var topics []string
	for _, t := range c.Topics {
		if !slices.Contains(topics, t.Topic) {
			topics = append(topics, t.Topic)
		}
	}
	if it != nil && len(topics) > 1 {
		for _, path := range []string{it.Packet, it.ExplainerPath()} {
			for _, t := range topics {
				if strings.Contains(filepath.ToSlash(path), "/topics/"+t+"/") {
					return []string{t}
				}
			}
		}
	}
	return topics
}

// AllUnreadable gathers every file or folder of the home that exists but
// could not be read.
func (s *Snapshot) AllUnreadable() []Evidence {
	out := append([]Evidence(nil), s.Unreadable...)
	for _, c := range s.Commissions {
		out = append(out, c.Unreadable...)
	}
	for _, t := range s.Topics {
		out = append(out, t.Unreadable...)
	}
	for _, r := range s.Reviews {
		out = append(out, r.Unreadable...)
	}
	for _, l := range s.Levels {
		out = append(out, l.Unreadable...)
	}
	return out
}

// ReadSince is whether the scheduler rewrote STATE.md after an entry, so it
// has read it.
func (c *Commission) ReadSince(e *Entry) Flag {
	fact, ok := stat(filepath.Join(c.Dir, StateFile))
	if !ok {
		return Flag{Evidence: Unknown("no STATE.md")}
	}
	return Flag{Value: fact.Modified.After(e.At), Evidence: From(fact.Path, fact.Modified)}
}
