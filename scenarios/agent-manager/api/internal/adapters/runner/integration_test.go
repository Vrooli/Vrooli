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

// TestIntegration_ClaudeCode_FileWrite verifies Claude Code can create
// files and produces proper events.
func TestIntegration_ClaudeCode_FileWrite(t *testing.T) {
	r := newClaudeRunner(t)

	available, msg := r.IsAvailable(context.Background())
	if !available {
		t.Skipf("Claude Code runner not available: %s", msg)
	}

	cfg := setupTest(t)
	testFile := filepath.Join(cfg.tempDir, "test-claude.txt")
	expectedContent := "Hello from Claude Code integration test"

	collector := &eventCollector{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := r.Execute(ctx, runner.ExecuteRequest{
		RunID:      uuid.New(),
		Prompt:     "Create a file called test-claude.txt with the content: " + expectedContent,
		WorkingDir: cfg.tempDir,
		EventSink:  collector,
	})
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got failure: %s", result.ErrorMessage)
	}

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Errorf("failed to read created file: %v", err)
	}
	if string(content) != expectedContent+"\n" && string(content) != expectedContent {
		t.Errorf("expected content '%s', got '%s'", expectedContent, string(content))
	}

	assertHasToolCallEvent(t, collector.events, "Write")
	assertHasMessageEventRequired(t, collector.events, "assistant")
	assertHasMetricEvent(t, collector.events)
}

// TestIntegration_Codex_FileWrite verifies the Codex runner can create
// files and produces proper events.
func TestIntegration_Codex_FileWrite(t *testing.T) {
	r := newCodexRunner(t)

	available, msg := r.IsAvailable(context.Background())
	if !available {
		t.Skipf("Codex runner not available: %s", msg)
	}

	cfg := setupTest(t)
	testFile := filepath.Join(cfg.tempDir, "test-codex.txt")
	expectedContent := "Hello from Codex integration test"

	collector := &eventCollector{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := r.Execute(ctx, runner.ExecuteRequest{
		RunID:      uuid.New(),
		Prompt:     "Create a file called test-codex.txt with the content: " + expectedContent,
		WorkingDir: cfg.tempDir,
		EventSink:  collector,
	})
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got failure: %s", result.ErrorMessage)
	}

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Errorf("failed to read created file: %v", err)
	}
	if string(content) != expectedContent+"\n" && string(content) != expectedContent {
		t.Errorf("expected content '%s', got '%s'", expectedContent, string(content))
	}

	assertHasToolCallEvent(t, collector.events, "file_change")
	assertHasMessageEvent(t, collector.events, "assistant")
	assertHasMetricEvent(t, collector.events)
}

// TestIntegration_OpenCode_FileWrite verifies the OpenCode runner can
// create files and produces proper events.
func TestIntegration_OpenCode_FileWrite(t *testing.T) {
	r := newOpenCodeRunner(t)

	available, msg := r.IsAvailable(context.Background())
	if !available {
		t.Skipf("OpenCode runner not available: %s", msg)
	}

	cfg := setupTest(t)
	testFile := filepath.Join(cfg.tempDir, "test-opencode.txt")
	expectedContent := "Hello from OpenCode integration test"

	collector := &eventCollector{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := r.Execute(ctx, runner.ExecuteRequest{
		RunID:      uuid.New(),
		Prompt:     "Create a file called test-opencode.txt with the content: " + expectedContent,
		WorkingDir: cfg.tempDir,
		EventSink:  collector,
	})
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got failure: %s", result.ErrorMessage)
	}

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Errorf("failed to read created file: %v", err)
	}
	if string(content) != expectedContent+"\n" && string(content) != expectedContent {
		t.Errorf("expected content '%s', got '%s'", expectedContent, string(content))
	}

	assertHasToolCallEvent(t, collector.events, "write")
	assertHasMetricEvent(t, collector.events)
}

// TestIntegration_ClaudeCode_SandboxDiff verifies sandbox diff capture
// with Claude Code.
func TestIntegration_ClaudeCode_SandboxDiff(t *testing.T) {
	r := newClaudeRunner(t)

	available, msg := r.IsAvailable(context.Background())
	if !available {
		t.Skipf("Claude Code runner not available: %s", msg)
	}

	provider, sandboxRef := setupWorkspaceSandbox(t, "tmp")
	testFile := "tmp/integration-claude.txt"
	expectedContent := "Hello from Claude Code sandbox integration test"

	collector := &eventCollector{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := r.Execute(ctx, runner.ExecuteRequest{
		RunID:      uuid.New(),
		Prompt:     "Create a file called " + testFile + " with the content: " + expectedContent,
		WorkingDir: sandboxRef.WorkDir,
		EventSink:  collector,
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
	if !strings.Contains(diff.UnifiedDiff, testFile) {
		t.Errorf("expected diff to include %s", testFile)
	}

	assertHasToolCallEvent(t, collector.events, "Write")
	assertHasMessageEventRequired(t, collector.events, "assistant")
	assertHasMetricEvent(t, collector.events)
}

// TestIntegration_Codex_SandboxDiff verifies sandbox diff capture with Codex.
func TestIntegration_Codex_SandboxDiff(t *testing.T) {
	r := newCodexRunner(t)

	available, msg := r.IsAvailable(context.Background())
	if !available {
		t.Skipf("Codex runner not available: %s", msg)
	}

	provider, sandboxRef := setupWorkspaceSandbox(t, "tmp")
	testFile := "tmp/integration-codex.txt"
	expectedContent := "Hello from Codex sandbox integration test"

	collector := &eventCollector{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := r.Execute(ctx, runner.ExecuteRequest{
		RunID:      uuid.New(),
		Prompt:     "Create a file called " + testFile + " with the content: " + expectedContent,
		WorkingDir: sandboxRef.WorkDir,
		EventSink:  collector,
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
	if !strings.Contains(diff.UnifiedDiff, testFile) {
		t.Errorf("expected diff to include %s", testFile)
	}

	assertHasToolCallEvent(t, collector.events, "file_change")
	assertHasMessageEvent(t, collector.events, "assistant")
	assertHasMetricEvent(t, collector.events)
}

// TestIntegration_OpenCode_SandboxDiff verifies sandbox diff capture
// with OpenCode.
func TestIntegration_OpenCode_SandboxDiff(t *testing.T) {
	r := newOpenCodeRunner(t)

	available, msg := r.IsAvailable(context.Background())
	if !available {
		t.Skipf("OpenCode runner not available: %s", msg)
	}

	provider, sandboxRef := setupWorkspaceSandbox(t, "tmp")
	testFile := "tmp/integration-opencode.txt"
	expectedContent := "Hello from OpenCode sandbox integration test"

	collector := &eventCollector{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := r.Execute(ctx, runner.ExecuteRequest{
		RunID:      uuid.New(),
		Prompt:     "Create a file called " + testFile + " with the content: " + expectedContent,
		WorkingDir: sandboxRef.WorkDir,
		EventSink:  collector,
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
	if !strings.Contains(diff.UnifiedDiff, testFile) {
		t.Errorf("expected diff to include %s", testFile)
	}

	assertHasToolCallEvent(t, collector.events, "write")
	assertHasMessageEvent(t, collector.events, "assistant")
	assertHasMetricEvent(t, collector.events)
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
			if costData, ok := evt.Data.(*domain.CostEventData); ok {
				if costData.InputTokens == 0 && costData.OutputTokens == 0 {
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
