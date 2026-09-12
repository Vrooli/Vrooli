package compile

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"flow-verifier/internal/flows/kinds/temporal/contract"
	"flow-verifier/internal/flows/kinds/temporal/model"
	"flow-verifier/internal/verification/quint"
)

var (
	quintIdentifierPattern       = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
	typeScriptStatusAccessorExpr = regexp.MustCompile(`^state\.[A-Za-z_][A-Za-z0-9_]*$`)
)

func Compile(raw contract.Contract) (model.Flow, error) {
	var errs []string
	if raw.SchemaVersion != model.SchemaVersion {
		errs = append(errs, fmt.Sprintf("schemaVersion must be %d", model.SchemaVersion))
	}
	requireString(&errs, "flowId", raw.FlowID)
	requireString(&errs, "domain", raw.Domain)
	requireString(&errs, "description", raw.Description)
	requireString(&errs, "model.module", raw.Model.Module)
	requireString(&errs, "model.seed", raw.Model.Seed)
	if raw.Model.MaxSteps < 1 {
		errs = append(errs, "model.maxSteps must be a positive integer")
	}
	if raw.Model.TraceCount < 1 {
		errs = append(errs, "model.traceCount must be a positive integer")
	}
	if len(raw.Model.Verify.Invariants) == 0 {
		errs = append(errs, "model.verify.invariants must declare at least one invariant")
	}

	indexes := buildIndexes(&errs, raw)
	validateQuintNames(&errs, quintNameMaps(indexes)...)
	initial, initialCount := initialState(raw.States)
	if initialCount != 1 {
		errs = append(errs, fmt.Sprintf("states must declare exactly one initial state, got %d", initialCount))
	}
	for _, invariant := range raw.Model.Verify.Invariants {
		if _, ok := indexes.InvariantByQuint[invariant]; !ok {
			errs = append(errs, fmt.Sprintf("model.verify.invariants references unknown invariant %s", invariant))
		}
	}
	if len(raw.Transitions) == 0 {
		errs = append(errs, "transitions must not be empty")
	}
	if raw.TransitionDefaults.Invalid == nil {
		errs = append(errs, "transitionDefaults.invalid is required")
	}

	var matrix model.TransitionMatrix
	if len(errs) == 0 {
		var matrixErrs []string
		matrix, matrixErrs = model.BuildTransitionMatrix(raw, indexes.StateByID, indexes.EventByID)
		errs = append(errs, matrixErrs...)
	}
	validateRuntime(&errs, raw)
	validateReplayShape(&errs, raw)
	if len(errs) == 0 {
		validateTraces(&errs, raw, matrix, indexes.StateByID, indexes.EventByID)
	}
	if len(errs) > 0 {
		sort.Strings(errs)
		return model.Flow{}, fmt.Errorf("  - %s", strings.Join(errs, "\n  - "))
	}
	return model.FromRaw(raw, initial, matrix, indexes), nil
}

func buildIndexes(errs *[]string, raw contract.Contract) model.Indexes {
	indexes := model.Indexes{
		StateByID:        map[string]contract.State{},
		StateQuintByID:   map[string]string{},
		EventByID:        map[string]contract.Event{},
		EventQuintByID:   map[string]string{},
		InvariantByQuint: map[string]contract.Invariant{},
	}
	stateQuints := map[string]bool{}
	if len(raw.States) == 0 {
		*errs = append(*errs, "states must not be empty")
	}
	for i, state := range raw.States {
		requireString(errs, fmt.Sprintf("states[%d].id", i), state.ID)
		requireString(errs, fmt.Sprintf("states[%d].quint", i), state.Quint)
		if _, ok := indexes.StateByID[state.ID]; ok {
			*errs = append(*errs, "duplicate states id "+state.ID)
		}
		if stateQuints[state.Quint] {
			*errs = append(*errs, "duplicate Quint tag "+state.Quint)
		}
		indexes.StateByID[state.ID] = state
		indexes.StateQuintByID[state.ID] = state.Quint
		stateQuints[state.Quint] = true
	}

	eventQuints := map[string]bool{}
	if len(raw.Events) == 0 {
		*errs = append(*errs, "events must not be empty")
	}
	for i, event := range raw.Events {
		requireString(errs, fmt.Sprintf("events[%d].id", i), event.ID)
		requireString(errs, fmt.Sprintf("events[%d].quint", i), event.Quint)
		if _, ok := indexes.EventByID[event.ID]; ok {
			*errs = append(*errs, "duplicate events id "+event.ID)
		}
		if eventQuints[event.Quint] {
			*errs = append(*errs, "duplicate Quint tag "+event.Quint)
		}
		indexes.EventByID[event.ID] = event
		indexes.EventQuintByID[event.ID] = event.Quint
		eventQuints[event.Quint] = true
	}

	if len(raw.Invariants) == 0 {
		*errs = append(*errs, "invariants must not be empty")
	}
	for i, invariant := range raw.Invariants {
		requireString(errs, fmt.Sprintf("invariants[%d].id", i), invariant.ID)
		requireString(errs, fmt.Sprintf("invariants[%d].quint", i), invariant.Quint)
		requireString(errs, fmt.Sprintf("invariants[%d].description", i), invariant.Description)
		if _, ok := indexes.InvariantByQuint[invariant.Quint]; ok {
			*errs = append(*errs, "duplicate Quint tag "+invariant.Quint)
		}
		indexes.InvariantByQuint[invariant.Quint] = invariant
	}
	return indexes
}

