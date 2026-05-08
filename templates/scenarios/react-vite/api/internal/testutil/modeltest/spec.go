package modeltest

import (
	"encoding/json"
	"fmt"
	"os"
)

// WorkflowSpec is a declarative workflow contract used by tests to keep
// hand-authored matrices and traces aligned with the documented model.
type WorkflowSpec struct {
	ID             string               `json:"id"`
	Domain         string               `json:"domain"`
	Description    string               `json:"description"`
	States         []string             `json:"states"`
	Events         []string             `json:"events"`
	InitialState   string               `json:"initialState"`
	TerminalStates []string             `json:"terminalStates"`
	Transitions    []SpecTransition     `json:"transitions"`
	Invariants     []string             `json:"invariants"`
	Traces         []SpecTrace          `json:"traces"`
	FormalModel    *SpecFormalModelInfo `json:"formalModel,omitempty"`
}

type SpecTransition struct {
	From      string `json:"from"`
	Event     string `json:"event"`
	To        string `json:"to"`
	WantError bool   `json:"wantError,omitempty"`
}

type SpecTrace struct {
	Name    string          `json:"name"`
	Initial string          `json:"initial"`
	Steps   []SpecTraceStep `json:"steps"`
}

type SpecTraceStep struct {
	Event     string `json:"event"`
	Want      string `json:"want"`
	WantError bool   `json:"wantError,omitempty"`
}

type SpecFormalModelInfo struct {
	Status             string `json:"status"`
	Tool               string `json:"tool,omitempty"`
	GeneratedArtifacts string `json:"generatedArtifacts,omitempty"`
	DriftCheck         string `json:"driftCheck,omitempty"`
}

// LoadWorkflowSpec reads a JSON workflow spec from path.
func LoadWorkflowSpec(t TestingT, path string) WorkflowSpec {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read workflow spec %s: %v", path, err)
	}
	var spec WorkflowSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatalf("parse workflow spec %s: %v", path, err)
	}
	return spec
}

// ValidateWorkflowSpecConformance verifies that a declarative workflow spec
// agrees with the production status/event lists plus matrix and trace tests.
func ValidateWorkflowSpecConformance[S comparable, E comparable](
	spec WorkflowSpec,
	statuses []S,
	events []E,
	rows []MatrixRow[S, E],
	traces []Trace[S, E],
) []error {
	var errs []error
	if spec.ID == "" {
		errs = append(errs, fmt.Errorf("spec id is required"))
	}
	if spec.Domain == "" {
		errs = append(errs, fmt.Errorf("spec domain is required"))
	}
	if spec.InitialState == "" {
		errs = append(errs, fmt.Errorf("spec initialState is required"))
	}
	if len(spec.Transitions) == 0 {
		errs = append(errs, fmt.Errorf("spec transitions must not be empty"))
	}

	specStates := stringSet(spec.States)
	for _, status := range statuses {
		value := fmt.Sprint(status)
		if !specStates[value] {
			errs = append(errs, fmt.Errorf("spec missing production state %s", value))
		}
	}
	for _, state := range spec.States {
		if !containsValue(statuses, state) {
			errs = append(errs, fmt.Errorf("spec state %s is not a production state", state))
		}
	}
	if !specStates[spec.InitialState] {
		errs = append(errs, fmt.Errorf("spec initialState %s is not a known state", spec.InitialState))
	}
	for _, terminal := range spec.TerminalStates {
		if !specStates[terminal] {
			errs = append(errs, fmt.Errorf("spec terminal state %s is not a known state", terminal))
		}
	}

	specEvents := stringSet(spec.Events)
	for _, event := range events {
		value := fmt.Sprint(event)
		if !specEvents[value] {
			errs = append(errs, fmt.Errorf("spec missing production event %s", value))
		}
	}
	for _, event := range spec.Events {
		if !containsValue(events, event) {
			errs = append(errs, fmt.Errorf("spec event %s is not a production event", event))
		}
	}

	matrixByPair := make(map[string]SpecTransition, len(rows))
	for _, row := range rows {
		matrixByPair[pairString(fmt.Sprint(row.From), fmt.Sprint(row.Event))] = SpecTransition{
			From:      fmt.Sprint(row.From),
			Event:     fmt.Sprint(row.Event),
			To:        fmt.Sprint(row.To),
			WantError: row.WantErr,
		}
	}
	specByPair := make(map[string]SpecTransition, len(spec.Transitions))
	for _, transition := range spec.Transitions {
		key := pairString(transition.From, transition.Event)
		if _, ok := specByPair[key]; ok {
			errs = append(errs, fmt.Errorf("spec duplicate transition %s/%s", transition.From, transition.Event))
			continue
		}
		specByPair[key] = transition
		row, ok := matrixByPair[key]
		if !ok {
			errs = append(errs, fmt.Errorf("spec transition %s/%s missing from matrix", transition.From, transition.Event))
			continue
		}
		if transition.To != row.To || transition.WantError != row.WantError {
			errs = append(errs, fmt.Errorf(
				"spec transition %s/%s mismatch: spec to=%s wantError=%v matrix to=%s wantError=%v",
				transition.From,
				transition.Event,
				transition.To,
				transition.WantError,
				row.To,
				row.WantError,
			))
		}
	}
	for key, row := range matrixByPair {
		if _, ok := specByPair[key]; !ok {
			errs = append(errs, fmt.Errorf("matrix transition %s/%s missing from spec", row.From, row.Event))
		}
	}

	traceByName := make(map[string]Trace[S, E], len(traces))
	for i, trace := range traces {
		name := trace.Name
		if name == "" {
			name = fmt.Sprintf("trace %d", i)
		}
		traceByName[name] = trace
	}
	for _, specTrace := range spec.Traces {
		trace, ok := traceByName[specTrace.Name]
		if !ok {
			errs = append(errs, fmt.Errorf("spec trace %s missing from tests", specTrace.Name))
			continue
		}
		if specTrace.Initial != fmt.Sprint(trace.Initial) {
			errs = append(errs, fmt.Errorf("spec trace %s initial=%s test initial=%v", specTrace.Name, specTrace.Initial, trace.Initial))
		}
		if len(specTrace.Steps) != len(trace.Steps) {
			errs = append(errs, fmt.Errorf("spec trace %s step count=%d test step count=%d", specTrace.Name, len(specTrace.Steps), len(trace.Steps)))
			continue
		}
		for i, specStep := range specTrace.Steps {
			testStep := trace.Steps[i]
			if specStep.Event != fmt.Sprint(testStep.Event) ||
				specStep.Want != fmt.Sprint(testStep.Want) ||
				specStep.WantError != testStep.WantErr {
				errs = append(errs, fmt.Errorf("spec trace %s step %d differs from test trace", specTrace.Name, i))
			}
		}
	}
	return errs
}

// AssertWorkflowSpecConformance fails t when the spec and executable tests drift.
func AssertWorkflowSpecConformance[S comparable, E comparable](
	t TestingT,
	spec WorkflowSpec,
	statuses []S,
	events []E,
	rows []MatrixRow[S, E],
	traces []Trace[S, E],
) {
	t.Helper()
	if errs := ValidateWorkflowSpecConformance(spec, statuses, events, rows, traces); len(errs) > 0 {
		t.Fatalf("workflow spec mismatch:\n%s", formatErrors(errs))
	}
}

func containsValue[T comparable](values []T, target string) bool {
	for _, value := range values {
		if fmt.Sprint(value) == target {
			return true
		}
	}
	return false
}

func pairString(state string, event string) string {
	return state + "\x00" + event
}

func stringSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}
