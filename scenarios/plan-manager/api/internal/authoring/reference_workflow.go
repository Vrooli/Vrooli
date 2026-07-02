package authoring

import (
	"context"
	"fmt"
	"strings"

	planmodel "plan-manager/internal/planmodel"
	"plan-manager/internal/readiness"
)

func (s *service) SuggestReferences(ctx context.Context, sessionID string) (Session, []ReferenceCandidate, GuidedStep, error) {
	// The suggester shells out to search-hub — run it against a pre-lock read so
	// a slow query never holds the session lock; only the append is locked.
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, nil, GuidedStep{}, err
	}
	query := referenceSuggestionQuery(sess)
	var candidates []ReferenceCandidate
	if s.suggester == nil {
		candidates = nil
	} else if found, suggestErr := s.suggester.Suggest(ctx, query); suggestErr == nil {
		candidates = found
	}
	for i := range candidates {
		candidates[i] = normalizeReferenceCandidate(candidates[i])
		candidates[i] = s.validateReferenceCandidate(ctx, candidates[i])
	}
	var batch DiscoveryBatch
	sess, err = s.withSessionLock(ctx, sessionID, func(sess *Session) (bool, error) {
		batch = mergeReferenceDiscoveryBatch(sess, candidates)
		return true, nil
	})
	if err != nil {
		return Session{}, nil, GuidedStep{}, err
	}
	return sess, referenceCandidatesForBatch(sess.ReferenceCandidates, batch.ID), stepForReferenceCandidates(sess), nil
}

// ListReferenceCandidates returns the session's reference candidates without
// changing wizard position.
func (s *service) ListReferenceCandidates(ctx context.Context, sessionID string) ([]ReferenceCandidate, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return nil, GuidedStep{}, err
	}
	return append([]ReferenceCandidate(nil), sess.ReferenceCandidates...), stepForReferenceCandidates(sess), nil
}

// AcceptReferenceCandidate promotes one pending reference candidate into the
// references section (with an optional inline edit of the locator). The accepted
// locator is appended to the section so the references gate (which reads the
// section for [CODE:]/[DOC:]/[REQ:] locators) passes only on reviewed state. A
// kind/path mismatch is rejected before the locator enters the section.
func (s *service) AcceptReferenceCandidate(ctx context.Context, sessionID, candidateID string, edit *planmodel.Reference) (Session, ReferenceCandidate, []StructureViolation, GuidedStep, error) {
	var (
		candidate  ReferenceCandidate
		violations []StructureViolation
	)
	sess, err := s.withSessionLock(ctx, sessionID, func(sess *Session) (bool, error) {
		idx := indexOfReferenceCandidate(sess.ReferenceCandidates, candidateID)
		if idx < 0 {
			return false, ErrInvalidSession{Reason: "reference candidate not found: " + candidateID}
		}
		var changed bool
		var acceptErr error
		candidate, _, violations, changed, acceptErr = acceptReferenceCandidateAt(sess, idx, edit)
		if acceptErr != nil || len(violations) > 0 {
			return false, acceptErr
		}
		if changed {
			if batchIdx := indexOfDiscoveryBatch(sess.ReferenceBatches, candidate.BatchID); batchIdx >= 0 {
				closeReferenceBatchIfResolved(sess, batchIdx, "applied item-by-item")
			}
		}
		return changed, nil
	})
	if err != nil {
		return Session{}, ReferenceCandidate{}, nil, GuidedStep{}, err
	}
	if len(violations) > 0 {
		return sess, candidate, violations, stepForReferenceCandidates(sess), nil
	}
	return sess, candidate, nil, stepForCurrentSessionState(sess), nil
}

