package shell

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
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

func Command(spec Spec) *exec.Cmd {
	var cmd *exec.Cmd
	if spec.Context != nil {
		cmd = exec.CommandContext(spec.Context, spec.Name, spec.Args...)
	} else {
		cmd = exec.Command(spec.Name, spec.Args...)
	}
	cmd.Dir = spec.Dir
	cmd.Env = spec.Env
	cmd.Stdin = spec.Stdin
	cmd.Stdout = spec.Stdout
	cmd.Stderr = spec.Stderr
	return cmd
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

func BashCommand(command string, spec Spec) *exec.Cmd {
	spec.Name = "bash"
	spec.Args = []string{"-lc", command}
	return CommandWithDefaults(spec)
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
