package authoring

import (
	"context"
	"fmt"
	"strings"

	planmodel "plan-manager/internal/planmodel"
	"plan-manager/internal/readiness"
)

func (s *service) SubmitRelevantContextItem(ctx context.Context, sessionID, phaseID string, item planmodel.RelevantContextItem) (Session, planmodel.RelevantContextItem, []StructureViolation, GuidedStep, error) {
	item = normalizeContextItem(item, phaseID)
	var violations []StructureViolation
	sess, err := s.withSessionLock(ctx, sessionID, func(sess *Session) (bool, error) {
		if phaseID != "" {
			idx := indexOfDraft(sess.PhaseDrafts, phaseID)
			if idx < 0 {
				return false, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + phaseID}
			}
			item.Scope = planmodel.RelevantContextScopePhase
			item.PhaseID = sess.PhaseDrafts[idx].ID
			// A phase-scoped item repeats on phase entry, not once-per-execution —
			// apply the scope default here (mirrors AcceptContextCandidate) so an unset
			// or contradictory once_per_execution policy is corrected at submit time.
			item.RepeatPolicy = defaultRepeatForScope(item.Scope, item.RepeatPolicy)
			violations = contextItemViolations(item)
			if len(violations) == 0 {
				sess.PhaseDrafts[idx].RelevantContext = append(sess.PhaseDrafts[idx].RelevantContext, item)
			}
			sess.CurrentPhaseID = nextIncompletePhaseID(sess.PhaseDrafts)
			*sess = syncPhaseSection(*sess)
		} else {
			item.Scope = planmodel.RelevantContextScopeGlobal
			item.PhaseID = ""
			item.RepeatPolicy = defaultRepeatForScope(item.Scope, item.RepeatPolicy)
			violations = contextItemViolations(item)
			if len(violations) == 0 {
				sess.RelevantContext = append(sess.RelevantContext, item)
				*sess = syncContextSection(*sess)
			}
		}
		if len(violations) > 0 {
			return false, nil
		}
		// D4: a direct submit disposes any pending discovered candidate for the
		// same target — the author decided about it by submitting it.
		*sess = acceptMatchingPendingCandidates(*sess, item)
		return true, nil
	})
	if err != nil {
		return Session{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, err
	}
	return sess, item, violations, stepForCurrentSessionState(sess), nil
}

// acceptMatchingPendingCandidates marks pending discovered candidates whose
// (kind, target/command) matches a directly submitted item as accepted, so a
// direct submit counts as a disposition (contract decision D4).
func acceptMatchingPendingCandidates(sess Session, item planmodel.RelevantContextItem) Session {
	key := contextDedupeKey(item)
	for i := range sess.ContextCandidates {
		if sess.ContextCandidates[i].Status != ContextCandidatePending {
			continue
		}
		if contextDedupeKey(sess.ContextCandidates[i].Item) == key {
			sess.ContextCandidates[i].Status = ContextCandidateAccepted
		}
	}
	return sess
}

// steeredSkillCandidate converts one resolver suggestion into an ordinary
// pending candidate (bare-slug target; the renderer derives the read command).
func steeredSkillCandidate(suggestion SkillSuggestion) (ContextCandidate, bool) {
	slug := strings.TrimSpace(suggestion.Slug)
	if slug == "" {
		return ContextCandidate{}, false
	}
	return ContextCandidate{
		Item: planmodel.RelevantContextItem{
			Kind:        planmodel.RelevantContextSkill,
			Scope:       planmodel.RelevantContextScopeGlobal,
			Label:       slug,
			Reason:      strings.TrimSpace(suggestion.Reason),
			Instruction: "Load this internal skill before implementation.",
			Target:      slug,
			Source:      planmodel.RelevantContextSourceDiscovered,
			Status:      planmodel.RelevantContextStatusReady,
		},
		Concept: "skill-applicability",
		Source:  "skill-applicability-resolver",
		Status:  ContextCandidatePending,
	}, true
}

