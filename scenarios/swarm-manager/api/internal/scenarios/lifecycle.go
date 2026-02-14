package scenarios

import (
	"context"
	"strings"
	"time"
)

// DOC: docs/concepts/ARCHITECTURE.md#key-flows
// DOC: docs/internal/SEAMS.md

const (
	defaultStartTimeout   = 60 * time.Second
	defaultStopTimeout    = 20 * time.Second
	defaultRestartTimeout = 90 * time.Second
)

// Lifecycle controls scenario start/stop/restart operations.
type Lifecycle interface {
	Start(ctx context.Context, name string) error
	Stop(ctx context.Context, name string) error
	Restart(ctx context.Context, name string) error
}

// CLILifecycle executes lifecycle actions via the Vrooli CLI.
type CLILifecycle struct {
	startTimeout   time.Duration
	stopTimeout    time.Duration
	restartTimeout time.Duration
}

// NewCLILifecycle creates a CLI-backed lifecycle controller.
func NewCLILifecycle() *CLILifecycle {
	return &CLILifecycle{
		startTimeout:   defaultStartTimeout,
		stopTimeout:    defaultStopTimeout,
		restartTimeout: defaultRestartTimeout,
	}
}

// Start starts a scenario using the Vrooli CLI.
func (c *CLILifecycle) Start(ctx context.Context, name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return errScenarioNameRequired
	}
	_, err := executeVrooliCommand(ctx, c.startTimeout, "scenario", "start", trimmed)
	return err
}

// Stop stops a scenario using the Vrooli CLI.
func (c *CLILifecycle) Stop(ctx context.Context, name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return errScenarioNameRequired
	}
	_, err := executeVrooliCommand(ctx, c.stopTimeout, "scenario", "stop", trimmed)
	return err
}

// Restart restarts a scenario using the Vrooli CLI.
func (c *CLILifecycle) Restart(ctx context.Context, name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return errScenarioNameRequired
	}
	_, err := executeVrooliCommand(ctx, c.restartTimeout, "scenario", "restart", trimmed)
	return err
}
