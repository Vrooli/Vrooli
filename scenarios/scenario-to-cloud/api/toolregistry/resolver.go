// Package toolregistry provides deployment resolution for scenario-to-cloud.
//
// This file implements the DeploymentResolver interface which resolves
// deployment identifiers (either name or UUID) to UUIDs.
package toolregistry

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"scenario-to-cloud/domain"
)

// DeploymentResolver resolves deployment identifiers (name or UUID) to UUIDs.
// This abstraction enables testability by allowing mock implementations.
type DeploymentResolver interface {
	// Resolve accepts either a deployment name or UUID and returns the UUID.
	// Returns error if not found or if the name is ambiguous (matches multiple deployments).
	Resolve(ctx context.Context, nameOrID string) (string, error)
}

// DeploymentRepository is the interface for querying deployments.
// This matches the methods needed from persistence.Repository.
type DeploymentRepository interface {
	GetDeployment(ctx context.Context, id string) (*domain.Deployment, error)
	ListDeployments(ctx context.Context, filter domain.ListFilter) ([]*domain.Deployment, error)
}

// ResolverImpl implements DeploymentResolver using a repository.
type ResolverImpl struct {
	repo DeploymentRepository
}

// NewResolver creates a new DeploymentResolver.
func NewResolver(repo DeploymentRepository) *ResolverImpl {
	return &ResolverImpl{repo: repo}
}

// Resolve accepts either a deployment name or UUID and returns the UUID.
// Returns error if not found or if the name is ambiguous.
func (r *ResolverImpl) Resolve(ctx context.Context, nameOrID string) (string, error) {
	if nameOrID == "" {
		return "", fmt.Errorf("deployment identifier is required")
	}

	// Fast path: if it looks like a UUID, try direct lookup
	if isUUID(nameOrID) {
		deployment, err := r.repo.GetDeployment(ctx, nameOrID)
		if err != nil {
			return "", fmt.Errorf("failed to get deployment: %w", err)
		}
		if deployment != nil {
			return deployment.ID, nil
		}
		// UUID format but not found - fall through to name search
	}

	// Try to find by name
	deployments, err := r.repo.ListDeployments(ctx, domain.ListFilter{})
	if err != nil {
		return "", fmt.Errorf("failed to list deployments: %w", err)
	}

	var matches []*domain.Deployment
	for _, d := range deployments {
		if d.Name == nameOrID {
			matches = append(matches, d)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("deployment not found: %s", nameOrID)
	case 1:
		return matches[0].ID, nil
	default:
		return "", fmt.Errorf("ambiguous deployment name %q matches %d deployments; use UUID instead", nameOrID, len(matches))
	}
}

// isUUID checks if a string is a valid UUID.
func isUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

// MustResolve is a convenience method that panics on error.
// Use only in tests or when the deployment is known to exist.
func (r *ResolverImpl) MustResolve(ctx context.Context, nameOrID string) string {
	id, err := r.Resolve(ctx, nameOrID)
	if err != nil {
		panic(fmt.Sprintf("MustResolve failed: %v", err))
	}
	return id
}
