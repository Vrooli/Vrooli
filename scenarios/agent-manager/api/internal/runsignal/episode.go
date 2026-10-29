package runsignal

import (
	"sort"
	"time"

	"agent-manager/internal/domain"
)

// EpisodeClassifierVersion pins the deterministic episode rules so retained
// evidence is never silently compared across classifier changes.
const EpisodeClassifierVersion = "friction-episode.v13" // v13: a stall is an idle gap of 5 min–2 h inside an active turn (not waiting on a tool, the operator, a park or a resume); wait loops and slow starts stay within one working stretch. v12: a tool call is counted once however often its events were duplicated and never pairs with itself; a pipeline filter (`| head`) is not a file read. v11: WallClockMS is the time the friction wasted, per pattern; read-then-reread pairs only within one working stretch (v10: handoff-continuation stops at the next user message, F-W5-4)

const idleEpisodeThreshold = 5 * time.Minute

// FrictionEpisode is the bounded unit used by investigations. It carries only
// event identifiers and derived classifications; raw event payloads remain in
// the event log.
type FrictionEpisode struct {
	EpisodeID         string   `json:"episodeId"`
	RunID             string   `json:"runId,omitempty"`
	ClassifierVersion string   `json:"classifierVersion"`
	Pattern           string   `json:"pattern"`
	CauseScope        string   `json:"causeScope"`
	Severity          string   `json:"severity"`
	HonestyFlags      []string `json:"honestyFlags"`
	StartEventID      string   `json:"startEventId"`
	EndEventID        string   `json:"endEventId"`
	EvidenceEventIDs  []string `json:"evidenceEventIds"`
	Turns             int      `json:"turns"`
	CycleCount        int      `json:"cycleCount,omitempty"`
	RepeatedElement   string   `json:"repeatedElement,omitempty"`
	Tokens            int      `json:"tokens"`
	// WallClockMS is the time the friction wasted, defined per pattern (see
	// wastedMS): the redundant call's own duration, the failed attempt's, the
	// idle gap, or the loop's span. It is 0 where extra work is not waste by
	// itself (handoff-continuation) or the duration is unknown.
	WallClockMS            int64  `json:"wallClockMs"`
	SuspectedOwnerScenario string `json:"suspectedOwnerScenario,omitempty"`
	SuspectedOwnerCommand  string `json:"suspectedOwnerCommand,omitempty"`
	OwnerConfidence        string `json:"ownerConfidence"`
	FailedJoinedCalls      int    `json:"failedJoinedCalls,omitempty"`
	Fingerprint            string `json:"fingerprint"`
}

// uniqueToolEvents keeps the first call event and the first result event of
// each tool call id, so a duplicated copy does not read as a stall or a gap.
func uniqueToolEvents(events []*domain.RunEvent) []*domain.RunEvent {
	type key struct {
		result bool
		id     string
	}
	seen := map[key]bool{}
	out := make([]*domain.RunEvent, 0, len(events))
	for _, event := range events {
		if event != nil {
			var k key
			switch data := event.Data.(type) {
			case *domain.ToolCallEventData:
				k = key{false, data.ToolCallID}
			case *domain.ToolResultEventData:
				k = key{true, data.ToolCallID}
			}
			if k.id != "" {
				if seen[k] {
					continue
				}
				seen[k] = true
			}
		}
		out = append(out, event)
	}
	return out
}

// uniqueToolCalls keeps the first fact of each tool call (and shell segment).
// An importer that appended a second copy of a run's events gives one call two
// facts with the same tool-call id; paired together they are a "re-read" or a
// "retry" of nothing, so no detector may see both.
func uniqueToolCalls(facts []InvocationFact) []InvocationFact {
	type key struct {
		call    string
		segment int
	}
	seen := make(map[key]bool, len(facts))
	out := make([]InvocationFact, 0, len(facts))
	for _, fact := range facts {
		if fact.ToolCallID != "" {
			k := key{fact.ToolCallID, fact.SegmentIndex}
			if seen[k] {
				continue
			}
			seen[k] = true
		}
		out = append(out, fact)
	}
	return out
}

// DeriveEpisodes deterministically folds invocation facts and event timing
// into bounded friction windows. It makes no network or model calls.
func DeriveEpisodes(facts []InvocationFact, events []*domain.RunEvent) []FrictionEpisode {
	events = uniqueToolEvents(events)
	byID := make(map[string]*domain.RunEvent, len(events))
	ordered := append([]*domain.RunEvent(nil), events...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Timestamp.Before(ordered[j].Timestamp) })
	for _, event := range ordered {
		if event != nil {
			byID[event.ID.String()] = event
		}
	}
	context := EpisodeDetectorContext{Facts: uniqueToolCalls(facts), Events: ordered, EventsByID: byID, SelfReports: DeriveSelfReportSpans(ordered), PreferredPrimitives: PreferredPrimitives()}
	// A malformed or imported transcript can repeat the same tool-call event
	// while preserving its original identifier. Detectors operate over facts,
	// so that repetition must not create two rows with the same durable episode
	// key. De-duplicate only identical classified windows; distinct patterns or
	// event bounds remain independent evidence.
	episodes := make([]FrictionEpisode, 0)
	seen := make(map[string]struct{})
	usage := AccountUsage(ordered)
	for _, detector := range EpisodeDetectors() {
		for _, episode := range detector.Detect(context) {
			key := episode.Pattern + "\x00" + episode.StartEventID + "\x00" + episode.EndEventID
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			episode.Tokens = episodeTokens(episode, byID, ordered, usage)
			episodes = append(episodes, episode)
		}
	}
	return episodes
}

