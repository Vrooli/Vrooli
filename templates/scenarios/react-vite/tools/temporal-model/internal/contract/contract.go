package contract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

type Contract struct {
	SchemaVersion       int                  `json:"schemaVersion"`
	FlowID              string               `json:"flowId"`
	Domain              string               `json:"domain"`
	Description         string               `json:"description"`
	Model               Model                `json:"model"`
	Outputs             Outputs              `json:"outputs"`
	States              []State              `json:"states"`
	Events              []Event              `json:"events"`
	TransitionDefaults  TransitionDefaults   `json:"transitionDefaults"`
	Transitions         []Transition         `json:"transitions"`
	Invariants          []Invariant          `json:"invariants"`
	Traces              []Trace              `json:"traces"`
	Runtime             map[string]any       `json:"runtime,omitempty"`
	ContractPath        string               `json:"-"`
	ExpandedTransitions []ExpandedTransition `json:"-"`
}

type Model struct {
	Module     string `json:"module"`
	Seed       string `json:"seed"`
	MaxSteps   int    `json:"maxSteps"`
	TraceCount int    `json:"traceCount"`
	Verify     Verify `json:"verify"`
}

type Verify struct {
	Invariants []string `json:"invariants"`
}

type Outputs struct {
	ModelPath    string `json:"modelPath"`
	ArtifactPath string `json:"artifactPath"`
}

type State struct {
	ID       string `json:"id"`
	Quint    string `json:"quint"`
	Initial  bool   `json:"initial,omitempty"`
	Terminal bool   `json:"terminal,omitempty"`
}

type Event struct {
	ID    string `json:"id"`
	Quint string `json:"quint"`
}

type TransitionDefaults struct {
	Invalid  *DefaultTransition `json:"invalid,omitempty"`
	Terminal *DefaultTransition `json:"terminal,omitempty"`
}

type DefaultTransition struct {
	To        string `json:"to"`
	WantError bool   `json:"wantError"`
}

type Transition struct {
	From      StringList `json:"from"`
	Event     StringList `json:"event"`
	To        string     `json:"to"`
	WantError *bool      `json:"wantError,omitempty"`
}

type ExpandedTransition struct {
	From      string `json:"from"`
	Event     string `json:"event"`
	To        string `json:"to"`
	WantError bool   `json:"wantError"`
}

type Invariant struct {
	ID          string `json:"id"`
	Quint       string `json:"quint"`
	Description string `json:"description"`
	Expression  string `json:"expression,omitempty"`
}

type Trace struct {
	Name    string      `json:"name"`
	Initial string      `json:"initial"`
	Steps   []TraceStep `json:"steps"`
}

type TraceStep struct {
	Event     string `json:"event"`
	Want      string `json:"want"`
	WantError bool   `json:"wantError"`
}

type StringList []string

func (s *StringList) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*s = []string{single}
		return nil
	}
	var many []string
	if err := json.Unmarshal(data, &many); err != nil {
		return err
	}
	*s = many
	return nil
}

func Load(path string, relPath string) (Contract, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Contract{}, err
	}
	var c Contract
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&c); err != nil {
		return Contract{}, fmt.Errorf("parse %s: %w", relPath, err)
	}
	c.ContractPath = relPath
	if err := ValidateAndExpand(&c); err != nil {
		return Contract{}, fmt.Errorf("invalid temporal flow contract %s:\n%s", relPath, err)
	}
	return c, nil
}

