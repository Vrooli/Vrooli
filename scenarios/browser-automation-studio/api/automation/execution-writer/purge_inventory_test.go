package executionwriter

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestInventoryRefreshRejectsUnknownTopLevelFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "result.json"), []byte("owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "unrelated.txt"), []byte("not BAS owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := inventoryDirectoryFiles(root); err == nil {
		t.Fatal("inventory refresh blessed an unknown top-level file")
	}
}

func TestInventoryRefreshRejectsUnknownArtifactSubtree(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "artifacts", "private", "payload.bin")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("not BAS owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := inventoryDirectoryFiles(root); err == nil {
		t.Fatal("inventory refresh blessed an unknown artifact subtree")
	}
}

func TestReadPurgeInventoryArtifactPathsUsesWriterSchemaAndExactIdentity(t *testing.T) {
	executionID, workflowID := uuid.New(), uuid.New()
	rootPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(rootPath, purgeInventoryFileName), mustMarshalInventory(t, purgeInventory{
		SchemaVersion: "bas-execution-directory/v1",
		ExecutionID:   executionID.String(),
		WorkflowID:    workflowID.String(),
		Files:         []purgeInventoryFile{{Path: ".tmp-published", Bytes: 4, SHA256: testInventoryDigest("file")}},
		ArtifactBindings: []purgeArtifactBinding{
			{ArtifactID: "artifact-file", Scope: "evidence", Storage: "execution_file", Path: ".tmp-bound", Bytes: 3, SHA256: testInventoryDigest("bin")},
			{ArtifactID: "artifact-external", Scope: "evidence", Storage: "external", ObjectKey: "artifacts/" + executionID.String() + "/trace.bin", Bytes: 5, SHA256: testInventoryDigest("trace")},
		},
	}), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	got, err := ReadPurgeInventoryArtifactPaths(root, executionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("referenced paths = %#v", got)
	}
	for _, want := range []string{".tmp-published", ".tmp-bound"} {
		if _, ok := got[want]; !ok {
			t.Fatalf("referenced path %q missing from %#v", want, got)
		}
	}
}

func TestReadPurgeInventoryArtifactPathsFailsClosedOnUntrustedInventory(t *testing.T) {
	executionID, workflowID := uuid.New(), uuid.New()
	valid := purgeInventory{SchemaVersion: "bas-execution-directory/v1", ExecutionID: executionID.String(), WorkflowID: workflowID.String(), Files: []purgeInventoryFile{}}
	for _, name := range []string{"foreign execution", "unknown field", "trailing value", "invalid path", "missing files", "missing bindings", "invalid digest"} {
		t.Run(name, func(t *testing.T) {
			rootPath := t.TempDir()
			raw := string(mustMarshalInventory(t, valid))
			switch name {
			case "unknown field":
				raw = strings.TrimSuffix(raw, "}") + `,"extra":true}`
			case "trailing value":
				raw += `{}`
			case "invalid path":
				raw = strings.Replace(raw, `"files":[]`, `"files":[{"path":"../.tmp-foreign","bytes":0,"sha256":"x"}]`, 1)
			case "missing files":
				raw = strings.Replace(raw, `"files":[]`, `"files":null`, 1)
			case "missing bindings":
				raw = strings.Replace(raw, `,"artifact_bindings":null`, "", 1)
			case "invalid digest":
				raw = strings.Replace(raw, `"files":[]`, `"files":[{"path":".tmp-x","bytes":0,"sha256":"x"}]`, 1)
			case "foreign execution":
				raw = strings.Replace(raw, executionID.String(), uuid.NewString(), 1)
			}
			if err := os.WriteFile(filepath.Join(rootPath, purgeInventoryFileName), []byte(raw), 0o600); err != nil {
				t.Fatal(err)
			}
			root, err := os.OpenRoot(rootPath)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			if _, err := ReadPurgeInventoryArtifactPaths(root, executionID); err == nil {
				t.Fatal("untrusted inventory accepted")
			}
		})
	}
}

func mustMarshalInventory(t *testing.T, inventory purgeInventory) []byte {
	t.Helper()
	raw, err := json.Marshal(inventory)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func testInventoryDigest(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}
