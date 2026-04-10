// DOC: docs/concepts/ARCHITECTURE.md#domain-modules
// DOC: docs/reference/data-model.md#validation
// DOC: docs/internal/SEAMS.md#change-axes
//
// Package validation runs structural and CLI tool validations against references.
//
// [REQ:REQ-P0-007] Structural Validation Engine
// [REQ:REQ-P0-008] CLI Tool Validation Engine
package validation

import (
	"time"

	"development-toolchain-validator/domain/expectation"
)

// ValidationStatus represents the outcome of a single validation check.
type ValidationStatus string

const (
	StatusPassed  ValidationStatus = "passed"
	StatusFailed  ValidationStatus = "failed"
	StatusSkipped ValidationStatus = "skipped"
	StatusError   ValidationStatus = "error"
)

// ExpectationResult holds the result of validating a single structural expectation.
//
// [REQ:REQ-P0-007] Structural Validation Engine
type ExpectationResult struct {
	ExpectationID string                       `json:"expectation_id"`
	Expectation   *expectation.StructuralExpectation `json:"expectation"`
	Status        ValidationStatus             `json:"status"`
	Message       string                       `json:"message,omitempty"`
	MatchedPaths  []string                     `json:"matched_paths,omitempty"`
	MissingPaths  []string                     `json:"missing_paths,omitempty"`
	ContentMatch  bool                         `json:"content_match,omitempty"`
	ValidatedAt   time.Time                    `json:"validated_at"`
}

// AssertionResult holds the result of validating a single CLI assertion.
//
// [REQ:REQ-P0-008] CLI Tool Validation Engine
type AssertionResult struct {
	AssertionID   string                    `json:"assertion_id"`
	Assertion     *expectation.CLIAssertion `json:"assertion"`
	Status        ValidationStatus          `json:"status"`
	Message       string                    `json:"message,omitempty"`
	ActualValue   interface{}               `json:"actual_value,omitempty"`
	ExpectedValue interface{}               `json:"expected_value,omitempty"`
	CommandOutput string                    `json:"command_output,omitempty"`
	CommandError  string                    `json:"command_error,omitempty"`
	ValidatedAt   time.Time                 `json:"validated_at"`
}

// ConnectionValidationResult aggregates all results for a skill connection.
type ConnectionValidationResult struct {
	ConnectionID         string              `json:"connection_id"`
	ReferenceID          string              `json:"reference_id"`
	SkillID              string              `json:"skill_id"`
	StructuralResults    []*ExpectationResult `json:"structural_results"`
	CLIResults           []*AssertionResult   `json:"cli_results"`
	StructuralPassCount  int                 `json:"structural_pass_count"`
	StructuralFailCount  int                 `json:"structural_fail_count"`
	CLIPassCount         int                 `json:"cli_pass_count"`
	CLIFailCount         int                 `json:"cli_fail_count"`
	OverallStatus        ValidationStatus    `json:"overall_status"`
	ValidatedAt          time.Time           `json:"validated_at"`
}

// ReferenceValidationReport is the comprehensive validation report for a reference.
// [REQ:REQ-P0-009] Validation Report API
type ReferenceValidationReport struct {
	ReferenceID         string                         `json:"reference_id"`
	ReferencePath       string                         `json:"reference_path"`
	ConnectionResults   []*ConnectionValidationResult  `json:"connection_results"`
	TotalConnections    int                            `json:"total_connections"`
	PassingConnections  int                            `json:"passing_connections"`
	FailingConnections  int                            `json:"failing_connections"`
	OverallStatus       ValidationStatus               `json:"overall_status"`
	ValidatedAt         time.Time                      `json:"validated_at"`
}

// ValidateOptions configures what to validate.
type ValidateOptions struct {
	ConnectionID      string `json:"connection_id,omitempty"`
	StructuralOnly    bool   `json:"structural_only,omitempty"`
	CLIOnly           bool   `json:"cli_only,omitempty"`
	ContinueOnFailure bool   `json:"continue_on_failure,omitempty"`
}
