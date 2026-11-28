//go:build integration
// +build integration

// Package runner_test contains integration tests that verify all three
// runners (Claude Code, Codex, OpenCode) produce expected events when
// executing real tasks against the live CLIs.
//
// These tests require the actual runner CLI binaries to be installed.
// Run with: go test -tags=integration ./internal/adapters/runner/...
package runner_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	runnercore "agent-manager/internal/adapters/runner/core"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// testConfig holds configuration for integration tests
type testConfig struct {
	tempDir string
}

func setupTest(t *testing.T) *testConfig {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "runner-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tempDir) })
	return &testConfig{tempDir: tempDir}
}

// eventCollector implements runner.EventSink to collect events.
type eventCollector struct {
	events []*domain.RunEvent
}

func (c *eventCollector) Emit(evt *domain.RunEvent) error {
	c.events = append(c.events, evt)
	return nil
}

func (c *eventCollector) Close() error { return nil }

// newClaudeRunner returns a generic core Runner driving the Claude codec.
func newClaudeRunner(t *testing.T) runner.Runner {
	t.Helper()
	codec, err := codecs.NewClaude()
	if err != nil {
		t.Fatalf("failed to create claude codec: %v", err)
	}
	return runnercore.NewRunner(codec, runner.NewHostLauncher(), nil)
}

// newCodexRunner returns a generic core Runner driving the Codex codec.
func newCodexRunner(t *testing.T) runner.Runner {
	t.Helper()
	codec, err := codecs.NewCodex()
	if err != nil {
		t.Fatalf("failed to create codex codec: %v", err)
	}
	return runnercore.NewRunner(codec, runner.NewHostLauncher(), nil)
}

// newOpenCodeRunner returns a generic core Runner driving the OpenCode codec.
func newOpenCodeRunner(t *testing.T) runner.Runner {
	t.Helper()
	codec, err := codecs.NewOpenCode()
	if err != nil {
		t.Fatalf("failed to create opencode codec: %v", err)
	}
	return runnercore.NewRunner(codec, runner.NewHostLauncher(), nil)
}

type providerIntegrationCase struct {
	newRunner       func(*testing.T) runner.Runner
	fileName        string
	content         string
	toolName        string
	requireMessage  bool
	messageOptional bool
}

var fileWriteCases = map[string]providerIntegrationCase{
	"ClaudeCode": {newRunner: newClaudeRunner, fileName: "test-claude.txt", content: "Hello from Claude Code integration test", toolName: "Write", requireMessage: true},
	"Codex":      {newRunner: newCodexRunner, fileName: "test-codex.txt", content: "Hello from Codex integration test", toolName: "file_change", messageOptional: true},
	"OpenCode":   {newRunner: newOpenCodeRunner, fileName: "test-opencode.txt", content: "Hello from OpenCode integration test", toolName: "write"},
}

func TestIntegration_ClaudeCode_FileWrite(t *testing.T) {
	runFileWriteIntegration(t, fileWriteCases["ClaudeCode"])
}
func TestIntegration_Codex_FileWrite(t *testing.T) {
	runFileWriteIntegration(t, fileWriteCases["Codex"])
}
func TestIntegration_OpenCode_FileWrite(t *testing.T) {
	runFileWriteIntegration(t, fileWriteCases["OpenCode"])
}

func runFileWriteIntegration(t *testing.T, tc providerIntegrationCase) {
	t.Helper()
	r := tc.newRunner(t)
	if available, msg := r.IsAvailable(context.Background()); !available {
		t.Skipf("runner not available: %s", msg)
	}
	cfg := setupTest(t)
	collector := &eventCollector{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	result, err := r.Execute(ctx, runner.ExecuteRequest{
		RunID: uuid.New(), Prompt: "Create a file called " + tc.fileName + " with the content: " + tc.content,
		WorkingDir: cfg.tempDir, EventSink: collector,
	})
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got failure: %s", result.ErrorMessage)
	}
	content, err := os.ReadFile(filepath.Join(cfg.tempDir, tc.fileName))
	if err != nil {
		t.Fatalf("failed to read created file: %v", err)
	}
	if string(content) != tc.content+"\n" && string(content) != tc.content {
		t.Errorf("expected content %q, got %q", tc.content, content)
	}
	assertHasToolCallEvent(t, collector.events, tc.toolName)
	assertProviderMessage(t, collector.events, tc)
	assertHasMetricEvent(t, collector.events)
}

var sandboxDiffCases = map[string]providerIntegrationCase{
	"ClaudeCode": {newRunner: newClaudeRunner, fileName: "tmp/integration-claude.txt", content: "Hello from Claude Code sandbox integration test", toolName: "Write", requireMessage: true},
	"Codex":      {newRunner: newCodexRunner, fileName: "tmp/integration-codex.txt", content: "Hello from Codex sandbox integration test", toolName: "file_change", messageOptional: true},
	"OpenCode":   {newRunner: newOpenCodeRunner, fileName: "tmp/integration-opencode.txt", content: "Hello from OpenCode sandbox integration test", toolName: "write"},
}

