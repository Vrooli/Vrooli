package invokers

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

// Every scenario Makefile includes mk/scenario.mk, carries no copied body of
// a shared target, and declares every custom body. The census tool under
// tools/makefile-include-census is the oracle; this test runs its check mode
// so a regenerated or hand-edited Makefile cannot drift back to a copy.
func TestScenarioMakefilesIncludeToolchainMk(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "mk", "scenario.mk")); err != nil {
		t.Fatalf("mk/scenario.mk: %v", err)
	}
	cmd := exec.Command("go", "run", "./tools/makefile-include-census", "--root", root)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("scenario Makefiles drifted from mk/scenario.mk:\n%s", strings.TrimSpace(string(output)))
	}
	for _, template := range []string{"templates/scenarios/react-vite/Makefile", "templates/scenarios/landing-page-react-vite/Makefile"} {
		data, err := os.ReadFile(filepath.Join(root, template))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "-include ../../mk/scenario.mk") {
			t.Errorf("%s does not include mk/scenario.mk", template)
		}
	}
	resources, _ := filepath.Glob(filepath.Join(root, "templates", "resources", "*", "Makefile"))
	for _, path := range resources {
		data, _ := os.ReadFile(path)
		if !strings.Contains(string(data), "-include ../../mk/toolchain.mk") {
			t.Errorf("%s does not include mk/toolchain.mk", path)
		}
	}
}
