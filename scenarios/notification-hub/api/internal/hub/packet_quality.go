package hub

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// The decision-quality rules of the packet check, beyond shape: does the page
// add anything (page_restates.go), can the operator read it
// (page_scheme.go), is a command the operator must run shown, and has this
// come up before with no way offered to stop it. Three of them refuse an
// agent's ask (refusal below); a scenario's code-built decision only gets
// advice, because no agent stands behind it to fix and resend.

// Codes of the decision-quality rules.
const (
	// WarnCommandHidden: an answer asks the operator to run something and
	// neither the page nor the cover shows the command.
	WarnCommandHidden = "command_hidden"
	// WarnRecurringSubject: the third ask from one sender on a similar subject
	// within RecurrenceWindow, and no option stops the repeats.
	WarnRecurringSubject = "recurring_subject"
)

// WarningBlocking is the severity of a finding that refuses an agent's ask.
// A dry run reports it so the agent fixes it before it sends.
const WarningBlocking = "blocking"

// blockingCodes are the findings that refuse an agent's ask: a recommendation
// with no reason, a page that only restates the cover, and a page the
// operator cannot read.
var blockingCodes = map[string]bool{
	WarnMissingRecommendReason: true,
	WarnExplainerRestates:      true,
	WarnPageUnreadable:         true,
}

// ErrAskRefused is returned for an agent's ask that a blocking finding
// refuses. Nothing is stored and nothing is sent; the agent fixes the named
// parts and sends again. The error is a *RefusedError.
var ErrAskRefused = errors.New("ask refused by the decision-quality check")

// RefusedError lists the findings that refused an ask.
type RefusedError struct{ Findings []PacketWarning }

func (e *RefusedError) Error() string {
	lines := make([]string, 0, len(e.Findings))
	for _, finding := range e.Findings {
		lines = append(lines, finding.Code+" ("+finding.Field+"): "+finding.Message)
	}
	return "refused, nothing was sent: " + strings.Join(lines, " | ") +
		" | Fix each one, check with --dry-run, then send again."
}

func (e *RefusedError) Unwrap() error { return ErrAskRefused }

// markBlocking gives the blocking findings of an agent's ask the blocking
// severity. A finding carried over from the open ask, about a part this
// revision did not touch, stays advice: a revision is never refused for what
// it left alone.
func markBlocking(spec AskSpec, warnings []PacketWarning) []PacketWarning {
	if !spec.AgentSent {
		return warnings
	}
	for i := range warnings {
		if blockingCodes[warnings[i].Code] && !warnings[i].CarriedOver && warnings[i].Severity == WarningAdvice {
			warnings[i].Severity = WarningBlocking
		}
	}
	return warnings
}

// refusal is the error for warnings that hold a blocking finding, nil otherwise.
func refusal(warnings []PacketWarning) error {
	var blocking []PacketWarning
	for _, warning := range warnings {
		if warning.Severity == WarningBlocking {
			blocking = append(blocking, warning)
		}
	}
	if len(blocking) == 0 {
		return nil
	}
	return &RefusedError{Findings: blocking}
}

// --- command_hidden ----------------------------------------------------------

var (
	// operatorRunsPattern finds an answer that has the operator run something,
	// anywhere on the cover.
	operatorRunsPattern = regexp.MustCompile(`(?i)\b(?:your|a|the) terminal\b|\bcommand line\b|\byou(?:'ll| will| need to| must| can| should| would)? run\b|\brun (?:it |this |that |them |one command |a command |the command )?yourself\b|\bcommands? (?:for you )?to run\b|\byou will be given (?:one|a) command\b|\bpaste (?:it|this|the)\b|\bsudo\b`)
	// operatorVoicePattern is the operator's own voice in an option label.
	operatorVoicePattern = regexp.MustCompile(`(?i)\bI(?:'ll| will)? run\b`)
	// runQuestionPattern is a question that asks the operator to run something.
	runQuestionPattern = regexp.MustCompile(`(?i)^\s*(?:will you |can you |could you |would you |please )?run\b`)
	backtickPattern    = regexp.MustCompile("`([^`]+)`")
)

