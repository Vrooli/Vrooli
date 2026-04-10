package runner_test

import (
	"context"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
)

// =============================================================================
// REGISTRY FLAG VALIDATOR TESTS
// =============================================================================

func TestRegistryFlagValidator_ValidateFlags_AllowedPasses(t *testing.T) {
	registry := runner.NewRegistry()
	mock := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mock.SetCapabilities(runner.Capabilities{
		AllowedExtraFlags: []string{"--verbose", "--allowedTools"},
	})
	if err := registry.Register(mock); err != nil {
		t.Fatalf("register: %v", err)
	}

	validator := runner.NewRegistryFlagValidator(registry)

	err := validator.ValidateFlags(domain.RunnerTypeClaudeCode, []string{"--verbose", "--allowedTools"})
	if err != nil {
		t.Errorf("expected no error for allowed flags, got: %v", err)
	}
}

func TestRegistryFlagValidator_ValidateFlags_DisallowedRejects(t *testing.T) {
	registry := runner.NewRegistry()
	mock := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mock.SetCapabilities(runner.Capabilities{
		AllowedExtraFlags: []string{"--verbose"},
	})
	if err := registry.Register(mock); err != nil {
		t.Fatalf("register: %v", err)
	}

	validator := runner.NewRegistryFlagValidator(registry)

	err := validator.ValidateFlags(domain.RunnerTypeClaudeCode, []string{"--forbidden"})
	if err == nil {
		t.Fatal("expected error for disallowed flag, got nil")
	}
}

func TestRegistryFlagValidator_ValidateFlags_EqualsNormalization(t *testing.T) {
	registry := runner.NewRegistry()
	mock := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mock.SetCapabilities(runner.Capabilities{
		AllowedExtraFlags: []string{"--timeout"},
	})
	if err := registry.Register(mock); err != nil {
		t.Fatalf("register: %v", err)
	}

	validator := runner.NewRegistryFlagValidator(registry)

	// Flag with =value should be normalized to just the flag name for validation
	err := validator.ValidateFlags(domain.RunnerTypeClaudeCode, []string{"--timeout=30"})
	if err != nil {
		t.Errorf("expected no error for --timeout=30 (equals normalization), got: %v", err)
	}
}

func TestRegistryFlagValidator_ValidateFlags_UnknownRunnerType(t *testing.T) {
	registry := runner.NewRegistry()
	validator := runner.NewRegistryFlagValidator(registry)

	err := validator.ValidateFlags(domain.RunnerType("unknown"), []string{"--flag"})
	if err == nil {
		t.Fatal("expected error for unknown runner type, got nil")
	}
}

func TestRegistryFlagValidator_AllowedFlags(t *testing.T) {
	registry := runner.NewRegistry()
	mock := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mock.SetCapabilities(runner.Capabilities{
		AllowedExtraFlags: []string{"--verbose", "--allowedTools"},
	})
	if err := registry.Register(mock); err != nil {
		t.Fatalf("register: %v", err)
	}

	validator := runner.NewRegistryFlagValidator(registry)

	flags := validator.AllowedFlags(domain.RunnerTypeClaudeCode)
	if len(flags) != 2 {
		t.Errorf("expected 2 allowed flags, got %d", len(flags))
	}

	// Unknown runner should return nil
	unknown := validator.AllowedFlags(domain.RunnerType("bogus"))
	if unknown != nil {
		t.Errorf("expected nil for unknown runner, got %v", unknown)
	}
}

func TestRegistryFlagValidator_SupportedFeatures(t *testing.T) {
	registry := runner.NewRegistry()
	mock := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mock.SetCapabilities(runner.Capabilities{
		SupportedFeatures: []string{"EnableBrowser"},
	})
	if err := registry.Register(mock); err != nil {
		t.Fatalf("register: %v", err)
	}

	validator := runner.NewRegistryFlagValidator(registry)

	features := validator.SupportedFeatures(domain.RunnerTypeClaudeCode)
	if len(features) != 1 || features[0] != "EnableBrowser" {
		t.Errorf("expected [EnableBrowser], got %v", features)
	}

	// Unknown runner should return nil
	unknown := validator.SupportedFeatures(domain.RunnerType("bogus"))
	if unknown != nil {
		t.Errorf("expected nil for unknown runner, got %v", unknown)
	}
}

