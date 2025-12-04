package structure

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"test-genie/internal/structure/content"
	"test-genie/internal/structure/existence"
	"test-genie/internal/structure/types"
)

// Mock validators for testing the runner orchestration logic in isolation

type mockExistenceValidator struct {
	dirsResult  types.Result
	filesResult types.Result
}

func (m *mockExistenceValidator) ValidateDirs(dirs []string) types.Result {
	return m.dirsResult
}

func (m *mockExistenceValidator) ValidateFiles(files []string) types.Result {
	return m.filesResult
}

type mockCLIValidator struct {
	result existence.CLIResult
}

func (m *mockCLIValidator) Validate() existence.CLIResult {
	return m.result
}

type mockSchemaValidator struct {
	result types.Result
}

func (m *mockSchemaValidator) Validate() types.Result {
	return m.result
}

type mockManifestValidator struct {
	result types.Result
}

func (m *mockManifestValidator) Validate() types.Result {
	return m.result
}

// Helper to create a runner with all mocks
func newMockedRunner(
	existenceVal *mockExistenceValidator,
	cliVal *mockCLIValidator,
	schemaVal *mockSchemaValidator,
	manifestVal *mockManifestValidator,
) *Runner {
	return New(
		Config{
			ScenarioDir:  "/mock/scenario",
			ScenarioName: "mock-scenario",
			SchemasDir:   "/mock/schemas",
			Expectations: DefaultExpectations(),
		},
		WithLogger(io.Discard),
		WithExistenceValidator(existenceVal),
		WithCLIValidator(cliVal),
		WithSchemaValidator(schemaVal),
		WithManifestValidator(manifestVal),
	)
}

func TestRunner_AllValidatorsPass(t *testing.T) {
	runner := newMockedRunner(
		&mockExistenceValidator{
			dirsResult:  types.OKWithCount(6),
			filesResult: types.OKWithCount(7),
		},
		&mockCLIValidator{
			result: existence.CLIResult{
				Approach: existence.CLIApproachCrossPlatform,
				Result:   types.OK().WithObservations(types.NewSuccessObservation("CLI valid")),
			},
		},
		&mockSchemaValidator{result: types.OKWithCount(10)},
		&mockManifestValidator{
			result: types.OK().WithObservations(
				types.NewSuccessObservation("Name matches"),
				types.NewSuccessObservation("Health checks defined"),
			),
		},
	)

	result := runner.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if len(result.Observations) == 0 {
		t.Error("expected observations to be recorded")
	}
	if result.Summary.DirsChecked != 6 {
		t.Errorf("expected 6 dirs checked, got %d", result.Summary.DirsChecked)
	}
	if result.Summary.FilesChecked != 7 {
		t.Errorf("expected 7 files checked, got %d", result.Summary.FilesChecked)
	}
	if result.Summary.JSONFilesValid != 10 {
		t.Errorf("expected 10 JSON files valid, got %d", result.Summary.JSONFilesValid)
	}
}

func TestRunner_DirectoryValidationFails(t *testing.T) {
	runner := newMockedRunner(
		&mockExistenceValidator{
			dirsResult: types.FailMisconfiguration(
				errors.New("missing directory: api"),
				"Create the api directory",
			),
			filesResult: types.OK(),
		},
		&mockCLIValidator{result: existence.CLIResult{Result: types.OK()}},
		&mockSchemaValidator{result: types.OK()},
		&mockManifestValidator{result: types.OK()},
	)

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when directory validation fails")
	}
	if result.FailureClass != FailureClassMisconfiguration {
		t.Errorf("expected misconfiguration, got %s", result.FailureClass)
	}
}

func TestRunner_FileValidationFails(t *testing.T) {
	runner := newMockedRunner(
		&mockExistenceValidator{
			dirsResult: types.OKWithCount(6),
			filesResult: types.FailMisconfiguration(
				errors.New("missing file: README.md"),
				"Create README.md",
			),
		},
		&mockCLIValidator{result: existence.CLIResult{Result: types.OK()}},
		&mockSchemaValidator{result: types.OK()},
		&mockManifestValidator{result: types.OK()},
	)

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when file validation fails")
	}
}

