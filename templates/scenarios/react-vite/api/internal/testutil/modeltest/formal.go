package modeltest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FormalArtifact struct {
	SchemaVersion int                        `json:"schemaVersion"`
	FlowID        string                     `json:"flowId"`
	Source        FormalArtifactSource       `json:"source"`
	Commands      map[string][]string        `json:"commands"`
	States        []string                   `json:"states"`
	Events        []string                   `json:"events"`
	Transitions   []FormalArtifactTransition `json:"transitions"`
	Traces        []FormalArtifactTrace      `json:"traces"`
	Checks        FormalArtifactChecks       `json:"checks"`
}

type FormalArtifactSource struct {
	ModelPath    string `json:"modelPath"`
	ModelSHA256  string `json:"modelSha256"`
	QuintVersion string `json:"quintVersion"`
}

type FormalArtifactChecks struct {
	Typechecked        bool `json:"typechecked"`
	Tested             bool `json:"tested"`
	Verified           bool `json:"verified"`
	GeneratedFromModel bool `json:"generatedFromModel"`
}

type FormalArtifactTransition struct {
	From      string `json:"from"`
	Event     string `json:"event"`
	To        string `json:"to"`
	WantError bool   `json:"wantError"`
}

type FormalArtifactTrace struct {
	Name    string                    `json:"name"`
	Initial string                    `json:"initial"`
	Steps   []FormalArtifactTraceStep `json:"steps"`
}

type FormalArtifactTraceStep struct {
	Event     string `json:"event"`
	Want      string `json:"want"`
	WantError bool   `json:"wantError"`
}

func LoadFormalArtifact(t TestingT, path string) FormalArtifact {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read formal artifact %s: %v", path, err)
	}
	var artifact FormalArtifact
	if err := json.Unmarshal(data, &artifact); err != nil {
		t.Fatalf("parse formal artifact %s: %v", path, err)
	}
	return artifact
}

func ValidateFormalArtifactFresh(artifact FormalArtifact, modelPath string) []error {
	var errs []error
	if artifact.SchemaVersion != 1 {
		errs = append(errs, fmt.Errorf("formal artifact schemaVersion=%d, want 1", artifact.SchemaVersion))
	}
	if artifact.FlowID == "" {
		errs = append(errs, fmt.Errorf("formal artifact flowId is required"))
	}
	if artifact.Source.ModelPath != modelPath {
		errs = append(errs, fmt.Errorf("formal artifact modelPath=%s, want %s", artifact.Source.ModelPath, modelPath))
	}
	if artifact.Source.QuintVersion == "" {
		errs = append(errs, fmt.Errorf("formal artifact quintVersion is required"))
	}
	if artifact.Source.ModelSHA256 == "" {
		errs = append(errs, fmt.Errorf("formal artifact modelSha256 is required"))
	} else {
		data, err := readModelFile(modelPath)
		if err != nil {
			errs = append(errs, fmt.Errorf("read formal model %s: %v", modelPath, err))
		} else {
			sum := sha256.Sum256(data)
			got := hex.EncodeToString(sum[:])
			if got != artifact.Source.ModelSHA256 {
				errs = append(errs, fmt.Errorf("formal artifact modelSha256=%s, want %s", artifact.Source.ModelSHA256, got))
			}
		}
	}
	if !artifact.Checks.Typechecked {
		errs = append(errs, fmt.Errorf("formal artifact was not typechecked"))
	}
	if !artifact.Checks.Tested {
		errs = append(errs, fmt.Errorf("formal artifact was not tested"))
	}
	if !artifact.Checks.Verified {
		errs = append(errs, fmt.Errorf("formal artifact was not verified"))
	}
	if !artifact.Checks.GeneratedFromModel {
		errs = append(errs, fmt.Errorf("formal artifact was not generated from model"))
	}
	if len(artifact.Transitions) == 0 {
		errs = append(errs, fmt.Errorf("formal artifact transitions must not be empty"))
	}
	if len(artifact.Traces) == 0 {
		errs = append(errs, fmt.Errorf("formal artifact traces must not be empty"))
	}
	return errs
}

