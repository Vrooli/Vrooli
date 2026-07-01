package authoring

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

func (s *service) AddPhase(ctx context.Context, sessionID string, title, intent string) (Session, PhaseDraft, []StructureViolation, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, PhaseDraft{}, nil, GuidedStep{}, err
	}
	unlock := s.lockSession(sess.ID)
	defer unlock()
	sess, err = s.load(ctx, sess.ID)
	if err != nil {
		return Session{}, PhaseDraft{}, nil, GuidedStep{}, err
	}
	title = strings.TrimSpace(title)
	intent = strings.TrimSpace(intent)
	phase := PhaseDraft{
		ID:     uuid.NewString(),
		Order:  len(sess.PhaseDrafts) + 1,
		Title:  title,
		Intent: intent,
	}
	sess.PhaseDrafts = append(sess.PhaseDrafts, phase)
	if sess.CurrentPhaseID == "" {
		sess.CurrentPhaseID = phase.ID
	}
	sess = syncPhaseSection(sess)
	sess.UpdatedAt = s.now()
	violations := phaseViolations(phase)
	if err := s.store.Save(ctx, sess); err != nil {
		return Session{}, PhaseDraft{}, nil, GuidedStep{}, err
	}
	return sess, phase, violations, stepForPhase(sess, phase), nil
}

func (s *service) MovePhase(ctx context.Context, sessionID, phaseID, beforePhaseID, afterPhaseID string) (Session, PhaseDraft, []StructureViolation, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, PhaseDraft{}, nil, GuidedStep{}, err
	}
	unlock := s.lockSession(sess.ID)
	defer unlock()
	sess, err = s.load(ctx, sess.ID)
	if err != nil {
		return Session{}, PhaseDraft{}, nil, GuidedStep{}, err
	}
	phaseID = strings.TrimSpace(phaseID)
	beforePhaseID = strings.TrimSpace(beforePhaseID)
	afterPhaseID = strings.TrimSpace(afterPhaseID)
	if phaseID == "" {
		return Session{}, PhaseDraft{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "phase id is required"}
	}
	if (beforePhaseID == "") == (afterPhaseID == "") {
		return Session{}, PhaseDraft{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "provide exactly one of before or after phase id"}
	}
	from := indexOfDraft(sess.PhaseDrafts, phaseID)
	if from < 0 {
		return Session{}, PhaseDraft{}, nil, GuidedStep{}, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + phaseID}
	}
	targetID := firstNonEmpty(beforePhaseID, afterPhaseID)
	target := indexOfDraft(sess.PhaseDrafts, targetID)
	if target < 0 {
		return Session{}, PhaseDraft{}, nil, GuidedStep{}, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + targetID}
	}
	if from == target {
		phase := sess.PhaseDrafts[from]
		return sess, phase, phaseViolations(phase), stepForPhase(sess, phase), nil
	}
	phase := sess.PhaseDrafts[from]
	remaining := append([]PhaseDraft{}, sess.PhaseDrafts[:from]...)
	remaining = append(remaining, sess.PhaseDrafts[from+1:]...)
	target = indexOfDraft(remaining, targetID)
	if target < 0 {
		return Session{}, PhaseDraft{}, nil, GuidedStep{}, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + targetID}
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
	sess = syncPhaseSection(sess)
	sess.UpdatedAt = s.now()
	moved, _ := findDraft(sess.PhaseDrafts, phase.ID)
	violations := phaseViolations(moved)
	if err := s.store.Save(ctx, sess); err != nil {
		return Session{}, PhaseDraft{}, nil, GuidedStep{}, err
	}
	return sess, moved, violations, stepForPhase(sess, moved), nil
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

func (s *service) SubmitPhaseField(ctx context.Context, sessionID, phaseID string, field PhaseField, content string) (Session, []StructureViolation, GuidedStep, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, nil, GuidedStep{}, err
	}
	unlock := s.lockSession(sess.ID)
	defer unlock()
	sess, err = s.load(ctx, sess.ID)
	if err != nil {
		return Session{}, nil, GuidedStep{}, err
	}
	idx := indexOfDraft(sess.PhaseDrafts, phaseID)
	if idx < 0 {
		return Session{}, nil, GuidedStep{}, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + phaseID}
	}
	if err := applyPhaseField(&sess.PhaseDrafts[idx], field, content); err != nil {
		return Session{}, nil, GuidedStep{}, err
	}
	sess.CurrentPhaseID = nextIncompletePhaseID(sess.PhaseDrafts)
	sess = syncPhaseSection(sess)
	sess.UpdatedAt = s.now()
	violations := phaseViolations(sess.PhaseDrafts[idx])
	if err := s.store.Save(ctx, sess); err != nil {
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
