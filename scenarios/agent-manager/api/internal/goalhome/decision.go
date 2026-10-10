package goalhome

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Operator decisions (DL-5). `agent-manager effort decision raise` renders a
// decision under ## Needs operator and publishes one decision_requested event;
// notification-hub turns it into one phone ask. `effort decision sync` writes
// the answer verbatim into FEEDBACK.md and removes the Needs operator item.
// The tool writes both texts; the orchestrator never hand-writes them.

// DecisionRequestedEventType is the event notification-hub turns into an ask.
const DecisionRequestedEventType = "agent_manager.effort.decision_requested.v1"

// Decision limits mirror the notification-hub ask contract, so a decision
// Agent Manager accepts is never refused by the hub.
const (
	MaxDecisionQuestion       = 600
	MaxDecisionReason         = 300
	MaxDecisionOptions        = 8
	MaxDecisionOptionLabel    = 200
	MinDecisionDefaultDelay   = 12 * time.Hour
	decisionMarkerPrefix      = "[decision "
	decisionAnsweredViaPhone  = "phone"
	decisionAnsweredViaDesk   = "desktop"
	decisionAnsweredByDefault = "default"
)

var (
	decisionIDRule  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
	decisionKeyRule = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$`)
	decisionUrgency = map[string]bool{"": true, "low": true, "normal": true, "high": true, "critical": true}
)

// DecisionOption is one answer: Key is what the orchestrator receives, Label
// what the operator sees.
type DecisionOption struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// Decision is the goal-home view of an operator decision.
type Decision struct {
	ID                   string           `json:"id"`
	Question             string           `json:"question"`
	Options              []DecisionOption `json:"options"`
	Recommended          string           `json:"recommended"`
	RecommendationReason string           `json:"recommendation_reason,omitempty"`
	DefaultAnswer        string           `json:"default_answer,omitempty"`
	Reversible           bool             `json:"reversible,omitempty"`
	Urgency              string           `json:"urgency,omitempty"`
	Note                 string           `json:"note,omitempty"`
	ContextURL           string           `json:"context_url,omitempty"`
	Deadline             time.Time        `json:"deadline,omitempty"`
	RaisedAt             time.Time        `json:"raised_at,omitempty"`
	State                string           `json:"state,omitempty"`
	Answer               string           `json:"answer,omitempty"`
	AnswerLabel          string           `json:"answer_label,omitempty"`
	AnswerNote           string           `json:"answer_note,omitempty"`
	AnsweredVia          string           `json:"answered_via,omitempty"`
	Late                 bool             `json:"late,omitempty"`
	AnsweredAt           time.Time        `json:"answered_at,omitempty"`
	AskID                string           `json:"ask_id,omitempty"`
}

// Validate checks a decision against the notification-hub ask contract. The
// recommended option must exist; a default is allowed only for a reversible
// decision and must be an option.
func (d Decision) Validate() error {
	switch {
	case !decisionIDRule.MatchString(d.ID):
		return fmt.Errorf("decision id %q must be 1-128 letters, digits, '.', '_', ':' or '-'", d.ID)
	case strings.TrimSpace(d.Question) == "" || len(d.Question) > MaxDecisionQuestion:
		return fmt.Errorf("the question is required and at most %d bytes", MaxDecisionQuestion)
	case len(d.Options) < 2 || len(d.Options) > MaxDecisionOptions:
		return fmt.Errorf("a decision needs 2-%d options", MaxDecisionOptions)
	case len(d.RecommendationReason) > MaxDecisionReason:
		return fmt.Errorf("the recommendation reason is at most %d bytes", MaxDecisionReason)
	case !decisionUrgency[d.Urgency]:
		return fmt.Errorf("urgency must be low, normal, high or critical")
	}
	seen := map[string]bool{}
	for _, option := range d.Options {
		if !decisionKeyRule.MatchString(option.Key) {
			return fmt.Errorf("option key %q must be 1-64 letters, digits, '.', '_', ':' or '-'", option.Key)
		}
		if seen[option.Key] {
			return fmt.Errorf("option key %q is repeated", option.Key)
		}
		if len(option.Label) > MaxDecisionOptionLabel {
			return fmt.Errorf("option %q label exceeds %d bytes", option.Key, MaxDecisionOptionLabel)
		}
		seen[option.Key] = true
	}
	if !seen[d.Recommended] {
		return fmt.Errorf("the recommended option %q is not one of the options", d.Recommended)
	}
	if d.DefaultAnswer != "" {
		if !seen[d.DefaultAnswer] {
			return fmt.Errorf("the default %q is not one of the options", d.DefaultAnswer)
		}
		if !d.Reversible {
			return fmt.Errorf("a default is allowed only for a reversible decision (--reversible)")
		}
	}
	return nil
}

// Label returns an option's label, or the key when the option has none.
func (d Decision) Label(key string) string {
	for _, option := range d.Options {
		if option.Key == key && option.Label != "" {
			return option.Label
		}
	}
	return key
}

// RequestPayload is the decision_requested event payload: facts only, the
// shape notification-hub's decision intake reads. Correlation comes back
// verbatim on notification_hub.ask.resolved.v1.
func (d Decision) RequestPayload(subject string, correlation map[string]string) map[string]any {
	options := make([]any, 0, len(d.Options))
	for _, option := range d.Options {
		options = append(options, map[string]any{"key": option.Key, "label": d.Label(option.Key)})
	}
	payload := map[string]any{
		"subject":               subject,
		"question":              d.Question,
		"options":               options,
		"recommended":           d.Recommended,
		"recommendation_reason": d.RecommendationReason,
		"default_answer":        d.DefaultAnswer,
		"reversible":            d.Reversible,
		"urgency":               d.Urgency,
		"context_url":           d.ContextURL,
	}
	if !d.Deadline.IsZero() {
		payload["deadline"] = d.Deadline.UTC().Format(time.RFC3339)
	}
	if len(correlation) > 0 {
		values := make(map[string]any, len(correlation))
		for key, value := range correlation {
			values[key] = value
		}
		payload["correlation"] = values
	}
	return payload
}

func decisionMarker(id string) string { return decisionMarkerPrefix + id + "]" }

// NeedsOperatorItem renders the decision as one ## Needs operator list item.
// Its first line carries the "[decision <id>]" marker that sync removes.
func NeedsOperatorItem(d Decision) string {
	var b strings.Builder
	question := strings.Join(strings.Fields(d.Question), " ")
	fmt.Fprintf(&b, "- %s %s (raised %s; answer arrives in FEEDBACK.md)\n", decisionMarker(d.ID), question, stamp(d.RaisedAt))
	options := make([]string, 0, len(d.Options))
	for _, option := range d.Options {
		text := "`" + option.Key + "` " + d.Label(option.Key)
		if option.Key == d.Recommended {
			text += " (recommended)"
		}
		options = append(options, text)
	}
	fmt.Fprintf(&b, "  - Options: %s\n", strings.Join(options, " · "))
	if d.RecommendationReason != "" {
		fmt.Fprintf(&b, "  - Why: %s\n", strings.Join(strings.Fields(d.RecommendationReason), " "))
	}
	if d.DefaultAnswer != "" {
		fmt.Fprintf(&b, "  - Default: `%s`, applied by notification-hub at the deadline and at least %s after the question reaches the operator\n", d.DefaultAnswer, MinDecisionDefaultDelay)
	}
	return b.String()
}

// AddNeedsOperatorItem returns queue with the decision's item at the end of
// ## Needs operator, creating the section before the first other level-2
// section when it is missing. changed is false when an item with the marker
// already exists. A section with no items (only placeholder prose such as
// "(none)") is replaced; existing items are kept verbatim.
func AddNeedsOperatorItem(queue []byte, d Decision) ([]byte, bool) {
	item := NeedsOperatorItem(d)
	doc := parseDocument(queue)
	section, ok := doc.section("needs operator")
	if !ok {
		at := len(queue)
		if sections := doc.sections(2, 2); len(sections) > 0 {
			at = sections[0].Start
		}
		return splice(queue, at, at, "## Needs operator\n\n"+item), true
	}
	if _, _, found := needsOperatorItemSpan(doc, section, d.ID); found {
		return queue, false
	}
	block := "## " + section.Title + "\n\n"
	if len(section.items()) > 0 {
		block += section.bodyText() + "\n\n"
	}
	return splice(queue, section.Start, section.End, block+item), true
}

// RemoveNeedsOperatorItem removes the decision's item (its marker line and
// the indented lines under it) from ## Needs operator. An item without the
// marker matches when its first line names the decision ID. When the section
// becomes empty it keeps the line "(none)".
func RemoveNeedsOperatorItem(queue []byte, id string) ([]byte, bool) {
	doc := parseDocument(queue)
	section, ok := doc.section("needs operator")
	if !ok {
		return queue, false
	}
	start, end, found := needsOperatorItemSpan(doc, section, id)
	if !found {
		return queue, false
	}
	remaining := strings.TrimSpace(string(queue[section.Start:start]) + string(queue[end:section.End]))
	remaining = strings.TrimSpace(strings.TrimPrefix(remaining, "## "+section.Title))
	block := "## " + section.Title + "\n\n"
	if remaining == "" {
		block += "(none)\n"
	} else {
		block += remaining + "\n"
	}
	return splice(queue, section.Start, section.End, block), true
}

// needsOperatorItemSpan finds the byte span of the decision's list item.
func needsOperatorItemSpan(doc *document, section section, id string) (int, int, bool) {
	marker := decisionMarker(id)
	for i, l := range section.Body {
		m := listItemPattern.FindStringSubmatch(l.Text)
		if m == nil || strings.HasPrefix(l.Text, " ") || strings.HasPrefix(l.Text, "\t") {
			continue
		}
		if !strings.Contains(l.Text, marker) && !namedIn(m[1], id) {
			continue
		}
		end := section.End
		for _, next := range section.Body[i+1:] {
			if strings.TrimSpace(next.Text) == "" {
				continue
			}
			if !strings.HasPrefix(next.Text, " ") && !strings.HasPrefix(next.Text, "\t") {
				end = next.Start
				break
			}
		}
		return l.Start, end, true
	}
	return 0, 0, false
}

// DecisionFeedbackEntry renders the answer as an open operator FEEDBACK.md entry. The
// answer and the operator's note are verbatim.
func DecisionFeedbackEntry(d Decision) string {
	how := "answered via phone"
	switch d.AnsweredVia {
	case decisionAnsweredViaDesk:
		how = "answered on the desktop"
	case decisionAnsweredByDefault:
		how = "applied by default"
	}
	if d.Late {
		how += "; supersedes the default"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "### %s — operator decision (%s, operator — %s)\n\n", d.ID, stamp(d.AnsweredAt), how)
	fmt.Fprintf(&b, "Question: %s\n\n", strings.Join(strings.Fields(d.Question), " "))
	if d.AnsweredVia == decisionAnsweredByDefault {
		fmt.Fprintf(&b, "> %s — %s; applied by default at the deadline with no operator answer\n\n", d.Answer, d.Label(d.Answer))
	} else {
		fmt.Fprintf(&b, "> %s — %s\n", d.Answer, d.Label(d.Answer))
		if note := strings.TrimSpace(d.AnswerNote); note != "" {
			for _, line := range strings.Split(note, "\n") {
				fmt.Fprintf(&b, "> %s\n", line)
			}
		}
		b.WriteString("\n")
	}
	offered := make([]string, 0, len(d.Options))
	for _, option := range d.Options {
		text := option.Key
		var tags []string
		if option.Key == d.Recommended {
			tags = append(tags, "recommended")
		}
		if option.Key == d.DefaultAnswer {
			tags = append(tags, "default")
		}
		if len(tags) > 0 {
			text += " (" + strings.Join(tags, ", ") + ")"
		}
		offered = append(offered, text)
	}
	fmt.Fprintf(&b, "Offered: %s.", strings.Join(offered, " | "))
	if d.AskID != "" {
		fmt.Fprintf(&b, " Ask %s.", d.AskID)
	}
	b.WriteString(" Recorded by `agent-manager effort decision sync`.\n\nStatus: open\n")
	return b.String()
}

// AddFeedbackEntry returns feedback with the decision's entry at the top of
// the first open level-2 section (before "## Resolved"), or before
// "## Resolved" when there is no open section. changed is false when an entry
// for this decision and answer time already exists, so sync is idempotent and
// a later answer that supersedes a default still gets its own entry.
func AddFeedbackEntry(feedback []byte, d Decision) ([]byte, bool) {
	entry := DecisionFeedbackEntry(d)
	doc := parseDocument(feedback)
	prefix := d.ID + " — operator decision (" + stamp(d.AnsweredAt)
	for _, s := range doc.sections(3, 3) {
		if strings.HasPrefix(s.Title, prefix) {
			return feedback, false
		}
	}
	for _, s := range doc.sections(2, 2) {
		if strings.HasPrefix(s.lowerTitle(), "resolved") {
			return splice(feedback, s.Start, s.Start, entry), true
		}
		bodyStart := s.End
		if len(s.Body) > 0 {
			bodyStart = s.Body[0].Start
		}
		for _, l := range s.Body {
			if headingPattern.MatchString(l.Text) {
				bodyStart = l.Start
				break
			}
			bodyStart = l.End
		}
		heading := strings.TrimRight(string(feedback[s.Start:bodyStart]), "\n")
		return splice(feedback, s.Start, bodyStart, heading+"\n\n"+entry), true
	}
	return splice(feedback, len(feedback), len(feedback), entry), true
}

func stamp(at time.Time) string {
	if at.IsZero() {
		return "unknown time"
	}
	return at.UTC().Format(time.RFC3339)
}
