package containment

import (
	"context"
	"os/exec"
	"testing"
)

func TestAssessSecurityLevel(t *testing.T) {
	tests := []struct {
		name          string
		providerType  ContainmentType
		securityLevel int
		wantAdequate  bool
		wantWarnings  bool
	}{
		{
			name:          "No containment type - always inadequate",
			providerType:  ContainmentTypeNone,
			securityLevel: 5,
			wantAdequate:  false,
			wantWarnings:  true,
		},
		{
			name:          "Security level 0 - always inadequate",
			providerType:  ContainmentTypeDocker,
			securityLevel: SecurityLevelNone,
			wantAdequate:  false,
			wantWarnings:  true,
		},
		{
			name:          "Security level 1-4 - weak isolation",
			providerType:  ContainmentTypeDocker,
			securityLevel: 4,
			wantAdequate:  false,
			wantWarnings:  true,
		},
		{
			name:          "Security level 5 - minimum acceptable",
			providerType:  ContainmentTypeDocker,
			securityLevel: SecurityLevelMinimumAcceptable,
			wantAdequate:  true,
			wantWarnings:  false,
		},
		{
			name:          "Security level 8 - good isolation",
			providerType:  ContainmentTypeDocker,
			securityLevel: 8,
			wantAdequate:  true,
			wantWarnings:  false,
		},
		{
			name:          "Security level 10 - maximum",
			providerType:  ContainmentTypeDocker,
			securityLevel: SecurityLevelMaximum,
			wantAdequate:  true,
			wantWarnings:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AssessSecurityLevel(tt.providerType, tt.securityLevel)

			if result.Adequate != tt.wantAdequate {
				t.Errorf("Adequate = %v, want %v", result.Adequate, tt.wantAdequate)
			}

			hasWarnings := len(result.Warnings) > 0
			if hasWarnings != tt.wantWarnings {
				t.Errorf("HasWarnings = %v, want %v", hasWarnings, tt.wantWarnings)
			}

			if result.Level != tt.securityLevel {
				t.Errorf("Level = %v, want %v", result.Level, tt.securityLevel)
			}
		})
	}
}

func TestSecurityLevelConstants(t *testing.T) {
	// Verify the constants make sense
	if SecurityLevelNone != 0 {
		t.Errorf("SecurityLevelNone should be 0, got %d", SecurityLevelNone)
	}

	if SecurityLevelMinimumAcceptable <= SecurityLevelNone {
		t.Error("SecurityLevelMinimumAcceptable should be greater than SecurityLevelNone")
	}

	if SecurityLevelMaximum < SecurityLevelMinimumAcceptable {
		t.Error("SecurityLevelMaximum should be >= SecurityLevelMinimumAcceptable")
	}
}

