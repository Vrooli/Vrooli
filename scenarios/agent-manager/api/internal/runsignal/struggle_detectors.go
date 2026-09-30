package runsignal

import (
	"strings"

	"agent-manager/internal/domain"
)

func detectWaitMisuse(ctx EpisodeDetectorContext) []FrictionEpisode {
	var out []FrictionEpisode
	for start := 0; start < len(ctx.Facts); {
		startKey := waitIdentity(ctx.Facts[start])
		if startKey == "" {
			start++
			continue
		}
		end := start + 1
		waitCount := 1
		lastWait := start
		for end < len(ctx.Facts) {
			fact := ctx.Facts[end]
			if fact.Capability == "wait" {
				if waitIdentity(fact) != startKey {
					break
				}
				waitCount++
				lastWait = end
			}
			end++
		}
		if waitCount >= 3 {
			e := newEpisode("wait-misuse", ctx.Facts[start], ctx.Facts[lastWait], ctx.EventsByID, ctx.Events)
			e.Turns = waitCount
			e.CycleCount = waitCount
			e.RepeatedElement = startKey
			e.Severity = "repeated"
			out = append(out, e)
			start = lastWait + 1
			continue
		}
		start++
	}
	return out
}

// waitIdentity deliberately does not use InvocationFact.Fingerprint. A wait
// call is a tool capability, not a command invocation, so its command
// fingerprint is correctly empty. The capability and redacted intent shape
// still provide a stable identity for detecting repeated polling.
func waitIdentity(fact InvocationFact) string {
	if fact.Capability != "wait" {
		return ""
	}
	if fact.IntentClass != "" {
		return fact.IntentClass
	}
	if fact.ToolName != "" {
		return "wait:" + fact.ToolName
	}
	return "wait"
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

func detectGuidanceRepair(ctx EpisodeDetectorContext) []FrictionEpisode {
	var out []FrictionEpisode
	for _, correction := range ctx.SelfReports {
		if !correction.OperatorCorrection {
			continue
		}
		operatorEvent := ctx.EventsByID[correction.EventID]
		if operatorEvent == nil {
			continue
		}
		var before, after *InvocationFact
		for i := range ctx.Facts {
			event := ctx.EventsByID[ctx.Facts[i].CallEventID]
			if event == nil {
				continue
			}
			if event.Timestamp.Before(operatorEvent.Timestamp) {
				copy := ctx.Facts[i]
				before = &copy
			} else if event.Timestamp.After(operatorEvent.Timestamp) && after == nil {
				copy := ctx.Facts[i]
				after = &copy
			}
		}
		if before == nil || after == nil || before.Fingerprint == after.Fingerprint {
			continue
		}
		e := newEpisode("guidance-repair", *before, *after, ctx.EventsByID, ctx.Events)
		e.EvidenceEventIDs = []string{correction.EventID, before.CallEventID, after.CallEventID}
		e.Turns = 2
		out = append(out, e)
	}
	return out
}

func detectHandoffContinuation(ctx EpisodeDetectorContext) []FrictionEpisode {
	var out []FrictionEpisode
	for _, event := range ctx.Events {
		message, ok := event.Data.(*domain.MessageEventData)
		if !ok || !message.Terminal || !strings.EqualFold(message.Role, "assistant") {
			continue
		}
		var first, last *InvocationFact
		for i := range ctx.Facts {
			factEvent := ctx.EventsByID[ctx.Facts[i].CallEventID]
			if factEvent != nil && factEvent.Timestamp.After(event.Timestamp) {
				if first == nil {
					copy := ctx.Facts[i]
					first = &copy
				}
				copy := ctx.Facts[i]
				last = &copy
			}
		}
		if first == nil || last == nil {
			continue
		}
		e := newEpisode("handoff-continuation", *first, *last, ctx.EventsByID, ctx.Events)
		e.EvidenceEventIDs = []string{event.ID.String(), first.CallEventID, last.CallEventID}
		e.Turns = len(ctx.Facts)
		out = append(out, e)
	}
	return out
}
