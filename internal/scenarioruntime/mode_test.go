package scenarioruntime

import "testing"

func TestModeFromStringNormalizesAndClassifiesMigrationModes(t *testing.T) {
	cases := []struct {
		raw          string
		want         string
		writeEnabled bool
		readEnabled  bool
		strictReads  bool
	}{
		{raw: "", want: ModeOff},
		{raw: " off ", want: ModeOff},
		{raw: "DUAL", want: ModeDual, writeEnabled: true},
		{raw: "prefer", want: ModePrefer, writeEnabled: true, readEnabled: true},
		{raw: "strict", want: ModeStrict, writeEnabled: true, readEnabled: true, strictReads: true},
	}

	for _, tc := range cases {
		got, err := ModeFromString(tc.raw)
		if err != nil {
			t.Fatalf("ModeFromString(%q) error = %v", tc.raw, err)
		}
		if got != tc.want {
			t.Fatalf("ModeFromString(%q) = %q, want %q", tc.raw, got, tc.want)
		}
		if WriteEnabled(got) != tc.writeEnabled {
			t.Fatalf("WriteEnabled(%q) = %v, want %v", got, WriteEnabled(got), tc.writeEnabled)
		}
		if ReadEnabled(got) != tc.readEnabled {
			t.Fatalf("ReadEnabled(%q) = %v, want %v", got, ReadEnabled(got), tc.readEnabled)
		}
		if StrictReads(got) != tc.strictReads {
			t.Fatalf("StrictReads(%q) = %v, want %v", got, StrictReads(got), tc.strictReads)
		}
	}
}

func TestModeFromStringRejectsUnknownMode(t *testing.T) {
	if _, err := ModeFromString("enabled"); err == nil {
		t.Fatal("ModeFromString(enabled) error = nil, want error")
	}
}

func TestScenarioAllowedTreatsEmptyAllowlistAsGlobal(t *testing.T) {
	if !ScenarioAllowed("", "workspace-sandbox") {
		t.Fatal("empty allowlist should allow all scenarios")
	}
	if !ScenarioAllowed(" , \n\t ", "workspace-sandbox") {
		t.Fatal("blank allowlist should allow all scenarios")
	}
}

func TestScenarioAllowedMatchesNormalizedScenarioNames(t *testing.T) {
	if !ScenarioAllowed("workspace-sandbox, browser-automation-studio", " Workspace-Sandbox ") {
		t.Fatal("allowlist should match case-insensitive trimmed scenario names")
	}
	if ScenarioAllowed("workspace-sandbox,browser-automation-studio", "web-console") {
		t.Fatal("allowlist should reject scenarios that are not listed")
	}
	if ScenarioAllowed("workspace-sandbox", "") {
		t.Fatal("non-empty allowlist should reject empty scenario names")
	}
}

func TestScenarioScopedModeClassificationHonorsAllowlist(t *testing.T) {
	t.Setenv(AllowlistEnv, "workspace-sandbox")

	if !WriteEnabledForScenario(ModePrefer, "workspace-sandbox") {
		t.Fatal("prefer mode should write for allowlisted scenario")
	}
	if !ReadEnabledForScenario(ModePrefer, "workspace-sandbox") {
		t.Fatal("prefer mode should read for allowlisted scenario")
	}
	if WriteEnabledForScenario(ModePrefer, "web-console") {
		t.Fatal("prefer mode should not write for non-allowlisted scenario")
	}
	if ReadEnabledForScenario(ModePrefer, "web-console") {
		t.Fatal("prefer mode should not read for non-allowlisted scenario")
	}
	if StrictReadsForScenario(ModeStrict, "web-console") {
		t.Fatal("strict mode should not apply strict reads to non-allowlisted scenario")
	}
	if !StrictReadsForScenario(ModeStrict, "workspace-sandbox") {
		t.Fatal("strict mode should apply strict reads to allowlisted scenario")
	}
}

func TestLocalPortURLOnlyAdvertisesKnownHTTPPortKinds(t *testing.T) {
	if got := LocalPortURL("api", 18080); got != "http://127.0.0.1:18080" {
		t.Fatalf("LocalPortURL(api, 18080) = %q", got)
	}
	if got := LocalPortURL("ui", 28080); got != "" {
		t.Fatalf("LocalPortURL(ui, 28080) = %q, want empty until UI claim URLs are explicitly supported", got)
	}
	if got := LocalPortURL("api", 0); got != "" {
		t.Fatalf("LocalPortURL(api, 0) = %q, want empty", got)
	}
}

func TestActiveInstanceStatusesAreCentralizedAndImmutable(t *testing.T) {
	statuses := ActiveInstanceStatuses()
	if len(statuses) != 2 || statuses[0] != StatusStarting || statuses[1] != StatusRunning {
		t.Fatalf("ActiveInstanceStatuses() = %#v, want starting/running", statuses)
	}
	statuses[0] = StatusFailed
	if IsActiveInstanceStatus(StatusStarting) != true {
		t.Fatal("mutating returned statuses should not change active-status policy")
	}
	for _, tc := range []struct {
		status string
		want   bool
	}{
		{status: StatusStarting, want: true},
		{status: StatusRunning, want: true},
		{status: StatusStopping},
		{status: StatusFailed},
		{status: StatusExpired},
		{status: StatusStopped},
	} {
		if got := IsActiveInstanceStatus(tc.status); got != tc.want {
			t.Fatalf("IsActiveInstanceStatus(%q) = %v, want %v", tc.status, got, tc.want)
		}
	}
}
