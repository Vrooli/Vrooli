package deployment

import (
	"sync"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/vps"
)

// Event represents a single progress update sent via SSE.
type Event struct {
	Type            string                    `json:"type"`                 // "step_started", "step_completed", "error", "completed", "progress_update"
	Step            string                    `json:"step"`                 // Step ID (e.g., "upload", "setup")
	StepTitle       string                    `json:"step_title,omitempty"` // Human-readable title
	Progress        float64                   `json:"progress"`             // 0-100 percentage
	Message         string                    `json:"message,omitempty"`    // Optional message
	Error           string                    `json:"error,omitempty"`      // Error details if type is "error"
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

// SetupSteps defines the steps in the VPS setup phase.
var SetupSteps = []StepInfo{
	{ID: "mkdir", Title: "Creating directories", Weight: vps.StepWeights["mkdir"]},
	{ID: "bootstrap", Title: "Installing prerequisites", Weight: vps.StepWeights["bootstrap"]},
	{ID: "upload", Title: "Uploading bundle", Weight: vps.StepWeights["upload"]},
	{ID: "extract", Title: "Extracting bundle", Weight: vps.StepWeights["extract"]},
	{ID: "setup", Title: "Running setup", Weight: vps.StepWeights["setup"]},
	{ID: "autoheal", Title: "Configuring autoheal", Weight: vps.StepWeights["autoheal"]},
	{ID: "verify_setup", Title: "Verifying installation", Weight: vps.StepWeights["verify_setup"]},
}

// DeploySteps defines the steps in the VPS deploy phase.
var DeploySteps = []StepInfo{
	{ID: "scenario_stop", Title: "Stopping existing scenario", Weight: vps.StepWeights["scenario_stop"]},
	{ID: "caddy_install", Title: "Installing Caddy", Weight: vps.StepWeights["caddy_install"]},
	{ID: "caddy_config", Title: "Configuring Caddy", Weight: vps.StepWeights["caddy_config"]},
	{ID: "firewall_inbound", Title: "Opening inbound HTTP/HTTPS", Weight: vps.StepWeights["firewall_inbound"]},
	{ID: "secrets_provision", Title: "Provisioning secrets", Weight: vps.StepWeights["secrets_provision"]},
	{ID: "resource_start", Title: "Starting resources", Weight: vps.StepWeights["resource_start"]},
	{ID: "scenario_deps", Title: "Starting dependencies", Weight: vps.StepWeights["scenario_deps"]},
	{ID: "scenario_target", Title: "Starting scenario", Weight: vps.StepWeights["scenario_target"]},
	{ID: "wait_for_ui", Title: "Waiting for UI to listen", Weight: vps.StepWeights["wait_for_ui"]},
	{ID: "verify_local", Title: "Verifying local health", Weight: vps.StepWeights["verify_local"]},
	{ID: "verify_https", Title: "Verifying HTTPS", Weight: vps.StepWeights["verify_https"]},
	{ID: "verify_origin", Title: "Verifying origin reachability", Weight: vps.StepWeights["verify_origin"]},
	{ID: "verify_public", Title: "Verifying public reachability", Weight: vps.StepWeights["verify_public"]},
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
