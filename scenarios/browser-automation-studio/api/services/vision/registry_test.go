package vision

import (
	"context"
	"errors"
	"testing"
)

// mockNavigator implements VisionNavigator for testing.
type mockNavigator struct {
	navType            NavigatorType
	available          bool
	unavailableReason  string
	creditPolicy       CreditPolicy
	clientSourcePolicy ClientSourcePolicy
	description        string
}

func (m *mockNavigator) Navigate(_ context.Context, _ NavigationRequest) (NavigationHandle, error) {
	return nil, errors.New("not implemented")
}

func (m *mockNavigator) CreditPolicy() CreditPolicy {
	return m.creditPolicy
}

func (m *mockNavigator) ClientSourcePolicy() ClientSourcePolicy {
	return m.clientSourcePolicy
}

func (m *mockNavigator) Type() NavigatorType {
	return m.navType
}

func (m *mockNavigator) IsAvailable(_ context.Context) bool {
	return m.available
}

func (m *mockNavigator) Description() string {
	return m.description
}

func (m *mockNavigator) UnavailableReason(_ context.Context) string {
	return m.unavailableReason
}

func TestNewNavigatorRegistry(t *testing.T) {
	registry := NewNavigatorRegistry()

	if registry == nil {
		t.Fatal("NewNavigatorRegistry() returned nil")
	}

	if registry.Count() != 0 {
		t.Errorf("Count() = %d, want 0", registry.Count())
	}
}

func TestNavigatorRegistry_Register(t *testing.T) {
	registry := NewNavigatorRegistry()

	nav := &mockNavigator{
		navType:     NavigatorPlaywright,
		available:   true,
		description: "Test navigator",
	}

	registry.Register(nav)

	if registry.Count() != 1 {
		t.Errorf("Count() = %d, want 1", registry.Count())
	}

	// Register same type again (should replace)
	nav2 := &mockNavigator{
		navType:     NavigatorPlaywright,
		available:   false,
		description: "Replaced navigator",
	}
	registry.Register(nav2)

	if registry.Count() != 1 {
		t.Errorf("Count() after re-register = %d, want 1", registry.Count())
	}

	// Verify the replacement
	got, err := registry.Get(NavigatorPlaywright)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Description() != "Replaced navigator" {
		t.Errorf("Description() = %q, want %q", got.Description(), "Replaced navigator")
	}
}

func TestNavigatorRegistry_Get(t *testing.T) {
	registry := NewNavigatorRegistry()

	nav := &mockNavigator{
		navType:     NavigatorPlaywright,
		available:   true,
		description: "Test navigator",
	}
	registry.Register(nav)

	t.Run("existing navigator", func(t *testing.T) {
		got, err := registry.Get(NavigatorPlaywright)
		if err != nil {
			t.Errorf("Get() error = %v, want nil", err)
		}
		if got != nav {
			t.Error("Get() returned wrong navigator")
		}
	})

	t.Run("non-existent navigator", func(t *testing.T) {
		_, err := registry.Get(NavigatorClaudeCode)
		if !errors.Is(err, ErrNavigatorNotFound) {
			t.Errorf("Get() error = %v, want ErrNavigatorNotFound", err)
		}
	})
}

