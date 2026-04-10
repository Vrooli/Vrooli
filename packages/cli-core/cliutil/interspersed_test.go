package cliutil

import (
	"flag"
	"testing"
)

func TestParseInterspersed(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(fs *flag.FlagSet) (strFlags map[string]*string, boolFlags map[string]*bool)
		args     []string
		wantStr  map[string]string
		wantBool map[string]bool
		wantArgs []string
		wantErr  bool
	}{
		{
			name: "flags before positional (baseline)",
			setup: func(fs *flag.FlagSet) (map[string]*string, map[string]*bool) {
				s := map[string]*string{"status": fs.String("status", "", "status")}
				return s, nil
			},
			args:     []string{"--status", "pending", "my-id"},
			wantStr:  map[string]string{"status": "pending"},
			wantArgs: []string{"my-id"},
		},
		{
			name: "flags after positional (the bug this fixes)",
			setup: func(fs *flag.FlagSet) (map[string]*string, map[string]*bool) {
				s := map[string]*string{"status": fs.String("status", "", "status")}
				return s, nil
			},
			args:     []string{"my-id", "--status", "pending"},
			wantStr:  map[string]string{"status": "pending"},
			wantArgs: []string{"my-id"},
		},
		{
			name: "mixed ordering",
			setup: func(fs *flag.FlagSet) (map[string]*string, map[string]*bool) {
				s := map[string]*string{
					"status": fs.String("status", "", "status"),
					"phase":  fs.String("phase", "", "phase"),
				}
				return s, nil
			},
			args:     []string{"--status", "active", "my-id", "--phase", "build"},
			wantStr:  map[string]string{"status": "active", "phase": "build"},
			wantArgs: []string{"my-id"},
		},
		{
			name: "double dash terminator",
			setup: func(fs *flag.FlagSet) (map[string]*string, map[string]*bool) {
				s := map[string]*string{"status": fs.String("status", "", "status")}
				return s, nil
			},
			args:     []string{"--status", "pending", "--", "--not-a-flag", "pos"},
			wantStr:  map[string]string{"status": "pending"},
			wantArgs: []string{"--not-a-flag", "pos"},
		},
		{
			name: "inline value syntax",
			setup: func(fs *flag.FlagSet) (map[string]*string, map[string]*bool) {
				s := map[string]*string{"status": fs.String("status", "", "status")}
				return s, nil
			},
			args:     []string{"my-id", "--status=pending"},
			wantStr:  map[string]string{"status": "pending"},
			wantArgs: []string{"my-id"},
		},
		{
			name: "bool flags (no value consumed)",
			setup: func(fs *flag.FlagSet) (map[string]*string, map[string]*bool) {
				s := map[string]*string{"status": fs.String("status", "", "status")}
				b := map[string]*bool{"json": fs.Bool("json", false, "json output")}
				return s, b
			},
			args:     []string{"my-id", "--json", "--status", "pending"},
			wantStr:  map[string]string{"status": "pending"},
			wantBool: map[string]bool{"json": true},
			wantArgs: []string{"my-id"},
		},
		{
			name: "unknown flags treated as positional",
			setup: func(fs *flag.FlagSet) (map[string]*string, map[string]*bool) {
				s := map[string]*string{"status": fs.String("status", "", "status")}
				return s, nil
			},
			// Unknown flags: reorder treats them as value flags (consumes next arg).
			// The FlagSet with ContinueOnError will return an error for unknown flags.
			args:    []string{"--unknown", "val", "--status", "ok"},
			wantErr: true,
		},
		{
			name: "empty args",
			setup: func(fs *flag.FlagSet) (map[string]*string, map[string]*bool) {
				s := map[string]*string{"status": fs.String("status", "", "status")}
				return s, nil
			},
			args:     []string{},
			wantStr:  map[string]string{"status": ""},
			wantArgs: []string{},
		},
		{
			name: "multiple positional args interspersed with flags",
			setup: func(fs *flag.FlagSet) (map[string]*string, map[string]*bool) {
				s := map[string]*string{"priority": fs.String("priority", "medium", "priority")}
				b := map[string]*bool{"json": fs.Bool("json", false, "json output")}
				return s, b
			},
			args:     []string{"scenario", "my-app", "--priority", "high", "--json"},
			wantStr:  map[string]string{"priority": "high"},
			wantBool: map[string]bool{"json": true},
			wantArgs: []string{"scenario", "my-app"},
		},
		{
			name: "single dash is positional",
			setup: func(fs *flag.FlagSet) (map[string]*string, map[string]*bool) {
				s := map[string]*string{"out": fs.String("out", "", "output")}
				return s, nil
			},
			args:     []string{"-", "--out", "file.txt"},
			wantStr:  map[string]string{"out": "file.txt"},
			wantArgs: []string{"-"},
		},
		{
			name: "short flag syntax",
			setup: func(fs *flag.FlagSet) (map[string]*string, map[string]*bool) {
				s := map[string]*string{"p": fs.String("p", "medium", "priority")}
				return s, nil
			},
			args:     []string{"my-id", "-p", "high"},
			wantStr:  map[string]string{"p": "high"},
			wantArgs: []string{"my-id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := flag.NewFlagSet(tt.name, flag.ContinueOnError)
			strFlags, boolFlags := tt.setup(fs)

			err := ParseInterspersed(fs, tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for k, want := range tt.wantStr {
				if got := *strFlags[k]; got != want {
					t.Errorf("flag %q = %q, want %q", k, got, want)
				}
			}
			for k, want := range tt.wantBool {
				if got := *boolFlags[k]; got != want {
					t.Errorf("flag %q = %v, want %v", k, got, want)
				}
			}

			gotArgs := fs.Args()
			if len(gotArgs) != len(tt.wantArgs) {
				t.Fatalf("positional args = %v, want %v", gotArgs, tt.wantArgs)
			}
			for i, want := range tt.wantArgs {
				if gotArgs[i] != want {
					t.Errorf("arg[%d] = %q, want %q", i, gotArgs[i], want)
				}
			}
		})
	}
}
