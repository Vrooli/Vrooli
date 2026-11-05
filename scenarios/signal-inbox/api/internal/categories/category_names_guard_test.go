package categories_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// [REQ:SIG-P0-004] Product code may not privilege workshop examples as a
// category taxonomy. The operator's runtime category data is the only set.
func TestProductCodeDoesNotHardCodeExampleCategoryNames(t *testing.T) {
	t.Log("[REQ:SIG-P0-004]")
	for _, root := range []string{".", "../../handlers/categories", "../../../cli/domains/categories"} {
		files, err := filepath.Glob(filepath.Join(root, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range files {
			if strings.HasSuffix(file, "_test.go") {
				continue
			}
			body, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			for _, forbidden := range []string{"app-ideas", "marketing", "meals", "fitness", "research"} {
				if strings.Contains(strings.ToLower(string(body)), forbidden) {
					t.Fatalf("hard-coded category %q in %s", forbidden, file)
				}
			}
		}
	}
}