func TestNavigatorRegistry_SelectNavigator(t *testing.T) {
	ctx := t.Context()

	t.Run("select preferred type when available", func(t *testing.T) {
		registry := NewNavigatorRegistry()

		playwright := &mockNavigator{
			navType:            NavigatorPlaywright,
			available:          true,
			clientSourcePolicy: AllSourcesPolicy(),
		}
		claudeCode := &mockNavigator{
			navType:            NavigatorClaudeCode,
			available:          true,
			clientSourcePolicy: CLIOnlyPolicy(),
		}

		registry.Register(playwright)
		registry.Register(claudeCode)

		got, err := registry.SelectNavigator(ctx, ClientSourceCLI, NavigatorClaudeCode)
		if err != nil {
			t.Fatalf("SelectNavigator() error = %v", err)
		}
		if got.Type() != NavigatorClaudeCode {
			t.Errorf("SelectNavigator() type = %v, want %v", got.Type(), NavigatorClaudeCode)
		}
	})

	t.Run("preferred type not found", func(t *testing.T) {
		registry := NewNavigatorRegistry()

		playwright := &mockNavigator{
			navType:            NavigatorPlaywright,
			available:          true,
			clientSourcePolicy: AllSourcesPolicy(),
		}
		registry.Register(playwright)

		_, err := registry.SelectNavigator(ctx, ClientSourceCLI, NavigatorClaudeCode)
		if !errors.Is(err, ErrNavigatorNotFound) {
			t.Errorf("SelectNavigator() error = %v, want ErrNavigatorNotFound", err)
		}
	})

	t.Run("preferred type not allowed for client source", func(t *testing.T) {
		registry := NewNavigatorRegistry()

		claudeCode := &mockNavigator{
			navType:            NavigatorClaudeCode,
			available:          true,
			clientSourcePolicy: CLIOnlyPolicy(),
		}
		registry.Register(claudeCode)

		// Try to use CLI-only navigator from UI
		_, err := registry.SelectNavigator(ctx, ClientSourceUI, NavigatorClaudeCode)
		if !errors.Is(err, ErrNavigatorNotAllowed) {
			t.Errorf("SelectNavigator() error = %v, want ErrNavigatorNotAllowed", err)
		}
	})

	t.Run("preferred type not available", func(t *testing.T) {
		registry := NewNavigatorRegistry()

		claudeCode := &mockNavigator{
			navType:            NavigatorClaudeCode,
			available:          false,
			unavailableReason:  "claude CLI not installed",
			clientSourcePolicy: CLIOnlyPolicy(),
		}
		registry.Register(claudeCode)

		_, err := registry.SelectNavigator(ctx, ClientSourceCLI, NavigatorClaudeCode)
		if !errors.Is(err, ErrNavigatorNotAvailable) {
			t.Errorf("SelectNavigator() error = %v, want ErrNavigatorNotAvailable", err)
		}
	})

	t.Run("auto-select first available", func(t *testing.T) {
		registry := NewNavigatorRegistry()

		playwright := &mockNavigator{
			navType:            NavigatorPlaywright,
			available:          true,
			clientSourcePolicy: AllSourcesPolicy(),
		}
		claudeCode := &mockNavigator{
			navType:            NavigatorClaudeCode,
			available:          true,
			clientSourcePolicy: CLIOnlyPolicy(),
		}

		registry.Register(playwright)
		registry.Register(claudeCode)

		// Auto-select (no preferred type)
		got, err := registry.SelectNavigator(ctx, ClientSourceUI, "")
		if err != nil {
			t.Fatalf("SelectNavigator() error = %v", err)
		}
		// Should select playwright (first registered, and allows UI)
		if got.Type() != NavigatorPlaywright {
			t.Errorf("SelectNavigator() type = %v, want %v", got.Type(), NavigatorPlaywright)
		}
	})

	t.Run("auto-select skips unavailable", func(t *testing.T) {
		registry := NewNavigatorRegistry()

		unavailable := &mockNavigator{
			navType:            NavigatorPlaywright,
			available:          false,
			clientSourcePolicy: AllSourcesPolicy(),
		}
		available := &mockNavigator{
			navType:            NavigatorClaudeCode,
			available:          true,
			clientSourcePolicy: AllSourcesPolicy(),
		}

		registry.Register(unavailable)
		registry.Register(available)

		got, err := registry.SelectNavigator(ctx, ClientSourceCLI, "")
		if err != nil {
			t.Fatalf("SelectNavigator() error = %v", err)
		}
		if got.Type() != NavigatorClaudeCode {
			t.Errorf("SelectNavigator() type = %v, want %v", got.Type(), NavigatorClaudeCode)
		}
	})

	t.Run("auto-select skips not allowed sources", func(t *testing.T) {
		registry := NewNavigatorRegistry()

		cliOnly := &mockNavigator{
			navType:            NavigatorClaudeCode,
			available:          true,
			clientSourcePolicy: CLIOnlyPolicy(),
		}
		allSources := &mockNavigator{
			navType:            NavigatorPlaywright,
			available:          true,
			clientSourcePolicy: AllSourcesPolicy(),
		}

		registry.Register(cliOnly)
		registry.Register(allSources)

		// From UI, should skip CLI-only and select playwright
		got, err := registry.SelectNavigator(ctx, ClientSourceUI, "")
		if err != nil {
			t.Fatalf("SelectNavigator() error = %v", err)
		}
		if got.Type() != NavigatorPlaywright {
			t.Errorf("SelectNavigator() type = %v, want %v", got.Type(), NavigatorPlaywright)
		}
	})

	t.Run("no navigators available", func(t *testing.T) {
		registry := NewNavigatorRegistry()

		unavailable := &mockNavigator{
			navType:            NavigatorPlaywright,
			available:          false,
			clientSourcePolicy: AllSourcesPolicy(),
		}
		registry.Register(unavailable)

		_, err := registry.SelectNavigator(ctx, ClientSourceUI, "")
		if !errors.Is(err, ErrNoNavigatorsAvailable) {
			t.Errorf("SelectNavigator() error = %v, want ErrNoNavigatorsAvailable", err)
		}
	})

	t.Run("empty registry", func(t *testing.T) {
		registry := NewNavigatorRegistry()

		_, err := registry.SelectNavigator(ctx, ClientSourceUI, "")
		if !errors.Is(err, ErrNoNavigatorsAvailable) {
			t.Errorf("SelectNavigator() error = %v, want ErrNoNavigatorsAvailable", err)
		}
	})
}

