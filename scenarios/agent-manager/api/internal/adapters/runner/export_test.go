package runner

import (
	"os/exec"

	"agent-manager/internal/domain"

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

// ParseCompactCommandForTest exposes parseCompactCommand for external testing.
func ParseCompactCommandForTest(content string) (bool, string) {
	return parseCompactCommand(content)
}

// IsCompactionSummaryForTest exposes isCompactionSummary for external testing.
func IsCompactionSummaryForTest(content string) bool {
	return isCompactionSummary(content)
}

// ExtractSummaryContentForTest exposes extractSummaryContent for external testing.
func ExtractSummaryContentForTest(content string) string {
	return extractSummaryContent(content)
}

// ParseStreamEventsForTest exposes parseStreamEvents for external testing.
func (r *ClaudeCodeRunner) ParseStreamEventsForTest(runID uuid.UUID, line string) ([]*domain.RunEvent, error) {
	return r.parseStreamEvents(runID, line)
}
