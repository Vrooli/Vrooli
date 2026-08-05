package evidence

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvidenceSchemaStoresReferencesNotArtifactBytesOrPaths(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("schema.sql"))
	if err != nil {
		t.Fatal(err)
	}
	schema := strings.ToLower(string(data))
	for _, forbidden := range []string{"blob", "bytea", "artifact_path", "file_path", "local_path"} {
		if strings.Contains(schema, forbidden) {
			t.Fatalf("evidence schema must not contain %q", forbidden)
		}
	}
	for _, required := range []string{"artifact_id", "checksum", "size_bytes", "producer"} {
		if !strings.Contains(schema, required) {
			t.Fatalf("evidence schema must contain reference field %q", required)
		}
	}
}
