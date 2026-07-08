package generate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestParseArgsParsesSupportedFlags(t *testing.T) {
	parsed, err := ParseArgs([]string{
		"demo",
		"--types", "unit,integration",
		"--coverage", "90",
		"--priority", "High",
		"--notes", "focus on regressions",
		"--json",
	})
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}

	if parsed.Scenario != "demo" {
		t.Fatalf("expected scenario demo, got %q", parsed.Scenario)
	}
	if parsed.Types != "unit,integration" {
		t.Fatalf("expected types to round-trip, got %q", parsed.Types)
	}
	if parsed.Coverage != 90 {
		t.Fatalf("expected coverage 90, got %d", parsed.Coverage)
	}
	if parsed.Priority != "High" {
		t.Fatalf("expected priority High, got %q", parsed.Priority)
	}
	if parsed.Notes != "focus on regressions" {
		t.Fatalf("expected notes to round-trip, got %q", parsed.Notes)
	}
	if !parsed.JSON {
		t.Fatal("expected json output to be enabled")
	}
}

func TestParseArgsRejectsInvalidCoverage(t *testing.T) {
	_, err := ParseArgs([]string{"demo", "--coverage", "101"})
	if err == nil {
		t.Fatal("expected invalid coverage to fail")
	}
	if !strings.Contains(err.Error(), "coverage must be between 0 and 100") {
		t.Fatalf("expected coverage validation error, got %v", err)
	}
}

func TestParseArgsRejectsInvalidPriority(t *testing.T) {
	_, err := ParseArgs([]string{"demo", "--priority", "rush"})
	if err == nil {
		t.Fatal("expected invalid priority to fail")
	}
	if !strings.Contains(err.Error(), "priority must be one of") {
		t.Fatalf("expected priority validation error, got %v", err)
	}
}

func TestIsAllowedPriorityIsCaseInsensitive(t *testing.T) {
	if !isAllowedPriority("URGENT") {
		t.Fatal("expected urgent priority to be accepted case-insensitively")
	}
	if isAllowedPriority("later") {
		t.Fatal("expected unsupported priority to be rejected")
	}
}

func TestRequestFromContextUsesCliCoreParsedInputs(t *testing.T) {
	dir := t.TempDir()
	notesPath := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(notesPath, []byte("focus on regressions"), 0o644); err != nil {
		t.Fatalf("write notes: %v", err)
	}
	ctx, err := cliapp.NewTestRunContextFromArgs(ArgsSchema, []string{
		"demo",
		"--types", "unit,integration",
		"--coverage", "90",
		"--priority", "High",
		"--notes", "ignored",
		"--notes-file", notesPath,
		"--json",
	}, nil, nil, nil)
	if err != nil {
		t.Fatalf("parse args: %v", err)
	}

	got, err := RequestFromContext(ctx)
	if err != nil {
		t.Fatalf("RequestFromContext: %v", err)
	}
	if got.ScenarioName != "demo" {
		t.Fatalf("scenario = %q, want demo", got.ScenarioName)
	}
	if strings.Join(got.RequestedTypes, ",") != "unit,integration" {
		t.Fatalf("types = %#v", got.RequestedTypes)
	}
	if got.CoverageTarget == nil || *got.CoverageTarget != 90 {
		t.Fatalf("coverage = %v, want 90", got.CoverageTarget)
	}
	if got.Priority != "high" {
		t.Fatalf("priority = %q, want high", got.Priority)
	}
	if got.Notes != "focus on regressions" {
		t.Fatalf("notes = %q", got.Notes)
	}
}

func TestRequestFromContextRejectsInvalidCoverage(t *testing.T) {
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema:      ArgsSchema,
		Positionals: map[string]string{"scenario": "demo"},
		Flags:       map[string]string{"coverage": "90x"},
	})
	if _, err := RequestFromContext(ctx); err == nil || !strings.Contains(err.Error(), "coverage must be between 0 and 100") {
		t.Fatalf("expected coverage validation error, got %v", err)
	}
}
