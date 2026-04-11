package main

import (
	"bytes"
	"strings"
	"testing"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=2 | LAST: 2026-04-11

func TestPrintErrorWithContextFormatsUnknownCommandSuggestions(t *testing.T) {
	var stderr bytes.Buffer
	printErrorWithContext(&stderr, newUnknownCommandError("statsu"))

	output := stderr.String()
	if !strings.Contains(output, "Unknown command: statsu") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "status") {
		t.Fatalf("expected suggestion in output, got %q", output)
	}
}

func TestShowMainHelpUsesPlainLabels(t *testing.T) {
	var stdout bytes.Buffer
	showMainHelp(&stdout)

	output := stdout.String()
	if strings.Contains(output, "🚀") || strings.Contains(output, "📋") {
		t.Fatalf("help output should avoid emoji markers, got %q", output)
	}
	if !strings.Contains(output, "Vrooli CLI - AI Platform Management Tool") {
		t.Fatalf("output = %q", output)
	}
}

func TestPrintErrorWithContextFormatsUnknownScenarioCommandSuggestions(t *testing.T) {
	var stderr bytes.Buffer
	printErrorWithContext(&stderr, newUnknownScenarioCommandError("statsu"))

	output := stderr.String()
	if !strings.Contains(output, "Unknown scenario command: statsu") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "status") {
		t.Fatalf("expected suggestion in output, got %q", output)
	}
}

func TestShowScenarioHelpIncludesWeek4CommandsFromRegistry(t *testing.T) {
	var stdout bytes.Buffer
	showScenarioHelp(&stdout)

	output := stdout.String()
	for _, command := range []string{"start-all", "stop-all", "generate", "heal-from-sandbox"} {
		if !strings.Contains(output, command) {
			t.Fatalf("scenario help missing %q: %q", command, output)
		}
	}
}
