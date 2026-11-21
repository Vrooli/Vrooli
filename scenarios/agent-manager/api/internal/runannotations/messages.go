package runannotations

import (
	"context"
	"encoding/json"
	"sort"
	"time"
)

// Message scope. A message-scope label is answered once per human operator
// message, so mood can be read as a series over time instead of a single value
// per run. The subject is the message (its event id is the scope id, its
// timestamp is stored with the row) and the state a model reads is that message
// plus the agent text just before it. Only text a person wrote is a subject:
// workflow briefs and harness text never are.

// DefaultMessagesPerRun bounds how many human messages of one run are answered
// in a pass. The newest are kept; a long conversation is read in pieces across
// passes only if the bound is raised, never silently sampled.
const DefaultMessagesPerRun = 24

// messageDigest is the bounded state a decision model reads for one message.
type messageDigest struct {
	RunID         string          `json:"run_id"`
	Workload      string          `json:"workload,omitempty"`
	Message       messageDigestAt `json:"message"`
	PreviousAgent string          `json:"previous_agent,omitempty"`
}

type messageDigestAt struct {
	Index int    `json:"index"`
	Of    int    `json:"of"`
	At    string `json:"at,omitempty"`
	Text  string `json:"text"`
}

// messageCap bounds the message text and the agent text before it.
type messageCap struct{ text, agent int }

// messageCapLadder shrinks a message state deterministically. The message text
// is kept longest: it is the evidence the label judges.
var messageCapLadder = []messageCap{{900, 320}, {600, 200}, {400, 120}, {250, 0}, {120, 0}}

// reviewMessageLadder starts wider, so a person judging a gold row reads the
// whole message (a message is read up to 2000 characters) and then shrinks like
// the model's state.
var reviewMessageLadder = append([]messageCap{{2000, 800}, {1400, 500}}, messageCapLadder...)

// BuildMessageDigest renders one human message and the agent text before it as
// valid JSON no larger than max characters.
func BuildMessageDigest(facts RunFacts, message MessageFact, index, total, max int) (string, error) {
	if max <= 0 {
		max = DefaultDigestLimit
	}
	return renderMessageDigest(facts, message, index, total, max, messageCapLadder)
}

// BuildMessageReviewDigest renders the state a reviewer judges one message from:
// the same fields as the model's, with wider caps, and never the stored answer.
func BuildMessageReviewDigest(facts RunFacts, message MessageFact, index, total, max int) (string, error) {
	if max <= 0 {
		max = DefaultReviewDigestLimit
	}
	return renderMessageDigest(facts, message, index, total, max, reviewMessageLadder)
}

func renderMessageDigest(facts RunFacts, message MessageFact, index, total, max int, ladder []messageCap) (string, error) {
	previous := previousAgentText(facts.AgentText, message.At)
	for _, caps := range ladder {
		digest := messageDigest{
			RunID: facts.RunID, Workload: facts.Workload.Kind,
			Message:       messageDigestAt{Index: index, Of: total, Text: BoundText(message.Text, caps.text)},
			PreviousAgent: BoundText(previous, caps.agent),
		}
		if !message.At.IsZero() {
			digest.Message.At = message.At.UTC().Format(time.RFC3339)
		}
		body, err := json.Marshal(digest)
		if err != nil {
			return "", err
		}
		if len(body) <= max {
			return string(body), nil
		}
	}
	minimal, err := json.Marshal(messageDigest{RunID: facts.RunID, Message: messageDigestAt{Index: index, Of: total, Text: BoundText(message.Text, 60)}})
	if err != nil {
		return "", err
	}
	return string(minimal), nil
}

// previousAgentText is the last agent message written before the operator
// message, or "" when the agent had said nothing yet or its text was not read.
func previousAgentText(agent []MessageFact, before time.Time) string {
	if before.IsZero() {
		return ""
	}
	for index := len(agent) - 1; index >= 0; index-- {
		if !agent[index].At.IsZero() && !agent[index].At.After(before) {
			return agent[index].Text
		}
	}
	return ""
}

