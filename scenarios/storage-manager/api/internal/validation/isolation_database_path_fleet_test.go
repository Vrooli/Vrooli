package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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

	a := isoDatabasePath{}
	var offences []string

	entries, err := os.ReadDir(scenariosRoot)
	if err != nil {
		t.Fatalf("read scenarios root: %v", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		scenario := entry.Name()
		if _, denied := deniedScenarios[scenario]; denied {
			continue
		}
		scenarioDir := filepath.Join(scenariosRoot, scenario)
		_ = filepath.Walk(scenarioDir, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil || info == nil {
				return nil
			}
			if info.IsDir() {
				if skipDirName(info.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			rel, relErr := filepath.Rel(scenarioDir, path)
			if relErr != nil {
				return nil
			}
			source, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			for _, f := range a.analyzeSource(string(source), rel) {
				offences = append(offences, scenario+"/"+f.Location+": "+f.Title)
			}
			return nil
		})
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
