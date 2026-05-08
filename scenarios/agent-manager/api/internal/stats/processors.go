// Processor implementations and registration for every typed-operational
// event. See registry.go for the dispatch contract.

package stats

import (
	"agent-manager/internal/domain"
	"agent-manager/internal/eventlog"
)

func init() {
	RegisterProcessor(domain.EventTypeRunnerFallbackAttempted, 1, processRunnerFallbackAttempted)
	RegisterProcessor(domain.EventTypeRunnerFallbackExhausted, 1, processRunnerFallbackExhausted)
	RegisterProcessor(domain.EventTypeModelFallbackAttempted, 1, processModelFallbackAttempted)
	RegisterProcessor(domain.EventTypeModelFallbackExhausted, 1, processModelFallbackExhausted)
	RegisterProcessor(domain.EventTypeModelHealthTransition, 1, processModelHealthTransition)
	RegisterProcessor(domain.EventTypeRunnerHealthTransition, 1, processRunnerHealthTransition)
	RegisterProcessor(domain.EventTypeSandboxOperation, 1, processSandboxOperation)
	RegisterProcessor(domain.EventTypeHeartbeatMiss, 1, processHeartbeatMiss)
	RegisterProcessor(domain.EventTypeCheckpointFailure, 1, processCheckpointFailure)
	RegisterProcessor(domain.EventTypeRetryAttempt, 1, processRetryAttempt)
}

func processRunnerFallbackAttempted(s *aggregateState, rec eventlog.Record) {
	p, ok := rec.Payload.(*eventlog.RunnerFallbackAttemptedPayload)
	if !ok {
		return
	}
	s.runnerFallbackAttempts++
	s.runnerByReason[p.Reason]++
	s.runnerPair[fallbackPairKey{From: p.From, To: p.To, Reason: p.Reason}]++
	if p.AttemptNo > 0 {
		s.runnerChainDepth[p.AttemptNo]++
	}
}

func processRunnerFallbackExhausted(s *aggregateState, rec eventlog.Record) {
	p, ok := rec.Payload.(*eventlog.RunnerFallbackExhaustedPayload)
	if !ok {
		return
	}
	s.runnerExhausted++
	s.runnerByReason[p.LastReason]++
}

func processModelFallbackAttempted(s *aggregateState, rec eventlog.Record) {
	p, ok := rec.Payload.(*eventlog.ModelFallbackAttemptedPayload)
	if !ok {
		return
	}
	s.modelFallbackAttempts++
	s.modelByReason[p.Reason]++
	s.modelPair[fallbackPairKey{From: p.From, To: p.To, Reason: p.Reason}]++
	if p.AttemptNo > 0 {
		s.modelChainDepth[p.AttemptNo]++
	}
}

func processModelFallbackExhausted(s *aggregateState, rec eventlog.Record) {
	p, ok := rec.Payload.(*eventlog.ModelFallbackExhaustedPayload)
	if !ok {
		return
	}
	s.modelExhausted++
	s.modelByReason[p.LastReason]++
	if p.Preset != "" {
		s.modelByPreset[p.Preset]++
	}
}

func processModelHealthTransition(s *aggregateState, rec eventlog.Record) {
	p, ok := rec.Payload.(*eventlog.ModelHealthTransitionPayload)
	if !ok {
		return
	}
	key := modelKey{Runner: p.Runner, Model: p.Model}
	s.modelHealth[key] = ModelHealthEntry{
		Runner:              p.Runner,
		Model:               p.Model,
		Status:              p.ToStatus,
		Reason:              p.Reason,
		Message:             p.Message,
		ObservedAt:          rec.Timestamp,
		TransitionsObserved: s.modelHealth[key].TransitionsObserved + 1,
	}
}

func processRunnerHealthTransition(s *aggregateState, rec eventlog.Record) {
	p, ok := rec.Payload.(*eventlog.RunnerHealthTransitionPayload)
	if !ok {
		return
	}
	cur := s.runnerHealth[p.Runner]
	s.runnerHealth[p.Runner] = RunnerHealthEntry{
		Runner:              p.Runner,
		Status:              p.ToStatus,
		Reason:              p.Reason,
		ObservedAt:          rec.Timestamp,
		TransitionsObserved: cur.TransitionsObserved + 1,
	}
}

func processSandboxOperation(s *aggregateState, rec eventlog.Record) {
	p, ok := rec.Payload.(*eventlog.SandboxOperationPayload)
	if !ok {
		return
	}
	s.sandboxTotal++
	bucket := s.sandboxByOp[p.Operation]
	bucket.Total++
	if p.Success {
		bucket.Success++
		s.sandboxSuccess++
	} else {
		bucket.Failure++
	}
	s.sandboxByOp[p.Operation] = bucket
	if p.DurationMS > 0 {
		s.sandboxDurationSum += float64(p.DurationMS)
		s.sandboxDurationCount++
	}
}

func processHeartbeatMiss(s *aggregateState, rec eventlog.Record) {
	p, ok := rec.Payload.(*eventlog.HeartbeatMissPayload)
	if !ok {
		return
	}
	s.heartbeatMisses++
	s.heartbeatByTarget[p.Target]++
}

func processCheckpointFailure(s *aggregateState, rec eventlog.Record) {
	p, ok := rec.Payload.(*eventlog.CheckpointFailurePayload)
	if !ok {
		return
	}
	s.checkpointFailures++
	if p.Step != "" {
		s.checkpointByStep[p.Step]++
	}
	if p.Phase != "" {
		s.checkpointByPhase[p.Phase]++
	}
}

func processRetryAttempt(s *aggregateState, rec eventlog.Record) {
	p, ok := rec.Payload.(*eventlog.RetryAttemptPayload)
	if !ok {
		return
	}
	s.retryAttempts++
	if p.Operation != "" {
		s.retryByOperation[p.Operation]++
	}
	if p.Reason != "" {
		s.retryByReason[p.Reason]++
	}
}
