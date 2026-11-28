package modeltest

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

// ProductObservation is a lossless named projection of production-owned
// state plus the effects observed at the same boundary. The verifier
// adapter compares complete maps so an omitted row or effect is not
// mistaken for an empty value.
type ProductObservation struct {
	State   map[string]string `json:"state"`
	Effects map[string]int64  `json:"effects"`
}

// AssertProductObservation binds a generated finite product tuple and its
// declared effects to an owner-provided observation from production.
func AssertProductObservation(t testing.TB, label string, want, got ProductObservation) {
	t.Helper()
	for _, err := range ValidateProductObservation(want, got) {
		t.Errorf("%s: %v", label, err)
	}
}

// ValidateProductObservation returns complete tuple/effect mismatches without
// accepting omitted keys or zero-filled values.
func ValidateProductObservation(want, got ProductObservation) []error {
	var errs []error
	if !reflect.DeepEqual(want.State, got.State) {
		errs = append(errs, fmt.Errorf("state mismatch: got %v want %v", got.State, want.State))
	}
	if !reflect.DeepEqual(want.Effects, got.Effects) {
		errs = append(errs, fmt.Errorf("effects mismatch: got %v want %v", got.Effects, want.Effects))
	}
	return errs
}

// AssertProductTraceObservations checks each ordered production observation
// against the generated trace contract, including exact trace length.
func AssertProductTraceObservations(t testing.TB, trace string, want, got []ProductObservation) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s observation count=%d want=%d", trace, len(got), len(want))
	}
	for i := range want {
		AssertProductObservation(t, fmt.Sprintf("%s step %d", trace, i), want[i], got[i])
	}
}

// ProductObservationKeys returns stable names for diagnostics and adapters.
func ProductObservationKeys(observation ProductObservation) []string {
	keys := make([]string, 0, len(observation.State)+len(observation.Effects))
	for key := range observation.State {
		keys = append(keys, "state."+key)
	}
	for key := range observation.Effects {
		keys = append(keys, "effects."+key)
	}
	sort.Strings(keys)
	return keys
}
