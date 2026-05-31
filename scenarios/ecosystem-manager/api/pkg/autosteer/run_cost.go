package autosteer

// RunCost is the token cost of a single agent run, captured from the
// agent-manager run result and fed to the controller's reduction-per-token
// selection bandit (see pkg/effectiveness).
//
// seam: RunCost is the controller's token-cost input. Production wires it from
// agent-manager's run summary via ExecutionOrchestrator.RecordRunCost (called by
// the queue when a run completes); tests inject a RunCost directly.
//
// agent-manager today reports only a combined total (RunSummary.tokens_used), so
// InputTokens/OutputTokens are reserved for when the substrate exposes the split
// and TotalTokens is authoritative. A zero TotalTokens is an explicit "unknown"
// sentinel — the bandit does not treat an unknown cost as cheap (it falls back to
// net-findings-per-run; see pkg/effectiveness.ExpectedEfficacyPerToken).
type RunCost struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
	TotalTokens  int64 `json:"total_tokens"`
}

// Known reports whether the cost was actually measured (a positive total). An
// unknown cost is recorded faithfully rather than guessed.
func (c RunCost) Known() bool { return c.TotalTokens > 0 }