func TestRunner_CLIValidationFails(t *testing.T) {
	runner := newMockedRunner(
		&mockExistenceValidator{
			dirsResult:  types.OKWithCount(6),
			filesResult: types.OKWithCount(7),
		},
		&mockCLIValidator{
			result: existence.CLIResult{
				Approach: existence.CLIApproachUnknown,
				Result: types.FailMisconfiguration(
					errors.New("CLI structure incomplete"),
					"Create cli/main.go and cli/go.mod",
				),
			},
		},
		&mockSchemaValidator{result: types.OK()},
		&mockManifestValidator{result: types.OK()},
	)

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when CLI validation fails")
	}
}

func TestRunner_ManifestValidationFails(t *testing.T) {
	runner := newMockedRunner(
		&mockExistenceValidator{
			dirsResult:  types.OKWithCount(6),
			filesResult: types.OKWithCount(7),
		},
		&mockCLIValidator{
			result: existence.CLIResult{
				Approach: existence.CLIApproachCrossPlatform,
				Result:   types.OK(),
			},
		},
		&mockSchemaValidator{result: types.OK()},
		&mockManifestValidator{
			result: types.FailMisconfiguration(
				errors.New("service name mismatch"),
				"Update service.name to match scenario directory",
			),
		},
		
	)

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when manifest validation fails")
	}
}

func TestRunner_JSONValidationFails(t *testing.T) {
	runner := newMockedRunner(
		&mockExistenceValidator{
			dirsResult:  types.OKWithCount(6),
			filesResult: types.OKWithCount(7),
		},
		&mockCLIValidator{
			result: existence.CLIResult{
				Approach: existence.CLIApproachCrossPlatform,
				Result:   types.OK(),
			},
		},
		&mockSchemaValidator{
			result: types.FailMisconfiguration(
				errors.New("invalid JSON: config.json"),
				"Fix JSON syntax in config.json",
			),
		},
		&mockManifestValidator{result: types.OK()},
		
	)

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when JSON validation fails")
	}
}

func TestRunner_SchemaValidationDisabled(t *testing.T) {
	// When SchemasDir is not set and no schema validator is provided,
	// schema validation should be skipped.
	runner := New(
		Config{
			ScenarioDir:  "/mock/scenario",
			ScenarioName: "mock-scenario",
			SchemasDir:   "", // No schemas dir = no schema validation
			Expectations: DefaultExpectations(),
		},
		WithLogger(io.Discard),
		WithExistenceValidator(&mockExistenceValidator{
			dirsResult:  types.OKWithCount(6),
			filesResult: types.OKWithCount(7),
		}),
		WithCLIValidator(&mockCLIValidator{
			result: existence.CLIResult{
				Approach: existence.CLIApproachCrossPlatform,
				Result:   types.OK(),
			},
		}),
		// Note: No WithSchemaValidator - schema validation is disabled
		WithManifestValidator(&mockManifestValidator{result: types.OK()}),
	)

	result := runner.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success when schema validation disabled, got error: %v", result.Error)
	}
	if result.Summary.JSONFilesValid != 0 {
		t.Errorf("expected 0 JSON files checked when disabled, got %d", result.Summary.JSONFilesValid)
	}
}

func TestRunner_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	runner := newMockedRunner(
		&mockExistenceValidator{dirsResult: types.OK(), filesResult: types.OK()},
		&mockCLIValidator{result: existence.CLIResult{Result: types.OK()}},
		&mockSchemaValidator{result: types.OK()},
		&mockManifestValidator{result: types.OK()},
		
	)

	result := runner.Run(ctx)

	if result.Success {
		t.Fatal("expected failure when context cancelled")
	}
	if result.FailureClass != FailureClassSystem {
		t.Errorf("expected system failure class for cancellation, got %s", result.FailureClass)
	}
}

func TestValidationSummary_TotalChecks(t *testing.T) {
	tests := []struct {
		name     string
		summary  ValidationSummary
		expected int
	}{
		{
			name: "all categories",
			summary: ValidationSummary{
				DirsChecked:    6,
				FilesChecked:   7,
				JSONFilesValid: 10,
			},
			expected: 23, // 6 + 7 + 10
		},
		{
			name:     "empty",
			summary:  ValidationSummary{},
			expected: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.summary.TotalChecks(); got != tc.expected {
				t.Errorf("TotalChecks() = %d, want %d", got, tc.expected)
			}
		})
	}
}

