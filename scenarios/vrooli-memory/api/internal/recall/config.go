package recall

import "vrooli-memory/internal/policy"

const (
	DefaultFrontierTarget = policy.DefaultFrontierTarget
	DefaultWakeBudget     = policy.DefaultWakeBudget
	DefaultMaxEntryLines  = policy.DefaultMaxEntryLines
	FrontierTargetEnv     = policy.FrontierTargetEnv
	WakeBudgetEnv         = policy.WakeBudgetEnv
	MaxEntryLinesEnv      = policy.MaxEntryLinesEnv
)

// ConfigFromEnv loads independent compaction and prompt-size controls. A
// malformed value is an operator error: silently falling back would make the
// active memory budget impossible to reason about.
func ConfigFromEnv(lookupEnv func(string) (string, bool)) (Config, error) {
	c, err := policy.Resolve(lookupEnv)
	if err != nil {
		return Config{}, err
	}
	return Config{FrontierTarget: c.FrontierTarget, WakeBudget: c.WakeBudget, MaxEntryLines: c.MaxEntryLines}, nil
}
