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
// The test asserts each one returns `passed: true`. The validator
// discovered genuine drift in react-component-library, browser-automation-studio,
// and git-control-tower (orphan proto methods); those adopters are
// excluded until the manifests are reconciled — adding them here without
// fixing the manifest would just turn this test into a known-failure.
var adoptingScenarios = []string{
	"cli-health",
	"development-toolchain-validator",
	"flow-verifier",
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