// runCue is the cover text that asks the operator to run something, "" when
// none does.
func runCue(spec AskSpec) string {
	if runQuestionPattern.MatchString(spec.Question) {
		return spec.Question
	}
	for _, text := range []string{spec.Question, spec.BigPicture, spec.RecommendationReason} {
		if operatorRunsPattern.MatchString(text) {
			return text
		}
	}
	for _, option := range spec.Options {
		if operatorRunsPattern.MatchString(option.Label) || operatorVoicePattern.MatchString(option.Label) {
			return option.Label
		}
	}
	return ""
}

// looksLikeCommand is text set as code that reads as a command line: at
// least two words, one of them with a letter.
func looksLikeCommand(text string) bool {
	fields := strings.Fields(text)
	return len(fields) >= 2 && strings.ContainsAny(strings.ToLower(text), "abcdefghijklmnopqrstuvwxyz")
}

// pageShowsCommand reports a command on the page: a kit command block, or a
// pre block or a code, kbd or samp line of its own that reads as a command
// line. A command inside a sentence does not count: the operator has to find
// it, cannot copy it alone, and is not told where it runs or how to undo it.
func pageShowsCommand(root *element) bool {
	found := false
	var walk func(e *element)
	walk = func(e *element) {
		if found {
			return
		}
		if e.Classes["hub-command"] {
			found = true
			return
		}
		switch e.Tag {
		case "pre", "code", "kbd", "samp":
			if (e.Tag == "pre" || standsAlone(e)) && looksLikeCommand(allText(e)) {
				found = true
				return
			}
		}
		for _, child := range e.Children {
			walk(child)
		}
	}
	walk(root)
	return found
}

// standsAlone reports an element that is the whole of its parent block, not
// a phrase in a sentence.
func standsAlone(e *element) bool {
	return e.Parent != nil && e.Parent.Text == "" && len(e.Parent.Children) == 1
}

func allText(e *element) string {
	parts := []string{e.Text}
	for _, child := range e.Children {
		parts = append(parts, allText(child))
	}
	return strings.Join(parts, " ")
}

func coverShowsCommand(spec AskSpec) bool {
	for _, text := range []string{spec.Question, spec.BigPicture} {
		for _, match := range backtickPattern.FindAllStringSubmatch(text, -1) {
			if looksLikeCommand(match[1]) {
				return true
			}
		}
	}
	return false
}

// checkCommandShown advises an ask whose answer has the operator run
// something that the page and the cover never show. Technical details do not
// count: the phone does not show them unless asked.
func checkCommandShown(spec AskSpec, page *element) []PacketWarning {
	cue := runCue(spec)
	if cue == "" || coverShowsCommand(spec) || page != nil && pageShowsCommand(page) {
		return nil
	}
	return []PacketWarning{advice(WarnCommandHidden, "explainer", fmt.Sprintf(
		"an answer has the operator run something (%q), but the page shows no command. Put the exact command on the page's first screen "+
			`with the kit's command block, <section class="hub-command">: where to run it, the command with a Copy button, what it shows before it changes anything, `+
			"and how to undo it (operatordecision.Page.Command does this). --technical-details is not shown on the phone.", shorten(cue, 90)))}
}

// --- recurring_subject ----------------------------------------------------------

// RecurrenceWindow is how far back the recurring-subject rule looks.
const RecurrenceWindow = 14 * 24 * time.Hour

// recurrenceThreshold is the earlier similar asks that make this one a repeat
// to fix at its cause (this ask is the third).
const recurrenceThreshold = 2

