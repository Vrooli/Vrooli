package invocationreadmodel

import (
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/runreport"
)

// Project is the one adapter from durable run events to the analytical row
// shape. Classification itself remains owned by runreport.
func Project(run *domain.Run, events []*domain.RunEvent, projectedAt time.Time) ([]Fact, Watermark) {
	timestamps := make(map[string]time.Time, len(events))
	var last *domain.RunEvent
	for _, event := range events {
		if event == nil {
			continue
		}
		timestamps[event.ID.String()] = event.Timestamp
		if last == nil || event.Sequence > last.Sequence {
			last = event
		}
	}
	profileID, runnerType, model := runDimensions(run)
	tag := run.Tag
	if tag == "" {
		tag = "unknown"
	}
	facts := make([]Fact, 0)
	for _, fact := range runreport.DeriveInvocationFacts(events) {
		occurredAt, ok := timestamps[fact.CallEventID]
		timeBasis := "call_event"
		if !ok {
			occurredAt, timeBasis = projectedAt, "projection_time"
		}
		facts = append(facts, Fact{InvocationFact: fact, RunID: run.ID.String(), OccurredAt: occurredAt, TimeBasis: timeBasis, ProfileID: profileID, RunnerType: runnerType, Model: model, Tag: tag, RunStatus: string(run.Status)})
	}
	watermark := Watermark{RunID: run.ID.String(), ClassifierVersion: runreport.InvocationFactVersion, ProjectedAt: projectedAt}
	if last != nil {
		watermark.LastEventID, watermark.LastEventAt = last.ID.String(), last.Timestamp
	}
	return facts, watermark
}

// ProjectErrors folds typed error events into compact, durable analytical
// facts. It intentionally retains no human-readable error content.
func ProjectErrors(run *domain.Run, events []*domain.RunEvent, projectedAt time.Time) []ErrorFact {
	profileID, runnerType, model := runDimensions(run)
	tag := run.Tag
	if tag == "" {
		tag = "unknown"
	}
	facts := make([]ErrorFact, 0)
	for _, event := range events {
		if event == nil {
			continue
		}
		payload, ok := event.Data.(*domain.ErrorEventData)
		if !ok {
			continue
		}
		occurredAt, basis := event.Timestamp, "error_event"
		if occurredAt.IsZero() {
			occurredAt, basis = projectedAt, "projection_time"
		}
		code := payload.Code
		if code == "" {
			code = "unknown"
		}
		facts = append(facts, ErrorFact{RunID: run.ID.String(), EventID: event.ID.String(), OccurredAt: occurredAt, TimeBasis: basis, ErrorCode: code, ProfileID: profileID, RunnerType: runnerType, Model: model, Tag: tag})
	}
	return facts
}

// ProjectRun folds a terminal run and its retained cost events into one
// durable throughput fact. Cost events are additive because runners can emit
// usage for several turns; cache tokens are usage and therefore included.
func ProjectRun(run *domain.Run, events []*domain.RunEvent, projectedAt time.Time) RunFact {
	profileID, runnerType, model := runDimensions(run)
	tag := run.Tag
	if tag == "" {
		tag = "unknown"
	}
	occurredAt, timeBasis := run.CreatedAt, "created_at"
	if run.EndedAt != nil {
		occurredAt, timeBasis = *run.EndedAt, "ended_at"
	}
	if occurredAt.IsZero() {
		occurredAt, timeBasis = projectedAt, "projection_time"
	}
	var cost, authoritativeCost, estimatedCost, unknownCost, inputCost, outputCost, cacheReadCost, cacheCreationCost float64
	var tokens, inputTokens, outputTokens, cacheReadTokens, cacheCreationTokens int64
	var readCalls, rereads int64
	fileReads := map[string]int{}
	for _, event := range events {
		if event == nil {
			continue
		}
		if usage, ok := event.Data.(*domain.CostEventData); ok {
			cost += usage.TotalCostUSD
			switch usage.CostSource {
			case domain.CostSourceRunnerReported, domain.CostSourceProviderUsageAPI:
				authoritativeCost += usage.TotalCostUSD
			case domain.CostSourcePricingTableEstimate:
				estimatedCost += usage.TotalCostUSD
			default:
				unknownCost += usage.TotalCostUSD
			}
			inputCost += usage.InputCostUSD
			outputCost += usage.OutputCostUSD
			cacheReadCost += usage.CacheReadCostUSD
			cacheCreationCost += usage.CacheCreationCostUSD
			tokens += int64(usage.InputTokens + usage.OutputTokens + usage.CacheCreationTokens + usage.CacheReadTokens)
			inputTokens += int64(usage.InputTokens)
			outputTokens += int64(usage.OutputTokens)
			cacheReadTokens += int64(usage.CacheReadTokens)
			cacheCreationTokens += int64(usage.CacheCreationTokens)
		}
		if call, ok := event.Data.(*domain.ToolCallEventData); ok {
			if path := runreport.ReadPath(call); path != "" {
				readCalls++
				fileReads[path]++
				if fileReads[path] == 2 {
					rereads++
				}
			}
		}
	}
	duration := int64(0)
	if run.StartedAt != nil && run.EndedAt != nil {
		duration = run.EndedAt.Sub(*run.StartedAt).Milliseconds()
		if duration < 0 {
			duration = 0
		}
	}
	return RunFact{RunID: run.ID.String(), OccurredAt: occurredAt, CreatedAt: run.CreatedAt, StartedAt: run.StartedAt, EndedAt: run.EndedAt, DurationMS: duration, Status: string(run.Status), ProfileID: profileID, RunnerType: runnerType, Model: model, Tag: tag, TotalCostUSD: cost, AuthoritativeCostUSD: authoritativeCost, EstimatedCostUSD: estimatedCost, UnknownCostUSD: unknownCost, InputCostUSD: inputCost, OutputCostUSD: outputCost, CacheReadCostUSD: cacheReadCost, CacheCreationCostUSD: cacheCreationCost, TotalTokens: tokens, InputTokens: inputTokens, OutputTokens: outputTokens, CacheReadTokens: cacheReadTokens, CacheCreationTokens: cacheCreationTokens, ReadCalls: readCalls, FileRereads: rereads, CostTimeBasis: timeBasis, ProjectedAt: projectedAt}
}

func runDimensions(run *domain.Run) (profileID, runnerType, model string) {
	profileID, runnerType, model = "unknown", "unknown", "unknown"
	if run.AgentProfileID != nil {
		profileID = run.AgentProfileID.String()
	}
	if run.ResolvedConfig != nil {
		runnerType = string(run.ResolvedConfig.RunnerType)
		if run.ResolvedConfig.Model != "" {
			model = run.ResolvedConfig.Model
		}
	}
	if run.ActualModel != "" {
		model = run.ActualModel
	} else if run.RequestedModel != "" {
		model = run.RequestedModel
	}
	return profileID, runnerType, model
}
