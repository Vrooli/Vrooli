package shell

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/vrooli/envkit-go"
	"github.com/vrooli/vrooli/internal/tuning"
)

// StderrTail holds the last N lines of stderr written through TeeStderr.
// It is safe for concurrent writes from a single command's stderr stream.
type StderrTail struct {
	max   int
	mu    sync.Mutex
	buf   strings.Builder
	lines []string
}

// NewStderrTail returns a tail that retains the most recent maxLines complete
// lines of stderr (a partial trailing line is kept in the internal buffer
// until a newline arrives). maxLines <= 0 falls back to 10.
func NewStderrTail(maxLines int) *StderrTail {
	if maxLines <= 0 {
		maxLines = 10
	}
	return &StderrTail{max: maxLines}
}

// Write implements io.Writer; it splits incoming bytes on '\n' and keeps the
// trailing N complete lines.
func (t *StderrTail) Write(p []byte) (int, error) {
	if t == nil {
		return len(p), nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf.Write(p)
	for {
		current := t.buf.String()
		idx := strings.IndexByte(current, '\n')
		if idx < 0 {
			break
		}
		line := strings.TrimRight(current[:idx], "\r")
		t.lines = append(t.lines, line)
		if len(t.lines) > t.max {
			t.lines = t.lines[len(t.lines)-t.max:]
		}
		t.buf.Reset()
		t.buf.WriteString(current[idx+1:])
	}
	return len(p), nil
}

// Tail returns the captured lines (most recent at the end). A non-empty
// trailing partial line is included as the last entry.
func (t *StderrTail) Tail() []string {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	out := append([]string(nil), t.lines...)
	if rem := strings.TrimRight(t.buf.String(), "\r"); rem != "" {
		out = append(out, rem)
		if len(out) > t.max {
			out = out[len(out)-t.max:]
		}
	}
	return out
}

// String returns the tail joined by '\n'.
func (t *StderrTail) String() string {
	return strings.Join(t.Tail(), "\n")
}

type Spec struct {
	Context context.Context
	Name    string
	Args    []string
	Dir     string
	Env     []string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
}

// Runner is the shared boundary for observing external commands. Run returns
// combined standard output and standard error so callers retain diagnostic
// output when a probe exits unsuccessfully.
type Runner interface {
	LookPath(name string) (string, error)
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

// OSRunner executes commands through the operating system.
type OSRunner struct{}

func (OSRunner) LookPath(name string) (string, error) { return LookPath(name) }

func (OSRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return CombinedOutput(Spec{Context: ctx, Name: name, Args: args})
}

// Command builds the process for a Spec. A toolchain program (go, pnpm, npm,
// cargo, uv, vite, tsc) always receives the build-width floor from
// envkit.Toolchain, composed over the Spec's environment or, when the Spec
// has none, over this process's environment. Every other program keeps the
// Spec's environment untouched, including nil (inherit).
func Command(spec Spec) *exec.Cmd {
	var cmd *exec.Cmd
	if spec.Context != nil {
		cmd = exec.CommandContext(spec.Context, spec.Name, spec.Args...)
	} else {
		cmd = exec.Command(spec.Name, spec.Args...)
	}
	cmd.Dir = spec.Dir
	cmd.Env = spec.Env
	if IsToolchainProgram(spec.Name) {
		parent := envkit.Env(spec.Env)
		if parent == nil {
			parent = envkit.Env(os.Environ())
		}
		cmd.Env = envkit.Toolchain(parent, envkit.ToolchainOptions{Width: tuning.BuildWidth()})
	}
	cmd.Stdin = spec.Stdin
	cmd.Stdout = spec.Stdout
	cmd.Stderr = spec.Stderr
	return cmd
}

// NewCommand constructs a command through the shared shell boundary while
// preserving the standard library's variadic call shape. Use Command with a
// Spec when the caller also needs directory, environment, or stream settings.
func NewCommand(name string, args ...string) *exec.Cmd {
	return Command(Spec{Name: name, Args: args})
}

// NewCommandContext is the context-aware counterpart to NewCommand.
func NewCommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	return Command(Spec{Context: ctx, Name: name, Args: args})
}

func CommandWithDefaults(spec Spec) *exec.Cmd {
	cmd := Command(spec)
	if cmd.Stdin == nil {
		cmd.Stdin = os.Stdin
	}
	if cmd.Stdout == nil {
		cmd.Stdout = os.Stdout
	}
	if cmd.Stderr == nil {
		cmd.Stderr = os.Stderr
	}
	return cmd
}

func Run(spec Spec) error {
	return CommandWithDefaults(spec).Run()
}

func Output(spec Spec) ([]byte, error) {
	return Command(spec).Output()
}

func CombinedOutput(spec Spec) ([]byte, error) {
	return Command(spec).CombinedOutput()
}

func LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

// toolchainPrograms are the build tools whose parallelism the floor bounds.
var toolchainPrograms = map[string]bool{"go": true, "pnpm": true, "npm": true, "cargo": true, "uv": true, "vite": true, "tsc": true}

// IsToolchainProgram reports whether name (bare or a path, with or without a
// Windows extension) is one of the build tools the toolchain floor governs.
func IsToolchainProgram(name string) bool {
	base := filepath.Base(strings.TrimSpace(name))
	base = strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(base, ".exe"), ".cmd"), ".bat")
	return toolchainPrograms[strings.ToLower(base)]
}
