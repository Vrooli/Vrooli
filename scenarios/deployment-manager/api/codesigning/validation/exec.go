package validation

import (
	"bytes"
	"context"
	"os/exec"
)

// runCommand executes a command and returns stdout, stderr, and any error.
// This is a variable so it can be replaced in tests.
var runCommand = func(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

// lookupPath searches for an executable in the PATH.
// This is a variable so it can be replaced in tests.
var lookupPath = func(name string) (string, error) {
	return exec.LookPath(name)
}
