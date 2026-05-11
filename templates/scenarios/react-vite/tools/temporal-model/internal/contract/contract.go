package contract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const SchemaVersion = 3

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
	Runtime             Runtime              `json:"runtime,omitempty"`
	Replay              Replay               `json:"replay"`
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
	ModelPath        string `json:"modelPath"`
	ArtifactPath     string `json:"artifactPath"`
	DeclarationsPath string `json:"declarationsPath"`
}

type Runtime struct {
	Go              *GoRuntime         `json:"go,omitempty"`
	TypeScript      *TypeScriptRuntime `json:"typescript,omitempty"`
	SideEffects     []string           `json:"sideEffects,omitempty"`
	StaleCompletion string             `json:"staleCompletion,omitempty"`
}

type GoRuntime struct {
	Package        string `json:"package"`
	StatusType     string `json:"statusType"`
	EventType      string `json:"eventType"`
	ConstantPrefix string `json:"constantPrefix"`
}

type TypeScriptRuntime struct {
	StatusType             string                       `json:"statusType"`
	EventType              string                       `json:"eventType"`
	StatusesConst          string                       `json:"statusesConst"`
	EventsConst            string                       `json:"eventsConst"`
	FormalExpectationConst string                       `json:"formalExpectationConst"`
	StateUnionType         string                       `json:"stateUnionType,omitempty"`
	EventUnionType         string                       `json:"eventUnionType,omitempty"`
	PayloadTypes           map[string]string            `json:"payloadTypes,omitempty"`
	StateVariants          map[string]map[string]string `json:"stateVariants,omitempty"`
	EventVariants          map[string]map[string]string `json:"eventVariants,omitempty"`
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

type Replay struct {
	Bindings []ReplayBinding `json:"bindings"`
}

type ReplayBinding struct {
	Kind      string `json:"kind"`
	Path      string `json:"path"`
	Assertion string `json:"assertion"`
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
	if err := validateJSONSchema(data); err != nil {
		return Contract{}, fmt.Errorf("schema validation failed for %s:\n%s", relPath, err)
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

var (
	flowSchemaOnce sync.Once
	flowSchema     *jsonschema.Schema
	flowSchemaErr  error
)

func validateJSONSchema(data []byte) error {
	schema, err := compiledFlowSchema()
	if err != nil {
		return err
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if err := schema.Validate(value); err != nil {
		return err
	}
	return nil
}

func compiledFlowSchema() (*jsonschema.Schema, error) {
	flowSchemaOnce.Do(func() {
		_, current, _, ok := runtime.Caller(0)
		if !ok {
			flowSchemaErr = fmt.Errorf("locate flow.schema.json: runtime caller unavailable")
			return
		}
		schemaPath := filepath.Join(filepath.Dir(current), "..", "..", "flow.schema.json")
		data, err := os.ReadFile(schemaPath)
		if err != nil {
			flowSchemaErr = fmt.Errorf("read flow.schema.json: %w", err)
			return
		}
		var schemaDoc any
		if err := json.Unmarshal(data, &schemaDoc); err != nil {
			flowSchemaErr = fmt.Errorf("parse flow.schema.json: %w", err)
			return
		}
		compiler := jsonschema.NewCompiler()
		const schemaURL = "https://vrooli.dev/schemas/react-vite/temporal-flow.json"
		if err := compiler.AddResource(schemaURL, schemaDoc); err != nil {
			flowSchemaErr = fmt.Errorf("load flow.schema.json: %w", err)
			return
		}
		flowSchema, flowSchemaErr = compiler.Compile(schemaURL)
	})
	return flowSchema, flowSchemaErr
}

func ValidateAndExpand(c *Contract) error {
	var errs []string
	if c.SchemaVersion != SchemaVersion {
		errs = append(errs, fmt.Sprintf("schemaVersion must be %d", SchemaVersion))
	}
	requireString(&errs, "flowId", c.FlowID)
	requireString(&errs, "domain", c.Domain)
	requireString(&errs, "description", c.Description)
	requireString(&errs, "model.module", c.Model.Module)
	requireString(&errs, "model.seed", c.Model.Seed)
	requireString(&errs, "outputs.modelPath", c.Outputs.ModelPath)
	requireString(&errs, "outputs.artifactPath", c.Outputs.ArtifactPath)
	requireString(&errs, "outputs.declarationsPath", c.Outputs.DeclarationsPath)
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
	validateRuntime(&errs, *c)
	validateReplayBindingsShape(&errs, *c)
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
	matrix := map[string]ExpandedTransition{}
	for _, transition := range c.ExpandedTransitions {
		matrix[pair(transition.From, transition.Event)] = transition
	}
	for i, trace := range c.Traces {
		if trace.Name == "" {
			*errs = append(*errs, fmt.Sprintf("traces[%d].name is required", i))
		}
		if !states[trace.Initial] {
			*errs = append(*errs, fmt.Sprintf("traces[%d].initial references unknown state %s", i, trace.Initial))
		}
		current := trace.Initial
		for j, step := range trace.Steps {
			if !events[step.Event] {
				*errs = append(*errs, fmt.Sprintf("traces[%d].steps[%d].event references unknown event %s", i, j, step.Event))
			}
			if !states[step.Want] {
				*errs = append(*errs, fmt.Sprintf("traces[%d].steps[%d].want references unknown state %s", i, j, step.Want))
			}
			if !states[current] || !events[step.Event] {
				continue
			}
			expanded, ok := matrix[pair(current, step.Event)]
			if !ok {
				*errs = append(*errs, fmt.Sprintf("traces[%d:%s].steps[%d] has no expanded transition for %s/%s", i, trace.Name, j, current, step.Event))
				continue
			}
			if step.Want != expanded.To || step.WantError != expanded.WantError {
				*errs = append(*errs, fmt.Sprintf(
					"traces[%d:%s].steps[%d] from %s with %s declares want=%s wantError=%v, expanded transition wants %s wantError=%v",
					i,
					trace.Name,
					j,
					current,
					step.Event,
					step.Want,
					step.WantError,
					expanded.To,
					expanded.WantError,
				))
			}
			current = expanded.To
		}
	}
}

func validateRuntime(errs *[]string, c Contract) {
	if strings.HasSuffix(c.Outputs.DeclarationsPath, ".go") {
		if c.Runtime.Go == nil {
			*errs = append(*errs, "runtime.go is required for Go declarations")
			return
		}
		requireString(errs, "runtime.go.package", c.Runtime.Go.Package)
		requireString(errs, "runtime.go.statusType", c.Runtime.Go.StatusType)
		requireString(errs, "runtime.go.eventType", c.Runtime.Go.EventType)
		requireString(errs, "runtime.go.constantPrefix", c.Runtime.Go.ConstantPrefix)
	}
	if strings.HasSuffix(c.Outputs.DeclarationsPath, ".ts") {
		if c.Runtime.TypeScript == nil {
			*errs = append(*errs, "runtime.typescript is required for TypeScript declarations")
			return
		}
		requireString(errs, "runtime.typescript.statusType", c.Runtime.TypeScript.StatusType)
		requireString(errs, "runtime.typescript.eventType", c.Runtime.TypeScript.EventType)
		requireString(errs, "runtime.typescript.statusesConst", c.Runtime.TypeScript.StatusesConst)
		requireString(errs, "runtime.typescript.eventsConst", c.Runtime.TypeScript.EventsConst)
		requireString(errs, "runtime.typescript.formalExpectationConst", c.Runtime.TypeScript.FormalExpectationConst)
		validateTypeScriptRuntimeVariants(errs, *c.Runtime.TypeScript, c)
	}
}

func validateTypeScriptRuntimeVariants(errs *[]string, rt TypeScriptRuntime, c Contract) {
	stateIDs := idsFromStates(c.States)
	eventIDs := idsFromEvents(c.Events)
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

func idsFromStates(states []State) map[string]bool {
	out := map[string]bool{}
	for _, state := range states {
		out[state.ID] = true
	}
	return out
}

func idsFromEvents(events []Event) map[string]bool {
	out := map[string]bool{}
	for _, event := range events {
		out[event.ID] = true
	}
	return out
}

func validateReplayBindingsShape(errs *[]string, c Contract) {
	if len(c.Replay.Bindings) == 0 {
		*errs = append(*errs, "replay.bindings must declare at least one production replay test binding")
	}
	kinds := map[string]bool{"go-test": true, "vitest": true}
	for i, binding := range c.Replay.Bindings {
		if !kinds[binding.Kind] {
			*errs = append(*errs, fmt.Sprintf("replay.bindings[%d].kind must be one of go-test, vitest", i))
		}
		requireString(errs, fmt.Sprintf("replay.bindings[%d].path", i), binding.Path)
		requireString(errs, fmt.Sprintf("replay.bindings[%d].assertion", i), binding.Assertion)
		if binding.Path != "" {
			clean := filepath.ToSlash(filepath.Clean(binding.Path))
			if filepath.IsAbs(binding.Path) || clean == "." || strings.HasPrefix(clean, "../") || clean == ".." {
				*errs = append(*errs, fmt.Sprintf("replay.bindings[%d].path must be a relative path inside the scenario root", i))
			}
		}
	}
}

func ValidateReplayBindings(c Contract, root string) error {
	var errs []string
	for i, binding := range c.Replay.Bindings {
		clean := filepath.ToSlash(filepath.Clean(binding.Path))
		if filepath.IsAbs(binding.Path) || clean == "." || strings.HasPrefix(clean, "../") || clean == ".." {
			errs = append(errs, fmt.Sprintf("replay.bindings[%d].path must be a relative path inside the scenario root", i))
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(clean)))
		if err != nil {
			errs = append(errs, fmt.Sprintf("replay binding %s for %s is missing or unreadable: %v", clean, c.FlowID, err))
			continue
		}
		body := string(data)
		if !strings.Contains(body, binding.Assertion) {
			errs = append(errs, fmt.Sprintf("replay binding %s for %s does not contain assertion marker %q", clean, c.FlowID, binding.Assertion))
		}
		for _, helper := range helperMarkers(binding.Kind) {
			if !strings.Contains(body, helper) {
				errs = append(errs, fmt.Sprintf("replay binding %s for %s does not contain helper marker %q", clean, c.FlowID, helper))
			}
		}
	}
	if len(errs) > 0 {
		sort.Strings(errs)
		return fmt.Errorf("invalid replay bindings for %s:\n  - %s", c.FlowID, strings.Join(errs, "\n  - "))
	}
	return nil
}

func helperMarkers(kind string) []string {
	switch kind {
	case "go-test":
		return []string{"AssertFormalArtifactFresh", "AssertFormalTransitionsReplay", "AssertFormalTracesReplay"}
	case "vitest":
		return []string{"assertFormalArtifactFresh", "assertFormalTransitionsReplay", "assertFormalTracesReplay"}
	default:
		return nil
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
