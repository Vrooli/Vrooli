package vision

import (
	"github.com/vrooli/browser-automation-studio/services/credits"
)

// BypassCondition represents a condition under which credit charging is bypassed.
type BypassCondition string

const (
	// BypassBYOK bypasses credit charging when the user provides their own API key.
	BypassBYOK BypassCondition = "byok"

	// BypassResourceOpenrouter bypasses credit charging when using resource openrouter.
	BypassResourceOpenrouter BypassCondition = "resource_openrouter"

	// BypassLocalExecution bypasses credit charging for local-only execution.
	BypassLocalExecution BypassCondition = "local_execution"
)

// CreditPolicy defines how credits are charged for a navigator.
type CreditPolicy struct {
	// RequiresCredits indicates whether this navigator charges credits.
	RequiresCredits bool

	// OperationType is the credit operation type for charging.
	OperationType credits.OperationType

	// PerStepCharging indicates whether credits are charged per step.
	PerStepCharging bool

	// CreditsPerStep is the number of credits charged per navigation step.
	CreditsPerStep int

	// BypassConditions lists conditions that bypass credit charging.
	BypassConditions []BypassCondition
}

// ShouldChargeCredits determines if credits should be charged given the conditions.
func (p CreditPolicy) ShouldChargeCredits(isBYOK, hasResourceOpenrouter, isLocalExecution bool) bool {
	if !p.RequiresCredits {
		return false
	}

	for _, condition := range p.BypassConditions {
		switch condition {
		case BypassBYOK:
			if isBYOK {
				return false
			}
		case BypassResourceOpenrouter:
			if hasResourceOpenrouter {
				return false
			}
		case BypassLocalExecution:
			if isLocalExecution {
				return false
			}
		}
	}

	return true
}

// ToInfo converts the policy to a JSON-serializable form.
func (p CreditPolicy) ToInfo() CreditPolicyInfo {
	return CreditPolicyInfo{
		RequiresCredits:  p.RequiresCredits,
		CreditsPerStep:   p.CreditsPerStep,
		BypassConditions: p.BypassConditions,
	}
}

// ClientSource identifies the source of a client request.
type ClientSource string

const (
	// ClientSourceUI is the web UI.
	ClientSourceUI ClientSource = "ui"

	// ClientSourceCLI is the command-line interface.
	ClientSourceCLI ClientSource = "cli"

	// ClientSourceAPI is a direct API call.
	ClientSourceAPI ClientSource = "api"
)

// ClientSourceFromHeader parses the X-Client-Source header value.
func ClientSourceFromHeader(header string) ClientSource {
	switch header {
	case "cli":
		return ClientSourceCLI
	case "ui":
		return ClientSourceUI
	case "api":
		return ClientSourceAPI
	default:
		// Default to API for unknown or missing headers
		return ClientSourceAPI
	}
}

// ClientSourcePolicy defines which client sources can use a navigator.
type ClientSourcePolicy struct {
	// AllowedSources lists the allowed client sources.
	// If empty, all sources are allowed.
	AllowedSources []ClientSource
}

// IsAllowed checks if the given client source is allowed.
func (p ClientSourcePolicy) IsAllowed(source ClientSource) bool {
	if len(p.AllowedSources) == 0 {
		// Empty list means all sources are allowed
		return true
	}

	for _, allowed := range p.AllowedSources {
		if allowed == source {
			return true
		}
	}

	return false
}

// AllSourcesPolicy returns a policy that allows all client sources.
func AllSourcesPolicy() ClientSourcePolicy {
	return ClientSourcePolicy{AllowedSources: nil}
}

// CLIOnlyPolicy returns a policy that only allows CLI clients.
func CLIOnlyPolicy() ClientSourcePolicy {
	return ClientSourcePolicy{AllowedSources: []ClientSource{ClientSourceCLI}}
}
