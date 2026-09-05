// Response builders.
//
// Translates the engine's flat aggregateState into the public response
// types defined in types.go. Public Get* methods on Engine call into
// these builders under the read lock, so each response is a consistent
// snapshot.

package stats

import (
	"sort"
	"time"
)

// GetSummary returns the bundled top-level snapshot used by the
// dashboard's single-call read.
func (e *Engine) GetSummary() Summary {
	e.mu.RLock()
	defer e.mu.RUnlock()
	now := e.now()
	hist := e.state.historyWindow(now)
	return Summary{
		GeneratedAt: now,
		History:     hist,
		EventCount:  e.state.totalEvents,
		Fallback:    e.state.buildFallbackInsights(now, hist),
		Health:      e.state.buildHealthSummary(now, hist),
		Sandbox:     e.state.buildSandboxSummary(now, hist),
		Heartbeat:   e.state.buildHeartbeatSummary(now, hist),
		Checkpoint:  e.state.buildCheckpointSummary(now, hist),
		Retry:       e.state.buildRetrySummary(now, hist),
	}
}

// GetFallback returns the standalone fallback-insights view.
func (e *Engine) GetFallback() FallbackInsights {
	e.mu.RLock()
	defer e.mu.RUnlock()
	now := e.now()
	return e.state.buildFallbackInsights(now, e.state.historyWindow(now))
}

// GetHealth returns the engine-derived health summary. (Authoritative
// current-status reads come from the persisted health.Store; this view
// is event-derived and useful for "transitions in window" math.)
func (e *Engine) GetHealth() HealthSummary {
	e.mu.RLock()
	defer e.mu.RUnlock()
	now := e.now()
	return e.state.buildHealthSummary(now, e.state.historyWindow(now))
}

// GetSandbox returns the sandbox-operation summary.
func (e *Engine) GetSandbox() SandboxSummary {
	e.mu.RLock()
	defer e.mu.RUnlock()
	now := e.now()
	return e.state.buildSandboxSummary(now, e.state.historyWindow(now))
}

// GetHeartbeat returns the heartbeat-miss summary.
func (e *Engine) GetHeartbeat() HeartbeatSummary {
	e.mu.RLock()
	defer e.mu.RUnlock()
	now := e.now()
	return e.state.buildHeartbeatSummary(now, e.state.historyWindow(now))
}

// GetCheckpoint returns the checkpoint-failure summary.
func (e *Engine) GetCheckpoint() CheckpointSummary {
	e.mu.RLock()
	defer e.mu.RUnlock()
	now := e.now()
	return e.state.buildCheckpointSummary(now, e.state.historyWindow(now))
}

// GetRetry returns the retry-attempt summary.
func (e *Engine) GetRetry() RetrySummary {
	e.mu.RLock()
	defer e.mu.RUnlock()
	now := e.now()
	return e.state.buildRetrySummary(now, e.state.historyWindow(now))
}

// ---- builders on aggregateState ----

func (s *aggregateState) historyWindow(now time.Time) HistoryWindow {
	if !s.earliestRecorded {
		return HistoryWindow{MinSampleMeaningful: MinSampleMeaningful}
	}
	span := now.Sub(s.earliestEventAt)
	if span < 0 {
		span = 0
	}
	return HistoryWindow{
		EarliestEventAt:     s.earliestEventAt,
		HistoryDays:         span.Hours() / 24,
		HasHistory:          true,
		MinSampleMeaningful: MinSampleMeaningful,
	}
}

func (s *aggregateState) buildFallbackInsights(now time.Time, hist HistoryWindow) FallbackInsights {
	return FallbackInsights{
		GeneratedAt:           now,
		History:               hist,
		EventCount:            s.totalEvents,
		RunnerAttempts:        s.runnerFallbackAttempts,
		RunnerExhausted:       s.runnerExhausted,
		RunnerByReason:        copyStrIntMap(s.runnerByReason),
		RunnerByPair:          pairsToList(s.runnerPair),
		RunnerChainDepth:      copyIntIntMap(s.runnerChainDepth),
		ModelAttempts:         s.modelFallbackAttempts,
		ModelExhausted:        s.modelExhausted,
		ModelByReason:         copyStrIntMap(s.modelByReason),
		ModelByPair:           pairsToList(s.modelPair),
		ModelChainDepth:       copyIntIntMap(s.modelChainDepth),
		ModelByPreset:         copyStrIntMap(s.modelByPreset),
		PolicyCandidateEvents: s.policyCandidateEvents,
		PolicyByOutcome:       copyStrIntMap(s.policyByOutcome),
		PolicyByFailureClass:  copyStrIntMap(s.policyByFailureClass),
	}
}

