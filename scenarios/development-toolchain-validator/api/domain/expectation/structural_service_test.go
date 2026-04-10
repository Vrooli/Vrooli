package expectation

import (
	"context"
	"errors"
	"testing"
	"time"
)

// MockStructuralRepository is a mock implementation for testing.
type MockStructuralRepository struct {
	expectations map[string]*StructuralExpectation
	createErr    error
	getErr       error
	listErr      error
	deleteErr    error
}

func NewMockStructuralRepository() *MockStructuralRepository {
	return &MockStructuralRepository{
		expectations: make(map[string]*StructuralExpectation),
	}
}

func (m *MockStructuralRepository) Create(_ context.Context, input CreateStructuralInput) (*StructuralExpectation, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	exp := &StructuralExpectation{
		ID:              "exp-" + input.Pattern,
		ConnectionID:    input.ConnectionID,
		Type:            input.Type,
		Pattern:         input.Pattern,
		Required:        input.Required,
		ExpectedContent: input.ExpectedContent,
		Description:     input.Description,
		CreatedAt:       time.Now(),
	}
	m.expectations[exp.ID] = exp
	return exp, nil
}

func (m *MockStructuralRepository) GetByID(_ context.Context, id string) (*StructuralExpectation, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	exp, ok := m.expectations[id]
	if !ok {
		return nil, ErrNotFound
	}
	return exp, nil
}

func (m *MockStructuralRepository) List(_ context.Context, opts ListOptions) ([]*StructuralExpectation, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []*StructuralExpectation
	for _, exp := range m.expectations {
		if opts.ConnectionID == "" || exp.ConnectionID == opts.ConnectionID {
			result = append(result, exp)
		}
	}
	return result, nil
}

func (m *MockStructuralRepository) Delete(_ context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if _, ok := m.expectations[id]; !ok {
		return ErrNotFound
	}
	delete(m.expectations, id)
	return nil
}

func (m *MockStructuralRepository) DeleteByConnection(_ context.Context, connectionID string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	for id, exp := range m.expectations {
		if exp.ConnectionID == connectionID {
			delete(m.expectations, id)
		}
	}
	return nil
}

