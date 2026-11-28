package recall

import "source-ledger/internal/policy"

const (
	DefaultFrontierTarget  = policy.DefaultFrontierTarget
	DefaultWakeBudget      = policy.DefaultWakeBudget
	DefaultWakeBudgetChars = policy.DefaultWakeBudgetChars
	DefaultMaxEntryLines   = policy.DefaultMaxEntryLines
	DefaultMaxEntryChars   = policy.DefaultMaxEntryChars
)

// ConfigFromPolicy copies the engine-independent policy into recall's local
// rendering shape. Policy is resolved per request by the registry; this helper
// exists for composition-root defaults and tests.
func ConfigFromPolicy(c policy.Config) Config {
	return Config{
		FrontierTarget: c.FrontierTarget, WakeBudget: c.WakeBudget,
		WakeBudgetChars: c.WakeBudgetChars, MaxEntryLines: c.MaxEntryLines,
		MaxEntryChars: c.MaxEntryChars, FacetBudgets: c.FacetBudgets,
	}
}
