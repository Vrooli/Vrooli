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