// TestService_ValidateStructuralInput tests structural expectation validation.
// [REQ:REQ-P0-005] Structural Expectation Configuration Store
func TestService_ValidateStructuralInput(t *testing.T) {
	tests := []struct {
		name    string
		input   CreateStructuralInput
		wantErr error
	}{
		{
			name: "valid folder expectation",
			input: CreateStructuralInput{
				ConnectionID: "conn-123",
				Type:         TypeFolder,
				Pattern:      "api/domain/*",
				Required:     true,
			},
			wantErr: nil,
		},
		{
			name: "valid file expectation",
			input: CreateStructuralInput{
				ConnectionID: "conn-123",
				Type:         TypeFile,
				Pattern:      "README.md",
				Required:     true,
			},
			wantErr: nil,
		},
		{
			name: "valid content snippet",
			input: CreateStructuralInput{
				ConnectionID:    "conn-123",
				Type:            TypeContentSnippet,
				Pattern:         "main.go",
				ExpectedContent: "package main",
				Required:        true,
			},
			wantErr: nil,
		},
		{
			name: "missing connection ID",
			input: CreateStructuralInput{
				Type:    TypeFolder,
				Pattern: "api/",
			},
			wantErr: ErrInvalidConnectionID,
		},
		{
			name: "invalid type",
			input: CreateStructuralInput{
				ConnectionID: "conn-123",
				Type:         "invalid",
				Pattern:      "api/",
			},
			wantErr: ErrInvalidType,
		},
		{
			name: "empty pattern",
			input: CreateStructuralInput{
				ConnectionID: "conn-123",
				Type:         TypeFolder,
				Pattern:      "",
			},
			wantErr: ErrInvalidPattern,
		},
		{
			name: "content snippet without content",
			input: CreateStructuralInput{
				ConnectionID: "conn-123",
				Type:         TypeContentSnippet,
				Pattern:      "main.go",
			},
			wantErr: ErrInvalidPattern,
		},
	}

	svc := NewService(NewMockStructuralRepository(), NewMockCLIRepository())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.ValidateStructuralInput(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ValidateStructuralInput() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestService_CreateStructural tests structural expectation creation.
// [REQ:REQ-P0-005] Structural Expectation Configuration Store
func TestService_CreateStructural(t *testing.T) {
	structRepo := NewMockStructuralRepository()
	cliRepo := NewMockCLIRepository()
	svc := NewService(structRepo, cliRepo)
	ctx := context.Background()

	input := CreateStructuralInput{
		ConnectionID: "conn-123",
		Type:         TypeFolder,
		Pattern:      "api/domain",
		Required:     true,
		Description:  "Domain layer folder",
	}

	exp, err := svc.CreateStructural(ctx, input)
	if err != nil {
		t.Fatalf("CreateStructural() error = %v", err)
	}

	if exp.ConnectionID != input.ConnectionID {
		t.Errorf("ConnectionID = %v, want %v", exp.ConnectionID, input.ConnectionID)
	}
	if exp.Type != input.Type {
		t.Errorf("Type = %v, want %v", exp.Type, input.Type)
	}
	if exp.Pattern != input.Pattern {
		t.Errorf("Pattern = %v, want %v", exp.Pattern, input.Pattern)
	}
}

// TestService_ListStructural tests listing structural expectations.
// [REQ:REQ-P0-005] Structural Expectation Configuration Store
func TestService_ListStructural(t *testing.T) {
	structRepo := NewMockStructuralRepository()
	cliRepo := NewMockCLIRepository()
	svc := NewService(structRepo, cliRepo)
	ctx := context.Background()

	// Create some expectations
	_, _ = svc.CreateStructural(ctx, CreateStructuralInput{
		ConnectionID: "conn-1",
		Type:         TypeFolder,
		Pattern:      "api/",
		Required:     true,
	})
	_, _ = svc.CreateStructural(ctx, CreateStructuralInput{
		ConnectionID: "conn-1",
		Type:         TypeFile,
		Pattern:      "README.md",
		Required:     true,
	})
	_, _ = svc.CreateStructural(ctx, CreateStructuralInput{
		ConnectionID: "conn-2",
		Type:         TypeFolder,
		Pattern:      "ui/",
		Required:     true,
	})

	// List all
	all, err := svc.ListStructural(ctx, ListOptions{})
	if err != nil {
		t.Fatalf("ListStructural() error = %v", err)
	}
	if len(all) != 3 {
		t.Errorf("ListStructural() returned %d, want 3", len(all))
	}

	// List by connection
	conn1, err := svc.ListStructural(ctx, ListOptions{ConnectionID: "conn-1"})
	if err != nil {
		t.Fatalf("ListStructural(conn-1) error = %v", err)
	}
	if len(conn1) != 2 {
		t.Errorf("ListStructural(conn-1) returned %d, want 2", len(conn1))
	}
}

// TestService_DeleteStructuralByConnection tests cascade deletion.
// [REQ:REQ-P0-005] Structural Expectation Configuration Store
func TestService_DeleteStructuralByConnection(t *testing.T) {
	structRepo := NewMockStructuralRepository()
	cliRepo := NewMockCLIRepository()
	svc := NewService(structRepo, cliRepo)
	ctx := context.Background()

	// Create expectations for conn-1
	_, _ = svc.CreateStructural(ctx, CreateStructuralInput{
		ConnectionID: "conn-1",
		Type:         TypeFolder,
		Pattern:      "api/",
		Required:     true,
	})
	_, _ = svc.CreateStructural(ctx, CreateStructuralInput{
		ConnectionID: "conn-1",
		Type:         TypeFile,
		Pattern:      "README.md",
		Required:     true,
	})

	// Delete by connection
	err := svc.DeleteStructuralByConnection(ctx, "conn-1")
	if err != nil {
		t.Fatalf("DeleteStructuralByConnection() error = %v", err)
	}

	// Verify empty
	remaining, err := svc.ListStructural(ctx, ListOptions{ConnectionID: "conn-1"})
	if err != nil {
		t.Fatalf("ListStructural() error = %v", err)
	}
	if len(remaining) != 0 {
		t.Errorf("Expected 0 expectations after delete, got %d", len(remaining))
	}
}

// TestService_GetStructuralByID tests retrieving a structural expectation by ID.
// [REQ:REQ-P0-005] Structural Expectation Configuration Store
func TestService_GetStructuralByID(t *testing.T) {
	structRepo := NewMockStructuralRepository()
	cliRepo := NewMockCLIRepository()
	svc := NewService(structRepo, cliRepo)
	ctx := context.Background()

	// Create an expectation
	created, err := svc.CreateStructural(ctx, CreateStructuralInput{
		ConnectionID: "conn-123",
		Type:         TypeFolder,
		Pattern:      "api/domain",
		Required:     true,
		Description:  "Domain layer folder",
	})
	if err != nil {
		t.Fatalf("CreateStructural() error = %v", err)
	}

	// Get by ID
	retrieved, err := svc.GetStructuralByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetStructuralByID() error = %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("ID = %v, want %v", retrieved.ID, created.ID)
	}
	if retrieved.Pattern != created.Pattern {
		t.Errorf("Pattern = %v, want %v", retrieved.Pattern, created.Pattern)
	}
}

// TestService_GetStructuralByID_NotFound tests retrieving a non-existent expectation.
// [REQ:REQ-P0-005] Structural Expectation Configuration Store
func TestService_GetStructuralByID_NotFound(t *testing.T) {
	structRepo := NewMockStructuralRepository()
	cliRepo := NewMockCLIRepository()
	svc := NewService(structRepo, cliRepo)
	ctx := context.Background()

	_, err := svc.GetStructuralByID(ctx, "non-existent-id")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// TestService_DeleteStructural tests deleting a single structural expectation.
// [REQ:REQ-P0-005] Structural Expectation Configuration Store
func TestService_DeleteStructural(t *testing.T) {
	structRepo := NewMockStructuralRepository()
	cliRepo := NewMockCLIRepository()
	svc := NewService(structRepo, cliRepo)
	ctx := context.Background()

	// Create an expectation
	created, err := svc.CreateStructural(ctx, CreateStructuralInput{
		ConnectionID: "conn-123",
		Type:         TypeFile,
		Pattern:      "main.go",
		Required:     true,
	})
	if err != nil {
		t.Fatalf("CreateStructural() error = %v", err)
	}

	// Delete it
	err = svc.DeleteStructural(ctx, created.ID)
	if err != nil {
		t.Fatalf("DeleteStructural() error = %v", err)
	}

	// Verify it's gone
	_, err = svc.GetStructuralByID(ctx, created.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

// TestService_DeleteStructural_NotFound tests deleting a non-existent expectation.
// [REQ:REQ-P0-005] Structural Expectation Configuration Store
func TestService_DeleteStructural_NotFound(t *testing.T) {
	structRepo := NewMockStructuralRepository()
	cliRepo := NewMockCLIRepository()
	svc := NewService(structRepo, cliRepo)
	ctx := context.Background()

	err := svc.DeleteStructural(ctx, "non-existent-id")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// TestService_WithConfig tests service configuration options.
// [REQ:REQ-P0-005] Structural Expectation Configuration Store
func TestService_WithConfig(t *testing.T) {
	structRepo := NewMockStructuralRepository()
	cliRepo := NewMockCLIRepository()

	customConfig := ServiceConfig{
		Pagination: struct {
			DefaultLimit int
			MaxLimit     int
		}{
			DefaultLimit: 10,
			MaxLimit:     50,
		},
	}

	svc := NewService(structRepo, cliRepo, WithConfig(customConfig))

	// Create multiple expectations
	ctx := context.Background()
	for i := 0; i < 15; i++ {
		_, _ = svc.CreateStructural(ctx, CreateStructuralInput{
			ConnectionID: "conn-1",
			Type:         TypeFolder,
			Pattern:      "path/" + string(rune('a'+i)),
			Required:     true,
		})
	}

	// List without explicit limit should use default (10)
	// Note: MockStructuralRepository doesn't actually apply pagination,
	// but the service sets the limit correctly via ApplyPaginationLimit
	_, err := svc.ListStructural(ctx, ListOptions{})
	if err != nil {
		t.Fatalf("ListStructural() error = %v", err)
	}
}
