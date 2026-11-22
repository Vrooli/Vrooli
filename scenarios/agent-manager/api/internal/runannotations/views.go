package runannotations

import (
	"context"
	"strings"
)

// Read models: the views built from in-progress work and from stored rows.

// runLabels counts the run-scope labels of the pack: the run view's denominator.
// Message-scope labels are answered per message and reported under Messages.
func (s *Service) runLabels() []Label {
	var out []Label
	for _, label := range s.pack.Pack.Labels {
		if label.Scope != ScopeMessage {
			out = append(out, label)
		}
	}
	return out
}

func (s *Service) viewFor(work *runWork) RunView {
	view := RunView{
		RunID: work.runID, Pack: s.pack.Pack.Name, Version: s.pack.Pack.Version,
		PackHash: s.pack.PackHash, State: "evaluated", Skipped: work.skipped,
	}
	for _, label := range s.runLabels() {
		observation, ok := work.observations[label.ID]
		if !ok {
			if !skippedFor(work.skipped, label.ID) {
				view.Missing = append(view.Missing, label.ID)
			}
			continue
		}
		view.Labels = append(view.Labels, s.annotationView(observation))
		addCount(&view.Counts, observation.State)
	}
	view.Coverage = RunCoverage{Observed: view.Counts.Observed + view.Counts.Compatible, Labels: len(s.runLabels())}
	if len(view.Labels) == 0 {
		view.Notes = append(view.Notes, "no annotations recorded; unknown is not zero friction")
	}
	for _, skipped := range work.skipped {
		if skipped.LabelID == "*" && strings.HasPrefix(skipped.Reason, "persist_failed") {
			view.Notes = append(view.Notes, skipped.Reason)
		}
	}
	return view
}

// Run reads the latest annotations for one run without evaluating.
func (s *Service) Run(ctx context.Context, runID string) (RunView, error) {
	return s.readRun(ctx, runID, s.store, "read")
}

// RunTrial reads the answers a pinned bake-off stored for one run. They are kept
// apart from production answers and are never part of a production read.
func (s *Service) RunTrial(ctx context.Context, runID string) (RunView, error) {
	return s.readRun(ctx, runID, s.store.Trials(), "read_trial")
}

func (s *Service) readRun(ctx context.Context, runID string, store *Store, state string) (RunView, error) {
	if s.features != nil {
		if _, err := s.features.Facts(ctx, ScopeRun, runID); err != nil {
			return RunView{}, err
		}
	}
	annotations, err := store.LatestForScope(ctx, ScopeRun, runID)
	if err != nil {
		return RunView{}, err
	}
	view := RunView{
		RunID: runID, Pack: s.pack.Pack.Name, Version: s.pack.Pack.Version,
		PackHash: s.pack.PackHash, State: state,
	}
	for _, annotation := range annotations {
		view.Labels = append(view.Labels, s.annotationView(annotation))
		addCount(&view.Counts, annotation.State)
	}
	for _, label := range s.runLabels() {
		if !hasAnnotation(annotations, label.ID) {
			view.Missing = append(view.Missing, label.ID)
		}
	}
	view.Coverage = RunCoverage{Observed: view.Counts.Observed + view.Counts.Compatible, Labels: len(s.runLabels())}
	if rows, err := store.LatestMessagesForRun(ctx, runID); err == nil {
		view.Messages = s.messageViewsFromRows(rows)
	} else {
		return RunView{}, err
	}
	if len(view.Labels) == 0 && len(view.Messages) == 0 {
		view.Notes = append(view.Notes, "no annotations recorded; unknown is not zero friction")
	}
	return view, nil
}

// annotationView is the contract view of one stored or fresh observation. Its
// gate is the proof of the model that produced it, never the label's best: an
// observation from an unmeasured model is unproven even when another model of
// the same label is proven. The calibrated score applies that model's own
// temperature; the stored vector is untouched.
func (s *Service) annotationView(annotation Annotation) AnnotationView {
	view := AnnotationView{
		ID: annotation.LabelID, State: annotation.State, Method: coarseMethod(annotation.Method),
		EvaluatorMethod: annotation.Method, Value: annotation.Value, Probability: annotation.Probability,
		Confidence: annotation.Confidence, Model: annotation.Model, ResourceRole: annotation.ModelVersion,
		RequestedRole: annotation.RequestedRole, Provider: annotation.Provider,
		DefinitionHash: annotation.DefinitionHash, ObservedAt: annotation.ObservedAt,
		EvidenceEventIDs: annotation.EvidenceEventIDs, Reason: annotation.Reason,
	}
	if !annotation.OccurredAt.IsZero() {
		at := annotation.OccurredAt
		view.OccurredAt = &at
	}
	compiled, ok := s.pack.Label(annotation.LabelID)
	if !ok {
		return view
	}
	status := s.pack.StatusFor(compiled, annotation)
	view.Gate, view.GateState, view.Proven = status, status.State, status.State == GateProven
	view.Friction = compiled.label.Friction
	temperature, measured := calibrationOf(compiled.label, annotation)
	if sample, usable := sampleFor(compiled.label, annotation); usable {
		score := sample.Score(temperature)
		view.CalibratedProbability = &score
		if temperature != 1 {
			view.Temperature = temperature
		}
		if _, _, directional := frictionScoreBound(compiled.label); directional && !sample.Fixed {
			view.FrictionProbability = &score
		}
		if compiled.label.Friction.Enabled {
			positive := isFrictionPositive(compiled.label, annotation, temperature, measured)
			view.FrictionPositive = &positive
		}
	}
	return view
}

// annotationFromView rebuilds the stored shape from a view, so an evaluate
// response is counted by the same code that counts stored rows.
func (s *Service) annotationFromView(view AnnotationView, scope Scope) Annotation {
	return Annotation{
		ScopeKind: scope, LabelID: view.ID, DefinitionHash: view.DefinitionHash, State: view.State,
		Method: view.EvaluatorMethod, Model: view.Model, ModelVersion: view.ResourceRole,
		Value: view.Value, Probability: view.Probability, Confidence: view.Confidence,
	}
}

func coarseMethod(method string) string {
	normalized := strings.ToLower(method)
	switch {
	case normalized == "deterministic", normalized == "signal", normalized == "composite":
		return normalized
	case strings.Contains(normalized, "decision"), strings.Contains(normalized, "llm"):
		return "decision"
	default:
		return normalized
	}
}
