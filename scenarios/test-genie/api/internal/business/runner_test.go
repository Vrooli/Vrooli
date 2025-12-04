package business

import (
	"context"
	"errors"
	"io"
	"testing"

	"test-genie/internal/business/discovery"
	"test-genie/internal/business/existence"
	"test-genie/internal/business/parsing"
	"test-genie/internal/business/validation"
	reqparsing "test-genie/internal/requirements/parsing"
	"test-genie/internal/requirements/types"
	structtypes "test-genie/internal/structure/types"
)

// =============================================================================
// Mock Validators
// =============================================================================

type mockExistenceValidator struct {
	reqDirResult   structtypes.Result
	indexResult    structtypes.Result
}

func (m *mockExistenceValidator) ValidateRequirementsDir() structtypes.Result {
	return m.reqDirResult
}

func (m *mockExistenceValidator) ValidateIndexFile() structtypes.Result {
	return m.indexResult
}

type mockDiscoveryValidator struct {
	result discovery.DiscoveryResult
}

func (m *mockDiscoveryValidator) Discover(ctx context.Context, requireModules bool) discovery.DiscoveryResult {
	return m.result
}

type mockParsingValidator struct {
	result parsing.ParsingResult
}

func (m *mockParsingValidator) Parse(ctx context.Context, files []parsing.DiscoveredFile) parsing.ParsingResult {
	return m.result
}

type mockStructuralValidator struct {
	result validation.ValidationResult
}

func (m *mockStructuralValidator) Validate(ctx context.Context, index *reqparsing.ModuleIndex, scenarioRoot string) validation.ValidationResult {
	return m.result
}

// =============================================================================
// Helper to create a runner with all mocks
// =============================================================================

func newMockedRunner(
	existVal *mockExistenceValidator,
	discVal *mockDiscoveryValidator,
	parseVal *mockParsingValidator,
	structVal *mockStructuralValidator,
) *Runner {
	return New(
		Config{
			ScenarioDir:  "/mock/scenario",
			ScenarioName: "mock-scenario",
			Expectations: DefaultExpectations(),
		},
		WithLogger(io.Discard),
		WithExistenceValidator(existVal),
		WithDiscoveryValidator(discVal),
		WithParsingValidator(parseVal),
		WithStructuralValidator(structVal),
	)
}

// =============================================================================
// Runner Tests
// =============================================================================

func TestRunner_AllValidatorsPass(t *testing.T) {
	runner := newMockedRunner(
		&mockExistenceValidator{
			reqDirResult: structtypes.OK().WithObservations(structtypes.NewSuccessObservation("requirements found")),
			indexResult:  structtypes.OK().WithObservations(structtypes.NewSuccessObservation("index found")),
		},
		&mockDiscoveryValidator{
			result: discovery.DiscoveryResult{
				Result:      structtypes.OK().WithObservations(structtypes.NewSuccessObservation("discovered 2 modules")),
				ModuleCount: 2,
				Files: []discovery.DiscoveredFile{
					{AbsolutePath: "/test/index.json", IsIndex: true},
					{AbsolutePath: "/test/01-core/module.json", ModuleDir: "01-core"},
					{AbsolutePath: "/test/02-extra/module.json", ModuleDir: "02-extra"},
				},
			},
		},
		&mockParsingValidator{
			result: parsing.ParsingResult{
				Result:           structtypes.OK().WithObservations(structtypes.NewSuccessObservation("parsed 5 requirements")),
				ModuleCount:      2,
				RequirementCount: 5,
				Index:            reqparsing.NewModuleIndex(),
			},
		},
		&mockStructuralValidator{
			result: validation.ValidationResult{
				Result:       structtypes.OK(),
				ErrorCount:   0,
				WarningCount: 0,
			},
		},
	)

	result := runner.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if len(result.Observations) == 0 {
		t.Error("expected observations to be recorded")
	}
	if result.Summary.ModulesFound != 2 {
		t.Errorf("expected 2 modules found, got %d", result.Summary.ModulesFound)
	}
	if result.Summary.RequirementsFound != 5 {
		t.Errorf("expected 5 requirements found, got %d", result.Summary.RequirementsFound)
	}
}

func TestRunner_RequirementsDirValidationFails(t *testing.T) {
	runner := newMockedRunner(
		&mockExistenceValidator{
			reqDirResult: structtypes.FailMisconfiguration(
				errors.New("requirements directory not found"),
				"Run vrooli scenario requirements init",
			),
			indexResult: structtypes.OK(),
		},
		&mockDiscoveryValidator{result: discovery.DiscoveryResult{Result: structtypes.OK()}},
		&mockParsingValidator{result: parsing.ParsingResult{Result: structtypes.OK()}},
		&mockStructuralValidator{result: validation.ValidationResult{Result: structtypes.OK()}},
	)

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when requirements directory validation fails")
	}
	if result.FailureClass != FailureClassMisconfiguration {
		t.Errorf("expected misconfiguration, got %s", result.FailureClass)
	}
}

