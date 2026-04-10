// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md#service-tests
// [REQ:REQ-P0-001] Reference Scenario Database Schema - Service layer tests for CRUD operations
// [REQ:REQ-P0-002] Reference Scenario API Endpoints - Unit tests for business logic
//
// NOTE: This file uses the external test package (reference_test) to avoid
// import cycles when importing mocks that depend on the reference package.
package reference_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"development-toolchain-validator/domain/reference"
	"development-toolchain-validator/internal/mocks"
	"development-toolchain-validator/internal/testutil"
)

// TestService_Create tests the Create method with table-driven tests.
func TestService_Create(t *testing.T) {
	// Create a temporary directory that exists for path validation tests
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		input       reference.CreateInput
		setupMock   func(*mocks.MockRepository)
		wantErr     error
		wantErrMsg  string
		category    string // happy_path, boundary, error, edge_case
		description string
	}{
		// Happy path tests
		{
			name: "valid_input_creates_reference",
			input: reference.CreateInput{
				Slug:        "test-scenario",
				Name:        "Test Scenario",
				Template:    "react-vite",
				Path:        tempDir,
				Description: "A test scenario",
			},
			setupMock:   func(m *mocks.MockRepository) {},
			wantErr:     nil,
			category:    "happy_path",
			description: "Valid input should create reference successfully",
		},
		{
			name: "valid_input_minimal_slug",
			input: reference.CreateInput{
				Slug:     "ab",
				Name:     "Minimal Slug",
				Template: "go-api",
				Path:     tempDir,
			},
			setupMock:   func(m *mocks.MockRepository) {},
			wantErr:     nil,
			category:    "boundary",
			description: "Minimum valid slug length (2 chars) should work",
		},

		// Error cases - invalid slug format
		{
			name: "invalid_slug_empty",
			input: reference.CreateInput{
				Slug:     "",
				Name:     "Test",
				Template: "react-vite",
				Path:     tempDir,
			},
			wantErr:     reference.ErrInvalidSlug,
			category:    "error",
			description: "Empty slug should fail validation",
		},
		{
			name: "invalid_slug_too_short",
			input: reference.CreateInput{
				Slug:     "a",
				Name:     "Test",
				Template: "react-vite",
				Path:     tempDir,
			},
			wantErr:     reference.ErrInvalidSlug,
			category:    "boundary",
			description: "Single character slug should fail (minimum is 2)",
		},
		{
			name: "invalid_slug_uppercase",
			input: reference.CreateInput{
				Slug:     "Test-Scenario",
				Name:     "Test",
				Template: "react-vite",
				Path:     tempDir,
			},
			wantErr:     reference.ErrInvalidSlug,
			category:    "error",
			description: "Uppercase letters in slug should fail",
		},
		{
			name: "invalid_slug_starts_with_hyphen",
			input: reference.CreateInput{
				Slug:     "-test-scenario",
				Name:     "Test",
				Template: "react-vite",
				Path:     tempDir,
			},
			wantErr:     reference.ErrInvalidSlug,
			category:    "error",
			description: "Slug starting with hyphen should fail",
		},
		{
			name: "invalid_slug_ends_with_hyphen",
			input: reference.CreateInput{
				Slug:     "test-scenario-",
				Name:     "Test",
				Template: "react-vite",
				Path:     tempDir,
			},
			wantErr:     reference.ErrInvalidSlug,
			category:    "error",
			description: "Slug ending with hyphen should fail",
		},

		// Error cases - duplicate slug
		{
			name: "duplicate_slug_error",
			input: reference.CreateInput{
				Slug:     "existing-slug",
				Name:     "Test",
				Template: "react-vite",
				Path:     tempDir,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().
					WithSlug("existing-slug").
					Build())
			},
			wantErr:     reference.ErrSlugExists,
			category:    "error",
			description: "Duplicate slug should return ErrSlugExists",
		},

		// Error cases - path validation
		{
			name: "nonexistent_path",
			input: reference.CreateInput{
				Slug:     "test-scenario",
				Name:     "Test",
				Template: "react-vite",
				Path:     "/nonexistent/path/that/does/not/exist",
			},
			wantErr:     reference.ErrPathNotExists,
			category:    "error",
			description: "Nonexistent path should fail validation",
		},

		// Edge cases
		{
			name: "slug_with_numbers",
			input: reference.CreateInput{
				Slug:     "test-123-scenario",
				Name:     "Test 123",
				Template: "react-vite",
				Path:     tempDir,
			},
			setupMock:   func(m *mocks.MockRepository) {},
			wantErr:     nil,
			category:    "edge_case",
			description: "Slug with numbers should be valid",
		},
		{
			name: "empty_description_allowed",
			input: reference.CreateInput{
				Slug:        "test-scenario",
				Name:        "Test",
				Template:    "react-vite",
				Path:        tempDir,
				Description: "",
			},
			setupMock:   func(m *mocks.MockRepository) {},
			wantErr:     nil,
			category:    "edge_case",
			description: "Empty description should be allowed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			repo := mocks.NewMockRepository()
			if tc.setupMock != nil {
				tc.setupMock(repo)
			}
			service := reference.NewService(repo)
			ctx := context.Background()

			// ACT
			result, err := service.Create(ctx, tc.input)

			// ASSERT
			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tc.wantErr)
				}
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got %v", tc.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
			if result.Slug != tc.input.Slug {
				t.Errorf("expected slug %q, got %q", tc.input.Slug, result.Slug)
			}
		})
	}
}

