package vision

import (
	"errors"
	"io"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestNewClaudeCodeVisionNavigator(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	nav := NewClaudeCodeVisionNavigator(log)

	if nav == nil {
		t.Fatal("NewClaudeCodeVisionNavigator returned nil")
	}
	if nav.log != log {
		t.Error("logger not set correctly")
	}
}

func TestClaudeCodeVisionNavigator_Type(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	nav := NewClaudeCodeVisionNavigator(log)

	if nav.Type() != NavigatorClaudeCode {
		t.Errorf("Type() = %v, want %v", nav.Type(), NavigatorClaudeCode)
	}
}

func TestClaudeCodeVisionNavigator_Description(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	nav := NewClaudeCodeVisionNavigator(log)

	desc := nav.Description()
	if desc == "" {
		t.Error("Description() returned empty string")
	}
}

func TestClaudeCodeVisionNavigator_IsAvailable(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	nav := NewClaudeCodeVisionNavigator(log)

	// Currently always returns false since it's a stub
	available := nav.IsAvailable(t.Context())
	if available {
		t.Error("IsAvailable() should return false for stub implementation")
	}
}

func TestClaudeCodeVisionNavigator_UnavailableReason(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	nav := NewClaudeCodeVisionNavigator(log)

	reason := nav.UnavailableReason(t.Context())
	if reason == "" {
		t.Error("UnavailableReason() should return a reason for stub implementation")
	}
}

func TestClaudeCodeVisionNavigator_CreditPolicy(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	nav := NewClaudeCodeVisionNavigator(log)

	policy := nav.CreditPolicy()

	// Claude Code runs locally, so no credits required
	if policy.RequiresCredits {
		t.Error("RequiresCredits should be false for local execution")
	}
	if policy.CreditsPerStep != 0 {
		t.Errorf("CreditsPerStep = %d, want 0", policy.CreditsPerStep)
	}
	if len(policy.BypassConditions) != 1 {
		t.Errorf("BypassConditions length = %d, want 1", len(policy.BypassConditions))
	}
	if len(policy.BypassConditions) > 0 && policy.BypassConditions[0] != BypassLocalExecution {
		t.Errorf("BypassConditions[0] = %v, want %v", policy.BypassConditions[0], BypassLocalExecution)
	}
}

func TestClaudeCodeVisionNavigator_ClientSourcePolicy(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	nav := NewClaudeCodeVisionNavigator(log)

	policy := nav.ClientSourcePolicy()

	// Should only allow CLI
	if !policy.IsAllowed(ClientSourceCLI) {
		t.Error("should allow CLI")
	}
	if policy.IsAllowed(ClientSourceUI) {
		t.Error("should not allow UI")
	}
	if policy.IsAllowed(ClientSourceAPI) {
		t.Error("should not allow API")
	}
}

func TestClaudeCodeVisionNavigator_Navigate(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	nav := NewClaudeCodeVisionNavigator(log)

	// Should return ErrNavigatorNotAvailable since it's a stub
	_, err := nav.Navigate(t.Context(), NavigationRequest{
		SessionID: "session123",
		Prompt:    "Click button",
		Model:     "claude-sonnet-4",
	})

	if err == nil {
		t.Error("Navigate() should return error for stub implementation")
	}
	if !errors.Is(err, ErrNavigatorNotAvailable) {
		t.Errorf("Navigate() error = %v, want ErrNavigatorNotAvailable", err)
	}
}
