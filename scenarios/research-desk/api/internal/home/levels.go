package home

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

// Levels (docs/agent-system/LEVELS.md). A topic keeps its own open decisions
// in topics/<t>/DECISIONS.md "## Needs you"; the supervisor keeps its own in
// the research home's ATTENTION.md "## Actionable now". An answer to either is
// an `answer` entry in that level's INBOX.md, which the level's next agent
// drains to history/INBOX.md. Nothing about a level item lives with a
// commission, so a commission ending strands nothing.

// Level files.
const (
	DecisionsFile    = "DECISIONS.md"
	DecisionsSection = StateNeedsSection
	InboxFile        = "INBOX.md"
)

// SupervisorOwner owns the research home's own decisions (ATTENTION.md items
// without a `Commission:` line); a topic item's owner is TopicOwner(slug).
const SupervisorOwner = "supervisor"

// TopicOwner is the owner key of a topic's DECISIONS.md items. A commission
// ID never holds a colon, so the two never meet.
func TopicOwner(topic string) string { return "topic:" + topic }

// IsLevelOwner reports whether an owner key names a level (a topic or the
// supervisor) rather than a commission.
func IsLevelOwner(owner string) bool {
	return owner == SupervisorOwner || strings.HasPrefix(owner, "topic:")
}

// InboxAnswerBody is an answer entry's body in a level's inbox
// (research-coordination topic-folder.md): `<ID>: <key> — <label>`, then the
// operator's note, the packet the answer executes and where it came from.
func InboxAnswerBody(id, key, label, note, packet, source string) string {
	key, label, note = strings.TrimSpace(key), strings.TrimSpace(label), strings.TrimSpace(note)
	lines := []string{id + ": " + key}
	if key != "other" {
		if label == "" {
			label = key
		}
		lines[0] += " — " + label
	}
	if note != "" {
		lines = append(lines, "Note: "+strings.Join(strings.Fields(note), " "))
	}
	if packet != "" {
		lines = append(lines, "Packet: "+packet)
	}
	return strings.Join(append(lines, "Source: "+source), "\n")
}

// Level is one level's folder: its decisions (topics only; the supervisor's
// are Snapshot.Attention) and the answers in its inbox and inbox history.
type Level struct {
	// Topic is the topic slug; empty for the supervisor (the research home).
	Topic   string
	Dir     string
	Items   []NeedsYouItem
	Legacy  []Legacy
	Answers []InboxAnswer
	// Unreadable lists the level's decisions or inbox files that exist but
	// cannot be read: its items and answers are unknown, not absent.
	Unreadable []Evidence
}

// Owner is the level's owner key.
func (l *Level) Owner() string {
	if l.Topic == "" {
		return SupervisorOwner
	}
	return TopicOwner(l.Topic)
}

// InboxAnswer is one `### <UTC> · from <sender> · answer` inbox entry.
type InboxAnswer struct {
	Entry
	// Drained is true for an entry in history/INBOX.md: the level's agent
	// has read and acted on it.
	Drained bool
	Source  string
}

// Names reports whether the answer is about item id: a line `<ID>: <key> —
// <label>`, or an older desk entry's "item <ID> of". needs_you.py open uses
// the same rule.
func (a InboxAnswer) Names(id string) bool {
	for _, line := range strings.Split(a.Body, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), id+":") {
			return true
		}
	}
	for rest := a.Body; ; {
		i := strings.Index(rest, "item "+id)
		if i < 0 {
			return false
		}
		rest = rest[i+len("item "+id):]
		if rest == "" || !idByte(rest[0]) {
			return true
		}
	}
}

func idByte(b byte) bool {
	return b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' || b >= '0' && b <= '9' || b == '_' || b == '-'
}

// AnswerTo is the newest answer entry naming id, or nil.
func (l *Level) AnswerTo(id string) *InboxAnswer {
	var newest *InboxAnswer
	for i := range l.Answers {
		if a := &l.Answers[i]; a.Names(id) && (newest == nil || a.At.After(newest.At)) {
			newest = a
		}
	}
	return newest
}

// Holds reports whether the inbox or its history already holds text (an ask
// id), so an answer is never appended twice.
func (l *Level) Holds(text string) bool {
	return slices.ContainsFunc(l.Answers, func(a InboxAnswer) bool { return strings.Contains(a.Body, text) })
}

