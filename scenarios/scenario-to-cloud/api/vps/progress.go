package vps

import (
	"context"
	"time"

	"scenario-to-cloud/domain"
)

// ProgressEvent represents a single progress update sent via SSE.
// This mirrors the main package's ProgressEvent to allow decoupled progress tracking.
type ProgressEvent struct {
	Type            string                    `json:"type"`                 // "step_started", "step_completed", "error", "completed", "progress_update"
	Step            string                    `json:"step"`                 // Step ID (e.g., "upload", "setup")
	StepTitle       string                    `json:"step_title,omitempty"` // Human-readable title
	Progress        float64                   `json:"progress"`             // 0-100 percentage
	Message         string                    `json:"message,omitempty"`    // Optional message
	Error           string                    `json:"error,omitempty"`      // Error details if type is "error"
	PreflightResult *domain.PreflightResponse `json:"preflight_result,omitempty"`
	Timestamp       string                    `json:"timestamp"` // ISO8601 timestamp
}

// ProgressBroadcaster broadcasts progress events to subscribers.
type ProgressBroadcaster interface {
	Broadcast(deploymentID string, event ProgressEvent)
}

// ProgressRepo persists progress to storage.
type ProgressRepo interface {
	UpdateDeploymentProgress(ctx context.Context, id, step string, percent float64) error
}

// NoopProgressHub is a no-op implementation of ProgressBroadcaster.
type NoopProgressHub struct{}

// Broadcast does nothing.
func (NoopProgressHub) Broadcast(string, ProgressEvent) {}

// NoopProgressRepo is a no-op implementation of ProgressRepo.
type NoopProgressRepo struct{}

// UpdateDeploymentProgress does nothing.
func (NoopProgressRepo) UpdateDeploymentProgress(context.Context, string, string, float64) error {
	return nil
}

// StepWeights defines the percentage weight for each deployment step.
// These sum to 100%.
var StepWeights = map[string]float64{
	"preflight":         2,
	"bundle_build":      5,
	"mkdir":             0, // Trivial, no weight
	"bootstrap":         5, // apt update + install prereqs
	"upload":            17,
	"extract":           5,
	"setup":             11, // Reduced from 15 (bootstrap handles some work now)
	"autoheal":          2,
	"verify_setup":      1, // Reduced from 3
	"scenario_stop":     3, // Stop existing scenario before deployment
	"caddy_install":     5,
	"caddy_config":      5,
	"firewall_inbound":  1,
	"secrets_provision": 5, // Generate and write secrets before resource start
	"resource_start":    8,
	"scenario_deps":     10,
	"scenario_target":   9,
	"wait_for_ui":       1,
	"verify_local":      1,
	"verify_https":      1,
	"verify_origin":     1,
	"verify_public":     2,
}

// CalculateWeightsForSteps returns normalized weights for a subset of steps.
// The weights are scaled so they sum to 100%.
func CalculateWeightsForSteps(steps []string) map[string]float64 {
	if len(steps) == 0 {
		return StepWeights
	}
	// Calculate total weight of selected steps
	var total float64
	for _, step := range steps {
		if w, ok := StepWeights[step]; ok {
			total += w
		}
	}
	if total == 0 {
		return nil
	}
	// Normalize weights to sum to 100%
	normalized := make(map[string]float64, len(steps))
	for _, step := range steps {
		if w, ok := StepWeights[step]; ok {
			normalized[step] = (w / total) * 100
		}
	}
	return normalized
}

// StartSteps defines the steps to run when starting/resuming a stopped deployment.
// These skip setup steps (Caddy, firewall) and focus on starting services.
var StartSteps = []string{
	"scenario_stop",
	"secrets_provision",
	"resource_start",
	"scenario_deps",
	"scenario_target",
	"wait_for_ui",
	"verify_local",
	"verify_https",
	"verify_origin",
	"verify_public",
}

// NewProgressEvent creates a new ProgressEvent with the current timestamp.
func NewProgressEvent(eventType, stepID, stepTitle string, progress float64) ProgressEvent {
	return ProgressEvent{
		Type:      eventType,
		Step:      stepID,
		StepTitle: stepTitle,
		Progress:  progress,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// NewErrorEvent creates a ProgressEvent with error details.
func NewErrorEvent(stepID, stepTitle string, progress float64, errMsg string) ProgressEvent {
	return ProgressEvent{
		Type:      "deployment_error",
		Step:      stepID,
		StepTitle: stepTitle,
		Progress:  progress,
		Error:     errMsg,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}
