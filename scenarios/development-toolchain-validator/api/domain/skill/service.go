// DOC: docs/concepts/ARCHITECTURE.md#skill-connection-flow
// DOC: docs/internal/SEAMS.md#api-handlers--domain-services
// DOC: docs/reference/api-endpoints.md#skill-connections
// DOC: docs/reference/configuration.md#api-configuration
// DOC: docs/internal/UTILS_UNIFICATION_NOTES.md
package skill

import (
	"context"
	"development-toolchain-validator/internal/config"
	"development-toolchain-validator/internal/validation"
	"errors"
	"fmt"
)

var (
	// ErrNotFound indicates the requested connection does not exist.
	ErrNotFound = errors.New("connection not found")
	// ErrInvalidSkillID indicates the skill ID format is invalid.
	ErrInvalidSkillID = errors.New("invalid skill ID format")
	// ErrConnectionExists indicates a connection already exists for this reference-skill pair.
	ErrConnectionExists = errors.New("connection already exists for this reference-skill pair")
	// ErrInvalidReferenceID indicates the reference ID is invalid or missing.
	ErrInvalidReferenceID = errors.New("invalid or missing reference ID")
)

// Note: Skill ID validation moved to internal/validation package.
// See: validation.IsValidSkillIDFormat()

// ServiceConfig holds configuration for the skill service.
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

// Service provides business logic for skill connection management.
// It coordinates between the repository (storage) and external validations.
// [REQ:REQ-P0-003] Prompt-Manager Skill Connection Store
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

// NewService creates a new skill service with the given repository.
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

// ValidateConnect validates input without persisting.
// Returns nil if valid, or an error describing the validation failure.
// Used by dry-run requests to verify input without side effects.
func (s *Service) ValidateConnect(ctx context.Context, input ConnectInput) error {
	// Validate reference ID
	if input.ReferenceID == "" {
		return ErrInvalidReferenceID
	}

	// Validate skill ID format
	if !s.isValidSkillID(input.SkillID) {
		return ErrInvalidSkillID
	}

	// Check if connection already exists
	existing, err := s.repo.GetByReferenceAndSkill(ctx, input.ReferenceID, input.SkillID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return fmt.Errorf("checking connection existence: %w", err)
	}
	if existing != nil {
		return ErrConnectionExists
	}

	return nil
}

// Connect validates input and creates a new skill-reference connection.
func (s *Service) Connect(ctx context.Context, input ConnectInput) (*Connection, error) {
	// Use ValidateConnect for validation
	if err := s.ValidateConnect(ctx, input); err != nil {
		return nil, err
	}

	return s.repo.Connect(ctx, input)
}

// GetByID retrieves a connection by its ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Connection, error) {
	return s.repo.GetByID(ctx, id)
}

// GetByReferenceAndSkill retrieves a connection by reference and skill IDs.
func (s *Service) GetByReferenceAndSkill(ctx context.Context, referenceID, skillID string) (*Connection, error) {
	return s.repo.GetByReferenceAndSkill(ctx, referenceID, skillID)
}

// List retrieves connections with optional filtering.
func (s *Service) List(ctx context.Context, opts ListOptions) ([]*Connection, error) {
	// Apply pagination constraints from configuration
	opts.Limit = s.config.Pagination.ApplyPaginationLimit(opts.Limit)
	return s.repo.List(ctx, opts)
}

// ValidateUpdate validates update input without persisting.
// Returns nil if valid, or an error describing the validation failure.
func (s *Service) ValidateUpdate(ctx context.Context, id string, input UpdateInput) error {
	// Check if connection exists
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// No additional validation needed for version/hash updates
	return nil
}

// Update modifies an existing connection (e.g., to refresh version/hash).
func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (*Connection, error) {
	// Use ValidateUpdate for validation
	if err := s.ValidateUpdate(ctx, id, input); err != nil {
		return nil, err
	}

	return s.repo.Update(ctx, id, input)
}

// Disconnect removes a skill-reference connection by ID.
func (s *Service) Disconnect(ctx context.Context, id string) error {
	return s.repo.Disconnect(ctx, id)
}

// DisconnectByReferenceAndSkill removes a connection by reference and skill IDs.
func (s *Service) DisconnectByReferenceAndSkill(ctx context.Context, referenceID, skillID string) error {
	return s.repo.DisconnectByReferenceAndSkill(ctx, referenceID, skillID)
}

// CheckDrift compares a connection's stored version/hash against current values.
// The currentVersion and currentHash should be fetched from prompt-manager API.
func (s *Service) CheckDrift(ctx context.Context, id string, currentVersion, currentHash string) (*DriftStatus, error) {
	conn, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	versionChanged := conn.SkillVersion != currentVersion
	contentChanged := conn.SkillContentHash != currentHash
	hasDrifted := versionChanged || contentChanged

	return &DriftStatus{
		ConnectionID:   conn.ID,
		SkillID:        conn.SkillID,
		StoredVersion:  conn.SkillVersion,
		StoredHash:     conn.SkillContentHash,
		CurrentVersion: currentVersion,
		CurrentHash:    currentHash,
		HasDrifted:     hasDrifted,
		VersionChanged: versionChanged,
		ContentChanged: contentChanged,
	}, nil
}

// isValidSkillID checks if a skill ID meets format requirements.
// Uses shared validation utilities from internal/validation package.
func (s *Service) isValidSkillID(skillID string) bool {
	if !validation.IsLengthInRange(skillID, 2, 100) {
		return false
	}
	return validation.IsValidSkillIDFormat(skillID)
}
