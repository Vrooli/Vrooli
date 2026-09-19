// Package intelligence owns AI-provider contracts and provider-specific error semantics.
package intelligence

import "errors"

// ErrProvider indicates that the upstream AI provider rejected or could not
// complete a request. HTTP adapters map this domain error to Bad Gateway.
var ErrProvider = errors.New("AI provider error")

// MeteredInferenceProvider errors are owned by the intelligence domain even when the HTTP
// transport maps them to a public API response. Keeping their identities here
// prevents provider, gateway, and credit orchestration code from depending on
// the API composition package.
var (
	ErrInsufficientCredits         = errors.New("insufficient credits for this operation")
	ErrNoAPIKeyConfigured          = errors.New("no OpenRouter API key configured")
	ErrModelNotAllowed             = errors.New("model not in allowed list")
	ErrRoleNotAllowed              = errors.New("inference role not allowed")
	ErrMeteredInferenceUnavailable = errors.New("metered inference provider service unavailable")
	ErrStreamingNotSupported       = errors.New("streaming not supported by client")
)
