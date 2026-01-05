package main

import (
	"sync"
	"time"
)

// ProgressEvent represents a single progress update sent via SSE.
type ProgressEvent struct {
	Type      string  `json:"type"`                 // "step_started", "step_completed", "error", "completed", "progress_update"
	Step      string  `json:"step"`                 // Step ID (e.g., "upload", "setup")
	StepTitle string  `json:"step_title,omitempty"` // Human-readable title
	Progress  float64 `json:"progress"`             // 0-100 percentage
	Message   string  `json:"message,omitempty"`    // Optional message
	Error     string  `json:"error,omitempty"`      // Error details if type is "error"
	Timestamp string  `json:"timestamp"`            // ISO8601 timestamp
}

// ProgressEmitter is an interface for emitting progress events.
type ProgressEmitter interface {
	Emit(event ProgressEvent)
	Close()
}

// StepWeights defines the percentage weight for each deployment step.
// These sum to 100%.
var StepWeights = map[string]float64{
	"bundle_build":      5,
	"mkdir":             0, // Trivial, no weight
	"bootstrap":         5, // apt update + install prereqs
	"upload":            18,
	"extract":           5,
	"setup":             12, // Reduced from 15 (bootstrap handles some work now)
	"autoheal":          2,
	"verify_setup":      1, // Reduced from 3
	"scenario_stop":     3, // Stop existing scenario before deployment
	"caddy_install":     5,
	"caddy_config":      5,
	"secrets_provision": 5, // Generate and write secrets before resource start
	"resource_start":    10,
	"scenario_deps":     10,
	"scenario_target":   9,
	"wait_for_ui":       1,
	"verify_local":      2,
	"verify_https":      2,
}

// StepInfo provides metadata for each deployment step.
type StepInfo struct {
	ID     string
	Title  string
	Weight float64
}

// SetupSteps defines the steps in the VPS setup phase.
var SetupSteps = []StepInfo{
	{ID: "mkdir", Title: "Creating directories", Weight: StepWeights["mkdir"]},
	{ID: "bootstrap", Title: "Installing prerequisites", Weight: StepWeights["bootstrap"]},
	{ID: "upload", Title: "Uploading bundle", Weight: StepWeights["upload"]},
	{ID: "extract", Title: "Extracting bundle", Weight: StepWeights["extract"]},
	{ID: "setup", Title: "Running setup", Weight: StepWeights["setup"]},
	{ID: "autoheal", Title: "Configuring autoheal", Weight: StepWeights["autoheal"]},
	{ID: "verify_setup", Title: "Verifying installation", Weight: StepWeights["verify_setup"]},
}

// DeploySteps defines the steps in the VPS deploy phase.
var DeploySteps = []StepInfo{
	{ID: "scenario_stop", Title: "Stopping existing scenario", Weight: StepWeights["scenario_stop"]},
	{ID: "caddy_install", Title: "Installing Caddy", Weight: StepWeights["caddy_install"]},
	{ID: "caddy_config", Title: "Configuring Caddy", Weight: StepWeights["caddy_config"]},
	{ID: "secrets_provision", Title: "Provisioning secrets", Weight: StepWeights["secrets_provision"]},
	{ID: "resource_start", Title: "Starting resources", Weight: StepWeights["resource_start"]},
	{ID: "scenario_deps", Title: "Starting dependencies", Weight: StepWeights["scenario_deps"]},
	{ID: "scenario_target", Title: "Starting scenario", Weight: StepWeights["scenario_target"]},
	{ID: "wait_for_ui", Title: "Waiting for UI to listen", Weight: StepWeights["wait_for_ui"]},
	{ID: "verify_local", Title: "Verifying local health", Weight: StepWeights["verify_local"]},
	{ID: "verify_https", Title: "Verifying HTTPS", Weight: StepWeights["verify_https"]},
}

// AllSteps returns all deployment steps in order.
func AllSteps() []StepInfo {
	steps := []StepInfo{
		{ID: "bundle_build", Title: "Building bundle", Weight: StepWeights["bundle_build"]},
	}
	steps = append(steps, SetupSteps...)
	steps = append(steps, DeploySteps...)
	return steps
}

// ChannelEmitter implements ProgressEmitter using Go channels.
type ChannelEmitter struct {
	ch     chan ProgressEvent
	closed bool
	mu     sync.Mutex
}

// NewChannelEmitter creates a new ChannelEmitter with a buffered channel.
func NewChannelEmitter(bufferSize int) *ChannelEmitter {
	return &ChannelEmitter{
		ch: make(chan ProgressEvent, bufferSize),
	}
}

// Emit sends a progress event to the channel.
func (e *ChannelEmitter) Emit(event ProgressEvent) {
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
func (e *ChannelEmitter) Channel() <-chan ProgressEvent {
	return e.ch
}

// NoOpEmitter is a ProgressEmitter that does nothing.
// Used when progress tracking is not needed.
type NoOpEmitter struct{}

// Emit does nothing.
func (NoOpEmitter) Emit(ProgressEvent) {}

// Close does nothing.
func (NoOpEmitter) Close() {}
