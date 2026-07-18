package validation

import "workflow-health/internal/workflows"

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

const (
	CodeSurfaceAbsent            = "workflow.surface_absent"
	CodeRegistryMissing          = "workflow.registry_missing"
	CodeRegistryStale            = "workflow.registry_stale"
	CodeParseError               = "workflow.parse_error"
	CodeMetadataIncomplete       = "workflow.metadata_incomplete"
	CodeRequirementUnlinked      = "workflow.requirement_unlinked"
	CodeSelectorUnregistered     = "workflow.selector_unregistered"
	CodeSubflowUnresolved        = "workflow.subflow_unresolved"
	CodeExecutionModeInvalid     = "workflow.execution_mode_invalid"
	CodeResetLegacy              = "workflow.reset_legacy"
	CodeMutatingSafety           = "workflow.mutating_safety_missing"
	CodeSeedMissing              = "workflow.seed_missing"
	CodeExecutionRefused         = "workflow.execution_refused"
	CodeExecutionFailed          = "workflow.execution_failed"
	CodeExperienceUnavailable    = "workflow.experience_profile_unavailable"
	CodeExperienceRouteMissing   = "workflow.experience_route_undeclared"
	CodeExperienceBindingMissing = "workflow.experience_region_uncovered"
	CodeExperienceStateMissing   = "workflow.experience_lifecycle_uncovered"
)

var codeSeverity = map[string]Severity{
	CodeSurfaceAbsent:            SeverityError,
	CodeRegistryMissing:          SeverityWarning,
	CodeRegistryStale:            SeverityWarning,
	CodeParseError:               SeverityError,
	CodeMetadataIncomplete:       SeverityWarning,
	CodeRequirementUnlinked:      SeverityError,
	CodeSelectorUnregistered:     SeverityWarning,
	CodeSubflowUnresolved:        SeverityError,
	CodeExecutionModeInvalid:     SeverityError,
	CodeResetLegacy:              SeverityWarning,
	CodeMutatingSafety:           SeverityError,
	CodeSeedMissing:              SeverityError,
	CodeExecutionRefused:         SeverityError,
	CodeExecutionFailed:          SeverityError,
	CodeExperienceUnavailable:    SeverityWarning,
	CodeExperienceRouteMissing:   SeverityError,
	CodeExperienceBindingMissing: SeverityError,
	CodeExperienceStateMissing:   SeverityError,
}

type Finding struct {
	Code             string
	Severity         Severity
	Title            string
	Description      string
	FilePath         string
	AssetID          string
	AutofixAvailable bool
	Remediation      string
}

type Report struct {
	Scenario       string
	TargetPath     string
	Catalog        *workflows.ScenarioWorkflowCatalog
	Findings       []Finding
	DegradedReason string
}
