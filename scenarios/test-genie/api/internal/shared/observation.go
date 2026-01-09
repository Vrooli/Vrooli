// Package shared provides common utilities used across test-genie packages.
package shared

// ObservationType represents the kind of observation.
// This is the canonical definition used across all packages.
type ObservationType int

const (
	ObservationSection ObservationType = iota
	ObservationSuccess
	ObservationWarning
	ObservationError
	ObservationInfo
	ObservationSkip
)

// String returns the string representation of the observation type.
func (t ObservationType) String() string {
	switch t {
	case ObservationSection:
		return "section"
	case ObservationSuccess:
		return "success"
	case ObservationWarning:
		return "warning"
	case ObservationError:
		return "error"
	case ObservationInfo:
		return "info"
	case ObservationSkip:
		return "skip"
	default:
		return "unknown"
	}
}

// ObservationLike is an interface that all observation types must satisfy.
// This enables generic conversion from package-specific types to phases.Observation.
type ObservationLike interface {
	GetType() ObservationType
	GetIcon() string
	GetMessage() string
}

// ObservationData holds the extracted data from an observation.
// This is used as an intermediate representation for conversion in the orchestrator.
type ObservationData struct {
	Type    ObservationType
	Icon    string
	Message string
}

// GetType returns the observation type.
func (o ObservationData) GetType() ObservationType { return o.Type }

// GetIcon returns the observation icon.
func (o ObservationData) GetIcon() string { return o.Icon }

// GetMessage returns the observation message.
func (o ObservationData) GetMessage() string { return o.Message }

// ExtractObservations converts a slice of Observations to ObservationLike interface slice.
// This is useful for generic observation handling in the orchestrator.
func ExtractObservations(obs []Observation) []ObservationLike {
	result := make([]ObservationLike, len(obs))
	for i, o := range obs {
		result[i] = o
	}
	return result
}
