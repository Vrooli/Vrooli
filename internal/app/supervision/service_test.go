package supervision

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/coreset"
)

func TestComputeAlwaysRetainsOperationalRecoveryPlane(t *testing.T) {
	report := Compute("", coreset.Authority{Seed: []string{"writer"}, TrustedBase: []string{"writer"}})

	for _, name := range coreset.RequiredOperationalScenarios {
		if !contains(report.Seed, name) || !contains(report.CoreSet, name) || !contains(report.TrustedBase, name) {
			t.Errorf("required operational scenario %q missing from report: %+v", name, report)
		}
	}
}

func TestComputeSharedDependencyUsesStrongestSupervisionIntent(t *testing.T) {
	root := t.TempDir()
	writeManifest := func(name, body string) {
		t.Helper()
		dir := filepath.Join(root, name, ".vrooli")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "service.json"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// The try_start parent sorts first deliberately. The later required edge
	// must still upgrade the shared member to the critical supervision mode.
	writeManifest("a-optional", `{"service":{"name":"a-optional"},"dependencies":{"scenarios":{"shared":{"enabled":true,"required":false,"startup_policy":"try_start"}}}}`)
	writeManifest("z-required", `{"service":{"name":"z-required"},"dependencies":{"scenarios":{"shared":{"enabled":true,"required":true,"startup_policy":"must_start"}}}}`)
	writeManifest("shared", `{"service":{"name":"shared"}}`)

	report := Compute(root, coreset.Authority{Seed: []string{"a-optional", "z-required"}, TrustedBase: []string{"a-optional", "z-required"}})
	for _, member := range report.Members {
		if member.Name == "shared" && member.Kind == coreset.MemberKindScenario {
			if member.SupervisionIntent != coreset.IntentMustStart {
				t.Fatalf("shared supervision intent = %q, want %q; report=%+v", member.SupervisionIntent, coreset.IntentMustStart, report)
			}
			return
		}
	}
	t.Fatalf("shared dependency missing from supervision report: %+v", report)
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
