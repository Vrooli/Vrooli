package scenariocli

import "testing"

func TestRuntimeReleaseParsing(t *testing.T) {
	for _, tc := range []struct {
		id                  CommandID
		args                []string
		name, path, release string
		json                bool
	}{
		{CommandCaptureRuntime, []string{"demo@shadow", "--path", "/fixture", "--json"}, "demo@shadow", "/fixture", "", true},
		{CommandVerifyRuntime, []string{"demo", "abc123"}, "demo", "", "abc123", false},
	} {
		got, err := ParseRuntimeReleaseRequest(tc.id, false, tc.args)
		if err != nil {
			t.Fatal(err)
		}
		if got.Name != tc.name || got.CustomPath != tc.path || got.ReleaseID != tc.release || got.JSON != tc.json {
			t.Fatalf("got %#v", got)
		}
	}
}

func TestRuntimeReleaseParsingRejectsInvalidRequests(t *testing.T) {
	for _, tc := range []struct {
		id   CommandID
		args []string
	}{
		{CommandCaptureRuntime, nil}, {CommandVerifyRuntime, []string{"demo"}},
		{CommandVerifyRuntime, []string{"demo", "source"}}, {CommandCaptureRuntime, []string{"node/demo"}},
		{CommandCaptureRuntime, []string{"demo", "--force"}}, {CommandVerifyRuntime, []string{"demo", "id", "--path", "/tmp"}},
	} {
		if _, err := ParseRuntimeReleaseRequest(tc.id, false, tc.args); err == nil {
			t.Fatalf("accepted %s %v", tc.id, tc.args)
		}
	}
}

func TestRestartRuntimeReleaseRequiresSelectionPair(t *testing.T) {
	for _, args := range [][]string{
		{"demo", "--runtime-release", "abc"}, {"demo", "--expected-runtime-release", "source"},
		{"demo", "--runtime-release", "abc", "--force"},
		{"demo", "--runtime-release", "", "--expected-runtime-release", "source"},
	} {
		if _, err := ParseRestartRequest(false, args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	if _, err := ParseStartRequest(false, []string{"demo", "--runtime-release", "abc", "--expected-runtime-release", "source"}); err == nil {
		t.Fatal("start accepted selection change")
	}
}

func TestRestartRuntimeReleasePreservesExactSelectionAndForce(t *testing.T) {
	for _, id := range []string{"exact-release-id", "source"} {
		got, err := ParseRestartRequest(false, []string{"demo", "--runtime-release", id, "--expected-runtime-release", "old-id", "--force"})
		if err != nil {
			t.Fatal(err)
		}
		if got.Options.RetainedReleaseID != id || got.Options.ExpectedRetainedReleaseID != "old-id" || !got.Options.ForceSetup {
			t.Fatalf("options=%#v", got.Options)
		}
	}
}
