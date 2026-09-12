package invokers

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

const supervisoryModulesFile = "tools/supervisory-modules.txt"

var rootReplace = regexp.MustCompile(`(?m)^replace github\.com/vrooli/vrooli => `)

// [REQ:CLI-ARGV-004] Every scenario module that replaces the control-plane
// root module AND spawns the vrooli CLI through the invoker seam is on the
// supervisory module list, so `make install` cannot install a CLI those
// modules can no longer build against. Modules that still spawn the CLI
// ad hoc are the deferred census on docs/reference/cli-invokers.md.
func TestEverySpawningRootReplacerIsASupervisoryModule(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	listed := map[string]bool{}
	file, err := os.Open(filepath.Join(root, supervisoryModulesFile))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		listed[line] = true
		if _, err := os.Stat(filepath.Join(root, line, "go.mod")); err != nil {
			t.Errorf("%s lists %s, which has no go.mod", supervisoryModulesFile, line)
		}
	}
	candidates, _ := filepath.Glob(filepath.Join(root, "scenarios", "*", "api", "go.mod"))
	more, _ := filepath.Glob(filepath.Join(root, "scenarios", "*", "cli", "go.mod"))
	candidates = append(candidates, more...)
	var missing []string
	for _, gomod := range candidates {
		data, err := os.ReadFile(gomod)
		if err != nil || !rootReplace.Match(data) {
			continue
		}
		dir := filepath.Dir(gomod)
		if !moduleSpawnsVrooli(dir) {
			continue
		}
		rel, _ := filepath.Rel(root, dir)
		rel = filepath.ToSlash(rel)
		if !listed[rel] {
			missing = append(missing, rel)
		}
	}
	sort.Strings(missing)
	if len(missing) != 0 {
		t.Fatalf("modules replace the root module and spawn vrooli but are not in %s:\n  %s", supervisoryModulesFile, strings.Join(missing, "\n  "))
	}
}

var spawnMarkers = regexp.MustCompile(`cliinvoke\.Invocation\{`)

func moduleSpawnsVrooli(dir string) bool {
	found := false
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if d.IsDir() {
			if name := d.Name(); name == "node_modules" || name == "vendor" || name == "testdata" {
				return filepath.SkipDir
			}
			if path != dir {
				if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr == nil && spawnMarkers.Match(data) {
			found = true
		}
		return nil
	})
	return found
}
