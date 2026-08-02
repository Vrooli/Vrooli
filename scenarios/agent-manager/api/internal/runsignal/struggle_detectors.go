package runsignal

import (
	"strings"

	"agent-manager/internal/domain"
)

func detectWaitMisuse(ctx EpisodeDetectorContext) []FrictionEpisode {
	var out []FrictionEpisode
	for start := 0; start < len(ctx.Facts); {
		if ctx.Facts[start].Capability != "wait" || ctx.Facts[start].Fingerprint == "" {
			start++
			continue
		}
		end := start + 1
		for end < len(ctx.Facts) && ctx.Facts[end].Capability == "wait" && ctx.Facts[end].Fingerprint == ctx.Facts[start].Fingerprint {
			end++
		}
		if end-start >= 3 {
			e := newEpisode("wait-misuse", ctx.Facts[start], ctx.Facts[end-1], ctx.EventsByID, ctx.Events)
			e.Turns = end - start
			e.CycleCount = end - start
			e.RepeatedElement = ctx.Facts[start].Fingerprint
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
		successAfter := false
		for _, fact := range ctx.Facts {
			if event := ctx.EventsByID[fact.CallEventID]; event != nil && event.Timestamp.After(blockedAt.Timestamp) && fact.Outcome == "success" {
				successAfter = true
				break
			}
		}
		if successAfter {
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
