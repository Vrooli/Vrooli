package generate

import (
	"strings"
	"testing"
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
