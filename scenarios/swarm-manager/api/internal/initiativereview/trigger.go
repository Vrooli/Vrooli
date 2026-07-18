package initiativereview

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/initiativelock"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/review"
)

// TriggerIfReady checks whether the initiative is ready for review and, if
// so, spawns the review agent and returns TriggerResult{Started: true}. If
// the initiative is already under review (in_review / review_pending) or
// has outstanding non-terminal items, returns Started=false with a reason.
// Idempotent: safe to call from multiple places (item transitions, manual
// triggers, recovery).
//
// Serialized under triggerGate so two items reaching terminal in the
// same tick both try TriggerForItem and only one wins the race to spawn
// a round — the second sees the freshly-saved in_review status and
// reports Started=false with reason "initiative status is \"in_review\"".
func (s *Service) TriggerIfReady(ctx context.Context, initiativeName string) (TriggerResult, error) {
	s.triggerGate.Lock()
	defer s.triggerGate.Unlock()
	init, err := s.initStore.Load(initiativeName)
	if err != nil {
		return TriggerResult{}, fmt.Errorf("load initiative: %w", err)
	}

	// Guard: only active initiatives can enter review. Anything else means
	// review has already started, the user has already decided, or the
	// initiative is archived — none of which should re-trigger.
	if init.Status != initiatives.InitiativeStatusActive {
		return TriggerResult{
			Started: false,
			Reason:  fmt.Sprintf("initiative status is %q; review triggers only from %q", init.Status, initiatives.InitiativeStatusActive),
		}, nil
	}

	// Guard: every member item must be in a terminal state. A fresh
	// initiative with zero items is also not "ready" — nothing to review.
	if len(init.Items) == 0 {
		return TriggerResult{Started: false, Reason: "initiative has no items"}, nil
	}
	nonTerminal, err := s.findNonTerminalItems(init)
	if err != nil {
		return TriggerResult{}, fmt.Errorf("scan items: %w", err)
	}
	if len(nonTerminal) > 0 {
		return TriggerResult{
			Started: false,
			Reason:  fmt.Sprintf("%d item(s) not yet terminal: %s", len(nonTerminal), strings.Join(nonTerminal, ", ")),
		}, nil
	}

	return s.startReview(ctx, init)
}

// TriggerForItem is the hook the backlog review-decide path calls after a
// member item flips to a terminal status. It resolves the item's initiative
// (if any) and calls TriggerIfReady. Errors are logged but never block the
// item decision — the review phase is a downstream consequence, not a
// precondition.
func (s *Service) TriggerForItem(ctx context.Context, kind, name string) {
	item, err := s.backlogLoader.LoadItem(backlog.BacklogKind(kind), name)
	if err != nil {
		return
	}
	initiativeName := strings.TrimSpace(item.Initiative)
	if initiativeName == "" {
		return
	}
	result, err := s.TriggerIfReady(ctx, initiativeName)
	if err != nil {
		// Lock conflicts are expected when a feedback round is in flight;
		// they're signal for the user, not an operator-facing alarm. Any
		// other error is unexpected (load/save failure, etc.) and stays WARN.
		var conflict *initiativelock.Conflict
		if errors.As(err, &conflict) {
			slog.Info("initiative review deferred: lock held",
				"initiative", initiativeName, "holder_purpose", conflict.Holder.Purpose)
			return
		}
		slog.Warn("initiative review trigger failed", "initiative", initiativeName, "kind", kind, "name", name, "err", err)
		return
	}
	if result.Started {
		slog.Info("initiative review started",
			"initiative", initiativeName,
			"round", result.Round,
			"run_id", result.RunID,
			"trigger", fmt.Sprintf("item:%s/%s", kind, name),
		)
	}
}

