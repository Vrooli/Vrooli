package home

import (
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FeedItem is one needs-you item (FILE-CONTRACT.md §5). Its identity is the
// owning commission plus the item ID: a commission's STATE.md item and an
// ATTENTION.md item with the same ID and a `Commission:` line naming that
// commission are one item; an ATTENTION.md item without that line is global
// and never merged with a commission's item.
type FeedItem struct {
	// Item is the commission's STATE.md copy when one exists, else
	// ATTENTION.md's.
	Item NeedsYouItem
	// Sources are every file holding the item.
	Sources []string
	// CommissionID owns the item; empty for a global item.
	CommissionID string
	// Commission is nil for a global item or one naming a missing folder.
	Commission *Commission
	// Answer is the newest answer entry for the ID in the commission's
	// OPERATOR.md.
	Answer *Entry
	// Read is whether the supervisor has read the answer: STATE.md was
	// rewritten after it ("a unit has completed since").
	Read Flag
	// Delivered is the desk's newest `delivered <ID>` entry.
	Delivered *Entry
}

// Key is the item's identity, "<commission>/<ID>" or "/<ID>" for a global item.
func (f FeedItem) Key() string { return f.CommissionID + "/" + f.Item.ID }

// Answerable reports whether an answer can be written for the item, and why not.
func (f FeedItem) Answerable() (bool, string) {
	switch {
	case f.Item.Secret != "":
		return false, "secrets never go into a file; put it where the item says: " + f.Item.Secret
	case f.CommissionID == "":
		return false, "a global ATTENTION.md item has no commission whose supervisor would read an answer; act where the item says"
	case f.Commission == nil:
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
		return DeliverNone, ReasonAnswered, "already answered in OPERATOR.md"
	case f.Item.Secret != "":
		return DeliverNotice, ReasonSecret, "a secret item is sent as a notice saying where the secret goes; it is never answered into a file"
	case f.CommissionID == "" || f.Commission == nil:
		return DeliverNotice, ReasonNoCommission, "no commission would read an answer, so it is sent as a notice"
	case !f.Commission.Steering.Value:
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
	// owner is ATTENTION.md, whose items may name any commission.
	blocked map[string][]string
}

// Blocked says why an owner's items (a commission ID, or "" for global
// items) cannot be known, or "" when they can. An unreadable ATTENTION.md
// blocks every owner.
func (f Feed) Blocked(owner string) string {
	reasons := f.blocked[""]
	if owner != "" {
		reasons = append(append([]string(nil), reasons...), f.blocked[owner]...)
	}
	return strings.Join(reasons, "; ")
}

// NeedsYou is the merged feed. Items owned by ended commissions are left
// out; commission items come first.
func (s *Snapshot) NeedsYou() Feed {
	out := Feed{blocked: map[string][]string{}}
	byKey := map[string]int{}
	add := func(owner string, c *Commission, item NeedsYouItem) {
		key := owner + "/" + item.ID
		if i, ok := byKey[key]; ok {
			out.Items[i].Sources = append(out.Items[i].Sources, item.Source)
			return
		}
		byKey[key] = len(out.Items)
		f := FeedItem{Item: item, Sources: []string{item.Source}, CommissionID: owner, Commission: c}
		if c != nil {
			f.Answer = c.Operator.Newest(EntryAnswer, item.ID)
			f.Delivered = c.Operator.Newest(EntryDelivered, item.ID)
			if f.Answer != nil {
				f.Read = c.ReadSince(f.Answer)
			}
		}
		out.Items = append(out.Items, f)
	}
	for _, c := range s.Commissions {
		if c.State == StateEnded {
			continue
		}
		for _, e := range c.Unreadable {
			out.Unreadable = append(out.Unreadable, e)
			out.blocked[c.ID] = append(out.blocked[c.ID], e.Reason)
		}
		for _, l := range c.Legacy {
			l.CommissionID = c.ID
			out.Legacy = append(out.Legacy, l)
		}
		for _, item := range c.NeedsYou {
			add(c.ID, c, item)
		}
	}
	for _, e := range s.Unreadable {
		if filepath.Base(e.Source) == AttentionFile {
			out.Unreadable = append(out.Unreadable, e)
			out.blocked[""] = append(out.blocked[""], e.Reason)
		}
	}
	out.Legacy = append(out.Legacy, s.AttentionLegacy...)
	for _, item := range s.Attention {
		c := s.Commission(item.Commission)
		if c != nil && c.State == StateEnded {
			continue
		}
		add(item.Commission, c, item)
	}
	sort.SliceStable(out.Items, func(i, j int) bool { return out.Items[i].CommissionID != "" && out.Items[j].CommissionID == "" })
	return out
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
