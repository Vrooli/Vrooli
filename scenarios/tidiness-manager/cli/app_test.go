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
		{name: "config alias", in: []string{"config"}, want: []string{"configure"}},
		{name: "campaign default list", in: []string{"campaigns"}, want: []string{"campaigns", "list"}},
		{name: "issues legacy list positional", in: []string{"issues", "my-scenario"}, want: []string{"issues", "list", "my-scenario"}},
		{name: "issues explicit subcommand", in: []string{"issues", "resolve", "12"}, want: []string{"issues", "resolve", "12"}},
		{name: "visit alias", in: []string{"visit", "api/main.go", "--scenario", "foo"}, want: []string{"tracking", "visit", "api/main.go", "--scenario", "foo"}},
		{name: "campaign note alias", in: []string{"campaign-note", "--scenario", "foo"}, want: []string{"tracking", "campaign-note", "--scenario", "foo"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := app.normalizeArgs(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("normalizeArgs(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