func ValidateAndExpand(c *Contract) error {
	var errs []string
	if c.SchemaVersion != 2 {
		errs = append(errs, "schemaVersion must be 2")
	}
	requireString(&errs, "flowId", c.FlowID)
	requireString(&errs, "domain", c.Domain)
	requireString(&errs, "description", c.Description)
	requireString(&errs, "model.module", c.Model.Module)
	requireString(&errs, "model.seed", c.Model.Seed)
	requireString(&errs, "outputs.modelPath", c.Outputs.ModelPath)
	requireString(&errs, "outputs.artifactPath", c.Outputs.ArtifactPath)
	if c.Model.MaxSteps < 1 {
		errs = append(errs, "model.maxSteps must be a positive integer")
	}
	if c.Model.TraceCount < 1 {
		errs = append(errs, "model.traceCount must be a positive integer")
	}
	if len(c.Model.Verify.Invariants) == 0 {
		errs = append(errs, "model.verify.invariants must declare at least one invariant")
	}

	stateIDs, stateQuints := namedStateMaps(&errs, c.States)
	eventIDs, eventQuints := namedEventMaps(&errs, c.Events)
	invariantQuints := namedInvariantMaps(&errs, c.Invariants)
	validateQuintNames(&errs, stateQuints, eventQuints, invariantQuints)

	initials := 0
	terminal := map[string]bool{}
	for _, state := range c.States {
		if state.Initial {
			initials++
		}
		if state.Terminal {
			terminal[state.ID] = true
		}
	}
	if initials != 1 {
		errs = append(errs, fmt.Sprintf("states must declare exactly one initial state, got %d", initials))
	}
	for _, inv := range c.Model.Verify.Invariants {
		if !invariantQuints[inv] {
			errs = append(errs, fmt.Sprintf("model.verify.invariants references unknown invariant %s", inv))
		}
	}
	if len(c.Transitions) == 0 {
		errs = append(errs, "transitions must not be empty")
	}
	if c.TransitionDefaults.Invalid == nil {
		errs = append(errs, "transitionDefaults.invalid is required")
	}
	if len(errs) == 0 {
		c.ExpandedTransitions = expandTransitions(&errs, *c, stateIDs, eventIDs, terminal)
	}
	validateTraces(&errs, *c, stateIDs, eventIDs)
	if len(errs) > 0 {
		sort.Strings(errs)
		return fmt.Errorf("  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

func expandTransitions(errs *[]string, c Contract, stateIDs map[string]bool, eventIDs map[string]bool, terminal map[string]bool) []ExpandedTransition {
	matrix := map[string]ExpandedTransition{}
	for _, state := range c.States {
		for _, event := range c.Events {
			def := c.TransitionDefaults.Invalid
			if terminal[state.ID] && c.TransitionDefaults.Terminal != nil {
				def = c.TransitionDefaults.Terminal
			}
			to := resolveTo(*errs, state.ID, def.To)
			if !stateIDs[to] {
				*errs = append(*errs, fmt.Sprintf("transition default for %s/%s references unknown state %s", state.ID, event.ID, to))
			}
			matrix[pair(state.ID, event.ID)] = ExpandedTransition{From: state.ID, Event: event.ID, To: to, WantError: def.WantError}
		}
	}
	explicit := map[string]bool{}
	for i, t := range c.Transitions {
		if len(t.From) == 0 {
			*errs = append(*errs, fmt.Sprintf("transitions[%d].from is required", i))
		}
		if len(t.Event) == 0 {
			*errs = append(*errs, fmt.Sprintf("transitions[%d].event is required", i))
		}
		for _, from := range t.From {
			if !stateIDs[from] {
				*errs = append(*errs, fmt.Sprintf("transitions[%d].from references unknown state %s", i, from))
			}
			for _, event := range t.Event {
				if !eventIDs[event] {
					*errs = append(*errs, fmt.Sprintf("transitions[%d].event references unknown event %s", i, event))
				}
				key := pair(from, event)
				if explicit[key] {
					*errs = append(*errs, fmt.Sprintf("duplicate transition pair %s/%s", from, event))
				}
				explicit[key] = true
				to := resolveTo(*errs, from, t.To)
				if !stateIDs[to] {
					*errs = append(*errs, fmt.Sprintf("transitions[%d].to references unknown state %s", i, to))
				}
				wantError := false
				if t.WantError != nil {
					wantError = *t.WantError
				}
				matrix[key] = ExpandedTransition{From: from, Event: event, To: to, WantError: wantError}
			}
		}
	}
	out := make([]ExpandedTransition, 0, len(c.States)*len(c.Events))
	for _, state := range c.States {
		for _, event := range c.Events {
			row, ok := matrix[pair(state.ID, event.ID)]
			if !ok {
				*errs = append(*errs, fmt.Sprintf("missing transition pair %s/%s", state.ID, event.ID))
				continue
			}
			out = append(out, row)
		}
	}
	return out
}

func resolveTo(_ []string, from string, to string) string {
	if to == "self" {
		return from
	}
	return to
}

func pair(state string, event string) string {
	return state + "\x00" + event
}

func validateTraces(errs *[]string, c Contract, states map[string]bool, events map[string]bool) {
	if len(c.Traces) == 0 {
		*errs = append(*errs, "traces must not be empty")
	}
	for i, trace := range c.Traces {
		if trace.Name == "" {
			*errs = append(*errs, fmt.Sprintf("traces[%d].name is required", i))
		}
		if !states[trace.Initial] {
			*errs = append(*errs, fmt.Sprintf("traces[%d].initial references unknown state %s", i, trace.Initial))
		}
		for j, step := range trace.Steps {
			if !events[step.Event] {
				*errs = append(*errs, fmt.Sprintf("traces[%d].steps[%d].event references unknown event %s", i, j, step.Event))
			}
			if !states[step.Want] {
				*errs = append(*errs, fmt.Sprintf("traces[%d].steps[%d].want references unknown state %s", i, j, step.Want))
			}
		}
	}
}

func requireString(errs *[]string, name string, value string) {
	if strings.TrimSpace(value) == "" {
		*errs = append(*errs, name+" is required")
	}
}

func namedStateMaps(errs *[]string, states []State) (map[string]bool, map[string]bool) {
	ids := map[string]bool{}
	quints := map[string]bool{}
	if len(states) == 0 {
		*errs = append(*errs, "states must not be empty")
	}
	for i, state := range states {
		requireString(errs, fmt.Sprintf("states[%d].id", i), state.ID)
		requireString(errs, fmt.Sprintf("states[%d].quint", i), state.Quint)
		if ids[state.ID] {
			*errs = append(*errs, "duplicate states id "+state.ID)
		}
		if quints[state.Quint] {
			*errs = append(*errs, "duplicate Quint tag "+state.Quint)
		}
		ids[state.ID] = true
		quints[state.Quint] = true
	}
	return ids, quints
}

func namedEventMaps(errs *[]string, events []Event) (map[string]bool, map[string]bool) {
	ids := map[string]bool{}
	quints := map[string]bool{}
	if len(events) == 0 {
		*errs = append(*errs, "events must not be empty")
	}
	for i, event := range events {
		requireString(errs, fmt.Sprintf("events[%d].id", i), event.ID)
		requireString(errs, fmt.Sprintf("events[%d].quint", i), event.Quint)
		if ids[event.ID] {
			*errs = append(*errs, "duplicate events id "+event.ID)
		}
		if quints[event.Quint] {
			*errs = append(*errs, "duplicate Quint tag "+event.Quint)
		}
		ids[event.ID] = true
		quints[event.Quint] = true
	}
	return ids, quints
}

func namedInvariantMaps(errs *[]string, invariants []Invariant) map[string]bool {
	quints := map[string]bool{}
	if len(invariants) == 0 {
		*errs = append(*errs, "invariants must not be empty")
	}
	for i, invariant := range invariants {
		requireString(errs, fmt.Sprintf("invariants[%d].id", i), invariant.ID)
		requireString(errs, fmt.Sprintf("invariants[%d].quint", i), invariant.Quint)
		requireString(errs, fmt.Sprintf("invariants[%d].description", i), invariant.Description)
		if quints[invariant.Quint] {
			*errs = append(*errs, "duplicate Quint tag "+invariant.Quint)
		}
		quints[invariant.Quint] = true
	}
	return quints
}

func validateQuintNames(errs *[]string, groups ...map[string]bool) {
	reserved := map[string]bool{
		"Status": true, "Event": true, "init": true, "step": true, "apply": true,
		"isValid": true, "nextStatus": true, "transitionTable": true, "rejected": true,
		"status": true, "event": true,
	}
	valid := regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
	seen := map[string]bool{}
	for _, group := range groups {
		for name := range group {
			if !valid.MatchString(name) {
				*errs = append(*errs, "invalid Quint identifier "+name)
			}
			if reserved[name] {
				*errs = append(*errs, "Quint identifier collides with generated helper "+name)
			}
			if seen[name] {
				*errs = append(*errs, "duplicate Quint tag "+name)
			}
			seen[name] = true
		}
	}
}

func Initial(c Contract) State {
	for _, state := range c.States {
		if state.Initial {
			return state
		}
	}
	return State{}
}
