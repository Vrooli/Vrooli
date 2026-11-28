package cliinvoke

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// directSpawn matches exec.Command / exec.CommandContext whose first literal
// argument is "vrooli": the shape every migrated call site used to have.
var directSpawn = regexp.MustCompile(`exec\.Command(Context)?\((ctx,\s*|[a-zA-Z_]+,\s*)?"vrooli"`)

// [REQ:CLI-INVOKE-004] No Go file on the boot-recovery path spawns the vrooli
// CLI directly; every spawn goes through this package.
func TestNoDirectVrooliSpawnOutsideThisPackage(t *testing.T) {
	root := repoRoot(t)
	roots := []string{"internal", "cmd", "packages", filepath.Join("scenarios", "vrooli-autoheal")}
	var hits []string
	for _, base := range roots {
		_ = filepath.WalkDir(filepath.Join(root, base), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				name := d.Name()
				if name == "node_modules" || name == "vendor" || name == "testdata" || name == ".git" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			if strings.HasPrefix(rel, filepath.Join("packages", "cli-invoke-go")) {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			for i, line := range strings.Split(string(data), "\n") {
				if directSpawn.MatchString(line) {
					hits = append(hits, rel+":"+itoa(i+1)+": "+strings.TrimSpace(line))
				}
			}
			return nil
		})
	}
	if len(hits) != 0 {
		t.Fatalf("direct vrooli spawns outside cli-invoke-go:\n  %s", strings.Join(hits, "\n  "))
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".vrooli", "repo-contract.json")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repo root not found")
		}
		dir = parent
	}
}

func itoa(i int) string { return strconv.Itoa(i) }
