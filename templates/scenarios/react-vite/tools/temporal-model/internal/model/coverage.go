package model

import "react-vite-temporal-model/internal/contract"

type TraceCoverageSummary struct {
	CoveredStates []string
	MissingStates []string
	CoveredEvents []string
	MissingEvents []string
}

func NamedTraceCoverage(flow Flow) TraceCoverageSummary {
	coveredStates := map[string]bool{}
	coveredEvents := map[string]bool{}
	for _, trace := range flow.Traces {
		coveredStates[trace.Initial] = true
		for _, step := range trace.Steps {
			coveredEvents[step.Event] = true
			coveredStates[step.Want] = true
		}
	}
	return traceCoverageSummary(flow.States, flow.Events, coveredStates, coveredEvents)
}

func traceCoverageSummary(states []contract.State, events []contract.Event, coveredStates map[string]bool, coveredEvents map[string]bool) TraceCoverageSummary {
	var summary TraceCoverageSummary
	for _, state := range states {
		if coveredStates[state.ID] {
			summary.CoveredStates = append(summary.CoveredStates, state.ID)
		} else {
			summary.MissingStates = append(summary.MissingStates, state.ID)
		}
	}
	for _, event := range events {
		if coveredEvents[event.ID] {
			summary.CoveredEvents = append(summary.CoveredEvents, event.ID)
		} else {
			summary.MissingEvents = append(summary.MissingEvents, event.ID)
		}
	}
	return summary
}
