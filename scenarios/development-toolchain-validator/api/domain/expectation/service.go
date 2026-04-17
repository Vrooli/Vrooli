// DOC: docs/internal/SEAMS.md#api-handlers--domain-services
// DOC: docs/reference/api-endpoints.md#expectations
// DOC: docs/reference/configuration.md#api-configuration
// DOC: docs/internal/UTILS_UNIFICATION_NOTES.md
package expectation

import (
	"context"
	"development-toolchain-validator/internal/config"
	"development-toolchain-validator/internal/validation"
	"errors"
)

var (
	// ErrNotFound indicates the requested expectation does not exist.
	ErrNotFound = errors.New("expectation not found")
	// ErrInvalidConnectionID indicates the connection ID is invalid or missing.
	ErrInvalidConnectionID = errors.New("invalid or missing connection ID")
	// ErrInvalidType indicates the expectation type is not valid.
	ErrInvalidType = errors.New("invalid expectation type")
	// ErrInvalidPattern indicates the pattern is empty or invalid.
	ErrInvalidPattern = errors.New("invalid pattern")
	// ErrInvalidOperator indicates the assertion operator is not valid.
	ErrInvalidOperator = errors.New("invalid assertion operator")
	// ErrInvalidCommand indicates the CLI command is empty or invalid.
	ErrInvalidCommand = errors.New("invalid command")
	// ErrInvalidJSONPath indicates the JSONPath expression is invalid.
	ErrInvalidJSONPath = errors.New("invalid JSONPath expression")
	// ErrDangerousCommand indicates the command contains dangerous patterns.
	ErrDangerousCommand = errors.New("command contains dangerous patterns")
)

// Note: Dangerous patterns and allowed commands moved to internal/validation package.
// See: validation.ValidateCommandSafety() and validation.IsCommandSafe()

// ServiceConfig holds configuration for the expectation service.
type ServiceConfig struct {
	Pagination config.PaginationConfig
}

// DefaultServiceConfig returns default configuration for the service.
func DefaultServiceConfig() ServiceConfig {
	cfg := config.DefaultConfig()
	return ServiceConfig{
		Pagination: cfg.Pagination,
	}
}

// Service provides business logic for expectation management.
// [REQ:REQ-P0-004] Structural Expectation Config
// [REQ:REQ-P0-005] CLI Tool Expectation Config
type Service struct {
	structuralRepo StructuralRepository
	cliRepo        CLIRepository
	config         ServiceConfig
}

// ServiceOption is a functional option for configuring the service.
type ServiceOption func(*Service)

// WithConfig sets the service configuration.
func WithConfig(cfg ServiceConfig) ServiceOption {
	return func(s *Service) {
		s.config = cfg
	}
}

