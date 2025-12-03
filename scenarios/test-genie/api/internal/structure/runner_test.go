package structure

import (
	"context"
	"errors"
	"io"
	"testing"

	"test-genie/internal/structure/content"
	"test-genie/internal/structure/existence"
	"test-genie/internal/structure/smoke"
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

type mockJSONValidator struct {
	result types.Result
}

func (m *mockJSONValidator) Validate() types.Result {
	return m.result
}

type mockManifestValidator struct {
	result types.Result
}

func (m *mockManifestValidator) Validate() types.Result {
	return m.result
}

type mockSmokeValidator struct {
	result types.Result
}

func (m *mockSmokeValidator) Validate(ctx context.Context) types.Result {
	return m.result
}

// Helper to create a runner with all mocks
func newMockedRunner(
	existenceVal *mockExistenceValidator,
	cliVal *mockCLIValidator,
	jsonVal *mockJSONValidator,
	manifestVal *mockManifestValidator,
	smokeVal *mockSmokeValidator,
) *Runner {
	return New(
		Config{
			ScenarioDir:  "/mock/scenario",
			ScenarioName: "mock-scenario",
			Expectations: DefaultExpectations(),
		},
		WithLogger(io.Discard),
		WithExistenceValidator(existenceVal),
		WithCLIValidator(cliVal),
		WithJSONValidator(jsonVal),
		WithManifestValidator(manifestVal),
		WithSmokeValidator(smokeVal),
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
		&mockJSONValidator{result: types.OKWithCount(10)},
		&mockManifestValidator{
			result: types.OK().WithObservations(
				types.NewSuccessObservation("Name matches"),
				types.NewSuccessObservation("Health checks defined"),
			),
		},
		&mockSmokeValidator{
			result: types.Result{
				Success:      true,
				ItemsChecked: 1,
				Observations: []types.Observation{types.NewSuccessObservation("Smoke passed")},
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
	if result.Summary.DirsChecked != 6 {
		t.Errorf("expected 6 dirs checked, got %d", result.Summary.DirsChecked)
	}
	if result.Summary.FilesChecked != 7 {
		t.Errorf("expected 7 files checked, got %d", result.Summary.FilesChecked)
	}
	if result.Summary.JSONFilesValid != 10 {
		t.Errorf("expected 10 JSON files valid, got %d", result.Summary.JSONFilesValid)
	}
	if !result.Summary.SmokeChecked {
		t.Error("expected smoke to be checked")
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
		&mockJSONValidator{result: types.OK()},
		&mockManifestValidator{result: types.OK()},
		&mockSmokeValidator{result: types.Result{Success: true}},
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
		&mockJSONValidator{result: types.OK()},
		&mockManifestValidator{result: types.OK()},
		&mockSmokeValidator{result: types.Result{Success: true}},
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
		&mockJSONValidator{result: types.OK()},
		&mockManifestValidator{result: types.OK()},
		&mockSmokeValidator{result: types.Result{Success: true}},
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
		&mockJSONValidator{result: types.OK()},
		&mockManifestValidator{
			result: types.FailMisconfiguration(
				errors.New("service name mismatch"),
				"Update service.name to match scenario directory",
			),
		},
		&mockSmokeValidator{result: types.Result{Success: true}},
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
		&mockJSONValidator{
			result: types.FailMisconfiguration(
				errors.New("invalid JSON: config.json"),
				"Fix JSON syntax in config.json",
			),
		},
		&mockManifestValidator{result: types.OK()},
		&mockSmokeValidator{result: types.Result{Success: true}},
	)

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when JSON validation fails")
	}
}

func TestRunner_SmokeValidationFails(t *testing.T) {
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
		&mockJSONValidator{result: types.OKWithCount(5)},
		&mockManifestValidator{result: types.OK()},
		&mockSmokeValidator{
			result: types.FailSystem(
				errors.New("UI failed to load"),
				"Check browserless and UI configuration",
			),
		},
	)

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when smoke validation fails")
	}
	if result.FailureClass != FailureClassSystem {
		t.Errorf("expected system failure class, got %s", result.FailureClass)
	}
}

func TestRunner_SmokeDisabled(t *testing.T) {
	expectations := DefaultExpectations()
	expectations.UISmoke.Enabled = false

	runner := New(
		Config{
			ScenarioDir:  "/mock/scenario",
			ScenarioName: "mock-scenario",
			Expectations: expectations,
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
		WithJSONValidator(&mockJSONValidator{result: types.OKWithCount(5)}),
		WithManifestValidator(&mockManifestValidator{result: types.OK()}),
		WithSmokeValidator(&mockSmokeValidator{
			result: types.FailSystem(errors.New("should not be called"), ""),
		}),
	)

	result := runner.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success when smoke disabled, got error: %v", result.Error)
	}
	if result.Summary.SmokeChecked {
		t.Error("expected smoke to be skipped when disabled")
	}

	// Check for skip observation
	hasSkipObs := false
	for _, obs := range result.Observations {
		if obs.Type == ObservationSkip {
			hasSkipObs = true
			break
		}
	}
	if !hasSkipObs {
		t.Error("expected skip observation for disabled smoke test")
	}
}

func TestRunner_JSONValidationDisabled(t *testing.T) {
	expectations := DefaultExpectations()
	expectations.ValidateJSONFiles = false
	expectations.UISmoke.Enabled = false

	runner := New(
		Config{
			ScenarioDir:  "/mock/scenario",
			ScenarioName: "mock-scenario",
			Expectations: expectations,
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
		WithJSONValidator(&mockJSONValidator{
			result: types.FailMisconfiguration(errors.New("should not be called"), ""),
		}),
		WithManifestValidator(&mockManifestValidator{result: types.OK()}),
		WithSmokeValidator(&mockSmokeValidator{result: types.Result{Success: true}}),
	)

	result := runner.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success when JSON validation disabled, got error: %v", result.Error)
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
		&mockJSONValidator{result: types.OK()},
		&mockManifestValidator{result: types.OK()},
		&mockSmokeValidator{result: types.Result{Success: true}},
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
				SmokeChecked:   true,
			},
			expected: 24, // 6 + 7 + 10 + 1
		},
		{
			name: "no smoke",
			summary: ValidationSummary{
				DirsChecked:    6,
				FilesChecked:   7,
				JSONFilesValid: 10,
				SmokeChecked:   false,
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
		SmokeChecked:   true,
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
	if !containsStrRunner(str, "smoke") {
		t.Error("expected smoke mention in string")
	}
}

func TestValidationSummary_StringNoSmoke(t *testing.T) {
	summary := ValidationSummary{
		DirsChecked:    6,
		FilesChecked:   7,
		JSONFilesValid: 10,
		SmokeChecked:   false,
	}

	str := summary.String()
	if containsStrRunner(str, "smoke") {
		t.Error("expected no smoke mention when not checked")
	}
}

// Ensure mock types satisfy interfaces at compile time
var (
	_ existence.Validator    = (*mockExistenceValidator)(nil)
	_ existence.CLIValidator = (*mockCLIValidator)(nil)
	_ content.JSONValidator  = (*mockJSONValidator)(nil)
	_ content.ManifestValidator = (*mockManifestValidator)(nil)
	_ smoke.Validator        = (*mockSmokeValidator)(nil)
)

func containsStrRunner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
