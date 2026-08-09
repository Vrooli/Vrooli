package invocationreadmodel

import (
	"encoding/json"
	"math"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/runsignal"
	"agent-manager/internal/tokenaccounting"
)

// DeriveRunSubject turns bounded invocation dimensions into a stable, human
// readable subject. It consumes only recorded tool-call facts; raw command
// arguments and agent-authored completion fields are excluded.
func DeriveRunSubject(facts []Fact, events []*domain.RunEvent) []string {
	seen := make(map[string]struct{})
	for _, fact := range facts {
		for _, candidate := range []string{fact.Executable, fact.CommandPath} {
			candidate = strings.Join(strings.Fields(candidate), " ")
			if candidate == "" || candidate == "unknown" {
				continue
			}
			seen[SubjectTool+candidate] = struct{}{}
		}
	}
	for _, area := range DeriveRunAreas(events) {
		seen[SubjectPath+area] = struct{}{}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

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
	argTokens, resultTokens, tokenBases, residencyTurns, residencySegments := tokenFactors(events)
	preambleInjectedTokens, preambleFixedTokens, _ := derivePreamble(run, events)
	incurredByCall, _, _ := attributeIncurred(events, preambleInjectedTokens, preambleFixedTokens)
	facts := make([]Fact, 0)
	for _, fact := range runsignal.DeriveInvocationFactsWithResolver(events, capability, command) {
		fact.ArgTokens = argTokens[fact.CallEventID]
		fact.ResidencyTurns = residencyTurns[fact.ResultEventID]
		if incurred, ok := incurredByCall[fact.CallEventID]; ok {
			fact.IncurredInputTokens = incurred.Input
			fact.IncurredOutputTokens = incurred.Output
			fact.IncurredCacheReadTokens = incurred.CacheRead
			fact.IncurredCacheCreationTokens = incurred.CacheCreation
		}
		// A result is required before an estimated footprint can be treated as
		// a complete call footprint. Unpaired calls retain their argument count
		// but are explicitly unknown rather than silently estimated.
		if result, ok := resultTokens[fact.ResultEventID]; ok {
			fact.ResultTokens = result
			fact.TokenBasis = tokenBases[fact.ResultEventID]
		} else {
			fact.TokenBasis = tokenaccounting.BasisUnknown
		}
		fact.ResidencySegment = residencySegments[fact.ResultEventID]
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
	} else {
		// Imported or newly-created runs may legitimately have no retained
		// events. The durable watermark still needs a non-null time so that
		// an empty projection is distinguishable from a failed projection.
		watermark.LastEventAt = projectedAt
	}
	return facts, watermark
}

// derivePreamble returns the run-level prefix buckets from measured per-turn
// input. The injected estimate is persisted on the resolved run config at
// creation time; only the fixed remainder is derived during projection.
func derivePreamble(run *domain.Run, events []*domain.RunEvent) (int64, int64, tokenaccounting.Basis) {
	segment := int64(0)
	segmentMinimums := map[int64]int64{}
	hasPerTurnUsage := false
	for _, event := range events {
		if event == nil {
			continue
		}
		if _, ok := event.Data.(*domain.CompactionEventData); ok {
			segment++
			continue
		}
		usage, ok := event.Data.(*domain.UsageEventData)
		if !ok || usage.ReconciliationAuthority {
			continue
		}
		hasPerTurnUsage = true
		if previous, exists := segmentMinimums[segment]; !exists || int64(usage.InputTokens) < previous {
			segmentMinimums[segment] = int64(usage.InputTokens)
		}
	}
	if !hasPerTurnUsage {
		return 0, 0, tokenaccounting.BasisUnknown
	}
	injected := int64(0)
	basis := tokenaccounting.BasisUnknown
	if run != nil && run.ResolvedConfig != nil {
		injected = run.ResolvedConfig.PreambleInjectedTokens
		basis = run.ResolvedConfig.PreambleTokenBasis
		if basis == "" {
			basis = tokenaccounting.BasisUnknown
		}
	}
	var fixed int64
	for _, minimum := range segmentMinimums {
		if remainder := minimum - injected; remainder > 0 {
			fixed += remainder
		}
	}
	return injected, fixed, basis
}

type incurredAttribution struct {
	Input, Output, CacheRead, CacheCreation int64
}

// attributeIncurred assigns each eligible per-turn usage event to the latest
// tool call before it. Terminal provider usage is a reconciliation authority,
// not a second attribution source. The returned residual is usage that had no
// preceding tool call and therefore belongs in the run's unattributed bucket.
func attributeIncurred(events []*domain.RunEvent, preambleInjected, preambleFixed int64) (map[string]incurredAttribution, int64, bool) {
	type usageIdentity struct {
		input, output, cacheRead, cacheCreation, turn, turnIndex int
	}
	var hasAuthority bool
	for _, event := range events {
		if usage, ok := eventDataUsage(event); ok && usage.ReconciliationAuthority {
			hasAuthority = true
		}
	}
	seen := map[usageIdentity]struct{}{}
	byCall := map[string]incurredAttribution{}
	var unattributed int64
	var eligible bool
	remainingPreamble := preambleInjected + preambleFixed
	latestCall := ""
	for _, event := range events {
		if event == nil {
			continue
		}
		if call, ok := event.Data.(*domain.ToolCallEventData); ok {
			latestCall = event.ID.String()
			_ = call
			continue
		}
		usage, ok := eventDataUsage(event)
		if !ok || (hasAuthority && usage.ReconciliationAuthority) {
			continue
		}
		identity := usageIdentity{input: usage.InputTokens, output: usage.OutputTokens, cacheRead: usage.CacheReadTokens, cacheCreation: usage.CacheCreationTokens, turn: usage.Turns, turnIndex: usage.TurnIndex}
		if _, duplicate := seen[identity]; duplicate {
			continue
		}
		seen[identity] = struct{}{}
		eligible = true
		input := int64(usage.InputTokens)
		if remainingPreamble > 0 && input > 0 {
			deducted := input
			if deducted > remainingPreamble {
				deducted = remainingPreamble
			}
			input -= deducted
			remainingPreamble -= deducted
		}
		if latestCall == "" {
			unattributed += input + int64(usage.OutputTokens) + int64(usage.CacheReadTokens) + int64(usage.CacheCreationTokens)
			continue
		}
		value := byCall[latestCall]
		value.Input += input
		value.Output += int64(usage.OutputTokens)
		value.CacheRead += int64(usage.CacheReadTokens)
		value.CacheCreation += int64(usage.CacheCreationTokens)
		byCall[latestCall] = value
	}
	return byCall, unattributed, eligible
}

func eventDataUsage(event *domain.RunEvent) (*domain.UsageEventData, bool) {
	if event == nil {
		return nil, false
	}
	usage, ok := event.Data.(*domain.UsageEventData)
	return usage, ok
}

// tokenFactors estimates bounded payload factors while the source events are
// still available. The durable projection never needs to reopen event JSON.
func tokenFactors(events []*domain.RunEvent) (map[string]int64, map[string]int64, map[string]tokenaccounting.Basis, map[string]int64, map[string]int64) {
	argTokens := make(map[string]int64)
	resultTokens := make(map[string]int64)
	tokenBases := make(map[string]tokenaccounting.Basis)
	residencyTurns := make(map[string]int64)
	residencySegments := make(map[string]int64)
	resultIndexes := make(map[string]int)
	segments := make([]int64, len(events))
	attenuation := []float64{1}
	segment := int64(0)
	hasPerTurnUsage := false
	for index, event := range events {
		if event == nil {
			continue
		}
		segments[index] = segment
		switch payload := event.Data.(type) {
		case *domain.UsageEventData:
			if !payload.ReconciliationAuthority {
				hasPerTurnUsage = true
			}
		case *domain.CompactionEventData:
			if payload.TokensBefore > 0 && payload.TokensAfter > 0 {
				ratio := float64(payload.TokensAfter) / float64(payload.TokensBefore)
				if ratio > 1 {
					ratio = 1
				}
				attenuation = append(attenuation, attenuation[len(attenuation)-1]*ratio)
			} else {
				attenuation = append(attenuation, attenuation[len(attenuation)-1])
			}
			segment++
		case *domain.ToolCallEventData:
			encoded, err := json.Marshal(payload.Input)
			if err == nil {
				argTokens[event.ID.String()] = tokenaccounting.EstimateText(string(encoded)).Tokens
			}
		case *domain.ToolResultEventData:
			resultIndexes[event.ID.String()] = index
			output := payload.Output
			if payload.Error != "" {
				output += "\n" + payload.Error
			}
			estimate := tokenaccounting.EstimateText(output)
			resultTokens[event.ID.String()] = estimate.Tokens
			tokenBases[event.ID.String()] = estimate.Basis
		}
	}
	for resultID, index := range resultIndexes {
		residencySegments[resultID] = segments[index]
		weightedTurns := 0.0
		for laterIndex := index + 1; laterIndex < len(events); laterIndex++ {
			event := events[laterIndex]
			if event == nil {
				continue
			}
			countsAsTurn := false
			if usage, ok := event.Data.(*domain.UsageEventData); ok {
				countsAsTurn = hasPerTurnUsage && !usage.ReconciliationAuthority
			} else if message, ok := event.Data.(*domain.MessageEventData); ok {
				countsAsTurn = !hasPerTurnUsage && message.Role == "assistant"
			}
			if !countsAsTurn {
				continue
			}
			laterSegment := segments[laterIndex]
			weight := 1.0
			if laterSegment >= 0 && laterSegment < int64(len(attenuation)) && segments[index] >= 0 && segments[index] < int64(len(attenuation)) && attenuation[segments[index]] > 0 {
				weight = attenuation[laterSegment] / attenuation[segments[index]]
			}
			weightedTurns += weight
		}
		if weightedTurns > 0 {
			residencyTurns[resultID] = int64(math.Ceil(weightedTurns))
		}
	}
	return argTokens, resultTokens, tokenBases, residencyTurns, residencySegments
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
	// Keep the historical workload recovery calculation explicit even though
	// the legacy RunFact shape still reads the original run instance.
	_ = workloadInstance
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
	preambleInjectedTokens, preambleFixedTokens, preambleBasis := derivePreamble(run, events)
	unpriced := false
	charged := false
	var readCalls, rereads int64
	fileReads := map[string]int{}
	// Retained streams can contain the same cumulative usage snapshot twice
	// (live capture plus terminal recovery). Deduplicate identical snapshots so
	// replay remains idempotent while distinct usage facts are still additive.
	type usageIdentity struct {
		input, output, cacheCreate, cacheRead, turns, turnIndex int
		authority                                               bool
		model, runner, tier                                     string
		webSearch, serverTool                                   int
	}
	seenUsage := map[usageIdentity]struct{}{}
	terminalUsage := false
	for _, event := range events {
		if event == nil {
			continue
		}
		if usage, ok := event.Data.(*domain.UsageEventData); ok && usage.ReconciliationAuthority {
			terminalUsage = true
		}
	}
	incurredByCall, noPrecedingToolCallTokens, hasEligibleUsage := attributeIncurred(events, preambleInjectedTokens, preambleFixedTokens)
	var incurredTokens int64
	for _, incurred := range incurredByCall {
		incurredTokens += incurred.Input + incurred.Output + incurred.CacheRead + incurred.CacheCreation
	}
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
			if terminalUsage && !usage.ReconciliationAuthority {
				continue
			}
			identity := usageIdentity{input: usage.InputTokens, output: usage.OutputTokens, cacheCreate: usage.CacheCreationTokens, cacheRead: usage.CacheReadTokens, turns: usage.Turns, turnIndex: usage.TurnIndex, authority: usage.ReconciliationAuthority, model: usage.Model, runner: usage.RunnerType, tier: usage.ServiceTier, webSearch: usage.WebSearchRequests, serverTool: usage.ServerToolUseRequests}
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
	accountedTokens := preambleInjectedTokens + preambleFixedTokens + incurredTokens + noPrecedingToolCallTokens
	residual := tokens - accountedTokens
	unattributedTokens := noPrecedingToolCallTokens + residual
	unattributedReason := ""
	switch {
	case !hasEligibleUsage:
		unattributedReason = "no provider usage events available for tool attribution"
	case noPrecedingToolCallTokens != 0 && residual != 0:
		unattributedReason = "usage without a preceding tool call and a run-total reconciliation residual"
	case noPrecedingToolCallTokens != 0:
		unattributedReason = "usage turn had no preceding tool call"
	case residual != 0:
		unattributedReason = "run total differed from attributed buckets"
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
	return RunFact{RunID: run.ID.String(), GoalID: run.GoalID, GoalStatus: "", OccurredAt: occurredAt, CreatedAt: run.CreatedAt, StartedAt: run.StartedAt, EndedAt: run.EndedAt, DurationMS: duration, Status: string(run.Status), ProfileID: profileID, RunnerType: runnerType, Model: model, Tag: tag, WorkloadKind: workloadKind, WorkloadKey: workloadKey, WorkloadInstance: run.Workload.Instance, TotalCostUSD: cost, InputCostUSD: inputCost, OutputCostUSD: outputCost, CacheReadCostUSD: cacheReadCost, CacheCreationCostUSD: cacheCreationCost, TotalTokens: tokens, InputTokens: inputTokens, OutputTokens: outputTokens, CacheReadTokens: cacheReadTokens, CacheCreationTokens: cacheCreationTokens, Turns: turns, ToolCalls: toolCalls, TotalChargeMicroUSD: totalChargeMicroUSD, MeteredChargeMicroUSD: meteredChargeMicroUSD, UnpricedTokenCount: unpricedTokenCount, PreambleInjectedTokens: preambleInjectedTokens, PreambleFixedTokens: preambleFixedTokens, PreambleTokenBasis: preambleBasis, UnattributedTokens: unattributedTokens, UnattributedReason: unattributedReason, ReadCalls: readCalls, FileRereads: rereads, TimeAccounting: runsignal.DeriveTimeAccounting(events, run.StartedAt, run.EndedAt), CostTimeBasis: costBasis, TimeBasis: eventTimeBasis, ProjectedAt: projectedAt}
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