func readModelFile(modelPath string) ([]byte, error) {
	var firstErr error
	candidates := []string{modelPath}
	if trimmed, ok := strings.CutPrefix(modelPath, "api/"); ok {
		candidates = append(candidates, trimmed)
	}
	candidates = append(candidates, filepath.Base(modelPath))

	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		for _, candidate := range candidates {
			data, err := os.ReadFile(filepath.Join(dir, candidate))
			if err == nil {
				return data, nil
			}
			if firstErr == nil {
				firstErr = err
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return nil, firstErr
}

func AssertFormalArtifactFresh(t TestingT, artifact FormalArtifact, modelPath string) {
	t.Helper()
	if errs := ValidateFormalArtifactFresh(artifact, modelPath); len(errs) > 0 {
		t.Fatalf("formal artifact is stale or incomplete:\n%s", formatErrors(errs))
	}
}

func ValidateFormalTransitionsReplay[S comparable, E comparable](
	artifact FormalArtifact,
	statuses []S,
	events []E,
	transition Transition[S, E],
) []error {
	rows, errs := formalRows(artifact, statuses, events)
	if len(errs) > 0 {
		return errs
	}
	return ValidateTransitionMatrix(statuses, events, rows, transition)
}

func AssertFormalTransitionsReplay[S comparable, E comparable](
	t TestingT,
	artifact FormalArtifact,
	statuses []S,
	events []E,
	transition Transition[S, E],
) {
	t.Helper()
	if errs := ValidateFormalTransitionsReplay(artifact, statuses, events, transition); len(errs) > 0 {
		t.Fatalf("formal transition replay mismatch:\n%s", formatErrors(errs))
	}
}

func ValidateFormalTracesReplay[S comparable, E comparable](
	artifact FormalArtifact,
	statuses []S,
	events []E,
	transition Transition[S, E],
) []error {
	traces, errs := formalTraces(artifact, statuses, events)
	if len(errs) > 0 {
		return errs
	}
	return ValidateTraces(traces, transition)
}

func AssertFormalTracesReplay[S comparable, E comparable](
	t TestingT,
	artifact FormalArtifact,
	statuses []S,
	events []E,
	transition Transition[S, E],
) {
	t.Helper()
	if errs := ValidateFormalTracesReplay(artifact, statuses, events, transition); len(errs) > 0 {
		t.Fatalf("formal trace replay mismatch:\n%s", formatErrors(errs))
	}
}

func formalRows[S comparable, E comparable](
	artifact FormalArtifact,
	statuses []S,
	events []E,
) ([]MatrixRow[S, E], []error) {
	statusByName := valuesByString(statuses)
	eventByName := valuesByString(events)
	rows := make([]MatrixRow[S, E], 0, len(artifact.Transitions))
	var errs []error
	for i, transition := range artifact.Transitions {
		bad := false
		from, ok := statusByName[transition.From]
		if !ok {
			errs = append(errs, fmt.Errorf("formal transition %d unknown from state %s", i, transition.From))
			bad = true
		}
		to, ok := statusByName[transition.To]
		if !ok {
			errs = append(errs, fmt.Errorf("formal transition %d unknown to state %s", i, transition.To))
			bad = true
		}
		event, ok := eventByName[transition.Event]
		if !ok {
			errs = append(errs, fmt.Errorf("formal transition %d unknown event %s", i, transition.Event))
			bad = true
		}
		if bad {
			continue
		}
		rows = append(rows, MatrixRow[S, E]{
			Name:    fmt.Sprintf("formal transition %s/%s", transition.From, transition.Event),
			From:    from,
			Event:   event,
			To:      to,
			WantErr: transition.WantError,
		})
	}
	return rows, errs
}

func formalTraces[S comparable, E comparable](
	artifact FormalArtifact,
	statuses []S,
	events []E,
) ([]Trace[S, E], []error) {
	statusByName := valuesByString(statuses)
	eventByName := valuesByString(events)
	traces := make([]Trace[S, E], 0, len(artifact.Traces))
	var errs []error
	for traceIndex, formalTrace := range artifact.Traces {
		initial, ok := statusByName[formalTrace.Initial]
		if !ok {
			errs = append(errs, fmt.Errorf("formal trace %s unknown initial state %s", formalTrace.Name, formalTrace.Initial))
			continue
		}
		steps := make([]TraceStep[S, E], 0, len(formalTrace.Steps))
		for stepIndex, step := range formalTrace.Steps {
			bad := false
			event, ok := eventByName[step.Event]
			if !ok {
				errs = append(errs, fmt.Errorf("formal trace %s step %d unknown event %s", formalTrace.Name, stepIndex, step.Event))
				bad = true
			}
			want, ok := statusByName[step.Want]
			if !ok {
				errs = append(errs, fmt.Errorf("formal trace %s step %d unknown want state %s", formalTrace.Name, stepIndex, step.Want))
				bad = true
			}
			if bad {
				continue
			}
			steps = append(steps, TraceStep[S, E]{
				Name:    fmt.Sprintf("step %d", stepIndex),
				Event:   event,
				Want:    want,
				WantErr: step.WantError,
			})
		}
		traces = append(traces, Trace[S, E]{
			Name:    traceName(formalTrace.Name, traceIndex),
			Initial: initial,
			Steps:   steps,
		})
	}
	return traces, errs
}

func valuesByString[T comparable](values []T) map[string]T {
	byName := make(map[string]T, len(values))
	for _, value := range values {
		byName[fmt.Sprint(value)] = value
	}
	return byName
}

func traceName(name string, index int) string {
	if name != "" {
		return name
	}
	return fmt.Sprintf("formal trace %d", index)
}
