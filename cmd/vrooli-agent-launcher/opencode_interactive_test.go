package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeVersionScript(t *testing.T, output string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "opencode")
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho '"+output+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAdaptOpenCodeInteractive(t *testing.T) {
	v2 := writeVersionScript(t, "opencode v2.3.2")
	v1 := writeVersionScript(t, "opencode v1.19.5")
	cases := []struct {
		name        string
		path        string
		args        []string
		env         []string
		wantArgs    []string
		wantContent string
	}{
		{
			name: "v2 interactive model moves to config content",
			path: v2, args: []string{"-m", "opencode-go/mimo-v2.5"},
			wantArgs: []string{"--standalone"}, wantContent: `{"model":"opencode-go/mimo-v2.5"}`,
		},
		{
			name: "v2 long model equals form and existing provider block",
			path: v2, args: []string{"--model=opencode-go/mimo-v2.5"},
			env:      []string{`OPENCODE_CONFIG_CONTENT={"provider":{"vrooli-source":{"name":"x"}}}`},
			wantArgs: []string{"--standalone"},
			wantContent: `{"model":"opencode-go/mimo-v2.5","provider":{"vrooli-source":{"name":"x"}}}`,
		},
		{
			name: "v2 explicit server is respected",
			path: v2, args: []string{"--server", "http://127.0.0.1:1", "-m", "opencode-go/mimo-v2.5"},
			wantArgs: []string{"--server", "http://127.0.0.1:1"}, wantContent: `{"model":"opencode-go/mimo-v2.5"}`,
		},
		{
			name: "v2 run is non-interactive",
			path: v2, args: []string{"run", "hi", "-m", "opencode-go/mimo-v2.5"},
			wantArgs: []string{"run", "hi", "-m", "opencode-go/mimo-v2.5"},
		},
		{
			name: "v2 informational invocation is unchanged",
			path: v2, args: []string{"--version"},
			wantArgs: []string{"--version"},
		},
		{
			name: "v1 keeps the native model flag",
			path: v1, args: []string{"-m", "opencode-go/mimo-v2.5"},
			wantArgs: []string{"-m", "opencode-go/mimo-v2.5"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotArgs, gotEnv := adaptOpenCodeInteractive("opencode", tc.path, tc.args, tc.env)
			if !reflect.DeepEqual(gotArgs, tc.wantArgs) {
				t.Fatalf("args = %q, want %q", gotArgs, tc.wantArgs)
			}
			content := environmentValue(gotEnv, "OPENCODE_CONFIG_CONTENT")
			if content != tc.wantContent {
				t.Fatalf("OPENCODE_CONFIG_CONTENT = %q, want %q", content, tc.wantContent)
			}
		})
	}
}

func TestAdaptOpenCodeInteractiveIgnoresOtherAgents(t *testing.T) {
	v2 := writeVersionScript(t, "opencode v2.3.2")
	args, env := adaptOpenCodeInteractive("codex", v2, []string{"-m", "x"}, nil)
	if len(args) != 2 || args[0] != "-m" || len(env) != 0 {
		t.Fatalf("non-opencode launch was changed: args=%q env=%q", args, env)
	}
}