var (
	stopRepeatsPattern = regexp.MustCompile(`(?i)stop asking|stop the (?:daily|weekly|repeated)|don'?t ask|do not ask|no longer ask|without asking|from now on|standing|\balways\b|every time|automatic|by default|\brules?\b|\bpolic(?:y|ies)\b|until (?:i|you) (?:revoke|say|change|turn)|permanent|fix the cause|root cause`)
	numberPattern      = regexp.MustCompile(`\d+(?:[.,]\d+)*`)
)

// priorAsk is an earlier ask the recurring-subject rule compares with.
type priorAsk struct {
	id, question, subjectKey string
	createdAt                time.Time
}

// subjectWords are a question's content words with every number made alike,
// so "about 5 days" and "about 6 days" are the same subject.
func subjectWords(question string) map[string]bool {
	words := map[string]bool{}
	for _, word := range contentWords(numberPattern.ReplaceAllString(question, "#")) {
		words[word] = true
	}
	return words
}

func similarSubject(a, b map[string]bool) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	shared := 0
	for word := range a {
		if b[word] {
			shared++
		}
	}
	return float64(shared)/float64(len(a)+len(b)-shared) >= 0.5
}

// checkRecurrence advises the third ask from one sender on a similar subject
// within the window when no option ends the repeats. history holds the
// sender's earlier asks in the window, oldest first.
func checkRecurrence(spec AskSpec, sender string, history []priorAsk) []PacketWarning {
	for _, option := range spec.Options {
		if stopRepeatsPattern.MatchString(option.Label) {
			return nil
		}
	}
	words := subjectWords(spec.Question)
	var similar []priorAsk
	for _, prior := range history {
		if spec.SubjectKey != "" && prior.subjectKey == spec.SubjectKey || similarSubject(words, subjectWords(prior.question)) {
			similar = append(similar, prior)
		}
	}
	if len(similar) < recurrenceThreshold {
		return nil
	}
	var dates []string
	for _, prior := range similar {
		dates = append(dates, prior.createdAt.UTC().Format("2 Jan"))
	}
	return []PacketWarning{advice(WarnRecurringSubject, "options", fmt.Sprintf(
		"this is ask %d from %s on this subject in %d days (before: %s), and no option stops the repeats. Name the cause, and add one option that fixes it: "+
			"a standing rule (\"Yes, and do this from now on without asking\"), a policy change, or a repair. Show the history with the kit's Recurrence block "+
			`(<aside class="hub-recurrence">, operatordecision.Page.Recurrence).`,
		len(similar)+1, sender, int(RecurrenceWindow/(24*time.Hour)), strings.Join(dates, ", ")))}
}

// recentAsks reads the sender's asks from the window, oldest first, leaving
// out except (the ask being revised or just stored) and asks a replacement
// superseded (the same decision with a changed menu).
func (s *Service) recentAsks(ctx context.Context, source, sender, except string, now time.Time) ([]priorAsk, error) {
	since := storedTime(now.Add(-RecurrenceWindow))
	rows, err := s.db.QueryContext(ctx, `SELECT id, question, subject_key, created_at FROM asks
		WHERE source = ? AND sender = ? AND created_at >= ? AND id <> ? AND superseded_by_ask_id = ''
		ORDER BY created_at`, source, sender, since, except)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []priorAsk
	for rows.Next() {
		var prior priorAsk
		var created string
		if err := rows.Scan(&prior.id, &prior.question, &prior.subjectKey, &created); err != nil {
			return nil, err
		}
		prior.createdAt, _ = time.Parse(time.RFC3339Nano, created)
		out = append(out, prior)
	}
	return out, rows.Err()
}

// checkRecurring runs the recurring-subject rule for a sealed ask. A history
// the Hub cannot read gives no advice; it never fails the ask.
func (s *Service) checkRecurring(ctx context.Context, spec AskSpec, sender, except string) []PacketWarning {
	history, err := s.recentAsks(ctx, spec.Source, sender, except, s.clock.Now())
	if err != nil {
		s.log.Printf("packet check: recent asks of %s: %v", spec.Source, err)
		return nil
	}
	return checkRecurrence(spec, sender, history)
}
