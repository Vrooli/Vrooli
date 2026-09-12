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
		{name: "config alias", in: []string{"config", "api_base", "http://localhost:1"}, want: []string{"configure", "api_base", "http://localhost:1"}},
		{name: "approved dependency alias", in: []string{"deps", "approved", "list"}, want: []string{"deps-approved", "list"}},
		{name: "dependencies alias", in: []string{"dependencies", "demo"}, want: []string{"list", "demo"}},
		{name: "scenario alias", in: []string{"scenario", "demo"}, want: []string{"scenarios", "get", "demo"}},
		{name: "bundle alias", in: []string{"bundle-manifest", "demo"}, want: []string{"bundle", "manifest", "demo"}},
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
