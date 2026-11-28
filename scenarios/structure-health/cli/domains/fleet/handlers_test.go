package fleet

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteJSONArtifactUsesRepositoryRootForRelativePaths(t *testing.T) {
	root := t.TempDir()
	value := map[string]any{"count": 3, "names": []string{"alpha", "beta"}}

	if err := writeJSONArtifact(root, "docs/evidence.json", value); err != nil {
		t.Fatalf("writeJSONArtifact: %v", err)
	}

	path := filepath.Join(root, "docs", "evidence.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("decode artifact: %v", err)
	}
	if got["count"] != float64(3) {
		t.Fatalf("count = %v, want 3", got["count"])
	}
	if len(raw) == 0 || raw[len(raw)-1] != '\n' {
		t.Fatal("artifact does not end with a newline")
	}
}
