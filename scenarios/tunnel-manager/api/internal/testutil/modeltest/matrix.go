// Package modeltest contains test-only helpers for explicit workflow models.
package modeltest

import (
	"fmt"
	"strings"
)

// TestingT is the subset of *testing.T used by this package.
type TestingT interface {
	Helper()
	Fatalf(format string, args ...any)
}

// Transition applies event to state and returns the next state.
type Transition[S comparable, E comparable] func(state S, event E) (S, error)

// MatrixRow describes the expected result for one state/event pair.
type MatrixRow[S comparable, E comparable] struct {
	Name    string
	From    S
	Event   E
	To      S
	WantErr bool
}

type pair[S comparable, E comparable] struct {
	state S
	event E
}

// ValidateTransitionMatrix checks that rows cover every state/event pair
// exactly once and match the supplied production transition function.
func ValidateTransitionMatrix[S comparable, E comparable](
	statuses []S,
	events []E,
	rows []MatrixRow[S, E],
	transition Transition[S, E],
) []error {
	var errs []error
	errs = append(errs, validateMatrixInputs(statuses, events, transition)...)
	knownStatuses, statusErrs := collectKnownValues("status", statuses)
	errs = append(errs, statusErrs...)
	knownEvents, eventErrs := collectKnownValues("event", events)
	errs = append(errs, eventErrs...)
	seen, rowErrs := collectMatrixRows(rows, knownStatuses, knownEvents)
	errs = append(errs, rowErrs...)
	errs = append(errs, validatePairCoverage(statuses, events, seen)...)
	if len(errs) > 0 || transition == nil {
		return errs
	}
	return append(errs, validateTransitionRows(rows, transition)...)
}

func validateMatrixInputs[S comparable, E comparable](statuses []S, events []E, transition Transition[S, E]) []error {
	var errs []error
	if len(statuses) == 0 {
		errs = append(errs, fmt.Errorf("statuses must not be empty"))
	}
	if len(events) == 0 {
		errs = append(errs, fmt.Errorf("events must not be empty"))
	}
	if transition == nil {
		errs = append(errs, fmt.Errorf("transition must not be nil"))
	}
	return errs
}

func collectKnownValues[T comparable](kind string, values []T) (map[T]struct{}, []error) {
	errs := []error{}
	known := make(map[T]struct{}, len(values))
	for _, value := range values {
		if _, ok := known[value]; ok {
			errs = append(errs, fmt.Errorf("duplicate %s %v", kind, value))
			continue
		}
		known[value] = struct{}{}
	}
	return known, errs
}

func collectMatrixRows[S comparable, E comparable](
	rows []MatrixRow[S, E],
	knownStatuses map[S]struct{},
	knownEvents map[E]struct{},
) (map[pair[S, E]]string, []error) {
	var errs []error
	seen := make(map[pair[S, E]]string, len(rows))
	for i, row := range rows {
		label := row.Name
		if label == "" {
			label = fmt.Sprintf("row %d", i)
		}
		if _, ok := knownStatuses[row.From]; !ok {
			errs = append(errs, fmt.Errorf("%s: unknown from status %v", label, row.From))
		}
		if _, ok := knownStatuses[row.To]; !ok {
			errs = append(errs, fmt.Errorf("%s: unknown to status %v", label, row.To))
		}
		if _, ok := knownEvents[row.Event]; !ok {
			errs = append(errs, fmt.Errorf("%s: unknown event %v", label, row.Event))
		}

		key := pair[S, E]{state: row.From, event: row.Event}
		if first, ok := seen[key]; ok {
			errs = append(errs, fmt.Errorf("%s: duplicate pair %v/%v already covered by %s", label, row.From, row.Event, first))
			continue
		}
		seen[key] = label
	}
	return seen, errs
}

func validatePairCoverage[S comparable, E comparable](
	statuses []S,
	events []E,
	seen map[pair[S, E]]string,
) []error {
	var errs []error
	for _, status := range statuses {
		for _, event := range events {
			if _, ok := seen[pair[S, E]{state: status, event: event}]; !ok {
				errs = append(errs, fmt.Errorf("missing pair %v/%v", status, event))
			}
		}
	}
	return errs
}

func validateTransitionRows[S comparable, E comparable](
	rows []MatrixRow[S, E],
	transition Transition[S, E],
) []error {
	var errs []error
	for i, row := range rows {
		label := row.Name
		if label == "" {
			label = fmt.Sprintf("row %d", i)
		}
		got, err := transition(row.From, row.Event)
		if row.WantErr {
			if err == nil {
				errs = append(errs, fmt.Errorf("%s: expected error, got nil", label))
			}
		} else if err != nil {
			errs = append(errs, fmt.Errorf("%s: unexpected error: %v", label, err))
		}
		if got != row.To {
			errs = append(errs, fmt.Errorf("%s: got status %v, want %v", label, got, row.To))
		}
	}
	return errs
}

// AssertTransitionMatrix fails t when ValidateTransitionMatrix finds drift.
func AssertTransitionMatrix[S comparable, E comparable](
	t TestingT,
	statuses []S,
	events []E,
	rows []MatrixRow[S, E],
	transition Transition[S, E],
) {
	t.Helper()
	if errs := ValidateTransitionMatrix(statuses, events, rows, transition); len(errs) > 0 {
		t.Fatalf("transition matrix mismatch:\n%s", formatErrors(errs))
	}
}

func formatErrors(errs []error) string {
	lines := make([]string, 0, len(errs))
	for _, err := range errs {
		lines = append(lines, "  - "+err.Error())
	}
	return strings.Join(lines, "\n")
}
