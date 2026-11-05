package deliveryramp

import (
	"strings"
	"time"
)

const (
	JourneyCapabilityEnv = "S2D_JOURNEY_CAPABILITY"
	JourneyProfileEnv    = "S2D_JOURNEY_PROFILE"
)

type JourneyConfiguration struct {
	Capability string
	Profile    string
}

// ReadJourneyConfiguration keeps the existing environment names while
// allowing ramps and tests to supply configuration through any source.
func ReadJourneyConfiguration(getenv func(string) string) JourneyConfiguration {
	if getenv == nil {
		return JourneyConfiguration{}
	}
	return JourneyConfiguration{Capability: strings.TrimSpace(getenv(JourneyCapabilityEnv)), Profile: strings.TrimSpace(getenv(JourneyProfileEnv))}
}

type PacingProfile struct {
	ID          string          `json:"id"`
	Settle      SettlePolicy    `json:"settle"`
	StepTimeout ReadinessPolicy `json:"step_timeout"`
}

var pacingProfiles = map[string]PacingProfile{
	"normal-review": {
		ID:          "normal-review",
		Settle:      SettlePolicy{ID: "normal-settle", Reason: "allow the reviewed UI state to stabilize", Minimum: 250 * time.Millisecond, Maximum: 3 * time.Second, PollInterval: 50 * time.Millisecond, Cancellation: "context cancellation"},
		StepTimeout: ReadinessPolicy{ID: "normal-step", Reason: "bounded readiness wait", Timeout: 30 * time.Second, PollInterval: 100 * time.Millisecond, StabilityCount: 2, Cancellation: "context cancellation"},
	},
}

func PacingProfiles() map[string]PacingProfile {
	result := make(map[string]PacingProfile, len(pacingProfiles))
	for id, profile := range pacingProfiles {
		result[id] = profile
	}
	return result
}
