package runsignal

import (
	"strconv"
	"strings"
	"time"

	"agent-manager/internal/domain"
)

type EpisodeDetectorContext struct {
	Facts               []InvocationFact
	Events              []*domain.RunEvent
	EventsByID          map[string]*domain.RunEvent
	SelfReports         []SelfReportSpan
	PreferredPrimitives []PreferredPrimitive
	// GoalRun is a run with a goal (see WithGoalRun).
	GoalRun bool
}

// detectorWindow is constant, which bounds each scan and keeps both detectors
// linear in the size of even very large imported transcripts.
const detectorWindow = 32

func adjacentEpisode(pattern string, matches func(InvocationFact, InvocationFact) bool) func(EpisodeDetectorContext) []FrictionEpisode {
	return func(ctx EpisodeDetectorContext) []FrictionEpisode {
		out := []FrictionEpisode{}
		for i := 1; i < len(ctx.Facts); i++ {
			if comparableFingerprintFact(ctx.Facts[i-1]) && comparableFingerprintFact(ctx.Facts[i]) && matches(ctx.Facts[i-1], ctx.Facts[i]) {
				out = append(out, newEpisode(pattern, ctx.Facts[i-1], ctx.Facts[i], ctx.EventsByID, ctx.Events))
			}
		}
		return out
	}
}

// maxFailureLine bounds the first error line kept on a command-failure episode.
const maxFailureLine = 160

// detectCommandFailures reports scenario friction: a call of a Vrooli
// project-owned command (resolved through the catalog) that fails with an
// error, whatever the cause: wrong arguments, or the scenario or its
// infrastructure being broken or unavailable. A no-match, an interrupt, a
// help probe, a documented informational verdict and any command outside the
// project are not command failures. One failing call is one episode.
func detectCommandFailures(ctx EpisodeDetectorContext) []FrictionEpisode {
	out := []FrictionEpisode{}
	for _, fact := range ctx.Facts {
		if fact.Ownership != OwnershipResolved || fact.Executable == "" || fact.CommandPath == "" || !comparableFingerprintFact(fact) || !failed(fact) || fact.SemanticsVerdict != "" || isHelpProbe(fact, ctx.EventsByID) {
			continue
		}
		e := newEpisode("command-failure", fact, fact, ctx.EventsByID, ctx.Events)
		e.RepeatedElement = firstErrorLine(fact, ctx.EventsByID)
		out = append(out, e)
	}
	return out
}

// firstErrorLine is the first non-empty line of a failed call's error, else of
// its output, bounded; the exit code when the result says nothing.
func firstErrorLine(fact InvocationFact, byID map[string]*domain.RunEvent) string {
	if event := byID[fact.ResultEventID]; event != nil {
		if data, ok := event.Data.(*domain.ToolResultEventData); ok {
			texts := []string{data.Error, data.Output}
			for _, block := range data.OutputBlocks {
				texts = append(texts, block.Text)
			}
			for _, text := range texts {
				for _, line := range strings.Split(text, "\n") {
					if line = strings.TrimSpace(line); line != "" {
						if len(line) > maxFailureLine {
							line = strings.ToValidUTF8(line[:maxFailureLine], "") + "…"
						}
						return line
					}
				}
			}
		}
	}
	if fact.ExitCode != nil {
		return "exit code " + strconv.Itoa(*fact.ExitCode)
	}
	return "no error message"
}

// maxStallGap is the longest idle gap that can be a stall. Of the 48 gaps over
// five minutes the agent itself owned in the last nine days of live runs
// (the next call or reply was due from it), 46 were under two hours and the
// two longer ones (2–4 h, 4.1 h) lie beyond any model call or retry back-off:
// a suspended machine or a provider outage, not the agent stalling.
const maxStallGap = 2 * time.Hour

// isTurnEvent is an event of the conversation itself: a call, its result or a
// message.
func isTurnEvent(event *domain.RunEvent) bool {
	switch event.EventType {
	case domain.EventTypeToolCall, domain.EventTypeToolResult, domain.EventTypeMessage:
		return true
	}
	return false
}

// isBoundaryEvent ends a turn: a status change (park, wake, stop), a lifecycle
// transition or a goal change. Nothing before it and after it is one stretch of
// work.
func isBoundaryEvent(event *domain.RunEvent) bool {
	switch event.EventType {
	case domain.EventTypeStatus, domain.EventTypeLifecycle, domain.EventTypeGoalStatusChanged:
		return true
	}
	return false
}

// detectStalls finds an idle gap the agent owed: a tool's result was in, and
// the agent's next call came more than idleEpisodeThreshold later. Every other
// gap is someone else's wait: a tool running (the next event is its result),
// the operator or an injected message (the next event is a message), a
// terminal or evidence-only message, a park, wake, stop or other boundary, or
// a gap over maxStallGap. Usage, quota and log events fall in the gap without
// ending it.
func detectStalls(ctx EpisodeDetectorContext) []FrictionEpisode {
	out := []FrictionEpisode{}
	var prev *domain.RunEvent
	for _, event := range ctx.Events {
		if event == nil {
			continue
		}
		if isBoundaryEvent(event) || endsTurn(event) {
			prev = nil
			continue
		}
		if !isTurnEvent(event) {
			continue
		}
		if prev != nil && prev.EventType == domain.EventTypeToolResult && event.EventType == domain.EventTypeToolCall {
			if gap := event.Timestamp.Sub(prev.Timestamp); gap > idleEpisodeThreshold && gap <= maxStallGap {
				fact := InvocationFact{CallEventID: prev.ID.String(), ResultEventID: event.ID.String(), Ownership: "unknown"}
				e := newEpisode("stall", fact, fact, ctx.EventsByID, ctx.Events)
				e.EndEventID, e.EvidenceEventIDs = event.ID.String(), []string{e.StartEventID, event.ID.String()}
				e.WallClockMS = gap.Milliseconds()
				e.EpisodeID, e.Fingerprint = fingerprint(e.Pattern, e.StartEventID, e.EndEventID), fingerprint(e.Pattern, e.SuspectedOwnerCommand, e.CauseScope)
				out = append(out, e)
			}
		}
		prev = event
	}
	return out
}

// endsTurn is a message that closes the agent's turn: a terminal or
// evidence-only message.
func endsTurn(event *domain.RunEvent) bool {
	data, ok := event.Data.(*domain.MessageEventData)
	return ok && (data.Terminal || data.EvidenceOnly)
}