// TestService_GetByID tests the GetByID method.
func TestService_GetByID(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		setupMock func(*mocks.MockRepository)
		wantErr   error
		category  string
	}{
		{
			name: "existing_reference",
			id:   "ref-id-123",
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().
					WithID("ref-id-123").
					Build())
			},
			wantErr:  nil,
			category: "happy_path",
		},
		{
			name:      "nonexistent_reference",
			id:        "nonexistent-id",
			setupMock: func(m *mocks.MockRepository) {},
			wantErr:   reference.ErrNotFound,
			category:  "error",
		},
		{
			name:      "empty_id",
			id:        "",
			setupMock: func(m *mocks.MockRepository) {},
			wantErr:   reference.ErrNotFound,
			category:  "boundary",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			repo := mocks.NewMockRepository()
			tc.setupMock(repo)
			service := reference.NewService(repo)
			ctx := context.Background()

			// ACT
			result, err := service.GetByID(ctx, tc.id)

			// ASSERT
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got %v", tc.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.ID != tc.id {
				t.Errorf("expected ID %q, got %q", tc.id, result.ID)
			}
		})
	}
}

// TestService_List tests the List method with filtering and pagination.
func TestService_List(t *testing.T) {
	tests := []struct {
		name      string
		opts      reference.ListOptions
		setupMock func(*mocks.MockRepository)
		wantCount int
		wantErr   error
		category  string
	}{
		{
			name: "list_all_references",
			opts: reference.ListOptions{},
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().WithID("1").Build())
				m.WithReference(testutil.NewReferenceFactory().WithID("2").WithSlug("second").Build())
			},
			wantCount: 2,
			category:  "happy_path",
		},
		{
			name: "filter_by_template",
			opts: reference.ListOptions{Template: "react-vite"},
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().
					WithID("1").
					WithTemplate("react-vite").
					Build())
				m.WithReference(testutil.NewReferenceFactory().
					WithID("2").
					WithSlug("go-scenario").
					WithTemplate("go-api").
					Build())
			},
			wantCount: 1,
			category:  "happy_path",
		},
		{
			name:      "empty_list",
			opts:      reference.ListOptions{},
			setupMock: func(m *mocks.MockRepository) {},
			wantCount: 0,
			category:  "boundary",
		},
		{
			name: "default_limit_applied",
			opts: reference.ListOptions{Limit: 0}, // Should default to 20
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().Build())
			},
			wantCount: 1,
			category:  "edge_case",
		},
		{
			name: "limit_over_max_capped",
			opts: reference.ListOptions{Limit: 500}, // Should be capped to 100, then default to 20
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().Build())
			},
			wantCount: 1,
			category:  "boundary",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			repo := mocks.NewMockRepository()
			tc.setupMock(repo)
			service := reference.NewService(repo)
			ctx := context.Background()

			// ACT
			results, err := service.List(ctx, tc.opts)

			// ASSERT
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got %v", tc.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(results) != tc.wantCount {
				t.Errorf("expected %d results, got %d", tc.wantCount, len(results))
			}
		})
	}
}

