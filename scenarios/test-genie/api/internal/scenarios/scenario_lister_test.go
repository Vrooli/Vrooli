package scenarios

import (
	"context"
	"os/exec"
	"reflect"
	"testing"
)

func TestVrooliScenarioListerUsesNoStaleCheck(t *testing.T) {
	original := execCommandContext
	t.Cleanup(func() {
		execCommandContext = original
	})

	var gotName string
	var gotArgs []string
	execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		gotName = name
		gotArgs = append([]string(nil), args...)
		return exec.CommandContext(ctx, "bash", "-lc", "printf '%s' '{\"success\":true,\"scenarios\":[]}'")
	}

	lister := NewVrooliScenarioLister()
	_, err := lister.ListScenarios(context.Background())
	if err != nil {
		t.Fatalf("ListScenarios: %v", err)
	}

	if gotName != "vrooli" {
		t.Fatalf("command = %q, want %q", gotName, "vrooli")
	}
	wantArgs := []string{"--no-stale-check", "scenario", "list", "--json"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("args = %v, want %v", gotArgs, wantArgs)
	}
}
