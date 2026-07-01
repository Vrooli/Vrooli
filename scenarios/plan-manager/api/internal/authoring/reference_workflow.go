package authoring

import (
	"context"
	"strings"

	planmodel "plan-manager/internal/planmodel"
)

func (s *service) SuggestReferences(ctx context.Context, sessionID string) (Session, []ReferenceCandidate, GuidedStep, error) {
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
	}
	sess.ReferenceCandidates = append(sess.ReferenceCandidates, candidates...)
	sess.UpdatedAt = s.now()
	if err := s.store.Save(ctx, sess); err != nil {
		return Session{}, nil, GuidedStep{}, err
	}
	return sess, candidates, stepForReferenceCandidates(sess), nil
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
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, ReferenceCandidate{}, nil, GuidedStep{}, err
	}
	idx := indexOfReferenceCandidate(sess.ReferenceCandidates, candidateID)
	if idx < 0 {
		return Session{}, ReferenceCandidate{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "reference candidate not found: " + candidateID}
	}
	candidate := sess.ReferenceCandidates[idx]
	if candidate.Status == ReferenceCandidateRejected {
		return Session{}, ReferenceCandidate{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "reference candidate was rejected: " + candidateID}
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
		return sess, candidate, []StructureViolation{{SectionKey: SectionReferences, Message: "reference candidate has no target locator"}}, stepForReferenceCandidates(sess), nil
	}
	if msg := referenceKindMismatch(ref.Kind, ref.Target); msg != "" {
		return sess, candidate, []StructureViolation{{SectionKey: SectionReferences, Message: msg}}, stepForReferenceCandidates(sess), nil
	}
	candidate.Reference = ref
	candidate.Status = ReferenceCandidateAccepted
	sess.ReferenceCandidates[idx] = candidate
	appendAcceptedReference(&sess, ref)
	sess.CurrentSectionKey = firstUnfilledMandatory(sess.Sections)
	sess.UpdatedAt = s.now()
	if err := s.store.Save(ctx, sess); err != nil {
		return Session{}, ReferenceCandidate{}, nil, GuidedStep{}, err
	}
	return sess, candidate, nil, stepForCurrentSessionState(sess), nil
}

// RejectReferenceCandidate records why a suggested reference is not relevant. The
// rejected candidate stays as an authoring audit trail; it never enters the
// references section.
func (s *service) RejectReferenceCandidate(ctx context.Context, sessionID, candidateID, reason string) (Session, ReferenceCandidate, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, ReferenceCandidate{}, GuidedStep{}, err
	}
	idx := indexOfReferenceCandidate(sess.ReferenceCandidates, candidateID)
	if idx < 0 {
		return Session{}, ReferenceCandidate{}, GuidedStep{}, ErrInvalidSession{Reason: "reference candidate not found: " + candidateID}
	}
	candidate := sess.ReferenceCandidates[idx]
	candidate.Status = ReferenceCandidateRejected
	candidate.RejectionReason = strings.TrimSpace(reason)
	sess.ReferenceCandidates[idx] = candidate
	sess.UpdatedAt = s.now()
	if err := s.store.Save(ctx, sess); err != nil {
		return Session{}, ReferenceCandidate{}, GuidedStep{}, err
	}
	return sess, candidate, stepForReferenceCandidates(sess), nil
}
