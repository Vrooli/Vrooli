package shared

// ValidationCheckStatus represents the status of a validation check.
type ValidationCheckStatus string

const (
	ValidationStatusPass  ValidationCheckStatus = "pass"
	ValidationStatusWarn  ValidationCheckStatus = "warn"
	ValidationStatusError ValidationCheckStatus = "error"
	ValidationStatusInfo  ValidationCheckStatus = "info"
)

// ValidationCheck represents a single validation check result.
type ValidationCheck struct {
	ID          string                `json:"id"`
	Status      ValidationCheckStatus `json:"status"`
	Label       string                `json:"label"`
	Description string                `json:"description"`
	// Context provides additional information about the check (e.g., count of workflows found)
	Context map[string]any `json:"context,omitempty"`
}

// ValidationSummary provides overall validation status with all checks.
type ValidationSummary struct {
	OverallStatus ValidationCheckStatus `json:"overall_status"`
	PassCount     int                   `json:"pass_count"`
	WarnCount     int                   `json:"warn_count"`
	ErrorCount    int                   `json:"error_count"`
	InfoCount     int                   `json:"info_count"`
	Checks        []ValidationCheck     `json:"checks"`
}

// NewValidationSummary creates a new ValidationSummary with an empty checks slice.
func NewValidationSummary() *ValidationSummary {
	return &ValidationSummary{
		Checks: []ValidationCheck{},
	}
}

// AddCheck adds a validation check and updates counts.
func (v *ValidationSummary) AddCheck(check ValidationCheck) {
	v.Checks = append(v.Checks, check)
	switch check.Status {
	case ValidationStatusPass:
		v.PassCount++
	case ValidationStatusWarn:
		v.WarnCount++
	case ValidationStatusError:
		v.ErrorCount++
	case ValidationStatusInfo:
		v.InfoCount++
	}
}

// ComputeOverallStatus calculates the overall status from checks.
func (v *ValidationSummary) ComputeOverallStatus() {
	if v.ErrorCount > 0 {
		v.OverallStatus = ValidationStatusError
	} else if v.WarnCount > 0 {
		v.OverallStatus = ValidationStatusWarn
	} else {
		v.OverallStatus = ValidationStatusPass
	}
}
