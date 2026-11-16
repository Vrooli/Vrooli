package vision

// CreditPolicy describes the meter surfaced to clients. Authorization and
// usage reservation/charging are exclusively owned by AI Gateway.
type CreditPolicy struct {
	// RequiresCredits indicates account-backed use may be metered by Gateway.
	RequiresCredits bool `json:"requires_credits"`

	// PerStepCharging indicates whether credits are charged per step.
	PerStepCharging bool `json:"per_step_charging"`

	// CreditsPerStep is a fixed price hint; zero means Gateway determines usage.
	CreditsPerStep int `json:"credits_per_step"`
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
