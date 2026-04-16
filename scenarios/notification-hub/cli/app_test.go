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
		{name: "config alias", args: []string{"config", "api_base", "http://localhost:1"}, want: []string{"configure", "api_base", "http://localhost:1"}},
		{name: "profiles default list", args: []string{"profiles"}, want: []string{"profiles", "list"}},
		{name: "analytics default delivery stats", args: []string{"analytics"}, want: []string{"analytics", "delivery-stats"}},
		{name: "notifications explicit subcommand", args: []string{"notifications", "send"}, want: []string{"notifications", "send"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := app.normalizeArgs(tt.args)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("normalizeArgs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
