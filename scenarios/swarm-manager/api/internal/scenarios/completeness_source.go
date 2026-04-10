package scenarios

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"time"
)

// DOC: docs/reference/operational-targets.md
// DOC: docs/internal/INTEROP_AUDIT.md

const defaultCompletenessTimeout = 30 * time.Second

// CompletenessSource provides completeness scores for scenarios.
type CompletenessSource interface {
	Scores(ctx context.Context) (map[string]int, error)
}

// CLICompletenessSource fetches completeness scores via scenario-completeness-scoring CLI.
type CLICompletenessSource struct {
	timeout time.Duration
}

// NewCLICompletenessSource creates a CLI-backed completeness provider.
func NewCLICompletenessSource(timeout time.Duration) *CLICompletenessSource {
	if timeout <= 0 {
		timeout = defaultCompletenessTimeout
	}
	return &CLICompletenessSource{timeout: timeout}
}

// Scores retrieves completeness scores using `scenario-completeness-scoring scores --json`.
func (c *CLICompletenessSource) Scores(ctx context.Context) (map[string]int, error) {
	output, err := executeCommand(ctx, c.timeout, "scenario-completeness-scoring", "scores", "--json")
	if err != nil {
		return nil, err
	}

	var resp completenessScoresResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, err
	}

	scores := make(map[string]int, len(resp.Scenarios))
	for _, item := range resp.Scenarios {
		name := strings.TrimSpace(item.Scenario)
		if name == "" {
			continue
		}
		score := int(math.Round(item.Score))
		if score < 0 {
			score = 0
		} else if score > 100 {
			score = 100
		}
		scores[name] = score
	}

	return scores, nil
}

type completenessScoresResponse struct {
	Scenarios []completenessScoreItem `json:"scenarios"`
}

type completenessScoreItem struct {
	Scenario string  `json:"scenario"`
	Score    float64 `json:"score"`
}
