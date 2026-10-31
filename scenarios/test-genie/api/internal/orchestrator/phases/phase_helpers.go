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

	"github.com/vrooli/envkit-go"
	"test-genie/internal/shared"
)

var (
	commandLookup        = exec.LookPath
	phaseCommandExecutor = runCommand
	phaseCommandCapture  = runCommandCapture
)

func normalizeCommandInvocation(name string, args []string) (string, []string) {
	return name, append([]string(nil), args...)
}

// ParseJSON parses JSON from a string into a target value.
// This is the standard helper for parsing JSON across phases.
func ParseJSON(data string, v interface{}) error {
	return json.Unmarshal([]byte(data), v)
}

func EnsureCommandAvailable(name string) error {
	if _, err := commandLookup(name); err != nil {
		return fmt.Errorf("required command '%s' is not available: %w", name, err)
	}
	return nil
}

// Logging functions - aliases to shared package for backwards compatibility.
// New code should use shared.Log* directly.
var logPhaseStep = shared.LogStep

// phaseCommandEnv is the environment of every phase command: color disabled
// so logs stay readable, and the build-width floor composed over the
// inherited environment so a phase's go, pnpm or vite never runs wide.
func phaseCommandEnv(parent envkit.Env) envkit.Env {
	return envkit.Toolchain(envkit.WithOverlay(parent, envkit.SameScenario, envkit.Env{
		"NO_COLOR=1",
		"FORCE_COLOR=0",
		"CLICOLOR=0",
		"TERM=dumb",
	}), envkit.ToolchainOptions{})
}

func runCommand(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	name, args = normalizeCommandInvocation(name, args)
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = phaseCommandEnv(envkit.Env(os.Environ()))
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
	cmd.Env = phaseCommandEnv(envkit.Env(os.Environ()))
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
