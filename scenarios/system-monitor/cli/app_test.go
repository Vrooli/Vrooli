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
		{name: "empty", in: nil, want: []string{}},
		{name: "config alias", in: []string{"config", "api_base", "http://localhost:1"}, want: []string{"configure", "api_base", "http://localhost:1"}},
		{name: "metrics default", in: []string{"metrics"}, want: []string{"metrics", "current"}},
		{name: "report alias", in: []string{"report", "weekly"}, want: []string{"reports", "generate", "weekly"}},
		{name: "investigate alias", in: []string{"investigate"}, want: []string{"investigations", "latest"}},
		{name: "reports default", in: []string{"reports"}, want: []string{"reports", "list"}},
		{name: "settings default", in: []string{"settings"}, want: []string{"settings", "get"}},
		{name: "trim blanks", in: []string{" metrics ", "", " "}, want: []string{"metrics", "current"}},
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