func TestIntegration_ClaudeCode_SandboxDiff(t *testing.T) {
	runSandboxDiffIntegration(t, sandboxDiffCases["ClaudeCode"])
}
func TestIntegration_Codex_SandboxDiff(t *testing.T) {
	runSandboxDiffIntegration(t, sandboxDiffCases["Codex"])
}
func TestIntegration_OpenCode_SandboxDiff(t *testing.T) {
	runSandboxDiffIntegration(t, sandboxDiffCases["OpenCode"])
}

func runSandboxDiffIntegration(t *testing.T, tc providerIntegrationCase) {
	t.Helper()
	r := tc.newRunner(t)
	if available, msg := r.IsAvailable(context.Background()); !available {
		t.Skipf("runner not available: %s", msg)
	}
	provider, sandboxRef := setupWorkspaceSandbox(t, "tmp")
	collector := &eventCollector{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	result, err := r.Execute(ctx, runner.ExecuteRequest{
		RunID: uuid.New(), Prompt: "Create a file called " + tc.fileName + " with the content: " + tc.content,
		WorkingDir: sandboxRef.WorkDir, EventSink: collector,
	})
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got failure: %s", result.ErrorMessage)
	}
	diff, err := provider.GetDiff(ctx, sandboxRef.ID)
	if err != nil {
		t.Fatalf("failed to get diff: %v", err)
	}
	if !strings.Contains(diff.UnifiedDiff, tc.fileName) {
		t.Errorf("expected diff to include %s", tc.fileName)
	}
	assertHasToolCallEvent(t, collector.events, tc.toolName)
	assertProviderMessage(t, collector.events, tc)
	assertHasMetricEvent(t, collector.events)
}

func assertProviderMessage(t *testing.T, events []*domain.RunEvent, tc providerIntegrationCase) {
	t.Helper()
	if tc.requireMessage {
		assertHasMessageEventRequired(t, events, "assistant")
	} else if tc.messageOptional {
		assertHasMessageEvent(t, events, "assistant")
	}
}

// Assertion helpers

func assertHasToolCallEvent(t *testing.T, events []*domain.RunEvent, expectedToolName string) {
	t.Helper()
	for _, evt := range events {
		if evt.EventType == domain.EventTypeToolCall {
			if toolData, ok := evt.Data.(*domain.ToolCallEventData); ok {
				if toolData.ToolName == expectedToolName {
					if len(toolData.Input) == 0 {
						t.Errorf("tool_call event for '%s' has empty input", expectedToolName)
					}
					return
				}
			}
		}
	}
	t.Errorf("no tool_call event found for tool '%s'", expectedToolName)
}

func assertHasMessageEvent(t *testing.T, events []*domain.RunEvent, expectedRole string) {
	t.Helper()
	for _, evt := range events {
		if evt.EventType == domain.EventTypeMessage {
			if msgData, ok := evt.Data.(*domain.MessageEventData); ok {
				if msgData.Role == expectedRole {
					if msgData.Content == "" {
						t.Errorf("message event for role '%s' has empty content", expectedRole)
					}
					return
				}
			}
		}
	}
	t.Logf("WARN: no message event found for role '%s' (some runners may not emit these)", expectedRole)
}

func assertHasMessageEventRequired(t *testing.T, events []*domain.RunEvent, expectedRole string) {
	t.Helper()
	for _, evt := range events {
		if evt.EventType == domain.EventTypeMessage {
			if msgData, ok := evt.Data.(*domain.MessageEventData); ok {
				if msgData.Role == expectedRole && msgData.Content != "" {
					return
				}
			}
		}
	}
	t.Fatalf("expected message event for role '%s' with non-empty content", expectedRole)
}

func assertHasMetricEvent(t *testing.T, events []*domain.RunEvent) {
	t.Helper()
	for _, evt := range events {
		if evt.EventType == domain.EventTypeMetric {
			if usageData, ok := evt.Data.(*domain.UsageEventData); ok {
				if usageData.InputTokens == 0 && usageData.OutputTokens == 0 {
					t.Error("metric event has zero tokens")
				}
				return
			}
			if _, ok := evt.Data.(*domain.MetricEventData); ok {
				return
			}
		}
	}
	t.Error("no metric event found")
}

func setupWorkspaceSandbox(t *testing.T, scopePath string) (*sandbox.WorkspaceSandboxProvider, *sandbox.Sandbox) {
	t.Helper()
	baseURL := os.Getenv("WORKSPACE_SANDBOX_URL")
	if baseURL == "" {
		t.Skip("WORKSPACE_SANDBOX_URL not set")
	}

	projectRoot := findRepoRoot(t)
	provider := sandbox.NewWorkspaceSandboxProvider(baseURL)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sandboxRef, err := provider.Create(ctx, sandbox.CreateRequest{
		ScopePath:   scopePath,
		ProjectRoot: projectRoot,
		Owner:       "runner-integration",
		OwnerType:   "test",
	})
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}

	t.Cleanup(func() {
		_ = provider.Delete(context.Background(), sandboxRef.ID)
	})

	return provider, sandboxRef
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}

	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	t.Skip("unable to locate repo root (AGENTS.md not found)")
	return ""
}
