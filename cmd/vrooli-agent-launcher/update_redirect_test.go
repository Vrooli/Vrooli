package main

import (
	"reflect"
	"testing"
)

func TestManagedUpdateCommand(t *testing.T) {
	tests := []struct {
		name   string
		runner string
		args   []string
		want   []string
	}{
		{"codex update is redirected", "codex", []string{"update"}, []string{"vrooli", "resource", "update", "codex"}},
		{"update with extra arguments reaches the agent", "codex", []string{"update", "--help"}, nil},
		{"other subcommands reach the agent", "codex", []string{"exec", "update"}, nil},
		{"no arguments reach the agent", "codex", nil, nil},
		{"agents without a redirect are untouched", "claude", []string{"update"}, nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := managedUpdateCommand(test.runner, test.args)
			if ok != (test.want != nil) || !reflect.DeepEqual(got, test.want) {
				t.Fatalf("managedUpdateCommand(%q, %q) = %q, %v; want %q", test.runner, test.args, got, ok, test.want)
			}
		})
	}
}
