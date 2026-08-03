package invocationreadmodel

import (
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/runsignal"
)

// Project is the one adapter from durable run events to the analytical row
// shape. Classification itself remains owned by runsignal.
func Project(run *domain.Run, events []*domain.RunEvent, projectedAt time.Time, resolver ...runsignal.CapabilityResolver) ([]Fact, Watermark) {
	var capability runsignal.CapabilityResolver
	if len(resolver) > 0 {
		capability = resolver[0]
	}
	return project(run, events, projectedAt, capability, nil)
}

// ProjectWithResolvers is the production projection entry point. It wires
// both runner-owned capability and command extraction contracts while keeping
// Project available to focused callers that only need capability labels.
func ProjectWithResolvers(run *domain.Run, events []*domain.RunEvent, projectedAt time.Time, capability runsignal.CapabilityResolver, command runsignal.CommandResolver) ([]Fact, Watermark) {
	return project(run, events, projectedAt, capability, command)
}

func project(run *domain.Run, events []*domain.RunEvent, projectedAt time.Time, capability runsignal.CapabilityResolver, command runsignal.CommandResolver) ([]Fact, Watermark) {
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
	for _, fact := range runsignal.DeriveInvocationFactsWithResolver(events, capability, command) {
		occurredAt, ok := timestamps[fact.CallEventID]
		timeBasis := "call_event"
		if !ok {
			occurredAt, timeBasis = projectedAt, "projection_time"
		}
		facts = append(facts, Fact{InvocationFact: fact, RunID: run.ID.String(), OccurredAt: occurredAt, TimeBasis: timeBasis, ProfileID: profileID, RunnerType: runnerType, Model: model, Tag: tag, RunStatus: string(run.Status)})
	}
	watermark := Watermark{RunID: run.ID.String(), ClassifierVersion: runsignal.InvocationFactVersion, EpisodeClassifierVersion: runsignal.EpisodeClassifierVersion, SelfReportClassifierVersion: runsignal.SelfReportClassifierVersion, ProjectedAt: projectedAt}
	if last != nil {
		watermark.LastEventID, watermark.LastEventAt = last.ID.String(), last.Timestamp
	}
	return facts, watermark
}

// ProjectEpisodes folds the same retained event set as invocation facts and
// attaches the shared run dimensions used by every durable cohort query.
func ProjectEpisodes(run *domain.Run, facts []Fact, events []*domain.RunEvent, projectedAt time.Time) []Episode {
	profileID, runnerType, model := runDimensions(run)
	tag := run.Tag
	if tag == "" {
		tag = "unknown"
	}
	invocations := make([]runsignal.InvocationFact, 0, len(facts))
	for _, fact := range facts {
		invocations = append(invocations, fact.InvocationFact)
	}
	timestamps := eventTimestamps(events)
	episodes := runsignal.DeriveEpisodes(invocations, events)
	out := make([]Episode, 0, len(episodes))
	for _, episode := range episodes {
		occurredAt, basis := timestamps[episode.StartEventID], "start_event"
		if occurredAt.IsZero() {
			occurredAt, basis = projectedAt, "projection_time"
		}
		episode.RunID = run.ID.String()
		out = append(out, Episode{FrictionEpisode: episode, RunID: run.ID.String(), OccurredAt: occurredAt, TimeBasis: basis, ProfileID: profileID, RunnerType: runnerType, Model: model, Tag: tag, RunStatus: string(run.Status)})
	}
	return out
}

// ProjectSelfReportSpans folds assistant messages from the same retained
// event set and records their event-time basis explicitly.
func ProjectSelfReportSpans(run *domain.Run, events []*domain.RunEvent, projectedAt time.Time) []SelfReportSpan {
	profileID, runnerType, model := runDimensions(run)
	tag := run.Tag
	if tag == "" {
		tag = "unknown"
	}
	timestamps := eventTimestamps(events)
	spans := runsignal.DeriveSelfReportSpans(events)
	out := make([]SelfReportSpan, 0, len(spans))
	for _, span := range spans {
		occurredAt, basis := timestamps[span.EventID], "message_event"
		if occurredAt.IsZero() {
			occurredAt, basis = projectedAt, "projection_time"
		}
		out = append(out, SelfReportSpan{SelfReportSpan: span, RunID: run.ID.String(), OccurredAt: occurredAt, TimeBasis: basis, ProfileID: profileID, RunnerType: runnerType, Model: model, Tag: tag, RunStatus: string(run.Status)})
	}
	return out
}

