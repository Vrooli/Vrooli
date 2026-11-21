package runannotations

import (
	"context"
	"strings"
	"time"
)

// runWork is one unit of evaluation in progress: a run, or one message of a run.
// Every label of the unit's scope is evaluated against the same bounded state.
type runWork struct {
	scopeKind Scope
	// scopeID is the run id for a run and the message event id for a message;
	// runID is always the run. occurredAt is when a message was written.
	scopeID    string
	runID      string
	occurredAt time.Time
	pass       *passState
	// parent is the run's own work for a message work, so a message's answers
	// can be reported under the run they belong to.
	parent *runWork
	// children are a run's message units, evaluated before the run's own
	// composites so those can read their observations.
	children []*runWork
	// force re-evaluates cached answers.
	force bool

	facts    RunFacts
	features RunFeatures
	digest   string
	// build renders this unit's state at a character limit, keeping what the
	// labels being asked declare as their evidence (see DigestPriority) longest.
	// A model with a smaller window, or one that overflows, is sent a
	// re-rendered state, never a truncated one.
	build func(limit int, priority []string) (string, error)
	sized map[sizedState]string
	// evidenceHash identifies the bounded state every label on this unit reads.
	// A cached answer is reused only when it was computed from the same
	// evidence, so new messages, tool failures or signals invalidate it.
	evidenceHash string
	// digestShrunk records that a model could not read the full digest and
	// this work already retried with a smaller one. The second overflow is
	// reported, never retried forever.
	digestShrunk bool
	// sentChars is the size of the state most recently rendered for a model, so
	// an overflow shrinks from what the model refused.
	sentChars  int
	overflowed bool
	// retryable marks a transport failure that has not been retried yet.
	// transportRetried bounds that retry to one per pass.
	retryable        bool
	transportRetried bool
	// subjectEvidence is the event id a message unit's answers cite.
	subjectEvidence []string
	activation      map[string]any
	observations    map[string]Annotation
	evidence        map[string][]string
	order           []string
	toSave          []Annotation
	needs           []string
	pending         []string
	skipped         []SkippedLabel
}

// sizedState keys one rendering of a unit's state.
type sizedState struct {
	limit    int
	priority string
}

// stateFor is the state sent to a model that can take at most limit characters
// and is asked the labels whose digest priority is given. A rendering is built
// once per size and priority and reused.
func (w *runWork) stateFor(limit int, priority []string) string {
	if w.digestShrunk || limit <= 0 || w.build == nil {
		return w.digest
	}
	key := sizedState{limit: limit, priority: strings.Join(priority, ",")}
	if cached, ok := w.sized[key]; ok {
		w.sentChars = len(cached)
		return cached
	}
	body, err := w.build(limit, priority)
	if err != nil || body == "" {
		return w.digest
	}
	if w.sized == nil {
		w.sized = map[sizedState]string{}
	}
	w.sized[key] = body
	w.sentChars = len(body)
	return body
}

func (s *Service) record(work *runWork, observation Annotation) {
	observation.ScopeKind = work.scopeKind
	observation.ScopeID = work.scopeID
	observation.RunID = work.runID
	observation.OccurredAt = work.occurredAt
	if observation.ObservedAt.IsZero() {
		observation.ObservedAt = s.now().UTC()
	}
	if len(observation.EvidenceEventIDs) == 0 {
		observation.EvidenceEventIDs = work.evidence[observation.LabelID]
	}
	if _, exists := work.observations[observation.LabelID]; !exists {
		work.order = append(work.order, observation.LabelID)
	}
	work.observations[observation.LabelID] = observation
	work.toSave = append(work.toSave, observation)
}

func (s *Service) save(ctx context.Context, work *runWork) {
	if len(work.toSave) == 0 {
		return
	}
	// Every answer in this work read the same bounded state, so it carries the
	// same fingerprint. Without it the cache could not tell a fresh answer from
	// one computed before the run appended more evidence.
	for index := range work.toSave {
		if work.toSave[index].EvidenceHash == "" {
			work.toSave[index].EvidenceHash = work.evidenceHash
		}
	}
	// Save errors are surfaced by the view's note; the run is still readable
	// from what the evaluators produced. The caller sees the failure in Notes.
	if err := work.pass.store.Save(ctx, work.toSave); err != nil {
		work.skipped = append(work.skipped, SkippedLabel{LabelID: "*", Reason: "persist_failed: " + err.Error()})
	}
}

// saveProvenance records whose words a run carried. It is a by-product of
// annotating (the facts are already read) and belongs to production passes only.
func (s *Service) saveProvenance(ctx context.Context, work *runWork) {
	if work.pass.pinned() || work.scopeKind != ScopeRun {
		return
	}
	row := ProvenanceRow{
		RunID: work.runID, WorkloadKind: work.facts.Workload.Kind,
		OperatorMessages: len(work.facts.HumanMessages) + work.facts.BriefMessages,
		HumanMessages:    len(work.facts.HumanMessages), BriefMessages: work.facts.BriefMessages,
		UpdatedAt: s.now().UTC(), PolicyHash: s.pack.Pack.Runtime.Provenance.Hash(),
	}
	if count := len(work.facts.HumanMessages); count > 0 {
		row.LastHumanAt = work.facts.HumanMessages[count-1].At
	}
	if err := s.store.SaveProvenance(ctx, row); err != nil {
		work.skipped = append(work.skipped, SkippedLabel{LabelID: "*", Reason: "persist_failed: " + err.Error()})
	}
}

func (s *Service) observation(label Label, hash string, state State, reason string) Annotation {
	return Annotation{
		ScopeKind: label.Scope, ScopeID: "", LabelID: label.ID, DefinitionHash: hash,
		Method: string(label.Method), State: state, Reason: reason,
	}
}

// markLabelUnavailable records an unavailable state for a label on every work
// that needs it.
func (s *Service) markLabelUnavailable(works []*runWork, labelID, reason string) {
	for _, work := range works {
		if contains(work.needs, labelID) {
			s.markLabelUnavailableForWork(work, labelID, reason)
		}
	}
}

func (s *Service) markLabelUnavailableForWork(work *runWork, labelID, reason string) {
	compiled, ok := s.pack.Label(labelID)
	if !ok {
		return
	}
	observation := s.observation(compiled.label, compiled.hash, StateUnavailable, reason)
	observation.ObservedAt = s.now().UTC()
	s.record(work, observation)
	work.needs = remove(work.needs, labelID)
}

func (s *Service) markNeedsUnavailable(works []*runWork, reason string, filter ...func(string) bool) {
	allow := func(string) bool { return true }
	if len(filter) > 0 && filter[0] != nil {
		allow = filter[0]
	}
	for _, work := range works {
		for _, labelID := range append([]string(nil), work.needs...) {
			if allow(labelID) {
				s.markLabelUnavailableForWork(work, labelID, reason)
			}
		}
	}
}

func (s *Service) labelIndex(labelID string) int {
	for i, id := range s.pack.Order() {
		if id == labelID {
			return i
		}
	}
	return len(s.pack.Order())
}