func (s *aggregateState) buildHealthSummary(now time.Time, hist HistoryWindow) HealthSummary {
	models := make([]ModelHealthEntry, 0, len(s.modelHealth))
	for _, v := range s.modelHealth {
		models = append(models, v)
	}
	sort.Slice(models, func(i, j int) bool {
		if models[i].Runner != models[j].Runner {
			return models[i].Runner < models[j].Runner
		}
		return models[i].Model < models[j].Model
	})

	runners := make([]RunnerHealthEntry, 0, len(s.runnerHealth))
	for _, v := range s.runnerHealth {
		runners = append(runners, v)
	}
	sort.Slice(runners, func(i, j int) bool { return runners[i].Runner < runners[j].Runner })

	hourAgo := now.Add(-time.Hour)
	failingLastHour := make([]ModelHealthEntry, 0)
	for _, m := range models {
		if m.Status != "ok" && m.ObservedAt.After(hourAgo) {
			failingLastHour = append(failingLastHour, m)
		}
	}
	return HealthSummary{
		GeneratedAt:     now,
		History:         hist,
		Models:          models,
		Runners:         runners,
		FailingLastHour: failingLastHour,
	}
}

func (s *aggregateState) buildSandboxSummary(now time.Time, hist HistoryWindow) SandboxSummary {
	successRate := 0.0
	if s.sandboxTotal > 0 {
		successRate = float64(s.sandboxSuccess) / float64(s.sandboxTotal)
	}
	avgDur := 0.0
	if s.sandboxDurationCount > 0 {
		avgDur = s.sandboxDurationSum / float64(s.sandboxDurationCount)
	}
	byOp := make(map[string]OperationCount, len(s.sandboxByOp))
	for k, v := range s.sandboxByOp {
		byOp[k] = v
	}
	return SandboxSummary{
		GeneratedAt:     now,
		History:         hist,
		TotalOps:        s.sandboxTotal,
		SuccessRate:     successRate,
		SampleSize:      s.sandboxTotal,
		ByOperation:     byOp,
		AvgDurationMs:   avgDur,
		DurationSamples: s.sandboxDurationCount,
	}
}

func (s *aggregateState) buildHeartbeatSummary(now time.Time, hist HistoryWindow) HeartbeatSummary {
	return HeartbeatSummary{
		GeneratedAt: now,
		History:     hist,
		TotalMisses: s.heartbeatMisses,
		ByTarget:    copyStrIntMap(s.heartbeatByTarget),
	}
}

func (s *aggregateState) buildCheckpointSummary(now time.Time, hist HistoryWindow) CheckpointSummary {
	return CheckpointSummary{
		GeneratedAt:   now,
		History:       hist,
		TotalFailures: s.checkpointFailures,
		ByStep:        copyStrIntMap(s.checkpointByStep),
		ByPhase:       copyStrIntMap(s.checkpointByPhase),
	}
}

func (s *aggregateState) buildRetrySummary(now time.Time, hist HistoryWindow) RetrySummary {
	return RetrySummary{
		GeneratedAt:   now,
		History:       hist,
		TotalAttempts: s.retryAttempts,
		ByOperation:   copyStrIntMap(s.retryByOperation),
		ByReason:      copyStrIntMap(s.retryByReason),
	}
}

// ---- helpers ----

func copyStrIntMap(in map[string]int) map[string]int {
	if len(in) == 0 {
		return map[string]int{}
	}
	out := make(map[string]int, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func copyIntIntMap(in map[int]int) map[int]int {
	if len(in) == 0 {
		return map[int]int{}
	}
	out := make(map[int]int, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func pairsToList(in map[fallbackPairKey]int) []FallbackPair {
	out := make([]FallbackPair, 0, len(in))
	for k, v := range in {
		out = append(out, FallbackPair{From: k.From, To: k.To, Reason: k.Reason, Count: v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		if out[i].From != out[j].From {
			return out[i].From < out[j].From
		}
		return out[i].To < out[j].To
	})
	return out
}
