package deployment

import (
	"sync"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/vps"
)

// Event represents a single progress update sent via SSE.
type Event struct {
	Type            string                    `json:"type"`                     // "step_started", "step_completed", "error", "completed", "progress_update"
	Step            string                    `json:"step"`                     // Step ID (e.g., "upload", "setup")
	StepTitle       string                    `json:"step_title,omitempty"`     // Human-readable title
	Progress        float64                   `json:"progress"`                 // 0-100 percentage
	Message         string                    `json:"message,omitempty"`        // Optional message
	Error           string                    `json:"error,omitempty"`          // Error details if type is "error"
	ErrorCategory   string                    `json:"error_category,omitempty"` // Machine-readable error category (e.g. "disk_full")
	Retryable       bool                      `json:"retryable,omitempty"`      // Whether the error is retryable
	Hint            string                    `json:"hint,omitempty"`           // Actionable recovery suggestion
	PreflightResult *domain.PreflightResponse `json:"preflight_result,omitempty"`
	Timestamp       string                    `json:"timestamp"` // ISO8601 timestamp
}

// Emitter is an interface for emitting progress events.
type Emitter interface {
	Emit(event Event)
	Close()
}

// StepInfo provides metadata for each deployment step.
type StepInfo struct {
	ID     string
	Title  string
	Weight float64
}

// SetupSteps defines the install-scope plan actions in order.
var SetupSteps = []StepInfo{
	{ID: "host.prepare", Title: "Preparing host", Weight: vps.StepWeights["host.prepare"]},
	{ID: "edge.firewall.allow", Title: "Opening inbound HTTP/HTTPS", Weight: vps.StepWeights["edge.firewall.allow"]},
	{ID: "data.inventory", Title: "Inventorying persistent data", Weight: vps.StepWeights["data.inventory"]},
	{ID: "release.deliver", Title: "Delivering release", Weight: vps.StepWeights["release.deliver"]},
	{ID: "release.verify", Title: "Verifying release", Weight: vps.StepWeights["release.verify"]},
	{ID: "release.stage", Title: "Staging release", Weight: vps.StepWeights["release.stage"]},
	{ID: "data.backup", Title: "Backing up persistent data", Weight: vps.StepWeights["data.backup"]},
	{ID: "release.activate", Title: "Activating release", Weight: vps.StepWeights["release.activate"]},
	{ID: "config.apply", Title: "Applying configuration", Weight: vps.StepWeights["config.apply"]},
}

// DeploySteps defines the runtime-scope plan actions in order.
var DeploySteps = []StepInfo{
	{ID: "workload.stop", Title: "Stopping existing scenario", Weight: vps.StepWeights["workload.stop"]},
	{ID: "edge.route.apply", Title: "Configuring edge route", Weight: vps.StepWeights["edge.route.apply"]},
	{ID: "credentials.provision", Title: "Provisioning credentials", Weight: vps.StepWeights["credentials.provision"]},
	{ID: "runtime.start_dependencies", Title: "Starting dependencies", Weight: vps.StepWeights["runtime.start_dependencies"]},
	{ID: "workload.start", Title: "Starting scenario", Weight: vps.StepWeights["workload.start"]},
	{ID: "verify.readiness", Title: "Verifying readiness", Weight: vps.StepWeights["verify.readiness"]},
	{ID: "release.retain_predecessor", Title: "Retaining predecessor", Weight: vps.StepWeights["release.retain_predecessor"]},
}

// AllSteps returns all deployment steps in order.
func AllSteps() []StepInfo {
	steps := []StepInfo{
		{ID: "bundle_build", Title: "Building bundle", Weight: vps.StepWeights["bundle_build"]},
		{ID: "preflight", Title: "Running preflight checks", Weight: vps.StepWeights["preflight"]},
	}
	steps = append(steps, SetupSteps...)
	steps = append(steps, DeploySteps...)
	return steps
}

// ChannelEmitter implements Emitter using Go channels.
type ChannelEmitter struct {
	ch     chan Event
	closed bool
	mu     sync.Mutex
}

// NewChannelEmitter creates a new ChannelEmitter with a buffered channel.
func NewChannelEmitter(bufferSize int) *ChannelEmitter {
	return &ChannelEmitter{
		ch: make(chan Event, bufferSize),
	}
}

// Emit sends a progress event to the channel.
func (e *ChannelEmitter) Emit(event Event) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return
	}

	// Set timestamp if not already set
	if event.Timestamp == "" {
		event.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	// Non-blocking send to avoid deadlock if channel is full
	select {
	case e.ch <- event:
	default:
		// Channel full, drop event (shouldn't happen with proper buffer size)
	}
}

// Close closes the emitter channel.
func (e *ChannelEmitter) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.closed {
		e.closed = true
		close(e.ch)
	}
}

// Channel returns the underlying channel for reading events.
func (e *ChannelEmitter) Channel() <-chan Event {
	return e.ch
}

// NoOpEmitter is an Emitter that does nothing.
// Used when progress tracking is not needed.
type NoOpEmitter struct{}

// Emit does nothing.
func (NoOpEmitter) Emit(Event) {}

// Close does nothing.
func (NoOpEmitter) Close() {}
