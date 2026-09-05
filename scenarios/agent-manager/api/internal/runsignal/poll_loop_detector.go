package runsignal

import (
	"strings"
	"time"

	"agent-manager/internal/domain"
)

const (
	pollLoopMinimumCalls = 3
	pollLoopMaximumGap   = 30 * time.Second
)

// detectPollLoops finds a bounded sequence of identical read-only calls that
// keep asking for state without any intervening state-changing event. It uses
// the fingerprint and event structure only; command text is never inspected.
func detectPollLoops(ctx EpisodeDetectorContext) []FrictionEpisode {
	var out []FrictionEpisode
	for start := 0; start+pollLoopMinimumCalls <= len(ctx.Facts); {
		first := ctx.Facts[start]
		if !comparableFingerprintFact(first) || !isReadOnlyFact(first, ctx.EventsByID) {
			start++
			continue
		}
		end := start + 1
		for end < len(ctx.Facts) && samePollStep(ctx, ctx.Facts[end-1], ctx.Facts[end]) {
			end++
		}
		if end-start < pollLoopMinimumCalls {
			start++
			continue
		}
		e := newEpisode("poll-loop", first, ctx.Facts[end-1], ctx.EventsByID, ctx.Events)
		e.Turns = end - start
		e.Severity = "repeated"
		e.HonestyFlags = append(e.HonestyFlags, "generic-read-only-repeat")
		out = append(out, e)
		start = end
	}
	return out
}

func samePollStep(ctx EpisodeDetectorContext, previous, next InvocationFact) bool {
	if !comparableFingerprintFact(previous) || !comparableFingerprintFact(next) || previous.Fingerprint != next.Fingerprint || !isReadOnlyFact(next, ctx.EventsByID) {
		return false
	}
	previousEvent, nextEvent := ctx.EventsByID[previous.ResultEventID], ctx.EventsByID[next.CallEventID]
	if previousEvent == nil || nextEvent == nil || nextEvent.Timestamp.Sub(previousEvent.Timestamp) > pollLoopMaximumGap {
		return false
	}
	return !hasStateTransition(ctx.Events, previousEvent.Timestamp, nextEvent.Timestamp)
}

func isReadOnlyFact(fact InvocationFact, events map[string]*domain.RunEvent) bool {
	if fact.Outcome != "success" || fact.ResultEventID == "" {
		return false
	}
	call := events[fact.CallEventID]
	if call == nil {
		return false
	}
	data, ok := call.Data.(*domain.ToolCallEventData)
	if !ok {
		return false
	}
	for key := range data.Input {
		key = strings.ToLower(key)
		if strings.Contains(key, "write") || strings.Contains(key, "patch") || strings.Contains(key, "delete") || strings.Contains(key, "append") || strings.Contains(key, "content") {
			return false
		}
	}
	return true
}

func hasStateTransition(events []*domain.RunEvent, after, before time.Time) bool {
	for _, event := range events {
		if event == nil || !event.Timestamp.After(after) || !event.Timestamp.Before(before) {
			continue
		}
		switch event.EventType {
		case domain.EventTypeMessage, domain.EventTypeStatus, domain.EventTypeArtifact, domain.EventTypeError, domain.EventTypeLifecycle:
			return true
		}
	}
	return false
}
