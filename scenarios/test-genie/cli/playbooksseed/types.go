package playbooksseed

// ApplyRequest triggers seed apply.
type ApplyRequest struct {
	Retain bool `json:"retain,omitempty"`

	// ScenarioPath overrides the scenario directory path. Set by the CLI
	// when running inside a sandboxed agent.
	// See packages/cli-core/cliutil/sandbox.go for sandbox path resolution.
	ScenarioPath string `json:"scenarioPath,omitempty"`
}

// ApplyResponse captures seed apply output.
type ApplyResponse struct {
	Status       string           `json:"status"`
	Scenario     string           `json:"scenario"`
	RunID        string           `json:"run_id"`
	SeedState    map[string]any   `json:"seed_state,omitempty"`
	CleanupToken string           `json:"cleanup_token"`
	Resources    []map[string]any `json:"resources,omitempty"`
}

// CleanupRequest triggers seed cleanup.
type CleanupRequest struct {
	CleanupToken string `json:"cleanup_token"`
}

// CleanupResponse captures cleanup output.
type CleanupResponse struct {
	Status   string `json:"status"`
	Scenario string `json:"scenario"`
	RunID    string `json:"run_id"`
}
