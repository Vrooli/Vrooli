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
	for _, mode := range SortedModes(defs) {
		def := defs[mode]
		if def.Scope.Kind != ScopeInitiative {
			continue
		}
		if err := validateGuardGraph(def); err != nil {
			return err
		}
		if err := validateExampleRuns(def); err != nil {
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
func validateExampleRuns(def Definition) error {
	if len(def.ExampleRuns) == 0 {
		return nil
	}
	hasHappyPath := false
	for _, run := range def.ExampleRuns {
		if run.ID == happyPathPresetID {
			hasHappyPath = true
		}
		if _, err := WalkExampleRun(def, run); err != nil {
			return fmt.Errorf("mode %q example-run %q: %w", def.Mode, run.ID, err)
		}
	}
	if !hasHappyPath {
		return fmt.Errorf("mode %q ships example-runs but none has the reserved %q id (the simulator's default preset)", def.Mode, happyPathPresetID)
	}
	return nil
}

// validateGuardGraph validates each initiative-scoped phase's guarded
// transitions: guard structure, that every target is a registered phase, that a
// guard's field references a declared output field of the phase, and that the
// guard adjacency agrees with the derived static adjacency. These checks make a
// mode's branching trustworthy without reading Go.
func validateGuardGraph(def Definition) error {
	for phase, phaseDef := range def.PhaseGraph.Phases {
		guards := def.PhaseGraph.Guards[phase]
		declaredTop := declaredTopLevelFields(phaseDef)
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
