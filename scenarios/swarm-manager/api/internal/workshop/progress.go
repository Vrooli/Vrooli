package operatingmode

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ProgressDecision string

const (
	ProgressContinue ProgressDecision = "continue"
	ProgressBlocked  ProgressDecision = "blocked"
	ProgressReplan   ProgressDecision = "replan"
	ProgressComplete ProgressDecision = "complete"
)

type ProgressState struct {
	Decision        ProgressDecision `json:"decision"`
	CompletedPhases []string         `json:"completed_phases,omitempty"`
	CurrentPhase    string           `json:"current_phase,omitempty"`
	Rationale       string           `json:"rationale,omitempty"`
	UpdatedAt       string           `json:"updated_at,omitempty"`
}

func ParseProgressState(data []byte) (ProgressState, error) {
	var state ProgressState
	if err := json.Unmarshal(data, &state); err != nil {
		return ProgressState{}, err
	}
	if err := state.Validate(); err != nil {
		return ProgressState{}, err
	}
	return state, nil
}

func (s ProgressState) Validate() error {
	return s.Decision.Validate()
}

func (d ProgressDecision) Validate() error {
	switch ProgressDecision(strings.TrimSpace(string(d))) {
	case ProgressContinue, ProgressBlocked, ProgressReplan, ProgressComplete:
		return nil
	default:
		return fmt.Errorf("progress decision must be continue, blocked, replan, or complete")
	}
}