func quintNameMaps(indexes model.Indexes) []map[string]bool {
	stateQuints := map[string]bool{}
	for _, quint := range indexes.StateQuintByID {
		stateQuints[quint] = true
	}
	eventQuints := map[string]bool{}
	for _, quint := range indexes.EventQuintByID {
		eventQuints[quint] = true
	}
	invariantQuints := map[string]bool{}
	for quint := range indexes.InvariantByQuint {
		invariantQuints[quint] = true
	}
	return []map[string]bool{stateQuints, eventQuints, invariantQuints}
}

func initialState(states []contract.State) (contract.State, int) {
	initials := 0
	var initial contract.State
	for _, state := range states {
		if state.Initial {
			initials++
			initial = state
		}
	}
	return initial, initials
}

func validateTraces(errs *[]string, raw contract.Contract, matrix model.TransitionMatrix, states map[string]contract.State, events map[string]contract.Event) {
	if len(raw.Traces) == 0 {
		*errs = append(*errs, "traces must not be empty")
	}
	for i, trace := range raw.Traces {
		if trace.Name == "" {
			*errs = append(*errs, fmt.Sprintf("traces[%d].name is required", i))
		}
		if _, ok := states[trace.Initial]; !ok {
			*errs = append(*errs, fmt.Sprintf("traces[%d].initial references unknown state %s", i, trace.Initial))
		}
		current := trace.Initial
		for j, step := range trace.Steps {
			if _, ok := events[step.Event]; !ok {
				*errs = append(*errs, fmt.Sprintf("traces[%d].steps[%d].event references unknown event %s", i, j, step.Event))
			}
			if _, ok := states[step.Want]; !ok {
				*errs = append(*errs, fmt.Sprintf("traces[%d].steps[%d].want references unknown state %s", i, j, step.Want))
			}
			if _, ok := states[current]; !ok {
				continue
			}
			if _, ok := events[step.Event]; !ok {
				continue
			}
			transition, ok := matrix.Lookup(current, step.Event)
			if !ok {
				*errs = append(*errs, fmt.Sprintf("traces[%d:%s].steps[%d] has no expanded transition for %s/%s", i, trace.Name, j, current, step.Event))
				continue
			}
			if step.Want != transition.To || step.WantError != transition.WantError {
				*errs = append(*errs, fmt.Sprintf(
					"traces[%d:%s].steps[%d] from %s with %s declares want=%s wantError=%v, expanded transition wants %s wantError=%v",
					i,
					trace.Name,
					j,
					current,
					step.Event,
					step.Want,
					step.WantError,
					transition.To,
					transition.WantError,
				))
			}
			current = transition.To
		}
	}
}

func validateRuntime(errs *[]string, raw contract.Contract) {
	if raw.Runtime.Go != nil && raw.Runtime.TypeScript != nil {
		*errs = append(*errs, "runtime must declare exactly one of go or typescript")
		return
	}
	switch {
	case raw.Runtime.Go != nil:
		requireString(errs, "runtime.go.package", raw.Runtime.Go.Package)
		requireString(errs, "runtime.go.statusType", raw.Runtime.Go.StatusType)
		requireString(errs, "runtime.go.eventType", raw.Runtime.Go.EventType)
		requireString(errs, "runtime.go.constantPrefix", raw.Runtime.Go.ConstantPrefix)
	case raw.Runtime.TypeScript != nil:
		requireString(errs, "runtime.typescript.statusType", raw.Runtime.TypeScript.StatusType)
		requireString(errs, "runtime.typescript.eventType", raw.Runtime.TypeScript.EventType)
		requireString(errs, "runtime.typescript.statusesConst", raw.Runtime.TypeScript.StatusesConst)
		requireString(errs, "runtime.typescript.eventsConst", raw.Runtime.TypeScript.EventsConst)
		requireString(errs, "runtime.typescript.formalExpectationConst", raw.Runtime.TypeScript.FormalExpectationConst)
		validateTypeScriptRuntimeVariants(errs, *raw.Runtime.TypeScript, raw)
	default:
		*errs = append(*errs, "runtime must declare either go or typescript")
	}
}

