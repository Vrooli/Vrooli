package runsignal

import (
	"regexp"
	"sort"
	"strings"

	"agent-manager/internal/domain"
)

// waitIdentity names what a wait waits for: the wait command and the job or
// cell it names. Waits on different jobs are not one loop.
func waitIdentity(fact InvocationFact, byID map[string]*domain.RunEvent) string {
	if fact.Capability != "wait" {
		return ""
	}
	target := ""
	if call := toolCallData(byID[fact.CallEventID]); call != nil {
		for _, key := range []string{"cell_id", "session_id", "job_id", "run_id", "task_id", "shell_id", "id"} {
			if value, ok := call.Input[key].(string); ok && value != "" {
				target = value
				break
			}
		}
	}
	if target == "" {
		if tokens := factTokens(fact, byID); len(tokens) > 1 && !isFlag(tokens[len(tokens)-1]) {
			target = tokens[len(tokens)-1]
		}
	}
	name := fact.CommandPath
	if name == "" {
		name = "wait:" + fact.ToolName
	}
	return name + "|" + target
}

// blockingWait is a wait that is already the one blocking wait: a designated
// command, or the native wait tool asked to yield for a minute or more.
func blockingWait(fact InvocationFact, byID map[string]*domain.RunEvent) bool {
	if fact.CommandPath == "test-genie runs wait" || strings.HasSuffix(fact.CommandPath, " validate wait") {
		return true
	}
	call := toolCallData(byID[fact.CallEventID])
	if call == nil || fact.CommandPath != "" {
		return false
	}
	for key, unit := range map[string]float64{"yield_time_ms": 0.001, "timeout_ms": 0.001, "yield_ms": 0.001, "seconds": 1, "timeout": 1, "timeout_seconds": 1} {
		if value, ok := call.Input[key].(float64); ok && value*unit >= 60 {
			return true
		}
		if value, ok := call.Input[key].(int); ok && float64(value)*unit >= 60 {
			return true
		}
	}
	return false
}

// detectWaitMisuse finds three or more consecutive short waits on one job:
// any other call between them, a wait on another job or a break in the work
// ends the run, and a blocking wait is never misuse.
func detectWaitMisuse(ctx EpisodeDetectorContext) []FrictionEpisode {
	var out []FrictionEpisode
	work := newWorkSpans(ctx.Events)
	for start := 0; start < len(ctx.Facts); {
		key := waitIdentity(ctx.Facts[start], ctx.EventsByID)
		if key == "" || blockingWait(ctx.Facts[start], ctx.EventsByID) {
			start++
			continue
		}
		end := start + 1
		for end < len(ctx.Facts) && waitIdentity(ctx.Facts[end], ctx.EventsByID) == key && work.sameStretch(ctx.Facts[start].CallEventID, ctx.Facts[end].CallEventID) {
			end++
		}
		if end-start >= 3 {
			e := newEpisode("wait-misuse", ctx.Facts[start], ctx.Facts[end-1], ctx.EventsByID, ctx.Events)
			e.price(ctx.Facts[start:end], ctx.EventsByID)
			e.CycleCount = end - start
			e.RepeatedElement = key
			e.Severity = "repeated"
			out = append(out, e)
			start = end
			continue
		}
		start++
	}
	return out
}

func detectBlockedThenAbandoned(ctx EpisodeDetectorContext) []FrictionEpisode {
	var out []FrictionEpisode
	for _, span := range ctx.SelfReports {
		if span.RuleID != "blocked" {
			continue
		}
		blockedAt := ctx.EventsByID[span.EventID]
		if blockedAt == nil {
			continue
		}
		structuralFailure := false
		successAfter := false
		for _, fact := range ctx.Facts {
			if event := ctx.EventsByID[fact.CallEventID]; event != nil && event.Timestamp.After(blockedAt.Timestamp) && fact.Outcome == "success" {
				successAfter = true
				break
			}
			if event := ctx.EventsByID[fact.CallEventID]; event != nil && event.Timestamp.Before(blockedAt.Timestamp) && fact.Outcome == "failure" {
				structuralFailure = true
			}
		}
		if successAfter || !structuralFailure {
			continue
		}
		fact := InvocationFact{CallEventID: span.EventID, ResultEventID: span.EventID}
		e := newEpisode("blocked-then-abandoned", fact, fact, ctx.EventsByID, ctx.Events)
		e.EvidenceEventIDs = []string{span.EventID}
		e.Turns = 1
		out = append(out, e)
	}
	return out
}

// correctionPattern is an operator turning the agent back about what it just
// did: the agent's output or action is called wrong ("that's not", "wrong
// file"), the operator asks why it did something, restates what they said or
// wanted, names what it should have done, or tells it not to do that again.
// Approvals ("no problem"), plain instructions ("stop here and commit", "you
// need to run X") and bare openers like "no" or "stop" are not corrections.
var correctionPattern = regexp.MustCompile(`(?i)\bthat('s| is| was| isn't| wasn't)\s+(not|wrong|incorrect|a mistake)\b|\b(that|this|it) (isn't|wasn't|is not|was not) (right|correct|what)|\bnot (the right|what i)\b|\bwhy (the \w+ )?(did|do|would|are|were) you\b|\bi (said|told you|asked you|already said|just said)\b|\byou (should have|should've|shouldn't have|were supposed to|got (that|it) wrong)\b|\b(don'?t|do not|never) (do|run|use|touch|change|delete|remove|edit|modify|revert|undo|create|add|write|call|make) (that|this|it)\b|\b(don'?t|do not|never) (file|commit|push|restart|kill)\b|\bcorrection:|\bdumb (this|it) down\b|\bi (don't|do not) understand\b|\b(wrong|incorrect) (file|approach|way|command|path|answer|one|thing|fix|direction)\b|(^|[.!?\n]\s*)(no|nope|wrong)[,.!]?\s+(that|this|it|you|not|don'?t|wrong)\b`)