// TestService_Update tests the Update method.
func TestService_Update(t *testing.T) {
	tempDir := t.TempDir()
	nonExistentPath := filepath.Join(tempDir, "does-not-exist")

	tests := []struct {
		name      string
		id        string
		input     reference.UpdateInput
		setupMock func(*mocks.MockRepository)
		wantErr   error
		category  string
	}{
		{
			name: "update_name",
			id:   "ref-123",
			input: reference.UpdateInput{
				Name: testutil.StringPtr("Updated Name"),
			},
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().
					WithID("ref-123").
					Build())
			},
			wantErr:  nil,
			category: "happy_path",
		},
		{
			name: "update_nonexistent_reference",
			id:   "nonexistent",
			input: reference.UpdateInput{
				Name: testutil.StringPtr("Updated"),
			},
			setupMock: func(m *mocks.MockRepository) {},
			wantErr:   reference.ErrNotFound,
			category:  "error",
		},
		{
			name: "update_with_invalid_path",
			id:   "ref-123",
			input: reference.UpdateInput{
				Path: &nonExistentPath,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().
					WithID("ref-123").
					Build())
			},
			wantErr:  reference.ErrPathNotExists,
			category: "error",
		},
		{
			name: "update_with_valid_path",
			id:   "ref-123",
			input: reference.UpdateInput{
				Path: &tempDir,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().
					WithID("ref-123").
					Build())
			},
			wantErr:  nil,
			category: "happy_path",
		},
		{
			name:  "empty_update_returns_existing",
			id:    "ref-123",
			input: reference.UpdateInput{}, // No fields to update
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().
					WithID("ref-123").
					Build())
			},
			wantErr:  nil,
			category: "edge_case",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			repo := mocks.NewMockRepository()
			tc.setupMock(repo)
			service := reference.NewService(repo)
			ctx := context.Background()

			// ACT
			result, err := service.Update(ctx, tc.id, tc.input)

			// ASSERT
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got %v", tc.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

// TestService_Delete tests the Delete method.
func TestService_Delete(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		setupMock func(*mocks.MockRepository)
		wantErr   error
		category  string
	}{
		{
			name: "delete_existing",
			id:   "ref-123",
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().
					WithID("ref-123").
					Build())
			},
			wantErr:  nil,
			category: "happy_path",
		},
		{
			name:      "delete_nonexistent",
			id:        "nonexistent",
			setupMock: func(m *mocks.MockRepository) {},
			wantErr:   reference.ErrNotFound,
			category:  "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			repo := mocks.NewMockRepository()
			tc.setupMock(repo)
			service := reference.NewService(repo)
			ctx := context.Background()

			// ACT
			err := service.Delete(ctx, tc.id)

			// ASSERT
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got %v", tc.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify deletion
			if repo.DeleteCallCount() != 1 {
				t.Errorf("expected 1 delete call, got %d", repo.DeleteCallCount())
			}
		})
	}
}

// TestService_ValidateCreate tests the ValidateCreate method for dry-run support.
func TestService_ValidateCreate(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		input       reference.CreateInput
		setupMock   func(*mocks.MockRepository)
		wantErr     error
		wantPath    string
		category    string
		description string
	}{
		{
			name: "valid_input_returns_normalized_path",
			input: reference.CreateInput{
				Slug:     "test-scenario",
				Name:     "Test Scenario",
				Template: "react-vite",
				Path:     tempDir,
			},
			setupMock:   func(m *mocks.MockRepository) {},
			wantErr:     nil,
			category:    "happy_path",
			description: "Valid input should return normalized path without error",
		},
		{
			name: "invalid_slug_returns_error",
			input: reference.CreateInput{
				Slug:     "INVALID",
				Name:     "Test",
				Template: "react-vite",
				Path:     tempDir,
			},
			wantErr:     reference.ErrInvalidSlug,
			category:    "error",
			description: "Invalid slug should return validation error",
		},
		{
			name: "duplicate_slug_returns_error",
			input: reference.CreateInput{
				Slug:     "existing-slug",
				Name:     "Test",
				Template: "react-vite",
				Path:     tempDir,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().
					WithSlug("existing-slug").
					Build())
			},
			wantErr:     reference.ErrSlugExists,
			category:    "error",
			description: "Duplicate slug should return conflict error",
		},
		{
			name: "nonexistent_path_returns_error",
			input: reference.CreateInput{
				Slug:     "test-scenario",
				Name:     "Test",
				Template: "react-vite",
				Path:     "/nonexistent/path",
			},
			wantErr:     reference.ErrPathNotExists,
			category:    "error",
			description: "Nonexistent path should return validation error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			repo := mocks.NewMockRepository()
			if tc.setupMock != nil {
				tc.setupMock(repo)
			}
			service := reference.NewService(repo)
			ctx := context.Background()

			// ACT
			absPath, err := service.ValidateCreate(ctx, tc.input)

			// ASSERT
			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tc.wantErr)
				}
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got %v", tc.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !filepath.IsAbs(absPath) {
				t.Errorf("expected absolute path, got %q", absPath)
			}
			// Verify no create was called (dry-run)
			if repo.CreateCallCount() != 0 {
				t.Errorf("expected 0 create calls, got %d", repo.CreateCallCount())
			}
		})
	}
}

