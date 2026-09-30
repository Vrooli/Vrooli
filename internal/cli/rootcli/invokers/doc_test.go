package invokers

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

var update = flag.Bool("update", false, "rewrite docs/reference/cli-invokers.md from the registry")

// TestDocIsCurrent fails when the committed reference page no longer matches
// the registry. Regenerate with -update.
func TestDocIsCurrent(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	want, err := RenderDoc()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, DocPath)
	if *update {
		if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s: %v (run with -update)", DocPath, err)
	}
	if string(got) != want {
		t.Fatalf("%s is out of date; run: go test ./internal/cli/rootcli/invokers -run TestDocIsCurrent -update", DocPath)
	}
}
