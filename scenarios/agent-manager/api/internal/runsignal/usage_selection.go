package runsignal

import "agent-manager/internal/domain"

// BeginsRunInvocation identifies the durable owner status boundary emitted
// before a fresh execution or continuation starts consuming provider work.
// Consecutive start events before usage describe the same empty interval.
func BeginsRunInvocation(event *domain.RunEvent) bool {
	if event == nil {
		return false
	}
	status, ok := event.Data.(*domain.StatusEventData)
	return ok && status.NewStatus == string(domain.RunStatusRunning)
}

type usageIdentity struct {
	invocation                                              int
	input, output, cacheCreate, cacheRead, turns, turnIndex int
	authority                                               bool
	model, runner, tier                                     string
	webSearch, serverTool                                   int
}

// UsageSelection carries the existing invocation boundary and one selection
// bit per source event. Callers retain all events for timeline accounting;
// Selected controls only which usage payloads contribute token quantities.
type UsageSelection struct {
	Selected    []bool
	Invocations []int
}

// SelectCanonicalUsage preserves the report projection's existing policy: an
// authoritative reconciliation snapshot replaces interim usage within its
// invocation, exact replayed snapshots count once, and identical usage in
// distinct invocations remains additive.
func SelectCanonicalUsage(events []*domain.RunEvent) UsageSelection {
	selection := UsageSelection{Selected: make([]bool, len(events)), Invocations: make([]int, len(events))}
	invocations := selection.Invocations
	terminalUsage := make(map[int]bool)
	invocation := 0
	for index, event := range events {
		if event == nil {
			continue
		}
		if BeginsRunInvocation(event) {
			invocation++
		}
		invocations[index] = invocation
		if usage, ok := event.Data.(*domain.UsageEventData); ok && usage.ReconciliationAuthority {
			terminalUsage[invocation] = true
		}
	}

	seen := make(map[usageIdentity]struct{})
	for index, event := range events {
		if event == nil {
			continue
		}
		usage, ok := event.Data.(*domain.UsageEventData)
		if !ok || (terminalUsage[invocations[index]] && !usage.ReconciliationAuthority) {
			continue
		}
		identity := usageIdentity{
			invocation: invocations[index], input: usage.InputTokens, output: usage.OutputTokens,
			cacheCreate: usage.CacheCreationTokens, cacheRead: usage.CacheReadTokens,
			turns: usage.Turns, turnIndex: usage.TurnIndex, authority: usage.ReconciliationAuthority,
			model: usage.Model, runner: usage.RunnerType, tier: usage.ServiceTier,
			webSearch: usage.WebSearchRequests, serverTool: usage.ServerToolUseRequests,
		}
		if _, duplicate := seen[identity]; duplicate {
			continue
		}
		seen[identity] = struct{}{}
		selection.Selected[index] = true
	}
	return selection
}
