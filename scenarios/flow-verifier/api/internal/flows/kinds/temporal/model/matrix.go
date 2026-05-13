package model

import (
	"fmt"

	"flow-verifier/internal/flows/contract"
)

type Pair struct {
	StateID string
	EventID string
}

type Transition struct {
	From      string `json:"from"`
	Event     string `json:"event"`
	To        string `json:"to"`
	WantError bool   `json:"wantError"`
}

type TransitionMatrix struct {
	states  []contract.State
	events  []contract.Event
	rows    []Transition
	byPair  map[Pair]Transition
	byState map[string][]Transition
}

func BuildTransitionMatrix(raw contract.Contract, stateByID map[string]contract.State, eventByID map[string]contract.Event) (TransitionMatrix, []string) {
	var errs []string
	matrix := map[Pair]Transition{}
	for _, state := range raw.States {
		for _, event := range raw.Events {
			def := raw.TransitionDefaults.Invalid
			if state.Terminal && raw.TransitionDefaults.Terminal != nil {
				def = raw.TransitionDefaults.Terminal
			}
			if def == nil {
				continue
			}
			to := ResolveTarget(state.ID, def.To)
			if _, ok := stateByID[to]; !ok {
				errs = append(errs, fmt.Sprintf("transition default for %s/%s references unknown state %s", state.ID, event.ID, to))
			}
			matrix[Pair{StateID: state.ID, EventID: event.ID}] = Transition{From: state.ID, Event: event.ID, To: to, WantError: def.WantError}
		}
	}

	explicit := map[Pair]bool{}
	for i, transition := range raw.Transitions {
		if len(transition.From) == 0 {
			errs = append(errs, fmt.Sprintf("transitions[%d].from is required", i))
		}
		if len(transition.Event) == 0 {
			errs = append(errs, fmt.Sprintf("transitions[%d].event is required", i))
		}
		for _, from := range transition.From {
			if _, ok := stateByID[from]; !ok {
				errs = append(errs, fmt.Sprintf("transitions[%d].from references unknown state %s", i, from))
			}
			for _, event := range transition.Event {
				if _, ok := eventByID[event]; !ok {
					errs = append(errs, fmt.Sprintf("transitions[%d].event references unknown event %s", i, event))
				}
				key := Pair{StateID: from, EventID: event}
				if explicit[key] {
					errs = append(errs, fmt.Sprintf("duplicate transition pair %s/%s", from, event))
				}
				explicit[key] = true
				to := ResolveTarget(from, transition.To)
				if _, ok := stateByID[to]; !ok {
					errs = append(errs, fmt.Sprintf("transitions[%d].to references unknown state %s", i, to))
				}
				wantError := false
				if transition.WantError != nil {
					wantError = *transition.WantError
				}
				matrix[key] = Transition{From: from, Event: event, To: to, WantError: wantError}
			}
		}
	}

	rows := make([]Transition, 0, len(raw.States)*len(raw.Events))
	byPair := make(map[Pair]Transition, len(raw.States)*len(raw.Events))
	byState := make(map[string][]Transition, len(raw.States))
	for _, state := range raw.States {
		for _, event := range raw.Events {
			key := Pair{StateID: state.ID, EventID: event.ID}
			row, ok := matrix[key]
			if !ok {
				errs = append(errs, fmt.Sprintf("missing transition pair %s/%s", state.ID, event.ID))
				continue
			}
			rows = append(rows, row)
			byPair[key] = row
			byState[state.ID] = append(byState[state.ID], row)
		}
	}
	return TransitionMatrix{
		states:  append([]contract.State(nil), raw.States...),
		events:  append([]contract.Event(nil), raw.Events...),
		rows:    rows,
		byPair:  byPair,
		byState: byState,
	}, errs
}

func ResolveTarget(from string, to string) string {
	if to == SelfTarget {
		return from
	}
	return to
}

func (m TransitionMatrix) Rows() []Transition {
	return append([]Transition(nil), m.rows...)
}

func (m TransitionMatrix) RowsFrom(stateID string) []Transition {
	return append([]Transition(nil), m.byState[stateID]...)
}

func (m TransitionMatrix) Lookup(stateID string, eventID string) (Transition, bool) {
	transition, ok := m.byPair[Pair{StateID: stateID, EventID: eventID}]
	return transition, ok
}

func (m TransitionMatrix) Len() int {
	return len(m.rows)
}

func (m TransitionMatrix) Complete() bool {
	return len(m.rows) == len(m.states)*len(m.events)
}

func (m TransitionMatrix) TerminalTransitionsChecked() bool {
	if !m.Complete() {
		return false
	}
	for _, state := range m.states {
		if state.Terminal && len(m.byState[state.ID]) != len(m.events) {
			return false
		}
	}
	return true
}
