package clioutput

import (
	"reflect"
	"testing"
)

func TestMapRowsSortsKeysAndRendersValues(t *testing.T) {
	got := MapRows(map[string]interface{}{
		"zeta":  "ready",
		"alpha": float64(3),
	})
	want := []string{"alpha: 3", "zeta: ready"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MapRows() = %v, want %v", got, want)
	}
}

func TestMapRowsUsesEmptyPayloadForNoData(t *testing.T) {
	if got := MapRows(nil); !reflect.DeepEqual(got, []string{"(empty payload)"}) {
		t.Fatalf("MapRows(nil) = %v", got)
	}
}

func TestRenderValuePreservesDisplayForms(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  string
	}{
		{name: "nil", value: nil, want: "null"},
		{name: "string stays unquoted", value: "hello", want: "hello"},
		{name: "boolean", value: true, want: "true"},
		{name: "integral JSON number", value: float64(4), want: "4"},
		{name: "fractional JSON number", value: float64(1.25), want: "1.25"},
		{name: "nested JSON", value: map[string]interface{}{"b": true, "a": "x"}, want: `{"a":"x","b":true}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := RenderValue(test.value); got != test.want {
				t.Fatalf("RenderValue(%#v) = %q, want %q", test.value, got, test.want)
			}
		})
	}
}