// RejectReferenceCandidate records why a suggested reference is not relevant. The
// rejected candidate stays as an authoring audit trail; it never enters the
// references section.
func (s *service) RejectReferenceCandidate(ctx context.Context, sessionID, candidateID, reason string) (Session, ReferenceCandidate, GuidedStep, error) {
	if strings.TrimSpace(reason) == "" {
		return Session{}, ReferenceCandidate{}, GuidedStep{}, ErrInvalidSession{Reason: "reference candidate rejection requires a --reason"}
	}
	var candidate ReferenceCandidate
	sess, err := s.withSessionLock(ctx, sessionID, func(sess *Session) (bool, error) {
		idx := indexOfReferenceCandidate(sess.ReferenceCandidates, candidateID)
		if idx < 0 {
			return false, ErrInvalidSession{Reason: "reference candidate not found: " + candidateID}
		}
		candidate = rejectReferenceCandidateAt(sess, idx, strings.TrimSpace(reason))
		if batchIdx := indexOfDiscoveryBatch(sess.ReferenceBatches, candidate.BatchID); batchIdx >= 0 {
			closeReferenceBatchIfResolved(sess, batchIdx, "applied item-by-item")
		}
		return true, nil
	})
	if err != nil {
		return Session{}, ReferenceCandidate{}, GuidedStep{}, err
	}
	return sess, candidate, stepForReferenceCandidates(sess), nil
}

