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

	// BaselineDiffEnabled turns the before/after baseline-diff feature on.
	// When true (and a BaselineClient is configured), execution captures a
	// pre-execution GCT baseline for each declared scenario and diffs it
	// during finalization so the review agent can tell regressions this item
	// caused apart from pre-existing failures. Acts as a kill-switch: when
	// false, the pipeline degrades to the absolute-threshold review only.
	BaselineDiffEnabled bool

	// BaselineRetainAfterFinalization keeps pre-execution baselines on disk
	// after finalization instead of deleting them. Default false (delete) so
	// baselines do not accumulate; set true for debugging or audit.
	BaselineRetainAfterFinalization bool

	// BaselineDiffTimeout bounds a single baseline diff call during
	// finalization (the diff re-runs test-genie surfaces against the working
	// tree and can take minutes).
	BaselineDiffTimeout time.Duration
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

		BaselineDiffEnabled:             true,
		BaselineRetainAfterFinalization: false,
		BaselineDiffTimeout:             15 * time.Minute,
	}
}
