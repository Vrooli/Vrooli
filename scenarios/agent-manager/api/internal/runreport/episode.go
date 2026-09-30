package runreport

import (
	"agent-manager/internal/domain"
	"agent-manager/internal/runsignal"
)

// UpgradeEpisodeOwnership adds receipt-backed presentation evidence without
// changing the deterministic classifier's original signal vocabulary.
func UpgradeEpisodeOwnership(episodes []runsignal.FrictionEpisode, events []*domain.RunEvent, calls []CrossScenarioCall, availability Availability) []runsignal.FrictionEpisode {
	if availability.State == AvailabilityUnobserved || availability.State == AvailabilityUnavailable {
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
