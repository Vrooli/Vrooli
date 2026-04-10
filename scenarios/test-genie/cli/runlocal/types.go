// Package runlocal provides local test runner triggering capabilities.
package runlocal

// Request represents a local test run request.
type Request struct {
	Type      string   `json:"type,omitempty"`
	Paths     []string `json:"paths,omitempty"`
	Playbooks []string `json:"playbooks,omitempty"`
	Filter    string   `json:"filter,omitempty"`

	// ScenarioPath overrides the scenario directory path. Set by the CLI
	// when running inside a sandboxed agent, pointing to the overlay's merged
	// directory. When empty, the API resolves via VROOLI_ROOT + scenario name.
	// See packages/cli-core/cliutil/sandbox.go for sandbox path resolution.
	ScenarioPath string `json:"scenarioPath,omitempty"`
}

// Response represents the local test run response.
type Response struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	LogPath string `json:"logPath"`
	Command struct {
		Command    []string `json:"command"`
		WorkingDir string   `json:"workingDir"`
	} `json:"command"`
}

// Args holds parsed CLI inputs for the run-tests command.
type Args struct {
	Scenario  string
	Type      string
	Paths     []string
	Playbooks []string
	Filter    string
	JSON      bool
}
