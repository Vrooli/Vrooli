package authoring

import (
	"context"
	"strings"

	planmodel "plan-manager/internal/planmodel"
)

func (s *service) SubmitRelevantContextItem(ctx context.Context, sessionID, phaseID string, item planmodel.RelevantContextItem) (Session, planmodel.RelevantContextItem, []StructureViolation, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, err
	}
	item = normalizeContextItem(item, phaseID)
	var violations []StructureViolation
	if phaseID != "" {
		idx := indexOfDraft(sess.PhaseDrafts, phaseID)
		if idx < 0 {
			return Session{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + phaseID}
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
		sess = syncPhaseSection(sess)
	} else {
		item.Scope = planmodel.RelevantContextScopeGlobal
		item.PhaseID = ""
		item.RepeatPolicy = defaultRepeatForScope(item.Scope, item.RepeatPolicy)
		violations = contextItemViolations(item)
		if len(violations) == 0 {
			sess.RelevantContext = append(sess.RelevantContext, item)
			sess = syncContextSection(sess)
		}
	}
	sess.UpdatedAt = s.now()
	if len(violations) == 0 {
		if err := s.store.Save(ctx, sess); err != nil {
			return Session{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, err
		}
	}
	return sess, item, violations, stepForCurrentSessionState(sess), nil
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
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, err
	}
	if sess.Finalized {
		return Session{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "relevant context cannot be edited after finalize"}
	}
	itemID = strings.TrimSpace(itemID)
	if itemID == "" {
		return Session{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "item_id is required"}
	}
	item.ID = itemID
	item = normalizeContextItem(item, phaseID)
	if phaseID != "" {
		phaseIdx := indexOfDraft(sess.PhaseDrafts, phaseID)
		if phaseIdx < 0 {
			return Session{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + phaseID}
		}
		item.Scope = planmodel.RelevantContextScopePhase
		item.PhaseID = sess.PhaseDrafts[phaseIdx].ID
		item.RepeatPolicy = defaultRepeatForScope(item.Scope, item.RepeatPolicy)
		pos := indexOfContextItem(sess.PhaseDrafts[phaseIdx].RelevantContext, itemID)
		if pos < 0 {
			return Session{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "phase context item not found: " + itemID}
		}
		if violations := contextItemViolations(item); len(violations) > 0 {
			return sess, item, violations, stepForCurrentSessionState(sess), nil
		}
		sess.PhaseDrafts[phaseIdx].RelevantContext[pos] = item
		sess = syncPhaseSection(sess)
	} else {
		item.Scope = planmodel.RelevantContextScopeGlobal
		item.PhaseID = ""
		item.RepeatPolicy = defaultRepeatForScope(item.Scope, item.RepeatPolicy)
		pos := indexOfContextItem(sess.RelevantContext, itemID)
		if pos < 0 {
			return Session{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "global context item not found: " + itemID}
		}
		if violations := contextItemViolations(item); len(violations) > 0 {
			return sess, item, violations, stepForCurrentSessionState(sess), nil
		}
		sess.RelevantContext[pos] = item
		sess = syncContextSection(sess)
	}
	sess.UpdatedAt = s.now()
	if err := s.store.Save(ctx, sess); err != nil {
		return Session{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, err
	}
	return sess, item, nil, stepForCurrentSessionState(sess), nil
}

// RemoveRelevantContextItem deletes one accepted context item (by id) before
// finalize, recomputing structure violations so a resulting gate (e.g. a phase
// left with no context) is reported with its recovery step.
func (s *service) RemoveRelevantContextItem(ctx context.Context, sessionID, phaseID, itemID string) (Session, []StructureViolation, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, nil, GuidedStep{}, err
	}
	if sess.Finalized {
		return Session{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "relevant context cannot be edited after finalize"}
	}
	itemID = strings.TrimSpace(itemID)
	if itemID == "" {
		return Session{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "item_id is required"}
	}
	var violations []StructureViolation
	if phaseID != "" {
		phaseIdx := indexOfDraft(sess.PhaseDrafts, phaseID)
		if phaseIdx < 0 {
			return Session{}, nil, GuidedStep{}, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + phaseID}
		}
		pos := indexOfContextItem(sess.PhaseDrafts[phaseIdx].RelevantContext, itemID)
		if pos < 0 {
			return Session{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "phase context item not found: " + itemID}
		}
		sess.PhaseDrafts[phaseIdx].RelevantContext = removeContextItemAt(sess.PhaseDrafts[phaseIdx].RelevantContext, pos)
		sess.CurrentPhaseID = nextIncompletePhaseID(sess.PhaseDrafts)
		sess = syncPhaseSection(sess)
		violations = phaseViolations(sess.PhaseDrafts[phaseIdx])
	} else {
		pos := indexOfContextItem(sess.RelevantContext, itemID)
		if pos < 0 {
			return Session{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "global context item not found: " + itemID}
		}
		sess.RelevantContext = removeContextItemAt(sess.RelevantContext, pos)
		sess = syncContextSection(sess)
	}
	sess.UpdatedAt = s.now()
	if err := s.store.Save(ctx, sess); err != nil {
		return Session{}, nil, GuidedStep{}, err
	}
	step := stepForCurrentSessionState(sess)
	if phaseID == "" && !globalContextResolved(sess) {
		step = stepForGlobalContextCheckpoint(sess)
	}
	return sess, violations, step, nil
}