func eventTimestamps(events []*domain.RunEvent) map[string]time.Time {
	timestamps := make(map[string]time.Time, len(events))
	for _, event := range events {
		if event != nil {
			timestamps[event.ID.String()] = event.Timestamp
		}
	}
	return timestamps
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
func ProjectRun(run *domain.Run, events []*domain.RunEvent, projectedAt time.Time, resolver ...runsignal.CapabilityResolver) RunFact {
	profileID, runnerType, model := runDimensions(run)
	tag := run.Tag
	if tag == "" {
		tag = "unknown"
	}
	workloadKind, workloadKey, workloadInstance := string(run.Workload.Kind), run.Workload.Key, run.Workload.Instance
	if (workloadKind == "" || (workloadKind == string(domain.WorkloadKindAdhoc) && workloadKey == "")) && run.Tag != "" {
		if recovered, ok := domain.WorkloadFromHistoricalTag(run.Tag); ok {
			workloadKind, workloadKey, workloadInstance = string(recovered.Kind), recovered.Key, recovered.Instance
		}
	}
	if workloadKind == "" {
		workloadKind = string(domain.WorkloadKindAdhoc)
	}
	occurredAt, timeBasis := run.CreatedAt, "created_at"
	if run.EndedAt != nil {
		occurredAt, timeBasis = *run.EndedAt, "ended_at"
	}
	if occurredAt.IsZero() {
		occurredAt, timeBasis = projectedAt, "projection_time"
	}
	var cost, inputCost, outputCost, cacheReadCost, cacheCreationCost float64
	var tokens, inputTokens, outputTokens, cacheReadTokens, cacheCreationTokens int64
	var turns, toolCalls, totalChargeMicroUSD, meteredChargeMicroUSD, unpricedTokenCount int64
	unpriced := false
	charged := false
	var readCalls, rereads int64
	fileReads := map[string]int{}
	// Retained streams can contain the same cumulative usage snapshot twice
	// (live capture plus terminal recovery). Deduplicate identical snapshots so
	// replay remains idempotent while distinct usage facts are still additive.
	type usageIdentity struct {
		input, output, cacheCreate, cacheRead, turns int
		model, runner, tier                          string
		webSearch, serverTool                        int
	}
	seenUsage := map[usageIdentity]struct{}{}
	addCharge := func(charge *domain.ChargeEventData) {
		if charge == nil {
			return
		}
		if charge.Basis == domain.ChargeBasisUnpriced {
			unpriced = true
		}
		if charge.AmountMicroUSD == nil {
			return
		}
		charged = true
		amount := float64(*charge.AmountMicroUSD) / 1_000_000
		totalChargeMicroUSD += *charge.AmountMicroUSD
		cost += amount
		if charge.Basis == domain.ChargeBasisMetered {
			meteredChargeMicroUSD += *charge.AmountMicroUSD
		}
		if charge.InputMicroUSD != nil {
			inputCost += float64(*charge.InputMicroUSD) / 1_000_000
		}
		if charge.OutputMicroUSD != nil {
			outputCost += float64(*charge.OutputMicroUSD) / 1_000_000
		}
		if charge.CacheReadMicroUSD != nil {
			cacheReadCost += float64(*charge.CacheReadMicroUSD) / 1_000_000
		}
		if charge.CacheCreateMicroUSD != nil {
			cacheCreationCost += float64(*charge.CacheCreateMicroUSD) / 1_000_000
		}
	}
	for _, event := range events {
		if event == nil {
			continue
		}
		if usage, ok := event.Data.(*domain.UsageEventData); ok {
			identity := usageIdentity{input: usage.InputTokens, output: usage.OutputTokens, cacheCreate: usage.CacheCreationTokens, cacheRead: usage.CacheReadTokens, turns: usage.Turns, model: usage.Model, runner: usage.RunnerType, tier: usage.ServiceTier, webSearch: usage.WebSearchRequests, serverTool: usage.ServerToolUseRequests}
			if _, duplicate := seenUsage[identity]; duplicate {
				continue
			}
			seenUsage[identity] = struct{}{}
			if int64(usage.Turns) > turns {
				turns = int64(usage.Turns)
			}
			tokens += int64(usage.InputTokens + usage.OutputTokens + usage.CacheCreationTokens + usage.CacheReadTokens)
			inputTokens += int64(usage.InputTokens)
			outputTokens += int64(usage.OutputTokens)
			cacheReadTokens += int64(usage.CacheReadTokens)
			cacheCreationTokens += int64(usage.CacheCreationTokens)
			addCharge(usage.Charge)
		}
		if charge, ok := event.Data.(*domain.ChargeEventData); ok {
			addCharge(charge)
		}
		if call, ok := event.Data.(*domain.ToolCallEventData); ok {
			toolCalls++
			if path := runsignal.ReadPath(call); path != "" {
				readCalls++
				fileReads[path]++
				if fileReads[path] == 2 {
					rereads++
				}
			}
		}
	}
	if run.Summary != nil {
		turns = int64(run.Summary.TurnsUsed)
	}
	if unpriced {
		unpricedTokenCount = tokens
	}
	duration := int64(0)
	if run.StartedAt != nil && run.EndedAt != nil {
		duration = run.EndedAt.Sub(*run.StartedAt).Milliseconds()
		if duration < 0 {
			duration = 0
		}
	}
	eventTimeBasis := "ingestion"
	if run.ExecutionMode == domain.ExecutionModeImported && hasEventTimestamp(events) {
		eventTimeBasis = "transcript"
	}
	costBasis := timeBasis
	if run.ExecutionMode == domain.ExecutionModeImported && cost == 0 && !charged && !unpriced {
		// A transcript without usage is not a free run. Keep that distinction
		// durable so cost aggregates can exclude it or report it explicitly.
		costBasis = "unknown"
	}
	return RunFact{RunID: run.ID.String(), GoalID: run.GoalID, GoalStatus: run.GoalStatus, OccurredAt: occurredAt, CreatedAt: run.CreatedAt, StartedAt: run.StartedAt, EndedAt: run.EndedAt, DurationMS: duration, Status: string(run.Status), ProfileID: profileID, RunnerType: runnerType, Model: model, Tag: tag, WorkloadKind: workloadKind, WorkloadKey: workloadKey, WorkloadInstance: workloadInstance, TotalCostUSD: cost, InputCostUSD: inputCost, OutputCostUSD: outputCost, CacheReadCostUSD: cacheReadCost, CacheCreationCostUSD: cacheCreationCost, TotalTokens: tokens, InputTokens: inputTokens, OutputTokens: outputTokens, CacheReadTokens: cacheReadTokens, CacheCreationTokens: cacheCreationTokens, Turns: turns, ToolCalls: toolCalls, TotalChargeMicroUSD: totalChargeMicroUSD, MeteredChargeMicroUSD: meteredChargeMicroUSD, UnpricedTokenCount: unpricedTokenCount, ReadCalls: readCalls, FileRereads: rereads, TimeAccounting: runsignal.DeriveTimeAccounting(events, run.StartedAt, run.EndedAt), CostTimeBasis: costBasis, TimeBasis: eventTimeBasis, ProjectedAt: projectedAt}
}

func hasEventTimestamp(events []*domain.RunEvent) bool {
	for _, event := range events {
		if event != nil && !event.Timestamp.IsZero() {
			return true
		}
	}
	return false
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
