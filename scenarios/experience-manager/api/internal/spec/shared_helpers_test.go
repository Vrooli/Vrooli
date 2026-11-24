package spec

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestParamStringSelectsFirstStringAndTrimsIt(t *testing.T) {
	params := map[string]any{"wrong": 7, "selected": "  value  ", "later": "ignored"}
	if got := ParamString(params, "wrong", "selected", "later"); got != "value" {
		t.Fatalf("ParamString = %q, want %q", got, "value")
	}
	if got := ParamString(map[string]any{"empty": "  ", "later": "value"}, "empty", "later"); got != "" {
		t.Fatalf("ParamString empty first value = %q, want empty", got)
	}
}

func TestNumericParamSupportsContractJSONNumberTypesAndContinuesPastInvalid(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value any
		want  float64
	}{
		{name: "float64", value: float64(1.25), want: 1.25},
		{name: "float32", value: float32(2.5), want: 2.5},
		{name: "int", value: int(3), want: 3},
		{name: "int64", value: int64(4), want: 4},
		{name: "json number", value: json.Number("5.5"), want: 5.5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := NumericParam(map[string]any{"n": tc.value}, "n")
			if !ok || got != tc.want {
				t.Fatalf("NumericParam = (%v, %v), want (%v, true)", got, ok, tc.want)
			}
		})
	}
	params := map[string]any{"bad": json.Number("nope"), "later": int64(8)}
	if got, ok := NumericParam(params, "bad", "later"); !ok || got != 8 {
		t.Fatalf("NumericParam after invalid JSON number = (%v, %v), want (8, true)", got, ok)
	}
	if got, ok := NumericParam(map[string]any{"n": int32(9)}, "n"); ok || got != 0 {
		t.Fatalf("NumericParam unsupported type = (%v, %v), want (0, false)", got, ok)
	}
}

func TestFindingHelpersPreservePrimaryLocationAndCanonicalOrder(t *testing.T) {
	if got := FirstLocation(nil); got != "" {
		t.Fatalf("FirstLocation(nil) = %q, want empty", got)
	}
	if got := FirstLocation([]string{"primary", "secondary"}); got != "primary" {
		t.Fatalf("FirstLocation = %q, want primary", got)
	}
	findings := []Finding{
		{Code: "b", Locations: []string{"z"}},
		{Code: "a", Locations: []string{"z"}},
		{Code: "a", Locations: []string{"a", "b"}},
	}
	SortFindings(findings)
	want := []Finding{
		{Code: "a", Locations: []string{"a", "b"}},
		{Code: "a", Locations: []string{"z"}},
		{Code: "b", Locations: []string{"z"}},
	}
	if !reflect.DeepEqual(findings, want) {
		t.Fatalf("SortFindings = %#v, want %#v", findings, want)
	}
}
