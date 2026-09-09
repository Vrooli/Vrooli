package programsvalidation

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"

	internalbindings "program-runtime/internal/bindings"
)

// The shipped corpus is the fixture. Every scenario that declares programs
// must pass the static checks against the repo's own manifests, and no
// program may carry a local copy of the transport table the kernel binds as
// `program.classify`. The per-program details are printed so a failure names
// the file and line rather than a finding id.
func TestShippedCorpusIsPreflightCleanAndCarriesNoDuplicatedHelper(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Skip("repo root unavailable")
	}
	registry, err := internalbindings.Load(root)
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(root, "scenarios"))
	if err != nil {
		t.Fatal(err)
	}
	var scenarios []string
	for _, entry := range entries {
		if entry.IsDir() {
			if _, statErr := os.Stat(filepath.Join(root, "scenarios", entry.Name(), ".vrooli", "program-runtime")); statErr == nil {
				scenarios = append(scenarios, entry.Name())
			}
		}
	}
	sort.Strings(scenarios)
	if len(scenarios) == 0 {
		t.Skip("no scenario declares programs")
	}
	var failures []string
	for _, scenario := range scenarios {
		var details []any
		findings := validateScenario(root, scenario, registry, false, &details)
		for _, finding := range findings {
			switch finding {
			case "programs.duplicated_helper", "programs.preflight_diagnostic", "programs.envelope_missing", "programs.contract_invalid":
				failures = append(failures, scenario+": "+finding)
			}
		}
		for _, detail := range details {
			if m, ok := detail.(map[string]any); ok {
				failures = append(failures, fmt.Sprintf("%s: %v line %v %v: %v", scenario, m["program"], m["line"], m["name"], m["message"]))
			}
		}
	}
	if len(failures) > 0 {
		t.Fatalf("%d scenarios with programs; corpus defects:\n  %s", len(scenarios), strings.Join(failures, "\n  "))
	}
	t.Logf("%d scenarios with declared programs are preflight-clean with no duplicated helper", len(scenarios))
}
