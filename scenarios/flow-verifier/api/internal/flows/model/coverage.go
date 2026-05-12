package model

import "flow-verifier/internal/flows/contract"

type TraceCoverage struct {
	AllStatesCovered bool     `json:"allStatesCovered"`
	AllEventsCovered bool     `json:"allEventsCovered"`
	CoveredStates    []string `json:"coveredStates"`
	CoveredEvents    []string `json:"coveredEvents"`
	CoveredPairs     []string `json:"coveredPairs,omitempty"`
	AllPairsCovered  *bool    `json:"allPairsCovered,omitempty"`
	MissingStates    []string `json:"-"`
	MissingEvents    []string `json:"-"`
}

type CoverageTrace struct {
	Initial string
	Steps   []CoverageTraceStep
}

type CoverageTraceStep struct {
	Event string
	Want  string
}

func NamedTraceCoverage(flow Flow) TraceCoverage {
	traces := make([]CoverageTrace, 0, len(flow.Traces))
	for _, trace := range flow.Traces {
		steps := make([]CoverageTraceStep, 0, len(trace.Steps))
		for _, step := range trace.Steps {
			steps = append(steps, CoverageTraceStep{Event: step.Event, Want: step.Want})
		}
		traces = append(traces, CoverageTrace{Initial: trace.Initial, Steps: steps})
	}
	return TraceCoverageFor(flow, traces, false)
}

func TraceCoverageFor(flow Flow, traces []CoverageTrace, includePairs bool) TraceCoverage {
	coveredStates := map[string]bool{}
	coveredEvents := map[string]bool{}
	coveredPairs := map[string]bool{}
	for _, trace := range traces {
		current := trace.Initial
		coveredStates[current] = true
		for _, step := range trace.Steps {
			coveredEvents[step.Event] = true
			if includePairs {
				coveredPairs[current+"/"+step.Event] = true
			}
			current = step.Want
			coveredStates[current] = true
		}
	}
	coverage := traceCoverageSummary(flow.States, flow.Events, coveredStates, coveredEvents)
	if includePairs {
		coverage.CoveredPairs = coveredPairIDs(flow.States, flow.Events, coveredPairs)
		coverage.AllPairsCovered = boolPtr(allPairsCovered(flow.States, flow.Events, coveredPairs))
	}
	return coverage
}

func traceCoverageSummary(states []contract.State, events []contract.Event, coveredStates map[string]bool, coveredEvents map[string]bool) TraceCoverage {
	var summary TraceCoverage
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
	summary.AllStatesCovered = len(summary.MissingStates) == 0
	summary.AllEventsCovered = len(summary.MissingEvents) == 0
	return summary
}

func allPairsCovered(states []contract.State, events []contract.Event, seen map[string]bool) bool {
	for _, state := range states {
		for _, event := range events {
			if !seen[state.ID+"/"+event.ID] {
				return false
			}
		}
	}
	return true
}

func coveredPairIDs(states []contract.State, events []contract.Event, seen map[string]bool) []string {
	out := []string{}
	for _, state := range states {
		for _, event := range events {
			key := state.ID + "/" + event.ID
			if seen[key] {
				out = append(out, key)
			}
		}
	}
	return out
}

func boolPtr(value bool) *bool {
	return &value
}
