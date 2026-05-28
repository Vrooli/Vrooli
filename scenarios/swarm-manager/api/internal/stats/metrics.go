package stats

import (
	"sort"
	"time"
)

// MinSampleMeaningful is the threshold below which a metric renders as
// insufficient-data rather than a value. Five is the smallest sample size at
// which summary statistics carry any signal across a short-horizon workflow.
const MinSampleMeaningful = 5

func (s *aggregateState) buildResponse() StatsResponse {
	now := s.now()
	return StatsResponse{
		GeneratedAt: now,
		EventCount:  s.totalEvents,
		History:     s.buildHistory(now),
		Throughput:  s.buildThroughput(now),
		Timing:      s.buildTiming(),
		Scope:       s.buildScope(),
		Blocking:    s.buildBlocking(),
		Agent:       s.buildAgent(),
		Dashboard:   s.buildDashboard(now),
		Review:      s.buildReview(),
		Mode:        s.buildMode(),
		Session:     s.buildSession(),
		Records:     s.buildRecords(now),
	}
}

// buildRecords folds the per-record counters into the API response shape.
// Counters initialize to zero, so this is safe to call on scenarios that
// pre-date the record events — empty maps and zero numerators stay JSON-friendly.
func (s *aggregateState) buildRecords(now time.Time) RecordStats {
	last7 := 0
	last30 := 0
	cutoff7 := now.Add(-7 * 24 * time.Hour)
	cutoff30 := now.Add(-30 * 24 * time.Hour)
	for _, t := range s.recordCreatedAt {
		if t.After(cutoff7) {
			last7++
		}
		if t.After(cutoff30) {
			last30++
		}
	}
	without := s.recordTotal - s.recordsWithBacklogRef
	if without < 0 {
		without = 0
	}
	var regression float64
	if s.recordTotal > 0 {
		regression = float64(s.recordsSupersedeCount) / float64(s.recordTotal)
	}
	// Copy the maps so consumers can't mutate the engine's running state.
	byKind := make(map[string]int, len(s.recordsByKind))
	for k, v := range s.recordsByKind {
		byKind[k] = v
	}
	byScenario := make(map[string]int, len(s.recordsByScenario))
	for k, v := range s.recordsByScenario {
		byScenario[k] = v
	}
	return RecordStats{
		TotalRecords:      s.recordTotal,
		CreatedLast7Days:  last7,
		CreatedLast30Days: last30,
		ByKind:            byKind,
		ByScenario:        byScenario,
		WithBacklogRef:    s.recordsWithBacklogRef,
		WithoutBacklogRef: without,
		Stubs:             s.recordsStubs,
		SupersedeCount:    s.recordsSupersedeCount,
		RegressionRate:    regression,
	}
}

func (s *aggregateState) buildSession() SessionStats {
	byKind := make(map[string]int)
	byStatus := make(map[string]int)
	active := 0
	totalMessages := 0
	failed := 0
	terminal := 0

	for sessionID, kind := range s.sessionKind {
		if kind == "" {
			kind = "unknown"
		}
		byKind[kind]++
		status := s.sessionStatus[sessionID]
		if status == "" {
			status = "unknown"
		}
		byStatus[status]++
		if isActiveSessionStatus(status) {
			active++
		}
		if isTerminalSessionStatus(status) {
			terminal++
			if status == "failed" {
				failed++
			}
		}
		totalMessages += s.sessionMessageCount[sessionID]
	}

	var avgMessages float64
	if len(s.sessionKind) > 0 {
		avgMessages = float64(totalMessages) / float64(len(s.sessionKind))
	}

	var failedRate float64
	if terminal > 0 {
		failedRate = float64(failed) / float64(terminal)
	}

	return SessionStats{
		TotalSessions:                     len(s.sessionKind),
		ActiveSessions:                    active,
		SessionsByKind:                    byKind,
		SessionsByStatus:                  byStatus,
		ProposalCreatedByKind:             cloneIntMap(s.sessionProposalCreatedByKind),
		ProposalAppliedByKind:             cloneIntMap(s.sessionProposalAppliedByKind),
		ProposalApplyRateByKind:           buildRateMap(s.sessionProposalAppliedByKind, s.sessionProposalCreatedByKind),
		ArtifactsCreatedByKind:            cloneIntMap(s.sessionArtifactsCreatedByKind),
		ArtifactsByType:                   cloneIntMap(s.sessionArtifactsByType),
		AverageMessagesPerSession:         avgMessages,
		AverageTimeToFirstProposalSeconds: avgFloat(s.sessionFirstProposalSeconds),
		FirstProposalSampleSize:           len(s.sessionFirstProposalSeconds),
		FailedSessionRate:                 failedRate,
		FailedSessionSampleSize:           terminal,
		SessionCreatedBacklogItems:        s.sessionCreatedBacklogItems,
		SessionCreatedInitiatives:         s.sessionCreatedInitiatives,
	}
}

