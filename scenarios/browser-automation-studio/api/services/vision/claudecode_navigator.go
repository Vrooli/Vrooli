package vision

import (
	"context"
	"os/exec"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/services/credits"
)

// ClaudeCodeVisionNavigator implements VisionNavigator using Claude Code CLI.
// This is a stub implementation for future use - currently returns unavailable.
type ClaudeCodeVisionNavigator struct {
	log *logrus.Logger
}

// NewClaudeCodeVisionNavigator creates a new Claude Code-based navigator.
func NewClaudeCodeVisionNavigator(log *logrus.Logger) *ClaudeCodeVisionNavigator {
	return &ClaudeCodeVisionNavigator{
		log: log,
	}
}

// Type returns the navigator type.
func (n *ClaudeCodeVisionNavigator) Type() NavigatorType {
	return NavigatorClaudeCode
}

// Description returns a human-readable description.
func (n *ClaudeCodeVisionNavigator) Description() string {
	return "AI navigation using Claude Code CLI with Chrome"
}

// IsAvailable checks if Claude CLI is available.
func (n *ClaudeCodeVisionNavigator) IsAvailable(_ context.Context) bool {
	// Check if claude CLI is installed and meets version requirements
	_, err := exec.LookPath("claude")
	if err != nil {
		return false
	}

	// Future: check version with `claude --version` to ensure --chrome support
	// For now, always return false as this is a stub
	return false
}

// UnavailableReason returns why the navigator is unavailable.
func (n *ClaudeCodeVisionNavigator) UnavailableReason(_ context.Context) string {
	_, err := exec.LookPath("claude")
	if err != nil {
		return "claude CLI not found in PATH"
	}

	// Future: check version
	return "claude CLI --chrome support not yet implemented"
}

// CreditPolicy returns the credit policy for this navigator.
// Claude Code runs locally, so no credits are charged.
func (n *ClaudeCodeVisionNavigator) CreditPolicy() CreditPolicy {
	return CreditPolicy{
		RequiresCredits:  false,
		OperationType:    credits.OpAIVisionNavigate,
		PerStepCharging:  false,
		CreditsPerStep:   0,
		BypassConditions: []BypassCondition{BypassLocalExecution},
	}
}

// ClientSourcePolicy returns the client source policy (CLI only).
func (n *ClaudeCodeVisionNavigator) ClientSourcePolicy() ClientSourcePolicy {
	return CLIOnlyPolicy()
}

// Navigate starts an AI navigation session.
// Currently returns an error as this is a stub implementation.
func (n *ClaudeCodeVisionNavigator) Navigate(_ context.Context, _ NavigationRequest) (NavigationHandle, error) {
	return nil, ErrNavigatorNotAvailable
}
