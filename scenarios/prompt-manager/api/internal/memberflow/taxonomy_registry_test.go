package memberflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAllTaxonomiesResolvesDeclaredPackageFiles(t *testing.T) {
	for _, tc := range []struct {
		name          string
		requiredFiles string
		writeTaxonomy bool
		wantError     string
	}{
		{"explicit missing file", `,"requiredFiles":["taxonomy.json"]`, false, "no such file"},
		{"inherited files", "", true, ""},
		{"explicit override", `,"requiredFiles":["README.md"]`, true, "has no required taxonomy.json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeRepoFile(t, root, "docs/agent-system/team-plan-of-record.manifest.json", `{
  "contract":{"kind":"team-plan-of-record","schema":"team-plan-of-record/v1"},
  "sections":[{"id":"taxonomies","path":"taxonomies/","packageType":"taxonomy","requiredFiles":["taxonomy.json"]}]
}`)
			writeRepoFile(t, root, "docs/team-a/manifest.json", fmt.Sprintf(`{
  "contract":{"kind":"team-plan-of-record","schema":"team-plan-of-record/v1","base":"docs/agent-system/team-plan-of-record.manifest.json","team":"team-a"},
  "sections":[{"id":"taxonomies","path":"taxonomies/","packages":[{"id":"owned","path":"owned/","packageType":"taxonomy"%s}]}]
}`, tc.requiredFiles))
			if err := os.MkdirAll(filepath.Join(root, "docs/team-a/taxonomies/owned"), 0o755); err != nil {
				t.Fatal(err)
			}

			if tc.writeTaxonomy {
				writeRepoFile(t, root, "docs/team-a/taxonomies/owned/taxonomy.json", `{"id":"owned"}`)
			}
			registry, err := LoadAllTaxonomies(root)
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("error = %v, want %q", err, tc.wantError)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(registry) != 1 || registry["owned"] == nil {
				t.Fatalf("registry = %v, want inherited owned taxonomy", registry)
			}
		})
	}
}