func TestRunner_IndexValidationFails(t *testing.T) {
	runner := newMockedRunner(
		&mockExistenceValidator{
			reqDirResult: structtypes.OK().WithObservations(structtypes.NewSuccessObservation("dir found")),
			indexResult: structtypes.FailMisconfiguration(
				errors.New("index.json not found"),
				"Create requirements/index.json",
			),
		},
		&mockDiscoveryValidator{result: discovery.DiscoveryResult{Result: structtypes.OK()}},
		&mockParsingValidator{result: parsing.ParsingResult{Result: structtypes.OK()}},
		&mockStructuralValidator{result: validation.ValidationResult{Result: structtypes.OK()}},
	)

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when index validation fails")
	}
}

func TestRunner_DiscoveryFails(t *testing.T) {
	runner := newMockedRunner(
		&mockExistenceValidator{
			reqDirResult: structtypes.OK(),
			indexResult:  structtypes.OK(),
		},
		&mockDiscoveryValidator{
			result: discovery.DiscoveryResult{
				Result: structtypes.FailMisconfiguration(
					errors.New("no modules found"),
					"Run vrooli scenario requirements init",
				),
			},
		},
		&mockParsingValidator{result: parsing.ParsingResult{Result: structtypes.OK()}},
		&mockStructuralValidator{result: validation.ValidationResult{Result: structtypes.OK()}},
	)

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when discovery fails")
	}
}

func TestRunner_ParsingFails(t *testing.T) {
	runner := newMockedRunner(
		&mockExistenceValidator{
			reqDirResult: structtypes.OK(),
			indexResult:  structtypes.OK(),
		},
		&mockDiscoveryValidator{
			result: discovery.DiscoveryResult{
				Result:      structtypes.OK(),
				ModuleCount: 1,
				Files:       []discovery.DiscoveredFile{{AbsolutePath: "/test/module.json"}},
			},
		},
		&mockParsingValidator{
			result: parsing.ParsingResult{
				Result: structtypes.FailSystem(
					errors.New("invalid JSON"),
					"Fix JSON syntax",
				),
			},
		},
		&mockStructuralValidator{result: validation.ValidationResult{Result: structtypes.OK()}},
	)

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when parsing fails")
	}
	if result.FailureClass != FailureClassSystem {
		t.Errorf("expected system failure class, got %s", result.FailureClass)
	}
}

func TestRunner_StructuralValidationFails(t *testing.T) {
	runner := newMockedRunner(
		&mockExistenceValidator{
			reqDirResult: structtypes.OK(),
			indexResult:  structtypes.OK(),
		},
		&mockDiscoveryValidator{
			result: discovery.DiscoveryResult{
				Result:      structtypes.OK(),
				ModuleCount: 1,
				Files:       []discovery.DiscoveredFile{{AbsolutePath: "/test/module.json"}},
			},
		},
		&mockParsingValidator{
			result: parsing.ParsingResult{
				Result:           structtypes.OK(),
				ModuleCount:      1,
				RequirementCount: 2,
				Index:            reqparsing.NewModuleIndex(),
			},
		},
		&mockStructuralValidator{
			result: validation.ValidationResult{
				Result: structtypes.FailMisconfiguration(
					errors.New("duplicate IDs found"),
					"Fix duplicate requirement IDs",
				),
				ErrorCount:   1,
				WarningCount: 0,
				Issues: []types.ValidationIssue{
					{Severity: types.SeverityError, Message: "duplicate ID"},
				},
			},
		},
	)

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when structural validation fails")
	}
	if result.Summary.ValidationErrors != 1 {
		t.Errorf("expected 1 validation error, got %d", result.Summary.ValidationErrors)
	}
}

func TestRunner_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	runner := newMockedRunner(
		&mockExistenceValidator{reqDirResult: structtypes.OK(), indexResult: structtypes.OK()},
		&mockDiscoveryValidator{result: discovery.DiscoveryResult{Result: structtypes.OK()}},
		&mockParsingValidator{result: parsing.ParsingResult{Result: structtypes.OK()}},
		&mockStructuralValidator{result: validation.ValidationResult{Result: structtypes.OK()}},
	)

	result := runner.Run(ctx)

	if result.Success {
		t.Fatal("expected failure when context cancelled")
	}
	if result.FailureClass != FailureClassSystem {
		t.Errorf("expected system failure class for cancellation, got %s", result.FailureClass)
	}
}