func (s *service) ListRelevantContext(ctx context.Context, sessionID, phaseID string) ([]planmodel.RelevantContextItem, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return nil, GuidedStep{}, err
	}
	if phaseID == "" {
		return append([]planmodel.RelevantContextItem(nil), sess.RelevantContext...), stepForCurrentSessionState(sess), nil
	}
	phase, ok := findDraft(sess.PhaseDrafts, phaseID)
	if !ok {
		return nil, GuidedStep{}, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + phaseID}
	}
	return append([]planmodel.RelevantContextItem(nil), phase.RelevantContext...), stepForPhase(sess, phase), nil
}

// UpdateRelevantContextItem replaces one accepted context item in place (by id)
// so a bad item discovered in preview is corrected without deleting the whole
// phase/session. Legal only before finalize. On a content violation the session
// is left unchanged (mirrors SubmitRelevantContextItem).
func (s *service) UpdateRelevantContextItem(ctx context.Context, sessionID, phaseID, itemID string, item planmodel.RelevantContextItem) (Session, planmodel.RelevantContextItem, []StructureViolation, GuidedStep, error) {
	itemID = strings.TrimSpace(itemID)
	if itemID == "" {
		return Session{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "item_id is required"}
	}
	item.ID = itemID
	item = normalizeContextItem(item, phaseID)
	var violations []StructureViolation
	sess, err := s.withSessionLock(ctx, sessionID, func(sess *Session) (bool, error) {
		if sess.Finalized {
			return false, ErrInvalidSession{Reason: "relevant context cannot be edited after finalize"}
		}
		if phaseID != "" {
			phaseIdx := indexOfDraft(sess.PhaseDrafts, phaseID)
			if phaseIdx < 0 {
				return false, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + phaseID}
			}
			item.Scope = planmodel.RelevantContextScopePhase
			item.PhaseID = sess.PhaseDrafts[phaseIdx].ID
			item.RepeatPolicy = defaultRepeatForScope(item.Scope, item.RepeatPolicy)
			pos := indexOfContextItem(sess.PhaseDrafts[phaseIdx].RelevantContext, itemID)
			if pos < 0 {
				return false, ErrInvalidSession{Reason: "phase context item not found: " + itemID}
			}
			if violations = contextItemViolations(item); len(violations) > 0 {
				return false, nil
			}
			sess.PhaseDrafts[phaseIdx].RelevantContext[pos] = item
			*sess = syncPhaseSection(*sess)
		} else {
			item.Scope = planmodel.RelevantContextScopeGlobal
			item.PhaseID = ""
			item.RepeatPolicy = defaultRepeatForScope(item.Scope, item.RepeatPolicy)
			pos := indexOfContextItem(sess.RelevantContext, itemID)
			if pos < 0 {
				return false, ErrInvalidSession{Reason: "global context item not found: " + itemID}
			}
			if violations = contextItemViolations(item); len(violations) > 0 {
				return false, nil
			}
			sess.RelevantContext[pos] = item
			*sess = syncContextSection(*sess)
		}
		return true, nil
	})
	if err != nil {
		return Session{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, err
	}
	return sess, item, violations, stepForCurrentSessionState(sess), nil
}