func validateTypeScriptRuntimeVariants(errs *[]string, rt contract.TypeScriptRuntime, raw contract.Contract) {
	stateIDs := idsFromStates(raw.States)
	eventIDs := idsFromEvents(raw.Events)
	validateVariantMap(errs, "runtime.typescript.stateVariants", rt.StateVariants, stateIDs, rt.PayloadTypes)
	validateVariantMap(errs, "runtime.typescript.eventVariants", rt.EventVariants, eventIDs, rt.PayloadTypes)
	if rt.StateUnionType != "" && len(rt.StateVariants) == 0 {
		*errs = append(*errs, "runtime.typescript.stateUnionType requires exhaustive stateVariants")
	}
	if rt.EventUnionType != "" && len(rt.EventVariants) == 0 {
		*errs = append(*errs, "runtime.typescript.eventUnionType requires exhaustive eventVariants")
	}
}

func validateVariantMap(errs *[]string, path string, variants map[string]map[string]string, knownIDs map[string]bool, payloadTypes map[string]string) {
	if len(variants) == 0 {
		return
	}
	for id := range knownIDs {
		if _, ok := variants[id]; !ok {
			*errs = append(*errs, fmt.Sprintf("%s missing variant for %s", path, id))
		}
	}
	for id, fields := range variants {
		if !knownIDs[id] {
			*errs = append(*errs, fmt.Sprintf("%s references unknown id %s", path, id))
		}
		for field, alias := range fields {
			if strings.TrimSpace(field) == "" {
				*errs = append(*errs, fmt.Sprintf("%s.%s contains an empty field name", path, id))
			}
			if strings.TrimSpace(alias) == "" {
				*errs = append(*errs, fmt.Sprintf("%s.%s.%s contains an empty payload alias", path, id, field))
				continue
			}
			if _, ok := payloadTypes[alias]; !ok {
				*errs = append(*errs, fmt.Sprintf("%s.%s.%s references unknown payload alias %s", path, id, field, alias))
			}
		}
	}
}

func validateReplayShape(errs *[]string, raw contract.Contract) {
	requireString(errs, "replay.transition.function", raw.Replay.Transition.Function)
	switch {
	case raw.Runtime.Go != nil:
		requireString(errs, "replay.transition.stateType", raw.Replay.Transition.StateType)
		requireString(errs, "replay.transition.statusField", raw.Replay.Transition.StatusField)
	case raw.Runtime.TypeScript != nil:
		requireString(errs, "replay.transition.statusAccessor", raw.Replay.Transition.StatusAccessor)
		if accessor := raw.Replay.Transition.StatusAccessor; accessor != "" && !typeScriptStatusAccessorExpr.MatchString(accessor) {
			*errs = append(*errs, "replay.transition.statusAccessor must have the form state.<field>")
		}
	}
}

func idsFromStates(states []contract.State) map[string]bool {
	out := map[string]bool{}
	for _, state := range states {
		out[state.ID] = true
	}
	return out
}

func idsFromEvents(events []contract.Event) map[string]bool {
	out := map[string]bool{}
	for _, event := range events {
		out[event.ID] = true
	}
	return out
}

func requireString(errs *[]string, name string, value string) {
	if strings.TrimSpace(value) == "" {
		*errs = append(*errs, name+" is required")
	}
}

func validateQuintNames(errs *[]string, groups ...map[string]bool) {
	seen := map[string]bool{}
	for _, group := range groups {
		for name := range group {
			if !quintIdentifierPattern.MatchString(name) {
				*errs = append(*errs, "invalid Quint identifier "+name)
			}
			if quint.QuintReservedIdentifiers[name] {
				*errs = append(*errs, "Quint identifier collides with generated helper "+name)
			}
			if seen[name] {
				*errs = append(*errs, "duplicate Quint tag "+name)
			}
			seen[name] = true
		}
	}
}
