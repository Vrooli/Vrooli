package validation

import (
	"context"
	"io"
	"strings"
	"testing"

	"test-genie/internal/requirements/parsing"
	"test-genie/internal/requirements/types"
)

// mockValidator implements reqvalidation.Validator for testing.
type mockValidator struct {
	result *types.ValidationResult
}

func (m *mockValidator) Validate(ctx context.Context, index *parsing.ModuleIndex, scenarioRoot string) *types.ValidationResult {
	if m.result != nil {
		return m.result
	}
	return types.NewValidationResult()
}

func TestValidate_NoIssues(t *testing.T) {
	mock := &mockValidator{result: types.NewValidationResult()}
	v := NewWithValidator(mock, io.Discard)

	index := parsing.NewModuleIndex()
	result := v.Validate(context.Background(), index, "/test")

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if result.ErrorCount != 0 {
		t.Errorf("expected 0 errors, got %d", result.ErrorCount)
	}
	if result.WarningCount != 0 {
		t.Errorf("expected 0 warnings, got %d", result.WarningCount)
	}
}

func TestValidate_WithErrors(t *testing.T) {
	mock := &mockValidator{
		result: &types.ValidationResult{
			Issues: []types.ValidationIssue{
				{
					FilePath:      "/test/module.json",
					RequirementID: "REQ-001",
					Field:         "id",
					Message:       "duplicate requirement ID",
					Severity:      types.SeverityError,
				},
				{
					FilePath:      "/test/module.json",
					RequirementID: "REQ-002",
					Field:         "children",
					Message:       "references non-existent child",
					Severity:      types.SeverityError,
				},
			},
		},
	}
	v := NewWithValidator(mock, io.Discard)

	index := parsing.NewModuleIndex()
	result := v.Validate(context.Background(), index, "/test")

	if result.Success {
		t.Fatal("expected failure for validation errors")
	}
	if result.FailureClass != FailureClassMisconfiguration {
		t.Errorf("expected misconfiguration failure class, got %s", result.FailureClass)
	}
	if result.ErrorCount != 2 {
		t.Errorf("expected 2 errors, got %d", result.ErrorCount)
	}
	if result.Remediation == "" {
		t.Error("expected remediation guidance")
	}
	if len(result.Issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(result.Issues))
	}

	// Check for error observations
	errorObsCount := 0
	for _, obs := range result.Observations {
		if obs.Type == ObservationError {
			errorObsCount++
		}
	}
	if errorObsCount != 2 {
		t.Errorf("expected 2 error observations, got %d", errorObsCount)
	}
}

func TestValidate_WithWarningsOnly(t *testing.T) {
	mock := &mockValidator{
		result: &types.ValidationResult{
			Issues: []types.ValidationIssue{
				{
					FilePath:      "/test/module.json",
					RequirementID: "REQ-001",
					Field:         "title",
					Message:       "requirement is missing title",
					Severity:      types.SeverityWarning,
				},
				{
					FilePath:      "/test/module.json",
					RequirementID: "REQ-002",
					Field:         "status",
					Message:       "invalid status value",
					Severity:      types.SeverityWarning,
				},
			},
		},
	}
	v := NewWithValidator(mock, io.Discard)

	index := parsing.NewModuleIndex()
	result := v.Validate(context.Background(), index, "/test")

	// Should succeed with only warnings
	if !result.Success {
		t.Fatalf("expected success with only warnings, got error: %v", result.Error)
	}
	if result.ErrorCount != 0 {
		t.Errorf("expected 0 errors, got %d", result.ErrorCount)
	}
	if result.WarningCount != 2 {
		t.Errorf("expected 2 warnings, got %d", result.WarningCount)
	}

	// Check for warning observations
	warningObsCount := 0
	for _, obs := range result.Observations {
		if obs.Type == ObservationWarning {
			warningObsCount++
		}
	}
	if warningObsCount != 2 {
		t.Errorf("expected 2 warning observations, got %d", warningObsCount)
	}
}