func TestValidationSummary_String(t *testing.T) {
	summary := ValidationSummary{
		DirsChecked:    6,
		FilesChecked:   7,
		JSONFilesValid: 10,
	}

	str := summary.String()
	if str == "" {
		t.Error("expected non-empty string")
	}

	// Verify key information is included
	if !containsStrRunner(str, "6") {
		t.Error("expected dirs count in string")
	}
	if !containsStrRunner(str, "7") {
		t.Error("expected files count in string")
	}
	if !containsStrRunner(str, "10") {
		t.Error("expected JSON files count in string")
	}
}

// Benchmark tests

func BenchmarkRunnerAllPass(b *testing.B) {
	runner := newMockedRunner(
		&mockExistenceValidator{
			dirsResult:  types.OKWithCount(6),
			filesResult: types.OKWithCount(7),
		},
		&mockCLIValidator{
			result: existence.CLIResult{
				Approach: existence.CLIApproachCrossPlatform,
				Result:   types.OK(),
			},
		},
		&mockSchemaValidator{result: types.OKWithCount(10)},
		&mockManifestValidator{result: types.OK()},
		
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runner.Run(context.Background())
	}
}

func BenchmarkRunnerNoSchema(b *testing.B) {
	runner := New(
		Config{
			ScenarioDir:  "/mock/scenario",
			ScenarioName: "mock-scenario",
			SchemasDir:   "", // No schema validation
			Expectations: DefaultExpectations(),
		},
		WithLogger(io.Discard),
		WithExistenceValidator(&mockExistenceValidator{
			dirsResult:  types.OKWithCount(6),
			filesResult: types.OKWithCount(7),
		}),
		WithCLIValidator(&mockCLIValidator{
			result: existence.CLIResult{
				Approach: existence.CLIApproachCrossPlatform,
				Result:   types.OK(),
			},
		}),
		WithManifestValidator(&mockManifestValidator{result: types.OK()}),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runner.Run(context.Background())
	}
}

func BenchmarkRunnerEarlyFailure(b *testing.B) {
	runner := newMockedRunner(
		&mockExistenceValidator{
			dirsResult: types.FailMisconfiguration(
				errors.New("missing directory"),
				"Create required directories",
			),
			filesResult: types.OK(),
		},
		&mockCLIValidator{result: existence.CLIResult{Result: types.OK()}},
		&mockSchemaValidator{result: types.OK()},
		&mockManifestValidator{result: types.OK()},
		
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runner.Run(context.Background())
	}
}

// Ensure mock types satisfy interfaces at compile time
var (
	_ existence.Validator              = (*mockExistenceValidator)(nil)
	_ existence.CLIValidator           = (*mockCLIValidator)(nil)
	_ content.SchemaValidatorInterface = (*mockSchemaValidator)(nil)
	_ content.ManifestValidator        = (*mockManifestValidator)(nil)
)

func containsStrRunner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// =============================================================================
// Tests for New() default validator creation
// =============================================================================

func TestNew_DefaultValidators(t *testing.T) {
	// Test that New creates default validators when none are provided via options.
	// We use a temp directory to avoid file system errors from validators.
	tmpDir := t.TempDir()

	runner := New(Config{
		ScenarioDir:  tmpDir,
		ScenarioName: "test-scenario",
		SchemasDir:   "", // No schemas - schema validator will be nil
		Expectations: nil,
	})

	// Verify runner was created with defaults
	if runner.existenceValidator == nil {
		t.Error("expected default existence validator")
	}
	if runner.cliValidator == nil {
		t.Error("expected default CLI validator")
	}
	if runner.manifestValidator == nil {
		t.Error("expected default manifest validator")
	}
	// Schema validator should be nil when SchemasDir is empty
	if runner.schemaValidator != nil {
		t.Error("expected nil schema validator when SchemasDir is empty")
	}
}

func TestNew_DefaultValidatorsWithSchemasDir(t *testing.T) {
	tmpDir := t.TempDir()
	schemasDir := t.TempDir()

	runner := New(Config{
		ScenarioDir:  tmpDir,
		ScenarioName: "test-scenario",
		SchemasDir:   schemasDir,
		Expectations: nil,
	})

	// Schema validator should be created when SchemasDir is provided
	if runner.schemaValidator == nil {
		t.Error("expected default schema validator when SchemasDir is provided")
	}
}

func TestNew_WithExpectationsNameValidationFalse(t *testing.T) {
	tmpDir := t.TempDir()

	expectations := &Expectations{
		ValidateServiceName: false,
	}

	runner := New(Config{
		ScenarioDir:  tmpDir,
		ScenarioName: "test-scenario",
		Expectations: expectations,
	})

	// Verify runner was created - the manifest validator will have name validation disabled
	if runner.manifestValidator == nil {
		t.Error("expected manifest validator to be created")
	}
}