func isActiveSessionStatus(status string) bool {
	switch status {
	case "starting", "running", "waiting_for_user", "proposal_ready", "applying":
		return true
	default:
		return false
	}
}

func isTerminalSessionStatus(status string) bool {
	switch status {
	case "complete", "failed", "canceled":
		return true
	default:
		return false
	}
}

func (s *aggregateState) buildMode() ModeStats {
	usage := make(map[string]int)
	for name := range s.initiativeCreated {
		mode := s.initiativeMode[name]
		if mode == "" {
			mode = "item-level"
		}
		usage[mode]++
	}

	avgDurations := make(map[string]map[string]float64, len(s.modeDurationSums))
	for mode, phases := range s.modeDurationSums {
		avgDurations[mode] = make(map[string]float64, len(phases))
		for phase, total := range phases {
			if count := s.modeDurationCounts[mode][phase]; count > 0 {
				avgDurations[mode][phase] = total / float64(count)
			}
		}
	}

	avgRunsPerScope := make(map[string]float64, len(s.modeCompletedScopes))
	for mode, scopes := range s.modeCompletedScopes {
		if len(scopes) == 0 {
			continue
		}
		totalRuns := 0
		for _, count := range s.modePhaseRuns[mode] {
			totalRuns += count
		}
		avgRunsPerScope[mode] = float64(totalRuns) / float64(len(scopes))
	}

	return ModeStats{
		UsageByMode:              usage,
		ModeSwitchCount:          s.modeSwitchCount,
		PhaseRunsByMode:          cloneNestedInt(s.modePhaseRuns),
		CompletedByMode:          cloneIntMap(s.modeCompleted),
		FailedByMode:             cloneIntMap(s.modeFailed),
		CanceledByMode:           cloneIntMap(s.modeCanceled),
		ReplanRateByMode:         buildRateMap(s.modeReplanNumerator, s.modeReplanDenominator),
		AcceptanceRateByMode:     buildRateMap(s.modeAcceptanceNumerator, s.modeAcceptanceDenom),
		AvgPhaseDurationSeconds:  avgDurations,
		AvgRunsPerCompletedScope: avgRunsPerScope,
		BacklogSyncByMode:        cloneBacklogSyncMap(s.modeBacklogSync),
		UsageByProfile:           cloneIntMap(s.modeProfileUsage),
		PhaseRunsByProfile:       cloneNestedInt(s.modeProfilePhaseRuns),
		PhaseRunsByLane:          cloneIntMap(s.modePhaseRunsByLane),
	}
}

func buildRateMap(numerators, denominators map[string]int) map[string]KindRate {
	result := make(map[string]KindRate, len(denominators))
	for key, denom := range denominators {
		var rate float64
		if denom > 0 {
			rate = float64(numerators[key]) / float64(denom)
		}
		result[key] = KindRate{Rate: rate, SampleSize: denom}
	}
	return result
}

