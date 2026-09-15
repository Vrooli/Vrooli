package authoring

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

func (s *service) AddPhase(ctx context.Context, sessionID string, title, intent string) (Session, PhaseDraft, []StructureViolation, GuidedStep, error) {
	var phase PhaseDraft
	sess, err := s.withSessionLock(ctx, sessionID, func(sess *Session) (bool, error) {
		phase = PhaseDraft{
			ID:     uuid.NewString(),
			Order:  len(sess.PhaseDrafts) + 1,
			Title:  strings.TrimSpace(title),
			Intent: strings.TrimSpace(intent),
		}
		sess.PhaseDrafts = append(sess.PhaseDrafts, phase)
		if sess.CurrentPhaseID == "" {
			sess.CurrentPhaseID = phase.ID
		}
		*sess = syncPhaseSection(*sess)
		return true, nil
	})
	if err != nil {
		return Session{}, PhaseDraft{}, nil, GuidedStep{}, err
	}
	return sess, phase, phaseViolations(phase), stepForPhase(sess, phase), nil
}

func (s *service) MovePhase(ctx context.Context, sessionID, phaseID, beforePhaseID, afterPhaseID string) (Session, PhaseDraft, []StructureViolation, GuidedStep, error) {
	phaseID = strings.TrimSpace(phaseID)
	beforePhaseID = strings.TrimSpace(beforePhaseID)
	afterPhaseID = strings.TrimSpace(afterPhaseID)
	if phaseID == "" {
		return Session{}, PhaseDraft{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "phase id is required"}
	}
	if (beforePhaseID == "") == (afterPhaseID == "") {
		return Session{}, PhaseDraft{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "provide exactly one of before or after phase id"}
	}
	var moved PhaseDraft
	sess, err := s.withSessionLock(ctx, sessionID, func(sess *Session) (bool, error) {
		from := indexOfDraft(sess.PhaseDrafts, phaseID)
		if from < 0 {
			return false, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + phaseID}
		}
		targetID := firstNonEmpty(beforePhaseID, afterPhaseID)
		target := indexOfDraft(sess.PhaseDrafts, targetID)
		if target < 0 {
			return false, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + targetID}
		}
		if from == target {
			moved = sess.PhaseDrafts[from]
			return false, nil
		}
		phase := sess.PhaseDrafts[from]
		remaining := append([]PhaseDraft{}, sess.PhaseDrafts[:from]...)
		remaining = append(remaining, sess.PhaseDrafts[from+1:]...)
		target = indexOfDraft(remaining, targetID)
		if target < 0 {
			return false, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + targetID}
		}
		insertAt := target
		if afterPhaseID != "" {
			insertAt = target + 1
		}
		reordered := append([]PhaseDraft{}, remaining[:insertAt]...)
		reordered = append(reordered, phase)
		reordered = append(reordered, remaining[insertAt:]...)
		renumberPhaseDrafts(reordered)
		sess.PhaseDrafts = reordered
		sess.CurrentPhaseID = nextIncompletePhaseID(sess.PhaseDrafts)
		*sess = syncPhaseSection(*sess)
		moved, _ = findDraft(sess.PhaseDrafts, phase.ID)
		return true, nil
	})
	if err != nil {
		return Session{}, PhaseDraft{}, nil, GuidedStep{}, err
	}
	return sess, moved, phaseViolations(moved), stepForPhase(sess, moved), nil
}

func (s *service) GetPhase(ctx context.Context, sessionID, phaseID string) (PhaseDraft, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return PhaseDraft{}, GuidedStep{}, err
	}
	phase, ok := findDraft(sess.PhaseDrafts, phaseID)
	if !ok {
		return PhaseDraft{}, GuidedStep{}, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + phaseID}
	}
	return phase, stepForPhase(sess, phase), nil
}

// SubmitPhaseField is the single-item wrapper over the batch apply path
// (applyFieldWrite) — one code path, no drift possible. A quality-rejected
// write (acceptance duplicating validation) leaves the session unchanged and
// reports the violation; the phase's remaining gaps are reported as before.
func (s *service) SubmitPhaseField(ctx context.Context, sessionID, phaseID string, field PhaseField, content string) (Session, []StructureViolation, GuidedStep, error) {
	var violations []StructureViolation
	var idx int
	sess, err := s.withSessionLock(ctx, sessionID, func(sess *Session) (bool, error) {
		idx = indexOfDraft(sess.PhaseDrafts, phaseID)
		if idx < 0 {
			return false, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + phaseID}
		}
		result, err := s.applyFieldWrite(ctx, sess, FieldWrite{PhaseRef: phaseID, PhaseField: field, Content: content})
		if err != nil {
			return false, err
		}
		if !result.Accepted {
			violations = result.Violations
			return false, nil
		}
		sess.CurrentPhaseID = nextIncompletePhaseID(sess.PhaseDrafts)
		*sess = syncPhaseSection(*sess)
		violations = phaseViolations(sess.PhaseDrafts[idx])
		return true, nil
	})
	if err != nil {
		return Session{}, nil, GuidedStep{}, err
	}
	return sess, violations, stepForNextPhaseState(sess, sess.PhaseDrafts[idx]), nil
}

func (s *service) NextPhase(ctx context.Context, sessionID string) (PhaseDraft, GuidedStep, bool, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return PhaseDraft{}, GuidedStep{}, false, err
	}
	id := nextIncompletePhaseID(sess.PhaseDrafts)
	if id == "" {
		return PhaseDraft{}, stepForReview(sess), true, nil
	}
	phase, ok := findDraft(sess.PhaseDrafts, id)
	if !ok {
		return PhaseDraft{}, GuidedStep{}, false, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + id}
	}
	return phase, stepForPhase(sess, phase), false, nil
}

// PreviewPlan renders the in-progress session to its markdown review artifact
// WITHOUT persisting — the render-preview the wizard offers before finalize. It
// maps the session through the same sessionToPlan path Finalize uses, so the
// preview matches what will be saved (posture is filled in on save; the preview
// shows the default greenfield block). Malformed authored markup surfaces as a
// typed error so the agent fixes it before finalizing.
