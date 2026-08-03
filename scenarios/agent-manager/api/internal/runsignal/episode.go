package runsignal

import (
	"sort"
	"time"

	"agent-manager/internal/domain"
)

// EpisodeClassifierVersion pins the deterministic episode rules so retained
// evidence is never silently compared across classifier changes.
const EpisodeClassifierVersion = "friction-episode.v8"

const idleEpisodeThreshold = 5 * time.Minute

// FrictionEpisode is the bounded unit used by investigations. It carries only
// event identifiers and derived classifications; raw event payloads remain in
// the event log.
type FrictionEpisode struct {
	EpisodeID              string   `json:"episodeId"`
	RunID                  string   `json:"runId,omitempty"`
	ClassifierVersion      string   `json:"classifierVersion"`
	Pattern                string   `json:"pattern"`
	CauseScope             string   `json:"causeScope"`
	Severity               string   `json:"severity"`
	HonestyFlags           []string `json:"honestyFlags"`
	StartEventID           string   `json:"startEventId"`
	EndEventID             string   `json:"endEventId"`
	EvidenceEventIDs       []string `json:"evidenceEventIds"`
	Turns                  int      `json:"turns"`
	CycleCount             int      `json:"cycleCount,omitempty"`
	RepeatedElement        string   `json:"repeatedElement,omitempty"`
	Tokens                 int      `json:"tokens"`
	WallClockMS            int64    `json:"wallClockMs"`
	SuspectedOwnerScenario string   `json:"suspectedOwnerScenario,omitempty"`
	SuspectedOwnerCommand  string   `json:"suspectedOwnerCommand,omitempty"`
	OwnerConfidence        string   `json:"ownerConfidence"`
	FailedJoinedCalls      int      `json:"failedJoinedCalls,omitempty"`
	Fingerprint            string   `json:"fingerprint"`
}

// DeriveEpisodes deterministically folds invocation facts and event timing
// into bounded friction windows. It makes no network or model calls.
func DeriveEpisodes(facts []InvocationFact, events []*domain.RunEvent) []FrictionEpisode {
	byID := make(map[string]*domain.RunEvent, len(events))
	ordered := append([]*domain.RunEvent(nil), events...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Timestamp.Before(ordered[j].Timestamp) })
	for _, event := range ordered {
		if event != nil {
			byID[event.ID.String()] = event
		}
	}
	context := EpisodeDetectorContext{Facts: facts, Events: ordered, EventsByID: byID, SelfReports: DeriveSelfReportSpans(ordered)}
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

func newEpisode(pattern string, start, end InvocationFact, byID map[string]*domain.RunEvent, ordered []*domain.RunEvent) FrictionEpisode {
	startID, endID := start.CallEventID, end.CallEventID
	if end.ResultEventID != "" {
		endID = end.ResultEventID
	}
	wall := int64(0)
	if a, b := byID[startID], byID[endID]; a != nil && b != nil && b.Timestamp.After(a.Timestamp) {
		wall = b.Timestamp.Sub(a.Timestamp).Milliseconds()
	}
	cause := episodeCauseScope(pattern)
	owner, command, confidence := resolvedEpisodeOwner(end)
	if owner == "" {
		owner, command, confidence = resolvedEpisodeOwner(start)
	}
	e := FrictionEpisode{ClassifierVersion: EpisodeClassifierVersion, Pattern: pattern, CauseScope: cause, Severity: "one-off", HonestyFlags: []string{"auto-generated"}, StartEventID: startID, EndEventID: endID, EvidenceEventIDs: []string{startID, endID}, Turns: 2, Tokens: episodeTokens(startID, endID, byID, ordered), WallClockMS: wall, SuspectedOwnerScenario: owner, SuspectedOwnerCommand: command, OwnerConfidence: confidence}
	e.EpisodeID = fingerprint(pattern, startID, endID)
	e.Fingerprint = fingerprint(pattern, command, cause)
	return e
}

func episodeTokens(startID, endID string, byID map[string]*domain.RunEvent, ordered []*domain.RunEvent) int {
	start, startOK := byID[startID]
	end, endOK := byID[endID]
	if !startOK || !endOK {
		return 0
	}
	from, to := start.Timestamp, end.Timestamp
	if to.Before(from) {
		from, to = to, from
	}
	tokens := 0
	for _, event := range ordered {
		if event == nil || event.Timestamp.Before(from) || event.Timestamp.After(to) {
			continue
		}
		if usage, ok := event.Data.(*domain.UsageEventData); ok {
			tokens += usage.InputTokens + usage.OutputTokens + usage.CacheCreationTokens + usage.CacheReadTokens
		}
	}
	return tokens
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
