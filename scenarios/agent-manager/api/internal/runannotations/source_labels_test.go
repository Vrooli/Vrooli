package runannotations

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Which labels exist, how they are asked and what they mean is the pack's data.
// Production code that names one would stop working, or worse keep working on a
// stale meaning, the day the pack renames it. This reads every label id of the
// real pack and fails when API or CLI source (tests aside) mentions one, so a
// rename in the pack never leaves a dead reference behind.
func TestProductionSourceNamesNoLabelOfThePack(t *testing.T) {
	pack := loadDefaultPack(t)
	var ids []*regexp.Regexp
	for _, id := range pack.Order() {
		ids = append(ids, regexp.MustCompile(`(^|[^A-Za-z0-9_.])`+regexp.QuoteMeta(id)+`($|[^A-Za-z0-9_])`))
	}
	scenarioRoot := filepath.Join("..", "..", "..")
	for _, tree := range []string{"api", "cli"} {
		err := filepath.WalkDir(filepath.Join(scenarioRoot, tree), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if entry.Name() == "node_modules" || entry.Name() == "vendor" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			body, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			for index, pattern := range ids {
				if pattern.Match(body) {
					t.Errorf("%s names the pack label %q; take it from the pack instead", path, pack.Order()[index])
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
