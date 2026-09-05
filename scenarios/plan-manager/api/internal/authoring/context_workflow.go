package authoring

import (
	"context"
	"strconv"
	"strings"

	planmodel "plan-manager/internal/planmodel"
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
			// apply the scope default here so an unset
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
		return true, nil
	})
	if err != nil {
		return Session{}, planmodel.RelevantContextItem{}, nil, GuidedStep{}, err
	}
	return sess, item, violations, stepForCurrentSessionState(sess), nil
}

// steeredSkillItem converts one resolver suggestion into a ready global skill
// item (bare-slug target; the renderer derives the read command).
func steeredSkillItem(suggestion SkillSuggestion) (planmodel.RelevantContextItem, bool) {
	slug := strings.TrimSpace(suggestion.Slug)
	if slug == "" {
		return planmodel.RelevantContextItem{}, false
	}
	return planmodel.RelevantContextItem{
		Kind:         planmodel.RelevantContextSkill,
		Scope:        planmodel.RelevantContextScopeGlobal,
		Label:        slug,
		Reason:       strings.TrimSpace(suggestion.Reason),
		Instruction:  "Load this internal skill before implementation unless it is clearly irrelevant.",
		Target:       slug,
		Required:     true,
		RepeatPolicy: planmodel.RelevantContextOncePerExecution,
		Source:       planmodel.RelevantContextSourceDiscovered,
		Status:       planmodel.RelevantContextStatusReady,
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
	return sess, violations, step, nil
}

// discoveryComplexityLevels is prompt-manager's discovery complexity
// vocabulary. Prompt-manager owns this contract (its API rejects anything
// else with a 400); TestDiscoveryComplexityContract pins the two in sync.
var discoveryComplexityLevels = []string{"minor", "moderate", "major", "architectural"}

// discoveryComplexitySynonyms maps the size vocabularies agents habitually
// reach for onto prompt-manager's levels, so a guessed "high" becomes a
// working call instead of a degraded skill pack.
var discoveryComplexitySynonyms = map[string]string{
	"low":    "minor",
	"small":  "minor",
	"medium": "moderate",
	"high":   "major",
	"large":  "major",
	"xhigh":  "architectural",
	"max":    "architectural",
}

// normalizeDiscoveryComplexity validates the optional complexity value before
// it is shelled out to prompt-manager: empty stays empty (prompt-manager
// defaults to moderate), canonical levels pass, common synonyms are mapped,
// and anything else is rejected naming the valid values — failing fast beats
// a degraded discovery.
func normalizeDiscoveryComplexity(complexity string) (string, error) {
	complexity = strings.ToLower(strings.TrimSpace(complexity))
	if complexity == "" {
		return "", nil
	}
	for _, level := range discoveryComplexityLevels {
		if complexity == level {
			return complexity, nil
		}
	}
	if mapped, ok := discoveryComplexitySynonyms[complexity]; ok {
		return mapped, nil
	}
	return "", ErrInvalidSession{Reason: "complexity " + strconv.Quote(complexity) + " is not a prompt-manager discovery level; use one of: " + strings.Join(discoveryComplexityLevels, ", ") + " (or omit the flag)"}
}

// skillPackRecoveryHint names the known-good manual fallback so a degraded
// discovery is recoverable without tribal knowledge.
func skillPackRecoveryHint(sessionID string) string {
	return "; fallback: run `prompt-manager discover \"<concept>\" --type skill --json` directly, then add each skill via `plan-manager author context-submit " + sessionID + " --kind skill --label \"<name>\" --command \"prompt-manager skill read <slug>\" --reason \"<why>\"`"
}

func (s *service) DiscoverSkillPack(ctx context.Context, sessionID, phaseID string, concepts []string, complexity string) (Session, SkillPackResult, []planmodel.RelevantContextItem, []planmodel.RelevantContextItem, []StructureViolation, GuidedStep, error) {
	// Skill discovery shells out and can take tens of seconds — run it against a
	// pre-lock read so prompt-manager never holds the session lock; only the
	// upsert happens under the lock.
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, SkillPackResult{}, nil, nil, nil, GuidedStep{}, err
	}
	complexity, err = normalizeDiscoveryComplexity(complexity)
	if err != nil {
		return Session{}, SkillPackResult{}, nil, nil, nil, GuidedStep{}, err
	}
	var result SkillPackResult
	if s.skills == nil {
		result = SkillPackResult{Degraded: true, DegradedReason: "prompt-manager skill discovery unavailable"}
	} else if result, err = s.skills.DiscoverSkillPack(ctx, sess.Title, concepts, complexity); err != nil {
		result = SkillPackResult{Degraded: true, DegradedReason: err.Error() + skillPackRecoveryHint(sessionID)}
	}
	for _, suggestion := range s.skillSteer.SuggestSkills(ctx, sessionBoundary(sess)) {
		if item, ok := steeredSkillItem(suggestion); ok {
			result.Items = append(result.Items, item)
		}
	}
	var added, kept []planmodel.RelevantContextItem
	var violations []StructureViolation
	sess, err = s.withSessionLock(ctx, sessionID, func(sess *Session) (bool, error) {
		// Resolve the phase once, before the item loop: an unknown phase ref must
		// fail the whole call rather than silently demote the pack to global scope.
		phaseIdx := -1
		if phaseID != "" {
			if phaseIdx = indexOfDraft(sess.PhaseDrafts, phaseID); phaseIdx < 0 {
				return false, ErrSectionNotFound{SessionID: sessionID, SectionKey: "phase:" + phaseID}
			}
		}
		changed := false
		for _, item := range result.Items {
			item = normalizeContextItem(item, phaseID)
			if phaseIdx >= 0 {
				item.Scope = planmodel.RelevantContextScopePhase
				item.PhaseID = sess.PhaseDrafts[phaseIdx].ID
			} else {
				item.Scope = planmodel.RelevantContextScopeGlobal
				item.PhaseID = ""
			}
			item.RepeatPolicy = defaultRepeatForScope(item.Scope, item.RepeatPolicy)
			item.Source = planmodel.RelevantContextSourceDiscovered
			item.Status = planmodel.RelevantContextStatusReady
			itemViolations := contextItemViolations(item)
			if len(itemViolations) > 0 {
				violations = append(violations, itemViolations...)
				continue
			}
			target := &sess.RelevantContext
			if phaseIdx >= 0 {
				target = &sess.PhaseDrafts[phaseIdx].RelevantContext
			}
			if pos := indexOfContextItemByKey(*target, item); pos >= 0 {
				kept = append(kept, (*target)[pos])
				continue
			}
			*target = append(*target, item)
			added = append(added, item)
			changed = true
		}
		if changed {
			if phaseIdx >= 0 {
				sess.CurrentPhaseID = nextIncompletePhaseID(sess.PhaseDrafts)
				*sess = syncPhaseSection(*sess)
			} else {
				*sess = syncContextSection(*sess)
			}
		}
		return changed, nil
	})
	if err != nil {
		return Session{}, SkillPackResult{}, nil, nil, nil, GuidedStep{}, err
	}
	return sess, result, added, kept, violations, stepForCurrentSessionState(sess), nil
}

// runAutofill runs one source against the session in place. It NEVER fabricates a
// fill: a nil seam or an error leaves the section untouched and returns
// Degraded=true with the honest reason.
