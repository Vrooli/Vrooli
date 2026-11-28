package support

import (
	"reflect"
	"testing"
)

func TestRenderGeneric(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  []string
	}{
		{
			name:  "sorted map",
			value: map[string]interface{}{"zeta": "last", "alpha": "first"},
			want:  []string{"alpha: first", "zeta: last"},
		},
		{
			name:  "empty map",
			value: map[string]interface{}{},
			want:  []string{"(empty payload)"},
		},
		{
			name:  "array",
			value: []interface{}{"hello", float64(2), nil},
			want:  []string{"0: hello", "1: 2", "2: null"},
		},
		{
			name:  "empty array",
			value: []interface{}{},
			want:  []string{"(empty list)"},
		},
		{
			name:  "nil",
			value: nil,
			want:  []string{"(empty payload)"},
		},
		{
			name:  "scalar",
			value: "ready",
			want:  []string{"ready"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RenderGeneric(tt.value); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RenderGeneric(%#v) = %#v, want %#v", tt.value, got, tt.want)
			}
		})
	}
}
