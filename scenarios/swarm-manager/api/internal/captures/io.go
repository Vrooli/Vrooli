package captures

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Failure reason categories for capture classification.
// Each category implies a different recovery path.
const (
	// FailDependencyUnavailable means agent-manager or prompt-manager was not
	// running or not healthy. Transient — retry once the dependency is up.
	FailDependencyUnavailable = "dependency_unavailable"

	// FailClassificationTimeout means the classification agent did not finish
	// within the allowed window. Transient — retry should work.
	FailClassificationTimeout = "classification_timeout"

	// FailPromptMissing means the classification skill could not be resolved
	// from the prompt catalog. Configuration issue — check prompt-manager.
	FailPromptMissing = "prompt_missing"

	// FailAgentError means the agent spawn call itself failed for an
	// unexpected reason. Check agent-manager logs for details.
	FailAgentError = "agent_error"

	// FailInternal is a catch-all for unexpected server errors.
	FailInternal = "internal_error"
)

// capture represents the on-disk capture state.
type capture struct {
	ID            string   `json:"id"`
	Text          string   `json:"text"`
	Attachments   []string `json:"attachments"`
	Created       string   `json:"created"`
	Status        string   `json:"status"`
	FailureReason string   `json:"failure_reason,omitempty"`
	Note          string   `json:"note,omitempty"`
}

type classificationGoal struct {
	Name        string                        `json:"name"`
	Title       string                        `json:"title"`
	Description string                        `json:"description"`
	Priority    int                           `json:"priority"`
	Targets     []string                      `json:"targets"`
	Milestones  []classificationGoalMilestone `json:"milestones"`
}

type classificationGoalMilestone struct {
	Items              []string `json:"items,omitempty"`
	Name               string   `json:"name"`
	Title              string   `json:"title"`
	Description        string   `json:"description,omitempty"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	DependsOn          []string `json:"depends_on,omitempty"`
}

// classificationItem represents one suggested backlog item.
type classificationItem struct {
	Name            string   `json:"name,omitempty"`
	Kind            string   `json:"kind"`
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Priority        int      `json:"priority"`
	Tags            []string `json:"tags"`
	DependsOn       []string `json:"depends_on,omitempty"`
	Milestone       string   `json:"milestone,omitempty"`
	Effort          string   `json:"effort,omitempty"`
	AcceptanceAllow []string `json:"acceptance_allow,omitempty"`
	AcceptanceDeny  []string `json:"acceptance_deny,omitempty"`
	Confidence      float64  `json:"confidence"`
}

func (h *Handler) capturesDir() string {
	return filepath.Join(h.cacheRoot, "captures")
}

func (h *Handler) captureDir(id string) string {
	return filepath.Join(h.capturesDir(), id)
}

func (h *Handler) captureSpecPath(id string) string {
	return filepath.Join(h.captureDir(id), "capture.json")
}

// loadCapture reads the durable capture intake record from disk. Classification
// results are proposal payloads and are never persisted into the capture.
func (h *Handler) loadCapture(id string) (*capture, error) {
	data, err := os.ReadFile(h.captureSpecPath(id))
	if err != nil {
		return nil, err
	}
	var cap capture
	if err := json.Unmarshal(data, &cap); err != nil {
		return nil, fmt.Errorf("unmarshal capture %s: %w", id, err)
	}

	return &cap, nil
}

// writeCapture writes a capture to disk.
func (h *Handler) writeCapture(cap *capture) error {
	data, err := json.MarshalIndent(cap, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal capture: %w", err)
	}
	return os.WriteFile(h.captureSpecPath(cap.ID), data, 0o600)
}

// EnsureCapturesDir creates the captures directory if it doesn't exist.
func (h *Handler) EnsureCapturesDir() error {
	return os.MkdirAll(h.capturesDir(), 0o750)
}
