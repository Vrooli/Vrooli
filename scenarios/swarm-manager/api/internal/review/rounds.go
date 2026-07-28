package review

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"swarm-manager/internal/idgen"
	"swarm-manager/internal/transitionrunner"
)

// ListRounds returns all review evidence rounds for a backlog item.
// For any round still in gathering state, it checks the agent-manager run
// state inline and updates the round if the run has completed.
func (s *Service) ListRounds(kind, name string) ([]Round, error) {
	itemDir := s.resolveItemDir(kind, name)
	rounds, err := LoadRounds(itemDir)
	if err != nil {
		return nil, err
	}

	if s.inspector == nil {
		return rounds, nil
	}

	for i := range rounds {
		round := &rounds[i]
		if round.Status != RoundStatusGathering || round.RunID == "" {
			continue
		}
		// Runner-owned rounds are finalized by the operation runner's completion
		// bridge (commit-review-round), not this poll — defer so the two never
		// race to drive the same round.
		if round.RunnerOwned() || round.WorkflowOwned() {
			continue
		}
		state, stateErr := s.inspector.GetRunState(context.Background(), round.RunID)
		if stateErr != nil {
			continue
		}
		round.CurrentRunStatus = normalizeLiveRunStatus(state.Status)
		if mapRunStatusToRoundStatus(state.Status) == "" {
			continue
		}
		*round = finalizeRoundFromRunState(*round, state)
		_ = SaveRound(itemDir, *round)

		// Remove from active tracking if present.
		s.mu.Lock()
		delete(s.activeRounds, round.RunID)
		s.mu.Unlock()
	}

	return rounds, nil
}

// GetRound returns a specific review round by number.
func (s *Service) GetRound(kind, name string, roundNum int) (*Round, error) {
	itemDir := s.resolveItemDir(kind, name)
	return LoadRound(itemDir, roundNum)
}

