package operatingmode

import (
	"fmt"
	"strings"
)

// ValidateLoadedModes runs the full semantic validation over a data-loaded mode
// set: the same mode/phase/metric/profile invariants the runtime enforces, plus
// validation of the generic guard graph (the data-driven branching model that
// replaces the closed TransitionCondition vocabulary). It returns a typed,
// actionable error naming the offending mode/phase, or nil when every mode is
// valid.
func ValidateLoadedModes(defs map[Mode]Definition) error {
	if err := validateDefinitions(defs); err != nil {
		return err
	}
	if err := validateDelegations(defs); err != nil {
		return err
	}
	for _, mode := range SortedModes(defs) {
		def := defs[mode]
		if _, err := CompileInputContract(defs, def); err != nil {
			return err
		}
		if err := validateGuardGraph(defs, def); err != nil {
			return err
		}
		if err := validateExampleRuns(defs, def); err != nil {
			return err
		}
	}
	return nil
}

// validateExampleRuns replays every mode-owned example-run against the mode's
// real generic guard evaluator and requires the walked path to equal the
// fixture's declared expected_path. This is what makes an example-run a trusted
// test-before-use: a fixture that no longer matches the guards fails startup
// with an actionable error. A phase mode that ships any example-runs must own
// one with the reserved happy-path id (the simulator's default preset); a mode
// with none relies on the synthesized happy-path walk.
func validateExampleRuns(defs map[Mode]Definition, def Definition) error {
	if len(def.ExampleRuns) == 0 {
		return nil
	}
	hasHappyPath := false
	for _, run := range def.ExampleRuns {
		if run.ID == happyPathPresetID {
			hasHappyPath = true
		}
		if _, err := WalkExampleRun(defs, def, run); err != nil {
			return fmt.Errorf("mode %q example-run %q: %w", def.Mode, run.ID, err)
		}
	}
	if !hasHappyPath {
		return fmt.Errorf("mode %q ships example-runs but none has the reserved %q id (the simulator's default preset)", def.Mode, happyPathPresetID)
	}
	return nil
}

// validateGuardGraph validates each phase's guarded transitions: guard
// structure, that every target is a registered phase, that a guard's field
// references a declared output field of the phase (or, for a delegated phase,
// a field the sub-mode's terminal outcome can carry), and that the guard
// adjacency agrees with the derived static adjacency. These checks make a
// mode's branching trustworthy without reading Go.
func validateGuardGraph(defs map[Mode]Definition, def Definition) error {
	for phase, phaseDef := range def.PhaseGraph.Phases {
		if err := validateTransitionClassification(def, phase, phaseDef); err != nil {
			return err
		}
		guards := def.PhaseGraph.Guards[phase]
		var declaredTop map[string]struct{}
		if phaseDef.Delegated() {
			// A delegated phase's transitions route on the sub-mode's terminal
			// outcome — the union of the sub-phases' declared output fields and
			// edge-classified routing fields. Existence of the sub-mode is
			// validated by validateDelegations; a miss here means delegation
			// validation already failed with the actionable error.
			sub, ok := defs[phaseDef.ExecutedBy]
			if !ok {
				return fmt.Errorf("mode %q phase %q delegates to unknown sub-mode %q", def.Mode, phase, phaseDef.ExecutedBy)
			}
			declaredTop = delegatedOutcomeFields(sub)
		} else {
			declaredTop = declaredTopLevelFields(phaseDef)
			// The classification-derived routing field is guard-visible even though
			// the phase does not declare it as an output: deriving it at the edge is
			// the entire point of classification-on-transition.
			if c := phaseDef.TransitionClassification; c != nil {
				declaredTop[topFieldSegment(c.Field)] = struct{}{}
			}
		}
		for i, gt := range guards {
			if err := validateGuard(gt.When); err != nil {
				return fmt.Errorf("mode %q phase %q guard[%d]: %w", def.Mode, phase, i, err)
			}
			if err := validateGuardFields(gt.When, declaredTop); err != nil {
				return fmt.Errorf("mode %q phase %q guard[%d]: %w", def.Mode, phase, i, err)
			}
			for _, to := range gt.To {
				if _, ok := def.PhaseGraph.Phases[to]; !ok {
					return fmt.Errorf("mode %q phase %q guard[%d] routes to unregistered phase %q", def.Mode, phase, i, to)
				}
			}
		}
	}
	return nil
}