// RemoveRelevantContextItem deletes one accepted context item (by id) before
// finalize, recomputing structure violations so a resulting gate (e.g. a phase
// left with no context) is reported with its recovery step.
func (s *service) RemoveRelevantContextItem(ctx context.Context, sessionID, phaseID, itemID string) (Session, []StructureViolation, GuidedStep, error) {
	itemID = strings.TrimSpace(itemID)
	if itemID == "" {
		return Session{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "item_id is required"}
	}
	var violations []StructureViolation
	sess, err := s.withSessionLock(ctx, sessionID, func(sess *Session) (bool, error) {
		if sess.Finalized {
			return false, ErrInvalidSession{Reason: "relevant context cannot be edited after finalize"}
		}
		if phaseID != "" {
			phaseIdx := indexOfDraft(sess.PhaseDrafts, phaseID)
			if phaseIdx < 0 {
				return false, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + phaseID}
			}
			pos := indexOfContextItem(sess.PhaseDrafts[phaseIdx].RelevantContext, itemID)
			if pos < 0 {
				return false, ErrInvalidSession{Reason: "phase context item not found: " + itemID}
			}
			sess.PhaseDrafts[phaseIdx].RelevantContext = removeContextItemAt(sess.PhaseDrafts[phaseIdx].RelevantContext, pos)
			sess.CurrentPhaseID = nextIncompletePhaseID(sess.PhaseDrafts)
			*sess = syncPhaseSection(*sess)
			violations = phaseViolations(sess.PhaseDrafts[phaseIdx])
		} else {
			pos := indexOfContextItem(sess.RelevantContext, itemID)
			if pos < 0 {
				return false, ErrInvalidSession{Reason: "global context item not found: " + itemID}
			}
			sess.RelevantContext = removeContextItemAt(sess.RelevantContext, pos)
			*sess = syncContextSection(*sess)
		}
		return true, nil
	})
	if err != nil {
		return Session{}, nil, GuidedStep{}, err
	}
	step := stepForCurrentSessionState(sess)
	if phaseID == "" && (!globalContextResolved(sess) || !globalSkillContextResolved(sess)) {
		step = stepForGlobalContextCheckpoint(sess)
	}
	return sess, violations, step, nil
}

func (s *service) DiscoverContextCandidates(ctx context.Context, sessionID string, concepts []string, complexity string, refresh bool) (Session, []ContextCandidate, GuidedStep, error) {
	// Discovery shells out (probes, resolver) and can take tens of seconds — run
	// it against a pre-lock read so a slow probe never holds the session lock;
	// only the append happens under the lock.
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, nil, GuidedStep{}, err
	}
	if !refresh && len(normalizeConcepts(concepts, "")) == 0 {
		if batch := LatestDiscoveryBatch(sess); batch.ID != "" {
			return sess, contextCandidatesForBatch(sess.ContextCandidates, batch.ID), stepForContextDiscovery(sess), nil
		}
		if batch, ok := latestPendingDiscoveryBatch(sess); ok && batch.Source == "prefetch" {
			return sess, contextCandidatesForBatch(sess.ContextCandidates, batch.ID), stepForContextDiscovery(sess), nil
		}
		if done := s.pendingContextPrefetch(sessionID); done != nil {
			select {
			case <-done:
				refreshed, err := s.load(ctx, sessionID)
				if err != nil {
					return Session{}, nil, GuidedStep{}, err
				}
				if batch, ok := latestPendingDiscoveryBatch(refreshed); ok && batch.Source == "prefetch" {
					return refreshed, contextCandidatesForBatch(refreshed.ContextCandidates, batch.ID), stepForContextDiscovery(refreshed), nil
				}
				sess = refreshed
				if batch := LatestDiscoveryBatch(sess); batch.ID != "" {
					return sess, contextCandidatesForBatch(sess.ContextCandidates, batch.ID), stepForContextDiscovery(sess), nil
				}
			case <-ctx.Done():
				return Session{}, nil, GuidedStep{}, ctx.Err()
			}
		}
		return sess, nil, stepForContextDiscovery(sess), nil
	}
	var result ContextDiscoveryResult
	if s.context == nil {
		result.ProbeNotes = degradedContextProbeNotes(sess.Title, concepts, "context-discovery", "context discovery unavailable")
	} else {
		result, err = s.context.DiscoverContext(ctx, sess.Title, concepts, complexity)
		if err != nil {
			result = ContextDiscoveryResult{
				ProbeNotes: degradedContextProbeNotes(sess.Title, concepts, "context-discovery", err.Error()),
			}
		}
	}
	// D7: steered suggestions from the skill-applicability resolver enter the
	// same accept/reject disposition flow as probe-discovered candidates.
	for _, suggestion := range s.skillSteer.SuggestSkills(ctx, sessionBoundary(sess)) {
		if candidate, ok := steeredSkillCandidate(suggestion); ok {
			result.Candidates = append(result.Candidates, candidate)
		}
	}
	for i := range result.Candidates {
		result.Candidates[i] = normalizeContextCandidate(result.Candidates[i])
		result.Candidates[i] = s.validateContextCandidate(ctx, result.Candidates[i])
	}
	var batch DiscoveryBatch
	sess, err = s.withSessionLock(ctx, sessionID, func(sess *Session) (bool, error) {
		batch = mergeContextDiscoveryBatch(sess, concepts, complexity, result)
		return true, nil
	})
	if err != nil {
		return Session{}, nil, GuidedStep{}, err
	}
	return sess, contextCandidatesForBatch(sess.ContextCandidates, batch.ID), stepAfterContextDiscovery(sess), nil
}

