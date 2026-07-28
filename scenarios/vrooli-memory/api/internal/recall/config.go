package recall

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	DefaultFrontierTarget = 100
	DefaultWakeBudget     = 40
	FrontierTargetEnv     = "VROOLI_MEMORY_FRONTIER_TARGET"
	WakeBudgetEnv         = "VROOLI_MEMORY_WAKE_BUDGET"
)

// ConfigFromEnv loads independent compaction and prompt-size controls. A
// malformed value is an operator error: silently falling back would make the
// active memory budget impossible to reason about.
func ConfigFromEnv(lookupEnv func(string) (string, bool)) (Config, error) {
	config := Config{FrontierTarget: DefaultFrontierTarget, WakeBudget: DefaultWakeBudget}
	for _, setting := range []struct {
		name string
		set  func(int)
	}{
		{name: FrontierTargetEnv, set: func(value int) { config.FrontierTarget = value }},
		{name: WakeBudgetEnv, set: func(value int) { config.WakeBudget = value }},
	} {
		raw, ok := lookupEnv(setting.name)
		if !ok || strings.TrimSpace(raw) == "" {
			continue
		}
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			return Config{}, fmt.Errorf("%s must be a positive integer, got %q", setting.name, raw)
		}
		setting.set(value)
	}
	return config, nil
}
