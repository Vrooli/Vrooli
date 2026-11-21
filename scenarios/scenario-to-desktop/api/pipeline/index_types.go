package pipeline

// ScenarioIndex tracks pipeline state for a specific scenario.
// Each scenario has exactly one active pipeline at a time, with history
// of previous pipelines for reference.
type ScenarioIndex struct {
	// ScenarioName is the scenario this index tracks.
	ScenarioName string `json:"scenario_name"`

	// ActivePipelineID is the currently active pipeline for this scenario.
	// Empty if no active pipeline exists.
	ActivePipelineID string `json:"active_pipeline_id,omitempty"`

	// History contains IDs of previous pipelines, newest first.
	// Limited to MaxHistorySize entries.
	History []string `json:"history"`

	// MaxHistorySize limits how many historical pipelines to track.
	// Defaults to 10 if not set.
	MaxHistorySize int `json:"max_history_size"`

	// UpdatedAt is the Unix timestamp when this index was last modified.
	UpdatedAt int64 `json:"updated_at"`
}

// DefaultMaxHistorySize is the default number of historical pipelines to keep.
const DefaultMaxHistorySize = 10

// GetMaxHistorySize returns the max history size, using default if not set.
func (s *ScenarioIndex) GetMaxHistorySize() int {
	if s.MaxHistorySize <= 0 {
		return DefaultMaxHistorySize
	}
	return s.MaxHistorySize
}

// AddToHistory adds a pipeline ID to history, maintaining size limit.
// The newest entries are at the front of the slice.
func (s *ScenarioIndex) AddToHistory(pipelineID string) {
	if pipelineID == "" {
		return
	}

	// Add to front
	s.History = append([]string{pipelineID}, s.History...)

	// Trim to max size
	maxSize := s.GetMaxHistorySize()
	if len(s.History) > maxSize {
		s.History = s.History[:maxSize]
	}
}
