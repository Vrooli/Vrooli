package modeltest

import "fmt"

// TraceStep describes one event in a replayed workflow trace.
type TraceStep[S comparable, E comparable] struct {
	Name    string
	Event   E
	Want    S
	WantErr bool
}

// Trace describes an ordered workflow path to replay against production code.
type Trace[S comparable, E comparable] struct {
	Name    string
	Initial S
	Steps   []TraceStep[S, E]
}

// ValidateTraces replays traces against transition and reports all mismatches.
func ValidateTraces[S comparable, E comparable](
	traces []Trace[S, E],
	transition Transition[S, E],
) []error {
	var errs []error
	if transition == nil {
		return []error{fmt.Errorf("transition must not be nil")}
	}

	for traceIndex, trace := range traces {
		traceName := trace.Name
		if traceName == "" {
			traceName = fmt.Sprintf("trace %d", traceIndex)
		}
		state := trace.Initial
		for stepIndex, step := range trace.Steps {
			stepName := step.Name
			if stepName == "" {
				stepName = fmt.Sprintf("step %d", stepIndex)
			}
			got, err := transition(state, step.Event)
			if step.WantErr {
				if err == nil {
					errs = append(errs, fmt.Errorf("%s/%s: expected error, got nil", traceName, stepName))
				}
			} else if err != nil {
				errs = append(errs, fmt.Errorf("%s/%s: unexpected error: %v", traceName, stepName, err))
			}
			if got != step.Want {
				errs = append(errs, fmt.Errorf("%s/%s: got status %v, want %v", traceName, stepName, got, step.Want))
			}
			state = got
		}
	}
	return errs
}

// ReplayTraces fails t when any trace diverges from the transition function.
func ReplayTraces[S comparable, E comparable](
	t TestingT,
	traces []Trace[S, E],
	transition Transition[S, E],
) {
	t.Helper()
	if errs := ValidateTraces(traces, transition); len(errs) > 0 {
		t.Fatalf("trace replay mismatch:\n%s", formatErrors(errs))
	}
}
