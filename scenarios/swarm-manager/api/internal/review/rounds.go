package review

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/idgen"
	"swarm-manager/internal/pathutil"
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

// RequestMoreEvidence creates a new request thread and optionally spawns a
// targeted review agent run.
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

	// Spawn a targeted review agent with the user's request.
	if s.agentService != nil && s.agentService.IsEnabled() {
		// #nosec G118 -- intentional: the review agent is spawned detached from
		// the HTTP request, which returns immediately. Inheriting the request
		// context would cancel the spawn the moment the handler responds;
		// instead it gets its own 30s-bounded context.
		go func() {
			spawnCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			itemTitle := s.resolveItemTitle(kind, name, "")
			deliverableContent := loadReviewDeliverableContent(kind, itemDir)
			var changedPaths []string
			var affectedScenarios []string
			var gctResultsJSON string
			var baselineDiffJSON string
			if round.ExecutionID != "" && s.loadExecutionContext != nil {
				if execCtx, ctxErr := s.loadExecutionContext(spawnCtx, round.ExecutionID); ctxErr != nil {
					slog.Warn("load execution review context for evidence request", "execution_id", round.ExecutionID, "error", ctxErr)
				} else if execCtx != nil {
					changedPaths = flattenChangedPaths(execCtx.ChangedPathsByScenario)
					affectedScenarios = pathutil.UniqueSortedStrings(append([]string(nil), execCtx.AffectedScenarios...))
					gctResultsJSON = execCtx.GCTResultsJSON
					baselineDiffJSON = execCtx.BaselineDiffJSON
					itemTitle = s.resolveItemTitle(execCtx.BacklogKind, execCtx.BacklogName, execCtx.ItemTitle)
				}
			}

			// Inject agent activity spec for the tracked agent service.
			spawnCtx = agentactivity.WithSpec(spawnCtx, agentactivity.Spec{
				OwnerType:   agentactivity.OwnerBacklog,
				OwnerKind:   kind,
				OwnerName:   name,
				OwnerTitle:  itemTitle,
				ExecutionID: round.ExecutionID,
				Purpose:     agentactivity.PurposeReview,
				PhaseKind:   string(agentactivity.LaneReview),
				RequestedBy: "swarm-manager-ui",
				Metadata: map[string]string{
					"entrypoint": "review.request_more_evidence",
					"thread_id":  threadID,
				},
			})

			instructionVars := map[string]string{
				"ITEM_FOLDER":  itemDir,
				"ROUND_NUMBER": fmt.Sprintf("%03d", roundNum),
			}
			prompt, renderErr := s.promptClient.ReadSkill(spawnCtx, "swarm-manager-review", instructionVars, true)
			if renderErr != nil {
				slog.Error("render review skill for evidence request", "error", renderErr, "thread_id", threadID)
				return
			}

			reqAttachments := buildReviewAttachments(deliverableContent, changedPaths, affectedScenarios, gctResultsJSON, baselineDiffJSON, message)

			titlePreview := message
			if len(titlePreview) > 50 {
				titlePreview = titlePreview[:50]
			}

			result, spawnErr := s.agentService.SpawnBacklog(spawnCtx, agentmanager.BacklogSpawnRequest{
				Kind:               kind,
				Name:               name,
				Title:              "Evidence Request: " + titlePreview,
				Description:        prompt,
				Prompt:             prompt,
				ScopePath:          ".",
				ProjectRoot:        ".",
				CreatedBy:          "swarm-manager:review-request",
				Purpose:            "review",
				ContextAttachments: reqAttachments,
				Environment: map[string]string{
					"VROOLI_SPAWN_SOURCE": "swarm-manager-review-request",
				},
			})
			if spawnErr != nil {
				if errors.Is(spawnErr, agentactivity.ErrBacklogItemBusy) {
					slog.Info("evidence request skipped: agent already active", "kind", kind, "name", name, "thread_id", threadID)
				} else {
					slog.Error("spawn review agent for evidence request", "error", spawnErr, "thread_id", threadID)
				}
				return
			}

			// Update thread with agent RunID.
			r, loadErr := LoadRound(itemDir, roundNum)
			if loadErr != nil || r == nil {
				return
			}
			for i := range r.RequestThreads {
				if r.RequestThreads[i].ID == threadID {
					r.RequestThreads[i].RunID = result.RunID
					break
				}
			}
			_ = SaveRound(itemDir, *r)
			slog.Info("review evidence request agent spawned", "thread_id", threadID, "run_id", result.RunID)
		}()
	}

	return threadID, nil
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
