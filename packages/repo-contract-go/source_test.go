package repocontract

import (
	"io/fs"
	"path/filepath"
	"slices"
	"testing"

	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
)

// WalkSource must visit source files in every source root, including packages
// that happen to be named docs, and never descend into documentation trees
// whose retained evidence snapshots code.
func TestWalkSourceVisitsSourceAndSkipsDocumentationTrees(t *testing.T) {
	root := fixtureRoot(t)
	for rel, body := range map[string]string{
		"scenarios/test-genie/api/handlers/docs/module.go":               "package docs\n",
		"scenarios/test-genie/docs/internal/evidence/preimage/broken.go": "",
		"scenarios/test-genie/ui/node_modules/dep/index.js":              "",
		"docs/internal/research/sources/bundle/index.js":                 "",
		"resources/postgres/docs/snippet.go":                             "",
		"internal/app/service.go":                                        "package app\n",
	} {
		testkitgo.WriteFile(t, filepath.Join(root, filepath.FromSlash(rel)), body)
	}

	var visited []string
	err := mustLoadDefault(t, root).WalkSource(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() {
			rel, _ := filepath.Rel(root, path)
			visited = append(visited, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkSource() error = %v", err)
	}
	for _, want := range []string{"scenarios/test-genie/api/handlers/docs/module.go", "internal/app/service.go", "packages/repo-contract-go/load.go", "cmd/vrooli/main.go"} {
		if !slices.Contains(visited, want) {
			t.Errorf("WalkSource did not visit source file %s", want)
		}
	}
	for _, unwanted := range []string{"scenarios/test-genie/docs/internal/evidence/preimage/broken.go", "scenarios/test-genie/ui/node_modules/dep/index.js", "docs/internal/research/sources/bundle/index.js", "resources/postgres/docs/snippet.go"} {
		if slices.Contains(visited, unwanted) {
			t.Errorf("WalkSource visited non-source file %s", unwanted)
		}
	}
}
