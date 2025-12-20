// Package discovery provides runtime helpers for resolving scenario ports.
package discovery

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// CommandRunner executes a command and returns combined stdout/stderr.
type CommandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

// ResolverConfig configures a Resolver.
type ResolverConfig struct {
	// VrooliPath is the CLI binary to invoke. Defaults to "vrooli".
	VrooliPath string

	// CommandRunner overrides command execution (useful for tests).
	CommandRunner CommandRunner

	// Host defaults to "localhost" when building URLs.
	Host string

	// Scheme defaults to "http" when building URLs.
	Scheme string
}

// Resolver resolves scenario ports by shelling out to the Vrooli CLI.
// It intentionally performs a fresh lookup every call (no caching).
type Resolver struct {
	vrooliPath string
	runner     CommandRunner
	host       string
	scheme     string
}

const defaultPortKey = "API_PORT"

// ErrorKind identifies the class of discovery failure.
type ErrorKind string

const (
	ErrInvalidInput      ErrorKind = "invalid_input"
	ErrVrooliNotFound    ErrorKind = "vrooli_not_found"
	ErrScenarioNotRunning ErrorKind = "scenario_not_running"
	ErrTimeout           ErrorKind = "timeout"
	ErrInvalidPort       ErrorKind = "invalid_port"
	ErrCommandFailed     ErrorKind = "command_failed"
)

// Error provides structured details about discovery failures.
type Error struct {
	Kind     ErrorKind
	Scenario string
	PortKey  string
	Output   string
	Err      error
}

func (e *Error) Error() string {
	parts := []string{
		"api-core discovery",
		string(e.Kind),
		fmt.Sprintf("scenario=%q", e.Scenario),
		fmt.Sprintf("port=%q", e.PortKey),
	}
	if e.Output != "" {
		parts = append(parts, fmt.Sprintf("output=%q", e.Output))
	}
	if e.Err != nil {
		parts = append(parts, fmt.Sprintf("err=%v", e.Err))
	}
	return strings.Join(parts, " ")
}

func (e *Error) Unwrap() error {
	return e.Err
}

// IsScenarioNotRunning reports whether the error indicates a stopped scenario.
func IsScenarioNotRunning(err error) bool {
	var target *Error
	return errors.As(err, &target) && target.Kind == ErrScenarioNotRunning
}

// NewResolver constructs a Resolver with defaults applied.
func NewResolver(cfg ResolverConfig) *Resolver {
	vrooliPath := cfg.VrooliPath
	if vrooliPath == "" {
		vrooliPath = "vrooli"
	}
	runner := cfg.CommandRunner
	if runner == nil {
		runner = defaultRunner
	}
	host := cfg.Host
	if host == "" {
		host = "localhost"
	}
	scheme := cfg.Scheme
	if scheme == "" {
		scheme = "http"
	}
	return &Resolver{
		vrooliPath: vrooliPath,
		runner:     runner,
		host:       host,
		scheme:     scheme,
	}
}

// ResolveScenarioPort resolves a scenario's port by calling:
// `vrooli scenario port <slug> <portKey>`.
// This always executes the CLI; it does not cache results.
func (r *Resolver) ResolveScenarioPort(ctx context.Context, scenarioSlug, portKey string) (int, error) {
	if scenarioSlug == "" {
		return 0, &Error{
			Kind:     ErrInvalidInput,
			Scenario: scenarioSlug,
			PortKey:  portKey,
			Err:      errors.New("scenario slug is required"),
		}
	}
	if portKey == "" {
		portKey = defaultPortKey
	}

	output, err := r.runner(ctx, r.vrooliPath, "scenario", "port", scenarioSlug, portKey)
	text := strings.TrimSpace(string(output))
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return 0, &Error{
				Kind:     ErrTimeout,
				Scenario: scenarioSlug,
				PortKey:  portKey,
				Output:   text,
				Err:      ctxErr,
			}
		}
		if errors.Is(err, exec.ErrNotFound) {
			return 0, &Error{
				Kind:     ErrVrooliNotFound,
				Scenario: scenarioSlug,
				PortKey:  portKey,
				Output:   text,
				Err:      err,
			}
		}
		lower := strings.ToLower(text)
		if strings.Contains(lower, "not running") || strings.Contains(lower, "not started") {
			return 0, &Error{
				Kind:     ErrScenarioNotRunning,
				Scenario: scenarioSlug,
				PortKey:  portKey,
				Output:   text,
				Err:      err,
			}
		}
		return 0, &Error{
			Kind:     ErrCommandFailed,
			Scenario: scenarioSlug,
			PortKey:  portKey,
			Output:   text,
			Err:      err,
		}
	}

	port, parseErr := strconv.Atoi(text)
	if parseErr != nil || port <= 0 {
		return 0, &Error{
			Kind:     ErrInvalidPort,
			Scenario: scenarioSlug,
			PortKey:  portKey,
			Output:   text,
			Err:      parseErr,
		}
	}

	return port, nil
}

// ResolveScenarioURL resolves a scenario's port and returns a URL
// using the resolver's scheme and host.
func (r *Resolver) ResolveScenarioURL(ctx context.Context, scenarioSlug, portKey string) (string, error) {
	port, err := r.ResolveScenarioPort(ctx, scenarioSlug, portKey)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s://%s:%d", r.scheme, r.host, port), nil
}

// ResolveScenarioPortDefault resolves the standard API port for a scenario.
func (r *Resolver) ResolveScenarioPortDefault(ctx context.Context, scenarioSlug string) (int, error) {
	return r.ResolveScenarioPort(ctx, scenarioSlug, defaultPortKey)
}

// ResolveScenarioURLDefault resolves the standard API URL for a scenario.
func (r *Resolver) ResolveScenarioURLDefault(ctx context.Context, scenarioSlug string) (string, error) {
	return r.ResolveScenarioURL(ctx, scenarioSlug, defaultPortKey)
}

// ResolveScenarioPort is a convenience wrapper using default config.
func ResolveScenarioPort(ctx context.Context, scenarioSlug, portKey string) (int, error) {
	return NewResolver(ResolverConfig{}).ResolveScenarioPort(ctx, scenarioSlug, portKey)
}

// ResolveScenarioURL is a convenience wrapper using default config.
func ResolveScenarioURL(ctx context.Context, scenarioSlug, portKey string) (string, error) {
	return NewResolver(ResolverConfig{}).ResolveScenarioURL(ctx, scenarioSlug, portKey)
}

// ResolveScenarioPortDefault is a convenience wrapper using the standard API port.
func ResolveScenarioPortDefault(ctx context.Context, scenarioSlug string) (int, error) {
	return NewResolver(ResolverConfig{}).ResolveScenarioPortDefault(ctx, scenarioSlug)
}

// ResolveScenarioURLDefault is a convenience wrapper using the standard API port.
func ResolveScenarioURLDefault(ctx context.Context, scenarioSlug string) (string, error) {
	return NewResolver(ResolverConfig{}).ResolveScenarioURLDefault(ctx, scenarioSlug)
}

func defaultRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}