// isCorrection says whether operator text corrects the agent's course.
func isCorrection(text string) bool { return correctionPattern.MatchString(text) }

// detectGuidanceRepair finds an operator message that corrects an agent which
// had already acted and is followed by more agent work. Only operator-sourced
// messages count: injected context, wakes, notifications and teammates are not
// the operator.
func detectGuidanceRepair(ctx EpisodeDetectorContext) []FrictionEpisode {
	// Facts by event order: pos[i] is the event index of the call of facts[i].
	eventIndex := make(map[string]int, len(ctx.Events))
	for i, event := range ctx.Events {
		eventIndex[event.ID.String()] = i
	}
	type placed struct{ at, fact int }
	var facts []placed
	for j := range ctx.Facts {
		if at, ok := eventIndex[ctx.Facts[j].CallEventID]; ok {
			facts = append(facts, placed{at, j})
		}
	}
	sort.SliceStable(facts, func(a, b int) bool { return facts[a].at < facts[b].at })

	var out []FrictionEpisode
	acted := false
	for i, event := range ctx.Events {
		switch data := event.Data.(type) {
		case *domain.ToolCallEventData:
			acted = true
		case *domain.MessageEventData:
			if strings.EqualFold(data.Role, "assistant") {
				acted = true
				continue
			}
			if !acted || !strings.EqualFold(data.Role, "user") || domain.MessageSource(event) != domain.MessageSourceOperator || !isCorrection(data.Content) {
				continue
			}
			if !agentWorkedAfter(ctx.Events[i+1:]) {
				continue
			}
			split := sort.Search(len(facts), func(k int) bool { return facts[k].at > i })
			start, end := InvocationFact{CallEventID: event.ID.String(), ResultEventID: event.ID.String()}, InvocationFact{CallEventID: event.ID.String(), ResultEventID: event.ID.String()}
			if split > 0 {
				start = ctx.Facts[facts[split-1].fact]
			}
			if split < len(facts) {
				end = ctx.Facts[facts[split].fact]
			}
			e := newEpisode("guidance-repair", start, end, ctx.EventsByID, ctx.Events)
			e.StartEventID, e.EvidenceEventIDs = event.ID.String(), []string{event.ID.String(), start.CallEventID, end.CallEventID}
			e.EpisodeID = fingerprint("guidance-repair", event.ID.String())
			out = append(out, e)
		}
	}
	return out
}

func agentWorkedAfter(events []*domain.RunEvent) bool {
	for _, event := range events {
		switch data := event.Data.(type) {
		case *domain.ToolCallEventData:
			return true
		case *domain.MessageEventData:
			if strings.EqualFold(data.Role, "assistant") && !data.EvidenceOnly {
				return true
			}
		}
	}
	return false
}

// goalMode is a run that has a goal or is driven by injected continuations:
// its work after an answer is the goal's, not the agent's own.
func goalMode(events []*domain.RunEvent) bool {
	for _, event := range events {
		if event.EventType == domain.EventTypeGoalStatusChanged {
			return true
		}
		if message, ok := event.Data.(*domain.MessageEventData); ok && strings.EqualFold(message.Role, "user") && domain.MessageSource(event) == domain.MessageSourceGoalContinuation {
			return true
		}
	}
	return false
}

// detectHandoffContinuation finds work after the agent declared its answer
// terminal without new input: the tool calls after a terminal assistant
// message and before the next user message. It never fires in a goal-mode run,
// nor when the next user message is an injected one (a wake, a notification, a
// teammate, injected context): that work answers the injection. It is
// informational, not priced.
func detectHandoffContinuation(ctx EpisodeDetectorContext) []FrictionEpisode {
	if ctx.GoalRun || goalMode(ctx.Events) {
		return nil
	}
	var out []FrictionEpisode
	for i, event := range ctx.Events {
		message, ok := event.Data.(*domain.MessageEventData)
		if !ok || !message.Terminal || !strings.EqualFold(message.Role, "assistant") {
			continue
		}
		var until *domain.RunEvent
		for _, later := range ctx.Events[i+1:] {
			if next, ok := later.Data.(*domain.MessageEventData); ok && strings.EqualFold(next.Role, "user") {
				until = later
				break
			}
		}
		if until != nil && domain.MessageSource(until) != domain.MessageSourceOperator {
			continue
		}
		var first, last *InvocationFact
		turns := 0
		for j := range ctx.Facts {
			factEvent := ctx.EventsByID[ctx.Facts[j].CallEventID]
			if factEvent == nil || !factEvent.Timestamp.After(event.Timestamp) || (until != nil && !factEvent.Timestamp.Before(until.Timestamp)) {
				continue
			}
			if first == nil {
				first = &ctx.Facts[j]
			}
			last = &ctx.Facts[j]
			turns++
		}
		if first == nil {
			continue
		}
		e := newEpisode("handoff-continuation", *first, *last, ctx.EventsByID, ctx.Events)
		e.EvidenceEventIDs = []string{event.ID.String(), first.CallEventID, last.CallEventID}
		e.Turns = turns
		out = append(out, e)
	}
	return out
}
