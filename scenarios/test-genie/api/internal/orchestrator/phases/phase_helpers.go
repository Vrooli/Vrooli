package phases

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"unicode/utf8"

	"test-genie/internal/shared"
)

var (
	commandLookup        = exec.LookPath
	phaseCommandExecutor = runCommand
	phaseCommandCapture  = runCommandCapture
)

func normalizeCommandInvocation(name string, args []string) (string, []string) {
	if name != "vrooli" || containsArg(args, "--no-stale-check") {
		return name, args
	}
	normalized := make([]string, 0, len(args)+1)
	normalized = append(normalized, "--no-stale-check")
	normalized = append(normalized, args...)
	return name, normalized
}

func containsArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ParseJSON parses JSON from a string into a target value.
// This is the standard helper for parsing JSON across phases.
func ParseJSON(data string, v interface{}) error {
	return json.Unmarshal([]byte(data), v)
}

func extractCapturedJSONObject(provider string, raw []byte) ([]byte, error) {
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return nil, fmt.Errorf("%s produced empty output", provider)
	}
	start := strings.IndexByte(text, '{')
	if start < 0 {
		return nil, fmt.Errorf("%s output did not contain a JSON object", provider)
	}
	end := strings.LastIndexByte(text, '}')
	if end < start {
		return nil, fmt.Errorf("%s output contained an incomplete JSON object", provider)
	}
	return []byte(text[start : end+1]), nil
}

func EnsureCommandAvailable(name string) error {
	if _, err := commandLookup(name); err != nil {
		return fmt.Errorf("required command '%s' is not available: %w", name, err)
	}
	return nil
}

// Logging functions - aliases to shared package for backwards compatibility.
// New code should use shared.Log* directly.
var (
	logPhaseStep    = shared.LogStep
	logPhaseSuccess = shared.LogSuccess
)

func runCommand(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	name, args = normalizeCommandInvocation(name, args)
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = append(os.Environ(),
		"NO_COLOR=1",
		"FORCE_COLOR=0",
		"CLICOLOR=0",
		"TERM=dumb",
	)
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
	name, args = normalizeCommandInvocation(name, args)
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	// Disable ANSI color in captured output to keep logs readable.
	cmd.Env = append(os.Environ(),
		"NO_COLOR=1",
		"FORCE_COLOR=0",
		"CLICOLOR=0",
		"TERM=dumb",
	)
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

// stripANSIWriter removes ANSI escape sequences from writes before forwarding.
type stripANSIWriter struct {
	target io.Writer
}

func (w *stripANSIWriter) Write(p []byte) (int, error) {
	clean := stripANSI(p)
	n, err := w.target.Write(clean)
	if err != nil {
		return 0, err
	}
	if n != len(clean) {
		return 0, io.ErrShortWrite
	}
	// Report that we've consumed the full input so io.Copy doesn't treat stripping
	// as a short write and close the underlying exec pipes (which can SIGPIPE the
	// child process).
	return len(p), nil
}

// stripANSI removes ANSI escape sequences from a byte slice.
func stripANSI(p []byte) []byte {
	var out []rune
	for i := 0; i < len(p); {
		r, size := utf8.DecodeRune(p[i:])
		// Detect CSI sequences: ESC [
		if r == 0x1b && i+1 < len(p) && p[i+1] == '[' {
			// Skip until letter terminator
			j := i + 2
			for j < len(p) {
				if (p[j] >= 'A' && p[j] <= 'Z') || (p[j] >= 'a' && p[j] <= 'z') {
					j++
					break
				}
				j++
			}
			i = j
			continue
		}
		out = append(out, r)
		i += size
	}
	return []byte(string(out))
}

// wrapLogSansANSI ensures downstream logs are ANSI-free.
func wrapLogSansANSI(w io.Writer) io.Writer {
	if w == nil {
		return nil
	}
	return &stripANSIWriter{target: w}
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

// Scenario interaction utilities - used by playbooks, smoke, and other runtime phases.

// ResolveScenarioPort resolves a port for a scenario using vrooli CLI.
// Returns the port number as a string.
func ResolveScenarioPort(ctx context.Context, logWriter io.Writer, scenarioName, portName string) (string, error) {
	output, err := phaseCommandCapture(ctx, "", logWriter, "vrooli", "scenario", "port", scenarioName, portName)
	if err != nil {
		return "", fmt.Errorf("vrooli port lookup failed: %w", err)
	}
	value := strings.TrimSpace(output)
	if value == "" {
		return "", fmt.Errorf("port lookup returned empty output")
	}
	// Parse output which may contain "PORT_NAME=value" format
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if strings.TrimSpace(parts[0]) == portName {
				value = strings.TrimSpace(parts[1])
				break
			}
		}
	}
	value = strings.TrimSpace(value)
	value = strings.TrimSuffix(value, "\r")
	// Validate it's a number
	if _, err := fmt.Sscanf(value, "%d", new(int)); err != nil {
		return "", fmt.Errorf("invalid port value %q", value)
	}
	return value, nil
}

// ResolveScenarioBaseURL resolves the UI base URL for a scenario.
func ResolveScenarioBaseURL(ctx context.Context, logWriter io.Writer, scenarioName string) (string, error) {
	port, err := ResolveScenarioPort(ctx, logWriter, scenarioName, "UI_PORT")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("http://127.0.0.1:%s", port), nil
}