func (s *service) ApplyReferenceDisposition(ctx context.Context, sessionID, batchID string, takes []ReferenceDispositionTake, drops []ReferenceDispositionDrop, sweepNote string, takeAll bool) (Session, ReferenceDispositionSummary, []StructureViolation, GuidedStep, error) {
	var (
		summary    ReferenceDispositionSummary
		violations []StructureViolation
	)
	sess, err := s.withSessionLock(ctx, sessionID, func(sess *Session) (bool, error) {
		batchIdx := targetReferenceBatchIndex(*sess, batchID)
		if batchIdx < 0 {
			return false, ErrInvalidSession{Reason: "pending reference discovery batch not found"}
		}
		if sess.ReferenceBatches[batchIdx].Status == DiscoveryBatchApplied {
			batch := sess.ReferenceBatches[batchIdx]
			summary.Results = append(summary.Results, skippedReferenceDispositionResults(sess.ReferenceCandidates, batch.ID, takes, drops)...)
			summary.Batch = batch
			return false, nil
		}
		if sess.ReferenceBatches[batchIdx].Status != DiscoveryBatchPending {
			return false, ErrInvalidSession{Reason: "reference discovery batch is not pending: " + sess.ReferenceBatches[batchIdx].ID}
		}
		batch := sess.ReferenceBatches[batchIdx]
		takeByCandidate := map[int]ReferenceDispositionTake{}
		dropByCandidate := map[int]ReferenceDispositionDrop{}
		for _, take := range takes {
			id := strings.TrimSpace(take.CandidateID)
			idx := indexOfReferenceCandidateInBatch(sess.ReferenceCandidates, batch.ID, id)
			if idx < 0 {
				summary.Results = append(summary.Results, ReferenceDispositionResult{Action: "take", Message: "reference candidate not found in batch: " + id})
				continue
			}
			if sess.ReferenceCandidates[idx].Status != ReferenceCandidatePending {
				summary.Results = append(summary.Results, skippedReferenceDispositionResult(sess.ReferenceCandidates[idx], "take"))
				continue
			}
			takeByCandidate[idx] = take
		}
		for _, drop := range drops {
			id := strings.TrimSpace(drop.CandidateID)
			idx := indexOfReferenceCandidateInBatch(sess.ReferenceCandidates, batch.ID, id)
			if idx < 0 {
				summary.Results = append(summary.Results, ReferenceDispositionResult{Action: "drop", Message: "reference candidate not found in batch: " + id})
				continue
			}
			if sess.ReferenceCandidates[idx].Status != ReferenceCandidatePending {
				summary.Results = append(summary.Results, skippedReferenceDispositionResult(sess.ReferenceCandidates[idx], "drop"))
				continue
			}
			if _, exists := takeByCandidate[idx]; exists {
				return false, ErrInvalidSession{Reason: "reference candidate cannot be both taken and dropped: " + id}
			}
			dropByCandidate[idx] = drop
		}

		changed := false
		for _, idx := range pendingShortlistReferencesForBatch(sess.ReferenceCandidates, batch.ID) {
			candidate := sess.ReferenceCandidates[idx]
			if takeAll {
				if _, explicitlyDropped := dropByCandidate[idx]; !explicitlyDropped {
					takeByCandidate[idx] = ReferenceDispositionTake{CandidateID: firstNonEmpty(candidate.Handle, candidate.ID)}
				}
			}
			if _, ok := takeByCandidate[idx]; ok {
				accepted, ref, itemViolations, itemChanged, err := acceptReferenceCandidateAt(sess, idx, nil)
				result := ReferenceDispositionResult{
					Candidate:  accepted,
					Reference:  ref,
					Action:     "take",
					Accepted:   itemChanged && len(itemViolations) == 0 && err == nil,
					Violations: itemViolations,
				}
				if err != nil {
					result.Message = err.Error()
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
				summary.Results = append(summary.Results, ReferenceDispositionResult{
					Candidate: candidate,
					Action:    "drop",
					Message:   "high-confidence reference candidate requires a drop reason",
				})
				continue
			}
			if reason == "" {
				if explicitDrop {
					reason = "dropped by reference-apply"
				} else {
					reason = "swept by reference-apply"
				}
			}
			dropped := rejectReferenceCandidateAt(sess, idx, reason)
			summary.Results = append(summary.Results, ReferenceDispositionResult{
				Candidate: dropped,
				Reference: dropped.Reference,
				Action:    "drop",
				Accepted:  true,
				Message:   reason,
			})
			changed = true
		}
		closeReferenceBatchIfResolved(sess, batchIdx, sweepNote)
		summary.Batch = sess.ReferenceBatches[batchIdx]
		return changed || summary.Batch.Status == DiscoveryBatchApplied, nil
	})
	if err != nil {
		return Session{}, ReferenceDispositionSummary{}, nil, GuidedStep{}, err
	}
	return sess, summary, violations, stepForCurrentSessionState(sess), nil
}

func skippedReferenceDispositionResults(candidates []ReferenceCandidate, batchID string, takes []ReferenceDispositionTake, drops []ReferenceDispositionDrop) []ReferenceDispositionResult {
	out := make([]ReferenceDispositionResult, 0, len(takes)+len(drops))
	for _, take := range takes {
		id := strings.TrimSpace(take.CandidateID)
		idx := indexOfReferenceCandidateInBatch(candidates, batchID, id)
		if idx < 0 {
			out = append(out, ReferenceDispositionResult{Action: "take", Message: "reference candidate not found in batch: " + id})
			continue
		}
		out = append(out, skippedReferenceDispositionResult(candidates[idx], "take"))
	}
	for _, drop := range drops {
		id := strings.TrimSpace(drop.CandidateID)
		idx := indexOfReferenceCandidateInBatch(candidates, batchID, id)
		if idx < 0 {
			out = append(out, ReferenceDispositionResult{Action: "drop", Message: "reference candidate not found in batch: " + id})
			continue
		}
		out = append(out, skippedReferenceDispositionResult(candidates[idx], "drop"))
	}
	return out
}

func skippedReferenceDispositionResult(candidate ReferenceCandidate, action string) ReferenceDispositionResult {
	status := strings.TrimSpace(string(candidate.Status))
	if status == "" {
		status = "already dispositioned"
	} else {
		status = "already " + status
	}
	return ReferenceDispositionResult{
		Candidate: candidate,
		Reference: candidate.Reference,
		Action:    action,
		Accepted:  true,
		Message:   status + "; skipped",
	}
}

func acceptReferenceCandidateAt(sess *Session, idx int, edit *planmodel.Reference) (ReferenceCandidate, planmodel.Reference, []StructureViolation, bool, error) {
	candidate := sess.ReferenceCandidates[idx]
	if candidate.Status == ReferenceCandidateRejected {
		return candidate, planmodel.Reference{}, nil, false, ErrInvalidSession{Reason: "reference candidate was rejected: " + firstNonEmpty(candidate.Handle, candidate.ID)}
	}
	if candidate.Status == ReferenceCandidateAccepted {
		return candidate, candidate.Reference, nil, false, nil
	}
	if violations := referenceCandidateReadinessViolations(candidate); len(violations) > 0 {
		return candidate, candidate.Reference, violations, false, nil
	}
	ref := candidate.Reference
	if edit != nil {
		if edit.Kind != "" {
			ref.Kind = edit.Kind
		}
		if strings.TrimSpace(edit.Target) != "" {
			ref.Target = strings.TrimSpace(edit.Target)
		}
		ref.Future = edit.Future
	}
	if ref.Kind == "" {
		ref.Kind = planmodel.ReferenceCode
	}
	if strings.TrimSpace(ref.Target) == "" {
		return candidate, ref, []StructureViolation{{SectionKey: SectionReferences, Message: "reference candidate has no target locator"}}, false, nil
	}
	if msg := referenceKindMismatch(ref.Kind, ref.Target); msg != "" {
		return candidate, ref, []StructureViolation{{SectionKey: SectionReferences, Message: msg}}, false, nil
	}
	candidate.Reference = ref
	candidate.Status = ReferenceCandidateAccepted
	sess.ReferenceCandidates[idx] = candidate
	appendAcceptedReference(sess, ref)
	sess.CurrentSectionKey = firstUnfilledMandatory(sess.Sections)
	return candidate, ref, nil, true, nil
}

func rejectReferenceCandidateAt(sess *Session, idx int, reason string) ReferenceCandidate {
	candidate := sess.ReferenceCandidates[idx]
	candidate.Status = ReferenceCandidateRejected
	candidate.RejectionReason = strings.TrimSpace(reason)
	sess.ReferenceCandidates[idx] = candidate
	return candidate
}

func (s *service) validateReferenceCandidate(ctx context.Context, candidate ReferenceCandidate) ReferenceCandidate {
	ref := candidate.Reference
	if ref.Kind == "" || strings.TrimSpace(ref.Target) == "" {
		candidate.ValidationStatus = candidateValidationFailed
		candidate.ValidationDetail = "reference candidate has no target locator"
		return candidate
	}
	if msg := referenceKindMismatch(ref.Kind, ref.Target); msg != "" {
		candidate.ValidationStatus = candidateValidationFailed
		candidate.ValidationDetail = msg
		return candidate
	}
	item := planmodel.RelevantContextItem{
		Kind:   relevantContextKindForReference(ref.Kind),
		Target: strings.TrimSpace(ref.Target),
	}
	if item.Kind == "" {
		candidate.ValidationStatus = candidateValidationReady
		return candidate
	}
	result := readiness.Evaluate(ctx, planmodel.Plan{RelevantContext: []planmodel.RelevantContextItem{item}}, readiness.Options{
		Mode:              readiness.Mode{ContextReferences: true},
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

func relevantContextKindForReference(kind planmodel.ReferenceKind) planmodel.RelevantContextKind {
	switch kind {
	case planmodel.ReferenceCode:
		return planmodel.RelevantContextCodeRef
	case planmodel.ReferenceDoc:
		return planmodel.RelevantContextDoc
	case planmodel.ReferenceReq:
		return planmodel.RelevantContextReqRef
	default:
		return ""
	}
}

func referenceCandidateReadinessViolations(candidate ReferenceCandidate) []StructureViolation {
	switch candidate.ValidationStatus {
	case candidateValidationFailed, candidateValidationUnknown:
		return []StructureViolation{{
			SectionKey: SectionReferences,
			Message:    fmt.Sprintf("reference candidate %s is not ready (%s): %s", firstNonEmpty(candidate.Handle, candidate.ID), candidate.ValidationStatus, firstNonEmpty(candidate.ValidationDetail, candidate.Detail, "validation did not pass")),
		}}
	default:
		return nil
	}
}