var inboxHeading = regexp.MustCompile(`^### (\S+) · from (\S+) · (\S+)\s*$`)

// inboxTimes are the stamps inbox.py and hand-written entries use.
var inboxTimes = []string{time.RFC3339, "2006-01-02T15:04Z"}

// parseInboxAnswers reads the answer entries of one INBOX.md.
func parseInboxAnswers(text, source string, drained bool) []InboxAnswer {
	var out []InboxAnswer
	var current *InboxAnswer
	var body []string
	flush := func() {
		if current != nil {
			current.Body = strings.TrimSpace(strings.Join(body, "\n"))
			out = append(out, *current)
		}
		current, body = nil, nil
	}
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if !strings.HasPrefix(line, "### ") {
			body = append(body, line)
			continue
		}
		flush()
		m := inboxHeading.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil || m[3] != EntryAnswer {
			continue
		}
		for _, layout := range inboxTimes {
			if at, err := time.Parse(layout, m[1]); err == nil {
				current = &InboxAnswer{Entry: Entry{At: at, Kind: EntryAnswer, Via: m[2]}, Drained: drained, Source: source}
				break
			}
		}
	}
	flush()
	return out
}

// readLevel reads one level folder: its DECISIONS.md items when it is a
// topic, and its inbox answers.
func readLevel(dir, topic string) *Level {
	l := &Level{Topic: topic, Dir: dir}
	if topic != "" {
		path := filepath.Join(dir, DecisionsFile)
		var err error
		if l.Items, l.Legacy, err = readNeedsYou(path, DecisionsSection); err != nil {
			l.Unreadable = append(l.Unreadable, Unreadable(path, err))
		}
	}
	for _, f := range []struct {
		path    string
		drained bool
	}{{filepath.Join(dir, InboxFile), false}, {filepath.Join(dir, "history", InboxFile), true}} {
		data, err := os.ReadFile(f.path)
		switch {
		case absent(err):
		case err != nil:
			l.Unreadable = append(l.Unreadable, Unreadable(f.path, err))
		default:
			l.Answers = append(l.Answers, parseInboxAnswers(string(data), f.path, f.drained)...)
		}
	}
	return l
}

// readLevels reads the supervisor's level and every topic's.
func readLevels(researchDir string, topics []*Topic) []*Level {
	out := []*Level{readLevel(researchDir, "")}
	for _, t := range topics {
		out = append(out, readLevel(t.Dir, t.Slug))
	}
	return out
}

// Level returns the level an owner key names, or nil.
func (s *Snapshot) Level(owner string) *Level {
	for _, l := range s.Levels {
		if l.Owner() == owner {
			return l
		}
	}
	return nil
}

// AnswerWaiting is an operator's answer still in a level's INBOX.md: an approved decision nothing has acted on.
type AnswerWaiting struct {
	Owner  string
	ItemID string
	// Answer is the chosen option key, empty when the entry does not parse.
	Answer string
	At     time.Time
	Inbox  string
	// LiveWindowReads is true while a manager window is live on the topic: it drains the inbox at its next review.
	LiveWindowReads bool
}

var answerLine = regexp.MustCompile(`^([A-Z0-9][A-Z0-9_-]{0,127}): ([a-z0-9-]{1,32})\b`)

// AnswersWaiting lists the undrained answer entries of every level, oldest first. An answered item leaves the
// open list, so without this an answer that no window reads is invisible (seven sat up to five days, 2026-11-20).
func (s *Snapshot) AnswersWaiting() []AnswerWaiting {
	var out []AnswerWaiting
	for _, l := range s.Levels {
		live := false
		if t := s.Topic(l.Topic); l.Topic != "" && t != nil {
			live = len(t.Live) > 0
		}
		for _, a := range l.Answers {
			if a.Drained {
				continue
			}
			w := AnswerWaiting{Owner: l.Owner(), At: a.At, Inbox: a.Source, LiveWindowReads: live}
			for _, line := range strings.Split(a.Body, "\n") {
				if m := answerLine.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
					w.ItemID, w.Answer = m[1], m[2]
					break
				}
			}
			out = append(out, w)
		}
	}
	slices.SortFunc(out, func(a, b AnswerWaiting) int { return a.At.Compare(b.At) })
	return out
}
