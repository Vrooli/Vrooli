package privsep

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// osStepRunner is the production StepRunner. Like the runner's command executor
// it runs the argv via os/exec.CommandContext — the binary and its arguments as
// a pre-split slice, exec'd directly, NEVER through a shell — so there is no
// metacharacter interpretation and no command-injection surface. It streams
// combined stdout+stderr to onLog line-by-line and returns the process exit
// code. This is the PRIVILEGED execution path; in deployment the process runs as
// the dedicated provisioning principal (package service), structurally distinct
// from the non-privileged runner.
type osStepRunner struct{}

var _ StepRunner = osStepRunner{}

func (osStepRunner) Run(ctx context.Context, argv []string, dir string, onLog func(string)) (int, error) {
	return osStepRunner{}.RunWithInput(ctx, argv, dir, nil, onLog)
}

func (osStepRunner) RunWithEnvironment(ctx context.Context, argv []string, dir string, env []string, onLog func(string)) (int, error) {
	return osStepRunner{}.RunWithInputEnvironment(ctx, argv, dir, nil, env, onLog)
}

// RunWithInput is the only privileged helper path that accepts secret input.
// The input is connected to the child stdin, never interpolated into argv or a
// shell string. Callers must zero their own input buffer after this returns.
func (osStepRunner) RunWithInput(ctx context.Context, argv []string, dir string, input []byte, onLog func(string)) (int, error) {
	return osStepRunner{}.RunWithInputEnvironment(ctx, argv, dir, input, nil, onLog)
}

func (osStepRunner) RunWithInputEnvironment(ctx context.Context, argv []string, dir string, input []byte, env []string, onLog func(string)) (int, error) {
	if len(argv) == 0 {
		return startFailureExitCode, errors.New("empty argv")
	}

	// #nosec G204 — argv is a typed, validated token list (Steps rejects shell
	// metacharacters); the binaries are operator-configured. This is the
	// privileged provisioning path, not a shell.
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = dir
	if len(env) > 0 {
		cmd.Env = mergeEnvironment(os.Environ(), env)
	}
	if input != nil {
		cmd.Stdin = bytes.NewReader(input)
	}

	stdout, err := os.CreateTemp("", "vrooli-bridge-step-stdout-*")
	if err != nil {
		return startFailureExitCode, fmt.Errorf("create stdout capture: %w", err)
	}
	stdoutPath := stdout.Name()
	defer os.Remove(stdoutPath)
	defer stdout.Close()
	stderr, err := os.CreateTemp("", "vrooli-bridge-step-stderr-*")
	if err != nil {
		return startFailureExitCode, fmt.Errorf("create stderr capture: %w", err)
	}
	stderrPath := stderr.Name()
	defer os.Remove(stderrPath)
	defer stderr.Close()
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return startFailureExitCode, err
	}

	// stdout/stderr are regular files rather than pipes. A descendant that
	// inherits a pipe descriptor can keep the old streaming implementation open
	// after the command itself exits; regular files let Wait observe the actual
	// command lifecycle. Replay both complete streams below so JSON responses are
	// never truncated by racing Wait against pipe readers. The existing runner
	// contract does not promise cross-stream ordering, and the previous two-pipe
	// implementation did not provide it either.
	err = cmd.Wait()
	if closeErr := stdout.Close(); err == nil && closeErr != nil {
		err = closeErr
	}
	if closeErr := stderr.Close(); err == nil && closeErr != nil {
		err = closeErr
	}
	if replayErr := replayLogFile(stdoutPath, onLog); err == nil && replayErr != nil {
		err = replayErr
	}
	if replayErr := replayLogFile(stderrPath, onLog); err == nil && replayErr != nil {
		err = replayErr
	}
	if err == nil {
		return 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), nil
	}
	return startFailureExitCode, err
}

// mergeEnvironment replaces inherited variables instead of appending duplicate
// keys. POSIX environments technically permit duplicates, but lookup behavior
// differs across runtimes and launch contexts; cleanup relies on HOME,
// VROOLI_ROOT, and its service-deferral policy being the exact helper values.
func mergeEnvironment(base, overrides []string) []string {
	merged := append([]string(nil), base...)
	positions := make(map[string]int, len(merged))
	for i, entry := range merged {
		key, _, ok := strings.Cut(entry, "=")
		if ok {
			positions[key] = i
		}
	}
	for _, entry := range overrides {
		key, _, ok := strings.Cut(entry, "=")
		if !ok || key == "" {
			continue
		}
		if i, found := positions[key]; found {
			merged[i] = entry
			continue
		}
		positions[key] = len(merged)
		merged = append(merged, entry)
	}
	return merged
}

func streamLines(wg *sync.WaitGroup, r io.Reader, onLog func(string)) {
	defer wg.Done()
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for scanner.Scan() {
		onLog(scanner.Text() + "\n")
	}
}

func replayLogFile(path string, onLog func(string)) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	wg := sync.WaitGroup{}
	wg.Add(1)
	streamLines(&wg, file, onLog)
	wg.Wait()
	return nil
}

// osRevisionResolver reports the node's current revision via
// `git rev-parse HEAD`, run directly (no shell).
type osRevisionResolver struct {
	gitBin string
}

var _ RevisionResolver = osRevisionResolver{}

func (r osRevisionResolver) Current(ctx context.Context, dir string) (string, error) {
	bin := r.gitBin
	if strings.TrimSpace(bin) == "" {
		bin = "git"
	}
	// #nosec G204 — fixed argv, operator-configured git binary, no job input.
	cmd := exec.CommandContext(ctx, bin, "rev-parse", "HEAD")
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}
