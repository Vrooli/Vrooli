// DOC: docs/concepts/ARCHITECTURE.md#registration-flow
// DOC: docs/internal/SEAMS.md#api-handlers--domain-services
// DOC: docs/reference/api-endpoints.md#references
// DOC: docs/reference/configuration.md#api-configuration
// DOC: docs/internal/UTILS_UNIFICATION_NOTES.md
package reference

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"development-toolchain-validator/internal/config"
	"development-toolchain-validator/internal/validation"
)

var (
	// ErrNotFound indicates the requested reference does not exist.
	ErrNotFound = errors.New("reference not found")
	// ErrInvalidSlug indicates the slug format is invalid.
	ErrInvalidSlug = errors.New("invalid slug format")
	// ErrSlugExists indicates a reference with this slug already exists.
	ErrSlugExists = errors.New("reference with this slug already exists")
	// ErrPathNotExists indicates the scenario path does not exist on disk.
	ErrPathNotExists = errors.New("scenario path does not exist")
)

// Note: Slug format validation moved to internal/validation package.
// See: validation.IsValidSlugFormat()
// See: docs/internal/SEAMS.md#decision-slug-format-validation

// ServiceConfig holds configuration for the reference service.
// These levers are documented in docs/reference/configuration.md.
type ServiceConfig struct {
	Pagination config.PaginationConfig
	Validation config.ValidationConfig
}

// DefaultServiceConfig returns default configuration for the service.
func DefaultServiceConfig() ServiceConfig {
	cfg := config.DefaultConfig()
	return ServiceConfig{
		Pagination: cfg.Pagination,
		Validation: cfg.Validation,
	}
}

// Service provides business logic for reference scenario management.
// It coordinates between the repository (storage) and external validations.
// [REQ:REQ-P0-001] Reference Scenario Database Schema
// [REQ:REQ-P0-002] Reference Scenario API Endpoints
type Service struct {
	repo   Repository
	config ServiceConfig
}

// ServiceOption is a functional option for configuring the service.
type ServiceOption func(*Service)

// WithConfig sets the service configuration.
func WithConfig(cfg ServiceConfig) ServiceOption {
	return func(s *Service) {
		s.config = cfg
	}
}

// NewService creates a new reference service with the given repository.
// Options can be provided to customize behavior.
func NewService(repo Repository, opts ...ServiceOption) *Service {
	s := &Service{
		repo:   repo,
		config: DefaultServiceConfig(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// ValidateCreate validates input without persisting.
// Returns the normalized path if valid, or an error.
// Used by dry-run requests to verify input without side effects.
func (s *Service) ValidateCreate(ctx context.Context, input CreateInput) (string, error) {
	// Validate slug format
	if !s.isValidSlug(input.Slug) {
		return "", ErrInvalidSlug
	}

	// Check if slug already exists
	existing, err := s.repo.GetBySlug(ctx, input.Slug)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return "", fmt.Errorf("checking slug existence: %w", err)
	}
	if existing != nil {
		return "", ErrSlugExists
	}

	// Validate path exists
	absPath, err := filepath.Abs(input.Path)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return "", ErrPathNotExists
	}

	return absPath, nil
}

// Create validates input and creates a new reference scenario.
func (s *Service) Create(ctx context.Context, input CreateInput) (*Reference, error) {
	// Use ValidateCreate for validation
	absPath, err := s.ValidateCreate(ctx, input)
	if err != nil {
		return nil, err
	}

	// Normalize path
	input.Path = absPath

	return s.repo.Create(ctx, input)
}

// GetByID retrieves a reference by its ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Reference, error) {
	return s.repo.GetByID(ctx, id)
}

// GetBySlug retrieves a reference by its slug.
func (s *Service) GetBySlug(ctx context.Context, slug string) (*Reference, error) {
	return s.repo.GetBySlug(ctx, slug)
}

// List retrieves references with optional filtering.
func (s *Service) List(ctx context.Context, opts ListOptions) ([]*Reference, error) {
	// Apply pagination constraints from configuration
	opts.Limit = s.config.Pagination.ApplyPaginationLimit(opts.Limit)
	return s.repo.List(ctx, opts)
}

// ValidateUpdate validates update input without persisting.
// Returns the normalized path (if provided) or empty string, plus any validation error.
// Used by dry-run requests to verify input without side effects.
func (s *Service) ValidateUpdate(ctx context.Context, id string, input UpdateInput) (string, error) {
	// Check if reference exists
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	// Validate new path if provided
	if input.Path != nil {
		absPath, err := filepath.Abs(*input.Path)
		if err != nil {
			return "", fmt.Errorf("resolving path: %w", err)
		}
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			return "", ErrPathNotExists
		}
		return absPath, nil
	}

	return "", nil
}

// Update modifies an existing reference.
func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (*Reference, error) {
	// Use ValidateUpdate for validation
	absPath, err := s.ValidateUpdate(ctx, id, input)
	if err != nil {
		return nil, err
	}

	// Normalize path if provided
	if absPath != "" {
		input.Path = &absPath
	}

	return s.repo.Update(ctx, id, input)
}

// Delete removes a reference by ID.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// isValidSlug checks if a slug meets format and length requirements.
// Length constraints are configurable via ServiceConfig.
// Uses shared validation utilities from internal/validation package.
func (s *Service) isValidSlug(slug string) bool {
	if !s.config.Validation.IsValidSlugLength(len(slug)) {
		return false
	}
	return validation.IsValidSlugFormat(slug)
}
