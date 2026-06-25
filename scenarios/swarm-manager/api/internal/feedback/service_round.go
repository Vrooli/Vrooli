package feedback

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"swarm-manager/internal/initiativelock"
)

// StartRoundRequest is the caller-supplied input for starting a new round.
type StartRoundRequest struct {
	InitiativeName string
	Type           RoundType
	Text           string
	AttachmentIDs  []string
	SlugHint       string // optional; derived from Text if empty
	Override       bool   // preempt active lock if present
	DecidedBy      string // user identifier for audit

	// AttachmentLoader, when set, runs inside StartRound after the round
	// directory has been reserved (atomic) but before the round is
	// persisted. It must populate `roundDir` with attachment files and
	// return their relative IDs. Set by handlers serving multipart
	// uploads so attachments land in the dir the service actually
	// claimed — fixes the race where two concurrent submits would
	// otherwise compute the same predicted number and clobber each
	// other's attachments.
	AttachmentLoader func(roundDir string) ([]string, error)
}

// StartRound creates a new round, acquires the lock, and (for active round
// types) spawns the feedback agent. For note-type rounds the agent is
// skipped — the round lands in RoundStatusDismissed right away because
// notes are signal-only entries, not conversations.
func (s *Service) StartRound(ctx context.Context, req StartRoundRequest) (Round, error) {
	if err := validateStartRoundRequest(&req); err != nil {
		return Round{}, err
	}

	slug := ComputeSlug(req.SlugHint, req.Text)
	number, roundDir, err := s.store.ReserveRound(req.InitiativeName, slug)
	if err != nil {
		return Round{}, fmt.Errorf("reserve round: %w", err)
	}

	// Run the attachment loader against the reserved dir so multipart
	// callers can drop files in the slot the service actually claimed.
	attachmentIDs := append([]string(nil), req.AttachmentIDs...)
	if req.AttachmentLoader != nil {
		ids, loadErr := req.AttachmentLoader(roundDir)
		if loadErr != nil {
			// Best-effort cleanup: remove the reserved dir so the
			// number we claimed gets reused on the next attempt.
			if rmErr := os.RemoveAll(roundDir); rmErr != nil {
				slog.Warn("feedback: cleanup reserved round dir failed", "err", rmErr, "dir", roundDir)
			}
			return Round{}, fmt.Errorf("load attachments: %w", loadErr)
		}
		attachmentIDs = append(attachmentIDs, ids...)
	}

	now := s.clock().UTC().Format(time.RFC3339)
	round := Round{
		InitiativeName: req.InitiativeName,
		Number:         number,
		Slug:           slug,
		Type:           req.Type,
		Status:         RoundStatusSubmitting,
		Submission: Submission{
			Text:          strings.TrimSpace(req.Text),
			AttachmentIDs: attachmentIDs,
			CreatedAt:     now,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Note-type rounds skip the agent path: record the submission and the
	// user's authored message, then mark dismissed. They still count as a
	// feedback event for the future meta-optimizer, which is the whole
	// point of the note type.
	if req.Type == RoundTypeNote {
		return s.completeNoteRound(round, req.DecidedBy, now)
	}

	// Pre-flight: surface item-level activity so the override dialog can
	// list specific blockers rather than failing opaquely. When the caller
	// has opted in to override we skip the early return (they already saw
	// the warning), but we still collect the list so the preempt step
	// below can cancel those runs — otherwise "override" would be a lie
	// and a second agent would run behind the new one.
	var busy []ItemActivity
	if s.activity != nil {
		items, actErr := s.activity.ActiveRunsForInitiative(req.InitiativeName)
		if actErr != nil {
			slog.Warn("feedback: item activity check failed", "err", actErr, "initiative", req.InitiativeName)
		} else {
			busy = items
		}
		if len(busy) > 0 && !req.Override {
			return Round{}, &BusyError{Activities: busy}
		}
	}

	// Feedback rounds acquire the lock before any agent side-effect runs
	// so concurrent submissions can't race us into two spawns. The
	// provisional RunID is swapped for the agent-manager RunID once the
	// spawn succeeds; on spawn failure, Release uses the provisional to
	// clear the lock rather than leaving it wedged.
	holder := initiativelock.Holder{
		Purpose:     "feedback",
		RoundNumber: number,
		AcquiredBy:  req.DecidedBy,
	}
	if s.spawner != nil {
		holder.RunID = fmt.Sprintf("provisional-%d-%d", number, time.Now().UnixNano())
	} else {
		holder.RunID = fmt.Sprintf("no-spawner-%d", number)
	}
	provisionalRunID := holder.RunID
	if err := s.acquireInitiativeRoundLock(ctx, req, holder, busy); err != nil {
		return Round{}, err
	}

	// Record the submission to disk before spawning so a spawn failure
	// leaves evidence of the user's attempt.
	round.Status = RoundStatusAgentThinking
	round.Thread = []Message{{
		Role:          "user",
		Content:       round.Submission.Text,
		AttachmentIDs: round.Submission.AttachmentIDs,
		CreatedAt:     now,
	}}
	if err := s.store.SaveRound(round); err != nil {
		s.releaseRoundLock(req.InitiativeName, provisionalRunID)
		return Round{}, fmt.Errorf("save round: %w", err)
	}

	if s.spawner == nil {
		// Test / degraded mode: no agent spawn, just persist and flip to
		// awaiting_user so the caller can walk the rest of the lifecycle.
		round.Status = RoundStatusAwaitingUser
		round.UpdatedAt = s.clock().UTC().Format(time.RFC3339)
		if err := s.store.SaveRound(round); err != nil {
			return Round{}, err
		}
		s.releaseRoundLock(req.InitiativeName, provisionalRunID)
		return round, nil
	}

	runID, err := s.spawner.SpawnInitiativeFeedback(ctx, SpawnRequest{
		InitiativeName: req.InitiativeName,
		RoundNumber:    number,
		RoundSlug:      slug,
		Purpose:        "feedback",
		SubmissionText: round.Submission.Text,
		AttachmentIDs:  round.Submission.AttachmentIDs,
	})
	if err != nil {
		s.releaseRoundLock(req.InitiativeName, provisionalRunID)
		round.Status = RoundStatusAwaitingUser
		round.UpdatedAt = s.clock().UTC().Format(time.RFC3339)
		round.Thread = append(round.Thread, Message{
			Role:      "agent",
			Content:   fmt.Sprintf("agent spawn failed: %s", err.Error()),
			CreatedAt: round.UpdatedAt,
		})
		if saveErr := s.store.SaveRound(round); saveErr != nil {
			slog.Warn("feedback: persist round failed", "err", saveErr, "initiative", round.InitiativeName, "round", round.Number)
		}
		return round, fmt.Errorf("spawn agent: %w", err)
	}
	round.RunID = runID
	round.UpdatedAt = s.clock().UTC().Format(time.RFC3339)
	if err := s.store.SaveRound(round); err != nil {
		slog.Warn("feedback: persist run id after spawn failed", "err", err, "run_id", runID)
	}
	// Swap the provisional holder for the real run id so Inspect surfaces
	// the actual agent-manager run to the UI. If the swap fails — rare,
	// since AcquireOverride is a pure file write — fall back to releasing
	// the provisional so the initiative isn't wedged. We lose the lock
	// guarantee for the duration of this run, but that's a strictly
	// better failure mode than a stuck lock.
	holder.RunID = runID
	if swapErr := s.lock.AcquireOverride(req.InitiativeName, holder); swapErr != nil {
		slog.Warn("feedback: lock run-id swap failed; releasing provisional",
			"err", swapErr,
			"initiative", req.InitiativeName,
			"provisional", provisionalRunID,
			"run_id", runID,
		)
		s.releaseRoundLock(req.InitiativeName, provisionalRunID)
	}
	return round, nil
}

// releaseRoundLock releases the initiative lock held under runID, logging (but
// not propagating) a release failure — a wedged lock is reported for diagnosis
// but must never mask the original error that triggered the release.
func (s *Service) releaseRoundLock(initiativeName, runID string) {
	if relErr := s.lock.Release(initiativeName, runID); relErr != nil {
		slog.Warn("feedback: release lock failed", "err", relErr, "initiative", initiativeName, "run_id", runID)
	}
}

// acquireInitiativeRoundLock takes the initiative lock for a new round. On the
// override path it first preempts the displaced run(s) so the single-agent
// invariant holds, then force-acquires; otherwise it acquires normally.
func (s *Service) acquireInitiativeRoundLock(ctx context.Context, req StartRoundRequest, holder initiativelock.Holder, busy []ItemActivity) error {
	if req.Override {
		// Cancel the preempted run(s) before taking the lock so "override"
		// actually enforces single-agent-per-initiative rather than just
		// overwriting a lock file. Best-effort: a cancel failure is logged but
		// doesn't block the new round — the user has explicitly chosen to
		// preempt and the lock overwrite still happens.
		s.preemptForOverride(ctx, req.InitiativeName, busy)
		if err := s.lock.AcquireOverride(req.InitiativeName, holder); err != nil {
			return fmt.Errorf("override lock: %w", err)
		}
		return nil
	}
	return s.lock.Acquire(req.InitiativeName, holder)
}

// validateStartRoundRequest enforces the StartRound preconditions and applies
// the default round type. It mutates req in place (defaulting an empty Type to
// RoundTypeFeedback) so the caller sees the normalized request.
func validateStartRoundRequest(req *StartRoundRequest) error {
	if strings.TrimSpace(req.InitiativeName) == "" {
		return errors.New("initiative name is required")
	}
	if req.Type == "" {
		req.Type = RoundTypeFeedback
	}
	if !isValidRoundType(req.Type) {
		return fmt.Errorf("unsupported round type %q", req.Type)
	}
	if req.Type == RoundTypeResearch {
		return errors.New("research-type rounds are not implemented in this release")
	}
	if strings.TrimSpace(req.Text) == "" {
		return errors.New("text is required")
	}
	return nil
}

// completeNoteRound persists a note-type round: it records the user's authored
// message and immediately marks the round dismissed. Note rounds skip the agent
// path entirely but still count as a feedback event for the meta-optimizer.
func (s *Service) completeNoteRound(round Round, decidedBy, now string) (Round, error) {
	round.Thread = []Message{{
		Role:          "user",
		Content:       round.Submission.Text,
		AttachmentIDs: round.Submission.AttachmentIDs,
		CreatedAt:     now,
	}}
	round.Status = RoundStatusDismissed
	round.Decision = &Decision{
		Kind:      DecisionDismiss,
		DecidedAt: now,
		DecidedBy: decidedBy,
		Rationale: "note",
	}
	if err := s.store.SaveRound(round); err != nil {
		return Round{}, fmt.Errorf("save note round: %w", err)
	}
	return round, nil
}

// preemptForOverride cancels the agent runs that a subsequent AcquireOverride
// will displace, so the "single agent per initiative" invariant survives
// the override path. It runs best-effort: any failure is logged but does
// not block the new round, because the user has already explicitly opted
// in to preemption via the warning dialog.
//
// Targets:
//  1. The RunID recorded in the current lock holder (another in-flight
//     feedback round or initiative review).
//  2. The RunIDs of any busy member items supplied by the caller.
//
// For (1), if the holder identifies an in-flight feedback round on the
// same initiative (purpose "feedback" or "feedback_continue", matching
// RunID, status agent_thinking), we mark that round dismissed with a
// clear rationale so the feedback log carries an audit trail rather than
// silently leaving a half-finished round behind.
func (s *Service) preemptForOverride(ctx context.Context, initiativeName string, busyItems []ItemActivity) {
	// Current holder, if any.
	holder, inspectErr := s.lock.Inspect(initiativeName)
	if inspectErr != nil {
		slog.Warn("feedback: override preempt: inspect lock failed",
			"err", inspectErr, "initiative", initiativeName)
	}
	if holder != nil && holder.RunID != "" && s.canceller != nil {
		if err := s.canceller.StopRun(ctx, holder.RunID); err != nil {
			slog.Warn("feedback: override preempt: stop holder run failed",
				"err", err, "initiative", initiativeName, "run_id", holder.RunID)
		}
	}
	if holder != nil && isFeedbackPurpose(holder.Purpose) {
		s.markPreemptedFeedbackRound(initiativeName, holder.RunID, holder.RoundNumber)
	}

	// Item-level preemption: cancel any busy agents running against
	// member items so they don't fight the new feedback agent for
	// filesystem/state. The items' own backlog status transitions fall
	// out of the normal polling / finalization path — we don't touch
	// them here.
	if s.canceller != nil {
		for _, act := range busyItems {
			if strings.TrimSpace(act.RunID) == "" {
				continue
			}
			if err := s.canceller.StopRun(ctx, act.RunID); err != nil {
				slog.Warn("feedback: override preempt: stop item run failed",
					"err", err, "initiative", initiativeName,
					"ref", act.Ref, "run_id", act.RunID)
			}
		}
	}
}

// markPreemptedFeedbackRound finds the in-flight feedback round matching
// the preempted lock holder and records a dismiss decision so the feedback
// log makes the preemption explicit. Silent no-op when the round can't be
// resolved (already terminal, different initiative, etc.) — the audit
// trail is nice-to-have, not load-bearing.
func (s *Service) markPreemptedFeedbackRound(initiativeName, runID string, roundNumber int) {
	if roundNumber <= 0 {
		return
	}
	round, err := s.store.LoadRound(initiativeName, roundNumber)
	if err != nil || round.Status != RoundStatusAgentThinking || round.RunID != runID {
		return
	}
	now := s.clock().UTC().Format(time.RFC3339)
	round.Status = RoundStatusDismissed
	round.UpdatedAt = now
	round.Decision = &Decision{
		Kind:      DecisionDismiss,
		DecidedAt: now,
		Rationale: "preempted: user started a new feedback round via override",
	}
	if err := s.store.SaveRound(round); err != nil {
		slog.Warn("feedback: override preempt: save dismissed round failed",
			"err", err, "initiative", initiativeName, "round", roundNumber)
	}
}

// isFeedbackPurpose reports whether the given lock purpose identifies a
// feedback-service-owned holder. Initiative review locks use "review" and
// stay untouched by the round-preemption helper (the review round has its
// own store).
func isFeedbackPurpose(p string) bool {
	switch strings.TrimSpace(p) {
	case "feedback", "feedback_continue":
		return true
	}
	return false
}
