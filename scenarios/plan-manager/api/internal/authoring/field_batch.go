package authoring

import (
	"context"
	"fmt"
	"strings"

	planmodel "plan-manager/internal/planmodel"
)

// FieldWrite is one item in a batch submission: exactly one of SectionKey
// (whole-section write) or PhaseRef+PhaseField (one phase-native field).
type FieldWrite struct {
	SectionKey SectionKey
	// PhaseRef is a phase id or authored order number.
	PhaseRef   string
	PhaseField PhaseField
	Content    string
}

// IsSection reports whether the write targets a session section (vs. a phase
// field). Section writes take precedence when both are (incorrectly) set.
func (w FieldWrite) IsSection() bool { return strings.TrimSpace(string(w.SectionKey)) != "" }

// FieldWriteResult is the per-item outcome of a batch submission — each line
// names exactly what was parsed (or why it was not applied), so a partial
// batch reads as "6 of 7 landed, validation rejected because …".
type FieldWriteResult struct {
	Index    int
	Accepted bool
	Summary  string
	// Violations attributable to THIS item: content violations on an accepted
	// section write, or the quality violation that rejected a phase field.
	Violations []StructureViolation
}

// SubmitFields applies a batch of section/phase-field writes under ONE session
// lock with ONE save: per-item independent apply, never all-or-nothing. An
// unresolvable or unparsable item is rejected (recorded, batch continues); a
// malformed request (no items) is a call error. The returned GuidedStep
// carries the full-disclosure checklist for the resulting session state.
func (s *service) SubmitFields(ctx context.Context, sessionID string, writes []FieldWrite) (Session, []FieldWriteResult, GuidedStep, error) {
	if len(writes) == 0 {
		return Session{}, nil, GuidedStep{}, ErrInvalidSession{Reason: "at least one field write is required"}
	}
	var results []FieldWriteResult
	sess, err := s.withSessionLock(ctx, sessionID, func(sess *Session) (bool, error) {
		results = make([]FieldWriteResult, 0, len(writes))
		anyApplied := false
		for i, write := range writes {
			result, applyErr := s.applyFieldWrite(ctx, sess, write)
			result.Index = i
			if applyErr != nil {
				result = FieldWriteResult{
					Index:      i,
					Accepted:   false,
					Summary:    applyErr.Error(),
					Violations: []StructureViolation{{SectionKey: fieldWriteSectionKey(write), Message: applyErr.Error()}},
				}
			}
			if result.Accepted {
				anyApplied = true
			}
			results = append(results, result)
		}
		if anyApplied {
			sess.CurrentPhaseID = nextIncompletePhaseID(sess.PhaseDrafts)
			*sess = syncPhaseSection(*sess)
			sess.CurrentSectionKey = firstUnfilledMandatory(sess.Sections)
		}
		return anyApplied, nil
	})
	if err != nil {
		return Session{}, nil, GuidedStep{}, err
	}
	return sess, results, stepForCurrentSessionState(sess), nil
}

// applyFieldWrite is the ONE apply path shared by SubmitFields and the
// single-item wrappers (SubmitSection via applySection, SubmitPhaseField).
// It returns an error for an unresolvable/unparsable target (the batch turns
// it into a per-item rejection; a single-item wrapper propagates it) and a
// non-accepted result for a quality rejection (content not applied, violation
// recorded, session unchanged).
func (s *service) applyFieldWrite(ctx context.Context, sess *Session, write FieldWrite) (FieldWriteResult, error) {
	if write.IsSection() {
		idx := indexOf(sess.Sections, write.SectionKey)
		if idx < 0 {
			return FieldWriteResult{}, ErrSectionNotFound{SessionID: sess.ID, SectionKey: string(write.SectionKey)}
		}
		content := write.Content
		var rejected []StructureViolation
		if write.SectionKey == SectionDefinitions {
			content, rejected = acceptedDefinitionLines(write.Content)
		}
		violations := append(rejected, s.applySection(ctx, sess, idx, content)...)
		return FieldWriteResult{
			Accepted:   true,
			Summary:    SectionSummary(sess.Sections[idx]),
			Violations: violations,
		}, nil
	}
	idx := indexOfDraft(sess.PhaseDrafts, write.PhaseRef)
	if idx < 0 {
		return FieldWriteResult{}, ErrSectionNotFound{SessionID: sess.ID, SectionKey: "phase:" + write.PhaseRef}
	}
	scratch := clonePhaseDraft(sess.PhaseDrafts[idx])
	if err := applyPhaseField(&scratch, write.PhaseField, write.Content); err != nil {
		return FieldWriteResult{}, err
	}
	if violation := introducedPhaseViolation(scratch, write.PhaseField); violation != nil {
		return FieldWriteResult{
			Accepted:   false,
			Summary:    fmt.Sprintf("rejected phase %d field %q: %s", scratch.Order, write.PhaseField, violation.Message),
			Violations: []StructureViolation{*violation},
		}, nil
	}
	sess.PhaseDrafts[idx] = scratch
	return FieldWriteResult{
		Accepted: true,
		Summary:  fmt.Sprintf("phase %d %s: %s", scratch.Order, write.PhaseField, PhaseFieldSummary(write.PhaseField, scratch)),
	}, nil
}

// acceptedDefinitionLines drops only malformed definition lines, allowing the
// remaining batch content to land. The returned violations name each rejected
// source line so the CLI can correct it without resubmitting valid terms.
func acceptedDefinitionLines(content string) (string, []StructureViolation) {
	accepted := make([]string, 0)
	violations := make([]StructureViolation, 0)
	for lineNumber, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "-"))
		if line == "" {
			continue
		}
		term, meaning, found := cutDefinitionSeparator(line)
		if !found || term == "" || meaning == "" {
			violations = append(violations, StructureViolation{SectionKey: SectionDefinitions, Message: fmt.Sprintf("definition line %d %q rejected: use 'Term — meaning' or 'Term: meaning'", lineNumber+1, line)})
			continue
		}
		accepted = append(accepted, line)
	}
	return strings.Join(accepted, "\n"), violations
}

// introducedPhaseViolation is the field-level quality gate the apply path
// enforces at write time: a submission that would make acceptance identical to
// validation is rejected rather than silently stored (validation is the
// checking method; acceptance is the outcome gate). Only the two involved
// fields can introduce it.
func introducedPhaseViolation(phase PhaseDraft, field PhaseField) *StructureViolation {
	if field != PhaseFieldAcceptance && field != PhaseFieldValidation {
		return nil
	}
	if a, v := normalizeForCompare(phase.Acceptance), normalizeForCompare(phase.Validation); a != "" && a == v {
		return &StructureViolation{
			SectionKey: SectionPhases,
			Message:    fmt.Sprintf("phase %d acceptance must not be identical to its validation: acceptance is the outcome gate, validation is the checking method", phase.Order),
		}
	}
	return nil
}

// clonePhaseDraft copies a phase draft deeply enough that a rejected apply
// leaves the session's draft untouched (applyPhaseField appends to
// RelevantContext and replaces the other slices).
func clonePhaseDraft(phase PhaseDraft) PhaseDraft {
	phase.RelevantContext = append([]planmodel.RelevantContextItem(nil), phase.RelevantContext...)
	return phase
}

func fieldWriteSectionKey(write FieldWrite) SectionKey {
	if write.IsSection() {
		return write.SectionKey
	}
	return SectionPhases
}
