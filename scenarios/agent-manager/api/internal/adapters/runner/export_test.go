package runner

import (
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
		launched:    make(map[uuid.UUID]LaunchedProcess),
		streamState: make(map[uuid.UUID]*claudeStreamState),
		selector:    newLauncherSelector(NewHostLauncher(), nil),
	}
}

// BuildEnvForTest exposes ClaudeCodeRunner.buildEnv for external testing.
func (r *ClaudeCodeRunner) BuildEnvForTest(req ExecuteRequest) []string {
	return r.buildEnv(req)
}

// BuildJSONArgsForTest exposes CodexRunner.buildJSONArgs for external testing.
func (r *CodexRunner) BuildJSONArgsForTest(req ExecuteRequest) []string {
	return r.buildJSONArgs(req)
}

// NewTestCodexRunner creates a CodexRunner for testing without
// checking binary availability.
func NewTestCodexRunner() *CodexRunner {
	return &CodexRunner{
		codexCLIPath:  "/fake/codex",
		available:     false,
		message:       "test runner",
		launched:      make(map[uuid.UUID]LaunchedProcess),
		runModels:     make(map[uuid.UUID]string),
		useJSONStream: true,
		selector:      newLauncherSelector(NewHostLauncher(), nil),
	}
}

// BuildArgsForOpenCodeTest exposes OpenCodeRunner.buildArgs for external testing.
func (r *OpenCodeRunner) BuildArgsForTest(req ExecuteRequest) []string {
	return r.buildArgs(req)
}

// NewTestOpenCodeRunner creates an OpenCodeRunner for testing without
// checking binary availability.
func NewTestOpenCodeRunner() *OpenCodeRunner {
	return &OpenCodeRunner{
		binaryPath:    "/fake/opencode",
		available:     false,
		message:       "test runner",
		launched:      make(map[uuid.UUID]LaunchedProcess),
		runSessionIDs: make(map[uuid.UUID]string),
		selector:      newLauncherSelector(NewHostLauncher(), nil),
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
