package main

import (
	"reflect"
	"testing"
)

func TestNormalizeArgs(t *testing.T) {
	app := &App{}
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{name: "legacy list", args: []string{"list"}, want: []string{"funnels", "list"}},
		{name: "legacy create", args: []string{"create", "--name", "Demo"}, want: []string{"funnels", "create", "--name", "Demo"}},
		{name: "templates default to list", args: []string{"templates"}, want: []string{"templates", "list"}},
		{name: "projects default to list", args: []string{"projects"}, want: []string{"projects", "list"}},
		{name: "config alias", args: []string{"config", "api_base", "http://localhost:15000"}, want: []string{"configure", "api_base", "http://localhost:15000"}},
		{name: "trim blanks", args: []string{" ", "funnels", "", "get", "abc"}, want: []string{"funnels", "get", "abc"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := app.normalizeArgs(tt.args)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("normalizeArgs(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}
