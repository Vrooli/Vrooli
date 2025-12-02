package types

import "time"

// Requirement represents a single requirement with its validations.
type Requirement struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Status      DeclaredStatus `json:"status"`
	PRDRef      string         `json:"prd_ref,omitempty"`
	Category    string         `json:"category,omitempty"`
	Criticality Criticality    `json:"criticality,omitempty"`
	Description string         `json:"description,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	Children    []string       `json:"children,omitempty"`
	DependsOn   []string       `json:"depends_on,omitempty"`
	Blocks      []string       `json:"blocks,omitempty"`
	Validations []Validation   `json:"validation,omitempty"`

	// Enriched fields (not persisted to requirement files)
	LiveStatus       LiveStatus       `json:"-"`
	AggregatedStatus AggregatedStatus `json:"-"`
	SourceFile       string           `json:"-"`
	SourceModule     string           `json:"-"`
}

// AggregatedStatus contains computed status information.
type AggregatedStatus struct {
	DeclaredRollup    DeclaredStatus
	LiveRollup        LiveStatus
	ValidationSummary ValidationSummary
}

// ValidationSummary counts validations by status.
type ValidationSummary struct {
	Total          int
	Implemented    int
	Failing        int
	Planned        int
	NotImplemented int
}

// Validation represents a test/automation reference for a requirement.
type Validation struct {
	Type       ValidationType   `json:"type"`
	Ref        string           `json:"ref,omitempty"`
	WorkflowID string           `json:"workflow_id,omitempty"`
	Phase      string           `json:"phase,omitempty"`
	Status     ValidationStatus `json:"status"`
	Notes      string           `json:"notes,omitempty"`
	Scenario   string           `json:"scenario,omitempty"`
	Folder     string           `json:"folder,omitempty"`
	Metadata   map[string]any   `json:"metadata,omitempty"`

	// Enriched fields (not persisted)
	LiveStatus  LiveStatus   `json:"-"`
	LiveDetails *LiveDetails `json:"-"`
}

// LiveDetails contains test execution metadata.
type LiveDetails struct {
	Timestamp       time.Time `json:"timestamp,omitempty"`
	DurationSeconds float64   `json:"duration_seconds,omitempty"`
	Evidence        string    `json:"evidence,omitempty"`
	SourcePath      string    `json:"source_path,omitempty"`
}

// Clone creates a deep copy of the requirement.
func (r *Requirement) Clone() *Requirement {
	if r == nil {
		return nil
	}

	clone := &Requirement{
		ID:           r.ID,
		Title:        r.Title,
		Status:       r.Status,
		PRDRef:       r.PRDRef,
		Category:     r.Category,
		Criticality:  r.Criticality,
		Description:  r.Description,
		LiveStatus:   r.LiveStatus,
		SourceFile:   r.SourceFile,
		SourceModule: r.SourceModule,
	}

	if len(r.Tags) > 0 {
		clone.Tags = make([]string, len(r.Tags))
		copy(clone.Tags, r.Tags)
	}
	if len(r.Children) > 0 {
		clone.Children = make([]string, len(r.Children))
		copy(clone.Children, r.Children)
	}
	if len(r.DependsOn) > 0 {
		clone.DependsOn = make([]string, len(r.DependsOn))
		copy(clone.DependsOn, r.DependsOn)
	}
	if len(r.Blocks) > 0 {
		clone.Blocks = make([]string, len(r.Blocks))
		copy(clone.Blocks, r.Blocks)
	}
	if len(r.Validations) > 0 {
		clone.Validations = make([]Validation, len(r.Validations))
		for i, v := range r.Validations {
			clone.Validations[i] = v.Clone()
		}
	}

	return clone
}

// HasValidations returns true if the requirement has any validations.
func (r *Requirement) HasValidations() bool {
	return r != nil && len(r.Validations) > 0
}

// HasChildren returns true if the requirement has child requirements.
func (r *Requirement) HasChildren() bool {
	return r != nil && len(r.Children) > 0
}

// IsComplete returns true if the requirement status is complete.
func (r *Requirement) IsComplete() bool {
	return r != nil && r.Status == StatusComplete
}

// IsCriticalReq returns true if the requirement has P0 or P1 criticality.
func (r *Requirement) IsCriticalReq() bool {
	return r != nil && IsCritical(r.Criticality)
}

// Clone creates a copy of the validation.
func (v Validation) Clone() Validation {
	clone := Validation{
		Type:       v.Type,
		Ref:        v.Ref,
		WorkflowID: v.WorkflowID,
		Phase:      v.Phase,
		Status:     v.Status,
		Notes:      v.Notes,
		Scenario:   v.Scenario,
		Folder:     v.Folder,
		LiveStatus: v.LiveStatus,
	}

	if v.Metadata != nil {
		clone.Metadata = make(map[string]any, len(v.Metadata))
		for k, val := range v.Metadata {
			clone.Metadata[k] = val
		}
	}

	if v.LiveDetails != nil {
		details := *v.LiveDetails
		clone.LiveDetails = &details
	}

	return clone
}

// Key returns a unique identifier for the validation within its requirement.
func (v *Validation) Key() string {
	if v.Ref != "" {
		return v.Ref
	}
	if v.WorkflowID != "" {
		return v.WorkflowID
	}
	return ""
}

// IsTest returns true if the validation is a test type.
func (v *Validation) IsTest() bool {
	return v.Type == ValTypeTest
}

// IsAutomation returns true if the validation is an automation type.
func (v *Validation) IsAutomation() bool {
	return v.Type == ValTypeAutomation
}

// IsManual returns true if the validation is a manual type.
func (v *Validation) IsManual() bool {
	return v.Type == ValTypeManual
}

// ComputeValidationSummary calculates summary statistics for validations.
func ComputeValidationSummary(validations []Validation) ValidationSummary {
	summary := ValidationSummary{
		Total: len(validations),
	}

	for _, v := range validations {
		switch v.Status {
		case ValStatusImplemented:
			summary.Implemented++
		case ValStatusFailing:
			summary.Failing++
		case ValStatusPlanned:
			summary.Planned++
		case ValStatusNotImplemented:
			summary.NotImplemented++
		}
	}

	return summary
}
