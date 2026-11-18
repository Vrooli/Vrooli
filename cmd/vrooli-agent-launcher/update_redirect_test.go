package main

import (
	"reflect"
	"testing"
)

func TestManagedUpdateCommand(t *testing.T) {
	cases := []struct {
		name   string
		runner string
		args   []string
		want   []string
		ok     bool
	}{
		{"codex bare update", "codex", []string{"update"}, []string{"vrooli", "resource", "update", "codex"}, true},
		{"codex other", "codex", []string{"update", "--force"}, nil, false},
		{"opencode update", "opencode", []string{"update"}, []string{"vrooli", "resource", "update", "opencode"}, true},
		{"opencode upgrade", "opencode", []string{"upgrade"}, []string{"vrooli", "resource", "update", "opencode"}, true},
		{"opencode major", "opencode", []string{"upgrade", "--major"}, []string{"vrooli", "resource", "update", "opencode", "--channel", "v2"}, true},
		{"opencode pinned version", "opencode", []string{"upgrade", "2.3.2"}, nil, false},
		{"opencode help", "opencode", []string{"upgrade", "--help"}, nil, false},
		{"unmanaged runner", "claude", []string{"update"}, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := managedUpdateCommand(tc.runner, tc.args)
			if ok != tc.ok || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("managedUpdateCommand(%q, %q) = (%q, %v), want (%q, %v)", tc.runner, tc.args, got, ok, tc.want, tc.ok)
			}
		})
	}
}
