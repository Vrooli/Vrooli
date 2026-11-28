package identity

import "fmt"

// ExecutionLimits is the reviewed aggregate coding allowance. It does not
// authorize product inference, customer billing, or another effect.
type ExecutionLimits struct {
	MaxSlices         int   `json:"max_slices"`
	MaxTokens         int64 `json:"max_tokens"`
	MaxWallSeconds    int64 `json:"max_wall_seconds"`
	MaxTurns          int   `json:"max_turns"`
	MaxChargeMicroUSD int64 `json:"max_charge_micro_usd"`
	MaxChildren       int   `json:"max_children"`
	MaxNodeAttempts   int   `json:"max_node_attempts"`
	MaxRetries        int   `json:"max_retries"`
}

func (limits *ExecutionLimits) Validate() error {
	if limits == nil {
		return nil
	}
	for _, field := range []struct {
		name           string
		value, maximum int64
	}{
		{"max_slices", int64(limits.MaxSlices), 512},
		{"max_tokens", limits.MaxTokens, 2147483647},
		{"max_wall_seconds", limits.MaxWallSeconds, 604800},
		{"max_turns", int64(limits.MaxTurns), 100000},
		{"max_charge_micro_usd", limits.MaxChargeMicroUSD, 1000000000000},
		{"max_children", int64(limits.MaxChildren), 4096},
		{"max_node_attempts", int64(limits.MaxNodeAttempts), 8192},
		{"max_retries", int64(limits.MaxRetries), 4096},
	} {
		if field.value < 1 || field.value > field.maximum {
			return fmt.Errorf("execution_limits.%s must be between 1 and %d", field.name, field.maximum)
		}
	}
	return nil
}

func (limits *ExecutionLimits) Clone() *ExecutionLimits {
	if limits == nil {
		return nil
	}
	copy := *limits
	return &copy
}
