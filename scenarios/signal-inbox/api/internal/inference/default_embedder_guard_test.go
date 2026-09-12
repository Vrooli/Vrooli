package inference

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestScenarioNeverConstructsDefaultEmbedder prevents a later domain from
// bypassing Signal Inbox's gateway-owned inference policy.
func TestScenarioNeverConstructsDefaultEmbedder(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return walkErr
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(body), "aisearch.NewEmbedder(") || strings.Contains(string(body), "aisearch.NewEmbedderForConfig(") {
			t.Errorf("%s constructs ai-go's default embedder; wire inference.Embedder instead", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