func (s *service) AcceptContextCandidate(ctx context.Context, sessionID, candidateID, phaseID string) (Session, ContextCandidate, planmodel.RelevantContextItem, []StructureViolation, GuidedStep, error) {
	var (
		candidate  ContextCandidate
		item       planmodel.RelevantContextItem
		violations []StructureViolation
	)
	sess, err := s.withSessionLock(ctx, sessionID, func(sess *Session) (bool, error) {
		idx := indexOfCandidate(sess.ContextCandidates, candidateID)
		if idx < 0 {
			return false, ErrInvalidSession{Reason: "context candidate not found: " + candidateID}
		}
		var changed bool
		var acceptErr error
		candidate, item, violations, changed, acceptErr = acceptContextCandidateAt(sess, idx, phaseID)
		if acceptErr != nil || len(violations) > 0 {
			return false, acceptErr
		}
		if changed {
			if batchIdx := indexOfDiscoveryBatch(sess.DiscoveryBatches, candidate.BatchID); batchIdx >= 0 {
				closeContextBatchIfResolved(sess, batchIdx, "applied item-by-item")
			}
		}
		return changed, nil
	})
	if err != nil {
		return Session{}, ContextCandidate{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, err
	}
	if len(violations) > 0 {
		return sess, candidate, item, violations, stepForContextDiscovery(sess), nil
	}
	return sess, candidate, item, nil, stepForCurrentSessionState(sess), nil
}

func (s *service) RejectContextCandidate(ctx context.Context, sessionID, candidateID, reason string) (Session, ContextCandidate, GuidedStep, error) {
	// D4: a rejection is a disposition and must carry its judgment — an empty
	// reason would turn the sweep evidence into a rubber stamp.
	if strings.TrimSpace(reason) == "" {
		return Session{}, ContextCandidate{}, GuidedStep{}, ErrInvalidSession{Reason: "candidate rejection requires a --reason"}
	}
	var candidate ContextCandidate
	sess, err := s.withSessionLock(ctx, sessionID, func(sess *Session) (bool, error) {
		idx := indexOfCandidate(sess.ContextCandidates, candidateID)
		if idx < 0 {
			return false, ErrInvalidSession{Reason: "context candidate not found: " + candidateID}
		}
		candidate = rejectContextCandidateAt(sess, idx, strings.TrimSpace(reason))
		if batchIdx := indexOfDiscoveryBatch(sess.DiscoveryBatches, candidate.BatchID); batchIdx >= 0 {
			closeContextBatchIfResolved(sess, batchIdx, "applied item-by-item")
		}
		return true, nil
	})
	if err != nil {
		return Session{}, ContextCandidate{}, GuidedStep{}, err
	}
	return sess, candidate, stepForContextDiscovery(sess), nil
}

func (s *service) ApplyContextDisposition(ctx context.Context, sessionID, batchID string, takes []ContextDispositionTake, drops []ContextDispositionDrop, sweepNote string, takeAll bool) (Session, ContextDispositionSummary, []StructureViolation, GuidedStep, error) {
	var (
		summary    ContextDispositionSummary
		violations []StructureViolation
	)
	sess, err := s.withSessionLock(ctx, sessionID, func(sess *Session) (bool, error) {
		batchIdx := targetContextBatchIndex(*sess, batchID)
		if batchIdx < 0 {
			return false, ErrInvalidSession{Reason: "pending context discovery batch not found"}
		}
		if sess.DiscoveryBatches[batchIdx].Status == DiscoveryBatchApplied {
			batch := sess.DiscoveryBatches[batchIdx]
			summary.Results = append(summary.Results, skippedContextDispositionResults(sess.ContextCandidates, batch.ID, takes, drops)...)
			summary.Batch = batch
			return false, nil
		}
		if sess.DiscoveryBatches[batchIdx].Status != DiscoveryBatchPending {
			return false, ErrInvalidSession{Reason: "context discovery batch is not pending: " + sess.DiscoveryBatches[batchIdx].ID}
		}
		batch := sess.DiscoveryBatches[batchIdx]
		takeByCandidate := map[int]ContextDispositionTake{}
		dropByCandidate := map[int]ContextDispositionDrop{}
		for _, take := range takes {
			id := strings.TrimSpace(take.CandidateID)
			idx := indexOfContextCandidateInBatch(sess.ContextCandidates, batch.ID, id)
			if idx < 0 {
				summary.Results = append(summary.Results, ContextDispositionResult{Action: "take", Message: "context candidate not found in batch: " + id})
				continue
			}
			if sess.ContextCandidates[idx].Status != ContextCandidatePending {
				summary.Results = append(summary.Results, skippedContextDispositionResult(sess.ContextCandidates[idx], "take"))
				continue
			}
			takeByCandidate[idx] = take
		}
		for _, drop := range drops {
			id := strings.TrimSpace(drop.CandidateID)
			idx := indexOfContextCandidateInBatch(sess.ContextCandidates, batch.ID, id)
			if idx < 0 {
				summary.Results = append(summary.Results, ContextDispositionResult{Action: "drop", Message: "context candidate not found in batch: " + id})
				continue
			}
			if sess.ContextCandidates[idx].Status != ContextCandidatePending {
				summary.Results = append(summary.Results, skippedContextDispositionResult(sess.ContextCandidates[idx], "drop"))
				continue
			}
			if _, exists := takeByCandidate[idx]; exists {
				return false, ErrInvalidSession{Reason: "context candidate cannot be both taken and dropped: " + id}
			}
			dropByCandidate[idx] = drop
		}

		changed := false
		for _, idx := range pendingShortlistCandidatesForBatch(sess.ContextCandidates, batch.ID) {
			candidate := sess.ContextCandidates[idx]
			if takeAll {
				if _, explicitlyDropped := dropByCandidate[idx]; !explicitlyDropped {
					takeByCandidate[idx] = ContextDispositionTake{CandidateID: firstNonEmpty(candidate.Handle, candidate.ID)}
				}
			}
			if take, ok := takeByCandidate[idx]; ok {
				accepted, item, itemViolations, itemChanged, err := acceptContextCandidateAt(sess, idx, take.PhaseID)
				result := ContextDispositionResult{
					Candidate:  accepted,
					Item:       item,
					Action:     "take",
					Accepted:   itemChanged && len(itemViolations) == 0 && err == nil,
					Violations: itemViolations,
				}
				if err != nil {
					result.Message = err.Error()
				}
				if result.Accepted && strings.TrimSpace(take.Reason) != "" {
					result.Message = strings.TrimSpace(take.Reason)
				}
				summary.Results = append(summary.Results, result)
				if err != nil {
					return false, err
				}
				if len(itemViolations) > 0 {
					violations = append(violations, itemViolations...)
					continue
				}
				changed = changed || itemChanged
				continue
			}

			drop, explicitDrop := dropByCandidate[idx]
			reason := strings.TrimSpace(drop.Reason)
			if candidate.HighConfidence && reason == "" {
				summary.Results = append(summary.Results, ContextDispositionResult{
					Candidate: candidate,
					Action:    "drop",
					Accepted:  false,
					Message:   "high-confidence context candidate requires a drop reason",
				})
				continue
			}
			if reason == "" {
				if explicitDrop {
					reason = "dropped by context-apply"
				} else {
					reason = "swept by context-apply"
				}
			}
			dropped := rejectContextCandidateAt(sess, idx, reason)
			summary.Results = append(summary.Results, ContextDispositionResult{
				Candidate: dropped,
				Action:    "drop",
				Accepted:  true,
				Message:   reason,
			})
			changed = true
		}
		closeContextBatchIfResolved(sess, batchIdx, sweepNote)
		summary.Batch = sess.DiscoveryBatches[batchIdx]
		return changed || summary.Batch.Status == DiscoveryBatchApplied, nil
	})
	if err != nil {
		return Session{}, ContextDispositionSummary{}, nil, GuidedStep{}, err
	}
	return sess, summary, violations, stepForCurrentSessionState(sess), nil
}

func skippedContextDispositionResults(candidates []ContextCandidate, batchID string, takes []ContextDispositionTake, drops []ContextDispositionDrop) []ContextDispositionResult {
	out := make([]ContextDispositionResult, 0, len(takes)+len(drops))
	for _, take := range takes {
		id := strings.TrimSpace(take.CandidateID)
		idx := indexOfContextCandidateInBatch(candidates, batchID, id)
		if idx < 0 {
			out = append(out, ContextDispositionResult{Action: "take", Message: "context candidate not found in batch: " + id})
			continue
		}
		out = append(out, skippedContextDispositionResult(candidates[idx], "take"))
	}
	for _, drop := range drops {
		id := strings.TrimSpace(drop.CandidateID)
		idx := indexOfContextCandidateInBatch(candidates, batchID, id)
		if idx < 0 {
			out = append(out, ContextDispositionResult{Action: "drop", Message: "context candidate not found in batch: " + id})
			continue
		}
		out = append(out, skippedContextDispositionResult(candidates[idx], "drop"))
	}
	return out
}

func skippedContextDispositionResult(candidate ContextCandidate, action string) ContextDispositionResult {
	status := strings.TrimSpace(string(candidate.Status))
	if status == "" {
		status = "already dispositioned"
	} else {
		status = "already " + status
	}
	return ContextDispositionResult{
		Candidate: candidate,
		Item:      candidate.Item,
		Action:    action,
		Accepted:  true,
		Message:   status + "; skipped",
	}
}

func acceptContextCandidateAt(sess *Session, idx int, phaseID string) (ContextCandidate, planmodel.RelevantContextItem, []StructureViolation, bool, error) {
	candidate := sess.ContextCandidates[idx]
	if candidate.Status == ContextCandidateRejected {
		return candidate, planmodel.RelevantContextItem{}, nil, false, ErrInvalidSession{Reason: "context candidate was rejected: " + firstNonEmpty(candidate.Handle, candidate.ID)}
	}
	if candidate.Status == ContextCandidateAccepted {
		return candidate, candidate.Item, nil, false, nil
	}
	if violations := candidateReadinessViolations(candidate); len(violations) > 0 {
		return candidate, candidate.Item, violations, false, nil
	}
	item := normalizeContextItem(candidate.Item, phaseID)
	if phaseID != "" {
		phaseIdx := indexOfDraft(sess.PhaseDrafts, phaseID)
		if phaseIdx < 0 {
			return candidate, item, nil, false, ErrSectionNotFound{SessionID: sess.ID, SectionKey: "phase:" + phaseID}
		}
		item.Scope = planmodel.RelevantContextScopePhase
		item.PhaseID = sess.PhaseDrafts[phaseIdx].ID
		item.RepeatPolicy = defaultRepeatForScope(item.Scope, item.RepeatPolicy)
		violations := contextItemViolations(item)
		if len(violations) > 0 {
			return candidate, item, violations, false, nil
		}
		sess.PhaseDrafts[phaseIdx].RelevantContext = append(sess.PhaseDrafts[phaseIdx].RelevantContext, item)
		*sess = syncPhaseSection(*sess)
	} else {
		item.Scope = planmodel.RelevantContextScopeGlobal
		item.PhaseID = ""
		item.RepeatPolicy = defaultRepeatForScope(item.Scope, item.RepeatPolicy)
		violations := contextItemViolations(item)
		if len(violations) > 0 {
			return candidate, item, violations, false, nil
		}
		sess.RelevantContext = append(sess.RelevantContext, item)
		*sess = syncContextSection(*sess)
	}
	candidate.Status = ContextCandidateAccepted
	candidate.Item = item
	sess.ContextCandidates[idx] = candidate
	return candidate, item, nil, true, nil
}

func rejectContextCandidateAt(sess *Session, idx int, reason string) ContextCandidate {
	candidate := sess.ContextCandidates[idx]
	candidate.Status = ContextCandidateRejected
	candidate.RejectionReason = strings.TrimSpace(reason)
	sess.ContextCandidates[idx] = candidate
	return candidate
}

const (
	candidateValidationReady   = "ready"
	candidateValidationFailed  = "failed"
	candidateValidationUnknown = "unknown"
)

func (s *service) validateContextCandidate(ctx context.Context, candidate ContextCandidate) ContextCandidate {
	item := normalizeContextItem(candidate.Item, "")
	mode := readiness.Mode{}
	switch item.Kind {
	case planmodel.RelevantContextCommand, planmodel.RelevantContextSearch:
		mode.CommandReferences = true
	case planmodel.RelevantContextCodeRef, planmodel.RelevantContextDoc, planmodel.RelevantContextReqRef:
		mode.ContextReferences = true
	default:
		candidate.ValidationStatus = candidateValidationReady
		return candidate
	}
	result := readiness.Evaluate(ctx, planmodel.Plan{RelevantContext: []planmodel.RelevantContextItem{item}}, readiness.Options{
		Mode:              mode,
		CommandValidator:  commandReadinessAdapter{s.commands},
		ReferenceResolver: s.resolver,
	})
	candidate.ValidationStatus, candidate.ValidationDetail = candidateValidationFromReadiness(result)
	if candidate.ValidationStatus == candidateValidationUnknown {
		candidate.Degraded = true
	}
	if candidate.Detail == "" && candidate.ValidationDetail != "" {
		candidate.Detail = candidate.ValidationDetail
	}
	return candidate
}

func candidateValidationFromReadiness(result readiness.Result) (string, string) {
	detail := strings.TrimSpace(result.Detail)
	if detail == "" && len(result.Findings) > 0 {
		detail = result.Findings[0].Message
	}
	switch result.Verdict {
	case readiness.VerdictFail:
		return candidateValidationFailed, detail
	case readiness.VerdictUnknown:
		return candidateValidationUnknown, detail
	default:
		return candidateValidationReady, detail
	}
}

func candidateReadinessViolations(candidate ContextCandidate) []StructureViolation {
	switch candidate.ValidationStatus {
	case candidateValidationFailed, candidateValidationUnknown:
		return []StructureViolation{{
			SectionKey: SectionRelevantContext,
			Message:    fmt.Sprintf("context candidate %s is not ready (%s): %s", firstNonEmpty(candidate.Handle, candidate.ID), candidate.ValidationStatus, firstNonEmpty(candidate.ValidationDetail, candidate.Detail, "validation did not pass")),
		}}
	default:
		return nil
	}
}

type commandReadinessAdapter struct {
	validator CommandReferenceValidator
}

func (a commandReadinessAdapter) ValidateCommandReference(ctx context.Context, req readiness.CommandRequest) (readiness.CommandResult, error) {
	if a.validator == nil {
		return readiness.CommandResult{}, fmt.Errorf("CLI Health command validator unavailable")
	}
	got, err := a.validator.ValidateCommandReference(ctx, CommandReferenceRequest{
		CommandText: req.CommandText,
		Qualifiers:  append([]string(nil), req.Qualifiers...),
	})
	if err != nil {
		return readiness.CommandResult{}, err
	}
	issues := make([]readiness.CommandIssue, 0, len(got.Issues))
	for _, issue := range got.Issues {
		issues = append(issues, readiness.CommandIssue{Code: issue.Code, Message: issue.Message})
	}
	return readiness.CommandResult{
		Verdict:         got.Verdict,
		ValidationLevel: got.ValidationLevel,
		Issues:          issues,
		Suggestions:     append([]string(nil), got.Suggestions...),
		Guidance:        append([]string(nil), got.Guidance...),
	}, nil
}

// runAutofill runs one source against the session in place. It NEVER fabricates a
// fill: a nil seam or an error leaves the section untouched and returns
// Degraded=true with the honest reason.