// startReview materializes a new round file in gathering state, flips the
// initiative to in_review, and spawns the review agent. Split out from
// TriggerIfReady so manual-trigger and auto-trigger share one code path.
//
// Lock contract: when a lock is wired, startReview acquires it with a
// provisional RunID before the spawn, then overrides it with the agent-
// manager RunID once the workflow starts. On start failure the
// provisional release clears the lock so the initiative isn't wedged.
// Release happens in handleTerminalRound (or on decide, if a future verdict
// path lands while the round is still alive). Returns a *initiativelock.
// Conflict error if feedback holds the lock; the caller can inspect the
// holder to render a user-facing conflict dialog.
func (s *Service) startReview(ctx context.Context, init *initiatives.Initiative) (TriggerResult, error) {
	itemDir := s.initStore.InitDir(init.Name)
	roundNum, err := review.NextRoundNumber(itemDir)
	if err != nil {
		return TriggerResult{}, fmt.Errorf("next round: %w", err)
	}

	generatedAt := s.clock().UTC().Format(time.RFC3339)
	round := review.Round{
		RoundNum:    roundNum,
		GeneratedAt: generatedAt,
		Status:      review.RoundStatusGathering,
		Evidence:    []review.EvidenceItem{},
	}

	// Acquire the per-initiative lock before starting review work. Feedback
	// rounds use the same file (`.feedback-lock`), so a feedback holder here
	// is a real conflict even in degraded no-spawner mode.
	provisionalRunID := fmt.Sprintf("review-provisional-%d-%d", roundNum, s.clock().UnixNano())
	if s.lock != nil {
		if err := s.lock.Acquire(init.Name, initiativelock.Holder{
			RunID:       provisionalRunID,
			Purpose:     "review",
			RoundNumber: roundNum,
			AcquiredBy:  "swarm-manager:initiative-review",
		}); err != nil {
			return TriggerResult{}, err
		}
	}

	if s.workflow == nil {
		// Degraded mode (no runner): the round still lands on disk and the
		// initiative flips to in_review so the UI doesn't lie about lifecycle,
		// but no agent-manager work happens. The provisional lock is released
		// immediately because there is no live run to own it after the status
		// transition.
		if err := review.SaveRound(itemDir, round); err != nil {
			if s.lock != nil {
				_ = s.lock.Release(init.Name, provisionalRunID)
			}
			return TriggerResult{}, fmt.Errorf("save round: %w", err)
		}
		if err := s.setInitiativeStatus(init, initiatives.InitiativeStatusInReview, generatedAt); err != nil {
			if s.lock != nil {
				_ = s.lock.Release(init.Name, provisionalRunID)
			}
			return TriggerResult{}, err
		}
		if s.lock != nil {
			_ = s.lock.Release(init.Name, provisionalRunID)
		}
		return TriggerResult{Started: true, Round: roundNum}, nil
	}

	input, version, inputErr := initiativeReviewInput(init.Name, roundNum, init)
	if inputErr != nil {
		if s.lock != nil {
			_ = s.lock.Release(init.Name, provisionalRunID)
		}
		return TriggerResult{}, fmt.Errorf("build initiative review snapshot: %w", inputErr)
	}
	res, err := s.workflow.StartWorkflow(ctx, agentmanager.Invocation{Owner: "swarm-manager", WorkflowKey: "swarm-manager/initiative-review", Input: input, IdempotencyKey: fmt.Sprintf("initiative-review-%s-r%d", init.Name, roundNum), FirstRunNodeID: "review"})
	if err != nil {
		if s.lock != nil {
			_ = s.lock.Release(init.Name, provisionalRunID)
		}
		return TriggerResult{}, fmt.Errorf("start initiative review workflow: %w", err)
	}

	// Swap the provisional holder for the agent-manager RunID so a later
	// Release(res.RunID) actually clears the lock. AcquireOverride is a pure file
	// write; on the unlikely failure path we release the provisional to avoid a
	// wedged lock. triggerGate is still held so no other caller can observe or
	// replace the provisional holder.
	if s.lock != nil {
		if swapErr := s.lock.AcquireOverride(init.Name, initiativelock.Holder{
			RunID:       res.RunID,
			Purpose:     "review",
			RoundNumber: roundNum,
			AcquiredBy:  "swarm-manager:initiative-review",
		}); swapErr != nil {
			slog.Warn("initiative review: lock run-id swap failed; releasing provisional",
				"initiative", init.Name, "round", roundNum, "err", swapErr)
			_ = s.lock.Release(init.Name, provisionalRunID)
		}
	}

	round.RunID = res.RunID
	round.ExecutionID = "" // initiative reviews have no single execution owner
	round.AgentWorkflowExecutionID = res.ExecutionID
	round.AgentWorkflowDefinition = res.DefinitionDigest
	round.AgentWorkflowVersion = version
	if err := review.SaveRound(itemDir, round); err != nil {
		return TriggerResult{}, fmt.Errorf("save round: %w", err)
	}
	if err := s.setInitiativeStatus(init, initiatives.InitiativeStatusInReview, generatedAt); err != nil {
		return TriggerResult{}, err
	}
	// Workflow-owned rounds are finalized only by ApplyWorkflowRound.

	slog.Info("initiative review started",
		"initiative", init.Name,
		"round", roundNum,
		"run_id", res.RunID,
		"workflow_execution", res.ExecutionID,
	)

	return TriggerResult{Started: true, Round: roundNum, RunID: res.RunID}, nil
}

// findNonTerminalItems returns the "kind/name" refs of member items that
// are not yet in a terminal state. Missing-from-disk items are treated as
// non-terminal so a half-deleted initiative won't race into review.
func (s *Service) findNonTerminalItems(init *initiatives.Initiative) ([]string, error) {
	out := make([]string, 0)
	for _, ref := range init.Items {
		parts := strings.SplitN(ref, "/", 2)
		if len(parts) != 2 {
			out = append(out, ref)
			continue
		}
		kind, err := backlog.ParseBacklogKind(parts[0])
		if err != nil {
			out = append(out, ref)
			continue
		}
		item, err := s.backlogLoader.LoadItem(kind, parts[1])
		if err != nil {
			out = append(out, ref)
			continue
		}
		if item.ArchivedAt != nil {
			continue // archived items are terminal for rollup purposes
		}
		if !backlog.IsTerminalStatus(item.Status) {
			out = append(out, ref)
		}
	}
	return out, nil
}