func TestNew_WithCustomLogger(t *testing.T) {
	tmpDir := t.TempDir()
	var buf bytes.Buffer

	runner := New(
		Config{
			ScenarioDir:  tmpDir,
			ScenarioName: "test-scenario",
		},
		WithLogger(&buf),
	)

	if runner.logWriter != &buf {
		t.Error("expected custom logger to be set")
	}
}

func TestNew_PartialOptions(t *testing.T) {
	// Test that providing some validators still creates defaults for others
	tmpDir := t.TempDir()

	existenceVal := &mockExistenceValidator{
		dirsResult:  types.OK(),
		filesResult: types.OK(),
	}

	runner := New(
		Config{
			ScenarioDir:  tmpDir,
			ScenarioName: "test-scenario",
		},
		WithExistenceValidator(existenceVal),
		// Not providing CLI, manifest validators - should use defaults
	)

	// Provided validator should be used
	if runner.existenceValidator != existenceVal {
		t.Error("expected custom existence validator")
	}

	// Unprovided validators should have defaults
	if runner.cliValidator == nil {
		t.Error("expected default CLI validator")
	}
	if runner.manifestValidator == nil {
		t.Error("expected default manifest validator")
	}
}

// Test logging with nil writer
func TestRunner_LoggingWithNilWriter(t *testing.T) {
	runner := newMockedRunner(
		&mockExistenceValidator{
			dirsResult:  types.OKWithCount(6),
			filesResult: types.OKWithCount(7),
		},
		&mockCLIValidator{
			result: existence.CLIResult{
				Approach: existence.CLIApproachCrossPlatform,
				Result:   types.OK(),
			},
		},
		&mockSchemaValidator{result: types.OKWithCount(10)},
		&mockManifestValidator{result: types.OK()},
		
	)

	// Set logger to nil explicitly
	runner.logWriter = nil

	// Should not panic when logging with nil writer
	result := runner.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
}

// Test additional expectations options
func TestRunner_WithAdditionalDirsAndFiles(t *testing.T) {
	expectations := DefaultExpectations()
	expectations.AdditionalDirs = []string{"custom-dir"}
	expectations.AdditionalFiles = []string{"custom-file.md"}

	runner := New(
		Config{
			ScenarioDir:  "/mock/scenario",
			ScenarioName: "mock-scenario",
			Expectations: expectations,
		},
		WithLogger(io.Discard),
		WithExistenceValidator(&mockExistenceValidator{
			dirsResult:  types.OKWithCount(7), // 6 default + 1 additional
			filesResult: types.OKWithCount(8), // 7 default + 1 additional
		}),
		WithCLIValidator(&mockCLIValidator{
			result: existence.CLIResult{
				Approach: existence.CLIApproachCrossPlatform,
				Result:   types.OK(),
			},
		}),
		WithSchemaValidator(&mockSchemaValidator{result: types.OK()}),
		WithManifestValidator(&mockManifestValidator{result: types.OK()}),
	)

	result := runner.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if result.Summary.DirsChecked != 7 {
		t.Errorf("expected 7 dirs checked, got %d", result.Summary.DirsChecked)
	}
	if result.Summary.FilesChecked != 8 {
		t.Errorf("expected 8 files checked, got %d", result.Summary.FilesChecked)
	}
}

func TestRunner_WithExcludedDirsAndFiles(t *testing.T) {
	expectations := DefaultExpectations()
	expectations.ExcludedDirs = []string{"docs"}
	expectations.ExcludedFiles = []string{"PRD.md"}

	runner := New(
		Config{
			ScenarioDir:  "/mock/scenario",
			ScenarioName: "mock-scenario",
			Expectations: expectations,
		},
		WithLogger(io.Discard),
		WithExistenceValidator(&mockExistenceValidator{
			dirsResult:  types.OKWithCount(5), // 6 default - 1 excluded
			filesResult: types.OKWithCount(6), // 7 default - 1 excluded
		}),
		WithCLIValidator(&mockCLIValidator{
			result: existence.CLIResult{
				Approach: existence.CLIApproachCrossPlatform,
				Result:   types.OK(),
			},
		}),
		WithSchemaValidator(&mockSchemaValidator{result: types.OK()}),
		WithManifestValidator(&mockManifestValidator{result: types.OK()}),
	)

	result := runner.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
}
