//go:build integration
// +build integration

// Package runner provides runner adapter implementations.
//
// This file contains integration tests that verify all three runners
// (Claude Code, Codex, OpenCode) produce expected events when executing real tasks.
//
// These tests require the actual runner resources to be available.
// Run with: go test -tags=integration ./internal/adapters/runner/...
package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
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

// eventCollector implements EventSink to collect events during a run
type eventCollector struct {
	events []*domain.RunEvent
}

func (c *eventCollector) Emit(evt *domain.RunEvent) {
	c.events = append(c.events, evt)
}

func (c *eventCollector) Close() {}

// TestIntegration_ClaudeCode_FileWrite verifies Claude Code runner can create files
// and produces proper events.
func TestIntegration_ClaudeCode_FileWrite(t *testing.T) {
	runner, err := NewClaudeCodeRunner()
	if err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	available, msg := runner.IsAvailable(context.Background())
	if !available {
		t.Skipf("Claude Code runner not available: %s", msg)
	}

	cfg := setupTest(t)
	testFile := filepath.Join(cfg.tempDir, "test-claude.txt")
	expectedContent := "Hello from Claude Code integration test"

	collector := &eventCollector{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := runner.Execute(ctx, ExecuteRequest{
		RunID:      uuid.New(),
		Prompt:     "Create a file called test-claude.txt with the content: " + expectedContent,
		WorkingDir: cfg.tempDir,
		EventSink:  collector,
	})
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	// Verify success
	if !result.Success {
		t.Errorf("expected success, got failure: %s", result.ErrorMessage)
	}

	// Verify file was created
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Errorf("failed to read created file: %v", err)
	}
	if string(content) != expectedContent+"\n" && string(content) != expectedContent {
		t.Errorf("expected content '%s', got '%s'", expectedContent, string(content))
	}

	// Verify events
	assertHasToolCallEvent(t, collector.events, "Write")
	assertHasMessageEvent(t, collector.events, "assistant")
	assertHasMetricEvent(t, collector.events)
}

// TestIntegration_Codex_FileWrite verifies Codex runner can create files
// and produces proper events.
func TestIntegration_Codex_FileWrite(t *testing.T) {
	runner, err := NewCodexRunner()
	if err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	available, msg := runner.IsAvailable(context.Background())
	if !available {
		t.Skipf("Codex runner not available: %s", msg)
	}

	cfg := setupTest(t)
	testFile := filepath.Join(cfg.tempDir, "test-codex.txt")
	expectedContent := "Hello from Codex integration test"

	collector := &eventCollector{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := runner.Execute(ctx, ExecuteRequest{
		RunID:      uuid.New(),
		Prompt:     "Create a file called test-codex.txt with the content: " + expectedContent,
		WorkingDir: cfg.tempDir,
		EventSink:  collector,
	})
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	// Verify success
	if !result.Success {
		t.Errorf("expected success, got failure: %s", result.ErrorMessage)
	}

	// Verify file was created
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Errorf("failed to read created file: %v", err)
	}
	if string(content) != expectedContent+"\n" && string(content) != expectedContent {
		t.Errorf("expected content '%s', got '%s'", expectedContent, string(content))
	}

	// Verify events - Codex uses "file_change" for file operations
	assertHasToolCallEvent(t, collector.events, "file_change")
	assertHasMessageEvent(t, collector.events, "assistant")
	assertHasMetricEvent(t, collector.events)
}

// TestIntegration_OpenCode_FileWrite verifies OpenCode runner can create files
// and produces proper events.
func TestIntegration_OpenCode_FileWrite(t *testing.T) {
	runner, err := NewOpenCodeRunner()
	if err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	available, msg := runner.IsAvailable(context.Background())
	if !available {
		t.Skipf("OpenCode runner not available: %s", msg)
	}

	cfg := setupTest(t)
	testFile := filepath.Join(cfg.tempDir, "test-opencode.txt")
	expectedContent := "Hello from OpenCode integration test"

	collector := &eventCollector{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := runner.Execute(ctx, ExecuteRequest{
		RunID:      uuid.New(),
		Prompt:     "Create a file called test-opencode.txt with the content: " + expectedContent,
		WorkingDir: cfg.tempDir,
		EventSink:  collector,
	})
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	// Verify success
	if !result.Success {
		t.Errorf("expected success, got failure: %s", result.ErrorMessage)
	}

	// Verify file was created
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Errorf("failed to read created file: %v", err)
	}
	if string(content) != expectedContent+"\n" && string(content) != expectedContent {
		t.Errorf("expected content '%s', got '%s'", expectedContent, string(content))
	}

	// Verify events - OpenCode uses "write" for file operations
	assertHasToolCallEvent(t, collector.events, "write")
	assertHasMetricEvent(t, collector.events)
}

// Assertion helpers

func assertHasToolCallEvent(t *testing.T, events []*domain.RunEvent, expectedToolName string) {
	t.Helper()
	for _, evt := range events {
		if evt.EventType == domain.EventTypeToolCall {
			if toolData, ok := evt.Data.(*domain.ToolCallEventData); ok {
				if toolData.ToolName == expectedToolName {
					// Also verify input is not empty
					if toolData.Input == nil || len(toolData.Input) == 0 {
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
	// Message events are nice to have but not strictly required
	t.Logf("WARN: no message event found for role '%s' (some runners may not emit these)", expectedRole)
}

func assertHasMetricEvent(t *testing.T, events []*domain.RunEvent) {
	t.Helper()
	for _, evt := range events {
		if evt.EventType == domain.EventTypeMetric {
			if costData, ok := evt.Data.(*domain.CostEventData); ok {
				// Verify we have some token data
				if costData.InputTokens == 0 && costData.OutputTokens == 0 {
					t.Error("metric event has zero tokens")
				}
				return
			}
			// Also accept MetricEventData
			if _, ok := evt.Data.(*domain.MetricEventData); ok {
				return
			}
		}
	}
	t.Error("no metric event found")
}

// Compile check - ensure event package is imported
var _ = event.NewMemoryStore
