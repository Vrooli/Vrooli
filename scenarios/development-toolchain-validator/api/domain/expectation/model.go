// DOC: docs/concepts/ARCHITECTURE.md#domain-modules
// DOC: docs/reference/data-model.md#expectations
// DOC: docs/internal/SEAMS.md#change-axes
//
// Package expectation defines the Expectation Configuration domain.
//
// # Purpose
//
// Expectations define what a steer skill expects from a reference scenario.
// There are two types:
//   - StructuralExpectation: Expected files, folders, and content patterns
//   - CLIAssertion: Expected outputs from CLI tools with JSON assertions
//
// # Why This Domain Exists
//
// Steer skills are prose guidance that agents interpret. To validate them
// programmatically, we need explicit expectations. This domain stores the
// declarative configuration that the validation engine will execute.
//
// # Domain Boundaries
//
// This domain handles:
//   - CRUD operations for structural expectations
//   - CRUD operations for CLI assertions
//   - Listing expectations by connection
//
// This domain does NOT:
//   - Execute validations (see: validation domain)
//   - Modify reference scenarios (read-only principle)
//   - Generate reports (see: report domain)
//
// [REQ:REQ-P0-004] Structural Expectation Config
// [REQ:REQ-P0-005] CLI Tool Expectation Config
package expectation

import (
	"time"
)

// ExpectationType defines the kind of structural expectation.
type ExpectationType string

const (
	TypeFolder         ExpectationType = "folder"
	TypeFile           ExpectationType = "file"
	TypeContentSnippet ExpectationType = "content_snippet"
)

// StructuralExpectation defines an expected file, folder, or content pattern.
//
// # Fields
//
//   - ID: UUID for database relationships
//   - ConnectionID: UUID of the skill connection this belongs to
//   - Type: folder, file, or content_snippet
//   - Pattern: Glob pattern for files/folders, or path for content
//   - Required: Whether this is a required (must exist) or optional expectation
//   - ExpectedContent: For content_snippet type, the expected text
//   - Description: Human-readable explanation of this expectation
//   - CreatedAt: Timestamp when created
//
// [REQ:REQ-P0-004] Structural Expectation Config
type StructuralExpectation struct {
	ID              string          `json:"id"`
	ConnectionID    string          `json:"connection_id"`
	Type            ExpectationType `json:"type"`
	Pattern         string          `json:"pattern"`
	Required        bool            `json:"required"`
	ExpectedContent string          `json:"expected_content,omitempty"`
	Description     string          `json:"description,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

// CreateStructuralInput contains the required fields for creating a structural expectation.
type CreateStructuralInput struct {
	ConnectionID    string          `json:"connection_id"`
	Type            ExpectationType `json:"type"`
	Pattern         string          `json:"pattern"`
	Required        bool            `json:"required"`
	ExpectedContent string          `json:"expected_content,omitempty"`
	Description     string          `json:"description,omitempty"`
}

// AssertionOperator defines the comparison operator for CLI assertions.
type AssertionOperator string

const (
	OpEq       AssertionOperator = "eq"
	OpNeq      AssertionOperator = "neq"
	OpGt       AssertionOperator = "gt"
	OpGte      AssertionOperator = "gte"
	OpLt       AssertionOperator = "lt"
	OpLte      AssertionOperator = "lte"
	OpExists   AssertionOperator = "exists"
	OpContains AssertionOperator = "contains"
	OpMatches  AssertionOperator = "matches"
	OpBetween  AssertionOperator = "between"
)

// CLIAssertion defines an assertion against CLI tool output.
//
// # Fields
//
//   - ID: UUID for database relationships
//   - ConnectionID: UUID of the skill connection this belongs to
//   - Command: The CLI command to execute (must be read-only with --json output)
//   - JSONPath: JSONPath expression to extract value from output
//   - Operator: Comparison operator (eq, neq, gt, etc.)
//   - ExpectedValue: The expected value (as JSON)
//   - Description: Human-readable explanation of this assertion
//   - CreatedAt: Timestamp when created
//
// [REQ:REQ-P0-005] CLI Tool Expectation Config
type CLIAssertion struct {
	ID            string            `json:"id"`
	ConnectionID  string            `json:"connection_id"`
	Command       string            `json:"command"`
	JSONPath      string            `json:"json_path"`
	Operator      AssertionOperator `json:"operator"`
	ExpectedValue interface{}       `json:"expected_value,omitempty"`
	Description   string            `json:"description,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
}

// CreateCLIInput contains the required fields for creating a CLI assertion.
type CreateCLIInput struct {
	ConnectionID  string            `json:"connection_id"`
	Command       string            `json:"command"`
	JSONPath      string            `json:"json_path"`
	Operator      AssertionOperator `json:"operator"`
	ExpectedValue interface{}       `json:"expected_value,omitempty"`
	Description   string            `json:"description,omitempty"`
}

// ListOptions contains filtering and pagination options for listing expectations.
type ListOptions struct {
	ConnectionID string `json:"connection_id,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	Offset       int    `json:"offset,omitempty"`
}
