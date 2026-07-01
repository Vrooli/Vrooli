package authoring

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

var sessionSlugNonWord = regexp.MustCompile(`[^a-z0-9]+`)

func (s *service) uniqueSessionSlug(ctx context.Context, slug, title string) (string, error) {
	base := sessionSlugify(slug)
	if base == "" {
		base = sessionSlugify(title)
	}
	if base == "" {
		base = "session"
	}
	candidate := base
	for i := 2; ; i++ {
		_, ok, err := s.store.Get(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !ok {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
}

func sessionSlugify(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	v = sessionSlugNonWord.ReplaceAllString(v, "-")
	return strings.Trim(v, "-")
}

func (s *service) lockSession(sessionID string) func() {
	s.lockMu.Lock()
	mu := s.sessionLocks[sessionID]
	if mu == nil {
		mu = &sync.Mutex{}
		s.sessionLocks[sessionID] = mu
	}
	s.lockMu.Unlock()
	mu.Lock()
	return mu.Unlock
}

// prefillWorkPosture marks the Work Posture section as autofilled+reviewed (never
// agent-authored). The actual posture is derived from scenario maturity when the
// plan is persisted; the section here is an informational review marker so the
// wizard surfaces posture without asking the author to write the Greenfield block.
func prefillWorkPosture(sections []Section) {
	idx := indexOf(sections, SectionWorkPosture)
	if idx < 0 {
		return
	}
	sections[idx].Content = "Work posture is derived automatically from scenario maturity (default: greenfield). Do not author the Greenfield/Brownfield block — the renderer injects it."
	sections[idx].Filled = true
	sections[idx].Autofilled = true
	sections[idx].Mandatory = false
}

func (s *service) load(ctx context.Context, sessionID string) (Session, error) {
	sess, ok, err := s.store.Get(ctx, strings.TrimSpace(sessionID))
	if err != nil {
		return Session{}, err
	}
	if !ok {
		return Session{}, ErrSessionNotFound{ID: sessionID}
	}
	return sess, nil
}

func (s *service) now() string { return s.clock.Now().UTC().Format(sessionTimeFormat) }

// --- pure helpers (no I/O) ---

// stepForCurrentSessionState is the guided step a normal mutation returns. It
// delegates to nextGuidedStep so a mutation never reports final_review while
// global relevant context is unresolved or a phase is still incomplete — the
// premature-final-review friction. CurrentSectionKey alone is insufficient
// because it tracks only mandatory *sections* (the phases section reads "filled"
// as soon as one draft exists, even an incomplete one), and the global-context
// checkpoint is not a mandatory section at all.
func stepForCurrentSessionState(sess Session) GuidedStep {
	return nextGuidedStep(sess)
}

// nextGuidedStep selects the guided step for the session's true next required
// action, mirroring ContinueAuthoring's resolution order: finalized → first
// unfilled mandatory section → global relevant-context checkpoint → first
// incomplete phase → outstanding structure violation → final review. It is pure
// (no command-reference seam); that seam runs only at ValidateStructure/Finalize,
// so a clean nextGuidedStep is a "structurally ready" hint, never a guarantee.
func nextGuidedStep(sess Session) GuidedStep {
	return selectWorkItem(sess, sessionViolations(sess)).Step
}

func stepForNextPhaseState(sess Session, fallback PhaseDraft) GuidedStep {
	if id := nextIncompletePhaseID(sess.PhaseDrafts); id != "" {
		if phase, ok := findDraft(sess.PhaseDrafts, id); ok {
			return stepForPhase(sess, phase)
		}
	}
	return stepForPhase(sess, fallback)
}

// structureViolations is the structure-validation gate (PM-AUTHOR-002): every
// mandatory section must be non-empty, and the regression-anchor section must not
// be empty (it is a distinct violation even when not otherwise mandatory).
// referencesGateMessage is the single message for the references requirement.
// References is mandatory but satisfiable by a NO_CODE_REFS: reason, so its
// requirement is enforced by this gate (not the generic empty-mandatory message),
// which keeps the "unless NO_CODE_REFS" escape in the same sentence.
