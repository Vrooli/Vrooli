package jsonutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFileRedactedRemovesOperatorPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "artifact.json")
	if err := WriteFileRedacted(path, map[string]string{"clone": "/home/operator/Vrooli/scenarios/swarm-manager"}); err != nil {
		t.Fatalf("WriteFileRedacted: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	if strings.Contains(string(data), "/home/operator") {
		t.Fatalf("operator path leaked: %s", data)
	}
}
