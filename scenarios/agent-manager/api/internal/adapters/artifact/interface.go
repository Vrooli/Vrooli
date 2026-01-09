// Package artifact provides the artifact collection and storage interface.
//
// This package defines the SEAM for artifact management. Artifacts include:
// - Diffs generated from sandbox changes
// - Validation results (linting, type checking, tests)
// - Logs from agent execution
// - Screenshots or other output files
//
// The interface allows for different storage backends (local filesystem,
// object storage, database) without changing the orchestration code.
package artifact

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
)

// -----------------------------------------------------------------------------
// Collector Interface - The primary seam for artifact management
// -----------------------------------------------------------------------------

// Collector manages artifact storage and retrieval for runs.
type Collector interface {
	// Store saves an artifact and returns its metadata.
	Store(ctx context.Context, req StoreRequest) (*Artifact, error)

	// Get retrieves artifact metadata by ID.
	Get(ctx context.Context, id uuid.UUID) (*Artifact, error)

	// Read returns a reader for artifact content.
	Read(ctx context.Context, id uuid.UUID) (io.ReadCloser, error)

	// List returns artifacts for a run.
	List(ctx context.Context, runID uuid.UUID, opts ListOptions) ([]*Artifact, error)

	// Delete removes an artifact.
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteByRun removes all artifacts for a run.
	DeleteByRun(ctx context.Context, runID uuid.UUID) error
}

// -----------------------------------------------------------------------------
// Types
// -----------------------------------------------------------------------------

// StoreRequest contains parameters for storing an artifact.
type StoreRequest struct {
	RunID       uuid.UUID
	Type        ArtifactType
	Name        string
	Content     io.Reader
	ContentSize int64
	ContentType string // MIME type
	Metadata    map[string]string
}

// Artifact represents a stored artifact.
type Artifact struct {
	ID          uuid.UUID         `json:"id"`
	RunID       uuid.UUID         `json:"runId"`
	Type        ArtifactType      `json:"type"`
	Name        string            `json:"name"`
	StoragePath string            `json:"storagePath"`
	ContentSize int64             `json:"contentSize"`
	ContentType string            `json:"contentType"`
	Checksum    string            `json:"checksum,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
}

// ArtifactType categorizes artifacts.
type ArtifactType string

const (
	ArtifactTypeDiff       ArtifactType = "diff"
	ArtifactTypeLog        ArtifactType = "log"
	ArtifactTypeValidation ArtifactType = "validation"
	ArtifactTypeScreenshot ArtifactType = "screenshot"
	ArtifactTypeSummary    ArtifactType = "summary"
	ArtifactTypeOther      ArtifactType = "other"
)

// ListOptions specifies criteria for listing artifacts.
type ListOptions struct {
	Type   *ArtifactType
	Limit  int
	Offset int
}

// -----------------------------------------------------------------------------
// Validation Runner Interface
// -----------------------------------------------------------------------------

// ValidationRunner executes validation checks on sandbox changes.
type ValidationRunner interface {
	// Run executes validation and returns results.
	Run(ctx context.Context, req ValidationRequest) (*ValidationResult, error)

	// SupportedTypes returns validation types this runner supports.
	SupportedTypes() []ValidationType
}

// ValidationRequest specifies what to validate.
type ValidationRequest struct {
	RunID        uuid.UUID
	WorkDir      string
	Types        []ValidationType
	FilePatterns []string // Limit to specific file patterns
}

// ValidationType identifies a validation check.
type ValidationType string

const (
	ValidationTypeLint      ValidationType = "lint"
	ValidationTypeTypeCheck ValidationType = "typecheck"
	ValidationTypeTest      ValidationType = "test"
	ValidationTypeBuild     ValidationType = "build"
	ValidationTypeSecurity  ValidationType = "security"
)

// ValidationResult contains validation outcomes.
type ValidationResult struct {
	Passed  bool              `json:"passed"`
	Results []ValidationCheck `json:"results"`
}

// ValidationCheck is a single validation result.
type ValidationCheck struct {
	Type     ValidationType `json:"type"`
	Passed   bool           `json:"passed"`
	Message  string         `json:"message,omitempty"`
	Errors   []string       `json:"errors,omitempty"`
	Warnings []string       `json:"warnings,omitempty"`
	Duration time.Duration  `json:"duration"`
}
