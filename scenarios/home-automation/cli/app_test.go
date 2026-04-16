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
		{name: "devices default list", in: []string{"devices"}, want: []string{"devices", "list"}},
		{name: "profiles default list", in: []string{"profiles", "--json"}, want: []string{"profiles", "list", "--json"}},
		{name: "scenes alias contexts", in: []string{"scenes", "activate", "evening"}, want: []string{"contexts", "activate", "evening"}},
		{name: "ha alias", in: []string{"ha"}, want: []string{"home-assistant", "status"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := app.normalizeArgs(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("normalizeArgs(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
