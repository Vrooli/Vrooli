package courier

import (
	"path/filepath"
	"slices"
	"strings"
	"time"

	"research-desk/internal/home"
)

// Every answer lands at the level that owns its item, whether or not anything
// is running when the operator answers (docs/agent-system/LEVELS.md):
//   - an `answer` entry in that level's INBOX.md, which the level's next agent
//     drains: the topic for a topic's item, the research home for the
//     supervisor's, the topics a commission's item works for;
//   - for a commission's item, also its `answer <ID>` receipt in the
//     commission's OPERATOR.md, which a live commission reads.
//
// The operator answers whenever they read the phone, so the item may have left
// every file by then (its commission ended, or its agent removed it). The ask
// still names its owner and ID, and its technical details name the packet, so
// the answer lands all the same.

// Owners finds an answer's level and records from one snapshot.
type Owners struct {
	snap        *home.Snapshot
	commissions map[string]*home.Commission
}

func ownersOf(snap *home.Snapshot) Owners {
	o := Owners{snap: snap, commissions: map[string]*home.Commission{}}
	if snap != nil {
		for _, c := range snap.Commissions {
			o.commissions[c.ID] = c
		}
	}
	return o
}

// answered reports whether the commission's OPERATOR.md holds an answer to item.
func (o Owners) answered(commission, item string) bool {
	c := o.commissions[commission]
	return c != nil && c.Operator.Newest(home.EntryAnswer, item) != nil
}

// item is the needs-you item still written in the commission's STATE.md: an
// ended commission's files stay as they were.
func (o Owners) item(commission, id string) *home.NeedsYouItem {
	if c := o.commissions[commission]; c != nil {
		for i := range c.NeedsYou {
			if c.NeedsYou[i].ID == id {
				return &c.NeedsYou[i]
			}
		}
	}
	return nil
}

// topicLevels are the levels of the topics that have a folder.
func (o Owners) topicLevels(topics []string) []*home.Level {
	var out []*home.Level
	for _, t := range topics {
		if l := o.snap.Level(home.TopicOwner(t)); l != nil {
			out = append(out, l)
		}
	}
	return out
}

// askKey is the Hub key of an item's ask: AskPrefix + owner + "/" + ID. A
// topic's item that a commission on the topic raised before keeps that
// commission's ask, so moving an item from a commission's STATE.md to its
// topic's DECISIONS.md never asks the operator twice.
func (o Owners) askKey(f *home.FeedItem, byKey map[string]*Ask) string {
	key := AskPrefix + f.Owner() + "/" + f.Item.ID
	if f.Topic == "" || byKey[key] != nil || o.snap == nil {
		return key
	}
	for _, c := range o.snap.Commissions {
		alias := AskPrefix + c.ID + "/" + f.Item.ID
		if byKey[alias] != nil && slices.Contains(c.AnswerTopics(nil), f.Topic) {
			return alias
		}
	}
	return key
}

// itemOf is the item ID an ask key names.
func itemOf(key string) string { return key[strings.LastIndex(key, "/")+1:] }

// afterAnswer decides what to do with an answered Hub ask whose item is in no
// file. ok is false when the answer is already recorded, or its records
// cannot be read yet.
func afterAnswer(feed home.Feed, o Owners, a *Ask) (Step, bool) {
	owner, id := commissionOf(a.Key), itemOf(a.Key)
	step := Step{Key: a.Key, Ask: a, Delivery: home.DeliverNone, Action: ActLandAnswer, Reason: home.ReasonOrphaned}
	waits := func(blocked string) (Step, bool) {
		step.Action, step.Inboxes = ActNone, nil
		step.Why = "answered in Notification Hub after its item left the files; not recorded while its files cannot be read: " + blocked
		return step, true
	}
	if why := feed.Blocked(owner); why != "" {
		return waits(why)
	}
	why := "answered in Notification Hub after its item left the files; recorded in "
	if c := o.commissions[owner]; c != nil {
		if o.answered(owner, id) {
			return Step{}, false
		}
		step.Inboxes = o.topicLevels(c.AnswerTopics(o.item(owner, id)))
		for _, l := range step.Inboxes {
			if blocked := feed.Blocked(l.Owner()); blocked != "" {
				return waits(blocked)
			}
		}
		step.Why = why + inboxNames(step.Inboxes, filepath.Join(c.Dir, home.OperatorFile))
		if len(step.Inboxes) == 0 {
			step.Why += "; it names no topic whose next agent would read it, so say it again with research-desk say"
		}
		return step, true
	}
	level := o.snap.Level(owner)
	if level == nil {
		if !home.IsLevelOwner(owner) {
			return Step{}, false
		}
		step.Action, step.Why = ActNone, "answered in Notification Hub, but "+owner+" has no folder in the research home to record it in"
		return step, true
	}
	if level.Holds(a.ID) {
		return Step{}, false
	}
	step.Inboxes = []*home.Level{level}
	step.Why = why + inboxNames(step.Inboxes, "")
	return step, true
}

// answerRecord says where an item's answer is recorded, in words.
func answerRecord(f *home.FeedItem) string {
	receipt := ""
	if f.Commission != nil {
		receipt = filepath.Join(f.Commission.Dir, home.OperatorFile)
	}
	return inboxNames(f.Inboxes, receipt)
}

func inboxNames(levels []*home.Level, receipt string) string {
	var names []string
	for _, l := range levels {
		names = append(names, filepath.Join(l.Dir, home.InboxFile))
	}
	if receipt != "" {
		names = append(names, receipt)
	}
	return strings.Join(names, " and ")
}

// recordAnswer writes the answer to each inbox, then the commission's receipt.
// The receipt (or, for a level's item, the inbox entry) marks the answer
// landed, so a pass that fails part way repeats only what did not land: an
// inbox entry is found again by the ask id it names.
func (c *Courier) recordAnswer(s *Step, now time.Time) error {
	id := itemOf(s.Key)
	body := inboxAnswer(id, s.Ask)
	for _, l := range s.Inboxes {
		if _, _, err := c.Writer.AppendInbox(l.Topic, s.Ask.ID, now, "operator", home.EntryAnswer, body); err != nil {
			return err
		}
	}
	commission := commissionOf(s.Key)
	if s.Item != nil {
		commission = ""
		if s.Item.Commission != nil {
			commission = s.Item.CommissionID
		}
	}
	if commission == "" || home.IsLevelOwner(commission) {
		return nil
	}
	_, err := c.Writer.AppendOperator(commission, home.Entry{At: now, Kind: home.EntryAnswer, Subject: id, Via: home.ViaHub, Body: c.Writer.FitEntry(home.AnswerBody(s.Ask.Answer, s.Ask.AnswerLabel, s.Ask.Note))})
	return err
}

// inboxAnswer is the inbox entry body. The packet is read from the ask's
// technical details, which the desk wrote, so it is known even after the item
// left the files.
func inboxAnswer(id string, a *Ask) string {
	packet := ""
	for _, line := range strings.Split(a.TechnicalDetails, "\n") {
		if p, ok := strings.CutPrefix(strings.TrimSpace(line), "Packet: "); ok {
			packet, _, _ = strings.Cut(p, " (on each answer")
			break
		}
	}
	return home.InboxAnswerBody(id, a.Answer, a.AnswerLabel, a.Note, packet, "hub ask "+a.ID)
}
