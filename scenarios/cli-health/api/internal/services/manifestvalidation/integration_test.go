package manifestvalidation

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

// findRepoRoot walks up from this test file's location until it sees
// .vrooli/repo-contract.json. Cheaper than os.Getwd() + ResolveRepoRoot
// since tests run from the package directory.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	root, err := repocontract.FindRepoRoot(dir)
	if err != nil {
		t.Fatalf("find repo root: %v", err)
	}
	return root
}

func requireBuf(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("buf"); err != nil {
		t.Skip("buf binary not on PATH; skipping integration test")
	}
}

// adoptingScenarios lists scenarios known to validate cleanly at HEAD.
// The test asserts each one returns `passed: true`. After the manifest-as-SSOT
// CLI consolidation, every proto-first scenario (including the once-drifted
// react-component-library and the three formerly code-registered holdouts —
// web-console, audio-tools, swarm-manager) has its proto surface fully
// bound-or-omitted in cli/manifest.json, so the list now spans a representative
// cross-section of the migrated fleet. (browser-automation-studio and
// git-control-tower remain excluded: they are not gen-endpoints proto-first
// scenarios and still carry their own un-reconciled CLI surface.)
var adoptingScenarios = []string{
	"cli-health",
	"development-toolchain-validator",
	"flow-verifier",
	"image-tools",
	"proto-health",
	"react-component-library",
	"web-console",
	"audio-tools",
	"swarm-manager",
}

// TestIntegration_CLIHealthReachesVerifiedL4 proves the full maturity loop for
// the reference adopter: with the production static-evidence provider wired,
// cli-health validating ITSELF has zero command_architecture debt — every
// declared primitive is verified against the committed cli/primitive-evidence.json
// (no undeclared/unverified/mismatch, and the artifact is neither stale nor
// malformed). No scenario command is executed; the evidence is read statically.
func TestIntegration_CLIHealthReachesVerifiedL4(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	requireBuf(t)

	root := findRepoRoot(t)
	svc := New(Deps{
		Manifests:            NewFilesystemManifestLoader(root),
		Schema:               NewJSONSchemaValidator(root),
		Protos:               NewBufProtoLoader(root),
		ArchitectureEvidence: NewFilesystemArchitectureEvidence(root),
	})

	r, err := svc.ValidateScenario(context.Background(), "cli-health")
	if err != nil {
		t.Fatalf("ValidateScenario(cli-health): %v", err)
	}

	archDebt := map[string]bool{
		CodeArchPrimitiveUndecl:   true,
		CodeArchPrimitiveUnverif:  true,
		CodeArchPrimitiveMismatch: true,
		CodeArchEvidenceStale:     true,
		CodeArchEvidenceMalformed: true,
		CodeArchClaimedViolation:  true,
	}
	for _, f := range r.Findings {
		if archDebt[f.Code] {
			t.Errorf("cli-health should be a verified-L4 reference adopter but has %s: %s @ %s", f.Code, f.Message, f.Location)
		}
	}
}

func TestIntegration_AdoptingScenariosAllPass(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	requireBuf(t)

	root := findRepoRoot(t)
	svc := New(Deps{
		Manifests: NewFilesystemManifestLoader(root),
		Schema:    NewJSONSchemaValidator(root),
		Protos:    NewBufProtoLoader(root),
	})

	for _, scenario := range adoptingScenarios {
		scenario := scenario
		t.Run(scenario, func(t *testing.T) {
			// Skip scenarios that no longer ship a manifest (drift over time).
			manifestPath, _ := repocontract.ScenarioCLIManifestPath(root, scenario)
			if _, err := os.Stat(manifestPath); err != nil {
				t.Skipf("manifest missing at %s; remove from adoptingScenarios", manifestPath)
			}
			r, err := svc.ValidateScenario(context.Background(), scenario)
			if err != nil {
				t.Fatalf("ValidateScenario(%q): %v", scenario, err)
			}
			if !r.Passed {
				var msgs []string
				for _, f := range r.Findings {
					if f.Severity == SeverityError {
						msgs = append(msgs, f.Code+": "+f.Message+" @ "+f.Location)
					}
				}
				t.Fatalf("%s should pass; errors:\n%s", scenario, strings.Join(msgs, "\n"))
			}
		})
	}
}
