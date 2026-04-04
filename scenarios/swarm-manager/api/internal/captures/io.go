package captures

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// capture represents the on-disk capture state.
type capture struct {
	ID             string          `json:"id"`
	Text           string          `json:"text"`
	Attachments    []string        `json:"attachments"`
	Created        string          `json:"created"`
	Status         string          `json:"status"`
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

func (h *Handler) capturesDir() string {
	return filepath.Join(h.rootDir, "captures")
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

	// Auto-fail captures stuck in classifying for > 2 minutes.
	if cap.Status == "classifying" {
		created, parseErr := time.Parse(time.RFC3339, cap.Created)
		if parseErr == nil && time.Since(created) > 2*time.Minute {
			cap.Status = "failed"
			_ = h.writeCapture(&cap)
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
	return os.WriteFile(h.captureSpecPath(cap.ID), data, 0o644)
}

// EnsureCapturesDir creates the captures directory if it doesn't exist.
func (h *Handler) EnsureCapturesDir() error {
	return os.MkdirAll(h.capturesDir(), 0o755)
}
