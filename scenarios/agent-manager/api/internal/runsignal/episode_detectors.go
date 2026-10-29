package runsignal

import (
	"time"

	"agent-manager/internal/domain"
)

// EpisodeDetector is a pure, registered classification rule. Its declaration
// is intentionally data-bearing so labels, projections, and later metrics can
// identify the exact detector that made an episode without inspecting code.
type EpisodeDetector interface {
	Identifier() string
	ClassifierVersion() string
	CauseScope() string
	Detect(EpisodeDetectorContext) []FrictionEpisode
}

type EpisodeDetectorContext struct {
	Facts               []InvocationFact
	Events              []*domain.RunEvent
	EventsByID          map[string]*domain.RunEvent
	SelfReports         []SelfReportSpan
	PreferredPrimitives []PreferredPrimitive
}

type episodeDetector struct {
	id, version, cause string
	detect             func(EpisodeDetectorContext) []FrictionEpisode
}

func (d episodeDetector) Identifier() string                                  { return d.id }
func (d episodeDetector) ClassifierVersion() string                           { return d.version }
func (d episodeDetector) CauseScope() string                                  { return d.cause }
func (d episodeDetector) Detect(ctx EpisodeDetectorContext) []FrictionEpisode { return d.detect(ctx) }

// EpisodeDetectors is the complete deterministic registry. Adding a detector
// requires a named declaration and labelled coverage; DeriveEpisodes itself
// remains an unchanged generic dispatcher.
func EpisodeDetectors() []EpisodeDetector {
	return []EpisodeDetector{
		episodeDetector{"repeated-work", EpisodeClassifierVersion, "recurring-workaround", adjacentEpisode("repeated-work", func(a, b InvocationFact) bool { return b.Fingerprint != "" && b.Fingerprint == a.Fingerprint })},
		episodeDetector{"command-failure", EpisodeClassifierVersion, "toolchain", adjacentEpisode("command-failure", func(a, b InvocationFact) bool {
			return a.Outcome == "failure" && b.Outcome == "failure" && a.Executable != "" && a.Executable == b.Executable
		})},
		episodeDetector{"help-recovery", EpisodeClassifierVersion, "run-execution", adjacentEpisode("help-recovery", func(a, b InvocationFact) bool { return a.Outcome == "failure" && b.HelpRecovery })},
		episodeDetector{"stall", EpisodeClassifierVersion, "run-execution", detectStalls},
		episodeDetector{"poll-loop", EpisodeClassifierVersion, "recurring-workaround", detectPollLoops},
		episodeDetector{"wrong-primitive", EpisodeClassifierVersion, "recurring-workaround", func(ctx EpisodeDetectorContext) []FrictionEpisode {
			return DetectWrongPrimitives(ctx.Facts, ctx.EventsByID, ctx.Events)
		}},
		episodeDetector{"oscillation", EpisodeClassifierVersion, "recurring-workaround", detectOscillations},
		episodeDetector{"edit-revert", EpisodeClassifierVersion, "recurring-workaround", detectEditReverts},
		episodeDetector{"wait-misuse", EpisodeClassifierVersion, "toolchain", detectWaitMisuse},
		episodeDetector{"blocked-then-abandoned", EpisodeClassifierVersion, "run-execution", detectBlockedThenAbandoned},
		episodeDetector{"guidance-repair", EpisodeClassifierVersion, "prompt-team-agent-storage", detectGuidanceRepair},
		episodeDetector{"handoff-continuation", EpisodeClassifierVersion, "run-execution", detectHandoffContinuation},
		episodeDetector{"fallback-after-capability", EpisodeClassifierVersion, "toolchain", detectFallbackAfterCapability},
		episodeDetector{"capability-abandoned", EpisodeClassifierVersion, "toolchain", detectCapabilityAbandoned},
		episodeDetector{"retry-after-failure", EpisodeClassifierVersion, "run-execution", detectRetryAfterFailure},
		episodeDetector{"flag-hunting", EpisodeClassifierVersion, "toolchain", detectFlagHunting},
		episodeDetector{"abandoned-command-family", EpisodeClassifierVersion, "toolchain", detectAbandonedCommandFamily},
		episodeDetector{"read-then-reread", EpisodeClassifierVersion, "recurring-workaround", detectReadThenReread},
		episodeDetector{"time-to-first-success", EpisodeClassifierVersion, "run-execution", detectTimeToFirstSuccess},
	}
}

// detectorWindow is constant, which bounds each scan and keeps both detectors
// linear in the size of even very large imported transcripts.
const detectorWindow = 32

func episodeCauseScope(id string) string {
	for _, detector := range EpisodeDetectors() {
		if detector.Identifier() == id {
			return detector.CauseScope()
		}
	}
	return "unknown"
}

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

func messageRole(event *domain.RunEvent) string {
	if data, ok := event.Data.(*domain.MessageEventData); ok {
		return data.Role
	}
	return ""
}

// detectStalls finds an idle gap inside an active turn: between two turn
// events the agent's next call or reply came more than idleEpisodeThreshold
// later. A gap is not a stall when the next event is a tool's result (the tool
// was running) or the operator's message (the operator was), when it crosses a
// park, wake, stop or other boundary event, or when it exceeds maxStallGap. Usage, quota and log
// events fall in the gap without ending it.
func detectStalls(ctx EpisodeDetectorContext) []FrictionEpisode {
	out := []FrictionEpisode{}
	var prev *domain.RunEvent
	for _, event := range ctx.Events {
		if event == nil {
			continue
		}
		if isBoundaryEvent(event) {
			prev = nil
			continue
		}
		if !isTurnEvent(event) {
			continue
		}
		if prev != nil {
			gap := event.Timestamp.Sub(prev.Timestamp)
			// The agent owns the turn after any turn event; what it was waiting on is
			// excluded by what comes next: a tool's result (the tool was running) or
			// the operator's message (the operator was).
			agentNext := event.EventType == domain.EventTypeToolCall || (event.EventType == domain.EventTypeMessage && messageRole(event) != "user")
			if gap > idleEpisodeThreshold && gap <= maxStallGap && agentNext {
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