// NewService creates a new expectation service with the given repositories.
func NewService(structuralRepo StructuralRepository, cliRepo CLIRepository, opts ...ServiceOption) *Service {
	s := &Service{
		structuralRepo: structuralRepo,
		cliRepo:        cliRepo,
		config:         DefaultServiceConfig(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// ValidateStructuralInput validates input without persisting.
// Returns nil if valid, or an error describing the validation failure.
func (s *Service) ValidateStructuralInput(input CreateStructuralInput) error {
	if input.ConnectionID == "" {
		return ErrInvalidConnectionID
	}

	if !isValidExpectationType(input.Type) {
		return ErrInvalidType
	}

	if input.Pattern == "" {
		return ErrInvalidPattern
	}

	// For content_snippet type, expected content is required
	if input.Type == TypeContentSnippet && input.ExpectedContent == "" {
		return ErrInvalidPattern
	}

	return nil
}

// CreateStructural validates input and creates a new structural expectation.
func (s *Service) CreateStructural(ctx context.Context, input CreateStructuralInput) (*StructuralExpectation, error) {
	if err := s.ValidateStructuralInput(input); err != nil {
		return nil, err
	}

	return s.structuralRepo.Create(ctx, input)
}

// GetStructuralByID retrieves a structural expectation by ID.
func (s *Service) GetStructuralByID(ctx context.Context, id string) (*StructuralExpectation, error) {
	return s.structuralRepo.GetByID(ctx, id)
}

// ListStructural retrieves structural expectations with optional filtering.
func (s *Service) ListStructural(ctx context.Context, opts ListOptions) ([]*StructuralExpectation, error) {
	opts.Limit = s.config.Pagination.ApplyPaginationLimit(opts.Limit)
	return s.structuralRepo.List(ctx, opts)
}

// DeleteStructural removes a structural expectation by ID.
func (s *Service) DeleteStructural(ctx context.Context, id string) error {
	return s.structuralRepo.Delete(ctx, id)
}

// DeleteStructuralByConnection removes all structural expectations for a connection.
func (s *Service) DeleteStructuralByConnection(ctx context.Context, connectionID string) error {
	return s.structuralRepo.DeleteByConnection(ctx, connectionID)
}

// ValidateCLIInput validates CLI assertion input without persisting.
// Returns nil if valid, or an error describing the validation failure.
func (s *Service) ValidateCLIInput(input CreateCLIInput) error {
	if input.ConnectionID == "" {
		return ErrInvalidConnectionID
	}

	if input.Command == "" {
		return ErrInvalidCommand
	}

	if err := validateCommand(input.Command); err != nil {
		return err
	}

	if input.JSONPath == "" {
		return ErrInvalidJSONPath
	}

	if !isValidJSONPath(input.JSONPath) {
		return ErrInvalidJSONPath
	}

	if !isValidOperator(input.Operator) {
		return ErrInvalidOperator
	}

	return nil
}

// CreateCLI validates input and creates a new CLI assertion.
func (s *Service) CreateCLI(ctx context.Context, input CreateCLIInput) (*CLIAssertion, error) {
	if err := s.ValidateCLIInput(input); err != nil {
		return nil, err
	}

	return s.cliRepo.Create(ctx, input)
}

// GetCLIByID retrieves a CLI assertion by ID.
func (s *Service) GetCLIByID(ctx context.Context, id string) (*CLIAssertion, error) {
	return s.cliRepo.GetByID(ctx, id)
}

// ListCLI retrieves CLI assertions with optional filtering.
func (s *Service) ListCLI(ctx context.Context, opts ListOptions) ([]*CLIAssertion, error) {
	opts.Limit = s.config.Pagination.ApplyPaginationLimit(opts.Limit)
	return s.cliRepo.List(ctx, opts)
}

// DeleteCLI removes a CLI assertion by ID.
func (s *Service) DeleteCLI(ctx context.Context, id string) error {
	return s.cliRepo.Delete(ctx, id)
}

// DeleteCLIByConnection removes all CLI assertions for a connection.
func (s *Service) DeleteCLIByConnection(ctx context.Context, connectionID string) error {
	return s.cliRepo.DeleteByConnection(ctx, connectionID)
}

// isValidExpectationType checks if the type is valid.
func isValidExpectationType(t ExpectationType) bool {
	switch t {
	case TypeFolder, TypeFile, TypeContentSnippet:
		return true
	default:
		return false
	}
}

// isValidOperator checks if the operator is valid.
func isValidOperator(op AssertionOperator) bool {
	switch op {
	case OpEq, OpNeq, OpGt, OpGte, OpLt, OpLte, OpExists, OpContains, OpMatches, OpBetween:
		return true
	default:
		return false
	}
}

// validateCommand checks if a command is safe to execute.
// Uses shared validation utilities from internal/validation package.
func validateCommand(cmd string) error {
	if !validation.IsCommandSafe(cmd) {
		return ErrDangerousCommand
	}
	return nil
}

// isValidJSONPath checks if the expression is a valid JSONPath.
// Uses shared validation utilities from internal/validation package.
func isValidJSONPath(path string) bool {
	return validation.IsValidJSONPath(path)
}