// messageWorks builds one unit of work per human operator message of a run when
// the pack declares an active message-scope label. The units share the run's
// facts, so they cost no database reads.
func (s *Service) messageWorks(ctx context.Context, run *runWork, opts AnnotateOptions) []*runWork {
	if !s.hasMessageLabels(opts) || len(run.facts.HumanMessages) == 0 {
		return nil
	}
	total := len(run.facts.HumanMessages)
	type subject struct {
		index   int
		message MessageFact
	}
	var subjects []subject
	if len(opts.MessageIDs) > 0 {
		// Named messages are answered wherever they sit in the run: a gold set
		// names the messages it judges, and the newest-messages bound is for
		// passes that were not told which ones matter.
		named := map[string]bool{}
		for _, id := range opts.MessageIDs {
			named[id] = true
		}
		for position, message := range run.facts.HumanMessages {
			if named[message.EventID] {
				subjects = append(subjects, subject{position + 1, message})
			}
		}
	} else {
		start := max(0, total-s.messagesPerRun())
		for position := start; position < total; position++ {
			subjects = append(subjects, subject{position + 1, run.facts.HumanMessages[position]})
		}
	}
	works := make([]*runWork, 0, len(subjects))
	for _, entry := range subjects {
		index, message := entry.index, entry.message
		facts := run.facts
		build := func(limit int, _ []string) (string, error) {
			return BuildMessageDigest(facts, message, index, total, limit)
		}
		digest, err := build(s.digestLimit, nil)
		if err != nil {
			continue
		}
		features := run.features
		features.MessageText, features.MessageIndex, features.MessageCount = message.Text, index, total
		work := &runWork{
			scopeKind: ScopeMessage, scopeID: message.EventID, runID: run.runID, occurredAt: message.At,
			pass: run.pass, parent: run, force: run.force, facts: facts, features: features, digest: digest, build: build,
			evidenceHash: evidenceFingerprint(digest, RunFacts{}), subjectEvidence: []string{message.EventID},
			activation:   features.Activation(),
			observations: map[string]Annotation{}, evidence: map[string][]string{},
		}
		s.prepareLabels(ctx, work, opts)
		works = append(works, work)
		run.children = append(run.children, work)
	}
	return works
}

// messageReviewDigest is the review packet of one human message of a run, or ""
// when the run no longer has that message as human text.
func (s *Service) messageReviewDigest(facts RunFacts, messageID string) string {
	total := len(facts.HumanMessages)
	for position, message := range facts.HumanMessages {
		if message.EventID != messageID {
			continue
		}
		digest, err := BuildMessageReviewDigest(facts, message, position+1, total, DefaultReviewDigestLimit)
		if err != nil {
			return ""
		}
		return digest
	}
	return ""
}

// hasMessageLabels reports whether any message-scope label would be evaluated
// under these options.
func (s *Service) hasMessageLabels(opts AnnotateOptions) bool {
	allowed := s.pack.requested(opts.Labels)
	for _, label := range s.pack.Pack.Labels {
		if label.Scope != ScopeMessage || (label.Optional && !opts.IncludeOptional) {
			continue
		}
		if allowed == nil || allowed[label.ID] {
			return true
		}
	}
	return false
}

func (s *Service) messagesPerRun() int {
	if limit := s.pack.Pack.Runtime.Budgets.MaxMessagesPerRun; limit > 0 {
		return limit
	}
	return DefaultMessagesPerRun
}

// MessageView is one human message's labels, reported under its run.
type MessageView struct {
	MessageID string           `json:"message_id"`
	At        *time.Time       `json:"at,omitempty"`
	Labels    []AnnotationView `json:"labels"`
}

func (s *Service) messageView(work *runWork) MessageView {
	view := MessageView{MessageID: work.scopeID}
	if !work.occurredAt.IsZero() {
		at := work.occurredAt
		view.At = &at
	}
	for _, labelID := range s.pack.Order() {
		if observation, ok := work.observations[labelID]; ok {
			view.Labels = append(view.Labels, s.annotationView(observation))
		}
	}
	return view
}

// messageViewsFromRows groups stored message rows into per-message views,
// oldest message first.
func (s *Service) messageViewsFromRows(rows []Annotation) []MessageView {
	byMessage := map[string]*MessageView{}
	var order []string
	for _, row := range rows {
		view := byMessage[row.ScopeID]
		if view == nil {
			view = &MessageView{MessageID: row.ScopeID}
			if !row.OccurredAt.IsZero() {
				at := row.OccurredAt
				view.At = &at
			}
			byMessage[row.ScopeID] = view
			order = append(order, row.ScopeID)
		}
		view.Labels = append(view.Labels, s.annotationView(row))
	}
	sort.SliceStable(order, func(i, j int) bool {
		left, right := byMessage[order[i]].At, byMessage[order[j]].At
		if left == nil || right == nil {
			return left != nil
		}
		return left.Before(*right)
	})
	out := make([]MessageView, 0, len(order))
	for _, id := range order {
		out = append(out, *byMessage[id])
	}
	return out
}
