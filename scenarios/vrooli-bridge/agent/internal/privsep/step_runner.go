package privsep

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
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
	if len(argv) == 0 {
		return startFailureExitCode, errors.New("empty argv")
	}

	// #nosec G204 — argv is a typed, validated token list (Steps rejects shell
	// metacharacters); the binaries are operator-configured. This is the
	// privileged provisioning path, not a shell.
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = dir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return startFailureExitCode, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return startFailureExitCode, err
	}

	if err := cmd.Start(); err != nil {
		return startFailureExitCode, err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go streamLines(&wg, stdout, onLog)
	go streamLines(&wg, stderr, onLog)
	wg.Wait()

	err = cmd.Wait()
	if err == nil {
		return 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), nil
	}
	return startFailureExitCode, err
}

func streamLines(wg *sync.WaitGroup, r io.Reader, onLog func(string)) {
	defer wg.Done()
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for scanner.Scan() {
		onLog(scanner.Text() + "\n")
	}
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