func (s *service) DiscoverContextCandidates(ctx context.Context, sessionID string, concepts []string, complexity string) (Session, []ContextCandidate, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, nil, GuidedStep{}, err
	}
	var candidates []ContextCandidate
	if s.context == nil {
		candidates = degradedContextCandidates(sess.Title, concepts, "context discovery unavailable")
	} else {
		candidates, err = s.context.DiscoverContext(ctx, sess.Title, concepts, complexity)
		if err != nil {
			candidates = degradedContextCandidates(sess.Title, concepts, err.Error())
		}
	}
	for i := range candidates {
		candidates[i] = normalizeContextCandidate(candidates[i])
	}
	sess.ContextCandidates = append(sess.ContextCandidates, candidates...)
	sess.UpdatedAt = s.now()
	if err := s.store.Save(ctx, sess); err != nil {
		return Session{}, nil, GuidedStep{}, err
	}
	return sess, candidates, stepForContextDiscovery(sess), nil
}

func (s *service) AcceptContextCandidate(ctx context.Context, sessionID, candidateID, phaseID string) (Session, ContextCandidate, planmodel.RelevantContextItem, []StructureViolation, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, ContextCandidate{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, err
	}
	idx := indexOfCandidate(sess.ContextCandidates, candidateID)
	if idx < 0 {
		return Session{}, ContextCandidate{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "context candidate not found: " + candidateID}
	}
	candidate := sess.ContextCandidates[idx]
	if candidate.Status == ContextCandidateRejected {
		return Session{}, ContextCandidate{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "context candidate was rejected: " + candidateID}
	}
	item := normalizeContextItem(candidate.Item, phaseID)
	var violations []StructureViolation
	if phaseID != "" {
		phaseIdx := indexOfDraft(sess.PhaseDrafts, phaseID)
		if phaseIdx < 0 {
			return Session{}, ContextCandidate{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + phaseID}
		}
		item.Scope = planmodel.RelevantContextScopePhase
		item.PhaseID = sess.PhaseDrafts[phaseIdx].ID
		item.RepeatPolicy = defaultRepeatForScope(item.Scope, item.RepeatPolicy)
		violations = contextItemViolations(item)
		if len(violations) == 0 {
			sess.PhaseDrafts[phaseIdx].RelevantContext = append(sess.PhaseDrafts[phaseIdx].RelevantContext, item)
			sess = syncPhaseSection(sess)
		}
	} else {
		item.Scope = planmodel.RelevantContextScopeGlobal
		item.PhaseID = ""
		item.RepeatPolicy = defaultRepeatForScope(item.Scope, item.RepeatPolicy)
		violations = contextItemViolations(item)
		if len(violations) == 0 {
			sess.RelevantContext = append(sess.RelevantContext, item)
			sess = syncContextSection(sess)
		}
	}
	if len(violations) > 0 {
		return sess, candidate, item, violations, stepForContextDiscovery(sess), nil
	}
	candidate.Status = ContextCandidateAccepted
	candidate.Item = item
	sess.ContextCandidates[idx] = candidate
	sess.UpdatedAt = s.now()
	if err := s.store.Save(ctx, sess); err != nil {
		return Session{}, ContextCandidate{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, err
	}
	return sess, candidate, item, nil, stepForCurrentSessionState(sess), nil
}

func (s *service) RejectContextCandidate(ctx context.Context, sessionID, candidateID, reason string) (Session, ContextCandidate, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, ContextCandidate{}, GuidedStep{}, err
	}
	idx := indexOfCandidate(sess.ContextCandidates, candidateID)
	if idx < 0 {
		return Session{}, ContextCandidate{}, GuidedStep{}, ErrInvalidSession{Reason: "context candidate not found: " + candidateID}
	}
	candidate := sess.ContextCandidates[idx]
	candidate.Status = ContextCandidateRejected
	candidate.RejectionReason = strings.TrimSpace(reason)
	sess.ContextCandidates[idx] = candidate
	sess.UpdatedAt = s.now()
	if err := s.store.Save(ctx, sess); err != nil {
		return Session{}, ContextCandidate{}, GuidedStep{}, err
	}
	return sess, candidate, stepForContextDiscovery(sess), nil
}

// runAutofill runs one source against the session in place. It NEVER fabricates a
// fill: a nil seam or an error leaves the section untouched and returns
// Degraded=true with the honest reason.
