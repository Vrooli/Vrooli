package supervisor

import "context"

// State represents the current state of the supervisor.
type State string

const (
	// StateStarting indicates the supervisor is starting the process.
	StateStarting State = "starting"

	// StateRunning indicates the process is running and healthy.
	StateRunning State = "running"

	// StateStopping indicates the supervisor is stopping the process.
	StateStopping State = "stopping"

	// StateStopped indicates the process is stopped.
	StateStopped State = "stopped"

	// StateRestarting indicates the supervisor is restarting after a crash.
	StateRestarting State = "restarting"

	// StateUnrecoverable indicates too many restarts have occurred.
	// Manual intervention or scenario restart is required.
	StateUnrecoverable State = "unrecoverable"
)

// IsTerminal returns true if the state is terminal (stopped or unrecoverable).
func (s State) IsTerminal() bool {
	return s == StateStopped || s == StateUnrecoverable
}

// IsHealthy returns true if the state indicates a healthy, running process.
func (s State) IsHealthy() bool {
	return s == StateRunning
}

// StateEvent represents a state change event from the supervisor.
type StateEvent struct {
	// Previous is the state before the transition.
	Previous State

	// Current is the new state after the transition.
	Current State

	// RestartCount is the number of restarts within the current window.
	RestartCount int

	// Error is set if the state change was caused by an error.
	Error error
}

// Supervisor manages the lifecycle of a sidecar process.
// It handles starting, stopping, monitoring, and automatic restarts.
type Supervisor interface {
	// Start begins the sidecar process and waits for it to become healthy.
	// Returns an error if the process fails to start or become healthy.
	Start(ctx context.Context) error

	// Stop gracefully stops the sidecar process.
	// It first sends SIGTERM, then SIGKILL after GracefulStop timeout.
	Stop(ctx context.Context) error

	// Restart stops and then starts the sidecar process.
	// Useful for manual restart requests from the UI.
	Restart(ctx context.Context) error

	// State returns the current supervisor state.
	State() State

	// RestartCount returns the number of restarts within the current window.
	RestartCount() int

	// Subscribe returns a channel that receives state change events.
	// The channel is closed when the supervisor is stopped.
	// Multiple subscribers are supported.
	Subscribe() <-chan StateEvent

	// Unsubscribe removes a subscriber channel.
	Unsubscribe(ch <-chan StateEvent)
}

// compile-time check that ProcessSupervisor implements Supervisor
var _ Supervisor = (*ProcessSupervisor)(nil)
