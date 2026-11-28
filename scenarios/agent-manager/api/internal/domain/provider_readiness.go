package domain

import "time"

// ProviderReadiness is the conservative, read-only view of a runner provider's
// lifecycle admission state. Unknown means the owner could not be verified.
type ProviderReadiness struct {
	Provider   string                 `json:"provider"`
	State      ProviderReadinessState `json:"state"`
	Known      bool                   `json:"known"`
	ObservedAt *time.Time             `json:"observed_at,omitempty"`
	Message    string                 `json:"message"`
}

type ProviderReadinessState string

const (
	ProviderReadinessOpen    ProviderReadinessState = "open"
	ProviderReadinessClosed  ProviderReadinessState = "closed"
	ProviderReadinessUnknown ProviderReadinessState = "unknown"
)
