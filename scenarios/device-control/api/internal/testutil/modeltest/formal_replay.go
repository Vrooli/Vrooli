package modeltest

import "fmt"

func AssertFormalArtifactFresh(t TestingT, artifact FormalArtifact, expected FormalArtifactExpectation) {
	t.Helper()
	if errs := ValidateFormalArtifactFresh(artifact, expected); len(errs) > 0 {
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
	traces, errs := formalTraces(append(artifact.NamedTraces, artifact.GeneratedTraces...), statuses, events)
	if len(errs) > 0 {
		return errs
	}
	return ValidateTraces(traces, transition)
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
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
	artifactTraces []FormalArtifactTrace,
	statuses []S,
	events []E,
) ([]Trace[S, E], []error) {
	statusByName := valuesByString(statuses)
	eventByName := valuesByString(events)
	traces := make([]Trace[S, E], 0, len(artifactTraces))
	var errs []error
	for traceIndex, formalTrace := range artifactTraces {
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