// validateTransitionClassification checks a phase's transition-owned
// classification contract semantically (the structural rules — enum coverage,
// routes agreement, one contract per phase — were enforced by the loader
// expansion). Two invariants:
//
//   - `from`, when declared, must name a declared output field of the phase —
//     a classification cannot derive from output the phase never promises to
//     emit (the input-side twin of the guard-field check).
//   - the phase's declared_output must not declare the classification field as
//     REQUIRED: a required field is always emitted directly, so classification
//     would be permanently short-circuited (not_required) — a dead
//     declaration. Use plain guarded transitions instead. Declaring the field
//     as optional is allowed: an emitted value short-circuits, an absent one
//     is derived at the edge.
func validateTransitionClassification(def Definition, phase Phase, phaseDef PhaseDefinition) error {
	c := phaseDef.TransitionClassification
	if c == nil {
		return nil
	}
	declaredTop := declaredTopLevelFields(phaseDef)
	if c.From != "" {
		if _, ok := declaredTop[topFieldSegment(c.From)]; !ok {
			return fmt.Errorf("mode %q phase %q classify.from %q is not a declared output field of the phase", def.Mode, phase, c.From)
		}
	}
	if declaredRequiredFieldPaths(phaseDef)[c.Field] {
		return fmt.Errorf("mode %q phase %q classify.field %q is declared REQUIRED in declared_output: classification would always short-circuit; route on the emitted field with plain guarded transitions instead", def.Mode, phase, c.Field)
	}
	return nil
}

// declaredRequiredFieldPaths returns the set of dotted paths the phase's
// declared output marks required.
func declaredRequiredFieldPaths(phaseDef PhaseDefinition) map[string]bool {
	set := map[string]bool{}
	for _, path := range requiredFieldNames(phaseDef.DeclaredOutput) {
		set[path] = true
	}
	return set
}

// declaredTopLevelFields returns the set of top-level field-name segments the
// phase declares in its output schema. A guard field's first path segment must
// be one of these, so a guard can never branch on output the phase does not
// promise to emit.
func declaredTopLevelFields(phaseDef PhaseDefinition) map[string]struct{} {
	set := map[string]struct{}{}
	if phaseDef.DeclaredOutput == nil {
		return set
	}
	for _, field := range phaseDef.DeclaredOutput.Fields {
		set[topFieldSegment(field.Name)] = struct{}{}
	}
	return set
}

// validateGuardFields walks a guard and requires every leaf field's top-level
// segment to be a declared output field. always/composite guards recurse; the
// composite forms carry no field of their own.
func validateGuardFields(g Guard, declaredTop map[string]struct{}) error {
	switch {
	case len(g.All) > 0:
		for _, sub := range g.All {
			if err := validateGuardFields(sub, declaredTop); err != nil {
				return err
			}
		}
		return nil
	case len(g.Any) > 0:
		for _, sub := range g.Any {
			if err := validateGuardFields(sub, declaredTop); err != nil {
				return err
			}
		}
		return nil
	case g.Not != nil:
		return validateGuardFields(*g.Not, declaredTop)
	}
	if strings.TrimSpace(g.Field) == "" {
		return nil
	}
	top := topFieldSegment(g.Field)
	if _, ok := declaredTop[top]; !ok {
		return fmt.Errorf("guard field %q is not a declared output field of the phase", g.Field)
	}
	return nil
}

func topFieldSegment(field string) string {
	if idx := strings.IndexByte(field, '.'); idx >= 0 {
		return field[:idx]
	}
	return field
}
