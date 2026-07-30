package runreport

import (
	"sort"
	"time"

	"agent-manager/internal/domain"
)

// EpisodeClassifierVersion pins the deterministic episode rules so retained
// evidence is never silently compared across classifier changes.
const EpisodeClassifierVersion = "friction-episode.v2"

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
	Tokens                 int      `json:"tokens"`
	WallClockMS            int64    `json:"wallClockMs"`
	SuspectedOwnerScenario string   `json:"suspectedOwnerScenario,omitempty"`
	SuspectedOwnerCommand  string   `json:"suspectedOwnerCommand,omitempty"`
	OwnerConfidence        string   `json:"ownerConfidence"`
	FailedJoinedCalls      int      `json:"failedJoinedCalls,omitempty"`
	Fingerprint            string   `json:"fingerprint"`
}

// UpgradeEpisodeOwnership records only receipt evidence whose observed time
// falls inside an episode's event window. This prevents an unrelated target
// call elsewhere in the run from changing ownership attribution.
func UpgradeEpisodeOwnership(episodes []FrictionEpisode, events []*domain.RunEvent, calls []CrossScenarioCall, availability Availability) []FrictionEpisode {
	if availability.State == "unobserved" || availability.State == "unavailable" {
		return episodes
	}
	byID := make(map[string]*domain.RunEvent, len(events))
	for _, event := range events {
		if event != nil {
			byID[event.ID.String()] = event
		}
	}
	for index := range episodes {
		start, startOK := byID[episodes[index].StartEventID]
		end, endOK := byID[episodes[index].EndEventID]
		if !startOK || !endOK {
			continue
		}
		from, to := start.Timestamp, end.Timestamp
		if to.Before(from) {
			from, to = to, from
		}
		for _, call := range calls {
			if call.TargetScenario == "" || call.OccurredAt.Before(from) || call.OccurredAt.After(to) {
				continue
			}
			episodes[index].EvidenceEventIDs = append(episodes[index].EvidenceEventIDs, call.ReceiptEventID)
			if call.Outcome != "success" {
				episodes[index].FailedJoinedCalls++
			}
			if call.TargetScenario == episodes[index].SuspectedOwnerScenario {
				episodes[index].OwnerConfidence = "receipt-verified"
			} else {
				episodes[index].OwnerConfidence = "conflicting"
			}
		}
	}
	return episodes
}

var episodeCauseScopes = map[string]string{
	"repeated-work":   "recurring-workaround",
	"command-failure": "toolchain",
	"help-recovery":   "run-execution",
	"stall":           "run-execution",
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
	var episodes []FrictionEpisode
	for i := 1; i < len(facts); i++ {
		if facts[i].Fingerprint != "" && facts[i].Fingerprint == facts[i-1].Fingerprint {
			episodes = append(episodes, newEpisode("repeated-work", facts[i-1], facts[i], byID, ordered))
		}
		if facts[i-1].Outcome == "failure" && facts[i].Outcome == "failure" && facts[i-1].Executable != "" && facts[i-1].Executable == facts[i].Executable {
			episodes = append(episodes, newEpisode("command-failure", facts[i-1], facts[i], byID, ordered))
		}
		if facts[i-1].Outcome == "failure" && facts[i].HelpRecovery {
			episodes = append(episodes, newEpisode("help-recovery", facts[i-1], facts[i], byID, ordered))
		}
	}
	for i := 1; i < len(ordered); i++ {
		if ordered[i-1] == nil || ordered[i] == nil || ordered[i].Timestamp.Sub(ordered[i-1].Timestamp) <= idleEpisodeThreshold {
			continue
		}
		f := InvocationFact{CallEventID: ordered[i-1].ID.String(), ResultEventID: ordered[i].ID.String(), Ownership: "unknown"}
		e := newEpisode("stall", f, f, byID, ordered)
		e.EndEventID = ordered[i].ID.String()
		e.EvidenceEventIDs = []string{e.StartEventID, e.EndEventID}
		e.WallClockMS = ordered[i].Timestamp.Sub(ordered[i-1].Timestamp).Milliseconds()
		e.Tokens = episodeTokens(e.StartEventID, e.EndEventID, byID, ordered)
		e.EpisodeID = fingerprint(e.Pattern, e.StartEventID, e.EndEventID)
		e.Fingerprint = fingerprint(e.Pattern, e.SuspectedOwnerCommand, e.CauseScope)
		episodes = append(episodes, e)
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
	cause := episodeCauseScopes[pattern]
	if cause == "" {
		cause = "unknown"
	}
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
		if cost, ok := event.Data.(*domain.CostEventData); ok {
			tokens += cost.InputTokens + cost.OutputTokens + cost.CacheCreationTokens + cost.CacheReadTokens
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
