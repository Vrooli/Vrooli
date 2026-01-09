// Package runlocal provides local test runner triggering capabilities.
package runlocal

// Request represents a local test run request.
type Request struct {
	Type      string   `json:"type,omitempty"`
	Paths     []string `json:"paths,omitempty"`
	Playbooks []string `json:"playbooks,omitempty"`
	Filter    string   `json:"filter,omitempty"`
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