func TestRunner_IndexValidationSkippedWhenDisabled(t *testing.T) {
	expectations := DefaultExpectations()
	expectations.RequireIndex = false

	runner := New(
		Config{
			ScenarioDir:  "/mock/scenario",
			ScenarioName: "mock-scenario",
			Expectations: expectations,
		},
		WithLogger(io.Discard),
		WithExistenceValidator(&mockExistenceValidator{
			reqDirResult: structtypes.OK(),
			// Index validation returns error - but should be skipped
			indexResult: structtypes.FailMisconfiguration(errors.New("should not be called"), ""),
		}),
		WithDiscoveryValidator(&mockDiscoveryValidator{
			result: discovery.DiscoveryResult{
				Result:      structtypes.OK(),
				ModuleCount: 1,
				Files:       []discovery.DiscoveredFile{{AbsolutePath: "/test/module.json"}},
			},
		}),
		WithParsingValidator(&mockParsingValidator{
			result: parsing.ParsingResult{
				Result:           structtypes.OK(),
				ModuleCount:      1,
				RequirementCount: 1,
				Index:            reqparsing.NewModuleIndex(),
			},
		}),
		WithStructuralValidator(&mockStructuralValidator{
			result: validation.ValidationResult{Result: structtypes.OK()},
		}),
	)

	result := runner.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success when index validation disabled, got error: %v", result.Error)
	}
}

// =============================================================================
// ValidationSummary Tests
// =============================================================================

func TestValidationSummary_TotalChecks(t *testing.T) {
	tests := []struct {
		name     string
		summary  ValidationSummary
		expected int
	}{
		{
			name: "all categories",
			summary: ValidationSummary{
				ModulesFound:      3,
				RequirementsFound: 10,
				ValidationErrors:  2,
				ValidationWarns:   5,
			},
			expected: 13, // 3 + 10 (errors and warnings don't count as checks)
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
		ModulesFound:      2,
		RequirementsFound: 5,
		ValidationErrors:  1,
		ValidationWarns:   3,
	}

	str := summary.String()
	if str == "" {
		t.Error("expected non-empty string")
	}
}

// =============================================================================
// Config Tests
// =============================================================================

func TestDefaultExpectations(t *testing.T) {
	exp := DefaultExpectations()

	if !exp.RequireModules {
		t.Error("expected RequireModules to default to true")
	}
	if !exp.RequireIndex {
		t.Error("expected RequireIndex to default to true")
	}
	if exp.MinCoveragePercent != 80 {
		t.Errorf("expected MinCoveragePercent to default to 80, got %d", exp.MinCoveragePercent)
	}
	if exp.ErrorCoveragePercent != 60 {
		t.Errorf("expected ErrorCoveragePercent to default to 60, got %d", exp.ErrorCoveragePercent)
	}
}

// =============================================================================
// Interface Compile-Time Checks
// =============================================================================

// Ensure mock types satisfy interfaces at compile time.
var (
	_ existence.Validator  = (*mockExistenceValidator)(nil)
	_ discovery.Validator  = (*mockDiscoveryValidator)(nil)
	_ parsing.Validator    = (*mockParsingValidator)(nil)
	_ validation.Validator = (*mockStructuralValidator)(nil)
)

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkRunnerAllPass(b *testing.B) {
	runner := newMockedRunner(
		&mockExistenceValidator{
			reqDirResult: structtypes.OK(),
			indexResult:  structtypes.OK(),
		},
		&mockDiscoveryValidator{
			result: discovery.DiscoveryResult{
				Result:      structtypes.OK(),
				ModuleCount: 2,
				Files: []discovery.DiscoveredFile{
					{AbsolutePath: "/test/index.json", IsIndex: true},
					{AbsolutePath: "/test/01-core/module.json"},
				},
			},
		},
		&mockParsingValidator{
			result: parsing.ParsingResult{
				Result:           structtypes.OK(),
				ModuleCount:      1,
				RequirementCount: 5,
				Index:            reqparsing.NewModuleIndex(),
			},
		},
		&mockStructuralValidator{
			result: validation.ValidationResult{Result: structtypes.OK()},
		},
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runner.Run(context.Background())
	}
}

func BenchmarkRunnerEarlyFailure(b *testing.B) {
	runner := newMockedRunner(
		&mockExistenceValidator{
			reqDirResult: structtypes.FailMisconfiguration(
				errors.New("missing directory"),
				"Create required directories",
			),
			indexResult: structtypes.OK(),
		},
		&mockDiscoveryValidator{result: discovery.DiscoveryResult{Result: structtypes.OK()}},
		&mockParsingValidator{result: parsing.ParsingResult{Result: structtypes.OK()}},
		&mockStructuralValidator{result: validation.ValidationResult{Result: structtypes.OK()}},
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runner.Run(context.Background())
	}
}