func TestAssessSecurityLevelWarningMessages(t *testing.T) {
	t.Run("No containment warning mentions Docker", func(t *testing.T) {
		result := AssessSecurityLevel(ContainmentTypeNone, 0)
		if len(result.Warnings) == 0 {
			t.Fatal("Expected at least one warning")
		}

		warning := result.Warnings[0]
		if !containsString(warning, "Docker") {
			t.Errorf("Warning should mention Docker: %q", warning)
		}
	})

	t.Run("Weak isolation warning includes security level", func(t *testing.T) {
		result := AssessSecurityLevel(ContainmentTypeDocker, 3)
		if len(result.Warnings) == 0 {
			t.Fatal("Expected at least one warning")
		}

		warning := result.Warnings[0]
		if !containsString(warning, "3/10") {
			t.Errorf("Warning should include security level: %q", warning)
		}
	})
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- Tests for DecideProvider decision function ---

// MockProvider implements Provider for testing.
type MockProvider struct {
	providerType  ContainmentType
	available     bool
	securityLevel int
}

func (m *MockProvider) Type() ContainmentType                { return m.providerType }
func (m *MockProvider) IsAvailable(ctx context.Context) bool { return m.available }
func (m *MockProvider) PrepareCommand(ctx context.Context, config ExecutionConfig) (*exec.Cmd, error) {
	return nil, nil
}
func (m *MockProvider) Info() ProviderInfo {
	return ProviderInfo{
		Type:          m.providerType,
		Name:          string(m.providerType),
		SecurityLevel: m.securityLevel,
	}
}

func TestDecideProvider(t *testing.T) {
	tests := []struct {
		name         string
		providers    []Provider
		fallback     Provider
		wantType     ContainmentType
		wantFallback bool
	}{
		{
			name: "First available provider is selected",
			providers: []Provider{
				&MockProvider{providerType: ContainmentTypeDocker, available: true, securityLevel: 7},
				&MockProvider{providerType: ContainmentTypeBubblewrap, available: true, securityLevel: 5},
			},
			fallback:     &MockProvider{providerType: ContainmentTypeNone, available: true, securityLevel: 0},
			wantType:     ContainmentTypeDocker,
			wantFallback: false,
		},
		{
			name: "Falls back to second provider if first unavailable",
			providers: []Provider{
				&MockProvider{providerType: ContainmentTypeDocker, available: false, securityLevel: 7},
				&MockProvider{providerType: ContainmentTypeBubblewrap, available: true, securityLevel: 5},
			},
			fallback:     &MockProvider{providerType: ContainmentTypeNone, available: true, securityLevel: 0},
			wantType:     ContainmentTypeBubblewrap,
			wantFallback: false,
		},
		{
			name: "Falls back to fallback if no providers available",
			providers: []Provider{
				&MockProvider{providerType: ContainmentTypeDocker, available: false, securityLevel: 7},
				&MockProvider{providerType: ContainmentTypeBubblewrap, available: false, securityLevel: 5},
			},
			fallback:     &MockProvider{providerType: ContainmentTypeNone, available: true, securityLevel: 0},
			wantType:     ContainmentTypeNone,
			wantFallback: true,
		},
		{
			name:         "Empty providers list uses fallback",
			providers:    []Provider{},
			fallback:     &MockProvider{providerType: ContainmentTypeNone, available: true, securityLevel: 0},
			wantType:     ContainmentTypeNone,
			wantFallback: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager(tt.providers, tt.fallback)
			ctx := context.Background()
			decision := manager.DecideProvider(ctx)

			if decision.SelectedType != tt.wantType {
				t.Errorf("SelectedType = %v, want %v", decision.SelectedType, tt.wantType)
			}

			if decision.UsedFallback != tt.wantFallback {
				t.Errorf("UsedFallback = %v, want %v", decision.UsedFallback, tt.wantFallback)
			}

			// All decisions should have a reason
			if decision.Reason == "" {
				t.Error("Expected non-empty Reason")
			}

			// Should have checked all providers
			if len(decision.CheckedProviders) != len(tt.providers) && !decision.UsedFallback {
				// If not fallback, should have checked at least one that was available
				foundAvailable := false
				for _, check := range decision.CheckedProviders {
					if check.Available {
						foundAvailable = true
						break
					}
				}
				if !foundAvailable && len(tt.providers) > 0 {
					t.Error("Expected to find an available provider in CheckedProviders")
				}
			}

			// Verify backward compatibility wrapper
			selected := manager.SelectProvider(ctx)
			if selected.Type() != tt.wantType {
				t.Errorf("SelectProvider() = %v, want %v", selected.Type(), tt.wantType)
			}
		})
	}
}

func TestDecideProviderWarnsOnFallback(t *testing.T) {
	manager := NewManager(
		[]Provider{
			&MockProvider{providerType: ContainmentTypeDocker, available: false, securityLevel: 7},
		},
		&MockProvider{providerType: ContainmentTypeNone, available: true, securityLevel: 0},
	)

	ctx := context.Background()
	decision := manager.DecideProvider(ctx)

	if !decision.UsedFallback {
		t.Error("Expected UsedFallback to be true")
	}

	if !containsString(decision.Reason, "WARNING") {
		t.Errorf("Expected fallback reason to contain WARNING: %q", decision.Reason)
	}
}

// Suppress unused import warning for exec (used in MockProvider)
var _ = exec.Command
