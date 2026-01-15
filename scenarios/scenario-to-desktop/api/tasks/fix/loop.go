package fix

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"scenario-to-desktop-api/domain"
	"scenario-to-desktop-api/tasks/shared"
)

// LoopConfig configures the iterative fix loop.
type LoopConfig struct {
	MaxIterations    int
	IterationTimeout time.Duration
	RebuildTimeout   time.Duration
	VerificationURL  string
	PipelineAPIURL   string
}

// DefaultLoopConfig returns sensible defaults for the loop configuration.
func DefaultLoopConfig(maxIterations int, pipelineAPIURL string) LoopConfig {
	if maxIterations <= 0 {
		maxIterations = 5
	}
	return LoopConfig{
		MaxIterations:    maxIterations,
		IterationTimeout: 15 * time.Minute,
		RebuildTimeout:   10 * time.Minute,
		PipelineAPIURL:   pipelineAPIURL,
	}
}

// LoopState tracks the state of a fix loop execution.
type LoopState struct {
	Config           LoopConfig
	CurrentIteration int
	Iterations       []domain.FixIterationRecord
	FinalStatus      string
}

// NewLoopState creates a new loop state with the given configuration.
func NewLoopState(config LoopConfig) *LoopState {
	return &LoopState{
		Config:           config,
		CurrentIteration: 0,
		Iterations:       make([]domain.FixIterationRecord, 0),
	}
}

// StartIteration begins a new iteration.
func (s *LoopState) StartIteration() int {
	s.CurrentIteration++
	return s.CurrentIteration
}

// RecordIteration adds an iteration record to the state.
func (s *LoopState) RecordIteration(record domain.FixIterationRecord) {
	s.Iterations = append(s.Iterations, record)
}

// ShouldContinue returns true if the loop should continue.
func (s *LoopState) ShouldContinue() bool {
	if s.CurrentIteration >= s.Config.MaxIterations {
		return false
	}
	if len(s.Iterations) == 0 {
		return true
	}
	lastIteration := s.Iterations[len(s.Iterations)-1]
	return lastIteration.Outcome == "continue"
}

// ToFixIterationState converts the loop state to the domain type for storage.
func (s *LoopState) ToFixIterationState() domain.FixIterationState {
	return domain.FixIterationState{
		CurrentIteration: s.CurrentIteration,
		MaxIterations:    s.Config.MaxIterations,
		Iterations:       s.Iterations,
		FinalStatus:      s.FinalStatus,
	}
}

// ToJSON serializes the fix state for storage in investigation details.
func (s *LoopState) ToJSON() ([]byte, error) {
	state := s.ToFixIterationState()
	return json.Marshal(state)
}

// CheckBuildArtifacts verifies that build artifacts exist.
func CheckBuildArtifacts(ctx context.Context, pipelineAPIURL, pipelineID string, timeout time.Duration) (bool, error) {
	if pipelineAPIURL == "" || pipelineID == "" {
		return false, nil // Can't check, skip
	}

	client := &http.Client{
		Timeout: timeout,
	}

	url := fmt.Sprintf("%s/api/v1/pipeline/%s", pipelineAPIURL, pipelineID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, nil // Connection error = not available, not an error
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	// Parse response to check for artifacts
	var status struct {
		Status         string            `json:"status"`
		FinalArtifacts map[string]string `json:"final_artifacts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return false, nil
	}

	// Check if status is completed and artifacts exist
	return status.Status == "completed" && len(status.FinalArtifacts) > 0, nil
}

// TerminationReason provides human-readable termination messages.
func TerminationReason(status string, iteration, maxIterations int) string {
	switch status {
	case shared.FixStatusSuccess:
		return fmt.Sprintf("Fix successful after %d iteration(s)", iteration)
	case shared.FixStatusMaxIterations:
		return fmt.Sprintf("Reached maximum iterations (%d) without success", maxIterations)
	case shared.FixStatusAgentGaveUp:
		return fmt.Sprintf("Agent determined issue cannot be fixed after %d iteration(s)", iteration)
	case shared.FixStatusUserStopped:
		return "Fix loop stopped by user"
	case shared.FixStatusTimeout:
		return fmt.Sprintf("Fix loop timed out after %d iteration(s)", iteration)
	default:
		return fmt.Sprintf("Fix loop terminated: %s", status)
	}
}

// DetermineOutcome analyzes the fix state and determines the final outcome.
func DetermineOutcome(state *LoopState, lastResult *shared.AgentResult, artifactsExist bool) string {
	// Check artifacts first
	if artifactsExist {
		return shared.FixStatusSuccess
	}

	// Check agent's reported outcome
	if lastResult != nil {
		report, _ := ParseIterationReport(lastResult.Output)
		if report != nil {
			switch report.Outcome {
			case "success":
				return shared.FixStatusSuccess
			case "gave_up":
				return shared.FixStatusAgentGaveUp
			}
		}
	}

	// Check if we hit max iterations
	if state.CurrentIteration >= state.Config.MaxIterations {
		return shared.FixStatusMaxIterations
	}

	// Still running (shouldn't reach here normally)
	return ""
}
