package scenariocli

import "testing"

func TestParseTestArgsPassesThroughTestGenieFlags(t *testing.T) {
	got, err := ParseTestArgs(false, false, []string{
		"brand-manager",
		"--preset", "smoke",
		"--wait",
		"--diagnostics-preset=light",
		"unit",
	})
	if err != nil {
		t.Fatalf("ParseTestArgs returned error: %v", err)
	}
	if got.Name != "brand-manager" {
		t.Fatalf("name = %q, want brand-manager", got.Name)
	}
	want := []string{"--preset", "smoke", "--wait", "--diagnostics-preset=light", "unit"}
	if len(got.Args) != len(want) {
		t.Fatalf("args = %v, want %v", got.Args, want)
	}
	for i := range want {
		if got.Args[i] != want[i] {
			t.Fatalf("args = %v, want %v", got.Args, want)
		}
	}
}

func TestParseTestArgsPassesThroughJSON(t *testing.T) {
	got, err := ParseTestArgs(false, false, []string{"brand-manager", "--json", "--wait"})
	if err != nil {
		t.Fatalf("ParseTestArgs returned error: %v", err)
	}
	want := []string{"--json", "--wait"}
	if len(got.Args) != len(want) {
		t.Fatalf("args = %v, want %v", got.Args, want)
	}
	for i := range want {
		if got.Args[i] != want[i] {
			t.Fatalf("args = %v, want %v", got.Args, want)
		}
	}
}

func TestParseTestArgsDoesNotRewriteSelectors(t *testing.T) {
	got, err := ParseTestArgs(false, false, []string{"brand-manager", "e2e"})
	if err != nil {
		t.Fatalf("ParseTestArgs returned error: %v", err)
	}
	if len(got.Args) != 1 || got.Args[0] != "e2e" {
		t.Fatalf("args = %v, want [e2e]", got.Args)
	}
}

func TestParseTestArgsRequiresScenarioFirst(t *testing.T) {
	if _, err := ParseTestArgs(false, false, []string{"--preset", "smoke", "brand-manager"}); err == nil {
		t.Fatal("expected scenario-first usage error")
	}
}

func TestParseTestRequestOnlyInterceptsHelpBeforeScenario(t *testing.T) {
	if _, err := ParseTestRequest(false, false, []string{"--help"}); err == nil {
		t.Fatal("expected wrapper help before scenario")
	}
	got, err := ParseTestRequest(false, false, []string{"brand-manager", "--help"})
	if err != nil {
		t.Fatalf("ParseTestRequest returned error: %v", err)
	}
	if len(got.Args) != 1 || got.Args[0] != "--help" {
		t.Fatalf("args = %v, want [--help]", got.Args)
	}
}