func newEpisode(pattern string, start, end InvocationFact, byID map[string]*domain.RunEvent, ordered []*domain.RunEvent) FrictionEpisode {
	startID, endID := start.CallEventID, end.CallEventID
	if end.ResultEventID != "" {
		endID = end.ResultEventID
	}
	span := int64(0)
	if a, b := byID[startID], byID[endID]; a != nil && b != nil && b.Timestamp.After(a.Timestamp) {
		span = b.Timestamp.Sub(a.Timestamp).Milliseconds()
	}
	wall := wastedMS(pattern, start, end, span, byID)
	cause := episodeCauseScope(pattern)
	owner, command, confidence := resolvedEpisodeOwner(end)
	if owner == "" {
		owner, command, confidence = resolvedEpisodeOwner(start)
	}
	e := FrictionEpisode{ClassifierVersion: EpisodeClassifierVersion, Pattern: pattern, CauseScope: cause, Severity: "one-off", HonestyFlags: []string{"auto-generated"}, StartEventID: startID, EndEventID: endID, EvidenceEventIDs: []string{startID, endID}, Turns: 2, WallClockMS: wall, SuspectedOwnerScenario: owner, SuspectedOwnerCommand: command, OwnerConfidence: confidence}
	e.EpisodeID = fingerprint(pattern, startID, endID)
	e.Fingerprint = fingerprint(pattern, command, cause)
	return e
}

// episodeTokens is the run usage credited between an episode's bounding
// events, from the run's one usage account.
func episodeTokens(e FrictionEpisode, byID map[string]*domain.RunEvent, ordered []*domain.RunEvent, usage UsageAccount) int {
	start, startOK := byID[e.StartEventID]
	end, endOK := byID[e.EndEventID]
	if !startOK || !endOK {
		return 0
	}
	from, to := start.Timestamp, end.Timestamp
	if to.Before(from) {
		from, to = to, from
	}
	return int(usage.Between(ordered, from, to))
}

// resolvedEpisodeOwner accepts only the catalog-backed resolution that was
// captured on the invocation fact. In particular, a known executable with an
// unknown subcommand must not be presented as a manifest-derived owner.
func resolvedEpisodeOwner(fact InvocationFact) (owner, command, confidence string) {
	if fact.Ownership != "resolved" || fact.Executable == "" || fact.CommandPath == "" {
		return "", "", "unknown"
	}
	return fact.Executable, fact.CommandPath, "manifest-derived"
}

// callDurationMS is one tool call's own duration: the recorded duration, else
// the time from its call event to its result event. Unknown is 0.
func callDurationMS(f InvocationFact, byID map[string]*domain.RunEvent) int64 {
	if f.DurationMS != nil {
		return *f.DurationMS
	}
	a, b := byID[f.CallEventID], byID[f.ResultEventID]
	if a != nil && b != nil && b.Timestamp.After(a.Timestamp) {
		return b.Timestamp.Sub(a.Timestamp).Milliseconds()
	}
	return 0
}

// wastedMS is the time an episode of pattern wasted, from its bounding calls
// (start, end) and the span between them:
//   - the redundant call: read-then-reread (the second read), repeated-work,
//     fallback-after-capability (the fallback call), wrong-primitive (both
//     bounding calls: a lower bound, the calls between are not summed);
//   - the failed attempt: retry-after-failure, help-recovery, flag-hunting,
//     abandoned-command-family, capability-abandoned, guidance-repair (the work
//     the operator corrected); command-failure counts both failures;
//   - nothing: handoff-continuation (extra work is not waste by itself; its
//     count and tokens are reported) and blocked-then-abandoned;
//   - the span: stall (the idle gap), time-to-first-success, poll-loop,
//     wait-misuse, oscillation and edit-revert, whose span is the loop itself.
func wastedMS(pattern string, start, end InvocationFact, span int64, byID map[string]*domain.RunEvent) int64 {
	switch pattern {
	case "read-then-reread", "repeated-work", "fallback-after-capability":
		return callDurationMS(end, byID)
	case "wrong-primitive", "command-failure":
		if start.CallEventID == end.CallEventID {
			return callDurationMS(start, byID)
		}
		return callDurationMS(start, byID) + callDurationMS(end, byID)
	case "retry-after-failure", "help-recovery", "flag-hunting", "abandoned-command-family", "capability-abandoned", "guidance-repair":
		return callDurationMS(start, byID)
	case "handoff-continuation", "blocked-then-abandoned":
		return 0
	}
	return span
}
