package archtest

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestAttemptStoreIsTheOnlyProductionRoundIOOwner prevents a future domain
// from quietly reintroducing private round-file persistence. Test fixtures may
// expose compatibility helpers, but production code must call attemptstore.
func TestAttemptStoreIsTheOnlyProductionRoundIOOwner(t *testing.T) {
	root := filepath.Join("..")
	declarations := regexp.MustCompile(`(?m)^func (?:\([^)]*\) )?(?:LoadRounds|LoadRound|LoadLatestRound|SaveRound|NextRoundNumber|RoundFilename)\b`)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if filepath.Clean(filepath.Dir(path)) == filepath.Join("..", "attemptstore") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if declarations.Match(data) {
			t.Errorf("%s declares round-store I/O; use internal/attemptstore instead", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk internal packages: %v", err)
	}
}
