package invocationreadmodel

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/runsignal"
	"agent-manager/internal/tokenaccounting"
	"agent-manager/internal/toolfriction"
)

// BeginsRunInvocation preserves the existing invocationreadmodel API while
// the shared boundary rule is owned by runsignal.
func BeginsRunInvocation(event *domain.RunEvent) bool {
	return runsignal.BeginsRunInvocation(event)
}

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
// shape. Classification itself remains owned by runsignal; the runner codec
// that owns the tool surface supplies command extraction.
func Project(run *domain.Run, events []*domain.RunEvent, projectedAt time.Time, ports toolfriction.PortOwners, command ...runsignal.CommandResolver) ([]Fact, Watermark) {
	timestamps := make(map[string]time.Time, len(events))
	byID := make(map[string]*domain.RunEvent, len(events))
	var last *domain.RunEvent
	for _, event := range events {
		if event == nil {
			continue
		}
		timestamps[event.ID.String()] = event.Timestamp
		byID[event.ID.String()] = event
		if last == nil || event.Sequence > last.Sequence {
			last = event
		}
	}
	profileID, runnerType, model := runDimensions(run)
	tag := run.Tag
	if tag == "" {
		tag = "unknown"
	}
	argTokens, resultTokens, tokenBases, residencyRequests, residencySegments := tokenFactors(events)
	preambleInjectedTokens, preambleFixedTokens, _ := derivePreamble(run, events)
	incurredByCall, _, _ := attributeIncurred(events, runsignal.AccountUsage(events), preambleInjectedTokens, preambleFixedTokens)
	derived := runsignal.DeriveInvocationFacts(events, command...)
	resultShares := apportionResults(derived, resultTokens)
	facts := make([]Fact, 0, len(derived))
	var sources []toolfriction.Source
	var observed []int
	for index, fact := range derived {
		// Token quantities are per call, but a compound command emits one fact
		// per segment. Apportion them so a SUM over facts reconciles to the run
		// instead of multiplying the call's payload by its segment count.
		// Residency requests are a multiplier rather than a token quantity, so
		// they stay whole: result_tokens * residency_requests then apportions
		// with the result share it is paired with.
		share := func(total int64) int64 {
			return tokenaccounting.SegmentShare(total, fact.SegmentCount, fact.SegmentIndex)
		}
		fact.ArgTokens = share(argTokens[fact.CallEventID])
		fact.ResidencyRequests = residencyRequests[fact.OutputEventID]
		if incurred, ok := incurredByCall[fact.CallEventID]; ok {
			fact.IncurredInputTokens = share(incurred.Input)
			fact.IncurredOutputTokens = share(incurred.Output)
			fact.IncurredCacheReadTokens = share(incurred.CacheRead)
			fact.IncurredCacheCreationTokens = share(incurred.CacheCreation)
		}
		// A result is required before an estimated footprint can be treated as
		// a complete call footprint. Unpaired calls retain their argument count
		// but are explicitly unknown rather than silently estimated.
		if _, ok := resultTokens[fact.OutputEventID]; ok && fact.OutputShare != runsignal.ShareNone {
			fact.ResultTokens = resultShares[index]
			fact.TokenBasis = tokenBases[fact.OutputEventID]
		} else {
			fact.TokenBasis = tokenaccounting.BasisUnknown
		}
		fact.ResidencySegment = residencySegments[fact.OutputEventID]
		occurredAt, ok := timestamps[fact.CallEventID]
		timeBasis := "call_event"
		if !ok {
			occurredAt, timeBasis = projectedAt, "projection_time"
		}
		facts = append(facts, Fact{InvocationFact: fact, RunID: run.ID.String(), OccurredAt: occurredAt, TimeBasis: timeBasis, ProfileID: profileID, RunnerType: runnerType, Model: model, Tag: tag, RunStatus: string(run.Status)})
		if toolfriction.Profiled(fact) {
			source := toolfriction.Source{Fact: toolfriction.Fact{InvocationFact: fact, RunID: run.ID.String(), At: occurredAt, Runner: runnerType}}
			if call := byID[fact.CallEventID]; call != nil {
				source.Call, _ = call.Data.(*domain.ToolCallEventData)
			}
			if result := byID[fact.ResultEventID]; result != nil {
				source.Result, _ = result.Data.(*domain.ToolResultEventData)
			}
			sources, observed = append(sources, source), append(observed, len(facts)-1)
		}
	}
	for i, features := range toolfriction.Extract(sources, ports) {
		facts[observed[i]].Tool = &features
	}
	watermark := Watermark{RunID: run.ID.String(), ClassifierVersion: runsignal.InvocationFactVersion, SelfReportClassifierVersion: runsignal.SelfReportClassifierVersion, ProjectedAt: projectedAt}
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

// attributeIncurred assigns each turn-local usage amount to the latest tool
// call before it. Amounts come from the run's usage account, so a closing
// receipt or a replayed report is never a second attribution source. The
// returned residual is usage that had no preceding tool call and therefore
// belongs in the run's unattributed bucket.
func attributeIncurred(events []*domain.RunEvent, usage runsignal.UsageAccount, preambleInjected, preambleFixed int64) (map[string]incurredAttribution, int64, bool) {
	byCall := map[string]incurredAttribution{}
	var unattributed int64
	var eligible bool
	remainingPreamble := preambleInjected + preambleFixed
	latestCall := ""
	for index, event := range events {
		if event == nil {
			continue
		}
		if _, ok := event.Data.(*domain.ToolCallEventData); ok {
			latestCall = event.ID.String()
			continue
		}
		contribution := usage.PerTurn[index]
		if _, ok := event.Data.(*domain.UsageEventData); !ok || contribution.Total() == 0 {
			continue
		}
		eligible = true
		input := int64(contribution.InputTokens)
		if remainingPreamble > 0 && input > 0 {
			deducted := min(input, remainingPreamble)
			input -= deducted
			remainingPreamble -= deducted
		}
		if latestCall == "" {
			unattributed += input + int64(contribution.OutputTokens) + int64(contribution.CacheReadTokens) + int64(contribution.CacheCreationTokens)
			continue
		}
		value := byCall[latestCall]
		value.Input += input
		value.Output += int64(contribution.OutputTokens)
		value.CacheRead += int64(contribution.CacheReadTokens)
		value.CacheCreation += int64(contribution.CacheCreationTokens)
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

// apportionResults splits each call's result tokens (what the agent saw)
// across its facts so that each output counts once (runsignal.OutputShare): a
// command whose nested result reports its size takes the tokens it could have
// put there (OwnTokens), the rest of the result goes evenly to the facts that
// hold it, and segments whose output another segment holds take none. When
// the commands claim more than the result holds, or nothing holds the rest,
// the commands share the whole result by their claims. Shares of one call sum
// to its result tokens.
func apportionResults(facts []runsignal.InvocationFact, resultTokens map[string]int64) []int64 {
	shares := make([]int64, len(facts))
	for start := 0; start < len(facts); {
		end := start + 1
		for end < len(facts) && facts[end].CallEventID == facts[start].CallEventID {
			end++
		}
		total := resultTokens[facts[start].OutputEventID]
		var own, rest []int
		var claimed int64
		for i := start; i < end; i++ {
			switch facts[i].OutputShare {
			case runsignal.ShareOwn:
				own, claimed = append(own, i), claimed+facts[i].OwnTokens
			case runsignal.ShareRest:
				rest = append(rest, i)
			}
		}
		switch {
		case len(rest) > 0 && claimed <= total:
			for _, i := range own {
				shares[i] = facts[i].OwnTokens
			}
			for k, i := range rest {
				shares[i] = tokenaccounting.SegmentShare(total-claimed, len(rest), k)
			}
		case claimed > 0:
			largest, assigned := own[0], int64(0)
			for _, i := range own {
				shares[i] = total * facts[i].OwnTokens / claimed
				assigned += shares[i]
				if facts[i].OwnTokens > facts[largest].OwnTokens {
					largest = i
				}
			}
			// Integer division leaves at most one token per fact; the largest
			// claim takes it so the call still sums to its total.
			shares[largest] += total - assigned
		default:
			for k, i := range own {
				shares[i] = tokenaccounting.SegmentShare(total, len(own), k)
			}
		}
		start = end
	}
	return shares
}

// tokenFactors estimates bounded payload factors while the source events are
// still available. The durable projection never needs to reopen event JSON.
// A result's residency is the provider requests after it that re-read it:
// those before the next compaction (or the end of the run). Without per-turn
// usage, assistant messages stand in for requests. Requests are counted once
// per segment and each result takes the count after it when its segment
// closes, so the pass is linear in events.
func tokenFactors(events []*domain.RunEvent) (map[string]int64, map[string]int64, map[string]tokenaccounting.Basis, map[string]int64, map[string]int64) {
	argTokens := make(map[string]int64)
	resultTokens := make(map[string]int64)
	tokenBases := make(map[string]tokenaccounting.Basis)
	residencyRequests := make(map[string]int64)
	residencySegments := make(map[string]int64)
	hasPerTurnUsage := false
	for _, event := range events {
		if usage, ok := eventDataUsage(event); ok && !usage.ReconciliationAuthority {
			hasPerTurnUsage = true
			break
		}
	}
	type residentResult struct {
		id string
		at int64 // the segment's requests before the result
	}
	var resident []residentResult // results read by every request until the next compaction
	segment, requests := int64(0), int64(0)
	closeSegment := func() {
		for _, r := range resident {
			if n := requests - r.at; n > 0 {
				residencyRequests[r.id] += n
			}
		}
		resident, requests = resident[:0], 0
	}
	for _, event := range events {
		if event == nil {
			continue
		}
		switch payload := event.Data.(type) {
		case *domain.CompactionEventData:
			closeSegment()
			segment++
		case *domain.UsageEventData:
			if hasPerTurnUsage && !payload.ReconciliationAuthority {
				requests++
			}
		case *domain.MessageEventData:
			if !hasPerTurnUsage && payload.Role == "assistant" {
				requests++
			}
		case *domain.ToolCallEventData:
			encoded, err := json.Marshal(payload.Input)
			if err == nil {
				argTokens[event.ID.String()] = tokenaccounting.EstimateText(string(encoded)).Tokens
			}
		case *domain.ToolResultEventData:
			id := event.ID.String()
			estimate := tokenaccounting.EstimateText(resultPayload(payload))
			resultTokens[id], tokenBases[id], residencySegments[id] = estimate.Tokens, estimate.Basis, segment
			resident = append(resident, residentResult{id: id, at: requests})
		}
	}
	closeSegment()
	return argTokens, resultTokens, tokenBases, residencyRequests, residencySegments
}

// resultPayload is the text a tool result put in the agent's context: its
// output (else, for harnesses that return structured content such as Codex's
// exec, its text blocks) and its error. Result tokens, and so the context tax
// and output weight, are estimated from it. A command's own output size
// (Fact.OutputBytes) can exceed it when the harness cut the output first; the
// tokens are what the model read, the bytes what was printed.
func resultPayload(result *domain.ToolResultEventData) string {
	text := result.Output
	if text == "" {
		for _, block := range result.OutputBlocks {
			text += block.Text
		}
	}
	if result.Error != "" {
		text += "\n" + result.Error
	}
	return text
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

// ProjectRun folds a terminal run and its retained cost events into one
// durable throughput fact. Cost events are additive because runners can emit
// usage for several turns; cache tokens are usage and therefore included.
func ProjectRun(run *domain.Run, events []*domain.RunEvent, projectedAt time.Time) RunFact {
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
	// The run's one usage account: token aggregates, incurred attribution and
	// time-accounting buckets all read the same per-event contributions.
	usage := runsignal.AccountUsage(events)
	providerTurns := map[int]int{}
	incurredByCall, noPrecedingToolCallTokens, hasEligibleUsage := attributeIncurred(events, usage, preambleInjectedTokens, preambleFixedTokens)
	var incurredTokens int64
	costSource := "unknown"
	chargeReason := ""
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
		if costSource == "unknown" || costSource == "" {
			costSource = string(charge.Basis)
		}
		if charge.ChargeReason != "" && chargeReason == "" {
			chargeReason = charge.ChargeReason
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
	for index, event := range events {
		if event == nil {
			continue
		}
		if report, ok := event.Data.(*domain.UsageEventData); ok {
			// A repeated report cannot raise the maximum, so provider turns
			// read every report; quantities and charges read contributions.
			if report.Turns > providerTurns[usage.Invocations[index]] {
				providerTurns[usage.Invocations[index]] = report.Turns
			}
			if usage.Tokens(index) > 0 {
				addCharge(report.Charge)
			}
		}
		if charge, ok := event.Data.(*domain.ChargeEventData); ok {
			addCharge(charge)
		}
		if _, ok := event.Data.(*domain.ToolCallEventData); ok {
			toolCalls++
		}
	}
	tokens = usage.TotalTokens()
	inputTokens, outputTokens = int64(usage.Total.InputTokens), int64(usage.Total.OutputTokens)
	cacheReadTokens, cacheCreationTokens = int64(usage.Total.CacheReadTokens), int64(usage.Total.CacheCreationTokens)
	for _, count := range providerTurns {
		turns += int64(count)
	}
	// A compatibility summary can count assistant commentary messages. Use it
	// only when the provider supplied no invocation turn accounting.
	if turns == 0 && run.Summary != nil {
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
	return RunFact{RunID: run.ID.String(), GoalID: run.GoalID, GoalStatus: "", OccurredAt: occurredAt, CreatedAt: run.CreatedAt, StartedAt: run.StartedAt, EndedAt: run.EndedAt, DurationMS: duration, Status: string(run.Status), ProfileID: profileID, RunnerType: runnerType, Model: model, Tag: tag, WorkloadKind: workloadKind, WorkloadKey: workloadKey, WorkloadInstance: run.Workload.Instance, TotalCostUSD: cost, CostSource: costSource, ChargeReason: chargeReason, InputCostUSD: inputCost, OutputCostUSD: outputCost, CacheReadCostUSD: cacheReadCost, CacheCreationCostUSD: cacheCreationCost, TotalTokens: tokens, InputTokens: inputTokens, OutputTokens: outputTokens, CacheReadTokens: cacheReadTokens, CacheCreationTokens: cacheCreationTokens, Turns: turns, ToolCalls: toolCalls, TotalChargeMicroUSD: totalChargeMicroUSD, MeteredChargeMicroUSD: meteredChargeMicroUSD, UnpricedTokenCount: unpricedTokenCount, PreambleInjectedTokens: preambleInjectedTokens, PreambleFixedTokens: preambleFixedTokens, PreambleTokenBasis: preambleBasis, UnattributedTokens: unattributedTokens, UnattributedReason: unattributedReason, TimeAccounting: runsignal.DeriveTimeAccounting(events, run.StartedAt, run.EndedAt), CostTimeBasis: costBasis, TimeBasis: eventTimeBasis, ProjectedAt: projectedAt}
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
