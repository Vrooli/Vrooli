package runlocal

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseArgsParsesRepeatableSelectors(t *testing.T) {
	parsed, err := ParseArgs([]string{
		"demo",
		"--type", "phased",
		"--path", "api/foo.go",
		"--path", "ui/src/App.tsx",
		"--playbook", "bas/one.json",
		"--playbook", "bas/two.json",
		"--filter", "UserFlow",
		"--json",
	})
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}

	if parsed.Scenario != "demo" || parsed.Type != "phased" || parsed.Filter != "UserFlow" {
		t.Fatalf("unexpected parsed args: %+v", parsed)
	}
	if !reflect.DeepEqual(parsed.Paths, []string{"api/foo.go", "ui/src/App.tsx"}) {
		t.Fatalf("expected parsed paths, got %v", parsed.Paths)
	}
	if !reflect.DeepEqual(parsed.Playbooks, []string{"bas/one.json", "bas/two.json"}) {
		t.Fatalf("expected parsed playbooks, got %v", parsed.Playbooks)
	}
	if !parsed.JSON {
		t.Fatal("expected json output to be enabled")
	}
}

func TestParseArgsRejectsMissingScenario(t *testing.T) {
	_, err := ParseArgs(nil)
	if err == nil {
		t.Fatal("expected missing scenario to fail")
	}
	if !strings.Contains(err.Error(), "usage: run-tests") {
		t.Fatalf("expected usage error, got %v", err)
	}
}

func TestMultiFlagSetIgnoresBlankValues(t *testing.T) {
	var values multiFlag
	if err := values.Set(" first "); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if err := values.Set("   "); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if err := values.Set("second"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if !reflect.DeepEqual([]string(values), []string{"first", "second"}) {
		t.Fatalf("expected trimmed values, got %v", []string(values))
	}
}

func TestDefaultValueFallsBackForBlankStrings(t *testing.T) {
	if got := defaultValue("  ", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback value, got %q", got)
	}
	if got := defaultValue("present", "fallback"); got != "present" {
		t.Fatalf("expected explicit value, got %q", got)
	}
}
