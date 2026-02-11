package runner

import (
	"os/exec"

	"github.com/google/uuid"
)

// BuildArgsForTest exposes ClaudeCodeRunner.buildArgs for external testing.
func (r *ClaudeCodeRunner) BuildArgsForTest(req ExecuteRequest) []string {
	return r.buildArgs(req)
}

// NewTestClaudeCodeRunner creates a ClaudeCodeRunner for testing without
// checking binary availability.
func NewTestClaudeCodeRunner() *ClaudeCodeRunner {
	return &ClaudeCodeRunner{
		binaryPath:  "/fake/path",
		available:   false,
		message:     "test runner",
		runs:        make(map[uuid.UUID]*exec.Cmd),
		streamState: make(map[uuid.UUID]*claudeStreamState),
	}
}
