package validation

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/repo-contract-go/repocontracttest"
)

// deniedScenarios are excluded from the fleet guard with a stated reason.
// A scenario belongs here only while another owner is mid-change on it; the
// list is meant to shrink.
var deniedScenarios = map[string]string{
	"browser-automation-studio": "owned by a separate in-flight change; migrate with that work",
}

// TestFleetHasNoGenericDatabasePaths runs this analyzer over every scenario in
// the repository and fails if any generic database-path read or hand-rolled
// SQLite DSN has come back.
//
// A unit test over the analyzer proves the analyzer works. This proves the
// FLEET is clean, which is the property that actually matters: the defect it
// guards was created one reasonable-looking copy at a time, and the next copy
// will look reasonable too. Catching it here costs one test; catching it in
// production cost a 9.35 GB shared database and 146 misfiled run records.
//
// The test locates the repository by walking up from the working directory and
// skips when it cannot find one, so it stays correct in a packaged checkout.
func TestFleetHasNoGenericDatabasePaths(t *testing.T) {
	scenariosRoot, ok := findScenariosRoot()
	if !ok {
		t.Skip("no scenarios/ directory above the working directory")
	}

	offences, err := fleetDatabasePathOffences(filepath.Dir(scenariosRoot))
	if err != nil {
		t.Fatalf("audit fleet database paths: %v", err)
	}

	if len(offences) > 0 {
		t.Fatalf("a scenario resolves its database outside the owned storage seam:\n  %s\n\n"+
			"Resolve the database from the scenario's own identity with "+
			"storage.SQLiteDSN(storage.SQLiteConfig{Scenario: \"<scenario>\"}), or pass an "+
			"explicit path as an argument to storage.SQLiteDSNAt. See "+
			"packages/api-core/storage/sqlite.go.",
			strings.Join(offences, "\n  "))
	}
}

// fleetDatabasePathOffences uses the repository's source boundary, which excludes
// owner documentation evidence but retains source packages named docs.
func fleetDatabasePathOffences(repoRoot string) ([]string, error) {
	contract, err := repocontract.LoadDefault(repoRoot)
	if err != nil {
		return nil, err
	}
	scenariosRoot := filepath.Join(repoRoot, filepath.FromSlash(contract.Layout().ScenarioDir))
	var offences []string
	a := isoDatabasePath{}
	err = contract.WalkSource(repoRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(scenariosRoot, path)
		if err != nil {
			return err
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if entry.IsDir() {
			if skipDirName(entry.Name()) {
				return fs.SkipDir
			}
			if len(parts) == 1 {
				if _, denied := deniedScenarios[parts[0]]; denied {
					return fs.SkipDir
				}
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || len(parts) < 2 {
			return nil
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read source %s: %w", path, err)
		}
		for _, f := range a.analyzeSource(string(source), strings.Join(parts[1:], "/")) {
			offences = append(offences, parts[0]+"/"+f.Location+": "+f.Title)
		}
		return nil
	})
	return offences, err
}

func TestFleetDatabasePathsSourceBoundary(t *testing.T) {
	fixture := repocontracttest.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	fixture.WriteScenarioStub(t, "example")
	badSource := `package example
import "os"
func databasePath() string { return os.Getenv("SQLITE_PATH") }
`
	for _, rel := range []string{
		"scenarios/example/docs/internal/goal/evidence/source/broken.go",
		"scenarios/example/api/internal/live/connection.go",
		"scenarios/example/api/handlers/docs/connection.go",
	} {
		repocontracttest.WriteFile(t, filepath.Join(fixture.Root, filepath.FromSlash(rel)), badSource)
	}
	offences, err := fleetDatabasePathOffences(fixture.Root)
	if err != nil {
		t.Fatal(err)
	}
	if len(offences) != 2 {
		t.Fatalf("want both live source violations and no retained evidence violation, got %v", offences)
	}
	for _, want := range []string{"example/api/internal/live/connection.go:", "example/api/handlers/docs/connection.go:"} {
		if !strings.Contains(strings.Join(offences, "\n"), want) {
			t.Errorf("missing production violation %q: %v", want, offences)
		}
	}
}

// skipDirName reports whether a directory holds vendored, generated, or
// third-party sources that this repository does not own.
func skipDirName(name string) bool {
	switch name {
	case "node_modules", "dist", "build", "vendor", ".git", "testdata":
		return true
	}
	return false
}

// findScenariosRoot walks up from the working directory looking for a
// scenarios/ directory.
func findScenariosRoot() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}
	for {
		candidate := filepath.Join(dir, "scenarios")
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}
