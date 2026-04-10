package execution

import "time"

// FinalizationConfig holds tunable parameters for the post-run finalization
// workflow (restart, health check, review). All fields have safe defaults
// provided by DefaultFinalizationConfig.
type FinalizationConfig struct {
	// HealthPollInterval is how often to re-check scenario health during
	// the health-check phase.
	HealthPollInterval time.Duration

	// HealthPollTimeout is the maximum time to wait for a scenario to
	// become healthy after restart.
	HealthPollTimeout time.Duration

	// ReviewPollInterval is how often to re-check review job status.
	ReviewPollInterval time.Duration

	// ReviewPollTimeout is the maximum time to wait for a review job to
	// complete.
	ReviewPollTimeout time.Duration

	// MaxRestartAttempts is how many times to retry restarting a scenario
	// before giving up.
	MaxRestartAttempts int
}

// DefaultFinalizationConfig returns the production defaults for finalization
// tuning.
func DefaultFinalizationConfig() FinalizationConfig {
	return FinalizationConfig{
		HealthPollInterval: 5 * time.Second,
		HealthPollTimeout:  2 * time.Minute,
		ReviewPollInterval: 5 * time.Second,
		ReviewPollTimeout:  10 * time.Minute,
		MaxRestartAttempts: 2,
	}
}
