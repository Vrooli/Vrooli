package phases

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
)

var commandLookup = exec.LookPath
var phaseCommandExecutor = runCommand
var phaseCommandCapture = runCommandCapture

func ensureDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("required directory missing: %s: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("expected directory but found file: %s", path)
	}
	return nil
}

func ensureFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("required file missing: %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("expected file but found directory: %s", path)
	}
	return nil
}

func EnsureCommandAvailable(name string) error {
	if _, err := commandLookup(name); err != nil {
		return fmt.Errorf("required command '%s' is not available: %w", name, err)
	}
	return nil
}

func ensureExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("required executable missing: %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("expected executable but found directory: %s", path)
	}
	if runtime.GOOS == "windows" {
		// Windows does not expose POSIX execute bits, so existence is enough.
		return nil
	}
	if info.Mode()&0o111 == 0 {
		return fmt.Errorf("file is not executable: %s", path)
	}
	return nil
}

func logPhaseStep(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	fmt.Fprintf(w, format+"\n", args...)
}

func logPhaseWarn(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	fmt.Fprintf(w, "WARN: "+format+"\n", args...)
}

func runCommand(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if logWriter == nil {
		logWriter = io.Discard
	}
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	return cmd.Run()
}

func runCommandCapture(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var output bytes.Buffer
	if logWriter != nil {
		cmd.Stdout = io.MultiWriter(logWriter, &output)
		cmd.Stderr = logWriter
	} else {
		cmd.Stdout = &output
		cmd.Stderr = io.Discard
	}
	err := cmd.Run()
	return output.String(), err
}

// OverrideCommandLookup temporarily replaces the binary lookup used by phases.
func OverrideCommandLookup(fn func(string) (string, error)) func() {
	prev := commandLookup
	commandLookup = fn
	return func() { commandLookup = prev }
}

// OverrideCommandExecutor temporarily replaces the command executor used by phases.
func OverrideCommandExecutor(fn func(context.Context, string, io.Writer, string, ...string) error) func() {
	prev := phaseCommandExecutor
	phaseCommandExecutor = fn
	return func() { phaseCommandExecutor = prev }
}

// OverrideCommandCapture temporarily replaces the capture executor used by phases.
func OverrideCommandCapture(fn func(context.Context, string, io.Writer, string, ...string) (string, error)) func() {
	prev := phaseCommandCapture
	phaseCommandCapture = fn
	return func() { phaseCommandCapture = prev }
}
