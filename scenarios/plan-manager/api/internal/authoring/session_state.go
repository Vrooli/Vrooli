package authoring

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	planmodel "plan-manager/internal/planmodel"
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
	// Cap DERIVED handles at a typeable length (word-boundary truncation);
	// the collision suffix below may exceed it by its own length only.
	// Existing long slugs are untouched and keep resolving.
	base = planmodel.TruncateSlug(base, planmodel.MaxSlugLength)
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

// withSessionLock is the single mutation discipline for authoring sessions:
// resolve the session by id-or-slug, take its per-session lock, reload the
// authoritative state under the lock, run fn against it, and save exactly once
// when fn reports the session changed. Every authoring mutation goes through
// here so lock-reload-modify-save is structural — a mutation cannot forget the
// lock (the lost-write hole SubmitSection used to have). fn mutates sess in
// place; save=false means "session unchanged, do not persist" (e.g. a
// violation-rejected item). An fn error aborts without saving.
func (s *service) withSessionLock(ctx context.Context, sessionID string, fn func(sess *Session) (save bool, err error)) (Session, error) {
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return Session{}, err
	}
	unlock := s.lockSession(sess.ID)
	defer unlock()
	sess, err = s.load(ctx, sess.ID)
	if err != nil {
		return Session{}, err
	}
	save, err := fn(&sess)
	if err != nil {
		return Session{}, err
	}
	if save {
		sess.UpdatedAt = s.now()
		if err := s.store.Save(ctx, sess); err != nil {
			return Session{}, err
		}
	}
	return sess, nil
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
