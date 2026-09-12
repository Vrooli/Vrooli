package main

import (
	"reflect"
	"testing"
)

func TestNormalizeArgs(t *testing.T) {
	app := &App{}
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{name: "rules default list", in: []string{"rules"}, want: []string{"rules", "list"}},
		{name: "scenarios default list", in: []string{"scenarios"}, want: []string{"scenarios", "list"}},
		{name: "audit alias", in: []string{"audit", "--scenario", "foo"}, want: []string{"run", "--scenario", "foo"}},
		{name: "check alias", in: []string{"check", "foo"}, want: []string{"run", "foo"}},
		{name: "apply fixes alias", in: []string{"apply-fixes", "foo"}, want: []string{"fix", "foo"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := app.normalizeArgs(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("normalizeArgs(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
