package captures

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"swarm-manager/internal/attempt"
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
	ID             string          `json:"id"`
	Text           string          `json:"text"`
	Attachments    []string        `json:"attachments"`
	Created        string          `json:"created"`
	Status         string          `json:"status"`
	FailureReason  string          `json:"failure_reason,omitempty"`
	Classification *classification `json:"classification,omitempty"`
	Note           string          `json:"note,omitempty"`
}

// classification represents the AI-generated classification result.
type classification struct {
	Items        []classificationItem `json:"items"`
	ClassifiedAt string               `json:"classified_at"`
}

// classificationItem represents one suggested backlog item.
type classificationItem struct {
	Kind        string   `json:"kind"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    int      `json:"priority"`
	Tags        []string `json:"tags"`
	Confidence  float64  `json:"confidence"`
}

// asAttempt gives capture classification the same read-model shape as every
// other agentic feature. Classification items are proposals; applying them is
// still owned by the capture mutation boundary.
func (c capture) asAttempt() attempt.Attempt {
	proposals := make([]attempt.Proposal, 0)
	generatedAt := c.Created
	if c.Classification != nil {
		generatedAt = c.Classification.ClassifiedAt
		for index, item := range c.Classification.Items {
			payload, _ := json.Marshal(item)
			proposals = append(proposals, attempt.Proposal{ID: fmt.Sprintf("classification-%d", index+1), Type: item.Kind, Payload: string(payload)})
		}
	}
	return attempt.Attempt{SubjectKind: "capture", SubjectRef: c.ID, TransitionKey: "capture.classify", RoundNum: 1, Status: c.Status, GeneratedAt: generatedAt, Assessment: c.Note, Proposals: proposals}
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

func (h *Handler) classificationPath(id string) string {
	return filepath.Join(h.captureDir(id), "classification.json")
}

// loadCapture reads a capture from disk, merging classification.json if present.
func (h *Handler) loadCapture(id string) (*capture, error) {
	data, err := os.ReadFile(h.captureSpecPath(id))
	if err != nil {
		return nil, err
	}
	var cap capture
	if err := json.Unmarshal(data, &cap); err != nil {
		return nil, fmt.Errorf("unmarshal capture %s: %w", id, err)
	}

	// Merge classification.json if it exists.
	classData, err := os.ReadFile(h.classificationPath(id))
	if err == nil {
		var cls classification
		if err := json.Unmarshal(classData, &cls); err == nil {
			cap.Classification = &cls
			if cap.Status == "classifying" {
				cap.Status = "classified"
			}
		}
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
