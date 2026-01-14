package toolregistry

import (
	"context"
	"errors"
	"testing"

	"scenario-to-cloud/domain"
)

// mockDeploymentRepository is a test double for DeploymentRepository.
type mockDeploymentRepository struct {
	deployments map[string]*domain.Deployment
	listError   error
	getError    error
}

func newMockRepo() *mockDeploymentRepository {
	return &mockDeploymentRepository{
		deployments: make(map[string]*domain.Deployment),
	}
}

func (m *mockDeploymentRepository) GetDeployment(_ context.Context, id string) (*domain.Deployment, error) {
	if m.getError != nil {
		return nil, m.getError
	}
	return m.deployments[id], nil
}

func (m *mockDeploymentRepository) ListDeployments(_ context.Context, _ domain.ListFilter) ([]*domain.Deployment, error) {
	if m.listError != nil {
		return nil, m.listError
	}
	var result []*domain.Deployment
	for _, d := range m.deployments {
		result = append(result, d)
	}
	return result, nil
}

func (m *mockDeploymentRepository) addDeployment(d *domain.Deployment) {
	m.deployments[d.ID] = d
}

func TestResolver_ResolveByUUID(t *testing.T) {
	repo := newMockRepo()
	repo.addDeployment(&domain.Deployment{
		ID:   "550e8400-e29b-41d4-a716-446655440000",
		Name: "test-deployment",
	})

	resolver := NewResolver(repo)
	ctx := context.Background()

	// Resolve by UUID
	id, err := resolver.Resolve(ctx, "550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("expected UUID '550e8400-e29b-41d4-a716-446655440000', got %s", id)
	}
}

func TestResolver_ResolveByName(t *testing.T) {
	repo := newMockRepo()
	repo.addDeployment(&domain.Deployment{
		ID:   "550e8400-e29b-41d4-a716-446655440000",
		Name: "my-production-deploy",
	})

	resolver := NewResolver(repo)
	ctx := context.Background()

	// Resolve by name
	id, err := resolver.Resolve(ctx, "my-production-deploy")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("expected UUID '550e8400-e29b-41d4-a716-446655440000', got %s", id)
	}
}

func TestResolver_ResolveEmpty(t *testing.T) {
	repo := newMockRepo()
	resolver := NewResolver(repo)
	ctx := context.Background()

	// Empty identifier should error
	_, err := resolver.Resolve(ctx, "")
	if err == nil {
		t.Error("expected error for empty identifier")
	}
}

func TestResolver_ResolveNotFound(t *testing.T) {
	repo := newMockRepo()
	resolver := NewResolver(repo)
	ctx := context.Background()

	// Non-existent deployment
	_, err := resolver.Resolve(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent deployment")
	}
}

func TestResolver_ResolveAmbiguous(t *testing.T) {
	repo := newMockRepo()
	// Two deployments with the same name
	repo.addDeployment(&domain.Deployment{
		ID:   "550e8400-e29b-41d4-a716-446655440001",
		Name: "duplicate-name",
	})
	repo.addDeployment(&domain.Deployment{
		ID:   "550e8400-e29b-41d4-a716-446655440002",
		Name: "duplicate-name",
	})

	resolver := NewResolver(repo)
	ctx := context.Background()

	// Ambiguous name should error
	_, err := resolver.Resolve(ctx, "duplicate-name")
	if err == nil {
		t.Error("expected error for ambiguous name")
	}
}

func TestResolver_ResolveUUIDFormatButNotFound(t *testing.T) {
	repo := newMockRepo()
	// Add a deployment with a name matching a UUID format
	repo.addDeployment(&domain.Deployment{
		ID:   "111e8400-e29b-41d4-a716-446655440000",
		Name: "550e8400-e29b-41d4-a716-446655440000", // Name is a UUID
	})

	resolver := NewResolver(repo)
	ctx := context.Background()

	// Should find by name even if it looks like a UUID
	id, err := resolver.Resolve(ctx, "550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return the ID of the deployment found by name
	if id != "111e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("expected UUID '111e8400-e29b-41d4-a716-446655440000', got %s", id)
	}
}

func TestResolver_GetDeploymentError(t *testing.T) {
	repo := newMockRepo()
	repo.getError = errors.New("database error")

	resolver := NewResolver(repo)
	ctx := context.Background()

	// Should return error when GetDeployment fails
	_, err := resolver.Resolve(ctx, "550e8400-e29b-41d4-a716-446655440000")
	if err == nil {
		t.Error("expected error when GetDeployment fails")
	}
}

func TestResolver_ListDeploymentsError(t *testing.T) {
	repo := newMockRepo()
	repo.listError = errors.New("database error")

	resolver := NewResolver(repo)
	ctx := context.Background()

	// Should return error when ListDeployments fails (for name lookup)
	_, err := resolver.Resolve(ctx, "some-name")
	if err == nil {
		t.Error("expected error when ListDeployments fails")
	}
}

func TestResolver_MustResolve(t *testing.T) {
	repo := newMockRepo()
	repo.addDeployment(&domain.Deployment{
		ID:   "550e8400-e29b-41d4-a716-446655440000",
		Name: "test-deployment",
	})

	resolver := NewResolver(repo)
	ctx := context.Background()

	// Should not panic for valid deployment
	id := resolver.MustResolve(ctx, "test-deployment")
	if id != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("expected UUID '550e8400-e29b-41d4-a716-446655440000', got %s", id)
	}
}

func TestResolver_MustResolvePanics(t *testing.T) {
	repo := newMockRepo()
	resolver := NewResolver(repo)
	ctx := context.Background()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected MustResolve to panic for non-existent deployment")
		}
	}()

	// Should panic for non-existent deployment
	_ = resolver.MustResolve(ctx, "nonexistent")
}

func TestIsUUID(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"550e8400-e29b-41d4-a716-446655440000", true},
		{"550E8400-E29B-41D4-A716-446655440000", true}, // uppercase
		{"my-deployment", false},
		{"", false},
		{"not-a-uuid", false},
		{"550e8400e29b41d4a716446655440000", true}, // no dashes - still valid per google/uuid
	}

	for _, tt := range tests {
		result := isUUID(tt.input)
		if result != tt.expected {
			t.Errorf("isUUID(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}
