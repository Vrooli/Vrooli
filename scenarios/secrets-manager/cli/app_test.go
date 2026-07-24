package main

import (
	"reflect"
	"testing"
)

// [REQ:SEC-CLI-001] Automation-friendly command normalization
func TestNormalizeArgs(t *testing.T) {
	app := &App{}
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{name: "config alias", in: []string{"config"}, want: []string{"configure"}},
		{name: "vault default status", in: []string{"vault", "--resource", "postgres"}, want: []string{"vault", "status", "--resource", "postgres"}},
		{name: "root vulnerabilities alias", in: []string{"vulnerabilities", "--severity", "high"}, want: []string{"security", "vulnerabilities", "--severity", "high"}},
		{name: "root scan alias", in: []string{"scan", "--component", "foo"}, want: []string{"security", "scan", "--component", "foo"}},
		{name: "root compliance alias", in: []string{"compliance"}, want: []string{"security", "compliance"}},
		{name: "root plan alias", in: []string{"plan", "--scenario", "picker-wheel"}, want: []string{"deployment", "plan", "--scenario", "picker-wheel"}},
		{name: "campaign default list", in: []string{"campaigns"}, want: []string{"campaigns", "list"}},
		{name: "scenario default list", in: []string{"scenario"}, want: []string{"scenarios", "list"}},
		{name: "override singular alias", in: []string{"override", "foo"}, want: []string{"overrides", "list", "foo"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := app.normalizeArgs(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("normalizeArgs(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
