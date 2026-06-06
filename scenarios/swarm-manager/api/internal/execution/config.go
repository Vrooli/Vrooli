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

	// BaselineRegressionGateEnabled makes a detected regression — a surface
	// that passed in the pre-execution baseline and fails now — gate the
	// finalization outcome: the item is routed to needs-fixup / hand-back
	// (and auto-fixup when policy allows) instead of being accepted, even when
	// the absolute review came back ready. This is the swarm-manager half of
	// the Baseline Modes promote gate (plan P6 §200-201): the before/after
	// verdict is no longer merely recorded for the review agent, it decides
	// whether the change keeps or is handed back.
	//
	// Subordinate to BaselineDiffEnabled (no diff is computed ⇒ nothing to
	// gate). Only a genuine "regression" verdict gates; new-failure,
	// pre-existing, and not-comparable verdicts do not (they are not
	// attributable to this change). Default true; set false to observe
	// regressions (still recorded + warned) without enforcing the gate — the
	// shadow-observe-then-enforce rollout lever for the reflexive kernel.
	BaselineRegressionGateEnabled bool

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
		BaselineRegressionGateEnabled:   true,
		BaselineDiffTimeout:             15 * time.Minute,
	}
}