// TestService_ValidateUpdate tests the ValidateUpdate method for dry-run support.
func TestService_ValidateUpdate(t *testing.T) {
	tempDir := t.TempDir()
	nonExistentPath := filepath.Join(tempDir, "does-not-exist")

	tests := []struct {
		name        string
		id          string
		input       reference.UpdateInput
		setupMock   func(*mocks.MockRepository)
		wantErr     error
		category    string
		description string
	}{
		{
			name: "valid_update_no_path",
			id:   "ref-123",
			input: reference.UpdateInput{
				Name: testutil.StringPtr("Updated Name"),
			},
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().
					WithID("ref-123").
					Build())
			},
			wantErr:     nil,
			category:    "happy_path",
			description: "Valid update without path should validate successfully",
		},
		{
			name: "valid_update_with_path",
			id:   "ref-123",
			input: reference.UpdateInput{
				Path: &tempDir,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().
					WithID("ref-123").
					Build())
			},
			wantErr:     nil,
			category:    "happy_path",
			description: "Valid update with path should validate successfully",
		},
		{
			name: "nonexistent_reference",
			id:   "nonexistent",
			input: reference.UpdateInput{
				Name: testutil.StringPtr("Updated"),
			},
			setupMock:   func(m *mocks.MockRepository) {},
			wantErr:     reference.ErrNotFound,
			category:    "error",
			description: "Nonexistent reference should return not found error",
		},
		{
			name: "invalid_path",
			id:   "ref-123",
			input: reference.UpdateInput{
				Path: &nonExistentPath,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().
					WithID("ref-123").
					Build())
			},
			wantErr:     reference.ErrPathNotExists,
			category:    "error",
			description: "Invalid path should return validation error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			repo := mocks.NewMockRepository()
			tc.setupMock(repo)
			service := reference.NewService(repo)
			ctx := context.Background()

			// ACT
			_, err := service.ValidateUpdate(ctx, tc.id, tc.input)

			// ASSERT
			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tc.wantErr)
				}
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got %v", tc.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Verify no update was called (dry-run)
			if repo.UpdateCallCount() != 0 {
				t.Errorf("expected 0 update calls, got %d", repo.UpdateCallCount())
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Replay Safety Tests
// DOC: docs/internal/INVARIANTS.md#testing-replay-safety
// ─────────────────────────────────────────────────────────────────────────────

// TestService_Create_ReplayReturnsConflict verifies that Create is NOT idempotent -
// replaying the same create operation returns a conflict error, preventing duplicates.
func TestService_Create_ReplayReturnsConflict(t *testing.T) {
	tempDir := t.TempDir()
	repo := mocks.NewMockRepository()
	service := reference.NewService(repo)
	ctx := context.Background()

	input := reference.CreateInput{
		Slug:     "replay-test",
		Name:     "Replay Test",
		Template: "react-vite",
		Path:     tempDir,
	}

	// First create should succeed
	ref1, err := service.Create(ctx, input)
	if err != nil {
		t.Fatalf("first create should succeed: %v", err)
	}
	if ref1.Slug != input.Slug {
		t.Errorf("expected slug %q, got %q", input.Slug, ref1.Slug)
	}

	// Replay (second create) should fail with ErrSlugExists
	_, err = service.Create(ctx, input)
	if err == nil {
		t.Fatal("replay should fail, got nil error")
	}
	if !errors.Is(err, reference.ErrSlugExists) {
		t.Fatalf("expected ErrSlugExists, got %v", err)
	}
}

// TestService_Update_ReplayProducesSameState verifies that Update IS idempotent -
// applying the same update twice produces the same final state.
func TestService_Update_ReplayProducesSameState(t *testing.T) {
	tempDir := t.TempDir()
	repo := mocks.NewMockRepository()
	repo.WithReference(testutil.NewReferenceFactory().
		WithID("update-replay-test").
		WithName("Original Name").
		WithPath(tempDir).
		Build())
	service := reference.NewService(repo)
	ctx := context.Background()

	input := reference.UpdateInput{
		Name: testutil.StringPtr("Updated Name"),
	}

	// First update
	ref1, err := service.Update(ctx, "update-replay-test", input)
	if err != nil {
		t.Fatalf("first update should succeed: %v", err)
	}
	if ref1.Name != *input.Name {
		t.Errorf("expected name %q, got %q", *input.Name, ref1.Name)
	}

	// Replay (second update with same input)
	ref2, err := service.Update(ctx, "update-replay-test", input)
	if err != nil {
		t.Fatalf("replay should succeed: %v", err)
	}

	// Verify same result
	if ref1.Name != ref2.Name {
		t.Errorf("replay produced different name: %q vs %q", ref1.Name, ref2.Name)
	}
}

// TestService_Delete_ReplayIsSafe verifies that Delete is idempotent by outcome -
// replaying delete leaves the reference deleted (404 on replay is acceptable).
func TestService_Delete_ReplayIsSafe(t *testing.T) {
	repo := mocks.NewMockRepository()
	repo.WithReference(testutil.NewReferenceFactory().
		WithID("delete-replay-test").
		Build())
	service := reference.NewService(repo)
	ctx := context.Background()

	// First delete should succeed
	err := service.Delete(ctx, "delete-replay-test")
	if err != nil {
		t.Fatalf("first delete should succeed: %v", err)
	}

	// Replay (second delete) returns ErrNotFound - acceptable
	err = service.Delete(ctx, "delete-replay-test")
	if !errors.Is(err, reference.ErrNotFound) {
		t.Fatalf("expected ErrNotFound on replay, got %v", err)
	}

	// Either way, reference should be deleted
	_, err = service.GetByID(ctx, "delete-replay-test")
	if !errors.Is(err, reference.ErrNotFound) {
		t.Fatalf("reference should be deleted, got %v", err)
	}
}

// TestService_ValidateCreate_NoSideEffects verifies that ValidateCreate (dry-run)
// has no side effects and can be called repeatedly without changing state.
func TestService_ValidateCreate_NoSideEffects(t *testing.T) {
	tempDir := t.TempDir()
	repo := mocks.NewMockRepository()
	service := reference.NewService(repo)
	ctx := context.Background()

	input := reference.CreateInput{
		Slug:     "dry-run-test",
		Name:     "Dry Run Test",
		Template: "react-vite",
		Path:     tempDir,
	}

	// Multiple ValidateCreate calls should all succeed
	for i := 0; i < 3; i++ {
		_, err := service.ValidateCreate(ctx, input)
		if err != nil {
			t.Fatalf("ValidateCreate call %d should succeed: %v", i+1, err)
		}
	}

	// No references should be created
	if repo.CreateCallCount() != 0 {
		t.Errorf("expected 0 create calls, got %d", repo.CreateCallCount())
	}

	// Slug should still be available for actual creation
	ref, err := service.Create(ctx, input)
	if err != nil {
		t.Fatalf("create after dry-run should succeed: %v", err)
	}
	if ref.Slug != input.Slug {
		t.Errorf("expected slug %q, got %q", input.Slug, ref.Slug)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Path Normalization Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestService_Create_PathNormalization verifies that paths are normalized to absolute.
func TestService_Create_PathNormalization(t *testing.T) {
	// Create a temp directory with a known structure
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "scenarios")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Change to tempDir to test relative path handling
	originalWd, _ := os.Getwd()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	t.Cleanup(func() { os.Chdir(originalWd) })

	repo := mocks.NewMockRepository()
	service := reference.NewService(repo)
	ctx := context.Background()

	input := reference.CreateInput{
		Slug:     "test-scenario",
		Name:     "Test",
		Template: "react-vite",
		Path:     "scenarios", // Relative path
	}

	result, err := service.Create(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the path was normalized to absolute
	if !filepath.IsAbs(result.Path) {
		t.Errorf("expected absolute path, got %q", result.Path)
	}
}
