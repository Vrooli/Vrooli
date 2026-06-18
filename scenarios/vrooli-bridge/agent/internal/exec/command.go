package exec

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os/exec"
	"sync"
)

// osCommandRunner is the production CommandRunner. It runs the argv via
// os/exec.CommandContext — which takes the binary and its arguments as a
// pre-split slice and execs directly, NEVER through a shell — so there is no
// interpretation of metacharacters and no command-injection surface. It streams
// combined stdout+stderr to onLog line-by-line and returns the process exit
// code (cross-platform; no build tags needed — exec.ExitError carries the code
// on every OS).
type osCommandRunner struct{}

var _ CommandRunner = osCommandRunner{}

func (osCommandRunner) Run(ctx context.Context, argv []string, dir string, onLog func(string)) (int, error) {
	if len(argv) == 0 {
		return startFailureExitCode, errors.New("empty argv")
	}

	// #nosec G204 — argv is a typed, validated token list (BuildArgv rejects
	// shell metacharacters); the binary is operator-configured. This is the
	// allowlisted-verb execution path, not a shell.
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
		// A non-zero exit is a normal job outcome, not a runner error.
		return exitErr.ExitCode(), nil
	}
	// ctx cancellation / kill / start failure surfaces here.
	return startFailureExitCode, err
}

// streamLines reads r line-by-line and forwards each line (with its trailing
// newline preserved) to onLog until EOF.
func streamLines(wg *sync.WaitGroup, r io.Reader, onLog func(string)) {
	defer wg.Done()
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for scanner.Scan() {
		onLog(scanner.Text() + "\n")
	}
}
