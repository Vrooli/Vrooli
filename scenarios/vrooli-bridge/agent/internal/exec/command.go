package exec

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	return osCommandRunner{}.run(ctx, argv, dir, nil, onLog)
}

func (osCommandRunner) RunWithEnvironment(ctx context.Context, argv []string, dir string, env []string, onLog func(string)) (int, error) {
	return osCommandRunner{}.run(ctx, argv, dir, env, onLog)
}

func (osCommandRunner) run(ctx context.Context, argv []string, dir string, overlay []string, onLog func(string)) (int, error) {
	if len(argv) == 0 {
		return startFailureExitCode, errors.New("empty argv")
	}

	// Build the environment before resolving argv[0]. os/exec performs LookPath
	// during Command construction, so assigning Env afterwards cannot make a
	// service-manager PATH visible to lookup.
	env := commandEnvironment(argv[0])
	resolved, err := resolveExecutable(argv[0], env)
	if err != nil {
		return startFailureExitCode, err
	}
	// #nosec G204 — argv is a typed, validated token list (BuildArgv rejects
	// shell metacharacters); the binary is operator-configured. This is the
	// allowlisted-verb execution path, not a shell.
	cmd := exec.Command(resolved, argv[1:]...)
	prepareCommand(cmd)
	cmd.Dir = dir
	// Native service managers intentionally provide a minimal PATH. The Vrooli
	// CLI is often an absolute path in that environment, but its typed commands
	// may invoke the host Go/toolchain binaries. Reconstruct the project-owned
	// toolchain locations that bootstrap already treats as authoritative so a
	// background node behaves like the same node in an interactive shell.
	cmd.Env = env
	if len(overlay) > 0 {
		cmd.Env = append(cmd.Env, overlay...)
	}

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

	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()
	var waitErr error
	select {
	case waitErr = <-waitDone:
	case <-ctx.Done():
		terminateCommand(cmd)
		waitErr = <-waitDone
	}

	wg.Wait()

	err = waitErr
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

func resolveExecutable(binary string, env []string) (string, error) {
	if filepath.IsAbs(binary) {
		info, err := os.Stat(binary)
		if err != nil {
			return "", fmt.Errorf("resolve executable %q: %w", binary, err)
		}
		if info.IsDir() || info.Mode()&0o111 == 0 {
			return "", fmt.Errorf("resolve executable %q: not an executable file", binary)
		}
		return filepath.Clean(binary), nil
	}
	pathValue := ""
	for _, item := range env {
		if strings.HasPrefix(item, "PATH=") {
			pathValue = strings.TrimPrefix(item, "PATH=")
			break
		}
	}
	for _, dir := range strings.Split(pathValue, string(filepath.ListSeparator)) {
		if dir == "" {
			dir = "."
		}
		candidate := filepath.Join(dir, binary)
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			absolute, absErr := filepath.Abs(candidate)
			if absErr != nil {
				return "", absErr
			}
			return filepath.Clean(absolute), nil
		}
	}
	return "", fmt.Errorf("resolve executable %q in PATH %q: %w", binary, pathValue, exec.ErrNotFound)
}

func commandEnvironment(binary string) []string {
	env := os.Environ()
	currentPath := os.Getenv("PATH")
	paths := make([]string, 0, 8)
	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		if _, err := os.Stat(path); err != nil { // #nosec G703 -- PATH entries are local operator environment, used only for executable discovery.
			return
		}
		for _, existing := range paths {
			if existing == path {
				return
			}
		}
		paths = append(paths, path)
	}

	if binaryDir := filepath.Dir(binary); binaryDir != "." {
		add(binaryDir)
	}
	if home, err := os.UserHomeDir(); err == nil {
		add(filepath.Join(home, ".vrooli", "bin"))
		add(filepath.Join(home, ".local", "bin"))
		add(filepath.Join(home, "go", "bin"))
	}
	for _, path := range []string{"/usr/local/go/bin", "/opt/homebrew/bin", "/usr/local/bin"} {
		add(path)
	}

	for _, existing := range strings.Split(currentPath, string(os.PathListSeparator)) {
		add(existing)
	}
	if len(paths) == 0 {
		return env
	}
	pathValue := strings.Join(paths, string(os.PathListSeparator))
	for i, value := range env {
		if strings.HasPrefix(value, "PATH=") {
			env[i] = "PATH=" + pathValue
			return env
		}
	}
	return append(env, "PATH="+pathValue)
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