func cloneIntMap(in map[string]int) map[string]int {
	out := make(map[string]int, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneNestedInt(in map[string]map[string]int) map[string]map[string]int {
	out := make(map[string]map[string]int, len(in))
	for outer, inner := range in {
		out[outer] = cloneIntMap(inner)
	}
	return out
}

func cloneBacklogSyncMap(in map[string]*BacklogSyncStats) map[string]BacklogSyncStats {
	out := make(map[string]BacklogSyncStats, len(in))
	for key, value := range in {
		if value != nil {
			out[key] = *value
		}
	}
	return out
}

func (s *aggregateState) buildHistory(now time.Time) HistoryWindow {
	if !s.earliestEventRecorded {
		return HistoryWindow{MinSampleMeaningful: MinSampleMeaningful}
	}
	days := now.Sub(s.earliestEventAt).Hours() / 24.0
	if days < 0 {
		days = 0
	}
	return HistoryWindow{
		EarliestEventAt:     s.earliestEventAt,
		HistoryDays:         days,
		HasHistory:          true,
		MinSampleMeaningful: MinSampleMeaningful,
	}
}

func (s *aggregateState) buildReview() ReviewStats {
	var avgEvidence float64
	if len(s.reviewEvidenceCounts) > 0 {
		total := 0
		for _, c := range s.reviewEvidenceCounts {
			total += c
		}
		avgEvidence = float64(total) / float64(len(s.reviewEvidenceCounts))
	}

	totalEvidence := 0
	for _, c := range s.reviewEvidenceCounts {
		totalEvidence += c
	}

	var verificationRate float64
	if totalEvidence > 0 {
		verificationRate = float64(s.reviewEvidenceVerified) / float64(totalEvidence)
	}

	var requestMoreRate float64
	if s.reviewRoundsCompleted > 0 {
		requestMoreRate = float64(s.reviewRequestsCreated) / float64(s.reviewRoundsCompleted)
	}

	var avgDuration float64
	if len(s.reviewDurations) > 0 {
		total := 0.0
		for _, d := range s.reviewDurations {
			total += d
		}
		avgDuration = total / float64(len(s.reviewDurations))
	}

	return ReviewStats{
		RoundsCompleted:         s.reviewRoundsCompleted,
		AverageEvidencePerRound: avgEvidence,
		VerificationRate:        verificationRate,
		RequestMoreRate:         requestMoreRate,
		AverageReviewDuration:   avgDuration,
	}
}

func (s *aggregateState) buildThroughput(now time.Time) ThroughputStats {
	d7 := now.Add(-7 * 24 * time.Hour)
	d30 := now.Add(-30 * 24 * time.Hour)

	created7 := countAfter(s.createdEvents, d7)
	created30 := countAfter(s.createdEvents, d30)
	completed7 := countAfter(s.completedEvents, d7)
	completed30 := countAfter(s.completedEvents, d30)

	return ThroughputStats{
		CompletedLast7Days:  completed7,
		CompletedLast30Days: completed30,
		CreatedLast7Days:    created7,
		CreatedLast30Days:   created30,
		NetDelta7Days:       created7 - completed7,
		NetDelta30Days:      created30 - completed30,
	}
}

func (s *aggregateState) buildTiming() TimingStats {
	return TimingStats{
		AvgLeadTimeHours:         avgFloat(s.leadTimesH),
		MedianLeadTimeHours:      medianFloat(s.leadTimesH),
		LeadTimeSampleSize:       len(s.leadTimesH),
		AvgExecutionMinutes:      avgFloat(s.execDurations),
		MedianExecutionMinutes:   medianFloat(s.execDurations),
		ExecutionDurationSamples: len(s.execDurations),
	}
}

func (s *aggregateState) buildScope() ScopeStats {
	var inits []InitiativeHealth
	for name := range s.initiativeCreated {
		items := s.initiativeItems[name]
		ih := InitiativeHealth{
			Name:  name,
			Total: len(items),
		}
		for itemID := range items {
			switch s.itemStatus[itemID] {
			case "completed":
				ih.Completed++
			case "in_progress", "queued":
				ih.InProgress++
			}
			if _, blocked := s.blockedItems[itemID]; blocked {
				ih.Blocked++
			}
		}
		initial := s.initiativeInitial[name]
		if initial > 0 {
			ih.ScopeCreep = float64(len(items)-initial) / float64(initial)
		}
		inits = append(inits, ih)
	}
	sort.Slice(inits, func(i, j int) bool { return inits[i].Name < inits[j].Name })

	return ScopeStats{
		Initiatives: inits,
	}
}

func (s *aggregateState) buildBlocking() BlockingStats {
	blocked := len(s.blockedItems)
	total := len(s.currentBacklog)
	var ratio float64
	if total > 0 {
		ratio = float64(blocked) / float64(total)
	}

	// Top reasons sorted by count descending.
	var reasons []ReasonCount
	for reason, count := range s.blockReasons {
		reasons = append(reasons, ReasonCount{Reason: reason, Count: count})
	}
	sort.Slice(reasons, func(i, j int) bool { return reasons[i].Count > reasons[j].Count })
	if len(reasons) > 10 {
		reasons = reasons[:10]
	}

	return BlockingStats{
		CurrentlyBlocked: blocked,
		BlockedRatio:     ratio,
		TopReasons:       reasons,
		AvgBlockHours:    avgFloat(s.blockDurations),
	}
}

func (s *aggregateState) buildAgent() AgentStats {
	completed, failed, manuallyAccepted := s.countExecOutcomes()

	var successRate, failureRate, manualAcceptRate float64
	finished := completed + failed
	if finished > 0 {
		successRate = float64(completed) / float64(finished)
		failureRate = float64(failed) / float64(finished)
		manualAcceptRate = float64(manuallyAccepted) / float64(finished)
	}

	var followUpRate float64
	fixupCount := 0
	for _, had := range s.execHasFixup {
		if had {
			fixupCount++
		}
	}
	if completed > 0 {
		followUpRate = float64(fixupCount) / float64(completed)
	}

	var avgRounds float64
	if len(s.workshopRounds) > 0 {
		total := 0
		for _, r := range s.workshopRounds {
			total += r
		}
		avgRounds = float64(total) / float64(len(s.workshopRounds))
	}

	var (
		recAcceptanceRate    float64
		freeformOverrideRate float64
	)
	if s.decisionItemsAnswered > 0 {
		recAcceptanceRate = float64(s.decisionItemsRecommendedChosen) / float64(s.decisionItemsAnswered)
		freeformOverrideRate = float64(s.decisionItemsFreeformChosen) / float64(s.decisionItemsAnswered)
	}

	byKind := make(map[string]KindRate, len(s.decisionByKind))
	for kind, c := range s.decisionByKind {
		var rate float64
		if c.itemsAnswered > 0 {
			rate = float64(c.itemsRecommendedChosen) / float64(c.itemsAnswered)
		}
		byKind[kind] = KindRate{
			Rate:       rate,
			SampleSize: c.itemsAnswered,
		}
	}

	return AgentStats{
		TotalExecutions:                    s.execTotal,
		CompletedCount:                     completed,
		FailedCount:                        failed,
		ManuallyAcceptedCount:              manuallyAccepted,
		SuccessRate:                        successRate,
		FailureRate:                        failureRate,
		ManualAcceptRate:                   manualAcceptRate,
		FollowUpRate:                       followUpRate,
		AvgExecutionMinutes:                avgFloat(s.execDurations),
		AvgWorkshopRounds:                  avgRounds,
		SuccessRateSampleSize:              finished,
		ExecutionDurationSamples:           len(s.execDurations),
		WorkshopRoundsSampleSize:           len(s.workshopRounds),
		RecommendationAcceptanceRate:       recAcceptanceRate,
		RecommendationAcceptanceSampleSize: s.decisionItemsAnswered,
		FreeformOverrideRate:               freeformOverrideRate,
		DecisionItemsTotal:                 s.decisionItemsTotal,
		DecisionItemsAnswered:              s.decisionItemsAnswered,
		RecommendationAcceptanceByKind:     byKind,
	}
}

func (s *aggregateState) buildDashboard(now time.Time) DashboardStats {
	// Velocity trend: weekly completions over trailing 8 weeks.
	var trend []VelocityPoint
	for i := 7; i >= 0; i-- {
		weekStart := now.Add(-time.Duration(i*7*24) * time.Hour).Truncate(24 * time.Hour)
		weekEnd := weekStart.Add(7 * 24 * time.Hour)
		count := 0
		for _, t := range s.completedEvents {
			if !t.Before(weekStart) && t.Before(weekEnd) {
				count++
			}
		}
		trend = append(trend, VelocityPoint{
			WeekStart: weekStart.Format("2006-01-02"),
			Completed: count,
		})
	}

	// Avg velocity from last 4 full weeks (indices 4-7 in the 9 weeks).
	avgVelocity := 0.0
	if len(trend) >= 5 {
		total := 0
		weeks := 0
		for i := len(trend) - 5; i < len(trend)-1; i++ {
			total += trend[i].Completed
			weeks++
		}
		if weeks > 0 {
			avgVelocity = float64(total) / float64(weeks)
		}
	}

	var weeksRemaining float64
	if avgVelocity > 0 {
		weeksRemaining = float64(len(s.currentBacklog)) / avgVelocity
	}

	// weeksCovered counts the number of *non-zero* trend weeks. The Dashboard
	// uses this to decide whether the "Est. Remaining" pill has enough history
	// to be trustworthy.
	weeksCovered := 0
	for _, p := range trend {
		if p.Completed > 0 {
			weeksCovered++
		}
	}

	return DashboardStats{
		TotalBacklogSize:        len(s.currentBacklog),
		TotalCompletedAllTime:   s.completedAllTime,
		VelocityTrend:           trend,
		EstimatedWeeksRemaining: weeksRemaining,
		VelocityWeeksCovered:    weeksCovered,
	}
}

// --- helpers ---

func countAfter(timestamps []time.Time, after time.Time) int {
	count := 0
	for _, t := range timestamps {
		if t.After(after) {
			count++
		}
	}
	return count
}

func avgFloat(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func medianFloat(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}