func TestValidate_MixedErrorsAndWarnings(t *testing.T) {
	mock := &mockValidator{
		result: &types.ValidationResult{
			Issues: []types.ValidationIssue{
				{
					FilePath:      "/test/module.json",
					RequirementID: "REQ-001",
					Field:         "id",
					Message:       "duplicate ID",
					Severity:      types.SeverityError,
				},
				{
					FilePath:      "/test/module.json",
					RequirementID: "REQ-002",
					Field:         "title",
					Message:       "missing title",
					Severity:      types.SeverityWarning,
				},
			},
		},
	}
	v := NewWithValidator(mock, io.Discard)

	index := parsing.NewModuleIndex()
	result := v.Validate(context.Background(), index, "/test")

	if result.Success {
		t.Fatal("expected failure when errors present")
	}
	if result.ErrorCount != 1 {
		t.Errorf("expected 1 error, got %d", result.ErrorCount)
	}
	if result.WarningCount != 1 {
		t.Errorf("expected 1 warning, got %d", result.WarningCount)
	}
}

func TestValidate_NilIndex(t *testing.T) {
	mock := &mockValidator{result: types.NewValidationResult()}
	v := NewWithValidator(mock, io.Discard)

	result := v.Validate(context.Background(), nil, "/test")

	// Should succeed with nil index (no requirements to validate)
	if !result.Success {
		t.Fatalf("expected success with nil index, got error: %v", result.Error)
	}
}

// TestNew tests the default constructor.
func TestNew(t *testing.T) {
	v := New(io.Discard)
	if v == nil {
		t.Fatal("expected non-nil validator")
	}

	// Should work with empty index
	index := parsing.NewModuleIndex()
	result := v.Validate(context.Background(), index, "/test")

	if !result.Success {
		t.Fatalf("expected success with empty index, got error: %v", result.Error)
	}
}

// TestValidate_WithNilLogWriter tests that nil log writer is handled.
func TestValidate_WithNilLogWriter(t *testing.T) {
	mock := &mockValidator{result: types.NewValidationResult()}
	v := NewWithValidator(mock, nil)

	index := parsing.NewModuleIndex()
	result := v.Validate(context.Background(), index, "/test")

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
}

// TestValidate_WithRealLogWriter tests that logging works with real writer.
func TestValidate_WithRealLogWriter(t *testing.T) {
	mock := &mockValidator{
		result: &types.ValidationResult{
			Issues: []types.ValidationIssue{
				{Severity: types.SeverityError, Message: "test error"},
			},
		},
	}

	var buf strings.Builder
	v := NewWithValidator(mock, &buf)

	index := parsing.NewModuleIndex()
	_ = v.Validate(context.Background(), index, "/test")

	output := buf.String()
	if !strings.Contains(output, "validation") {
		t.Errorf("expected logging output to contain 'validation', got: %s", output)
	}
}

// Ensure validator satisfies the interface at compile time
var _ Validator = (*validator)(nil)

// Benchmarks

func BenchmarkValidate_NoIssues(b *testing.B) {
	mock := &mockValidator{result: types.NewValidationResult()}
	v := NewWithValidator(mock, io.Discard)

	index := parsing.NewModuleIndex()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.Validate(ctx, index, "/test")
	}
}

func BenchmarkValidate_WithIssues(b *testing.B) {
	mock := &mockValidator{
		result: &types.ValidationResult{
			Issues: []types.ValidationIssue{
				{Severity: types.SeverityError, Message: "error 1"},
				{Severity: types.SeverityError, Message: "error 2"},
				{Severity: types.SeverityWarning, Message: "warning 1"},
				{Severity: types.SeverityWarning, Message: "warning 2"},
			},
		},
	}
	v := NewWithValidator(mock, io.Discard)

	index := parsing.NewModuleIndex()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.Validate(ctx, index, "/test")
	}
}