func TestNavigatorRegistry_ListNavigators(t *testing.T) {
	ctx := t.Context()
	registry := NewNavigatorRegistry()

	playwright := &mockNavigator{
		navType:     NavigatorPlaywright,
		available:   true,
		description: "Playwright navigator",
		creditPolicy: CreditPolicy{
			RequiresCredits:  true,
			CreditsPerStep:   2,
			BypassConditions: []BypassCondition{BypassBYOK},
		},
		clientSourcePolicy: AllSourcesPolicy(),
	}

	claudeCode := &mockNavigator{
		navType:           NavigatorClaudeCode,
		available:         false,
		unavailableReason: "claude CLI not installed",
		description:       "Claude Code navigator",
		creditPolicy: CreditPolicy{
			RequiresCredits:  false,
			BypassConditions: []BypassCondition{BypassLocalExecution},
		},
		clientSourcePolicy: CLIOnlyPolicy(),
	}

	registry.Register(playwright)
	registry.Register(claudeCode)

	t.Run("list from UI client", func(t *testing.T) {
		infos := registry.ListNavigators(ctx, ClientSourceUI)

		if len(infos) != 2 {
			t.Fatalf("ListNavigators() returned %d items, want 2", len(infos))
		}

		// Check playwright info
		if infos[0].Type != NavigatorPlaywright {
			t.Errorf("infos[0].Type = %v, want %v", infos[0].Type, NavigatorPlaywright)
		}
		if !infos[0].Available {
			t.Error("infos[0].Available = false, want true")
		}
		if infos[0].CreditPolicy.RequiresCredits != true {
			t.Error("infos[0].CreditPolicy.RequiresCredits = false, want true")
		}

		// Check claude code info - not available for UI
		if infos[1].Type != NavigatorClaudeCode {
			t.Errorf("infos[1].Type = %v, want %v", infos[1].Type, NavigatorClaudeCode)
		}
		if infos[1].Available {
			t.Error("infos[1].Available = true, want false (not allowed for UI)")
		}
	})

	t.Run("list from CLI client", func(t *testing.T) {
		infos := registry.ListNavigators(ctx, ClientSourceCLI)

		// Claude code should still be unavailable (not installed), but for a different reason
		if infos[1].Available {
			t.Error("infos[1].Available = true, want false (not installed)")
		}
		if infos[1].UnavailableReason != "claude CLI not installed" {
			t.Errorf("infos[1].UnavailableReason = %q, want %q", infos[1].UnavailableReason, "claude CLI not installed")
		}
	})
}

func TestNavigatorRegistry_GetDefault(t *testing.T) {
	t.Run("empty registry", func(t *testing.T) {
		registry := NewNavigatorRegistry()

		if got := registry.GetDefault(); got != "" {
			t.Errorf("GetDefault() = %q, want empty string", got)
		}
	})

	t.Run("returns first registered", func(t *testing.T) {
		registry := NewNavigatorRegistry()

		registry.Register(&mockNavigator{navType: NavigatorPlaywright})
		registry.Register(&mockNavigator{navType: NavigatorClaudeCode})

		if got := registry.GetDefault(); got != NavigatorPlaywright {
			t.Errorf("GetDefault() = %v, want %v", got, NavigatorPlaywright)
		}
	})
}

func TestNavigatorRegistry_Count(t *testing.T) {
	registry := NewNavigatorRegistry()

	if got := registry.Count(); got != 0 {
		t.Errorf("Count() = %d, want 0", got)
	}

	registry.Register(&mockNavigator{navType: NavigatorPlaywright})
	if got := registry.Count(); got != 1 {
		t.Errorf("Count() = %d, want 1", got)
	}

	registry.Register(&mockNavigator{navType: NavigatorClaudeCode})
	if got := registry.Count(); got != 2 {
		t.Errorf("Count() = %d, want 2", got)
	}
}