// =============================================================================
// MOCK FLAG VALIDATOR TESTS
// =============================================================================

func TestMockFlagValidator_Defaults(t *testing.T) {
	mock := &runner.MockFlagValidator{}

	// Default ValidateFlags should return nil
	if err := mock.ValidateFlags(domain.RunnerTypeClaudeCode, []string{"--anything"}); err != nil {
		t.Errorf("expected nil from default ValidateFlags, got: %v", err)
	}

	// Default AllowedFlags should return nil
	if flags := mock.AllowedFlags(domain.RunnerTypeClaudeCode); flags != nil {
		t.Errorf("expected nil from default AllowedFlags, got: %v", flags)
	}

	// Default SupportedFeatures should return nil
	if features := mock.SupportedFeatures(domain.RunnerTypeClaudeCode); features != nil {
		t.Errorf("expected nil from default SupportedFeatures, got: %v", features)
	}
}

func TestMockFlagValidator_CustomFuncs(t *testing.T) {
	mock := &runner.MockFlagValidator{
		ValidateFlagsFunc: func(rt domain.RunnerType, flags []string) error {
			return domain.NewValidationError("extraFlags", "mock rejected")
		},
		AllowedFlagsFunc: func(rt domain.RunnerType) []string {
			return []string{"--custom"}
		},
		SupportedFeaturesFunc: func(rt domain.RunnerType) []string {
			return []string{"CustomFeature"}
		},
	}

	if err := mock.ValidateFlags(domain.RunnerTypeClaudeCode, nil); err == nil {
		t.Error("expected error from custom ValidateFlags")
	}

	flags := mock.AllowedFlags(domain.RunnerTypeClaudeCode)
	if len(flags) != 1 || flags[0] != "--custom" {
		t.Errorf("expected [--custom], got %v", flags)
	}

	features := mock.SupportedFeatures(domain.RunnerTypeClaudeCode)
	if len(features) != 1 || features[0] != "CustomFeature" {
		t.Errorf("expected [CustomFeature], got %v", features)
	}
}

// =============================================================================
// MOCK RUNNER CAPABILITIES TESTS
// =============================================================================

func TestMockRunner_SetCapabilities_FeaturesAndFlags(t *testing.T) {
	mock := runner.NewMockRunner(domain.RunnerTypeClaudeCode)

	mock.SetCapabilities(runner.Capabilities{
		SupportedFeatures: []string{"EnableBrowser"},
		AllowedExtraFlags: []string{"--verbose"},
		SupportsStreaming: true,
	})

	caps := mock.Capabilities()
	if len(caps.SupportedFeatures) != 1 || caps.SupportedFeatures[0] != "EnableBrowser" {
		t.Errorf("SupportedFeatures = %v, want [EnableBrowser]", caps.SupportedFeatures)
	}
	if len(caps.AllowedExtraFlags) != 1 || caps.AllowedExtraFlags[0] != "--verbose" {
		t.Errorf("AllowedExtraFlags = %v, want [--verbose]", caps.AllowedExtraFlags)
	}
	if !caps.SupportsStreaming {
		t.Error("expected SupportsStreaming to be true")
	}
}

// =============================================================================
// DEFAULT REGISTRY AVAILABLE TESTS
// =============================================================================

func TestDefaultRegistry_Available_FiltersUnavailable(t *testing.T) {
	registry := runner.NewRegistry()

	available := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	available.SetAvailable(true, "available")

	unavailable := runner.NewMockRunner(domain.RunnerTypeCodex)
	unavailable.SetAvailable(false, "not available")

	if err := registry.Register(available); err != nil {
		t.Fatalf("register available: %v", err)
	}
	if err := registry.Register(unavailable); err != nil {
		t.Fatalf("register unavailable: %v", err)
	}

	ctx := context.Background()
	avail := registry.Available(ctx)
	if len(avail) != 1 {
		t.Errorf("expected 1 available runner, got %d", len(avail))
	}
	if len(avail) > 0 && avail[0].Type() != domain.RunnerTypeClaudeCode {
		t.Errorf("expected available runner to be claude-code, got %s", avail[0].Type())
	}
}