// VerifyEvidence toggles the verified flag on an evidence item.
func (s *Service) VerifyEvidence(kind, name string, roundNum int, evidenceID string, verified bool, executionID string) error {
	itemDir := s.resolveItemDir(kind, name)
	round, err := LoadRound(itemDir, roundNum)
	if err != nil {
		return fmt.Errorf("load round %d: %w", roundNum, err)
	}
	if round == nil {
		return fmt.Errorf("round %d not found", roundNum)
	}

	found := false
	for i := range round.Evidence {
		if round.Evidence[i].ID == evidenceID {
			round.Evidence[i].Verified = verified
			if verified {
				round.Evidence[i].VerifiedAt = time.Now().UTC().Format(time.RFC3339)
			} else {
				round.Evidence[i].VerifiedAt = ""
			}
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("evidence item %s not found in round %d", evidenceID, roundNum)
	}

	if err := SaveRound(itemDir, *round); err != nil {
		return fmt.Errorf("save round: %w", err)
	}

	if s.eventLogger != nil && executionID != "" {
		s.eventLogger.EmitReviewEvidenceVerified(executionID, evidenceID)
	}
	return nil
}

// RequestMoreEvidence creates a new request thread and starts its declared
// evidence-request workflow when workflow support is available.
func (s *Service) RequestMoreEvidence(ctx context.Context, kind, name string, roundNum int, message string, evidenceID string) (string, error) {
	itemDir := s.resolveItemDir(kind, name)
	round, err := LoadRound(itemDir, roundNum)
	if err != nil {
		return "", fmt.Errorf("load round %d: %w", roundNum, err)
	}
	if round == nil {
		return "", fmt.Errorf("round %d not found", roundNum)
	}

	threadID := "rt-" + idgen.Generate()
	thread := RequestThread{
		ID:         threadID,
		EvidenceID: evidenceID,
		Status:     "pending",
		Messages: []RequestMessage{
			{
				Role:      "user",
				Content:   message,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			},
		},
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	round.RequestThreads = append(round.RequestThreads, thread)
	if err := SaveRound(itemDir, *round); err != nil {
		return "", fmt.Errorf("save round: %w", err)
	}

	if s.eventLogger != nil && round.ExecutionID != "" {
		s.eventLogger.EmitReviewRequestCreated(round.ExecutionID, threadID, message)
	}

	// Start the declared evidence-request workflow with the operator's request.
	// The immutable input carries the operator's ask; the typed terminal result
	// is applied to this thread by ApplyEvidenceRequestWorkflow.
	if s.transitionRunner == nil {
		return "", fmt.Errorf("transition runner is not configured")
	}
	{
		// #nosec G118 -- intentional: the operation is started detached from the
		// HTTP request, which returns immediately. Inheriting the request context
		// would cancel the start the moment the handler responds; instead it gets
		// its own 30s-bounded context.
		go func() {
			opCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			// The thread was saved above, so the runner's registered builder can
			// reproject exactly this snapshot here and again at apply time.
			var executionID, definitionDigest, runID, entityVersion string
			started, startErr := s.transitionRunner.StartWith(opCtx, "review.evidence_request", reviewThreadSubject(kind, name, roundNum, threadID), transitionrunner.PreparedInput{FirstRunNodeID: "gather", Activity: &transitionrunner.Activity{OwnerType: "backlog", OwnerKind: kind, OwnerName: name, Purpose: "review"}})
			if startErr != nil {
				slog.Error("start evidence-request transition", "error", startErr, "thread_id", threadID)
				return
			}
			executionID, definitionDigest, entityVersion = started.ExecutionID, started.DefinitionDigest, started.EntityVersion
			if len(started.Attempts) > 0 {
				runID = started.Attempts[0].RunID
			}

			// Stamp the run id on the thread so the request-evidence completion
			// handler correlates the gathered evidence back to it.
			r, loadErr := LoadRound(itemDir, roundNum)
			if loadErr != nil || r == nil {
				return
			}
			for i := range r.RequestThreads {
				if r.RequestThreads[i].ID == threadID {
					r.RequestThreads[i].RunID = runID
					r.RequestThreads[i].AgentWorkflowExecutionID = executionID
					r.RequestThreads[i].AgentWorkflowDefinition = definitionDigest
					r.RequestThreads[i].AgentWorkflowVersion = entityVersion
					break
				}
			}
			_ = SaveRound(itemDir, *r)
			slog.Info("evidence-request workflow started", "thread_id", threadID, "run_id", runID, "workflow_execution_id", executionID)
		}()
	}

	return threadID, nil
}

// ApplyEvidenceRequestWorkflow applies a terminal typed evidence result to one
// request thread exactly once; workflow execution itself remains Agent Manager's
// authority.
func (s *Service) ApplyEvidenceRequestWorkflow(ctx context.Context, kind, name string, roundNum int, threadID string) (Round, bool, error) {
	itemDir := s.resolveItemDir(kind, name)
	round, err := LoadRound(itemDir, roundNum)
	if err != nil {
		return Round{}, false, fmt.Errorf("load review round: %w", err)
	}
	if round == nil {
		return Round{}, false, fmt.Errorf("review round %d does not exist", roundNum)
	}
	if s.transitionRunner == nil {
		return Round{}, false, fmt.Errorf("transition runner is not configured")
	}
	for _, thread := range round.RequestThreads {
		if thread.ID != threadID {
			continue
		}
		if strings.TrimSpace(thread.AgentWorkflowExecutionID) == "" {
			return Round{}, false, fmt.Errorf("thread is not owned by an evidence-request transition")
		}
		alreadyApplied := thread.Status != "pending"
		if _, err := s.transitionRunner.ApplyExecution(ctx, thread.AgentWorkflowExecutionID); err != nil {
			return Round{}, false, err
		}
		applied, err := LoadRound(itemDir, roundNum)
		if err != nil || applied == nil {
			return Round{}, false, fmt.Errorf("reload applied review round: %w", err)
		}
		return *applied, alreadyApplied, nil
	}
	return Round{}, false, fmt.Errorf("thread %s not found", threadID)
}

// ContinueRequest appends a user message to an existing request thread.
func (s *Service) ContinueRequest(kind, name string, roundNum int, threadID, message string) error {
	itemDir := s.resolveItemDir(kind, name)
	round, err := LoadRound(itemDir, roundNum)
	if err != nil {
		return fmt.Errorf("load round %d: %w", roundNum, err)
	}
	if round == nil {
		return fmt.Errorf("round %d not found", roundNum)
	}

	found := false
	for i := range round.RequestThreads {
		if round.RequestThreads[i].ID == threadID {
			round.RequestThreads[i].Messages = append(round.RequestThreads[i].Messages, RequestMessage{
				Role:      "user",
				Content:   message,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			})
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("thread %s not found in round %d", threadID, roundNum)
	}

	return SaveRound(itemDir, *round)
}

// DismissRequest marks a request thread as dismissed.
func (s *Service) DismissRequest(kind, name string, roundNum int, threadID string) error {
	itemDir := s.resolveItemDir(kind, name)
	round, err := LoadRound(itemDir, roundNum)
	if err != nil {
		return fmt.Errorf("load round %d: %w", roundNum, err)
	}
	if round == nil {
		return fmt.Errorf("round %d not found", roundNum)
	}

	found := false
	for i := range round.RequestThreads {
		if round.RequestThreads[i].ID == threadID {
			round.RequestThreads[i].Status = "dismissed"
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("thread %s not found in round %d", threadID, roundNum)
	}

	return SaveRound(itemDir, *round)
}

// RecordUnavailableReview writes a synthetic terminal review round for an item
// whose finalization routed it straight to review_pending because no review
// agent ran (disabled or spawn failure). The round carries the reason so the
// review surface explains the empty evidence set, and the item already sits in
// review_pending — no in_review→review_pending flip is needed. Satisfies
// execution.ReviewServiceIntegration. Best-effort and idempotent-ish: a new
// round number is allocated each call, but callers invoke it once per
// finalization.
func (s *Service) RecordUnavailableReview(kind, name, executionID, reason string) error {
	itemDir := s.resolveItemDir(kind, name)
	roundNum, err := NextRoundNumber(itemDir)
	if err != nil {
		return fmt.Errorf("determine next round: %w", err)
	}
	if strings.TrimSpace(reason) == "" {
		reason = "review agent did not run; routed to review_pending for manual decision"
	}
	round := Round{
		RoundNum:      roundNum,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		ExecutionID:   executionID,
		Status:        RoundStatusFailed,
		FailureReason: reason,
		Notes:         []string{"No review agent ran for this item. Decide a terminal status via review-decide, or recover the item if it was handled out-of-band."},
		Evidence:      []EvidenceItem{},
	}
	if err := SaveRound(itemDir, round); err != nil {
		return fmt.Errorf("save review-unavailable round: %w", err)
	}
	slog.Info("recorded review-unavailable round", "kind", kind, "name", name, "round", roundNum, "execution_id", executionID)
	return nil
}
