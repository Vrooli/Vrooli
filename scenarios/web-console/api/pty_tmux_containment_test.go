package main

import (
	"bytes"
	"strings"
	"testing"

	platform "github.com/vrooli/platform-go"
)

// [REQ:STORM-002] The console's tmux server is born under the agent slice
// with the same ceilings a coding-agent session carries.
func TestWebConsoleScopeCarriesCeiling(t *testing.T) {
	t.Setenv("WC_TMUX_SCOPE_NAME", "wc-test-scope")
	var output bytes.Buffer
	spec := tmuxContainedSpec("wc-sock", []string{"new-session", "-d", "-s", "wc-x"}, []string{"HOME=/h"}, &output)
	if spec.Scope != "wc-test-scope" || spec.Path != "tmux" || spec.Args[0] != "-S" || spec.Args[1] != "wc-sock" {
		t.Fatalf("spec = %+v", spec)
	}
	if spec.Containment.Slice != "vrooli-agents.slice" || spec.Containment.TasksMax != 4096 || spec.Containment.CPUWeight != 50 {
		t.Fatalf("containment = %+v", spec.Containment)
	}
	contained, err := platform.ContainedCommand(spec)
	if err != nil {
		t.Fatalf("ContainedCommand: %v", err)
	}
	argv := strings.Join(contained.Cmd.Args, " ")
	if contained.Method == platform.MethodSystemdRun {
		for _, want := range []string{"--user", "--scope", "--unit=wc-test-scope", "--slice=vrooli-agents.slice", "TasksMax=4096", "CPUWeight=50", "-- tmux -S wc-sock new-session"} {
			if !strings.Contains(argv, want) {
				t.Fatalf("argv %q lacks %q", argv, want)
			}
		}
	}
}
