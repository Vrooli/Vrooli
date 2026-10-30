package runsignal

import (
	"sort"
	"time"

	"agent-manager/internal/domain"
)

// EpisodeClassifierVersion pins the deterministic episode rules so retained
// evidence is never silently compared across classifier changes. The version
// history is in docs/concepts/USAGE-INTELLIGENCE.md.
const EpisodeClassifierVersion = "friction-episode.v23"

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
	// Turns, Tokens and WallClockMS price the episode by its own redundant
	// calls: how many there are, their argument and result tokens, and their own
	// duration ("redundant call time"; a stall's is its "idle gap"). They are
	// 0 where the pattern is not priced or the duration is unknown.
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

// EpisodeOption states a fact about the run that its events do not carry.
type EpisodeOption func(*EpisodeDetectorContext)

// WithGoalRun marks a run that has a goal: its work after an answer is the
// goal's, not the agent's own.
func WithGoalRun(goal bool) EpisodeOption {
	return func(ctx *EpisodeDetectorContext) { ctx.GoalRun = goal }
}

// DeriveEpisodes deterministically folds invocation facts and event timing
// into bounded friction windows. It makes no network or model calls.
func DeriveEpisodes(facts []InvocationFact, events []*domain.RunEvent, options ...EpisodeOption) []FrictionEpisode {
	events = uniqueToolEvents(events)
	byID := make(map[string]*domain.RunEvent, len(events))
	ordered := append([]*domain.RunEvent(nil), events...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Timestamp.Before(ordered[j].Timestamp) })
	for _, event := range ordered {
		if event != nil {
			byID[event.ID.String()] = event
		}
	}
	context := EpisodeDetectorContext{Facts: uniqueToolCalls(withoutCode(facts)), Events: ordered, EventsByID: byID, SelfReports: DeriveSelfReportSpans(ordered), PreferredPrimitives: PreferredPrimitives()}
	for _, option := range options {
		option(&context)
	}
	// A malformed or imported transcript can repeat the same tool-call event
	// while preserving its original identifier. Detectors operate over facts,
	// so that repetition must not create two rows with the same durable episode
	// key. De-duplicate only identical classified windows; distinct patterns or
	// event bounds remain independent evidence.
	episodes := make([]FrictionEpisode, 0)
	seen := make(map[string]struct{})
	for _, detector := range EpisodeDetectors() {
		for _, episode := range detector.Detect(context) {
			key := episode.Pattern + "\x00" + episode.StartEventID + "\x00" + episode.EndEventID
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			episodes = append(episodes, episode)
		}
	}
	return episodes
}

// agentCauseScope is the cause scope of agent-behaviour patterns.
const agentCauseScope = "prompt-team-agent-storage"

func newEpisode(pattern string, start, end InvocationFact, byID map[string]*domain.RunEvent, ordered []*domain.RunEvent) FrictionEpisode {
	startID, endID := start.CallEventID, end.CallEventID
	if end.ResultEventID != "" {
		endID = end.ResultEventID
	}
	spec, _ := PatternByID(pattern)
	cause := spec.Cause
	if cause == "" {
		cause = "unknown"
	}
	// Only a tool or execution problem belongs to the command next to it; an
	// agent-behaviour episode (an operator correction) has no command owner.
	owner, command, confidence := "", "", "unknown"
	if cause != agentCauseScope {
		owner, command, confidence = resolvedEpisodeOwner(end)
		if owner == "" {
			owner, command, confidence = resolvedEpisodeOwner(start)
		}
	}
	e := FrictionEpisode{ClassifierVersion: EpisodeClassifierVersion, Pattern: pattern, CauseScope: cause, Severity: "one-off", HonestyFlags: []string{"auto-generated"}, StartEventID: startID, EndEventID: endID, EvidenceEventIDs: []string{startID, endID}, SuspectedOwnerScenario: owner, SuspectedOwnerCommand: command, OwnerConfidence: confidence}
	e.price(pricedCalls(spec.Cost, start, end), byID)
	e.EpisodeID = fingerprint(pattern, startID, endID)
	e.Fingerprint = fingerprint(pattern, command, cause)
	return e
}

// withoutCode drops facts of JavaScript or other code: it is not a command, so
// no command pattern may read it.
func withoutCode(facts []InvocationFact) []InvocationFact {
	out := make([]InvocationFact, 0, len(facts))
	for _, fact := range facts {
		if fact.Capability != "code" {
			out = append(out, fact)
		}
	}
	return out
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
