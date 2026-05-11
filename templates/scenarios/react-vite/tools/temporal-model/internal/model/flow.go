package model

import "react-vite-temporal-model/internal/contract"

const (
	SchemaVersion                 = 4
	SelfTarget                    = "self"
	GeneratedCheckTransitionTable = "transitionTable"
)

type ReplayKind string

const (
	ReplayKindGoTest ReplayKind = contract.ReplayKindGoTest
	ReplayKindVitest ReplayKind = contract.ReplayKindVitest
)

type Flow struct {
	SchemaVersion      int
	FlowID             string
	Domain             string
	Description        string
	ContractPath       string
	Model              contract.Model
	Outputs            contract.Outputs
	States             []contract.State
	Events             []contract.Event
	TransitionDefaults contract.TransitionDefaults
	Transitions        []contract.Transition
	Invariants         []contract.Invariant
	Traces             []contract.Trace
	Runtime            contract.Runtime
	Replay             contract.Replay
	Initial            contract.State
	Matrix             TransitionMatrix
	StateByID          map[string]contract.State
	EventByID          map[string]contract.Event
	InvariantByQuint   map[string]contract.Invariant
}

func FromRaw(raw contract.Contract, initial contract.State, matrix TransitionMatrix, indexes Indexes) Flow {
	return Flow{
		SchemaVersion:      raw.SchemaVersion,
		FlowID:             raw.FlowID,
		Domain:             raw.Domain,
		Description:        raw.Description,
		ContractPath:       raw.ContractPath,
		Model:              raw.Model,
		Outputs:            raw.Outputs,
		States:             append([]contract.State(nil), raw.States...),
		Events:             append([]contract.Event(nil), raw.Events...),
		TransitionDefaults: raw.TransitionDefaults,
		Transitions:        append([]contract.Transition(nil), raw.Transitions...),
		Invariants:         append([]contract.Invariant(nil), raw.Invariants...),
		Traces:             append([]contract.Trace(nil), raw.Traces...),
		Runtime:            raw.Runtime,
		Replay:             raw.Replay,
		Initial:            initial,
		Matrix:             matrix,
		StateByID:          indexes.StateByID,
		EventByID:          indexes.EventByID,
		InvariantByQuint:   indexes.InvariantByQuint,
	}
}

type Indexes struct {
	StateByID        map[string]contract.State
	StateQuintByID   map[string]string
	EventByID        map[string]contract.Event
	EventQuintByID   map[string]string
	InvariantByQuint map[string]contract.Invariant
}

func StateIDs(flow Flow) []string {
	out := make([]string, 0, len(flow.States))
	for _, state := range flow.States {
		out = append(out, state.ID)
	}
	return out
}

func EventIDs(flow Flow) []string {
	out := make([]string, 0, len(flow.Events))
	for _, event := range flow.Events {
		out = append(out, event.ID)
	}
	return out
}
