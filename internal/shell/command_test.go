package shell

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestOSRunnerLookPath(t *testing.T) {
	runner := OSRunner{}
	path, err := runner.LookPath("go")
	if err != nil || path == "" {
		t.Fatalf("LookPath(go) = %q, %v", path, err)
	}

	if _, err := runner.LookPath("vrooli-command-that-does-not-exist"); !errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("LookPath(missing) error = %v, want exec.ErrNotFound", err)
	}
}

func TestOSRunnerRun(t *testing.T) {
	runner := OSRunner{}

	output, err := runner.Run(context.Background(), "sh", "-c", "printf success")
	if err != nil || string(output) != "success" {
		t.Fatalf("Run(success) = %q, %v", output, err)
	}

	output, err = runner.Run(context.Background(), "sh", "-c", "printf failure >&2; exit 7")
	if err == nil || !strings.Contains(string(output), "failure") {
		t.Fatalf("Run(failure) = %q, %v; want non-zero error and stderr", output, err)
	}
}

func TestCommandConstructorsPreserveInvocation(t *testing.T) {
	plain := NewCommand("tool", "one", "two")
	if plain.Path != "tool" || strings.Join(plain.Args, " ") != "tool one two" {
		t.Fatalf("NewCommand = path %q args %#v", plain.Path, plain.Args)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	contextual := NewCommandContext(ctx, "tool", "three")
	if contextual.Path != "tool" || strings.Join(contextual.Args, " ") != "tool three" {
		t.Fatalf("NewCommandContext = path %q args %#v", contextual.Path, contextual.Args)
	}
}
