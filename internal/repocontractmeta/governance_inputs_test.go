package repocontractmeta_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// The repository's debt, portability, and skip-budget ratchets are project
// config files, not Go source. The seam scanner treats a missing seam file as
// an empty rule set, so a clean checkout without these inputs would report
// success while enforcing nothing. Keep the inputs themselves in the test
// surface so their absence is a red build, not a silent pass.
func TestProjectConfigGovernanceInputsArePresent(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	for _, name := range []string{
		"canonical-seams.json",
		"portability-lint-allowlist.json",
		"skip-budgets.json",
	} {
		path := filepath.Join(root, ".vrooli", name)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("required governance input .vrooli/%s is missing: %v", name, err)
		}
	}
}

func TestCanonicalSeamsDeclareAtLeastOneCompleteRule(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	data, err := os.ReadFile(filepath.Join(root, ".vrooli", "canonical-seams.json"))
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		SchemaVersion string `json:"schemaVersion"`
		Seams         []struct {
			ID          string `json:"id"`
			Canonical   string `json:"canonical"`
			Why         string `json:"why"`
			Remediation string `json:"remediation"`
			Scope       struct {
				Include []string `json:"include"`
			} `json:"scope"`
		} `json:"seams"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatalf("decode .vrooli/canonical-seams.json: %v", err)
	}
	if file.SchemaVersion != "1.0.0" {
		t.Fatalf("canonical seam schemaVersion = %q, want 1.0.0", file.SchemaVersion)
	}
	if len(file.Seams) == 0 {
		t.Fatal("canonical seam ledger has no rules; the seam gate would pass vacuously")
	}
	seen := make(map[string]struct{}, len(file.Seams))
	for i, seam := range file.Seams {
		if seam.ID == "" || seam.Canonical == "" || seam.Why == "" || seam.Remediation == "" || len(seam.Scope.Include) == 0 {
			t.Fatalf("canonical seam %d is incomplete: %+v", i, seam)
		}
		if _, duplicate := seen[seam.ID]; duplicate {
			t.Fatalf("canonical seam id %q is duplicated", seam.ID)
		}
		seen[seam.ID] = struct{}{}
	}
}
